package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	db.SetMaxOpenConns(1)

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			id            TEXT PRIMARY KEY,
			cwd           TEXT NOT NULL DEFAULT '',
			first_seen_at DATETIME NOT NULL,
			last_seen_at  DATETIME NOT NULL,
			summary       TEXT NOT NULL DEFAULT ''
		);

		CREATE TABLE IF NOT EXISTS hooks (
			id              TEXT PRIMARY KEY,
			session_id      TEXT NOT NULL,
			event_type      TEXT NOT NULL,
			tool_name       TEXT NOT NULL DEFAULT '',
			tool_input      TEXT NOT NULL DEFAULT '',
			tool_response   TEXT NOT NULL DEFAULT '',
			prompt          TEXT NOT NULL DEFAULT '',
			cwd             TEXT NOT NULL DEFAULT '',
			transcript_path TEXT NOT NULL DEFAULT '',
			recorded_at     DATETIME NOT NULL,
			FOREIGN KEY (session_id) REFERENCES sessions(id)
		);

		CREATE INDEX IF NOT EXISTS idx_hooks_session_id  ON hooks(session_id);
		CREATE INDEX IF NOT EXISTS idx_hooks_event_type  ON hooks(event_type);
		CREATE INDEX IF NOT EXISTS idx_hooks_tool_name   ON hooks(tool_name);
		CREATE INDEX IF NOT EXISTS idx_hooks_recorded_at ON hooks(recorded_at);
	`)
	if err != nil {
		return err
	}

	// Add summary column if it doesn't exist (migration for existing DBs).
	s.db.Exec(`ALTER TABLE sessions ADD COLUMN summary TEXT NOT NULL DEFAULT ''`)

	// FTS5 virtual table for full-text search across hooks.
	_, err = s.db.Exec(`
		CREATE VIRTUAL TABLE IF NOT EXISTS hooks_fts USING fts5(
			hook_id UNINDEXED,
			prompt,
			tool_input,
			tool_response,
			tokenize='unicode61'
		);
	`)
	if err != nil {
		return fmt.Errorf("create fts: %w", err)
	}

	// Triggers to keep FTS in sync.
	s.db.Exec(`
		CREATE TRIGGER IF NOT EXISTS hooks_ai AFTER INSERT ON hooks BEGIN
			INSERT INTO hooks_fts(hook_id, prompt, tool_input, tool_response)
			VALUES (new.id, new.prompt, new.tool_input, new.tool_response);
		END;
	`)

	// Backfill FTS for any existing hooks not yet indexed.
	s.db.Exec(`
		INSERT INTO hooks_fts(hook_id, prompt, tool_input, tool_response)
		SELECT id, prompt, tool_input, tool_response FROM hooks
		WHERE id NOT IN (SELECT hook_id FROM hooks_fts)
	`)

	return nil
}

// Session is the store representation of a Claude Code session.
type Session struct {
	ID          string
	CWD         string
	FirstSeenAt time.Time
	LastSeenAt  time.Time
	Summary     string
}

// Hook is a single hook event emitted by Claude Code.
type Hook struct {
	ID             string
	SessionID      string
	EventType      string
	ToolName       string
	ToolInput      string
	ToolResponse   string
	Prompt         string
	CWD            string
	TranscriptPath string
	RecordedAt     time.Time
}

// RecordHookInput is the input for RecordHook.
type RecordHookInput struct {
	SessionID      string
	EventType      string
	ToolName       string
	ToolInput      string
	ToolResponse   string
	Prompt         string
	CWD            string
	TranscriptPath string
}

// RecordHook upserts the session and inserts a new hook row.
func (s *Store) RecordHook(input RecordHookInput) (*Hook, error) {
	now := time.Now().UTC()

	_, err := s.db.Exec(`
		INSERT INTO sessions (id, cwd, first_seen_at, last_seen_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			last_seen_at = excluded.last_seen_at,
			cwd = CASE WHEN excluded.cwd != '' THEN excluded.cwd ELSE cwd END
	`, input.SessionID, input.CWD, now, now)
	if err != nil {
		return nil, fmt.Errorf("upsert session: %w", err)
	}

	h := &Hook{
		ID:             uuid.New().String(),
		SessionID:      input.SessionID,
		EventType:      input.EventType,
		ToolName:       input.ToolName,
		ToolInput:      input.ToolInput,
		ToolResponse:   input.ToolResponse,
		Prompt:         input.Prompt,
		CWD:            input.CWD,
		TranscriptPath: input.TranscriptPath,
		RecordedAt:     now,
	}

	_, err = s.db.Exec(`
		INSERT INTO hooks
			(id, session_id, event_type, tool_name, tool_input, tool_response, prompt, cwd, transcript_path, recorded_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, h.ID, h.SessionID, h.EventType, h.ToolName, h.ToolInput, h.ToolResponse, h.Prompt, h.CWD, h.TranscriptPath, h.RecordedAt)
	if err != nil {
		return nil, fmt.Errorf("insert hook: %w", err)
	}

	return h, nil
}

