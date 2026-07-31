package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"

	pb "github.com/darshan/p2p-fileshare/proto"
	"google.golang.org/grpc"
)

// ============================================================
// BOOTSTRAP SERVER
// Central registry for P2P peer discovery
// ============================================================

// BootstrapServer implements the BootstrapService gRPC interface.
type BootstrapServer struct {
	pb.UnimplementedBootstrapServiceServer
	mu    sync.RWMutex
	peers map[int32]*pb.PeerInfo
	rng   *rand.Rand
}

// NewBootstrapServer creates a new bootstrap server.
func NewBootstrapServer() *BootstrapServer {
	return &BootstrapServer{
		peers: make(map[int32]*pb.PeerInfo),
		rng:   rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// RegisterPeer registers a new peer and returns a list of known peers.
func (s *BootstrapServer) RegisterPeer(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Store the peer
	s.peers[req.PeerId] = &pb.PeerInfo{
		PeerId:   req.PeerId,
		Address:  req.Address,
		LastSeen: time.Now().Unix(),
	}

	fmt.Printf("[Bootstrap] Registered peer %d at %s\n", req.PeerId, req.Address)

	// Return up to 3 random OTHER peers
	otherPeers := s.getRandomPeers(req.PeerId, 3)

	return &pb.RegisterResponse{
		Success: true,
		Peers:   otherPeers,
	}, nil
}

// GetPeers returns a list of active peers up to the requested limit.
func (s *BootstrapServer) GetPeers(ctx context.Context, req *pb.GetPeersRequest) (*pb.GetPeersResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}

	peers := make([]*pb.PeerInfo, 0, len(s.peers))
	for _, p := range s.peers {
		peers = append(peers, p)
		if int32(len(peers)) >= limit {
			break
		}
	}

	return &pb.GetPeersResponse{
		Peers: peers,
	}, nil
}

// getRandomPeers returns up to 'count' random peers, excluding the given peer ID.
func (s *BootstrapServer) getRandomPeers(excludeID int32, count int) []*pb.PeerInfo {
	// Collect all peers except the excluded one
	candidates := make([]*pb.PeerInfo, 0, len(s.peers))
	for id, p := range s.peers {
		if id != excludeID {
			candidates = append(candidates, p)
		}
	}

	// Shuffle and take up to 'count'
	s.rng.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})

	if len(candidates) > count {
		candidates = candidates[:count]
	}

	return candidates
}

func main() {
	port := ":5000"

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("[Bootstrap] Failed to listen on %s: %v", port, err)
	}

	grpcServer := grpc.NewServer()
	bootstrapServer := NewBootstrapServer()
	pb.RegisterBootstrapServiceServer(grpcServer, bootstrapServer)

	fmt.Printf("[Bootstrap] Server listening on %s\n", port)

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("[Bootstrap] Failed to serve: %v", err)
	}
}
