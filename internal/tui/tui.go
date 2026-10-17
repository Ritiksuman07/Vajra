// Package tui implements Vajra's interactive terminal interface.
//
// It is built on Bubble Tea and streams model output token-by-token.
package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Ritiksuman07/Vajra/internal/agent"
	"github.com/Ritiksuman07/Vajra/internal/attach"
	"github.com/Ritiksuman07/Vajra/internal/config"
	"github.com/Ritiksuman07/Vajra/internal/llm"
	"github.com/Ritiksuman07/Vajra/internal/rag"
	"github.com/Ritiksuman07/Vajra/internal/session"
	"github.com/Ritiksuman07/Vajra/internal/verify"
)

type screen int

const (
	scrChat screen = iota
	scrHelp
	scrModels
	scrAgents
	scrSessions
	scrSwarm
)

// ---------------------------------------------------------------------------
// Messages
// ---------------------------------------------------------------------------

type tokenMsg struct{ text string }

// swarmMsg carries one event from the swarm engine to the UI.
type swarmMsg struct{ ev agent.SwarmEvent }

// sourcesMsg reports the passages local retrieval contributed to a prompt.
type sourcesMsg struct {
	dir     string
	sources []string
	err     error
}

type genDoneMsg struct {
	err    error
	tokens int
	report *verify.Report
	// swarm runs finish with a synthesised final answer and no single
	// streamed message, so the UI needs to know which kind of run ended.
	swarm bool
	final string
}
type modelsMsg struct {
	models []llm.Model
	err    error
}
type sessionsMsg struct{ sessions []session.Meta }
type healthMsg struct{ ok bool }

// ---------------------------------------------------------------------------
// Model
// ---------------------------------------------------------------------------

// Model is the Bubble Tea model.
type Model struct {
	cfg     *config.Config
	router  *llm.Router
	runner  *agent.Runner
	store   *session.Store
	version string

	width, height int
	ready         bool

	vp      viewport.Model
	input   textinput.Model
	spinner spinner.Model

	messages []session.Message
	sess     *session.Session

	streaming bool
	ch        chan tea.Msg
	acc       strings.Builder
	curAgent  string
	curModel  string

	screen   screen
	cursor   int
	models   []llm.Model
	agents   []agent.Role
	sessions []session.Meta

	verifyOn bool
	swarmOn  bool
	health   bool
	err      error
	notice   string

	// Swarm display state.
	swarmFeed    []*swarmItem
	swarmBus     []agent.BusMessage
	swarmOpen    map[string]int
	swarmAgents  []string
	swarmAfter   int
	swarmRunning bool

	// Local context: files attached to the next prompt, the folder being
	// searched, and the sources the last answer used.
	attached   []string
	ragDir     string
	ragChunks  int
	curSources []string
}

// Run starts the TUI and blocks until the user quits.
func Run(cfg *config.Config, version string) error {
	router := llm.NewRouter(cfg)
	m := newModel(cfg, router, version)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}

func newModel(cfg *config.Config, router *llm.Router, version string) *Model {
	in := textinput.New()
	in.Placeholder = "Ask anything…  (/help for commands)"
	in.Prompt = "❯ "
	in.CharLimit = 8000
	in.Focus()

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(colAccent)

	sess := session.New(cfg.Model, cfg.Agent)

	return &Model{
		cfg:        cfg,
		router:     router,
		runner:     agent.NewRunner(router),
		store:      session.NewStore(config.SessionsDir()),
		version:    version,
		input:      in,
		spinner:    sp,
		sess:       sess,
		agents:     agent.All(),
		curAgent:   cfg.Agent,
		curModel:   cfg.Model,
		verifyOn:   cfg.Verify,
		swarmOpen:  map[string]int{},
		swarmAfter: -1,
	}
}

// Init implements tea.Model.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.input.Focus(), m.spinner.Tick, loadModels(m.router), loadSessions(m.store), checkHealth(m.router))
}

// ---------------------------------------------------------------------------
// Commands
// ---------------------------------------------------------------------------

func loadModels(r *llm.Router) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		models, err := r.Models(ctx)
		return modelsMsg{models: models, err: err}
	}
}

func loadSessions(st *session.Store) tea.Cmd {
	return func() tea.Msg {
		metas, _ := st.List()
		return sessionsMsg{sessions: metas}
	}
}

func checkHealth(r *llm.Router) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		return healthMsg{ok: r.Ollama().Healthy(ctx)}
	}
}

func waitFor(ch chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return genDoneMsg{}
		}
		return msg
	}
}

