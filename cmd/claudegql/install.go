package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// hookEventTypes are the Claude Code hook events we register for.
var hookEventTypes = []string{
	"PreToolUse",
	"PostToolUse",
	"Stop",
	"SubagentStop",
	"Notification",
	"SessionStart",
	"SessionEnd",
	"UserPromptSubmit",
	"PreCompact",
	"PostCompact",
}

func runInstallHooks(binaryOverride string) {
	binaryPath := binaryOverride
	if binaryPath == "" {
		var err error
		binaryPath, err = os.Executable()
		if err != nil {
			fatalf("cannot determine binary path: %v\n", err)
		}
		binaryPath, _ = filepath.Abs(binaryPath)
	}

	settingsPath := claudeSettingsPath()
	settings := readSettings(settingsPath)

	// Ensure top-level hooks map exists.
	hooksRaw, ok := settings["hooks"]
	if !ok || hooksRaw == nil {
		hooksRaw = map[string]any{}
		settings["hooks"] = hooksRaw
	}
	hooks, ok := hooksRaw.(map[string]any)
	if !ok {
		fatalf("unexpected format for 'hooks' in %s\n", settingsPath)
	}

	command := binaryPath + " record"
	added := 0

	for _, eventType := range hookEventTypes {
		if commandPresentInEvent(hooks, eventType, command) {
			fmt.Printf("  %-20s already present\n", eventType)
			continue
		}

		entry := map[string]any{
			"hooks": []any{
				map[string]any{
					"type":    "command",
					"command": command,
				},
			},
		}

		existing, _ := hooks[eventType].([]any)
		hooks[eventType] = append(existing, entry)
		fmt.Printf("  %-20s added\n", eventType)
		added++
	}

	if added == 0 {
		fmt.Println("nothing to do — all hooks already present")
		return
	}

	writeSettings(settingsPath, settings)
	fmt.Printf("\n%d hook(s) written to %s\n", added, settingsPath)
	fmt.Printf("command: %s\n", command)
}

// commandPresentInEvent returns true if any hook entry for eventType already
// runs a command that looks like our record command.
func commandPresentInEvent(hooks map[string]any, eventType, command string) bool {
	entriesRaw, ok := hooks[eventType]
	if !ok {
		return false
	}
	entries, _ := entriesRaw.([]any)
	for _, raw := range entries {
		entry, _ := raw.(map[string]any)
		for _, hRaw := range asSlice(entry["hooks"]) {
			h, _ := hRaw.(map[string]any)
			if cmd, _ := h["command"].(string); commandMatches(cmd, command) {
				return true
			}
		}
	}
	return false
}

// commandMatches returns true if existing is the same command or already
// contains "claudegql record" (handles binary renamed/moved).
func commandMatches(existing, desired string) bool {
	return existing == desired || strings.HasSuffix(existing, "claudegql record")
}

func readSettings(path string) map[string]any {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string]any{}
	}
	if err != nil {
		fatalf("reading %s: %v\n", path, err)
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		fatalf("parsing %s: %v\n", path, err)
	}
	return out
}

func writeSettings(path string, settings map[string]any) {
	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		fatalf("marshaling settings: %v\n", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		fatalf("creating directory: %v\n", err)
	}
	if err := os.WriteFile(path, append(out, '\n'), 0644); err != nil {
		fatalf("writing %s: %v\n", path, err)
	}
}

func claudeSettingsPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "settings.json")
}

func asSlice(v any) []any {
	s, _ := v.([]any)
	return s
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format, args...)
	os.Exit(1)
}
