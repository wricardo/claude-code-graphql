# Projects API

Projects are discovered by reading `~/.claude/projects/`. Each subdirectory corresponds to a working directory where Claude Code was used.

## Queries

### projects

List all discovered projects.

```graphql
{
  projects {
    encodedName
    path
    sessionCount
    transcriptCount
    toolUsage { name count }
    skillsUsed { name count }
  }
}
```

| Field | Description |
|---|---|
| `encodedName` | Raw directory name (e.g. `-Users-alice-go-src-github-com-alice-myrepo`) |
| `path` | Decoded filesystem path (e.g. `/path/to/your/project`) |
| `sessionCount` | Sessions recorded in the DB with this CWD |
| `transcriptCount` | `.jsonl` transcript files found on disk |
| `toolUsage` | Top tools used across all sessions for this project |
| `skillsUsed` | Skills invoked across all sessions for this project |

### project

Fetch a single project by filesystem path.

```graphql
{
  project(path: "/path/to/your/project") {
    encodedName
    path
    sessionCount
    sessions(limit: 10, offset: 0) {
      id
      hookCount
      durationSeconds
      lastSeenAt
    }
  }
}
```

## Path decoding

Claude Code encodes project paths by replacing `/` and `.` with `-` in the directory name. This encoding is lossy — a path like `/Users/alice/my-project` and `/Users/alice/my/project` would encode the same way. The server uses a depth-first filesystem search to reconstruct the most likely path, but short or ambiguous names may decode incorrectly.

## userSkills

List skills discovered in `~/.claude/skills/`:

```graphql
{
  userSkills {
    name
    description
    dirName
  }
}
```
