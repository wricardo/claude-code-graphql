package claude

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// DefaultBaseDir returns ~/.claude.
func DefaultBaseDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude")
}

// Project represents a Claude Code project directory.
type Project struct {
	EncodedName string // directory name, e.g. "-Users-wricardo-go-src-..."
	Path        string // decoded filesystem path, e.g. "/Users/wricardo/go/src/..."
}

// ListProjects reads ~/.claude/projects/ and returns all project entries.
func ListProjects(baseDir string) ([]Project, error) {
	dir := filepath.Join(baseDir, "projects")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []Project
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		out = append(out, Project{
			EncodedName: name,
			Path:        decodeDirName(name),
		})
	}
	return out, nil
}

// CountProjectSessions counts .jsonl files in a project directory (each is a session transcript).
func CountProjectSessions(baseDir, encodedName string) int {
	dir := filepath.Join(baseDir, "projects", encodedName)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	n := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".jsonl") {
			n++
		}
	}
	return n
}

// Skill represents a Claude Code skill from ~/.claude/skills/.
type Skill struct {
	Name        string
	Description string
	DirName     string // directory name under skills/
}

// ListUserSkills reads ~/.claude/skills/ and parses SKILL.md frontmatter.
func ListUserSkills(baseDir string) ([]Skill, error) {
	dir := filepath.Join(baseDir, "skills")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []Skill
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		sk := Skill{DirName: e.Name()}
		parseSKILLmd(filepath.Join(dir, e.Name(), "SKILL.md"), &sk)
		if sk.Name == "" {
			sk.Name = e.Name()
		}
		out = append(out, sk)
	}
	return out, nil
}

// ListProjectSkills reads skill directories inside a project's config directory.
// Project-level skills don't exist as separate dirs currently, so this returns
// skills used in the project (extracted from hook data) — that's handled at the resolver level.
// This function checks for a skills/ subdirectory if it ever appears.
func ListProjectSkills(baseDir, encodedName string) ([]Skill, error) {
	dir := filepath.Join(baseDir, "projects", encodedName, "skills")
	entries, err := os.ReadDir(dir)
	if err != nil {
		// No skills directory is fine
		return nil, nil
	}
	var out []Skill
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		sk := Skill{DirName: e.Name()}
		parseSKILLmd(filepath.Join(dir, e.Name(), "SKILL.md"), &sk)
		if sk.Name == "" {
			sk.Name = e.Name()
		}
		out = append(out, sk)
	}
	return out, nil
}

// TranscriptMessage is a single entry in a session's .jsonl transcript.
type TranscriptMessage struct {
	Type        string `json:"type"`
	UUID        string `json:"uuid"`
	ParentUUID  string `json:"parentUuid"`
	IsSidechain bool   `json:"isSidechain"`
	Timestamp   string `json:"timestamp"`
	SessionID   string `json:"sessionId"`
	GitBranch   string `json:"gitBranch"`
	CWD         string `json:"cwd"`

	// For user/assistant messages
	Message struct {
		Role    string `json:"role"`
		Content any    `json:"content"`
		Model   string `json:"model"`
		Usage   struct {
			InputTokens              int `json:"input_tokens"`
			OutputTokens             int `json:"output_tokens"`
			CacheReadInputTokens     int `json:"cache_read_input_tokens"`
			CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
		} `json:"usage"`
	} `json:"message"`

	// For tool results
	ToolName string `json:"toolName"`

	// Raw JSON for full access
	Raw json.RawMessage `json:"-"`
}

// SubagentMeta holds the metadata from an agent-*.meta.json file.
type SubagentMeta struct {
	AgentType   string `json:"agentType"`
	Description string `json:"description"`
}

// Subagent represents a spawned subagent for a session.
type Subagent struct {
	ID          string // agent ID, e.g. "a884f9b5b5867af1b"
	AgentType   string
	Description string
	// TranscriptPath is the path to the subagent's .jsonl file.
	TranscriptPath string
}

// ListSubagents returns all subagents for a given session by reading the subagents/ directory.
func ListSubagents(baseDir, encodedProject, sessionID string) ([]Subagent, error) {
	dir := filepath.Join(baseDir, "projects", encodedProject, sessionID, "subagents")
	entries, err := os.ReadDir(dir)
	if err != nil {
		// No subagents directory is normal
		return nil, nil
	}

	metaByID := map[string]*SubagentMeta{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".meta.json") {
			continue
		}
		// e.g. "agent-abc123.meta.json" → id "abc123"
		base := strings.TrimSuffix(e.Name(), ".meta.json")
		id := strings.TrimPrefix(base, "agent-")
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var meta SubagentMeta
		if err := json.Unmarshal(data, &meta); err != nil {
			continue
		}
		metaByID[id] = &meta
	}

	var out []Subagent
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		base := strings.TrimSuffix(e.Name(), ".jsonl")
		id := strings.TrimPrefix(base, "agent-")
		sa := Subagent{
			ID:             id,
			TranscriptPath: filepath.Join(dir, e.Name()),
		}
		if meta, ok := metaByID[id]; ok {
			sa.AgentType = meta.AgentType
			sa.Description = meta.Description
		}
		out = append(out, sa)
	}
	return out, nil
}

// ReadSubagentTranscript reads a subagent's .jsonl transcript file.
func ReadSubagentTranscript(transcriptPath string) ([]TranscriptMessage, error) {
	f, err := os.Open(transcriptPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var out []TranscriptMessage
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var msg TranscriptMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}
		msg.Raw = json.RawMessage(make([]byte, len(line)))
		copy(msg.Raw, line)
		out = append(out, msg)
	}
	return out, scanner.Err()
}

