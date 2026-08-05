package main

import (
	"fmt"

	"github.com/darshan/p2p-fileshare/internal/storage"
)

func main() {
	fmt.Println("╔════════════════════════════════════╗")
	fmt.Println("║   HASH VERIFICATION TEST           ║")
	fmt.Println("╚════════════════════════════════════╝")
	fmt.Println()

	passed := 0
	failed := 0

	// Test 1: Valid chunk passes verification
	data := []byte("this is a test chunk of file data")
	hash := storage.ComputeHash(data)
	fmt.Printf("Original data hash: %s\n", hash)

	err := storage.VerifyChunk(data, hash)
	if err == nil {
		fmt.Println("PASS: Valid chunk verified correctly")
		passed++
	} else {
		fmt.Println("FAIL: Valid chunk rejected:", err)
		failed++
	}

	// Test 2: Tampered chunk fails verification
	tamperedData := []byte("this is a TAMPERED chunk of file data")
	err = storage.VerifyChunk(tamperedData, hash)
	if err != nil {
		fmt.Println("PASS: Tampered chunk correctly rejected:", err)
		passed++
	} else {
		fmt.Println("FAIL: Tampered chunk was NOT detected!")
		failed++
	}

	// Test 3: Merkle tree file integrity (unchanged)
	chunkHashes := []string{
		storage.ComputeHash([]byte("chunk-0")),
		storage.ComputeHash([]byte("chunk-1")),
		storage.ComputeHash([]byte("chunk-2")),
		storage.ComputeHash([]byte("chunk-3")),
	}

	root := storage.RootHash(chunkHashes)
	fmt.Printf("\nMerkle root: %s\n", root)

	valid := storage.VerifyFileIntegrity(chunkHashes, root)
	fmt.Printf("File integrity check (unchanged): %v (expect true)\n", valid)
	if valid {
		passed++
	} else {
		failed++
	}

	// Test 4: Merkle tree detects corruption
	chunkHashes[2] = storage.ComputeHash([]byte("CORRUPTED"))
	valid = storage.VerifyFileIntegrity(chunkHashes, root)
	fmt.Printf("File integrity check (corrupted):  %v (expect false)\n", valid)
	if !valid {
		fmt.Println("PASS: Merkle tree detected corruption")
		passed++
	} else {
		fmt.Println("FAIL: Merkle tree did NOT detect corruption!")
		failed++
	}

	fmt.Printf("\n══════════════════════════════════════\n")
	fmt.Printf("Results: %d passed, %d failed\n", passed, failed)
	if failed > 0 {
		fmt.Println("OVERALL: FAIL")
	} else {
		fmt.Println("OVERALL: ALL TESTS PASSED ✓")
	}
}