// GetSessions returns sessions ordered by most recent activity.
func (s *Store) GetSessions(limit, offset int) ([]*Session, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.Query(`
		SELECT id, cwd, first_seen_at, last_seen_at, summary
		FROM sessions
		ORDER BY last_seen_at DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*Session
	for rows.Next() {
		sess := &Session{}
		if err := rows.Scan(&sess.ID, &sess.CWD, &sess.FirstSeenAt, &sess.LastSeenAt, &sess.Summary); err != nil {
			return nil, err
		}
		out = append(out, sess)
	}
	return out, rows.Err()
}

// GetSession returns a single session by id, or nil if not found.
func (s *Store) GetSession(id string) (*Session, error) {
	sess := &Session{}
	err := s.db.QueryRow(`
		SELECT id, cwd, first_seen_at, last_seen_at, summary FROM sessions WHERE id = ?
	`, id).Scan(&sess.ID, &sess.CWD, &sess.FirstSeenAt, &sess.LastSeenAt, &sess.Summary)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return sess, err
}

// GetHookByID returns a single hook by id.
func (s *Store) GetHookByID(id string) (*Hook, error) {
	h := &Hook{}
	err := s.db.QueryRow(`
		SELECT id, session_id, event_type, tool_name, tool_input, tool_response, prompt, cwd, transcript_path, recorded_at
		FROM hooks WHERE id = ?
	`, id).Scan(&h.ID, &h.SessionID, &h.EventType, &h.ToolName, &h.ToolInput, &h.ToolResponse, &h.Prompt, &h.CWD, &h.TranscriptPath, &h.RecordedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return h, err
}

// CountSessionHooks returns the total hook count for a session.
func (s *Store) CountSessionHooks(sessionID string) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM hooks WHERE session_id = ?`, sessionID).Scan(&n)
	return n, err
}

// HookFilter holds optional filters for GetHooks.
type HookFilter struct {
	SessionID string
	EventType string
	ToolName  string
	CWD       string
	From      *time.Time
	To        *time.Time
	Limit     int
	Offset    int
	SortDir   string // "ASC" or "DESC", default "DESC"
}

