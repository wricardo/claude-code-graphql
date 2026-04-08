# Transcripts API

Read structured content from Claude Code's `.jsonl` session transcript files.

## Top-level query

```graphql
{
  transcript(sessionId: "abc-123", limit: 100, offset: 0) {
    type
    role
    uuid
    parentUuid
    timestamp
    content
    isSidechain
    contentBlocks {
      __typename
      ... on TextBlock       { text }
      ... on ToolUseBlock    { id name input }
      ... on ToolResultBlock { toolUseId content isError }
      ... on ThinkingBlock   { thinking }
    }
  }
}
```

## Via a session

```graphql
{
  session(id: "abc-123") {
    transcript(limit: 50) {
      type
      role
      content
      isSidechain
    }
  }
}
```

## TranscriptMessage fields

| Field | Description |
|---|---|
| `type` | Entry type: `user`, `assistant`, `file-history-snapshot`, `system`, `summary` |
| `role` | Message role: `user` or `assistant` (null for non-message entries) |
| `uuid` | Unique message ID within the transcript |
| `parentUuid` | Parent message UUID for threaded conversation reconstruction |
| `timestamp` | ISO timestamp from the transcript entry |
| `content` | Plain string for simple messages; JSON-encoded for block arrays. Prefer `contentBlocks`. |
| `isSidechain` | True if this message came from a subagent |
| `raw` | Complete raw JSON of the transcript entry |
| `contentBlocks` | Parsed content blocks (see below) |

## ContentBlock union

Messages with complex content are parsed into typed blocks:

```graphql
contentBlocks {
  __typename
  ... on TextBlock {
    text          # Plain text content
  }
  ... on ToolUseBlock {
    id            # Tool use ID (links to ToolResultBlock)
    name          # Tool name
    input         # JSON-encoded tool input
  }
  ... on ToolResultBlock {
    toolUseId     # Links back to the ToolUseBlock
    content       # Tool output
    isError       # Whether the result was an error
  }
  ... on ThinkingBlock {
    thinking      # Always empty — Claude's reasoning is encrypted
  }
}
```

## Subagent transcripts

Subagents spawned during a session have their own transcripts:

```graphql
{
  session(id: "abc-123") {
    subagents {
      id
      agentType
      description
      transcript(limit: 20) {
        type
        role
        content
      }
    }
  }
}
```
