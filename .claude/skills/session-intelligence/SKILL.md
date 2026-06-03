# Session Intelligence

Use this skill when working on claude-code-graphql features or docs related to querying Claude Code history: active sessions, skills used, subagents, tools, prompts, edited files, errors, token usage, transcripts, or search.

## Core idea

claude-code-graphql is a **session intelligence layer** for Claude Code, not just a hook logger. It records hook events and combines them with transcripts/project metadata so users can answer:

- What is Claude working on right now?
- Which skills are available, and which are actually used?
- Which tools dominate a session or project?
- What files did Claude edit?
- What prompts led to those edits?
- Which subagents were spawned and what were they assigned?
- What failed, and in which tool call?
- Where does a feature/file/error/table/endpoint appear in prior work?
- Which sessions are long-running or token-heavy?

## Essential CLI commands

Print compact schema:

```bash
claudegql schema
```

List active/recent sessions:

```bash
claudegql query '{ sessions(limit:10) { id status cwd model gitBranch lastSeenAt hookCount } }'
```

Available skills:

```bash
claudegql query '{ userSkills { name dirName description } }'
```

Actually used skills:

```bash
claudegql query '{ sessions(limit:10) { id status skillsUsed { name count } } }'
```

Subagents:

```bash
claudegql query '{ sessions(limit:5) { id subagents { agentType description } } }'
```

Prompts and edited files:

```bash
claudegql query '{ sessions(limit:5) { id prompts editedFiles } }'
```

Recent user prompts:

```bash
claudegql query '{ hooks(filter:{eventType:UserPromptSubmit}, sort:{field:recordedAt,direction:DESC}, limit:10) { sessionId recordedAt prompt cwd } }'
```

Search prompts/tool input/tool output:

```bash
claudegql query '{ search(query:"webhook", limit:5) { matchField snippet hook { sessionId recordedAt eventType toolName } } }'
```

Token usage:

```bash
claudegql query '{ sessions(limit:10) { id model durationSeconds tokenUsage { inputTokens outputTokens cacheReadTokens cacheCreationTokens } } }'
```

## Docs to keep updated

When adding or changing intelligence fields, update:

- `README.md`
- `wiki/Session-Intelligence.md`
- `wiki/Sessions.md`
- `wiki/API-Sessions.md`
- `wiki/API-Projects.md`
- `wiki/API-Stats.md`
- `wiki/API-Search.md`
- `wiki/Reference-CLI.md`
- `CLAUDE.md`

Important fields: `skillsUsed`, `subagents`, `editedFiles`, `prompts`, `toolUsage`, `errors`, `tokenUsage`, `search.matchField`, and typed `parsedInput`.
