// Command vajra is a local-first AI agent workspace.
//
//	vajra                 launch the terminal UI
//	vajra run "prompt"    one-shot, streams to stdout
//	vajra web             web UI in your browser
//	vajra serve           REST API + web UI
//	vajra doctor          check your local setup
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Ritiksuman07/Vajra/internal/agent"
	"github.com/Ritiksuman07/Vajra/internal/attach"
	"github.com/Ritiksuman07/Vajra/internal/config"
	"github.com/Ritiksuman07/Vajra/internal/llm"
	"github.com/Ritiksuman07/Vajra/internal/rag"
	"github.com/Ritiksuman07/Vajra/internal/server"
	"github.com/Ritiksuman07/Vajra/internal/session"
	"github.com/Ritiksuman07/Vajra/internal/tui"
	"github.com/Ritiksuman07/Vajra/internal/verify"
)

// version is overridable at build time with -ldflags "-X main.version=…".
var version = "1.3.0"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "\x1b[31mvajra:\x1b[0m", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return runTUI()
	}

	cmd := args[0]
	rest := args[1:]

	switch cmd {
	case "chat", "tui":
		return runTUI()
	case "run":
		return cmdRun(rest)
	case "serve", "server":
		return cmdServe(rest, false)
	case "web", "gui", "ui":
		return cmdServe(rest, true)
	case "models":
		return cmdModels(rest)
	case "pull":
		return cmdPull(rest)
	case "agents":
		return cmdAgents()
	case "verify":
		return cmdVerify(rest)
	case "index", "ingest":
		return cmdIndex(rest)
	case "rag", "search":
		return cmdRag(rest)
	case "export":
		return cmdExport(rest)
	case "connect":
		return cmdConnect(rest)
	case "providers":
		return cmdProviders()
	case "config":
		return cmdConfig(rest)
	case "doctor":
		return cmdDoctor()
	case "version", "--version", "-v":
		fmt.Printf("vajra %s (%s/%s)\n", version, runtime.GOOS, runtime.GOARCH)
		return nil
	case "help", "--help", "-h":
		printHelp()
		return nil
	default:
		// Treat unknown input as a prompt so `vajra "hi"` works.
		if strings.HasPrefix(cmd, "-") {
			printHelp()
			return nil
		}
		return cmdRun(args)
	}
}

// ---------------------------------------------------------------------------
// TUI
// ---------------------------------------------------------------------------

func runTUI() error {
	cfg := config.Load()
	return tui.Run(cfg, version)
}

// ---------------------------------------------------------------------------
// run — one-shot prompt
// ---------------------------------------------------------------------------