// generate launches a streaming generation and returns the first wait cmd.
func (m *Model) generate(prompt string) tea.Cmd {
	if m.streaming {
		return nil
	}
	// Anything the user attached (/attach or an @path mention) is read locally.
	files := m.collectAttachments(prompt)

	history := m.sess.History(16)
	llmsgs := make([]llm.Message, 0, len(history))
	for _, h := range history {
		llmsgs = append(llmsgs, llm.Message{Role: h.Role, Content: h.Content})
	}

	m.messages = append(m.messages,
		session.Message{Role: "user", Content: prompt, Attachments: attachNames(files)},
		session.Message{Role: "assistant", Model: m.curModel, Agent: m.curAgent},
	)
	m.acc.Reset()
	m.streaming = true
	m.err = nil
	m.curSources = nil
	if n := attachNotice(files); n != "" {
		m.notice = n
	}

	// Swarm mode renders its own transcript between the prompt and the answer.
	m.swarmAfter = len(m.messages) - 2
	m.swarmFeed = nil
	m.swarmBus = nil
	m.swarmAgents = nil
	m.swarmOpen = map[string]int{}
	m.swarmRunning = m.swarmOn
	m.refresh()

	ch := make(chan tea.Msg, 256)
	m.ch = ch

	role := m.curAgent
	model := m.curModel
	swarm := m.swarmOn
	ragDir := m.ragDir
	ctxTokens := m.cfg.MaxContext
	runner := m.runner
	router := m.router
	cfg := m.cfg
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		defer close(ch)

		// Attached files first, then retrieved passages, then the question.
		// Ending with the question matters: it is the part Vajra keeps if the
		// prompt has to be trimmed to fit the context window.
		grounding := attach.Grounding(files, 4000)
		input := prompt
		if block := attach.Block(files); block != "" {
			input = block + "\n\n" + input
		}
		if ragDir != "" {
			hits, err := m.searchRAG(ctx, ragDir, prompt)
			if len(hits) > 0 {
				input = rag.Context(hits, ctxTokens/3) + "\n\n" + input
				grounding = rag.Grounding(hits, 4000)
			}
			ch <- sourcesMsg{dir: ragDir, sources: rag.Sources(hits), err: err}
		}

		if swarm {
			res, err := runner.RunSwarm(ctx, input, agent.SwarmOptions{Handoffs: 1, Concurrent: 3},
				func(ev agent.SwarmEvent) { ch <- swarmMsg{ev} })
			done := genDoneMsg{err: err, swarm: true}
			if res != nil {
				done.final = res.Final
			}
			ch <- done
			return
		}

		resp, err := runner.RunModel(ctx, role, model, llmsgs, input, func(tok string) error {
			ch <- tokenMsg{tok}
			return nil
		})
		done := genDoneMsg{err: err}
		if resp != nil {
			done.tokens = resp.EvalTokens
		}
		if err == nil && cfg.Verify {
			content := resp.Content
			done.report = verify.Run(ctx, content, verify.Options{
				Router: router, JudgeModel: cfg.GetJudgeModel(), MinScore: 70,
				Context: grounding,
			})
		}
		ch <- done
	}()

	return waitFor(ch)
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.layout()
		m.ready = true
		return m, nil

	case healthMsg:
		m.health = msg.ok
		return m, nil

	case modelsMsg:
		if msg.err == nil {
			m.models = msg.models
			if len(m.models) > 0 && m.curModel == "" {
				m.curModel = m.models[0].Name
			}
		}
		return m, nil

	case sessionsMsg:
		m.sessions = msg.sessions
		return m, nil

	case tokenMsg:
		m.acc.WriteString(msg.text)
		m.updateAssistant()
		m.refresh()
		return m, waitFor(m.ch)

	case swarmMsg:
		m.applySwarm(msg.ev)
		m.refresh()
		return m, waitFor(m.ch)

	case sourcesMsg:
		switch {
		case msg.err != nil:
			m.notice = "📚 retrieval failed: " + msg.err.Error()
		case len(msg.sources) > 0:
			m.curSources = msg.sources
			m.notice = fmt.Sprintf("📚 %d passage(s) retrieved from %s", len(msg.sources), filepath.Base(msg.dir))
		default:
			m.notice = "📚 no passages matched — answering from the prompt alone"
		}
		m.refresh()
		return m, waitFor(m.ch)

	case ragIndexedMsg:
		if msg.err != nil {
			m.err = fmt.Errorf("indexing %s: %w", msg.dir, msg.err)
			m.ragDir = ""
			m.refresh()
			return m, nil
		}
		m.ragDir = msg.dir
		m.ragChunks = msg.chunks
		m.notice = fmt.Sprintf("📚 indexed %s — %d passages from %d files", msg.dir, msg.chunks, msg.files)
		m.refresh()
		return m, nil

	case exportedMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.notice = "💾 exported to " + msg.path
		}
		m.refresh()
		return m, nil

	case genDoneMsg:
		if m.ch == nil {
			return m, nil // already finished
		}
		m.ch = nil
		m.streaming = false
		m.swarmRunning = false
		if msg.err != nil {
			m.err = msg.err
		}
		// Finalise the assistant message.
		if n := len(m.messages); n > 0 {
			if msg.swarm {
				m.messages[n-1].Content = msg.final
				m.messages[n-1].Agent = "swarm"
				m.messages[n-1].Tokens = msg.tokens
				// Keep the agent bus so the conversation can be re-read and
				// exported with everything the agents said to each other.
				m.messages[n-1].Swarm = swarmTurns(m.swarmBus)
			} else {
				m.messages[n-1].Content = m.acc.String()
				m.messages[n-1].Tokens = msg.tokens
			}
			m.messages[n-1].Sources = m.curSources
			if msg.report != nil {
				m.messages[n-1].Verification = &session.Verification{
					Score:       msg.report.Score,
					Verified:    msg.report.Verified,
					NeedsReview: msg.report.NeedsReview,
					Summary:     msg.report.Summary,
				}
			}
		}
		m.swarmOpen = map[string]int{}
		// Persist the turn exactly as it was shown (prompt, attachments,
		// sources and the swarm transcript included).
		if n := len(m.messages); n >= 2 {
			m.sess.Add(m.messages[n-2])
			m.sess.Add(m.messages[n-1])
		}
		_ = m.store.Save(m.sess)
		m.refresh()
		return m, loadSessions(m.store)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *Model) updateAssistant() {
	if n := len(m.messages); n > 0 {
		m.messages[n-1].Content = m.acc.String()
	}
}

