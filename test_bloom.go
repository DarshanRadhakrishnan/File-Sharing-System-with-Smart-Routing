package main

import (
	"fmt"

	"github.com/darshan/p2p-fileshare/internal/storage"
)

func main() {
	fmt.Println("╔════════════════════════════════════╗")
	fmt.Println("║   BLOOM FILTER ACCURACY TEST       ║")
	fmt.Println("╚════════════════════════════════════╝")
	fmt.Println()

	bf := storage.NewBloomFilter(1000, 0.01)

	// Add 500 known items
	for i := 0; i < 500; i++ {
		bf.Add(fmt.Sprintf("chunk-%d", i))
	}

	// Test: all 500 should be found (no false negatives)
	falseNegatives := 0
	for i := 0; i < 500; i++ {
		if !bf.MightContain(fmt.Sprintf("chunk-%d", i)) {
			falseNegatives++
		}
	}

	// Test: check 500 items NOT added, count false positives
	falsePositives := 0
	for i := 1000; i < 1500; i++ {
		if bf.MightContain(fmt.Sprintf("chunk-%d", i)) {
			falsePositives++
		}
	}

	stats := bf.Stats()
	fmt.Printf("False Negatives: %d (MUST be 0)\n", falseNegatives)
	fmt.Printf("False Positives: %d / 500 (%.2f%%, target <1%%)\n", falsePositives, float64(falsePositives)/500*100)
	fmt.Printf("Bloom Stats: %+v\n", stats)
	fmt.Println()

	if falseNegatives > 0 {
		fmt.Println("FAIL: Bloom filter has false negatives (should be impossible)")
	} else {
		fmt.Println("PASS: No false negatives ✓")
	}

	if float64(falsePositives)/500*100 >= 1.0 {
		fmt.Println("WARN: False positive rate >= 1% (higher than target)")
	} else {
		fmt.Println("PASS: False positive rate < 1% ✓")
	}
}