func cmdRun(args []string) error {
	fl := parseFlags(args)
	prompt := fl.rest()
	if strings.TrimSpace(prompt) == "" {
		// Read from stdin when piped (e.g. `cat file | vajra run`).
		if stat, err := os.Stdin.Stat(); err == nil && (stat.Mode()&os.ModeCharDevice) == 0 {
			if b, err := io.ReadAll(os.Stdin); err == nil {
				prompt = string(b)
			}
		}
	}
	if strings.TrimSpace(prompt) == "" {
		// Tolerate `vajra run --swarm "do a thing"`, where the parser
		// would otherwise swallow the prompt as the flag's value.
		if v := fl.get("swarm"); v != "" {
			prompt = v
		} else if v := fl.get("team"); v != "" {
			prompt = v
		}
	}
	if strings.TrimSpace(prompt) == "" {
		return fmt.Errorf("no prompt given. Usage: vajra run \"your prompt\" [--swarm]")
	}

	cfg := config.Load()
	if v := fl.get("model"); v != "" {
		cfg.Model = v
	}
	role := fl.get("agent")
	if role == "" {
		role = cfg.Agent
	}
	router := llm.NewRouter(cfg)
	runner := agent.NewRunner(router)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// Attached files and retrieved passages are prepended to the prompt; the
	// question stays last so it survives any trimming.
	files := attach.Load(fl.listValues("file"), attach.Limits{})
	if n := attach.Summary(files); n != "" {
		fmt.Fprintf(os.Stderr, "\x1b[2m📎 %s\x1b[0m\n", n)
	}
	if sk := attach.SkippedList(files); len(sk) > 0 {
		fmt.Fprintf(os.Stderr, "\x1b[2m   skipped %s\x1b[0m\n", strings.Join(sk, ", "))
	}
	if block := attach.Block(files); block != "" {
		prompt = block + "\n\n" + prompt
	}
	// The same context is handed to the verifier, so "grounded" refers to what
	// the model was actually given.
	grounding := attach.Grounding(files, 4000)
	if dir := fl.get("rag"); dir != "" {
		store := rag.NewStore(config.IndexDir())
		idx, err := store.Load(dir)
		if err != nil {
			return fmt.Errorf("%s is not indexed yet — run: vajra index %q", dir, dir)
		}
		hits, err := rag.Search(ctx, router, idx, fl.rest(), rag.SearchOptions{})
		if err != nil {
			return err
		}
		if len(hits) > 0 {
			prompt = rag.Context(hits, cfg.MaxContext/3) + "\n\n" + prompt
			grounding = append(grounding, rag.Grounding(hits, 4000)...)
			fmt.Fprintf(os.Stderr, "\x1b[2m📚 %d passage(s): %s\x1b[0m\n",
				len(hits), strings.Join(rag.Sources(hits), ", "))
		}
	}
	if fl.has("swarm") || fl.has("team") {
		_, err := runner.RunSwarm(ctx, prompt, agent.SwarmOptions{Handoffs: 1, Concurrent: 3}, printSwarmEvent)
		fmt.Println()
		return err
	}
	var sb strings.Builder
	resp, err := runner.RunModel(ctx, role, cfg.Model, nil, prompt, func(tok string) error {
		sb.WriteString(tok)
		fmt.Print(tok)
		return nil
	})
	fmt.Println()
	if err != nil {
		return err
	}

	if fl.has("verify") || cfg.Verify {
		content := sb.String()
		if content == "" && resp != nil {
			content = resp.Content
		}
		fmt.Println()
		rep := verify.Run(ctx, content, verify.Options{
			Router: router, JudgeModel: cfg.GetJudgeModel(), MinScore: 70,
			Context: grounding,
		})
		printReport(rep)
	}
	return nil
}

// ---------------------------------------------------------------------------
// serve / web
// ---------------------------------------------------------------------------

func cmdServe(args []string, openBrowser bool) error {
	fl := parseFlags(args)
	cfg := config.Load()
	port := cfg.Port
	if v := fl.get("port"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			port = n
		}
	}

	// Vajra reads local files (attachments, RAG folders), so the web UI is
	// loopback-only unless the user explicitly opts in with --host. Binding
	// 0.0.0.0 by accident on a shared network would expose your folders.
	host := "127.0.0.1"
	if v := fl.get("host"); v != "" {
		host = v
	}

	server.Version = version
	srv := server.New(cfg)
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	url := server.URL(addr)

	fmt.Printf("\x1b[35m🛡️  Vajra\x1b[0m %s\n", version)
	fmt.Printf("   Web UI:  \x1b[36m%s\x1b[0m\n", url)
	fmt.Printf("   API:     \x1b[36m%s/api/health\x1b[0m\n", url)
	fmt.Printf("   Local:   %s\n", srv.Config().OllamaHost)
	if !isLoopback(host) {
		fmt.Printf("   \x1b[33m⚠ bound to %s — anyone who can reach this address can read the\n"+
			"     folders you attach or index. Only do this on a trusted network.\x1b[0m\n", host)
	}
	fmt.Println()

	if openBrowser {
		go func() {
			time.Sleep(400 * time.Millisecond)
			_ = openURL(url)
		}()
	}
	return srv.Listen(addr)
}

// isLoopback reports whether host refers to this machine only.
func isLoopback(host string) bool {
	switch strings.ToLower(host) {
	case "localhost", "127.0.0.1", "::1", "[::1]":
		return true
	}
	return strings.HasPrefix(host, "127.")
}

// ---------------------------------------------------------------------------
// index — embed a local folder for retrieval
// ---------------------------------------------------------------------------

