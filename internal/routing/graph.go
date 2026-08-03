package routing

import (
	"fmt"
)

// GraphAnalyzer provides network analysis operations built on top of the router.
type GraphAnalyzer struct {
	router *DijkstraRouter
}

// NewGraphAnalyzer creates a new graph analyzer.
func NewGraphAnalyzer(router *DijkstraRouter) *GraphAnalyzer {
	return &GraphAnalyzer{router: router}
}

// NetworkStats holds aggregate statistics about the network topology.
type NetworkStats struct {
	TotalPeers     int32
	TotalEdges     int32
	AvgLatency     int64
	MaxLatency     int64
	MinLatency     int64
	NetworkDensity float64 // Percentage of possible connections
}

// AnalyzeNetwork calculates aggregate statistics about the network.
func (ga *GraphAnalyzer) AnalyzeNetwork(graph Graph) *NetworkStats {
	nodes := graph.GetNodeList()
	edges := graph.GetEdgeList()

	stats := &NetworkStats{
		TotalPeers: int32(len(nodes)),
		TotalEdges: int32(len(edges)),
		MaxLatency: 0,
		MinLatency: int64(^uint64(0) >> 1), // MaxInt64
	}

	if stats.TotalEdges == 0 {
		stats.MinLatency = 0
		return stats
	}

	var totalLatency int64
	for _, edge := range edges {
		totalLatency += edge.LatencyMs

		if edge.LatencyMs > stats.MaxLatency {
			stats.MaxLatency = edge.LatencyMs
		}
		if edge.LatencyMs < stats.MinLatency {
			stats.MinLatency = edge.LatencyMs
		}
	}

	stats.AvgLatency = totalLatency / int64(stats.TotalEdges)

	// Density = actual edges / possible directed edges
	possibleEdges := int64(stats.TotalPeers) * int64(stats.TotalPeers-1)
	if possibleEdges > 0 {
		stats.NetworkDensity = float64(stats.TotalEdges) / float64(possibleEdges) * 100.0
	}

	return stats
}

// FindBottlenecks identifies peers whose average edge latency exceeds the threshold.
func (ga *GraphAnalyzer) FindBottlenecks(graph Graph, thresholdMs int64) []int32 {
	edges := graph.GetEdgeList()

	// Accumulate total latency and degree per peer
	peerTotalLatency := make(map[int32]int64)
	peerDegree := make(map[int32]int32)

	for _, edge := range edges {
		peerTotalLatency[edge.From] += edge.LatencyMs
		peerDegree[edge.From]++
		peerTotalLatency[edge.To] += edge.LatencyMs
		peerDegree[edge.To]++
	}

	var bottlenecks []int32
	for peerID, totalLat := range peerTotalLatency {
		degree := peerDegree[peerID]
		if degree > 0 {
			avgLatency := totalLat / int64(degree)
			if avgLatency > thresholdMs {
				bottlenecks = append(bottlenecks, peerID)
			}
		}
	}

	return bottlenecks
}

// GetNetworkMetrics returns a formatted string with network statistics.
func (ga *GraphAnalyzer) GetNetworkMetrics(graph Graph) string {
	stats := ga.AnalyzeNetwork(graph)

	return fmt.Sprintf(`
╔════════════════════════════════════╗
║       NETWORK METRICS              ║
╚════════════════════════════════════╝

  Total Peers:        %d
  Total Connections:  %d
  Network Density:    %.2f%%
  Avg Latency:        %dms
  Min Latency:        %dms
  Max Latency:        %dms
  Route Cache Size:   %d
`,
		stats.TotalPeers,
		stats.TotalEdges,
		stats.NetworkDensity,
		stats.AvgLatency,
		stats.MinLatency,
		stats.MaxLatency,
		ga.router.cache.Size(),
	)
}