func lastUser(msgs []session.Message) string {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == "user" {
			return msgs[i].Content
		}
	}
	return ""
}

func (m *Model) handleKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global quit.
	switch k.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		if m.screen != scrChat {
			m.screen = scrChat
			m.refresh()
			return m, nil
		}
	}

	// Screens with list navigation.
	if m.screen == scrModels || m.screen == scrAgents || m.screen == scrSessions {
		return m.handleListKey(k)
	}
	if m.screen == scrHelp {
		if k.String() == "enter" || k.String() == "q" {
			m.screen = scrChat
			m.refresh()
		}
		return m, nil
	}
	if m.screen == scrSwarm {
		if k.String() == "ctrl+b" || k.String() == "q" || k.String() == "enter" {
			m.screen = scrChat
			m.refresh()
			return m, nil
		}
		var cmd tea.Cmd
		m.vp, cmd = m.vp.Update(k)
		return m, cmd
	}

	// Chat screen shortcuts.
	switch k.String() {
	case "ctrl+q":
		return m, tea.Quit
	case "ctrl+n":
		m.newSession()
		return m, nil
	case "ctrl+m":
		m.screen = scrModels
		m.cursor = 0
		return m, nil
	case "ctrl+a":
		m.screen = scrAgents
		m.cursor = 0
		return m, nil
	case "ctrl+l":
		m.screen = scrSessions
		m.cursor = 0
		return m, loadSessions(m.store)
	case "ctrl+h", "f1":
		m.screen = scrHelp
		return m, nil
	case "ctrl+v":
		m.verifyOn = !m.verifyOn
		m.cfg.Verify = m.verifyOn
		_ = m.cfg.Save()
		m.notice = fmt.Sprintf("verify %s", onOff(m.verifyOn))
		return m, nil
	case "ctrl+t":
		m.swarmOn = !m.swarmOn
		m.notice = fmt.Sprintf("swarm mode %s", onOff(m.swarmOn))
		return m, nil
	case "ctrl+b":
		m.screen = scrSwarm
		m.refresh()
		return m, nil
	case "ctrl+e":
		format := "md"
		m.notice = "exporting…"
		return m, m.exportCmd(format)
	case "ctrl+u":
		m.messages = nil
		m.refresh()
		return m, nil
	case "enter":
		text := strings.TrimSpace(m.input.Value())
		if text == "" {
			return m, nil
		}
		m.input.SetValue("")
		if strings.HasPrefix(text, "/") {
			return m, m.runSlash(text)
		}
		if m.streaming {
			return m, nil
		}
		return m, m.generate(text)
	case "pgup", "pgdown", "up", "down", "home", "end":
		var cmd tea.Cmd
		m.vp, cmd = m.vp.Update(k)
		return m, cmd
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(k)
	return m, cmd
}

func (m *Model) handleListKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		m.cursor++
	case "enter":
		return m.selectCurrent()
	}
	return m, nil
}