func cmdIndex(args []string) error {
	fl := parseFlags(args)
	dir := fl.rest()
	if dir == "" {
		dir = "."
	}
	cfg := config.Load()
	router := llm.NewRouter(cfg)
	store := rag.NewStore(config.IndexDir())
	embed := cfg.GetEmbedModel()

	fmt.Printf("\x1b[35m🛡️  Vajra index\x1b[0m  %s\n", dir)
	fmt.Printf("   model %s   index %s\n\n", embed, store.Path(dir))

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Hour)
	defer cancel()

	last := time.Now()
	idx, err := rag.Build(ctx, router, dir, rag.Options{
		Model: embed,
		Progress: func(stage string, done, total int, current string) {
			switch stage {
			case "reading":
				if time.Since(last) > 120*time.Millisecond {
					last = time.Now()
					fmt.Printf("\r   reading   %-60.60s", current)
				}
			case "embedding":
				pct := 0
				if total > 0 {
					pct = done * 100 / total
				}
				fmt.Printf("\r   embedding %d/%d (%d%%) %-40.40s", done, total, pct, current)
			}
		},
	})
	fmt.Print("\r" + strings.Repeat(" ", 100) + "\r")
	if err != nil {
		return err
	}
	if err := store.Save(idx); err != nil {
		return err
	}
	fmt.Printf("   ✓ indexed %d file(s) → %d passage(s)\n", idx.Files, len(idx.Chunks))
	fmt.Printf("   Ask against it:\n")
	fmt.Printf("     \x1b[36mvajra run --rag %q \"your question\"\x1b[0m\n", idx.Dir)
	return nil
}

// ---------------------------------------------------------------------------
// rag — search an indexed folder
// ---------------------------------------------------------------------------