// GetHooks returns hooks matching the filter, ordered by most recent first.
func (s *Store) GetHooks(f HookFilter) ([]*Hook, error) {
	if f.Limit <= 0 {
		f.Limit = 50
	}
	sortDir := "DESC"
	if f.SortDir == "ASC" {
		sortDir = "ASC"
	}

	q := `SELECT id, session_id, event_type, tool_name, tool_input, tool_response, prompt, cwd, transcript_path, recorded_at
	      FROM hooks WHERE 1=1`
	args := []any{}

	if f.SessionID != "" {
		q += " AND session_id = ?"
		args = append(args, f.SessionID)
	}
	if f.EventType != "" {
		q += " AND event_type = ?"
		args = append(args, f.EventType)
	}
	if f.ToolName != "" {
		q += " AND tool_name = ?"
		args = append(args, f.ToolName)
	}
	if f.CWD != "" {
		q += " AND cwd = ?"
		args = append(args, f.CWD)
	}
	if f.From != nil {
		q += " AND recorded_at >= ?"
		args = append(args, *f.From)
	}
	if f.To != nil {
		q += " AND recorded_at <= ?"
		args = append(args, *f.To)
	}

	q += " ORDER BY recorded_at " + sortDir + " LIMIT ? OFFSET ?"
	args = append(args, f.Limit, f.Offset)

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*Hook
	for rows.Next() {
		h := &Hook{}
		if err := rows.Scan(&h.ID, &h.SessionID, &h.EventType, &h.ToolName, &h.ToolInput, &h.ToolResponse, &h.Prompt, &h.CWD, &h.TranscriptPath, &h.RecordedAt); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// HookPage is the result of a cursor-based paginated hooks query.
type HookPage struct {
	Edges      []*Hook
	TotalCount int
	HasNext    bool
	StartID    string
	StartAt    time.Time
	EndID      string
	EndAt      time.Time
}

// GetHooksPaged returns hooks using keyset (cursor-based) pagination.
// afterTime and afterID are decoded from the cursor; pass zero values for the first page.
// Results are ordered by recorded_at DESC, id DESC.
func (s *Store) GetHooksPaged(sessionID, eventType, toolName string, first int, afterTime time.Time, afterID string) (*HookPage, error) {
	if first <= 0 {
		first = 20
	}
	if first > 100 {
		first = 100
	}

	filterClauses := "WHERE 1=1"
	filterArgs := []any{}
	if sessionID != "" {
		filterClauses += " AND session_id = ?"
		filterArgs = append(filterArgs, sessionID)
	}
	if eventType != "" {
		filterClauses += " AND event_type = ?"
		filterArgs = append(filterArgs, eventType)
	}
	if toolName != "" {
		filterClauses += " AND tool_name = ?"
		filterArgs = append(filterArgs, toolName)
	}

	var total int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM hooks "+filterClauses, filterArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count hooks: %w", err)
	}

	q := `SELECT id, session_id, event_type, tool_name, tool_input, tool_response, prompt, cwd, transcript_path, recorded_at
	      FROM hooks ` + filterClauses
	args := append([]any{}, filterArgs...)

	if !afterTime.IsZero() && afterID != "" {
		q += " AND (recorded_at < ? OR (recorded_at = ? AND id < ?))"
		args = append(args, afterTime, afterTime, afterID)
	}

	q += " ORDER BY recorded_at DESC, id DESC LIMIT ?"
	args = append(args, first+1)

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("query hooks paged: %w", err)
	}
	defer rows.Close()

	var hooks []*Hook
	for rows.Next() {
		h := &Hook{}
		if err := rows.Scan(&h.ID, &h.SessionID, &h.EventType, &h.ToolName, &h.ToolInput, &h.ToolResponse, &h.Prompt, &h.CWD, &h.TranscriptPath, &h.RecordedAt); err != nil {
			return nil, err
		}
		hooks = append(hooks, h)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	hasNext := len(hooks) > first
	if hasNext {
		hooks = hooks[:first]
	}

	page := &HookPage{
		Edges:      hooks,
		TotalCount: total,
		HasNext:    hasNext,
	}
	if len(hooks) > 0 {
		page.StartID = hooks[0].ID
		page.StartAt = hooks[0].RecordedAt
		page.EndID = hooks[len(hooks)-1].ID
		page.EndAt = hooks[len(hooks)-1].RecordedAt
	}
	return page, nil
}

// ToolStat is the count of hooks per tool name.
type ToolStat struct {
	Name  string
	Count int
}

