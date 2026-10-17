/* Vajra Web UI — vanilla JS, no build step, no CDN. */
'use strict';

const $ = (id) => document.getElementById(id);
const state = {
  cfg: null,
  agents: [],
  models: [],
  sessions: [],
  sessionID: null,
  model: '',
  agent: 'assistant',
  verify: false,
  swarm: false,
  busy: false,
  // Local context: files read in the browser, and the folder being searched.
  attachments: [],
  ragDir: '',
  ragChunks: 0,
  ctxMax: 0,
  ctxUsed: 0,
  ctxDropped: 0,
};

// Browser-side limits mirror the server's, so nothing is silently dropped in
// transit: 256 KiB per file, 16 files, 512 KiB in total.
const FILE_MAX = 256 * 1024;
const FILE_TOTAL = 512 * 1024;
const FILE_COUNT = 16;

function fmtK(n) {
  if (!n) return '0';
  return n < 1000 ? String(n) : (n / 1000).toFixed(1) + 'k';
}

/* ── tiny fetch helper ───────────────────────────────── */
async function api(path, opts = {}) {
  const res = await fetch(path, {
    headers: { 'Content-Type': 'application/json' },
    ...opts,
  });
  const text = await res.text();
  let data = null;
  try { data = text ? JSON.parse(text) : null; } catch { data = text; }
  if (!res.ok) throw new Error((data && data.error) || res.statusText);
  return data;
}

function toast(msg) {
  const t = $('toast');
  t.textContent = msg;
  t.classList.add('show');
  clearTimeout(toast._t);
  toast._t = setTimeout(() => t.classList.remove('show'), 2200);
}

