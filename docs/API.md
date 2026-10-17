# HTTP API

Start the server with `vajra serve` (default `http://127.0.0.1:4096`) or
`vajra web`. All responses are JSON unless noted. CORS is enabled for local
tooling. The server binds to loopback only — `--host` opts out of that, and you
should only do so on a trusted network, because it can read the folders you
attach or index.

## Health

```
GET /api/health
```
```json
{ "name": "vajra", "status": "ok", "version": "1.3.0" }
```

## Status

```
GET /api/status
```
Returns the active config plus backend reachability.
```json
{ "config": { … }, "health": { "ollama": true }, "version": "1.3.0" }
```

## Models

```
GET /api/models
```
```json
{ "models": [ { "name": "llama3.1:8b", "backend": "ollama", "size": 4661234567 } ] }
```

## Agents

```
GET /api/agents
```
```json
{ "agents": [ { "id": "coder", "name": "Coder", "emoji": "💻", "tagline": "…" } ] }
```

## Chat (non-streaming)

```
POST /api/chat
```
```json
{
  "message": "explain this",
  "model": "llama3.1:8b",
  "agent": "assistant",
  "session_id": "",
  "files": [ { "name": "main.go", "content": "package main…" } ],
  "rag_dir": "C:/projects/notes"
}
```
```json
{
  "session_id": "b77773e3",
  "content": "Hello!",
  "model": "llama3.1:8b",
  "backend": "ollama",
  "tokens": 2,
  "verification": null
}
```

`files` are texts the client already read — the server never opens a path on a
client's behalf. `rag_dir` must be an indexed folder (see
[Local retrieval](#local-retrieval-rag)); if it is not, the request still
succeeds and simply answers without retrieved context.

## Chat (streaming, SSE)

```
POST /api/chat/stream
```
Same body. The response is `text/event-stream`. Each `data:` line is a JSON
event:

| `type` | Payload |
|--------|---------|
| `start` | `session_id`, `model`, `agent` |
| `context` | `attachments` (names used), `sources` (`file:line` citations) |
| `budget` | `used`, `limit`, `max`, `dropped` — how much of the context window this turn uses |
| `token` | `text` |
| `verifying` | — |
| `verification` | `report` |
| `done` | `session_id`, `tokens` |
| `error` | `error` |

`budget.dropped` is the number of older messages that had to be trimmed to fit
`max`; it is `0` for a short conversation.

```js
const res = await fetch('/api/chat/stream', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ message: 'hi' }),
});
const reader = res.body.getReader();
```

## Swarm (streaming, SSE)

```
POST /api/swarm
```
```json
{
  "message": "build a small app and review it",
  "min_agents": 3,
  "max_agents": 4,
  "handoffs": 1,
  "concurrent": 3,
  "review": true
}
```
All tuning fields are optional. `POST /api/team` is kept as an alias. The body
also accepts `files` and `rag_dir`, exactly as `/api/chat` does, and the goal
is augmented in the same way (attachments, then retrieved passages, then the
question). When the run finishes, a `done` event carries `session_id`: the
whole run — every agent message included — is saved as a session.

Each event is a serialised `agent.SwarmEvent`:

| `type` | Meaning | Key fields |
|--------|---------|-----------|
| `plan` | The Chief decomposed the goal | `plan[]`, `agents[]`, `text` |
| `round` | Round banner | `round`, `text` |
| `agent_start` | An agent began work | `role`, `to`, `kind`, `text` (the brief) |
| `token` | A streamed fragment | `role`, `text` |
| `agent_done` | An agent finished | `role`, `text` (full output) |
| `bus` | A message between agents | `from`, `to`, `kind`, `msg`, `text` |
| `context` | Files/ passages used | `attachments`, `sources` |
| `done` | Final synthesis complete | `session_id` |
| `error` | Something failed | `error` |

`kind` on a `bus`/`agent_*` event is one of `plan`, `task`, `reply`,
`critique`, `handoff` or `synthesis` — that is how a client distinguishes an
agent answering the team from one asking a teammate for help.

```js
await streamSSE('/api/swarm', { message: 'plan a 3-step migration' }, (ev) => {
  if (ev.type === 'agent_start') openBubble(ev.role, ev.to, ev.kind);
  if (ev.type === 'token') appendToken(ev.role, ev.text);
  if (ev.type === 'bus' && ev.kind === 'handoff') showNote(ev.from, ev.to, ev.text);
});
```

## Sessions

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/sessions` | List summaries |
| `POST` | `/api/sessions` | Create (`{"model","agent"}`) |
| `GET` | `/api/sessions/{id}` | Fetch a full session |
| `DELETE` | `/api/sessions/{id}` | Delete |
| `GET` | `/api/sessions/{id}/export` | Download as `?format=md` (default) or `json`; add `download=0` to render inline |

Stored messages carry the context that produced them:

```json
{
  "role": "assistant",
  "content": "They use exponential backoff…",
  "model": "llama3.1:8b",
  "agent": "swarm",
  "attachments": ["notes.md"],
  "sources": ["retries.md:12"],
  "swarm": [ { "from": "coder", "to": "researcher", "kind": "reply", "text": "…" } ]
}
```

## Local retrieval (RAG)

Index a folder once, then reference it per request with `rag_dir`.

```
POST /api/rag/index
```
```json
{ "dir": "C:/projects/notes" }
```
Streams `text/event-stream`:

| `type` | Payload |
|--------|---------|
| `progress` | `stage` (`reading`/`embedding`), `done`, `total`, `current` |
| `done` | `dir`, `files`, `chunks`, `model` |
| `error` | `error` |

```
GET /api/rag?dir=C%3A%2Fprojects%2Fnotes
```
```json
{ "dir": "C:/projects/notes", "indexed": true, "files": 42, "chunks": 310, "model": "nomic-embed-text", "dim": 768, "created": 1735689600 }
```

```
DELETE /api/rag?dir=C%3A%2Fprojects%2Fnotes
```
Removes the index for that folder (the files are untouched).

Indexing needs an embedding model (`ollama pull nomic-embed-text`). Indexes are
plain JSON in `~/.vajra/index`; nothing is uploaded and no vector database is
involved.

## Providers

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/providers` | List (keys never returned) |
| `POST` | `/api/providers` | Add `{"kind","name","api_key","base_url"}` |
| `DELETE` | `/api/providers/{name}` | Remove |

## Config

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/config` | Current config |
| `POST` | `/api/config` | Patch `model`, `agent`, `theme`, `verify`, `stream`, `temperature` |

## Verify

```
POST /api/verify
```
```json
{ "response": "text to check", "context": ["source 1", "source 2"] }
```
Returns the full verification report.

## Web UI

```
GET /
```
Serves the embedded single-page application (`/style.css`, `/app.js`).
