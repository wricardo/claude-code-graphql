package graph

import "github.com/wricardo/claude-code-graphql/internal/store"

// Resolver is the root GraphQL resolver.
type Resolver struct {
	Store    *store.Store
	ClaudeDir string // path to ~/.claude
}
