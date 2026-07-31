package network

import (
	"context"
	"fmt"
	"time"

	"github.com/darshan/p2p-fileshare/internal/peer"
	pb "github.com/darshan/p2p-fileshare/proto"
)

// PeerGRPCServer implements the PeerService gRPC interface.
type PeerGRPCServer struct {
	pb.UnimplementedPeerServiceServer
	PeerNode *peer.PeerNode
}

// NewPeerGRPCServer creates a new gRPC server backed by the given peer node.
func NewPeerGRPCServer(peerNode *peer.PeerNode) *PeerGRPCServer {
	return &PeerGRPCServer{
		PeerNode: peerNode,
	}
}

// Ping handles latency measurement requests.
// Returns the responder's ID and the current timestamp.
func (s *PeerGRPCServer) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	return &pb.PingResponse{
		ResponderId: s.PeerNode.ID,
		Timestamp:   time.Now().UnixNano(),
	}, nil
}

// GetNetworkGraph returns this peer's view of the network topology.
func (s *PeerGRPCServer) GetNetworkGraph(ctx context.Context, req *pb.GraphRequest) (*pb.GraphResponse, error) {
	fmt.Printf("Peer-%d: Graph requested by peer-%d\n", s.PeerNode.ID, req.RequesterId)
	return s.PeerNode.GetGraphProto(), nil
}

// NotifyNewPeer handles notifications about new peers joining the network.
// Stub implementation for future use.
func (s *PeerGRPCServer) NotifyNewPeer(ctx context.Context, req *pb.NewPeerNotification) (*pb.NotifyResponse, error) {
	fmt.Printf("Peer-%d: Notified about new peer-%d at %s\n",
		s.PeerNode.ID, req.PeerId, req.Address)

	// Add the new peer to our graph
	s.PeerNode.Graph.AddNode(req.PeerId, req.Address)

	return &pb.NotifyResponse{
		Acknowledged: true,
	}, nil
}
