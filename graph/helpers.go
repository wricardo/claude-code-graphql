package graph

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/wricardo/claude-code-graphql/internal/claude"
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
	if h.ToolUseID != "" {
		out.ToolUseID = &h.ToolUseID
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

// claudeMsgsToGQL converts a slice of claude.TranscriptMessage to GraphQL TranscriptMessage pointers,
// applying limit/offset pagination and populating all available fields.
func claudeMsgsToGQL(msgs []claude.TranscriptMessage, limit, offset int) []*TranscriptMessage {
	if offset > len(msgs) {
		return nil
	}
	msgs = msgs[offset:]
	if limit > 0 && limit < len(msgs) {
		msgs = msgs[:limit]
	}

	out := make([]*TranscriptMessage, len(msgs))
	for i, m := range msgs {
		tm := &TranscriptMessage{
			Type:        m.Type,
			Raw:         string(m.Raw),
			IsSidechain: m.IsSidechain,
		}
		if m.UUID != "" {
			tm.UUID = &m.UUID
		}
		if m.ParentUUID != "" {
			tm.ParentUUID = &m.ParentUUID
		}
		if m.Timestamp != "" {
			tm.Timestamp = &m.Timestamp
		}
		if m.Message.Role != "" {
			tm.Role = &m.Message.Role
		}
		if m.Message.Content != nil {
			switch v := m.Message.Content.(type) {
			case string:
				tm.Content = &v
			default:
				b, _ := json.Marshal(v)
				s := string(b)
				tm.Content = &s
			}
		}
		out[i] = tm
	}
	return out
}

// loadSessionTranscript finds the project for a session and reads its transcript.
func (r *sessionResolver) loadSessionTranscript(sessionID string) ([]claude.TranscriptMessage, error) {
	encodedProject, found := claude.FindProjectForSession(r.ClaudeDir, sessionID)
	if !found {
		return nil, nil
	}
	return claude.ReadTranscript(r.ClaudeDir, encodedProject, sessionID)
}

// satisfy fmt import used in generated stubs
var _ = fmt.Sprintf

// generateSessionSummary pipes a session digest to an LLM CLI and returns a 1-2 sentence summary.
// The CLI must read a prompt from its first argument and session data from stdin.
// Replace the command name and args to use your preferred LLM CLI.
func generateSessionSummary(digest string) (string, error) {
	cmd := exec.Command("llm", "Summarize this Claude Code session in 1-2 sentences. Focus on what was accomplished, not the tools used. Be specific about the code/features worked on.")
	cmd.Stdin = strings.NewReader(digest)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("llm: %w", err)
	}
	summary := strings.TrimSpace(out.String())
	if summary == "" {
		return "No summary generated.", nil
	}
	return summary, nil
}