/* ── Markdown (small, safe subset) ───────────────────── */
function esc(s) {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

function codeBlock(b) {
  const lang = (b.lang || 'text').toLowerCase();
  return `<div class="code"><div class="code-head"><span class="code-lang">${esc(lang)}</span>` +
    `<button class="code-copy" data-code="${encodeURIComponent(b.code)}">copy</button></div>` +
    `<pre><code>${esc(b.code)}</code></pre></div>`;
}

function listify(h) {
  const lines = h.split('\n');
  const out = [];
  let inUl = false, inOl = false;
  const close = () => { if (inUl) { out.push('</ul>'); inUl = false; } if (inOl) { out.push('</ol>'); inOl = false; } };
  for (const line of lines) {
    const ul = /^\s*[-*]\s+(.*)$/.exec(line);
    const ol = /^\s*\d+\.\s+(.*)$/.exec(line);
    if (ul) { if (inOl) { out.push('</ol>'); inOl = false; } if (!inUl) { out.push('<ul>'); inUl = true; } out.push('<li>' + ul[1] + '</li>'); }
    else if (ol) { if (inUl) { out.push('</ul>'); inUl = false; } if (!inOl) { out.push('<ol>'); inOl = true; } out.push('<li>' + ol[1] + '</li>'); }
    else { close(); out.push(line); }
  }
  close();
  return out.join('\n');
}

function md(src) {
  if (!src) return '';
  const blocks = [];
  src = src.replace(/```([\w+-]*)\n?([\s\S]*?)```/g, (_m, lang, code) => {
    blocks.push({ lang, code });
    return `\u0000B${blocks.length - 1}\u0000`;
  });
  let h = esc(src);
  h = h.replace(/^### (.*)$/gm, '<h3>$1</h3>');
  h = h.replace(/^## (.*)$/gm, '<h2>$1</h2>');
  h = h.replace(/^# (.*)$/gm, '<h1>$1</h1>');
  h = h.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>');
  h = h.replace(/(^|[^*\w])\*([^*\n]+)\*/g, '$1<em>$2</em>');
  h = h.replace(/`([^`]+)`/g, '<code>$1</code>');
  h = h.replace(/\[([^\]]+)\]\((https?:\/\/[^\s)]+)\)/g, '<a href="$2" target="_blank" rel="noopener">$1</a>');
  h = h.replace(/^&gt; (.*)$/gm, '<blockquote>$1</blockquote>');
  h = h.replace(/^(?:---+|\*\*\*+)$/gm, '<hr>');
  h = listify(h);
  h = h.split(/\n{2,}/).map((p) => {
    if (/^\s*<(h\d|ul|ol|blockquote|hr|pre|table)/.test(p)) return p;
    return '<p>' + p.replace(/\n/g, '<br>') + '</p>';
  }).join('\n');
  h = h.replace(/\u0000B(\d+)\u0000/g, (_m, i) => codeBlock(blocks[i]));
  return h;
}

/* ── Rendering ───────────────────────────────────────── */
function renderAgents() {
  const el = $('agentList');
  el.innerHTML = state.agents.map((a) =>
    `<div class="item ${a.id === state.agent ? 'active' : ''}" data-agent="${a.id}">
       <span class="em">${a.emoji}</span><span class="grow">${a.name}</span>
     </div>`).join('');
  el.querySelectorAll('[data-agent]').forEach((n) =>
    n.onclick = () => { state.agent = n.dataset.agent; renderAgents(); $('defAgent').value = state.agent; });

  const sel = $('defAgent');
  if (sel && !sel.dataset.filled) {
    sel.innerHTML = state.agents.map((a) => `<option value="${a.id}">${a.emoji} ${a.name}</option>`).join('');
    sel.dataset.filled = '1';
  }
}

function renderSessions() {
  const el = $('sessionList');
  if (!state.sessions.length) { el.innerHTML = `<div class="item" style="cursor:default">No saved sessions yet</div>`; return; }
  el.innerHTML = state.sessions.map((s) =>
    `<div class="item ${s.id === state.sessionID ? 'active' : ''}" data-sid="${s.id}">
       <span class="em">💬</span><span class="grow">${esc(s.title || 'Untitled')}</span>
       <span class="del" data-del="${s.id}" title="Delete">✕</span>
     </div>`).join('');
  el.querySelectorAll('[data-sid]').forEach((n) =>
    n.onclick = (e) => { if (e.target.dataset.del) return; openSession(n.dataset.sid); });
  el.querySelectorAll('[data-del]').forEach((n) =>
    n.onclick = async (e) => { e.stopPropagation(); await api('/api/sessions/' + n.dataset.del, { method: 'DELETE' }); if (state.sessionID === n.dataset.del) newChat(); loadSessions(); });
}

function clearThread() {
  const inner = $('threadInner');
  inner.innerHTML = '';
  inner.classList.remove('streaming');
}
function addMsg(role, contentHTML, opts = {}) {
  const wrap = document.createElement('div');
  wrap.className = 'msg';
  const emoji = role === 'user' ? '🧑' : (opts.emoji || '🛡️');
  let extra = '';
  if (opts.attachments && opts.attachments.length) {
    extra = `<div class="attach-line">📎 ${opts.attachments.map(esc).join(', ')}</div>`;
  }
  if (opts.sources && opts.sources.length) {
    extra += `<div class="source-line">📚 ${opts.sources.map(esc).join(', ')}</div>`;
  }
  wrap.innerHTML = `<div class="avatar ${role}">${emoji}</div>
    <div class="bubble"><div class="who">${opts.who || (role === 'user' ? 'You' : 'Vajra')}
    ${opts.badge || ''}</div><div class="body">${contentHTML}</div>${extra}</div>`;
  $('threadInner').appendChild(wrap);
  scrollDown();
  return wrap.querySelector('.body');
}

// swarmDetails renders a stored agent transcript as a collapsed panel, so a
// re-opened session still shows what the agents said to each other.
function swarmDetails(turns) {
  if (!turns || !turns.length) return '';
  const rows = turns.filter((t) => t.kind !== 'synthesis').map((t) =>
    `<div class="bus-row"><span class="bus-who">${esc(t.from)} → ${esc(t.to === 'all' ? 'everyone' : t.to)}</span>
     <span class="pill tiny">${esc(t.kind || '')}</span>
     <pre class="bus-text">${esc(t.text || '')}</pre></div>`).join('');
  return `<details class="bus"><summary>🐝 agent bus — ${turns.length} messages</summary>${rows}</details>`;
}
function addSys(text, isErr) {
  const d = document.createElement('div');
  d.className = 'sys' + (isErr ? ' error' : '');
  d.textContent = text;
  $('threadInner').appendChild(d);
  scrollDown();
  return d;
}
function scrollDown() {
  const t = $('thread');
  requestAnimationFrame(() => { t.scrollTop = t.scrollHeight; });
}
function hideWelcome() { const w = $('welcome'); if (w) w.remove(); }

/* ── Attachments (read locally in the browser) ───────── */
function totalBytes() {
  return state.attachments.reduce((n, f) => n + f.content.length, 0);
}

async function addFiles(fileList) {
  const files = Array.from(fileList || []);
  for (const f of files) {
    if (state.attachments.length >= FILE_COUNT) { toast(`At most ${FILE_COUNT} files`); break; }
    if (f.size > FILE_MAX) { toast(`${f.name} is larger than 256 KiB — skipped`); continue; }
    if (totalBytes() + f.size > FILE_TOTAL) { toast('Attachment budget (512 KiB) reached'); break; }
    try {
      const content = await f.text();
      if (content.includes('\u0000')) { toast(`${f.name} looks binary — skipped`); continue; }
      if (state.attachments.some((a) => a.name === f.name)) continue;
      state.attachments.push({ name: f.name, content, bytes: f.size });
    } catch { toast(`Could not read ${f.name}`); }
  }
  renderAttachments();
}

function renderAttachments() {
  const el = $('attachments');
  if (!el) return;
  if (!state.attachments.length) { el.innerHTML = ''; el.classList.remove('show'); return; }
  el.classList.add('show');
  el.innerHTML = state.attachments.map((f, i) =>
    `<span class="attach">📎 ${esc(f.name)}<button data-drop="${i}" title="remove">✕</button></span>`).join('') +
    `<span class="attach-note">${state.attachments.length} file(s) · ${fmtK(totalBytes())} chars · read locally</span>`;
  el.querySelectorAll('[data-drop]').forEach((n) =>
    n.onclick = () => { state.attachments.splice(Number(n.dataset.drop), 1); renderAttachments(); });
}

/* Context meter + retrieval chip */
function renderContext() {
  const pill = $('ctxPill');
  if (pill) {
    pill.textContent = state.ctxMax
      ? `ctx ${fmtK(state.ctxUsed || 0)}/${fmtK(state.ctxMax)}`
      : 'ctx —';
    pill.title = state.ctxDropped
      ? `${state.ctxDropped} older message(s) were trimmed to fit the context window`
      : 'How much of the model\'s context window this conversation uses';
  }
  const chip = $('ragChip');
  if (chip) {
    if (state.ragDir) {
      const name = state.ragDir.replace(/[\\/]+$/, '').split(/[\\/]/).pop();
      chip.textContent = `📚 ${name}`;
      chip.classList.add('on');
      chip.title = `Answering with passages from ${state.ragDir}`;
    } else {
      chip.textContent = '📚 Folder';
      chip.classList.remove('on');
      chip.title = 'Answer using passages from a folder on this machine';
    }
  }
  syncChips();
}

/* ── Local folder (RAG) modal ───────────────────────── */
function openRag() {
  $('ragDir').value = state.ragDir || '';
  $('ragProgress').textContent = '';
  $('ragStatus').innerHTML = state.ragDir
    ? `Searching <code>${esc(state.ragDir)}</code>${state.ragChunks ? ' · ' + state.ragChunks + ' passages' : ''}`
    : 'No folder selected — retrieval is off.';
  $('ragBack').classList.add('open');
}
function closeRag() { $('ragBack').classList.remove('open'); }

async function ragRefresh(dir) {
  try {
    const d = await api('/api/rag?dir=' + encodeURIComponent(dir));
    if (d.indexed) {
      $('ragStatus').innerHTML = `✓ indexed · ${d.chunks} passages from ${d.files} files · <code>${esc(d.model || '')}</code>`;
      return d;
    }
    $('ragStatus').innerHTML = 'Not indexed yet — press <strong>Index / re-index</strong>.';
  } catch { $('ragStatus').textContent = 'Could not read the index status.'; }
  return null;
}

async function ragIndex() {
  const dir = $('ragDir').value.trim();
  if (!dir) { toast('Enter a folder path'); return; }
  const prog = $('ragProgress');
  prog.textContent = 'Starting…';
  let indexed = null;
  try {
    await streamSSE('/api/rag/index', { dir }, (ev) => {
      if (ev.type === 'progress') {
        if (ev.stage === 'reading') prog.textContent = `reading ${ev.done} files… ${ev.current || ''}`;
        else prog.textContent = `embedding ${ev.done}/${ev.total} (${ev.total ? Math.round(ev.done * 100 / ev.total) : 0}%) ${ev.current || ''}`;
      } else if (ev.type === 'done') {
        indexed = ev;
        prog.textContent = `✓ ${ev.chunks} passages from ${ev.files} files`;
      } else if (ev.type === 'error') {
        prog.textContent = '✗ ' + ev.error;
        addSys('Indexing failed: ' + ev.error, true);
      }
    });
  } catch (e) { prog.textContent = '✗ ' + e.message; return; }
  if (indexed) {
    await ragRefresh(dir);
    toast('Indexed ' + indexed.files + ' files');
  }
}

function ragUse() {
  const dir = $('ragDir').value.trim();
  if (!dir) { toast('Enter a folder path'); return; }
  state.ragDir = dir;
  api('/api/rag?dir=' + encodeURIComponent(dir)).then((d) => { state.ragChunks = d.chunks || 0; renderContext(); }).catch(() => {});
  toast('Answers will cite passages from ' + dir);
  closeRag();
}
function ragOff() {
  state.ragDir = '';
  state.ragChunks = 0;
  renderContext();
  toast('Retrieval off');
  closeRag();
}

/* ── Export ─────────────────────────────────────────── */
function exportSession() {
  if (!state.sessionID) { toast('Send a message first — there is nothing to export yet'); return; }
  window.location.href = '/api/sessions/' + state.sessionID + '/export?format=md';
}

/* ── Swarm rendering ─────────────────────────────────── */
function agentMeta(role) {
  return state.agents.find((a) => a.id === role) || { emoji: '•', name: role };
}
function destLabel(to) {
  if (to === 'user') return 'you';
  if (!to || to === 'all') return 'everyone';
  const m = agentMeta(to);
  return m.emoji + ' ' + m.name;
}
function addSwarmPlan(ev) {
  const wrap = document.createElement('div');
  wrap.className = 'swarm-plan';
  wrap.innerHTML = `<div class="sp-head">🐝 swarm plan · ${(ev.agents || []).length} agents working in parallel</div>` + md(ev.text || '');
  $('threadInner').appendChild(wrap);
  scrollDown();
}
// addAgentBubble creates a streaming bubble for one agent and returns a
// handle the caller uses to append tokens.
function addAgentBubble(ev) {
  const m = agentMeta(ev.role);
  const wrap = document.createElement('div');
  wrap.className = 'msg' + (ev.kind === 'synthesis' ? ' final' : '');
  wrap.innerHTML = `<div class="avatar assistant">${m.emoji}</div>
    <div class="bubble"><div class="who">${esc(m.name)}
      <span class="arrow">→ ${esc(destLabel(ev.to))}</span>
      <span class="pill tiny">${esc(ev.kind || '')}</span></div>
    <div class="body"><span class="cursor"></span></div></div>`;
  $('threadInner').appendChild(wrap);
  scrollDown();
  const body = wrap.querySelector('.body');
  const handle = { acc: '', body };
  handle.paint = throttle(() => { body.innerHTML = md(handle.acc) + '<span class="cursor"></span>'; scrollDown(); });
  handle.finish = () => { if (!handle.acc.trim()) handle.acc = '_no output_'; body.innerHTML = md(handle.acc); };
  return handle;
}
function addHandoff(ev) {
  const from = agentMeta(ev.from);
  const to = agentMeta(ev.to);
  const d = document.createElement('div');
  d.className = 'handoff';
  d.textContent = `✉ ${from.name} → ${to.name}: ${ev.text || ''}`;
  $('threadInner').appendChild(d);
  scrollDown();
}
// streamSwarm runs the multi-agent swarm for the given message.
async function streamSwarm(text, files, ragDir) {
  const open = {};
  await streamSSE('/api/swarm', {
    message: text, handoffs: 1, concurrent: 3,
    files: files || [], rag_dir: ragDir || '',
  }, (ev) => {
    switch (ev.type) {
      case 'plan': addSwarmPlan(ev); break;
      case 'round': addSys('── ' + ev.text + ' ──'); break;
      case 'context': annotateLast(ev); break;
      case 'budget': {
        state.ctxUsed = ev.used; state.ctxMax = ev.max || ev.limit || 0;
        renderContext();
        break;
      }
      case 'agent_start': open[ev.role] = addAgentBubble(ev); break;
      case 'token': {
        const h = open[ev.role];
        if (h) { h.acc += ev.text; h.paint(); }
        break;
      }
      case 'agent_done': {
        const h = open[ev.role];
        if (h) { if (ev.text) h.acc = ev.text; h.finish(); }
        delete open[ev.role];
        break;
      }
      case 'bus': if (ev.kind === 'handoff') addHandoff(ev); break;
      case 'done': if (ev.session_id) state.sessionID = ev.session_id; break;
      case 'error': addSys('Error: ' + ev.error, true); break;
    }
  });
}

/* ── SSE ─────────────────────────────────────────────── */
async function streamSSE(path, body, onEvent) {
  const res = await fetch(path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
  if (!res.ok || !res.body) {
    let msg = res.statusText;
    try { msg = (await res.json()).error || msg; } catch {}
    throw new Error(msg);
  }
  const reader = res.body.getReader();
  const dec = new TextDecoder();
  let buf = '';
  for (;;) {
    const { value, done } = await reader.read();
    if (done) break;
    buf += dec.decode(value, { stream: true });
    const parts = buf.split('\n\n');
    buf = parts.pop();
    for (const part of parts) {
      const line = part.split('\n').find((l) => l.startsWith('data:'));
      if (!line) continue;
      try { onEvent(JSON.parse(line.slice(5).trim())); } catch {}
    }
  }
}

/* ── Sending ─────────────────────────────────────────── */
async function send() {
  const input = $('input');
  const text = input.value.trim();
  if (!text || state.busy) return;
  hideWelcome();
  input.value = '';
  autoResize();
  state.busy = true;
  $('sendBtn').disabled = true;

  addMsg('user', md(text), { who: 'You', attachments: state.attachments.map((f) => f.name) });

  const files = state.attachments.map((f) => ({ name: f.name, content: f.content }));
  const ragDir = state.ragDir;

  try {
    if (state.swarm) {
      await streamSwarm(text, files, ragDir);
      syncChips();
      return;
    }

    const bodyEl = addMsg('assistant', '<span class="cursor"></span>', { who: 'Vajra', emoji: '🛡️' });
    let acc = '';
    const paint = throttle(() => { bodyEl.innerHTML = md(acc) + '<span class="cursor"></span>'; scrollDown(); });

    await streamSSE('/api/chat/stream', {
      message: text, model: state.model, agent: state.agent, verify: state.verify,
      files, rag_dir: ragDir,
    }, (ev) => {
      if (ev.type === 'token') { acc += ev.text; paint(); }
      else if (ev.type === 'start' && ev.session_id) { state.sessionID = ev.session_id; }
      else if (ev.type === 'context') { annotateLast(ev); }
      else if (ev.type === 'budget') {
        state.ctxUsed = ev.used; state.ctxMax = ev.max || ev.limit || 0; state.ctxDropped = ev.dropped || 0;
        renderContext();
        if (ev.dropped) addSys(`Trimmed ${ev.dropped} older message(s) to fit ${fmtK(state.ctxMax)} tokens`);
      }
      else if (ev.type === 'verifying') { paint(true); }
      else if (ev.type === 'verification') {
        const r = ev.report;
        const badge = `<span class="verify ${r.needs_review ? 'review' : ''}">${r.needs_review ? '⚠ review' : '✓ verified'} · ${Math.round(r.score)}%</span>`;
        bodyEl.parentElement.querySelector('.who').insertAdjacentHTML('beforeend', badge);
      }
      else if (ev.type === 'error') { addSys('Error: ' + ev.error, true); }
    });
    if (!acc.trim()) acc = '_No content returned._';
    bodyEl.innerHTML = md(acc);
  } catch (e) {
    addSys('Error: ' + e.message, true);
  } finally {
    state.busy = false;
    $('sendBtn').disabled = false;
    loadSessions();
  }
}

// annotateLast shows, under the message just sent, which local files and which
// retrieved passages were actually used.
function annotateLast(ev) {
  const inner = $('threadInner');
  const target = inner.lastElementChild;
  if (!target) return;
  const bits = [];
  if ((ev.attachments || []).length) bits.push('📎 ' + ev.attachments.join(', '));
  if ((ev.sources || []).length) bits.push('📚 ' + ev.sources.join(', '));
  if (!bits.length) return;
  const d = document.createElement('div');
  d.className = 'sys note';
  d.textContent = bits.join('  ·  ');
  target.appendChild(d);
  scrollDown();
}

function throttle(fn) {
  let queued = false;
  return (immediate) => {
    if (immediate) { fn(); return; }
    if (queued) return;
    queued = true;
    requestAnimationFrame(() => { queued = false; fn(); });
  };
}

/* ── Sessions ────────────────────────────────────────── */
async function loadSessions() {
  try {
    const d = await api('/api/sessions');
    state.sessions = d.sessions || [];
    renderSessions();
  } catch {}
}
async function openSession(id) {
  try {
    const s = await api('/api/sessions/' + id);
    state.sessionID = s.id;
    if (s.model) { state.model = s.model; $('modelSelect').value = s.model; }
    clearThread();
    $('threadTitle').textContent = s.title || 'Chat';
    (s.messages || []).forEach((m) => {
      if (m.role === 'user') {
        addMsg('user', md(m.content), { who: 'You', attachments: m.attachments });
      } else if (m.role === 'assistant') {
        let badge = '';
        if (m.verification) badge = `<span class="verify ${m.verification.needs_review ? 'review' : ''}">${m.verification.needs_review ? '⚠ review' : '✓ verified'} · ${Math.round(m.verification.score)}%</span>`;
        const bodyEl = addMsg('assistant', md(m.content), {
          who: m.agent || 'Vajra', badge, sources: m.sources,
        });
        if (m.swarm && m.swarm.length) bodyEl.parentElement.insertAdjacentHTML('beforeend', swarmDetails(m.swarm));
      }
    });
    renderSessions();
    closeSidebar();
  } catch (e) { toast('Could not load session'); }
}
function newChat() {
  state.sessionID = null;
  clearThread();
  $('threadTitle').textContent = 'New chat';
  state.ctxUsed = 0; state.ctxDropped = 0;
  renderContext();
  const inner = $('threadInner');
  inner.innerHTML = welcomeHTML();
  bindSuggestions();
}

function welcomeHTML() {
  return `<div class="welcome" id="welcome">
    <div class="hero">🛡️</div>
    <h2>Your machine. Your models. Your data.</h2>
    <p>Vajra runs entirely on your computer through Ollama. Prompts never leave your device unless you add a cloud provider.</p>
    <div class="suggestions">
      <div class="suggestion" data-prompt="Explain how a transformer attention head works, with a small code example."><strong>Learn something</strong><span>Explain a concept with an example</span></div>
      <div class="suggestion" data-prompt="Write a Python function that reads a CSV and returns the top 5 rows by a given column."><strong>Write code</strong><span>Generate a tested function</span></div>
      <div class="suggestion" data-prompt="Plan a 3-step approach to add automated tests to a small Go project."><strong>Plan work</strong><span>Decompose a goal into steps</span></div>
      <div class="suggestion" data-prompt="Summarise the trade-offs between SQLite and Postgres for a local desktop app."><strong>Compare options</strong><span>Get a balanced analysis</span></div>
    </div></div>`;
}
function bindSuggestions() {
  document.querySelectorAll('.suggestion').forEach((n) =>
    n.onclick = () => { $('input').value = n.dataset.prompt; $('input').focus(); autoResize(); });
}

/* ── Settings / providers ────────────────────────────── */
async function openSettings() {
  try {
    const [cfg, provs] = await Promise.all([api('/api/config'), api('/api/providers')]);
    state.cfg = cfg;
    $('temp').value = cfg.temperature ?? 0.7;
    $('tempVal').textContent = (cfg.temperature ?? 0.7).toFixed(2);
    $('verifyDefault').checked = !!cfg.verify;
    $('defAgent').value = cfg.agent || 'assistant';
    $('provList').innerHTML = (provs.providers || []).map((p) =>
      `<div class="prov"><span class="grow"><div class="name">${esc(p.name)}</div>
       <div class="key">${p.has_key ? '••••••••' : 'no key'}${p.base_url ? ' · ' + esc(p.base_url) : ''}</div></span>
       ${p.kind === 'ollama' ? '<span class="pill">local</span>' : `<button class="btn ghost danger" data-rm="${p.name}">Remove</button>`}</div>`).join('');
    $('provList').querySelectorAll('[data-rm]').forEach((n) =>
      n.onclick = async () => { await api('/api/providers/' + n.dataset.rm, { method: 'DELETE' }); toast('Removed ' + n.dataset.rm); openSettings(); loadModels(); });
    $('modalBack').classList.add('open');
  } catch (e) { toast('Settings unavailable'); }
}
function closeModal() { $('modalBack').classList.remove('open'); }

async function addProvider() {
  const kind = $('provKind').value;
  const key = $('provKey').value.trim();
  const url = $('provURL').value.trim();
  if (kind !== 'lmstudio' && !key && kind !== 'custom') { toast('API key required'); return; }
  try {
    await api('/api/providers', { method: 'POST', body: JSON.stringify({ kind, name: kind, api_key: key, base_url: url }) });
    $('provKey').value = ''; $('provURL').value = '';
    toast('Added ' + kind);
    openSettings(); loadModels();
  } catch (e) { toast(e.message); }
}

/* ── Boot ────────────────────────────────────────────── */
async function loadModels() {
  try {
    const d = await api('/api/models');
    state.models = d.models || [];
    const sel = $('modelSelect');
    sel.innerHTML = state.models.map((m) =>
      `<option value="${m.name}">${m.name}${m.backend ? '  ·  ' + m.backend : ''}</option>`).join('');
    if (!state.models.find((m) => m.name === state.model)) {
      state.model = state.models[0] ? state.models[0].name : '';
    }
    if (state.model) sel.value = state.model;
  } catch {
    $('modelSelect').innerHTML = '<option value="">no models — start Ollama</option>';
  }
}
async function loadHealth() {
  try {
    const [h, s] = await Promise.all([api('/api/health'), api('/api/status')]);
    $('versionPill').textContent = 'v' + h.version;
    const health = s.health || {};
    const local = !!health.ollama;
    const cloud = Object.entries(health).filter(([k, v]) => k !== 'ollama' && v).map(([k]) => k);
    $('statusDot').className = 'dot ' + (local || cloud.length ? 'on' : 'off');
    $('statusText').textContent = local ? 'ollama ready' : (cloud.length ? cloud.join(', ') + ' ready' : 'offline');
    $('backendPill').textContent = state.models.find((m) => m.name === state.model)?.backend || 'ollama';
  } catch {
    $('statusDot').className = 'dot off';
    $('statusText').textContent = 'service offline';
  }
}
function syncChips() {
  $('verifyChip').classList.toggle('on', state.verify);
  $('teamChip').classList.toggle('on', state.swarm);
  $('teamBtn2').classList.toggle('on', state.swarm);
  $('attachChip').classList.toggle('on', state.attachments.length > 0);
  $('input').placeholder = state.swarm
    ? 'Give the swarm a goal…  (agents will plan, work in parallel and review)'
    : 'Message Vajra…  (Enter to send · Shift+Enter for newline)';
}
function autoResize() {
  const t = $('input');
  t.style.height = 'auto';
  t.style.height = Math.min(t.scrollHeight, 240) + 'px';
}
function openSidebar() { $('sidebar').classList.add('open'); }
function closeSidebar() { $('sidebar').classList.remove('open'); }

async function boot() {
  // Theme from localStorage.
  const theme = localStorage.getItem('vajra-theme');
  if (theme) document.documentElement.dataset.theme = theme;

  try {
    const agents = await api('/api/agents');
    state.agents = agents.agents || [];
  } catch {}
  renderAgents();

  await loadModels();
  await loadHealth();
  await loadSessions();

  const input = $('input');
  input.addEventListener('input', autoResize);
  input.addEventListener('keydown', (e) => {
    if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); send(); }
  });
  $('sendBtn').onclick = send;
  $('newChat').onclick = newChat;
  $('settingsBtn').onclick = openSettings;
  $('closeModal').onclick = closeModal;
  $('addProv').onclick = addProvider;
  $('modalBack').onclick = (e) => { if (e.target === $('modalBack')) closeModal(); };
  $('modelSelect').onchange = (e) => { state.model = e.target.value; loadHealth(); };
  $('verifyChip').onclick = () => { state.verify = !state.verify; syncChips(); };
  $('teamChip').onclick = () => { state.swarm = !state.swarm; syncChips(); };
  $('teamBtn2').onclick = () => { state.swarm = !state.swarm; syncChips(); };
  $('exportBtn').onclick = exportSession;

  // Attachments: picker plus drag-and-drop anywhere over the thread.
  $('attachChip').onclick = () => $('fileInput').click();
  $('fileInput').onchange = (e) => { addFiles(e.target.files); e.target.value = ''; };
  const thread = $('thread');
  ['dragenter', 'dragover'].forEach((ev) =>
    thread.addEventListener(ev, (e) => { e.preventDefault(); thread.classList.add('dragging'); }));
  ['dragleave', 'drop'].forEach((ev) =>
    thread.addEventListener(ev, (e) => { e.preventDefault(); thread.classList.remove('dragging'); }));
  thread.addEventListener('drop', (e) => { if (e.dataTransfer && e.dataTransfer.files) addFiles(e.dataTransfer.files); });
  // A file dropped on the composer should not navigate away.
  document.addEventListener('dragover', (e) => e.preventDefault());
  document.addEventListener('drop', (e) => e.preventDefault());

  // Local folder retrieval.
  $('ragChip').onclick = openRag;
  $('ragClose').onclick = closeRag;
  $('ragUse').onclick = ragUse;
  $('ragOff').onclick = ragOff;
  $('ragIndex').onclick = ragIndex;
  $('ragBack').onclick = (e) => { if (e.target === $('ragBack')) closeRag(); };
  $('ragDir').addEventListener('keydown', (e) => {
    if (e.key === 'Enter') { e.preventDefault(); ragIndex(); }
  });
  $('themeBtn').onclick = () => {
    const next = document.documentElement.dataset.theme === 'light' ? 'dark' : 'light';
    document.documentElement.dataset.theme = next;
    localStorage.setItem('vajra-theme', next);
  };
  $('menuBtn').onclick = openSidebar;
  $('temp').oninput = (e) => { $('tempVal').textContent = parseFloat(e.target.value).toFixed(2); };
  $('temp').onchange = (e) => api('/api/config', { method: 'POST', body: JSON.stringify({ temperature: parseFloat(e.target.value) }) });
  $('verifyDefault').onchange = (e) => { state.verify = e.target.checked; syncChips(); api('/api/config', { method: 'POST', body: JSON.stringify({ verify: e.target.checked }) }); };
  $('defAgent').onchange = (e) => { state.agent = e.target.value; renderAgents(); api('/api/config', { method: 'POST', body: JSON.stringify({ agent: e.target.value }) }); };

  // Copy buttons (delegated).
  document.addEventListener('click', (e) => {
    const b = e.target.closest('.code-copy');
    if (!b) return;
    navigator.clipboard.writeText(decodeURIComponent(b.dataset.code || ''));
    b.textContent = 'copied!';
    setTimeout(() => (b.textContent = 'copy'), 1200);
  });
  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') { closeModal(); closeSidebar(); }
    if ((e.ctrlKey || e.metaKey) && e.key === 'k') { e.preventDefault(); $('input').focus(); }
  });
  document.addEventListener('click', (e) => {
    if (!e.target.closest('.sidebar') && !e.target.closest('#menuBtn')) closeSidebar();
  });

  bindSuggestions();
  syncChips();
  renderAttachments();
  renderContext();
  setInterval(loadHealth, 20000);
  setInterval(loadSessions, 30000);
}

window.addEventListener('DOMContentLoaded', boot);
