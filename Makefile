.PHONY: build generate run install install-hooks record-test stop

BIN      := bin/claudegql
PORT     := 8765
DB       := claudegql.db

build:
	go build -o $(BIN) ./cmd/claudegql

install: build
	cp $(BIN) $(GOPATH)/bin/claudegql

generate:
	go run github.com/99designs/gqlgen generate

run: build
	CLAUDEGQL_PORT=$(PORT) CLAUDEGQL_DB=$(DB) ./$(BIN)

# Install hooks into ~/.claude/settings.json using the current binary path
install-hooks: build
	./$(BIN) install-hooks

# Send a test hook event to a running server
record-test:
	@echo '{"session_id":"test-session","hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{"command":"ls"},"cwd":"/tmp"}' \
	  | CLAUDEGQL_SERVER=http://localhost:$(PORT) ./$(BIN) record

stop:
	@lsof -ti:$(PORT) | xargs kill -9 2>/dev/null || true
