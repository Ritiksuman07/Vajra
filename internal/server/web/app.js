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
  team: false,
  busy: false,
};

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
  wrap.innerHTML = `<div class="avatar ${role}">${emoji}</div>
    <div class="bubble"><div class="who">${opts.who || (role === 'user' ? 'You' : 'Vajra')}
    ${opts.badge || ''}</div><div class="body">${contentHTML}</div></div>`;
  $('threadInner').appendChild(wrap);
  scrollDown();
  return wrap.querySelector('.body');
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

  addMsg('user', md(text), { who: 'You' });

  const bodyEl = addMsg('assistant', '<span class="cursor"></span>', {
    who: state.team ? 'Vajra Team' : 'Vajra', emoji: state.team ? '👥' : '🛡️',
  });
  let acc = '';

  const paint = throttle(() => { bodyEl.innerHTML = md(acc) + '<span class="cursor"></span>'; scrollDown(); });

  try {
    if (state.team) {
      await streamSSE('/api/team', { message: text }, (ev) => {
        if (ev.type === 'plan') { acc += `**Plan**\n\n${ev.text}\n\n---\n\n`; paint(true); }
        else if (ev.type === 'step_start') { acc += `\n\n**▶ ${ev.role}** — ${ev.text}\n\n`; paint(true); }
        else if (ev.type === 'token' || ev.type === 'final') { acc += ev.text; paint(); }
        else if (ev.type === 'step_done') { /* content already streamed */ }
        else if (ev.type === 'error') { addSys('Error: ' + ev.error, true); }
      });
      state.team = false; syncChips();
    } else {
      await streamSSE('/api/chat/stream', {
        message: text, model: state.model, agent: state.agent, verify: state.verify,
      }, (ev) => {
        if (ev.type === 'token') { acc += ev.text; paint(); }
        else if (ev.type === 'start' && ev.session_id) { state.sessionID = ev.session_id; }
        else if (ev.type === 'verifying') { paint(true); }
        else if (ev.type === 'verification') {
          const r = ev.report;
          const badge = `<span class="verify ${r.needs_review ? 'review' : ''}">${r.needs_review ? '⚠ review' : '✓ verified'} · ${Math.round(r.score)}%</span>`;
          bodyEl.parentElement.querySelector('.who').insertAdjacentHTML('beforeend', badge);
        }
        else if (ev.type === 'error') { addSys('Error: ' + ev.error, true); }
      });
    }
    if (!acc.trim()) acc = '_No content returned._';
    bodyEl.innerHTML = md(acc);
  } catch (e) {
    bodyEl.innerHTML = `<span style="color:var(--danger)">${esc(e.message)}</span>`;
  } finally {
    state.busy = false;
    $('sendBtn').disabled = false;
    loadSessions();
  }
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
      if (m.role === 'user') addMsg('user', md(m.content), { who: 'You' });
      else if (m.role === 'assistant') {
        let badge = '';
        if (m.verification) badge = `<span class="verify ${m.verification.needs_review ? 'review' : ''}">${m.verification.needs_review ? '⚠ review' : '✓ verified'} · ${Math.round(m.verification.score)}%</span>`;
        addMsg('assistant', md(m.content), { who: m.agent || 'Vajra', badge });
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
  location.reload === undefined; // no-op
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
  $('teamChip').classList.toggle('on', state.team);
  $('teamBtn2').classList.toggle('on', state.team);
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
  $('teamChip').onclick = () => { state.team = !state.team; syncChips(); };
  $('teamBtn2').onclick = () => { state.team = !state.team; syncChips(); };
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
  setInterval(loadHealth, 20000);
  setInterval(loadSessions, 30000);
}

window.addEventListener('DOMContentLoaded', boot);
