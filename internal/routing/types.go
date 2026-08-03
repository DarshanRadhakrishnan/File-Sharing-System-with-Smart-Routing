package routing

import (
	"fmt"
	"sync"
	"time"
)

// ============================================================
// Graph interface — defines what the router needs from a graph
// ============================================================

// GraphNode represents a node provided by the graph.
type GraphNode struct {
	ID       int32
	Address  string
	LastSeen time.Time
}

// GraphEdge represents a directed edge with latency.
type GraphEdge struct {
	From      int32
	To        int32
	LatencyMs int64
}

// Graph is the interface that the routing algorithms work against.
// This avoids a circular import between peer and routing packages.
type Graph interface {
	// GetNodeList returns a snapshot of all nodes.
	GetNodeList() []GraphNode
	// GetEdgeList returns a snapshot of all edges.
	GetEdgeList() []GraphEdge
}

// ============================================================
// Route result
// ============================================================

// Route represents a complete shortest path from source to destination.
type Route struct {
	FromPeerID   int32
	ToPeerID     int32
	Path         []int32   // Ordered sequence of peer IDs
	TotalLatency int64     // Total latency in ms (sum of edge weights)
	Hops         int32     // Number of hops (len(Path) - 1)
	CreatedAt    time.Time // When this route was calculated
}

// ============================================================
// Route cache
// ============================================================

// RouteCache stores calculated routes with time-based expiration.
// All methods are safe for concurrent use.
type RouteCache struct {
	routes map[string]*CachedRoute
	mu     sync.RWMutex
	ttl    time.Duration
}

// CachedRoute pairs a route with its expiration time.
type CachedRoute struct {
	Route     *Route
	ExpiresAt time.Time
}

// NewRouteCache creates a new route cache with the specified TTL.
func NewRouteCache(ttl time.Duration) *RouteCache {
	return &RouteCache{
		routes: make(map[string]*CachedRoute),
		ttl:    ttl,
	}
}

// Set stores a route in the cache.
func (rc *RouteCache) Set(route *Route) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	key := cacheKey(route.FromPeerID, route.ToPeerID)
	rc.routes[key] = &CachedRoute{
		Route:     route,
		ExpiresAt: time.Now().Add(rc.ttl),
	}
}

// Get retrieves a cached route. Returns nil if not found or expired.
func (rc *RouteCache) Get(fromID, toID int32) *Route {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	key := cacheKey(fromID, toID)
	cached, exists := rc.routes[key]
	if !exists {
		return nil
	}

	if time.Now().After(cached.ExpiresAt) {
		return nil // Expired
	}

	return cached.Route
}

// Invalidate clears all cached routes and returns the number of entries cleared.
func (rc *RouteCache) Invalidate() int {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	count := len(rc.routes)
	rc.routes = make(map[string]*CachedRoute)
	return count
}

// InvalidatePeer clears all cached routes that pass through the given peer.
func (rc *RouteCache) InvalidatePeer(peerID int32) int {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	var keysToDelete []string
	for key, cached := range rc.routes {
		for _, id := range cached.Route.Path {
			if id == peerID {
				keysToDelete = append(keysToDelete, key)
				break
			}
		}
	}

	for _, key := range keysToDelete {
		delete(rc.routes, key)
	}
	return len(keysToDelete)
}

// Size returns the number of entries currently in the cache (including expired).
func (rc *RouteCache) Size() int {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return len(rc.routes)
}

// Entries returns a snapshot of all non-expired cached routes.
func (rc *RouteCache) Entries() []*Route {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	now := time.Now()
	var routes []*Route
	for _, cached := range rc.routes {
		if now.Before(cached.ExpiresAt) {
			routes = append(routes, cached.Route)
		}
	}
	return routes
}

// TTL returns the cache's time-to-live duration.
func (rc *RouteCache) TTL() time.Duration {
	return rc.ttl
}

// cacheKey generates a deterministic cache key for a (from, to) pair.
func cacheKey(from, to int32) string {
	return fmt.Sprintf("%d->%d", from, to)
}
