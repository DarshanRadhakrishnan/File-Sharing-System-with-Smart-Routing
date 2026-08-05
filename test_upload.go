package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	pb "github.com/darshan/p2p-fileshare/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	fileData, err := os.ReadFile("test_files/sample.txt")
	if err != nil {
		log.Fatalf("Failed to read sample.txt: %v", err)
	}

	conn, err := grpc.NewClient("localhost:9001", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Peer-1: %v", err)
	}
	defer conn.Close()

	client := pb.NewPeerServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fmt.Println("Uploading sample.txt to Peer-1...")
	uploadResp, err := client.UploadFile(ctx, &pb.UploadFileRequest{
		FileName: "sample.txt",
		FileData: fileData,
	})
	if err != nil {
		log.Fatalf("Upload failed: %v", err)
	}

	fmt.Printf("[Peer-1] Uploaded 'sample.txt' as %d chunks (file_id=%s)\n\n", 
		uploadResp.TotalChunks, uploadResp.FileId)

	fmt.Println("Querying Peer-1 bloom filter for uploaded chunks...")
	bloomResp, err := client.GetBloomFilter(ctx, &pb.GetBloomFilterRequest{})
	if err != nil {
		log.Fatalf("GetBloomFilter failed: %v", err)
	}

	out := map[string]interface{}{
		"numBits":          bloomResp.NumBits,
		"numHashFunctions": bloomResp.NumHashFunctions,
		"itemsAdded":       bloomResp.ItemsAdded,
	}

	b, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(b))
}
