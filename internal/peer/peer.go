package peer

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/darshan/p2p-fileshare/internal/routing"
	pb "github.com/darshan/p2p-fileshare/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ============================================================
// NETWORK GRAPH DATA STRUCTURES
// ============================================================

// NetworkNode represents a peer in the network graph.
type NetworkNode struct {
	ID       int32
	Address  string
	LastSeen time.Time
}

// NetworkEdge represents a connection between two peers.
type NetworkEdge struct {
	From      int32
	To        int32
	LatencyMs int64
}

// NetworkGraph holds the in-memory representation of the P2P overlay network.
type NetworkGraph struct {
	mu    sync.RWMutex
	Nodes map[int32]*NetworkNode
	Edges map[string]*NetworkEdge // key: "from-to"
}

// NewNetworkGraph creates a new empty network graph.
func NewNetworkGraph() *NetworkGraph {
	return &NetworkGraph{
		Nodes: make(map[int32]*NetworkNode),
		Edges: make(map[string]*NetworkEdge),
	}
}

// AddNode adds or updates a node in the graph.
func (g *NetworkGraph) AddNode(id int32, address string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.Nodes[id] = &NetworkNode{
		ID:       id,
		Address:  address,
		LastSeen: time.Now(),
	}
}

// AddEdge adds or updates an edge in the graph.
func (g *NetworkGraph) AddEdge(from, to int32, latencyMs int64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	key := fmt.Sprintf("%d-%d", from, to)
	g.Edges[key] = &NetworkEdge{
		From:      from,
		To:        to,
		LatencyMs: latencyMs,
	}
}

// GetNodes returns a snapshot of all nodes.
func (g *NetworkGraph) GetNodes() []*NetworkNode {
	g.mu.RLock()
	defer g.mu.RUnlock()
	nodes := make([]*NetworkNode, 0, len(g.Nodes))
	for _, n := range g.Nodes {
		nodes = append(nodes, n)
	}
	return nodes
}

// GetEdges returns a snapshot of all edges.
func (g *NetworkGraph) GetEdges() []*NetworkEdge {
	g.mu.RLock()
	defer g.mu.RUnlock()
	edges := make([]*NetworkEdge, 0, len(g.Edges))
	for _, e := range g.Edges {
		edges = append(edges, e)
	}
	return edges
}

// GetNodeList returns nodes in the format expected by routing.Graph.
func (g *NetworkGraph) GetNodeList() []routing.GraphNode {
	g.mu.RLock()
	defer g.mu.RUnlock()
	nodes := make([]routing.GraphNode, 0, len(g.Nodes))
	for _, n := range g.Nodes {
		nodes = append(nodes, routing.GraphNode{
			ID:       n.ID,
			Address:  n.Address,
			LastSeen: n.LastSeen,
		})
	}
	return nodes
}

// GetEdgeList returns edges in the format expected by routing.Graph.
func (g *NetworkGraph) GetEdgeList() []routing.GraphEdge {
	g.mu.RLock()
	defer g.mu.RUnlock()
	edges := make([]routing.GraphEdge, 0, len(g.Edges))
	for _, e := range g.Edges {
		edges = append(edges, routing.GraphEdge{
			From:      e.From,
			To:        e.To,
			LatencyMs: e.LatencyMs,
		})
	}
	return edges
}

// ============================================================
// PEER NODE
// ============================================================

// PeerNode represents a single peer in the P2P network.
type PeerNode struct {
	ID            int32
	Port          int
	Address       string
	BootstrapAddr string
	Graph         *NetworkGraph
	Router        *routing.DijkstraRouter
	Analyzer      *routing.GraphAnalyzer
	mu            sync.RWMutex
	connections   map[int32]*grpc.ClientConn // peer_id → gRPC connection
}

// NewPeerNode creates a new peer node.
func NewPeerNode(id int32, port int, bootstrapAddr string) *PeerNode {
	router := routing.NewDijkstraRouter()
	analyzer := routing.NewGraphAnalyzer(router)

	return &PeerNode{
		ID:            id,
		Port:          port,
		Address:       fmt.Sprintf("localhost:%d", port),
		BootstrapAddr: bootstrapAddr,
		Graph:         NewNetworkGraph(),
		Router:        router,
		Analyzer:      analyzer,
		connections:   make(map[int32]*grpc.ClientConn),
	}
}

// CalculateRoute finds the shortest path from this peer to the target peer.
func (p *PeerNode) CalculateRoute(toID int32) (*routing.Route, error) {
	return p.Router.CalculateRoute(p.Graph, p.ID, toID)
}

// GetNetworkStats returns aggregate statistics about the network.
func (p *PeerNode) GetNetworkStats() *routing.NetworkStats {
	return p.Analyzer.AnalyzeNetwork(p.Graph)
}