// ReadToolResultFile reads a tool result file from <session>/tool-results/<toolUseId>.txt.
// Returns empty string if the file does not exist.
func ReadToolResultFile(baseDir, encodedProject, sessionID, toolUseID string) (string, error) {
	path := filepath.Join(baseDir, "projects", encodedProject, sessionID, "tool-results", toolUseID+".txt")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ReadTranscript reads a session's .jsonl transcript file.
func ReadTranscript(baseDir, encodedProjectName, sessionID string) ([]TranscriptMessage, error) {
	path := filepath.Join(baseDir, "projects", encodedProjectName, sessionID+".jsonl")
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var out []TranscriptMessage
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024) // 10MB max line
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var msg TranscriptMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}
		msg.Raw = json.RawMessage(make([]byte, len(line)))
		copy(msg.Raw, line)
		out = append(out, msg)
	}
	return out, scanner.Err()
}

// FindProjectForSession looks through all projects to find which one contains a given session ID.
func FindProjectForSession(baseDir, sessionID string) (encodedName string, found bool) {
	dir := filepath.Join(baseDir, "projects")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", false
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		jsonlPath := filepath.Join(dir, e.Name(), sessionID+".jsonl")
		if _, err := os.Stat(jsonlPath); err == nil {
			return e.Name(), true
		}
	}
	return "", false
}

// EncodeProjectPath converts a filesystem path to the encoded directory name.
// Claude Code replaces / with - and . with - in project directory names.
func EncodeProjectPath(path string) string {
	return strings.ReplaceAll(path, "/", "-")
}

// FindProjectByPath finds a project whose decoded path matches the given path.
func FindProjectByPath(baseDir, path string) (*Project, error) {
	projects, err := ListProjects(baseDir)
	if err != nil {
		return nil, err
	}
	for _, p := range projects {
		if p.Path == path {
			return &p, nil
		}
	}
	return nil, nil
}

// decodeDirName converts an encoded directory name back to a filesystem path.
// The encoding replaces / with -, which is lossy when path components contain -.
// We greedily reconstruct by checking which path segments exist on the filesystem.
func decodeDirName(name string) string {
	if name == "" {
		return ""
	}
	if name == "-" {
		return "/"
	}
	// Strip leading - (represents root /)
	if strings.HasPrefix(name, "-") {
		name = name[1:]
	}

	parts := strings.Split(name, "-")
	return "/" + resolvePathParts(parts)
}

// resolvePathParts greedily reconstructs a filesystem path from dash-separated segments.
// The encoding maps both / and . to -, so we try combinations using DFS with
// filesystem validation.
func resolvePathParts(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	result, _ := resolvePathDFS("/", parts, 0, "")
	if result != "" {
		return result[1:] // strip leading /
	}
	// Fallback: just join with /
	return strings.Join(parts, "/")
}

// resolvePathDFS tries to reconstruct the original path by testing whether
// joining segments with "/", ".", or "-" produces existing filesystem paths.
// prefix is the confirmed path so far (e.g. "/Users/wricardo/go/src/")
// current is the segment being built (e.g. "github" or "github.com")
// Returns the path and whether the final result exists on the filesystem.
func resolvePathDFS(prefix string, parts []string, idx int, current string) (string, bool) {
	if idx >= len(parts) {
		full := prefix + current
		return full, pathExists(full)
	}

	seg := parts[idx]

	if current == "" {
		return resolvePathDFS(prefix, parts, idx+1, seg)
	}

	type candidate struct {
		newPrefix  string
		newCurrent string
	}

	candidates := []candidate{
		{prefix + current + "/", seg},     // commit as directory
		{prefix, current + "-" + seg},     // extend with dash
		{prefix, current + "." + seg},     // extend with dot
	}

	// First pass: try candidates where intermediate path exists, and final path also exists
	for _, c := range candidates {
		testPath := c.newPrefix + c.newCurrent
		if pathExists(testPath) {
			if result, exists := resolvePathDFS(c.newPrefix, parts, idx+1, c.newCurrent); exists {
				return result, true
			}
		}
	}

	// Second pass: try all candidates regardless of intermediate path,
	// looking for a final result that exists
	for _, c := range candidates {
		if result, exists := resolvePathDFS(c.newPrefix, parts, idx+1, c.newCurrent); exists {
			return result, true
		}
	}

	// Nothing produces an existing path — fall back to directory separator
	result, _ := resolvePathDFS(prefix+current+"/", parts, idx+1, seg)
	return result, false
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// parseSKILLmd reads the YAML frontmatter from a SKILL.md file.
func parseSKILLmd(path string, sk *Skill) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	inFrontmatter := false
	var descLines []string
	inDesc := false

	for scanner.Scan() {
		line := scanner.Text()
		if line == "---" {
			if inFrontmatter {
				break // end of frontmatter
			}
			inFrontmatter = true
			continue
		}
		if !inFrontmatter {
			continue
		}

		if strings.HasPrefix(line, "name:") {
			sk.Name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
			inDesc = false
		} else if strings.HasPrefix(line, "description:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "description:"))
			if val == ">" || val == "|" {
				inDesc = true
			} else {
				sk.Description = val
				inDesc = false
			}
		} else if inDesc {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || (!strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t")) {
				inDesc = false
			} else {
				descLines = append(descLines, trimmed)
			}
		}
	}
	if len(descLines) > 0 && sk.Description == "" {
		sk.Description = strings.Join(descLines, " ")
	}
}
