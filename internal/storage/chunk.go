package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// DefaultChunkSize is the size of each chunk (256KB)
const DefaultChunkSize = 256 * 1024

// Chunk represents a single piece of a file
type Chunk struct {
	FileID string
	Index  int32
	Data   []byte
	Hash   string
	Size   int32
}

// FileMetadata holds info about a complete file
type FileMetadata struct {
	FileID      string
	FileName    string
	TotalSize   int64
	TotalChunks int32
	ChunkHashes []string
	RootHash    string // Merkle root
}

// ChunkStore manages chunk storage on disk + metadata in memory
type ChunkStore struct {
	baseDir string
	files   map[string]*FileMetadata
	chunks  map[string][]byte // key: "fileID:chunkIndex"
	mu      sync.RWMutex
}

// NewChunkStore creates a storage manager rooted at baseDir
func NewChunkStore(baseDir string) (*ChunkStore, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage dir: %v", err)
	}

	return &ChunkStore{
		baseDir: baseDir,
		files:   make(map[string]*FileMetadata),
		chunks:  make(map[string][]byte),
	}, nil
}

// SplitFile breaks file data into chunks, hashes each one, and stores them
func (cs *ChunkStore) SplitFile(fileID, fileName string, data []byte) (*FileMetadata, []*Chunk, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	totalSize := int64(len(data))
	numChunks := int32((totalSize + DefaultChunkSize - 1) / DefaultChunkSize)

	chunks := make([]*Chunk, 0, numChunks)
	hashes := make([]string, 0, numChunks)

	for i := int32(0); i < numChunks; i++ {
		start := int64(i) * DefaultChunkSize
		end := start + DefaultChunkSize
		if end > totalSize {
			end = totalSize
		}

		chunkData := data[start:end]
		hash := ComputeHash(chunkData)

		chunk := &Chunk{
			FileID: fileID,
			Index:  i,
			Data:   chunkData,
			Hash:   hash,
			Size:   int32(len(chunkData)),
		}

		chunks = append(chunks, chunk)
		hashes = append(hashes, hash)

		// Store chunk data in memory + disk
		key := chunkStoreKey(fileID, i)
		cs.chunks[key] = chunkData

		if err := cs.persistChunk(fileID, i, chunkData); err != nil {
			return nil, nil, err
		}
	}

	metadata := &FileMetadata{
		FileID:      fileID,
		FileName:    fileName,
		TotalSize:   totalSize,
		TotalChunks: numChunks,
		ChunkHashes: hashes,
		RootHash:    RootHash(hashes),
	}

	cs.files[fileID] = metadata

	return metadata, chunks, nil
}

// persistChunk writes chunk data to disk
func (cs *ChunkStore) persistChunk(fileID string, index int32, data []byte) error {
	fileDir := filepath.Join(cs.baseDir, fileID)
	if err := os.MkdirAll(fileDir, 0755); err != nil {
		return err
	}

	chunkPath := filepath.Join(fileDir, fmt.Sprintf("chunk_%d", index))
	return os.WriteFile(chunkPath, data, 0644)
}

// HasChunk checks if this store actually has a specific chunk (ground truth)
func (cs *ChunkStore) HasChunk(fileID string, index int32) bool {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	key := chunkStoreKey(fileID, index)
	_, exists := cs.chunks[key]
	return exists
}

// GetChunk retrieves chunk data, loading from disk if needed
func (cs *ChunkStore) GetChunk(fileID string, index int32) ([]byte, error) {
	cs.mu.RLock()
	key := chunkStoreKey(fileID, index)
	data, exists := cs.chunks[key]
	cs.mu.RUnlock()

	if exists {
		return data, nil
	}

	// Fallback: load from disk
	chunkPath := filepath.Join(cs.baseDir, fileID, fmt.Sprintf("chunk_%d", index))
	data, err := os.ReadFile(chunkPath)
	if err != nil {
		return nil, fmt.Errorf("chunk not found: %s chunk %d", fileID, index)
	}

	return data, nil
}

// StoreDownloadedChunk saves a chunk received from a peer after verification
func (cs *ChunkStore) StoreDownloadedChunk(fileID string, index int32, data []byte, expectedHash string) error {
	if err := VerifyChunk(data, expectedHash); err != nil {
		return fmt.Errorf("chunk verification failed: %v", err)
	}

	cs.mu.Lock()
	key := chunkStoreKey(fileID, index)
	cs.chunks[key] = data
	cs.mu.Unlock()

	return cs.persistChunk(fileID, index, data)
}

// GetFileMetadata returns metadata for a known file
func (cs *ChunkStore) GetFileMetadata(fileID string) (*FileMetadata, bool) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	meta, exists := cs.files[fileID]
	return meta, exists
}

// AssembleFile reconstructs a complete file from all its chunks
func (cs *ChunkStore) AssembleFile(fileID string) ([]byte, error) {
	meta, exists := cs.GetFileMetadata(fileID)
	if !exists {
		return nil, fmt.Errorf("unknown file: %s", fileID)
	}

	var result []byte
	for i := int32(0); i < meta.TotalChunks; i++ {
		chunkData, err := cs.GetChunk(fileID, i)
		if err != nil {
			return nil, fmt.Errorf("missing chunk %d: %v", i, err)
		}
		result = append(result, chunkData...)
	}

	return result, nil
}

// chunkStoreKey generates a map key for in-memory chunk lookup.
// Uses a simple "fileID:index" format (distinct from the bloom filter's ChunkKey
// which uses binary encoding for hash distribution).
func chunkStoreKey(fileID string, index int32) string {
	return fmt.Sprintf("%s:%d", fileID, index)
}
