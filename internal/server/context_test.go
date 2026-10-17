package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/Ritiksuman07/Vajra/internal/config"
	"github.com/Ritiksuman07/Vajra/internal/session"
)

// fakeOllama answers both chat and embedding requests, so the server can be
// exercised end to end (including retrieval and swarm) without a real model.
type fakeOllama struct {
	mu        sync.Mutex
	systems   []string
	users     []string
	planReply string
}

func (f *fakeOllama) start(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/embed":
			var body struct {
				Input any `json:"input"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			text, _ := body.Input.(string)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"embeddings": [][]float64{embedStub(text)},
			})

		case "/api/chat":
			var body struct {
				Messages []struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"messages"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			system, user := "", ""
			for _, m := range body.Messages {
				switch m.Role {
				case "system":
					system = m.Content
				case "user":
					user = m.Content
				}
			}
			f.mu.Lock()
			f.systems = append(f.systems, system)
			f.users = append(f.users, user)
			plan := f.planReply
			f.mu.Unlock()

			reply := "ok"
			if strings.Contains(user, "Reply with ONLY a JSON array") && plan != "" {
				reply = plan
			} else if strings.Contains(user, "Critique it") {
				reply = "Verdict: pass."
			}
			w.Header().Set("Content-Type", "application/x-ndjson")
			enc := json.NewEncoder(w)
			_ = enc.Encode(map[string]any{"message": map[string]string{"role": "assistant", "content": reply}})
			_ = enc.Encode(map[string]any{"done": true, "eval_count": 3})

		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// embedStub is a deterministic 8-dimension embedding: enough to exercise the
// retrieval plumbing without a real embedding model.
func embedStub(text string) []float64 {
	v := make([]float64, 8)
	for _, w := range strings.Fields(strings.ToLower(text)) {
		v[len(w)%8]++
	}
	return v
}

func newFakeServer(t *testing.T, fake *fakeOllama) (*httptest.Server, *config.Config) {
	t.Helper()
	t.Setenv("VAJRA_CONFIG_DIR", t.TempDir())
	ollama := fake.start(t)
	cfg := config.Load()
	cfg.OllamaHost = ollama.URL
	cfg.Providers[0].BaseURL = ollama.URL
	Version = "test"
	ts := httptest.NewServer(New(cfg).Handler())
	t.Cleanup(ts.Close)
	return ts, cfg
}

func postStream(t *testing.T, url, body string) string {
	t.Helper()
	resp, err := http.Post(url, "application/json", bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return string(b)
}

func TestChatStreamSendsAttachmentsAndBudget(t *testing.T) {
	fake := &fakeOllama{}
	ts, cfg := newFakeServer(t, fake)

	body := `{"message":"explain this","files":[{"name":"main.go","content":"package main\nfunc main(){}"}]}`
	stream := postStream(t, ts.URL+"/api/chat/stream", body)

	if !strings.Contains(stream, `"type":"context"`) || !strings.Contains(stream, `"main.go"`) {
		t.Errorf("stream should report the attachments it used:\n%s", stream)
	}
	if !strings.Contains(stream, `"type":"budget"`) || !strings.Contains(stream, `"max":8192`) {
		t.Errorf("stream should report the context budget:\n%s", stream)
	}

	// The attachment text must reach the model, before the question.
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if len(fake.users) == 0 {
		t.Fatal("no chat request reached the model")
	}
	last := fake.users[len(fake.users)-1]
	if !strings.Contains(last, "package main") {
		t.Errorf("the attached file was not included in the prompt:\n%s", last)
	}
	if !strings.HasSuffix(strings.TrimSpace(last), "explain this") {
		t.Errorf("the question must come last:\n%s", last)
	}

	// And it must be recorded on the session.
	store := session.NewStore(config.SessionsDir())
	metas, err := store.List()
	if err != nil || len(metas) == 0 {
		t.Fatalf("no session saved (err=%v)", err)
	}
	sess, err := store.Load(metas[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(sess.Messages) < 2 || len(sess.Messages[0].Attachments) != 1 || sess.Messages[0].Attachments[0] != "main.go" {
		t.Errorf("attachment not persisted: %+v", sess.Messages)
	}
	_ = cfg
}

func TestChatStreamRetrievesFromIndexedFolder(t *testing.T) {
	fake := &fakeOllama{}
	ts, cfg := newFakeServer(t, fake)

	// Index a folder through the API.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "retries.md"),
		[]byte("The retry policy uses exponential backoff with jitter."), 0o600); err != nil {
		t.Fatal(err)
	}
	idxStream := postStream(t, ts.URL+"/api/rag/index", `{"dir":`+jsonString(dir)+`}`)
	if !strings.Contains(idxStream, `"type":"done"`) || !strings.Contains(idxStream, `"chunks":1`) {
		t.Fatalf("indexing failed:\n%s", idxStream)
	}

	body := `{"message":"how does the retry policy work?","rag_dir":` + jsonString(dir) + `}`
	stream := postStream(t, ts.URL+"/api/chat/stream", body)
	if !strings.Contains(stream, `"sources"`) || !strings.Contains(stream, "retries.md") {
		t.Errorf("retrieval should cite the source file:\n%s", stream)
	}
	// The retrieved passage must reach the model.
	fake.mu.Lock()
	defer fake.mu.Unlock()
	joined := strings.Join(fake.users, "\n")
	if !strings.Contains(joined, "exponential backoff") {
		t.Errorf("retrieved passage was not sent to the model:\n%s", joined)
	}
	_ = cfg
}

func TestRagStatusAndRemove(t *testing.T) {
	fake := &fakeOllama{}
	ts, _ := newFakeServer(t, fake)
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "a.md"), []byte("some local text"), 0o600)

	status := postStream(t, ts.URL+"/api/rag/index", `{"dir":`+jsonString(dir)+`}`)
	if !strings.Contains(status, `"type":"done"`) {
		t.Fatalf("index failed:\n%s", status)
	}

	code, body := get(t, ts.URL+"/api/rag?dir="+urlQuery(dir))
	if code != http.StatusOK || !strings.Contains(body, `"indexed":true`) || !strings.Contains(body, `"chunks":1`) {
		t.Errorf("status = %d %s", code, body)
	}

	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/rag?dir="+urlQuery(dir), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("delete status = %d", resp.StatusCode)
	}
	if _, body := get(t, ts.URL+"/api/rag?dir="+urlQuery(dir)); !strings.Contains(body, `"indexed":false`) {
		t.Errorf("index should be gone: %s", body)
	}
}

func TestRagIndexRejectsMissingDir(t *testing.T) {
	fake := &fakeOllama{}
	ts, _ := newFakeServer(t, fake)
	if code, body := post(t, ts.URL+"/api/rag/index", `{}`); code != http.StatusBadRequest {
		t.Errorf("status = %d (%s), want 400", code, body)
	}
}

func TestSwarmPersistsTranscript(t *testing.T) {
	fake := &fakeOllama{}
	ts, _ := newFakeServer(t, fake)

	stream := postStream(t, ts.URL+"/api/swarm", `{"message":"do a thing","handoffs":0}`)
	if !strings.Contains(stream, `"type":"done"`) || !strings.Contains(stream, `"session_id"`) {
		t.Fatalf("swarm run did not finish with a session:\n%s", stream)
	}

	store := session.NewStore(config.SessionsDir())
	metas, err := store.List()
	if err != nil || len(metas) == 0 {
		t.Fatalf("swarm run was not persisted (err=%v)", err)
	}
	sess, err := store.Load(metas[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(sess.Messages) != 2 {
		t.Fatalf("session has %d messages, want 2", len(sess.Messages))
	}
	last := sess.Messages[1]
	if last.Agent != "swarm" {
		t.Errorf("agent = %q, want swarm", last.Agent)
	}
	if len(last.Swarm) < 4 {
		t.Errorf("swarm transcript not stored: %+v", last.Swarm)
	}
	kinds := map[string]bool{}
	for _, turn := range last.Swarm {
		kinds[turn.Kind] = true
	}
	for _, want := range []string{"plan", "reply", "critique", "synthesis"} {
		if !kinds[want] {
			t.Errorf("transcript missing a %q turn: %+v", want, last.Swarm)
		}
	}
}

func TestSessionExportEndpoint(t *testing.T) {
	fake := &fakeOllama{}
	ts, _ := newFakeServer(t, fake)

	store := session.NewStore(config.SessionsDir())
	sess := session.New("llama3.1:8b", "assistant")
	sess.Add(session.Message{Role: "user", Content: "what is backoff?"})
	sess.Add(session.Message{
		Role: "assistant", Content: "Exponential growth.", Agent: "swarm",
		Sources: []string{"retries.md:12"},
		Swarm:   []session.SwarmTurn{{From: "researcher", To: "all", Kind: "reply", Text: "facts"}},
	})
	if err := store.Save(sess); err != nil {
		t.Fatal(err)
	}

	resp, err := http.Get(ts.URL + "/api/sessions/" + sess.ID + "/export?format=md")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d (%s)", resp.StatusCode, b)
	}
	if cd := resp.Header.Get("Content-Disposition"); !strings.Contains(cd, "attachment") || !strings.Contains(cd, "vajra-"+sess.ID) {
		t.Errorf("Content-Disposition = %q", cd)
	}
	out := string(b)
	for _, want := range []string{"## 1 · You", "researcher → all", "retries.md:12"} {
		if !strings.Contains(out, want) {
			t.Errorf("export missing %q:\n%s", want, out)
		}
	}

	// JSON export, inline (no download header).
	resp2, err := http.Get(ts.URL + "/api/sessions/" + sess.ID + "/export?format=json&download=0")
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if resp2.Header.Get("Content-Disposition") != "" {
		t.Error("download=0 should not set a Content-Disposition")
	}
	if !strings.Contains(resp2.Header.Get("Content-Type"), "application/json") {
		t.Errorf("content-type = %q", resp2.Header.Get("Content-Type"))
	}
}

func TestSessionExportUnknownFormat(t *testing.T) {
	fake := &fakeOllama{}
	ts, _ := newFakeServer(t, fake)
	store := session.NewStore(config.SessionsDir())
	sess := session.New("m", "a")
	if err := store.Save(sess); err != nil {
		t.Fatal(err)
	}
	if code, _ := get(t, ts.URL+"/api/sessions/"+sess.ID+"/export?format=pdf"); code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", code)
	}
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// urlQuery escapes a Windows path for use in a query string.
func urlQuery(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, `\`, "%5C"), " ", "%20")
}

func TestVerifyUsesAttachmentsAsGrounding(t *testing.T) {
	fake := &fakeOllama{}
	ts, _ := newFakeServer(t, fake)

	body := `{"message":"what is the cap?","verify":true,"files":[{"name":"retries.md","content":"The retry cap is 30 seconds."}]}`
	stream := postStream(t, ts.URL+"/api/chat/stream", body)
	if !strings.Contains(stream, `"type":"verification"`) {
		t.Fatalf("no verification in the stream:\n%s", stream)
	}
	if !strings.Contains(stream, "semantic_similarity") {
		t.Errorf("the semantic layer should have run:\n%s", stream)
	}
	if strings.Contains(stream, "no grounding sources") {
		t.Errorf("attachments should be used as grounding, not skipped:\n%s", stream)
	}
	if !strings.Contains(stream, "cosine") {
		t.Errorf("the semantic layer should report a similarity:\n%s", stream)
	}
}

func TestVerifySkipsSemanticWithoutGrounding(t *testing.T) {
	// Honest behaviour: with nothing to compare against, layer 2 says so
	// instead of pretending to check.
	fake := &fakeOllama{}
	ts, _ := newFakeServer(t, fake)
	stream := postStream(t, ts.URL+"/api/chat/stream", `{"message":"hello","verify":true}`)
	if !strings.Contains(stream, "no grounding sources") {
		t.Errorf("layer 2 should report that it was skipped:\n%s", stream)
	}
}
