package storage

import (
	"fmt"
	"log"

	"github.com/google/uuid"
)

// StorageManager is the single entry point for chunk/bloom operations
// used by the PeerNode struct
type StorageManager struct {
	Store  *ChunkStore
	Bloom  *BloomFilter
	peerID int32
}

// NewStorageManager creates a manager with sensible defaults:
// expects up to 10,000 chunks with 1% false positive rate
func NewStorageManager(peerID int32, baseDir string) (*StorageManager, error) {
	store, err := NewChunkStore(baseDir)
	if err != nil {
		return nil, err
	}

	bloom := NewBloomFilter(10000, 0.01)

	return &StorageManager{
		Store:  store,
		Bloom:  bloom,
		peerID: peerID,
	}, nil
}

// UploadFile splits a file, stores chunks, and updates the bloom filter
func (sm *StorageManager) UploadFile(fileName string, data []byte) (*FileMetadata, error) {
	fileID := uuid.New().String()

	metadata, chunks, err := sm.Store.SplitFile(fileID, fileName, data)
	if err != nil {
		return nil, err
	}

	for _, chunk := range chunks {
		key := ChunkKey(chunk.FileID, chunk.Index)
		sm.Bloom.Add(key)
	}

	log.Printf("[Peer-%d] Uploaded '%s' as %d chunks (file_id=%s, root_hash=%s)\n",
		sm.peerID, fileName, metadata.TotalChunks, fileID, metadata.RootHash[:16])

	return metadata, nil
}

// MightHaveChunk performs the fast bloom filter check
func (sm *StorageManager) MightHaveChunk(fileID string, index int32) bool {
	key := ChunkKey(fileID, index)
	return sm.Bloom.MightContain(key)
}

// ConfirmHasChunk does the actual ground-truth check (used after bloom says maybe)
func (sm *StorageManager) ConfirmHasChunk(fileID string, index int32) bool {
	return sm.Store.HasChunk(fileID, index)
}

// DownloadChunk retrieves raw chunk bytes + hash for sending to a requesting peer
func (sm *StorageManager) DownloadChunk(fileID string, index int32) ([]byte, string, error) {
	data, err := sm.Store.GetChunk(fileID, index)
	if err != nil {
		return nil, "", err
	}

	hash := ComputeHash(data)
	return data, hash, nil
}

// ReceiveChunk verifies and stores a chunk downloaded from another peer
func (sm *StorageManager) ReceiveChunk(fileID string, index int32, data []byte, expectedHash string) error {
	if err := sm.Store.StoreDownloadedChunk(fileID, index, data, expectedHash); err != nil {
		return fmt.Errorf("[Peer-%d] REJECTED chunk %d of %s: %v", sm.peerID, index, fileID, err)
	}

	key := ChunkKey(fileID, index)
	sm.Bloom.Add(key)

	log.Printf("[Peer-%d] Verified & stored chunk %d of %s ✓\n", sm.peerID, index, fileID)
	return nil
}

// PrintStats displays bloom filter and storage statistics
func (sm *StorageManager) PrintStats() {
	stats := sm.Bloom.Stats()
	fmt.Printf(`
╔════════════════════════════════════╗
║   PEER-%d STORAGE STATS           ║
╚════════════════════════════════════╝
Bloom Filter:
  Bits:              %d
  Hash Functions:    %d
  Items Added:       %d
  Est. FP Rate:      %.4f%%
  Size:              %d bytes

`, sm.peerID, stats.NumBits, stats.NumHashes, stats.ItemCount,
		stats.EstimatedFPRate*100, stats.SizeBytes)
}
