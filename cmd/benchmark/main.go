package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/darshan/p2p-fileshare/internal/routing"
)

// benchGraph is a simple in-memory graph that satisfies routing.Graph.
type benchGraph struct {
	nodes []routing.GraphNode
	edges []routing.GraphEdge
}

func (g *benchGraph) GetNodeList() []routing.GraphNode { return g.nodes }
func (g *benchGraph) GetEdgeList() []routing.GraphEdge { return g.edges }

func main() {
	fmt.Println("╔════════════════════════════════════╗")
	fmt.Println("║  DIJKSTRA ROUTING BENCHMARK        ║")
	fmt.Println("╚════════════════════════════════════╝")
	fmt.Println()

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Create a 100-node random network graph
	graph := createRandomGraph(rng, 100)
	router := routing.NewDijkstraRouter()
	analyzer := routing.NewGraphAnalyzer(router)

	// Print network stats
	fmt.Print(analyzer.GetNetworkMetrics(graph))

	// Benchmark: 1000 route calculations
	routeCount := 1000
	successCount := 0
	cacheHits := 0
	var totalDuration time.Duration

	fmt.Println("Running 1000 route calculations...")
	fmt.Println("─────────────────────────────────────")

	for i := 0; i < routeCount; i++ {
		from := int32(rng.Intn(100)) + 1
		to := int32(rng.Intn(100)) + 1
		if from == to {
			continue
		}

		// Check if this would be a cache hit
		if router.GetCache().Get(from, to) != nil {
			cacheHits++
		}

		start := time.Now()
		_, err := router.CalculateRoute(graph, from, to)
		elapsed := time.Since(start)

		if err != nil {
			continue
		}

		totalDuration += elapsed
		successCount++
	}

	// Results
	fmt.Println()
	fmt.Println("╔════════════════════════════════════╗")
	fmt.Println("║  BENCHMARK RESULTS                 ║")
	fmt.Println("╚════════════════════════════════════╝")
	fmt.Println()

	if successCount > 0 {
		avgTime := totalDuration / time.Duration(successCount)
		routesPerSec := float64(time.Second) / float64(avgTime)

		fmt.Printf("  Successful routes:  %d / %d\n", successCount, routeCount)
		fmt.Printf("  Cache hits:         %d\n", cacheHits)
		fmt.Printf("  Cache entries:      %d\n", router.GetCache().Size())
		fmt.Printf("  Avg calculation:    %v\n", avgTime)
		fmt.Printf("  Routes/second:      %.0f\n", routesPerSec)
		fmt.Println()

		// Performance check
		if avgTime < 100*time.Millisecond {
			fmt.Println("  ✅ PASS: Avg route time < 100ms for 100 nodes")
		} else {
			fmt.Println("  ❌ FAIL: Avg route time >= 100ms for 100 nodes")
		}
	} else {
		fmt.Println("  No successful routes calculated (graph may be disconnected)")
	}

	fmt.Println()

	// Bottleneck analysis
	bottlenecks := analyzer.FindBottlenecks(graph, 30)
	fmt.Printf("  Bottleneck peers (avg latency > 30ms): %d found\n", len(bottlenecks))
	fmt.Println()
}

// createRandomGraph builds a random network with the given number of nodes.
// Each node is connected to 4-8 random neighbors, plus a ring for connectivity.
func createRandomGraph(rng *rand.Rand, nodeCount int) *benchGraph {
	g := &benchGraph{}

	// Create nodes
	for i := 1; i <= nodeCount; i++ {
		g.nodes = append(g.nodes, routing.GraphNode{
			ID:       int32(i),
			Address:  fmt.Sprintf("localhost:%d", 9000+i),
			LastSeen: time.Now(),
		})
	}

	edgeSet := make(map[string]bool)
	addEdge := func(from, to int32, latency int64) {
		key := fmt.Sprintf("%d-%d", from, to)
		if !edgeSet[key] {
			edgeSet[key] = true
			g.edges = append(g.edges, routing.GraphEdge{
				From:      from,
				To:        to,
				LatencyMs: latency,
			})
		}
	}

	// Random edges: 4-8 per node
	for i := 1; i <= nodeCount; i++ {
		neighbors := 4 + rng.Intn(5)
		for j := 0; j < neighbors; j++ {
			neighbor := int32(rng.Intn(nodeCount)) + 1
			if neighbor == int32(i) {
				continue
			}
			latency := int64(rng.Intn(50)) + 1
			addEdge(int32(i), neighbor, latency)
		}
	}

	// Ring to ensure connectivity
	for i := 1; i <= nodeCount; i++ {
		next := (i % nodeCount) + 1
		latency := int64(rng.Intn(20)) + 5
		addEdge(int32(i), int32(next), latency)
	}

	return g
}