func cmdRag(args []string) error {
	fl := parseFlags(args)
	dir := fl.get("dir")
	query := fl.rest()
	if dir == "" {
		// Allow `vajra rag <folder> "query"`.
		if fields := strings.SplitN(strings.TrimSpace(strings.Join(args, " ")), " ", 2); len(fields) == 2 {
			dir = strings.Trim(fields[0], `"`)
			query = strings.Trim(strings.TrimSpace(fields[1]), `"`)
		}
	}
	if dir == "" || query == "" {
		return fmt.Errorf("usage: vajra rag <folder> \"query\"")
	}

	cfg := config.Load()
	if v := fl.get("model"); v != "" {
		cfg.Model = v
	}
	router := llm.NewRouter(cfg)
	store := rag.NewStore(config.IndexDir())
	idx, err := store.Load(dir)
	if err != nil {
		return fmt.Errorf("%s is not indexed yet — run: vajra index %q", dir, dir)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	hits, err := rag.Search(ctx, router, idx, query, rag.SearchOptions{K: 6})
	if err != nil {
		return err
	}
	if len(hits) == 0 {
		fmt.Println("No passages matched.")
		return nil
	}
	for i, h := range hits {
		fmt.Printf("\n\x1b[35m[%d]\x1b[0m %s:%d  \x1b[2msimilarity %.2f\x1b[0m\n", i+1, h.Path, h.Line, h.Score)
		fmt.Println(indent(strings.TrimSpace(h.Text), "    "))
	}
	fmt.Println()
	return nil
}

// ---------------------------------------------------------------------------
// export — write a session out as Markdown or JSON
// ---------------------------------------------------------------------------

func cmdExport(args []string) error {
	fl := parseFlags(args)
	id := fl.rest()
	if id == "" {
		id = fl.get("session")
	}
	format := fl.get("format")
	if format == "" {
		format = "md"
	}

	store := session.NewStore(config.SessionsDir())
	if id == "" {
		metas, err := store.List()
		if err != nil {
			return err
		}
		if len(metas) == 0 {
			return fmt.Errorf("no sessions to export yet")
		}
		id = metas[0].ID
	}

	sess, err := store.Load(id)
	if err != nil {
		// Allow a prefix, which is easier than typing the whole id.
		metas, lerr := store.List()
		if lerr != nil {
			return err
		}
		found := ""
		for _, m := range metas {
			if strings.HasPrefix(m.ID, id) {
				found = m.ID
				break
			}
		}
		if found == "" {
			return fmt.Errorf("no session %q", id)
		}
		sess, err = store.Load(found)
		if err != nil {
			return err
		}
	}

	data, ext, err := sess.Export(format)
	if err != nil {
		return err
	}

	out := fl.get("out")
	if out == "" {
		out = sess.Filename(ext)
	}
	if err := os.WriteFile(out, data, 0o600); err != nil {
		return err
	}
	fmt.Printf("✓ exported %s (%d turns) → %s\n", sess.Title, len(sess.Messages), out)
	return nil
}

func openURL(url string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}

// ---------------------------------------------------------------------------
// models / pull
// ---------------------------------------------------------------------------

func cmdModels(args []string) error {
	cfg := config.Load()
	router := llm.NewRouter(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	models, err := router.Models(ctx)
	if err != nil && len(models) == 0 {
		return fmt.Errorf("could not reach Ollama at %s — is it running? (%v)", cfg.OllamaHost, err)
	}
	if len(models) == 0 {
		fmt.Println("No models found. Pull one with:  vajra pull llama3.1:8b")
		return nil
	}
	fmt.Printf("%-42s %-10s %s\n", "MODEL", "BACKEND", "SIZE")
	for _, m := range models {
		fmt.Printf("%-42s %-10s %s\n", m.Name, m.Backend, humanSize(m.Size))
	}
	return nil
}

func cmdPull(args []string) error {
	fl := parseFlags(args)
	name := fl.rest()
	if name == "" && len(args) > 0 {
		name = args[0]
	}
	if name == "" {
		return fmt.Errorf("usage: vajra pull <model>   e.g. vajra pull llama3.1:8b")
	}
	cfg := config.Load()
	client := llm.NewOllama(cfg.OllamaHost)
	fmt.Printf("Pulling %s from %s …\n", name, cfg.OllamaHost)
	last := -1
	err := client.Pull(context.Background(), name, func(pct float64, status string) {
		p := int(pct)
		if p != last || status == "success" {
			last = p
			fmt.Printf("\r  %3d%%  %-30s", p, status)
		}
	})
	fmt.Println()
	if err != nil {
		return err
	}
	fmt.Printf("✓ %s ready\n", name)
	return nil
}

// ---------------------------------------------------------------------------
// agents / verify
// ---------------------------------------------------------------------------

func cmdAgents() error {
	fmt.Println("Built-in agent roles:")
	fmt.Println()
	for _, r := range agent.All() {
		fmt.Printf("  %s  %-14s %s\n", r.Emoji, r.ID, r.Tagline)
	}
	fmt.Println("\nUse one with:  vajra run --agent coder \"...\"  or  /agent <id> in the TUI")
	return nil
}

func cmdVerify(args []string) error {
	text := strings.Join(args, " ")
	if strings.TrimSpace(text) == "" {
		return fmt.Errorf("usage: vajra verify \"<text to check>\"")
	}
	cfg := config.Load()
	router := llm.NewRouter(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	rep := verify.Run(ctx, text, verify.Options{Router: router, JudgeModel: cfg.GetJudgeModel(), MinScore: 70})
	printReport(rep)
	return nil
}

func printReport(rep *verify.Report) {
	for _, l := range rep.Layers {
		mark := "✓"
		if l.Skipped {
			mark = "·"
		} else if !l.Passed {
			mark = "✗"
		}
		fmt.Printf("  %s %-20s %s\n", mark, l.Name, l.Detail)
	}
	status := "\x1b[32mVERIFIED\x1b[0m"
	if rep.NeedsReview {
		status = "\x1b[33mNEEDS REVIEW\x1b[0m"
	}
	fmt.Printf("\n  %s  trust=%.0f%%\n", status, rep.Score)
}

// ---------------------------------------------------------------------------
// connect / providers / config
// ---------------------------------------------------------------------------

func cmdConnect(args []string) error {
	fl := parseFlags(args)
	kind := fl.get("kind")
	if kind == "" && len(args) > 0 {
		kind = args[0]
	}
	if kind == "" {
		return fmt.Errorf("usage: vajra connect <openai|anthropic|openrouter|groq|together|lmstudio> [api-key] [--url ...]")
	}
	key := fl.get("key")
	if key == "" && len(args) > 1 && !strings.HasPrefix(args[1], "-") {
		key = args[1]
	}
	if key == "" {
		if v := os.Getenv(strings.ToUpper(kind) + "_API_KEY"); v != "" {
			key = v
		} else if v := os.Getenv(kind + "_api_key"); v != "" {
			key = v
		}
	}
	if key == "" && kind != "lmstudio" {
		return fmt.Errorf("no API key supplied (pass it as the second argument or set %s_API_KEY)", strings.ToUpper(kind))
	}
	cfg := config.Load()
	cfg.AddProvider(config.Provider{
		Name: kind, Kind: kind, APIKey: key, BaseURL: fl.get("url"), Model: fl.get("model"), Enabled: true,
	})
	if err := cfg.Save(); err != nil {
		return err
	}
	fmt.Printf("✓ connected %s (saved to %s)\n", kind, config.Path())
	return nil
}

func cmdProviders() error {
	cfg := config.Load()
	fmt.Printf("Config: %s\n\n", config.Path())
	for _, p := range cfg.Providers {
		key := "—"
		if p.APIKey != "" {
			key = mask(p.APIKey)
		}
		state := "disabled"
		if p.Enabled {
			state = "enabled"
		}
		extra := ""
		if p.BaseURL != "" {
			extra = "  " + p.BaseURL
		}
		fmt.Printf("  ● %-12s %-9s key=%-14s%s\n", p.Name, state, key, extra)
	}
	return nil
}

func cmdConfig(args []string) error {
	fl := parseFlags(args)
	cfg := config.Load()
	if set := fl.get("set"); set != "" {
		kv := strings.SplitN(set, "=", 2)
		if len(kv) != 2 {
			return fmt.Errorf("expected --set key=value")
		}
		switch kv[0] {
		case "model":
			cfg.Model = kv[1]
		case "agent":
			cfg.Agent = kv[1]
		case "theme":
			cfg.Theme = kv[1]
		case "ollama_host":
			cfg.OllamaHost = kv[1]
		case "verify":
			cfg.Verify = kv[1] == "true" || kv[1] == "1"
		case "port":
			n, _ := strconv.Atoi(kv[1])
			cfg.Port = n
		default:
			return fmt.Errorf("unknown key %q", kv[0])
		}
		if err := cfg.Save(); err != nil {
			return err
		}
		fmt.Printf("✓ set %s\n", set)
		return nil
	}
	data, _ := json.MarshalIndent(cfg, "", "  ")
	fmt.Printf("%s\n\n(path: %s)\n", data, config.Path())
	return nil
}

// ---------------------------------------------------------------------------
// doctor
// ---------------------------------------------------------------------------

func cmdDoctor() error {
	fmt.Printf("\x1b[35m🛡️  Vajra doctor\x1b[0m  (v%s)\n\n", version)
	cfg := config.Load()

	fmt.Printf("  Platform     %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("  Config       %s\n", config.Path())
	if _, err := os.Stat(config.Path()); err == nil {
		fmt.Printf("               ✓ exists\n")
	} else {
		fmt.Printf("               · not created yet (defaults in use)\n")
	}
	fmt.Printf("  Model        %s\n", cfg.Model)
	fmt.Printf("  Agent        %s\n", cfg.Agent)
	fmt.Printf("  Context      %s tokens (VAJRA_MAX_CONTEXT)\n", kfmt(cfg.MaxContext))
	fmt.Printf("  Indexes      %s\n", config.IndexDir())
	if entries, err := os.ReadDir(config.IndexDir()); err == nil && len(entries) > 0 {
		fmt.Printf("               ✓ %d folder(s) indexed\n", len(entries))
	} else {
		fmt.Printf("               · none yet — vajra index <folder>\n")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	client := llm.NewOllama(cfg.OllamaHost)
	fmt.Printf("\n  Ollama       %s\n", cfg.OllamaHost)
	if client.Healthy(ctx) {
		ver, _ := client.Version(ctx)
		models, _ := client.Models(ctx)
		fmt.Printf("               ✓ running (v%s, %d models)\n", ver, len(models))
		embed := cfg.GetEmbedModel()
		if _, err := client.Embed(ctx, embed, "test"); err == nil {
			fmt.Printf("               ✓ embed model %q available\n", embed)
		} else {
			fmt.Printf("               · embed model %q missing → vajra pull %s\n", embed, embed)
		}
	} else {
		fmt.Printf("               ✗ not reachable. Start it: ollama serve\n")
	}

	fmt.Printf("\n  Providers\n")
	for _, p := range cfg.Providers {
		mark := "·"
		if p.Enabled {
			mark = "✓"
		}
		fmt.Printf("               %s %-12s %s\n", mark, p.Name, p.Kind)
	}

	router := llm.NewRouter(cfg)
	fmt.Printf("\n  Routing\n%s", indent(router.String(), "               "))

	fmt.Printf("\n  Status       ")
	if client.Healthy(ctx) {
		fmt.Printf("\x1b[32mready\x1b[0m\n")
	} else {
		fmt.Printf("\x1b[33moffline — run `ollama serve`\x1b[0m\n")
	}
	return nil
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func printHelp() {
	fmt.Printf("\x1b[35m🛡️  Vajra\x1b[0m %s — local-first AI agent workspace\n\n", version)
	fmt.Println("Usage:")
	fmt.Println("  vajra                      Launch the terminal UI")
	fmt.Println("  vajra run \"<prompt>\"       One-shot prompt (streams to stdout)")
	fmt.Println("     --model <m>  --agent <role>  --swarm  --verify")
	fmt.Println("     --file <path>   attach a local file (repeatable, or comma-separated)")
	fmt.Println("     --rag <folder>  answer using passages from an indexed folder")
	fmt.Println("     --swarm: a team of agents that plan, work and review together")
	fmt.Println("  vajra web [--port <n>]     Web UI in your browser")
	fmt.Println("  vajra serve [--port <n>]   REST API + web UI (127.0.0.1 only)")
	fmt.Println("     --host <addr>   bind elsewhere (exposes your folders on that network)")
	fmt.Println("  vajra index <folder>       Embed a folder for local retrieval (RAG)")
	fmt.Println("  vajra rag <folder> \"<q>\"    Search an indexed folder, with citations")
	fmt.Println("  vajra export [<id>] [--format md|json] [--out <path>]")
	fmt.Println("                             Export a session (default: the latest)")
	fmt.Println("  vajra models               List available models")
	fmt.Println("  vajra pull <model>         Download a model from Ollama")
	fmt.Println("  vajra agents               List agent roles")
	fmt.Println("  vajra verify \"<text>\"      Run the verification pipeline")
	fmt.Println("  vajra connect <provider>   Add a cloud provider (optional)")
	fmt.Println("  vajra providers            List configured providers")
	fmt.Println("  vajra config [--set k=v]   Show or change configuration")
	fmt.Println("  vajra doctor               Diagnose your local setup")
	fmt.Println("  vajra version | help")
	fmt.Println()
	fmt.Println("Environment:")
	fmt.Println("  VAJRA_MODEL  VAJRA_AGENT  VAJRA_PORT  VAJRA_VERIFY  OLLAMA_HOST")
	fmt.Println("  VAJRA_MAX_CONTEXT  VAJRA_CONFIG_DIR")
	fmt.Println()
	fmt.Println("Quick start:")
	fmt.Println("  ollama serve &  ollama pull llama3.1:8b && vajra")
}

// flagSet is a minimal `--key value` / `--flag` parser. A flag may be given
// more than once (`--file a.md --file b.go`); all occurrences are kept in
// multi, while values holds the last one for the simple callers.
type flagSet struct {
	values map[string]string
	multi  map[string][]string
	flags  map[string]bool
	free   []string
}

func parseFlags(args []string) *flagSet {
	fs := &flagSet{values: map[string]string{}, multi: map[string][]string{}, flags: map[string]bool{}}
	set := func(name, value string) {
		fs.values[name] = value
		fs.multi[name] = append(fs.multi[name], value)
	}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "--") {
			name := strings.TrimPrefix(a, "--")
			if eq := strings.Index(name, "="); eq >= 0 {
				set(name[:eq], name[eq+1:])
				continue
			}
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
				set(name, args[i+1])
				i++
			} else {
				fs.flags[name] = true
			}
			continue
		}
		fs.free = append(fs.free, a)
	}
	return fs
}

func (f *flagSet) get(k string) string { return f.values[k] }
func (f *flagSet) has(k string) bool   { _, ok := f.values[k]; return ok || f.flags[k] }
func (f *flagSet) rest() string        { return strings.Join(f.free, " ") }

// all returns every value given for a repeatable flag.
func (f *flagSet) all(k string) []string {
	if v, ok := f.multi[k]; ok {
		return v
	}
	return nil
}

// listValues splits a repeatable flag's values, so both `--file a --file b`
// and `--file a,b` work.
func (f *flagSet) listValues(k string) []string {
	var out []string
	for _, v := range f.all(k) {
		for _, part := range strings.Split(v, ",") {
			if part = strings.TrimSpace(part); part != "" {
				out = append(out, part)
			}
		}
	}
	return out
}

func mask(key string) string {
	if len(key) <= 6 {
		return "••••"
	}
	return key[:4] + "••••" + key[len(key)-2:]
}

// kfmt renders a token count compactly (8192 → 8.2k).
func kfmt(n int) string {
	if n < 1000 {
		return strconv.Itoa(n)
	}
	return fmt.Sprintf("%.1fk", float64(n)/1000)
}

func humanSize(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func indent(s, prefix string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i, l := range lines {
		lines[i] = prefix + l
	}
	return strings.Join(lines, "\n") + "\n"
}

// ---------------------------------------------------------------------------
// Swarm rendering (CLI)
// ---------------------------------------------------------------------------

// printSwarmEvent renders a swarm run for the terminal: the plan, each
// agent's output labelled with who is talking to whom, the handoffs agents
// make to each other, and the final synthesis.
//
// Tokens from the parallel work round arrive interleaved, so they are
// buffered per agent and printed as one labelled block when the agent
// finishes. The final synthesis streams live, because that is the answer
// the user is waiting for.
func printSwarmEvent(ev agent.SwarmEvent) {
	swarmMu.Lock()
	defer swarmMu.Unlock()

	switch ev.Type {
	case "plan":
		fmt.Printf("\n\x1b[36m🐝 swarm plan\x1b[0m\n%s\n", strings.TrimRight(ev.Text, "\n"))
	case "round":
		fmt.Printf("\n\x1b[2m── %s ──\x1b[0m\n", ev.Text)
	case "bus":
		if ev.Kind == "handoff" {
			fmt.Printf("\x1b[33m✉ %s → %s\x1b[0m  %s\n", roleLabel(ev.From), roleLabel(ev.To), ev.Text)
		}
	case "agent_start":
		swarmOpen[ev.Role] = &strings.Builder{}
		if ev.Kind == "synthesis" {
			swarmLive[ev.Role] = true
			fmt.Printf("\n\x1b[35m%s → %s\x1b[0m \x1b[2m%s\x1b[0m\n",
				roleLabel(ev.Role), roleDest(ev.To), ev.Kind)
		} else {
			fmt.Printf("\x1b[2m⟳ %s started…\x1b[0m\n", roleLabel(ev.Role))
		}
	case "token":
		if b, ok := swarmOpen[ev.Role]; ok {
			b.WriteString(ev.Text)
		}
		if swarmLive[ev.Role] {
			fmt.Print(ev.Text)
		}
	case "agent_done":
		b := swarmOpen[ev.Role]
		live := swarmLive[ev.Role]
		delete(swarmOpen, ev.Role)
		delete(swarmLive, ev.Role)

		text := ev.Text
		if text == "" && b != nil {
			text = b.String()
		}
		if live {
			fmt.Println()
			return
		}
		fmt.Printf("\n\x1b[35m%s → %s\x1b[0m \x1b[2m%s\x1b[0m\n",
			roleLabel(ev.Role), roleDest(ev.To), ev.Kind)
		fmt.Println(strings.TrimRight(text, "\n"))
	case "error":
		fmt.Fprintf(os.Stderr, "\x1b[31merror:\x1b[0m %s\n", ev.Error)
	}
}

// roleLabel renders a bus participant for the terminal.
func roleLabel(id string) string {
	if r, ok := agent.Get(id); ok {
		return r.Emoji + " " + r.Name
	}
	switch id {
	case "user":
		return "👤 You"
	case "all":
		return "👥 everyone"
	}
	return id
}

// roleDest renders the recipient of a message.
func roleDest(id string) string {
	switch id {
	case "all", "":
		return "👥 everyone"
	case "user":
		return "👤 you"
	}
	return roleLabel(id)
}

// State for the CLI swarm renderer (events arrive from worker goroutines).
var (
	swarmMu   sync.Mutex
	swarmOpen = map[string]*strings.Builder{}
	swarmLive = map[string]bool{}
)