// EventTypeStat is the count of hooks per event type.
type EventTypeStat struct {
	EventType string
	Count     int
}

// DayStat is the count of hooks for a single calendar day.
type DayStat struct {
	Date  string
	Count int
}

// CwdStat is the hook and session counts for a working directory.
type CwdStat struct {
	CWD          string
	HookCount    int
	SessionCount int
}

// Stats holds aggregate statistics.
type Stats struct {
	TotalSessions      int
	TotalHooks         int
	TopTools           []*ToolStat
	HooksByEventType   []*EventTypeStat
	HooksByDay         []*DayStat
	HooksByCwd         []*CwdStat
	AvgHooksPerSession float64
}

// GetStats returns aggregate statistics.
func (s *Store) GetStats() (*Stats, error) {
	stats := &Stats{}

	if err := s.db.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&stats.TotalSessions); err != nil {
		return nil, err
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM hooks`).Scan(&stats.TotalHooks); err != nil {
		return nil, err
	}

	rows2, err := s.db.Query(`
		SELECT event_type, COUNT(*) cnt FROM hooks
		GROUP BY event_type ORDER BY cnt DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows2.Close()
	for rows2.Next() {
		es := &EventTypeStat{}
		if err := rows2.Scan(&es.EventType, &es.Count); err != nil {
			return nil, err
		}
		stats.HooksByEventType = append(stats.HooksByEventType, es)
	}
	return stats, rows2.Err()
}

