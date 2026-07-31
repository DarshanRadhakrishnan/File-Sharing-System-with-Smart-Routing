package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/darshan/p2p-fileshare/internal/network"
	"github.com/darshan/p2p-fileshare/internal/peer"
	pb "github.com/darshan/p2p-fileshare/proto"
	"google.golang.org/grpc"
)

func main() {
	// Parse command line flags
	id := flag.Int("id", 1, "Peer ID (unique integer)")
	port := flag.Int("port", 9001, "Port to listen on")
	bootstrapAddr := flag.String("bootstrap", "localhost:5000", "Bootstrap server address")
	flag.Parse()

	peerID := int32(*id)

	// Create the peer node
	peerNode := peer.NewPeerNode(peerID, *port, *bootstrapAddr)

	// Start gRPC server in a goroutine
	grpcServer := grpc.NewServer()
	peerGRPC := network.NewPeerGRPCServer(peerNode)
	pb.RegisterPeerServiceServer(grpcServer, peerGRPC)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("Peer-%d: Failed to listen on port %d: %v", peerID, *port, err)
	}

	// Start serving in background
	go func() {
		fmt.Printf("Peer-%d: gRPC server listening on :%d\n", peerID, *port)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Peer-%d: Failed to serve gRPC: %v", peerID, err)
		}
	}()

	// Give the server a moment to start
	time.Sleep(500 * time.Millisecond)

	// Register with bootstrap server
	peers, err := peerNode.RegisterWithBootstrap()
	if err != nil {
		log.Fatalf("Peer-%d: %v", peerID, err)
	}

	// Connect to discovered peers and measure latency
	if len(peers) > 0 {
		peerNode.ConnectToPeers(peers)
	} else {
		fmt.Printf("Peer-%d: No other peers found yet (first peer in the network)\n", peerID)
	}

	// Print the network graph
	peerNode.PrintGraph()

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Printf("\nPeer-%d: Running. Press Ctrl+C to stop.\n", peerID)
	<-sigChan

	fmt.Printf("\nPeer-%d: Shutting down...\n", peerID)
	grpcServer.GracefulStop()
	peerNode.Close()
	fmt.Printf("Peer-%d: Shutdown complete.\n", peerID)
}
