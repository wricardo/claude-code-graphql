package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/wricardo/claude-code-graphql/graph"
	"github.com/wricardo/claude-code-graphql/internal/store"
)

func main() {
	if len(os.Args) < 2 {
		runServer()
		return
	}
	switch os.Args[1] {
	case "record":
		runRecord()
	case "install-hooks":
		// Optional: --binary /path/to/claudegql
		binary := ""
		for i, arg := range os.Args[2:] {
			if arg == "--binary" && i+1 < len(os.Args[2:]) {
				binary = os.Args[i+3]
			}
		}
		runInstallHooks(binary)
	default:
		fmt.Fprintf(os.Stderr, "usage: claudegql [record|install-hooks [--binary PATH]]\n")
		os.Exit(1)
	}
}

// runRecord reads a Claude Code hook JSON from stdin and forwards it to the server.
// Used as the hook command in Claude Code settings.json.
func runRecord() {
	var payload map[string]any
	if err := json.NewDecoder(os.Stdin).Decode(&payload); err != nil {
		os.Exit(0) // never block Claude Code
	}

	serverURL := os.Getenv("CLAUDEGQL_SERVER")
	if serverURL == "" {
		serverURL = "http://localhost:8765"
	}

	data, _ := json.Marshal(payload)
	resp, err := http.Post(serverURL+"/hook", "application/json", bytes.NewReader(data))
	if err != nil {
		os.Exit(0) // silent failure — Claude Code must not be blocked
	}
	resp.Body.Close()
}

// hookPayload matches the JSON structure Claude Code sends to hooks.
type hookPayload struct {
	SessionID            string         `json:"session_id"`
	HookEventName        string         `json:"hook_event_name"`
	ToolName             string         `json:"tool_name"`
	ToolUseID            string         `json:"tool_use_id"`
	ToolInput            map[string]any `json:"tool_input"`
	ToolResponse         map[string]any `json:"tool_response"`
	Prompt               string         `json:"prompt"`
	CWD                  string         `json:"cwd"`
	TranscriptPath       string         `json:"transcript_path"`
	// Stop/SubagentStop event fields.
	StopHookActive       *bool          `json:"stop_hook_active"`
	LastAssistantMessage string         `json:"last_assistant_message"`
}

func hookHandler(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var p hookPayload
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if p.SessionID == "" || p.HookEventName == "" {
			http.Error(w, "session_id and hook_event_name are required", http.StatusBadRequest)
			return
		}

		// For Stop/SubagentStop events, tool_input is absent but Claude Code sends
		// stop_hook_active and last_assistant_message as top-level fields. Fold them
		// into tool_input so resolvers can parse them from a single column.
		if p.StopHookActive != nil || p.LastAssistantMessage != "" {
			if p.ToolInput == nil {
				p.ToolInput = make(map[string]any)
			}
			if p.StopHookActive != nil {
				p.ToolInput["stop_hook_active"] = *p.StopHookActive
			}
			if p.LastAssistantMessage != "" {
				p.ToolInput["last_assistant_message"] = p.LastAssistantMessage
			}
		}
		toolInput := marshalJSON(p.ToolInput)
		toolResponse := marshalJSON(p.ToolResponse)

		_, err := s.RecordHook(store.RecordHookInput{
			SessionID:      p.SessionID,
			EventType:      p.HookEventName,
			ToolName:       p.ToolName,
			ToolUseID:      p.ToolUseID,
			ToolInput:      toolInput,
			ToolResponse:   toolResponse,
			Prompt:         p.Prompt,
			CWD:            p.CWD,
			TranscriptPath: p.TranscriptPath,
		})
		if err != nil {
			log.Printf("store.RecordHook: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func marshalJSON(v map[string]any) string {
	if v == nil {
		return ""
	}
	b, _ := json.Marshal(v)
	return string(b)
}

func runServer() {
	dbPath := os.Getenv("CLAUDEGQL_DB")
	if dbPath == "" {
		dbPath = "claudegql.db"
	}

	s, err := store.New(dbPath)
	if err != nil {
		log.Fatalf("store: %v", err)
	}

	claudeDir := os.Getenv("CLAUDEGQL_CLAUDE_DIR")
	if claudeDir == "" {
		home, _ := os.UserHomeDir()
		claudeDir = home + "/.claude"
	}

	gqlSrv := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{
		Resolvers: &graph.Resolver{Store: s, ClaudeDir: claudeDir},
	}))

	mux := http.NewServeMux()
	mux.Handle("/graphql", gqlSrv)
	mux.Handle("/playground", playground.Handler("Claude Code GraphQL", "/graphql"))
	mux.HandleFunc("/hook", hookHandler(s))
	addr := ":" + port()
	fmt.Printf("server:     http://localhost%s/graphql\n", addr)
	fmt.Printf("playground: http://localhost%s/playground\n", addr)
	fmt.Printf("hook:       POST http://localhost%s/hook\n", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func port() string {
	if p := os.Getenv("CLAUDEGQL_PORT"); p != "" {
		return p
	}
	return "8765"
}
