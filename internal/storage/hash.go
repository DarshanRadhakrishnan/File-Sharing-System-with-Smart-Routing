package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// ComputeHash returns the SHA-256 hex digest of data
func ComputeHash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// VerifyChunk checks that data matches the expected hash
// Returns nil if valid, error describing mismatch otherwise
func VerifyChunk(data []byte, expectedHash string) error {
	actualHash := ComputeHash(data)
	if actualHash != expectedHash {
		return fmt.Errorf("hash mismatch: expected %s, got %s", expectedHash, actualHash)
	}
	return nil
}

// MerkleNode represents a node in a Merkle tree for file-level verification
type MerkleNode struct {
	Hash  string
	Left  *MerkleNode
	Right *MerkleNode
}

// BuildMerkleTree constructs a Merkle tree from chunk hashes
// Allows verifying the whole file with a single root hash
func BuildMerkleTree(chunkHashes []string) *MerkleNode {
	if len(chunkHashes) == 0 {
		return nil
	}

	nodes := make([]*MerkleNode, len(chunkHashes))
	for i, h := range chunkHashes {
		nodes[i] = &MerkleNode{Hash: h}
	}

	for len(nodes) > 1 {
		var nextLevel []*MerkleNode

		for i := 0; i < len(nodes); i += 2 {
			if i+1 < len(nodes) {
				combined := ComputeHash([]byte(nodes[i].Hash + nodes[i+1].Hash))
				nextLevel = append(nextLevel, &MerkleNode{
					Hash:  combined,
					Left:  nodes[i],
					Right: nodes[i+1],
				})
			} else {
				// Odd node out, promote it
				nextLevel = append(nextLevel, nodes[i])
			}
		}

		nodes = nextLevel
	}

	return nodes[0]
}

// RootHash returns the Merkle root for file integrity verification
func RootHash(chunkHashes []string) string {
	tree := BuildMerkleTree(chunkHashes)
	if tree == nil {
		return ""
	}
	return tree.Hash
}

// VerifyFileIntegrity checks if a set of chunk hashes produces the expected root
func VerifyFileIntegrity(chunkHashes []string, expectedRoot string) bool {
	return RootHash(chunkHashes) == expectedRoot
}
