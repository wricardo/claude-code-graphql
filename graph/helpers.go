package graph

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/wricardo/claude-code-graphql/internal/store"
)

func storeSessionToGQL(s *store.Session) *Session {
	cwd := s.CWD
	return &Session{
		ID:          s.ID,
		Cwd:         &cwd,
		FirstSeenAt: s.FirstSeenAt,
		LastSeenAt:  s.LastSeenAt,
	}
}

func storeHookToGQL(h *store.Hook) *Hook {
	et := EventType(h.EventType)
	out := &Hook{
		ID:         h.ID,
		SessionID:  h.SessionID,
		EventType:  et,
		RecordedAt: h.RecordedAt,
	}
	if h.ToolName != "" {
		out.ToolName = &h.ToolName
	}
	if h.ToolInput != "" {
		out.ToolInput = &h.ToolInput
	}
	if h.ToolResponse != "" {
		out.ToolResponse = &h.ToolResponse
	}
	if h.Prompt != "" {
		out.Prompt = &h.Prompt
	}
	if h.CWD != "" {
		out.Cwd = &h.CWD
	}
	if h.TranscriptPath != "" {
		out.TranscriptPath = &h.TranscriptPath
	}
	return out
}

func storeHooksToGQL(rows []*store.Hook) []*Hook {
	out := make([]*Hook, len(rows))
	for i, h := range rows {
		out[i] = storeHookToGQL(h)
	}
	return out
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefInt(n *int, def int) int {
	if n == nil {
		return def
	}
	if *n <= 0 {
		return def
	}
	return *n
}

// encodeCursor encodes a recorded_at timestamp and hook id into a base64 cursor.
func encodeCursor(t time.Time, id string) string {
	raw := t.UTC().Format(time.RFC3339Nano) + "|" + id
	return base64.StdEncoding.EncodeToString([]byte(raw))
}

// decodeCursor decodes a base64 cursor into a timestamp and id.
func decodeCursor(cursor string) (time.Time, string, error) {
	b, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return time.Time{}, "", fmt.Errorf("invalid cursor encoding: %w", err)
	}
	parts := strings.SplitN(string(b), "|", 2)
	if len(parts) != 2 {
		return time.Time{}, "", fmt.Errorf("invalid cursor format")
	}
	t, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, "", fmt.Errorf("invalid cursor time: %w", err)
	}
	return t, parts[1], nil
}

// satisfy fmt import used in generated stubs
var _ = fmt.Sprintf

// generateSummaryViaVenu pipes a session digest to venu and returns a 1-2 sentence summary.
func generateSummaryViaVenu(digest string) (string, error) {
	cmd := exec.Command("venu", "Summarize this Claude Code session in 1-2 sentences. Focus on what was accomplished, not the tools used. Be specific about the code/features worked on.")
	cmd.Stdin = strings.NewReader(digest)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("venu: %w", err)
	}
	summary := strings.TrimSpace(out.String())
	if summary == "" {
		return "No summary generated.", nil
	}
	return summary, nil
}
