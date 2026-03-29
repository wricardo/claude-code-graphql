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
	Type      string `json:"type"`
	UUID      string `json:"uuid"`
	Timestamp string `json:"timestamp"`
	SessionID string `json:"sessionId"`
	CWD       string `json:"cwd"`

	// For user/assistant messages
	Message struct {
		Role    string `json:"role"`
		Content any    `json:"content"`
	} `json:"message"`

	// For tool results
	ToolName string `json:"toolName"`

	// Raw JSON for full access
	Raw json.RawMessage `json:"-"`
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