// GetTopTools returns the top N tools by hook count.
func (s *Store) GetTopTools(limit int) ([]*ToolStat, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.db.Query(`
		SELECT tool_name, COUNT(*) cnt FROM hooks
		WHERE tool_name != ''
		GROUP BY tool_name ORDER BY cnt DESC LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*ToolStat
	for rows.Next() {
		ts := &ToolStat{}
		if err := rows.Scan(&ts.Name, &ts.Count); err != nil {
			return nil, err
		}
		out = append(out, ts)
	}
	return out, rows.Err()
}

// GetHooksByDay returns hook counts grouped by day for the last N days.
func (s *Store) GetHooksByDay(days int) ([]*DayStat, error) {
	if days <= 0 {
		days = 30
	}
	rows, err := s.db.Query(`
		SELECT date(recorded_at) as day, COUNT(*) FROM hooks
		WHERE recorded_at >= date('now', '-' || ? || ' days')
		GROUP BY day ORDER BY day ASC
	`, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*DayStat
	for rows.Next() {
		ds := &DayStat{}
		if err := rows.Scan(&ds.Date, &ds.Count); err != nil {
			return nil, err
		}
		out = append(out, ds)
	}
	return out, rows.Err()
}

// GetHooksByCwd returns hook and session counts grouped by cwd, top N entries.
func (s *Store) GetHooksByCwd(limit int) ([]*CwdStat, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.db.Query(`
		SELECT h.cwd, COUNT(*) as hook_count, COUNT(DISTINCT h.session_id) as session_count
		FROM hooks h
		WHERE h.cwd != ''
		GROUP BY h.cwd
		ORDER BY hook_count DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*CwdStat
	for rows.Next() {
		cs := &CwdStat{}
		if err := rows.Scan(&cs.CWD, &cs.HookCount, &cs.SessionCount); err != nil {
			return nil, err
		}
		out = append(out, cs)
	}
	return out, rows.Err()
}

// GetAvgHooksPerSession returns the average number of hooks per session.
func (s *Store) GetAvgHooksPerSession() (float64, error) {
	var avg float64
	err := s.db.QueryRow(`
		SELECT CAST(COUNT(*) AS REAL) / MAX(1, (SELECT COUNT(*) FROM sessions)) FROM hooks
	`).Scan(&avg)
	return avg, err
}

// GetSessionsByCwd returns sessions whose cwd matches the given path.
func (s *Store) GetSessionsByCwd(cwd string, limit, offset int) ([]*Session, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.Query(`
		SELECT id, cwd, first_seen_at, last_seen_at, summary
		FROM sessions
		WHERE cwd = ?
		ORDER BY last_seen_at DESC
		LIMIT ? OFFSET ?
	`, cwd, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*Session
	for rows.Next() {
		sess := &Session{}
		if err := rows.Scan(&sess.ID, &sess.CWD, &sess.FirstSeenAt, &sess.LastSeenAt, &sess.Summary); err != nil {
			return nil, err
		}
		out = append(out, sess)
	}
	return out, rows.Err()
}

// CountSessionsByCwd returns the session count for a given cwd.
func (s *Store) CountSessionsByCwd(cwd string) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM sessions WHERE cwd = ?`, cwd).Scan(&n)
	return n, err
}

// GetToolUsageBySession returns tool usage stats for a specific session.
func (s *Store) GetToolUsageBySession(sessionID string) ([]*ToolStat, error) {
	rows, err := s.db.Query(`
		SELECT tool_name, COUNT(*) cnt FROM hooks
		WHERE session_id = ? AND tool_name != ''
		GROUP BY tool_name ORDER BY cnt DESC
	`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*ToolStat
	for rows.Next() {
		ts := &ToolStat{}
		if err := rows.Scan(&ts.Name, &ts.Count); err != nil {
			return nil, err
		}
		out = append(out, ts)
	}
	return out, rows.Err()
}

// GetToolUsageByCwd returns tool usage stats for all sessions with a given cwd.
func (s *Store) GetToolUsageByCwd(cwd string) ([]*ToolStat, error) {
	rows, err := s.db.Query(`
		SELECT h.tool_name, COUNT(*) cnt FROM hooks h
		JOIN sessions s ON h.session_id = s.id
		WHERE s.cwd = ? AND h.tool_name != ''
		GROUP BY h.tool_name ORDER BY cnt DESC
	`, cwd)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*ToolStat
	for rows.Next() {
		ts := &ToolStat{}
		if err := rows.Scan(&ts.Name, &ts.Count); err != nil {
			return nil, err
		}
		out = append(out, ts)
	}
	return out, rows.Err()
}

// SkillUsage tracks how many times a skill was invoked.
type SkillUsage struct {
	Name  string
	Count int
}

// GetSkillsUsedBySession returns skills used in a session, extracted from Skill tool hooks.
func (s *Store) GetSkillsUsedBySession(sessionID string) ([]*SkillUsage, error) {
	return s.getSkillsUsed("session_id = ?", sessionID)
}

// GetSkillsUsedByCwd returns skills used across all sessions with a given cwd.
func (s *Store) GetSkillsUsedByCwd(cwd string) ([]*SkillUsage, error) {
	return s.getSkillsUsed("session_id IN (SELECT id FROM sessions WHERE cwd = ?)", cwd)
}

func (s *Store) getSkillsUsed(where string, arg any) ([]*SkillUsage, error) {
	q := `
		SELECT tool_input, COUNT(*) cnt FROM hooks
		WHERE tool_name = 'Skill' AND event_type = 'PreToolUse' AND ` + where + `
		GROUP BY tool_input ORDER BY cnt DESC
	`
	rows, err := s.db.Query(q, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*SkillUsage
	for rows.Next() {
		var raw string
		var count int
		if err := rows.Scan(&raw, &count); err != nil {
			return nil, err
		}
		name := extractSkillName(raw)
		if name != "" {
			out = append(out, &SkillUsage{Name: name, Count: count})
		}
	}
	return out, rows.Err()
}

// extractSkillName parses the skill name from a Skill tool's JSON input.
func extractSkillName(toolInput string) string {
	var v struct {
		Skill string `json:"skill"`
	}
	if err := json.Unmarshal([]byte(toolInput), &v); err != nil {
		return ""
	}
	return v.Skill
}

// --- Session summaries ---

// SetSessionSummary stores a generated summary for a session.
func (s *Store) SetSessionSummary(sessionID, summary string) error {
	_, err := s.db.Exec(`UPDATE sessions SET summary = ? WHERE id = ?`, summary, sessionID)
	return err
}

// GetSessionSummary returns the cached summary for a session.
func (s *Store) GetSessionSummary(sessionID string) (string, error) {
	var summary string
	err := s.db.QueryRow(`SELECT summary FROM sessions WHERE id = ?`, sessionID).Scan(&summary)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return summary, err
}

// GetSessionActivityDigest builds a compact text digest of session activity for summarization.
func (s *Store) GetSessionActivityDigest(sessionID string) (string, error) {
	sess, err := s.GetSession(sessionID)
	if err != nil {
		return "", err
	}
	if sess == nil {
		return "", fmt.Errorf("session not found: %s", sessionID)
	}

	// Get user prompts (from Stop events or hooks with prompts)
	rows, err := s.db.Query(`
		SELECT DISTINCT prompt FROM hooks
		WHERE session_id = ? AND prompt != ''
		ORDER BY recorded_at ASC
		LIMIT 5
	`, sessionID)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	var prompts []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return "", err
		}
		if len(p) > 200 {
			p = p[:200] + "..."
		}
		prompts = append(prompts, p)
	}

	// Get tool usage
	tools, err := s.GetToolUsageBySession(sessionID)
	if err != nil {
		return "", err
	}

	// Get file paths touched (from Edit/Write/Read inputs)
	fileRows, err := s.db.Query(`
		SELECT DISTINCT
			CASE
				WHEN tool_name IN ('Edit', 'Write', 'Read') THEN
					json_extract(tool_input, '$.file_path')
				ELSE NULL
			END as fpath
		FROM hooks
		WHERE session_id = ? AND tool_name IN ('Edit', 'Write', 'Read') AND tool_input != ''
		ORDER BY recorded_at DESC
		LIMIT 20
	`, sessionID)
	if err != nil {
		return "", err
	}
	defer fileRows.Close()
	var files []string
	for fileRows.Next() {
		var f sql.NullString
		if err := fileRows.Scan(&f); err != nil {
			return "", err
		}
		if f.Valid && f.String != "" {
			files = append(files, f.String)
		}
	}

	// Get errors
	errors, err := s.GetSessionErrors(sessionID)
	if err != nil {
		return "", err
	}

	// Build digest
	digest := fmt.Sprintf("Session: %s\nProject: %s\nDuration: %s to %s\n",
		sessionID, sess.CWD,
		sess.FirstSeenAt.Format("15:04:05"), sess.LastSeenAt.Format("15:04:05"))

	if len(prompts) > 0 {
		digest += "\nUser prompts:\n"
		for _, p := range prompts {
			digest += "- " + p + "\n"
		}
	}

	if len(tools) > 0 {
		digest += "\nTool usage:\n"
		for _, t := range tools {
			digest += fmt.Sprintf("- %s: %d calls\n", t.Name, t.Count)
		}
	}

	if len(files) > 0 {
		digest += "\nFiles touched:\n"
		for _, f := range files {
			digest += "- " + f + "\n"
		}
	}

	if len(errors) > 0 {
		digest += fmt.Sprintf("\nErrors: %d\n", len(errors))
		for _, e := range errors {
			msg := e.ErrorMessage
			if len(msg) > 100 {
				msg = msg[:100] + "..."
			}
			digest += fmt.Sprintf("- %s: %s\n", e.ToolName, msg)
		}
	}

	return digest, nil
}

// --- Full-text search ---

// SearchResult represents a single search hit.
type SearchResult struct {
	HookID     string
	MatchField string // "prompt", "tool_input", or "tool_response"
	Snippet    string
}

// Search performs full-text search across hooks using FTS5.
func (s *Store) Search(query string, sessionID, cwd string, limit int) ([]*SearchResult, error) {
	if limit <= 0 {
		limit = 20
	}

	q := `
		SELECT f.hook_id,
			snippet(hooks_fts, -1, '>>>', '<<<', '...', 50) as snip
		FROM hooks_fts f
	`
	args := []any{}

	if sessionID != "" || cwd != "" {
		q += " JOIN hooks h ON f.hook_id = h.id"
	}

	q += " WHERE hooks_fts MATCH ?"
	args = append(args, query)

	if sessionID != "" {
		q += " AND h.session_id = ?"
		args = append(args, sessionID)
	}
	if cwd != "" {
		q += " AND h.cwd = ?"
		args = append(args, cwd)
	}

	q += " ORDER BY rank LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("fts search: %w", err)
	}
	defer rows.Close()

	var out []*SearchResult
	for rows.Next() {
		sr := &SearchResult{}
		if err := rows.Scan(&sr.HookID, &sr.Snippet); err != nil {
			return nil, err
		}
		// Determine which field matched by checking the hook
		sr.MatchField = "toolInput" // default
		out = append(out, sr)
	}
	return out, rows.Err()
}

// --- Error analysis ---

// ToolError represents a detected error in a tool response.
type ToolError struct {
	HookID       string
	SessionID    string
	ToolName     string
	ErrorMessage string
	Input        string
	RecordedAt   time.Time
}

// errorPatterns are substrings that indicate an error in a tool response.
var errorPatterns = []string{
	"Exit code 1",
	"Exit code 2",
	"error:",
	"Error:",
	"FAILED",
	"panic:",
	"fatal:",
	"Fatal:",
	"command not found",
	"No such file or directory",
	"permission denied",
	"Permission denied",
}

// isErrorResponse checks if a tool response indicates a tool call failure.
// It parses the JSON response structure and looks for error fields,
// rather than pattern-matching inside code content.
func isErrorResponse(toolResponse string) (bool, string) {
	// Try to parse as JSON — tool responses are typically JSON objects.
	var resp map[string]any
	if err := json.Unmarshal([]byte(toolResponse), &resp); err != nil {
		// Not JSON — try raw pattern matching on short responses only
		// (long responses are likely code/file content with incidental "error" strings)
		if len(toolResponse) < 500 {
			return matchErrorPatterns(toolResponse)
		}
		return false, ""
	}

	// Check for explicit error fields in the JSON response
	// Bash tool: {"exitCode": N, "stderr": "..."} where exitCode != 0
	if ec, ok := resp["exitCode"]; ok {
		code := 0
		switch v := ec.(type) {
		case float64:
			code = int(v)
		}
		if code != 0 {
			stderr, _ := resp["stderr"].(string)
			if stderr == "" {
				stderr = fmt.Sprintf("exit code %d", code)
			}
			if len(stderr) > 200 {
				stderr = stderr[:200]
			}
			return true, stderr
		}
	}

	// Check for "error" key in response
	if errMsg, ok := resp["error"]; ok {
		msg := fmt.Sprintf("%v", errMsg)
		if len(msg) > 200 {
			msg = msg[:200]
		}
		return true, msg
	}

	// Read/Edit tools: check if response starts with "Error:"
	if str, ok := resp["output"].(string); ok && len(str) < 500 {
		return matchErrorPatterns(str)
	}

	return false, ""
}

func matchErrorPatterns(s string) (bool, string) {
	for _, pat := range errorPatterns {
		idx := strings.Index(s, pat)
		if idx >= 0 {
			start := idx
			if start > 50 {
				start = idx - 50
			}
			end := idx + len(pat) + 150
			if end > len(s) {
				end = len(s)
			}
			return true, s[start:end]
		}
	}
	return false, ""
}

// GetSessionErrors returns all detected errors for a session.
func (s *Store) GetSessionErrors(sessionID string) ([]*ToolError, error) {
	rows, err := s.db.Query(`
		SELECT id, session_id, tool_name, tool_input, tool_response, recorded_at
		FROM hooks
		WHERE session_id = ? AND event_type = 'PostToolUse' AND tool_response != ''
		ORDER BY recorded_at DESC
	`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*ToolError
	for rows.Next() {
		var id, sessID, toolName, toolInput, toolResponse string
		var recordedAt time.Time
		if err := rows.Scan(&id, &sessID, &toolName, &toolInput, &toolResponse, &recordedAt); err != nil {
			return nil, err
		}
		if isErr, msg := isErrorResponse(toolResponse); isErr {
			out = append(out, &ToolError{
				HookID:       id,
				SessionID:    sessID,
				ToolName:     toolName,
				ErrorMessage: msg,
				Input:        toolInput,
				RecordedAt:   recordedAt,
			})
		}
	}
	return out, rows.Err()
}

// GetSessionErrorCount returns the number of errors in a session.
func (s *Store) GetSessionErrorCount(sessionID string) (int, error) {
	errors, err := s.GetSessionErrors(sessionID)
	if err != nil {
		return 0, err
	}
	return len(errors), nil
}

// GetSessionDuration returns duration in seconds between first and last hook.
func (s *Store) GetSessionDuration(sessionID string) (*float64, error) {
	var durSeconds sql.NullFloat64
	err := s.db.QueryRow(`
		SELECT (julianday(SUBSTR(MAX(recorded_at), 1, 26)) - julianday(SUBSTR(MIN(recorded_at), 1, 26))) * 86400.0
		FROM hooks WHERE session_id = ?
	`, sessionID).Scan(&durSeconds)
	if err != nil {
		return nil, err
	}
	if !durSeconds.Valid || durSeconds.Float64 == 0 {
		return nil, nil
	}
	return &durSeconds.Float64, nil
}

// ToolErrorRate holds error rate stats for a tool.
type ToolErrorRate struct {
	ToolName   string
	TotalCalls int
	ErrorCount int
}

// GetToolErrorRates returns error rates per tool across all sessions.
func (s *Store) GetToolErrorRates() ([]*ToolErrorRate, error) {
	// Get all PostToolUse hooks grouped by tool
	rows, err := s.db.Query(`
		SELECT tool_name, COUNT(*) as total
		FROM hooks
		WHERE event_type = 'PostToolUse' AND tool_name != ''
		GROUP BY tool_name
		ORDER BY total DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type toolTotal struct {
		name  string
		total int
	}
	var tools []toolTotal
	for rows.Next() {
		var t toolTotal
		if err := rows.Scan(&t.name, &t.total); err != nil {
			return nil, err
		}
		tools = append(tools, t)
	}

	// For each tool, count errors by scanning responses
	var out []*ToolErrorRate
	for _, t := range tools {
		respRows, err := s.db.Query(`
			SELECT tool_response FROM hooks
			WHERE event_type = 'PostToolUse' AND tool_name = ? AND tool_response != ''
		`, t.name)
		if err != nil {
			return nil, err
		}
		errCount := 0
		for respRows.Next() {
			var resp string
			if err := respRows.Scan(&resp); err != nil {
				respRows.Close()
				return nil, err
			}
			if isErr, _ := isErrorResponse(resp); isErr {
				errCount++
			}
		}
		respRows.Close()
		if errCount > 0 {
			out = append(out, &ToolErrorRate{
				ToolName:   t.name,
				TotalCalls: t.total,
				ErrorCount: errCount,
			})
		}
	}
	return out, nil
}

// GetTotalErrors returns total error count across all sessions.
func (s *Store) GetTotalErrors() (int, error) {
	rows, err := s.db.Query(`
		SELECT tool_response FROM hooks
		WHERE event_type = 'PostToolUse' AND tool_response != ''
	`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var resp string
		if err := rows.Scan(&resp); err != nil {
			return 0, err
		}
		if isErr, _ := isErrorResponse(resp); isErr {
			count++
		}
	}
	return count, rows.Err()
}
