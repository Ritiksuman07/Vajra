# HTTP API

Start the server with `vajra serve` (default `http://localhost:4096`) or
`vajra web`. All responses are JSON unless noted. CORS is enabled for
local tooling.

## Health

```
GET /api/health
```
```json
{ "name": "vajra", "status": "ok", "version": "1.1.0" }
```

## Status

```
GET /api/status
```
Returns the active config plus backend reachability.
```json
{ "config": { … }, "health": { "ollama": true }, "version": "1.1.0" }
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
{ "message": "hi", "model": "llama3.1:8b", "agent": "assistant", "session_id": "" }
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

## Chat (streaming, SSE)

```
POST /api/chat/stream
```
Same body. The response is `text/event-stream`. Each `data:` line is a JSON
event:

| `type` | Payload |
|--------|---------|
| `start` | `session_id`, `model`, `agent` |
| `token` | `text` |
| `verifying` | — |
| `verification` | `report` |
| `done` | `session_id`, `tokens` |
| `error` | `error` |

```js
const res = await fetch('/api/chat/stream', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ message: 'hi' }),
});
const reader = res.body.getReader();
```

## Team (streaming, SSE)

```
POST /api/team
```
```json
{ "message": "build a small app and review it" }
```
Events: `plan`, `step_start`, `token`, `final`, `done`, `error`.

## Sessions

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/sessions` | List summaries |
| `POST` | `/api/sessions` | Create (`{"model","agent"}`) |
| `GET` | `/api/sessions/{id}` | Fetch a full session |
| `DELETE` | `/api/sessions/{id}` | Delete |

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
