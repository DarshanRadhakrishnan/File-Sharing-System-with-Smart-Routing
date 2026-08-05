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

// ============================================================
// ROUTING RPCs (Day 2)
// ============================================================

// CalculateRoute finds the shortest path between two peers using Dijkstra's algorithm.
func (s *PeerGRPCServer) CalculateRoute(ctx context.Context, req *pb.RouteRequest) (*pb.RouteResponse, error) {
	route, err := s.PeerNode.Router.CalculateRoute(s.PeerNode.Graph, req.FromPeerId, req.ToPeerId)
	if err != nil {
		fmt.Printf("Peer-%d: Route calculation failed (%d → %d): %v\n",
			s.PeerNode.ID, req.FromPeerId, req.ToPeerId, err)
		return nil, err
	}

	return &pb.RouteResponse{
		Path:           route.Path,
		TotalLatencyMs: route.TotalLatency,
		Hops:           route.Hops,
	}, nil
}

// GetRouteCache returns all currently cached routes.
func (s *PeerGRPCServer) GetRouteCache(ctx context.Context, req *pb.GetCacheRequest) (*pb.GetCacheResponse, error) {
	cache := s.PeerNode.Router.GetCache()
	routes := cache.Entries()
	ttlSeconds := int32(cache.TTL().Seconds())

	entries := make([]*pb.RouteCacheEntry, 0, len(routes))
	for _, r := range routes {
		entries = append(entries, &pb.RouteCacheEntry{
			FromPeerId:     r.FromPeerID,
			ToPeerId:       r.ToPeerID,
			Path:           r.Path,
			TotalLatencyMs: r.TotalLatency,
			Hops:           r.Hops,
			CachedAt:       r.CreatedAt.Unix(),
			TtlSeconds:     ttlSeconds,
		})
	}

	return &pb.GetCacheResponse{
		Entries: entries,
	}, nil
}

// InvalidateRouteCache clears all cached routes.
func (s *PeerGRPCServer) InvalidateRouteCache(ctx context.Context, req *pb.InvalidateCacheRequest) (*pb.InvalidateCacheResponse, error) {
	cleared := s.PeerNode.Router.InvalidateCache()
	fmt.Printf("Peer-%d: Route cache invalidated (%d entries cleared)\n", s.PeerNode.ID, cleared)

	return &pb.InvalidateCacheResponse{
		Success:        true,
		EntriesCleared: int32(cleared),
	}, nil
}

// ============================================================
// CHUNK & BLOOM FILTER RPCs (Day 3)
// ============================================================

// HasChunk checks if this peer has a specific chunk using the bloom filter
// for a fast probabilistic check, then confirms with ground truth.
func (s *PeerGRPCServer) HasChunk(ctx context.Context, req *pb.HasChunkRequest) (*pb.HasChunkResponse, error) {
	mightHave := s.PeerNode.Storage.MightHaveChunk(req.FileId, req.ChunkIndex)

	confirmed := false
	if mightHave {
		confirmed = s.PeerNode.Storage.ConfirmHasChunk(req.FileId, req.ChunkIndex)
	}

	return &pb.HasChunkResponse{
		MightHave: mightHave,
		Confirmed: confirmed,
	}, nil
}

// GetBloomFilter returns this peer's serialized bloom filter for remote inspection.
func (s *PeerGRPCServer) GetBloomFilter(ctx context.Context, req *pb.GetBloomFilterRequest) (*pb.GetBloomFilterResponse, error) {
	bitArray, numBits, numHashes, itemCount := s.PeerNode.Storage.Bloom.Serialize()

	return &pb.GetBloomFilterResponse{
		BitArray:         bitArray,
		NumBits:          numBits,
		NumHashFunctions: numHashes,
		ItemsAdded:       itemCount,
	}, nil
}

// DownloadChunk returns the raw chunk data and its SHA-256 hash.
func (s *PeerGRPCServer) DownloadChunk(ctx context.Context, req *pb.DownloadChunkRequest) (*pb.DownloadChunkResponse, error) {
	data, hash, err := s.PeerNode.Storage.DownloadChunk(req.FileId, req.ChunkIndex)
	if err != nil {
		return &pb.DownloadChunkResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	return &pb.DownloadChunkResponse{
		Data:       data,
		Sha256Hash: hash,
		Success:    true,
	}, nil
}

// UploadFile splits a file into chunks, hashes each, and stores them.
func (s *PeerGRPCServer) UploadFile(ctx context.Context, req *pb.UploadFileRequest) (*pb.UploadFileResponse, error) {
	metadata, err := s.PeerNode.Storage.UploadFile(req.FileName, req.FileData)
	if err != nil {
		return nil, err
	}

	var chunkInfos []*pb.ChunkInfo
	for i, hash := range metadata.ChunkHashes {
		chunkInfos = append(chunkInfos, &pb.ChunkInfo{
			FileId:     metadata.FileID,
			ChunkIndex: int32(i),
			Sha256Hash: hash,
		})
	}

	return &pb.UploadFileResponse{
		FileId:      metadata.FileID,
		TotalChunks: metadata.TotalChunks,
		Chunks:      chunkInfos,
	}, nil
}