func (m *Model) selectCurrent() (tea.Model, tea.Cmd) {
	switch m.screen {
	case scrModels:
		if m.cursor < len(m.models) {
			m.curModel = m.models[m.cursor].Name
			m.cfg.Model = m.curModel
			_ = m.cfg.Save()
			m.notice = "model → " + m.curModel
		}
	case scrAgents:
		if m.cursor < len(m.agents) {
			m.curAgent = m.agents[m.cursor].ID
			m.cfg.Agent = m.curAgent
			_ = m.cfg.Save()
			m.notice = "agent → " + m.curAgent
		}
	case scrSessions:
		if m.cursor < len(m.sessions) {
			if s, err := m.store.Load(m.sessions[m.cursor].ID); err == nil {
				m.sess = s
				m.messages = s.Messages
				m.curModel = s.Model
				m.swarmBus = lastSwarmBus(s.Messages)
				m.swarmFeed = nil
				m.swarmAfter = -1
				m.notice = "loaded " + s.Title
			}
		}
	}
	m.screen = scrChat
	m.cursor = 0
	m.refresh()
	return m, nil
}

func (m *Model) runSlash(text string) tea.Cmd {
	parts := strings.Fields(text)
	cmd := parts[0]
	arg := strings.TrimSpace(strings.TrimPrefix(text, cmd))
	switch cmd {
	case "/help":
		m.screen = scrHelp
	case "/model":
		if arg != "" {
			m.curModel = arg
			m.cfg.Model = arg
			_ = m.cfg.Save()
			m.notice = "model → " + arg
		} else {
			m.screen = scrModels
		}
	case "/agent":
		if arg != "" {
			if _, ok := agent.Get(arg); ok {
				m.curAgent = arg
				m.cfg.Agent = arg
				_ = m.cfg.Save()
				m.notice = "agent → " + arg
			} else {
				m.err = fmt.Errorf("unknown agent %q", arg)
			}
		} else {
			m.screen = scrAgents
		}
	case "/models":
		m.screen = scrModels
	case "/agents":
		m.screen = scrAgents
	case "/sessions":
		m.screen = scrSessions
		return loadSessions(m.store)
	case "/verify":
		m.verifyOn = !m.verifyOn
		m.cfg.Verify = m.verifyOn
		_ = m.cfg.Save()
		m.notice = "verify " + onOff(m.verifyOn)
	case "/team", "/swarm":
		m.swarmOn = !m.swarmOn
		m.notice = "swarm mode " + onOff(m.swarmOn)
	case "/bus":
		m.screen = scrSwarm
		m.refresh()
	case "/attach", "/file":
		if arg == "" {
			m.notice = fmt.Sprintf("attached: %s", strings.Join(m.attached, ", "))
			if len(m.attached) == 0 {
				m.notice = "usage: /attach <path> — or mention @path/to/file in a prompt"
			}
		} else if fi, err := os.Stat(strings.Trim(arg, `"`)); err != nil || fi.IsDir() {
			m.err = fmt.Errorf("cannot attach %q", arg)
		} else {
			m.attached = append(m.attached, strings.Trim(arg, `"`))
			m.notice = fmt.Sprintf("📎 %d file(s) attached — /detach to clear", len(m.attached))
		}
	case "/detach":
		m.attached = nil
		m.notice = "attachments cleared"
	case "/rag":
		m.ragSlash(arg)
	case "/export":
		format := arg
		if format == "" {
			format = "md"
		}
		m.notice = "exporting…"
		return m.exportCmd(format)
	case "/new":
		m.newSession()
	case "/clear":
		m.messages = nil
		m.swarmFeed = nil
		m.swarmBus = nil
		m.refresh()
	case "/pull":
		if arg == "" {
			m.err = fmt.Errorf("usage: /pull <model>")
		} else {
			m.notice = "pulling " + arg + " (see terminal for progress)"
			go func() {
				_ = llm.NewOllama(m.cfg.OllamaHost).Pull(context.Background(), arg, nil)
			}()
		}
	case "/exit", "/quit":
		return tea.Quit
	default:
		m.err = fmt.Errorf("unknown command %q — press Ctrl+H for help", cmd)
	}
	return nil
}

func (m *Model) newSession() {
	m.sess = session.New(m.curModel, m.curAgent)
	m.messages = nil
	m.swarmFeed = nil
	m.swarmBus = nil
	m.swarmAgents = nil
	m.swarmOpen = map[string]int{}
	m.acc.Reset()
	m.err = nil
	m.notice = "new session"
	m.refresh()
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func (m *Model) layout() {
	headerH := 2
	footerH := 4
	vpH := m.height - headerH - footerH
	if vpH < 3 {
		vpH = 3
	}
	w := m.width
	if w < 20 {
		w = 20
	}
	if !m.ready {
		m.vp = viewport.New(w, vpH)
	} else {
		m.vp.Width = w
		m.vp.Height = vpH
	}
	m.input.Width = w - 6
	m.refresh()
}

func (m *Model) refresh() {
	if m.screen == scrSwarm {
		m.vp.SetContent(m.renderBusBody())
	} else {
		m.vp.SetContent(m.renderMessages())
	}
	m.vp.GotoBottom()
}

func onOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}
