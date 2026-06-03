package graph

import (
	"context"
	"sync"

	"github.com/wricardo/claude-code-graphql/internal/store"
)

type sessionCacheKey struct{}

// SessionDataCache is a request-scoped prefetch cache for session computed fields.
// It is populated once per request by the Sessions resolver and read by field resolvers,
// eliminating N+1 queries when listing multiple sessions.
type SessionDataCache struct {
	mu         sync.RWMutex
	toolUsage  map[string][]*store.ToolStat
	skillsUsed map[string][]*store.SkillUsage
	errors     map[string][]*store.ToolError
	durations  map[string]*float64
	hookCounts map[string]int
}

func newSessionDataCache() *SessionDataCache {
	return &SessionDataCache{
		toolUsage:  make(map[string][]*store.ToolStat),
		skillsUsed: make(map[string][]*store.SkillUsage),
		errors:     make(map[string][]*store.ToolError),
		durations:  make(map[string]*float64),
		hookCounts: make(map[string]int),
	}
}

// WithSessionCache returns a new context with a fresh SessionDataCache attached.
// Call this in HTTP middleware before serving each GraphQL request.
func WithSessionCache(ctx context.Context) context.Context {
	return context.WithValue(ctx, sessionCacheKey{}, newSessionDataCache())
}

func sessionCacheFromCtx(ctx context.Context) *SessionDataCache {
	c, _ := ctx.Value(sessionCacheKey{}).(*SessionDataCache)
	return c
}

// prefetch runs all batch queries for the given session IDs and populates the cache.
// Every requested ID gets an entry (possibly empty/zero) so field resolvers know
// the data was fetched and don't fall back to individual queries.
func (c *SessionDataCache) prefetch(s *store.Store, ids []string) {
	if len(ids) == 0 {
		return
	}

	toolUsage, _ := s.BatchGetToolUsage(ids)
	skillsUsed, _ := s.BatchGetSkillsUsed(ids)
	errors, _ := s.BatchGetErrors(ids)
	durations, _ := s.BatchGetDuration(ids)
	hookCounts, _ := s.BatchCountHooks(ids)

	c.mu.Lock()
	defer c.mu.Unlock()
	// Write an entry for every requested ID so resolvers know the data was fetched.
	// A nil slice / zero value means the session has no data for that field.
	for _, id := range ids {
		c.toolUsage[id] = toolUsage[id]
		c.skillsUsed[id] = skillsUsed[id]
		c.errors[id] = errors[id]
		c.durations[id] = durations[id]
		c.hookCounts[id] = hookCounts[id]
	}
}

func (c *SessionDataCache) getToolUsage(id string) ([]*store.ToolStat, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.toolUsage[id]
	return v, ok
}

func (c *SessionDataCache) getSkillsUsed(id string) ([]*store.SkillUsage, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.skillsUsed[id]
	return v, ok
}

func (c *SessionDataCache) getErrors(id string) ([]*store.ToolError, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.errors[id]
	return v, ok
}

func (c *SessionDataCache) getDuration(id string) (*float64, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.durations[id]
	return v, ok
}

func (c *SessionDataCache) getHookCount(id string) (int, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.hookCounts[id]
	return v, ok
}
