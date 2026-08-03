package routing

import (
	"fmt"
	"log"
	"math"
	"time"
)

// DijkstraRouter implements shortest-path finding with caching.
type DijkstraRouter struct {
	cache *RouteCache
}

// NewDijkstraRouter creates a new router with a 1-hour route cache TTL.
func NewDijkstraRouter() *DijkstraRouter {
	return &DijkstraRouter{
		cache: NewRouteCache(1 * time.Hour),
	}
}

// CalculateRoute finds the shortest path between two peers.
// It checks the cache first; on a miss it runs Dijkstra's algorithm.
func (dr *DijkstraRouter) CalculateRoute(graph Graph, fromID, toID int32) (*Route, error) {
	if fromID == toID {
		return &Route{
			FromPeerID:   fromID,
			ToPeerID:     toID,
			Path:         []int32{fromID},
			TotalLatency: 0,
			Hops:         0,
			CreatedAt:    time.Now(),
		}, nil
	}

	// Check cache first
	if cached := dr.cache.Get(fromID, toID); cached != nil {
		log.Printf("[Router] Cache hit: %d → %d (latency: %dms, hops: %d)\n",
			fromID, toID, cached.TotalLatency, cached.Hops)
		return cached, nil
	}

	// Run Dijkstra
	path, latency, err := dr.dijkstra(graph, fromID, toID)
	if err != nil {
		return nil, err
	}

	route := &Route{
		FromPeerID:   fromID,
		ToPeerID:     toID,
		Path:         path,
		TotalLatency: latency,
		Hops:         int32(len(path) - 1),
		CreatedAt:    time.Now(),
	}

	dr.cache.Set(route)
	log.Printf("[Router] Calculated route: %d → %d (latency: %dms, hops: %d, path: %v)\n",
		fromID, toID, latency, route.Hops, path)

	return route, nil
}

// dijkstra implements Dijkstra's shortest path algorithm using O(V²) scan.
// Returns: path (sequence of peer IDs), total latency, error.
func (dr *DijkstraRouter) dijkstra(graph Graph, source, target int32) ([]int32, int64, error) {
	nodes := graph.GetNodeList()
	edges := graph.GetEdgeList()

	// Build adjacency list from edges
	adjacency := make(map[int32][]GraphEdge)
	for _, e := range edges {
		adjacency[e.From] = append(adjacency[e.From], e)
	}

	// Initialize distances
	const inf = int64(math.MaxInt64)
	distances := make(map[int32]int64)
	previous := make(map[int32]int32)
	unvisited := make(map[int32]bool)

	for _, n := range nodes {
		distances[n.ID] = inf
		previous[n.ID] = -1
		unvisited[n.ID] = true
	}

	// Ensure source and target exist in the graph
	if _, exists := distances[source]; !exists {
		return nil, 0, fmt.Errorf("source peer %d not found in graph", source)
	}
	if _, exists := distances[target]; !exists {
		return nil, 0, fmt.Errorf("target peer %d not found in graph", target)
	}

	distances[source] = 0

	// Main Dijkstra loop: O(V²)
	for len(unvisited) > 0 {
		// Find unvisited node with minimum distance
		currentID := int32(-1)
		minDist := inf

		for nodeID := range unvisited {
			if distances[nodeID] < minDist {
				minDist = distances[nodeID]
				currentID = nodeID
			}
		}

		if currentID == -1 || minDist == inf {
			break // No more reachable nodes
		}

		// Early termination: we've settled the target
		if currentID == target {
			break
		}

		delete(unvisited, currentID)

		// Relax edges from current node
		for _, edge := range adjacency[currentID] {
			if !unvisited[edge.To] {
				continue // Already settled
			}
			newDist := distances[currentID] + edge.LatencyMs
			if newDist < distances[edge.To] {
				distances[edge.To] = newDist
				previous[edge.To] = currentID
			}
		}
	}

	// Check if target was reached
	if distances[target] == inf {
		return nil, 0, fmt.Errorf("no path found from peer %d to peer %d", source, target)
	}

	// Reconstruct path by walking backwards from target
	var path []int32
	current := target
	for current != -1 {
		path = append([]int32{current}, path...)
		current = previous[current]
	}

	return path, distances[target], nil
}

// GetCache returns the underlying route cache for inspection.
func (dr *DijkstraRouter) GetCache() *RouteCache {
	return dr.cache
}

// InvalidateCache clears all cached routes.
func (dr *DijkstraRouter) InvalidateCache() int {
	return dr.cache.Invalidate()
}

// InvalidatePeerRoutes clears routes involving a specific peer.
func (dr *DijkstraRouter) InvalidatePeerRoutes(peerID int32) int {
	return dr.cache.InvalidatePeer(peerID)
}
