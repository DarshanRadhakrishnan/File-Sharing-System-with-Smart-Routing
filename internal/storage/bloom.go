package storage

import (
	"encoding/binary"
	"hash/fnv"
	"math"
	"sync"
)

// BloomFilter is a space-efficient probabilistic data structure
// used to test whether an element is a member of a set.
// False positives are possible; false negatives are NOT.
type BloomFilter struct {
	bitArray  []byte
	numBits   int
	numHashes int
	itemCount int
	mu        sync.RWMutex
}

// NewBloomFilter creates a Bloom filter sized for expectedItems
// with target false positive rate falsePositiveRate (e.g. 0.01 for 1%)
func NewBloomFilter(expectedItems int, falsePositiveRate float64) *BloomFilter {
	numBits := optimalNumBits(expectedItems, falsePositiveRate)
	numHashes := optimalNumHashes(numBits, expectedItems)

	return &BloomFilter{
		bitArray:  make([]byte, (numBits+7)/8), // round up to bytes
		numBits:   numBits,
		numHashes: numHashes,
	}
}

// optimalNumBits calculates m = -(n * ln(p)) / (ln(2)^2)
func optimalNumBits(n int, p float64) int {
	m := -1 * float64(n) * math.Log(p) / (math.Ln2 * math.Ln2)
	return int(math.Ceil(m))
}

// optimalNumHashes calculates k = (m/n) * ln(2)
func optimalNumHashes(m, n int) int {
	k := (float64(m) / float64(n)) * math.Ln2
	rounded := int(math.Round(k))
	if rounded < 1 {
		return 1
	}
	return rounded
}

// Add inserts an item (string key) into the filter
func (bf *BloomFilter) Add(key string) {
	bf.mu.Lock()
	defer bf.mu.Unlock()

	h1, h2 := bf.baseHashes(key)
	for i := 0; i < bf.numHashes; i++ {
		bitPos := bf.combinedHash(h1, h2, i) % uint32(bf.numBits)
		bf.setBit(int(bitPos))
	}
	bf.itemCount++
}

// MightContain checks if item might be in the set.
// Returns true = "possibly in set" (could be false positive)
// Returns false = "definitely NOT in set" (100% guaranteed)
func (bf *BloomFilter) MightContain(key string) bool {
	bf.mu.RLock()
	defer bf.mu.RUnlock()

	h1, h2 := bf.baseHashes(key)
	for i := 0; i < bf.numHashes; i++ {
		bitPos := bf.combinedHash(h1, h2, i) % uint32(bf.numBits)
		if !bf.getBit(int(bitPos)) {
			return false // definitely not present
		}
	}
	return true // possibly present
}

// baseHashes generates two independent hashes using FNV
// Used with double hashing technique: h_i = h1 + i*h2
func (bf *BloomFilter) baseHashes(key string) (uint32, uint32) {
	h1 := fnv.New32a()
	h1.Write([]byte(key))
	sum1 := h1.Sum32()

	h2 := fnv.New32()
	h2.Write([]byte(key))
	sum2 := h2.Sum32()

	return sum1, sum2
}

// combinedHash implements Kirsch-Mitzenmacher double hashing:
// g_i(x) = h1(x) + i*h2(x) mod m
func (bf *BloomFilter) combinedHash(h1, h2 uint32, i int) uint32 {
	return h1 + uint32(i)*h2
}

func (bf *BloomFilter) setBit(pos int) {
	byteIndex := pos / 8
	bitIndex := uint(pos % 8)
	bf.bitArray[byteIndex] |= 1 << bitIndex
}

func (bf *BloomFilter) getBit(pos int) bool {
	byteIndex := pos / 8
	bitIndex := uint(pos % 8)
	return (bf.bitArray[byteIndex] & (1 << bitIndex)) != 0
}

// EstimatedFalsePositiveRate returns current estimated FP rate
// based on how full the filter is: (1 - e^(-kn/m))^k
func (bf *BloomFilter) EstimatedFalsePositiveRate() float64 {
	bf.mu.RLock()
	defer bf.mu.RUnlock()

	k := float64(bf.numHashes)
	n := float64(bf.itemCount)
	m := float64(bf.numBits)

	exponent := -k * n / m
	inner := 1 - math.Exp(exponent)
	return math.Pow(inner, k)
}

// Serialize exports the bit array for network transfer
func (bf *BloomFilter) Serialize() ([]byte, int32, int32, int32) {
	bf.mu.RLock()
	defer bf.mu.RUnlock()

	dataCopy := make([]byte, len(bf.bitArray))
	copy(dataCopy, bf.bitArray)

	return dataCopy, int32(bf.numBits), int32(bf.numHashes), int32(bf.itemCount)
}

// DeserializeBloomFilter reconstructs a Bloom filter from network data
func DeserializeBloomFilter(bitArray []byte, numBits, numHashes, itemCount int32) *BloomFilter {
	return &BloomFilter{
		bitArray:  bitArray,
		numBits:   int(numBits),
		numHashes: int(numHashes),
		itemCount: int(itemCount),
	}
}

// Merge combines another bloom filter into this one (union operation)
// Both filters MUST have same numBits and numHashes
func (bf *BloomFilter) Merge(other *BloomFilter) error {
	bf.mu.Lock()
	defer bf.mu.Unlock()
	other.mu.RLock()
	defer other.mu.RUnlock()

	if bf.numBits != other.numBits {
		return ErrIncompatibleFilters
	}

	for i := range bf.bitArray {
		bf.bitArray[i] |= other.bitArray[i]
	}
	return nil
}

// BloomStats contains human-readable filter statistics
type BloomStats struct {
	NumBits         int
	NumHashes       int
	ItemCount       int
	EstimatedFPRate float64
	SizeBytes       int
}

// Stats returns human-readable filter statistics
func (bf *BloomFilter) Stats() BloomStats {
	return BloomStats{
		NumBits:         bf.numBits,
		NumHashes:       bf.numHashes,
		ItemCount:       bf.itemCount,
		EstimatedFPRate: bf.EstimatedFalsePositiveRate(),
		SizeBytes:       len(bf.bitArray),
	}
}

// ErrIncompatibleFilters is returned when merging bloom filters of different sizes
var ErrIncompatibleFilters = bloomErr("bloom filters have different sizes and cannot be merged")

type bloomErr string

func (e bloomErr) Error() string { return string(e) }

// ChunkKey encodes a chunk key consistently for bloom filter operations
func ChunkKey(fileID string, chunkIndex int32) string {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(chunkIndex))
	return fileID + ":" + string(buf)
}