// RegisterWithBootstrap registers this peer with the bootstrap server
// and returns the list of known peers.
func (p *PeerNode) RegisterWithBootstrap() ([]*pb.PeerInfo, error) {
	// Connect to bootstrap server
	conn, err := grpc.NewClient(
		p.BootstrapAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to bootstrap at %s: %w", p.BootstrapAddr, err)
	}
	defer conn.Close()

	client := pb.NewBootstrapServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.RegisterPeer(ctx, &pb.RegisterRequest{
		PeerId:  p.ID,
		Address: p.Address,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to register with bootstrap: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("bootstrap registration failed")
	}

	// Add ourselves to the graph
	p.Graph.AddNode(p.ID, p.Address)

	fmt.Printf("Peer-%d: Registered with bootstrap. Got %d peers\n", p.ID, len(resp.Peers))
	return resp.Peers, nil
}

// ConnectToPeers connects to the given peers, measures latency, and builds the graph.
func (p *PeerNode) ConnectToPeers(peers []*pb.PeerInfo) {
	var wg sync.WaitGroup

	for _, peerInfo := range peers {
		// Don't connect to ourselves
		if peerInfo.PeerId == p.ID {
			continue
		}

		wg.Add(1)
		go func(info *pb.PeerInfo) {
			defer wg.Done()

			// Connect to the peer
			conn, err := grpc.NewClient(
				info.Address,
				grpc.WithTransportCredentials(insecure.NewCredentials()),
			)
			if err != nil {
				fmt.Printf("Peer-%d: Failed to connect to peer-%d at %s: %v\n",
					p.ID, info.PeerId, info.Address, err)
				return
			}

			// Store the connection
			p.mu.Lock()
			p.connections[info.PeerId] = conn
			p.mu.Unlock()

			// Measure latency
			latency, err := p.measureLatency(conn, info.PeerId)
			if err != nil {
				fmt.Printf("Peer-%d: Failed to measure latency to peer-%d: %v\n",
					p.ID, info.PeerId, err)
				// Still add the node even if latency measurement fails
				p.Graph.AddNode(info.PeerId, info.Address)
				return
			}

			// Add to graph
			p.Graph.AddNode(info.PeerId, info.Address)
			p.Graph.AddEdge(p.ID, info.PeerId, latency)

			fmt.Printf("Peer-%d: Connected to peer-%d (latency: %dms)\n",
				p.ID, info.PeerId, latency)
		}(peerInfo)
	}

	wg.Wait()
}

// measureLatency sends a Ping RPC and measures the round-trip time.
func (p *PeerNode) measureLatency(conn *grpc.ClientConn, targetID int32) (int64, error) {
	client := pb.NewPeerServiceClient(conn)

	// Take the average of 3 pings for accuracy
	var totalLatency int64
	numPings := 3

	for i := 0; i < numPings; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)

		start := time.Now()
		_, err := client.Ping(ctx, &pb.PingRequest{
			SenderId:  p.ID,
			Timestamp: start.UnixNano(),
		})
		elapsed := time.Since(start)
		cancel()

		if err != nil {
			return 0, fmt.Errorf("ping failed: %w", err)
		}

		totalLatency += elapsed.Milliseconds()

		// Small delay between pings
		if i < numPings-1 {
			time.Sleep(50 * time.Millisecond)
		}
	}

	avgLatency := totalLatency / int64(numPings)

	// Ensure at least 1ms for localhost connections
	if avgLatency == 0 {
		// Use microsecond precision for very fast connections
		avgLatency = 1
	}

	return avgLatency, nil
}

// PrintGraph prints the network graph in a formatted way.
func (p *PeerNode) PrintGraph() {
	nodes := p.Graph.GetNodes()
	edges := p.Graph.GetEdges()

	// Sort nodes by ID for consistent output
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].ID < nodes[j].ID
	})

	// Sort edges for consistent output
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From != edges[j].From {
			return edges[i].From < edges[j].From
		}
		return edges[i].To < edges[j].To
	})

	fmt.Println()
	fmt.Printf("=== PEER-%d NETWORK GRAPH ===\n", p.ID)
	fmt.Printf("Nodes (%d):\n", len(nodes))
	for _, n := range nodes {
		addr := n.Address
		if n.ID == p.ID {
			// Show just "localhost" for self
			addr = "localhost"
		}
		fmt.Printf("  %d: %s\n", n.ID, addr)
	}

	fmt.Printf("Edges (%d):\n", len(edges))
	for _, e := range edges {
		fmt.Printf("  %d → %d (latency: %dms)\n", e.From, e.To, e.LatencyMs)
	}
	fmt.Println(strings.Repeat("=", 30))
}

// Close gracefully shuts down all connections.
func (p *PeerNode) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for id, conn := range p.connections {
		if err := conn.Close(); err != nil {
			fmt.Printf("Peer-%d: Error closing connection to peer-%d: %v\n", p.ID, id, err)
		}
	}
	p.connections = make(map[int32]*grpc.ClientConn)
}

// GetGraphProto converts the network graph to protobuf format.
func (p *PeerNode) GetGraphProto() *pb.GraphResponse {
	nodes := p.Graph.GetNodes()
	edges := p.Graph.GetEdges()

	protoNodes := make([]*pb.NetworkNode, len(nodes))
	for i, n := range nodes {
		protoNodes[i] = &pb.NetworkNode{
			Id:       n.ID,
			Address:  n.Address,
			LastSeen: n.LastSeen.Unix(),
		}
	}

	protoEdges := make([]*pb.NetworkEdge, len(edges))
	for i, e := range edges {
		protoEdges[i] = &pb.NetworkEdge{
			FromId:    e.From,
			ToId:      e.To,
			LatencyMs: e.LatencyMs,
		}
	}

	return &pb.GraphResponse{
		Nodes: protoNodes,
		Edges: protoEdges,
	}
}

// init seeds the random number generator.
func init() {
	rand.New(rand.NewSource(time.Now().UnixNano()))
}
