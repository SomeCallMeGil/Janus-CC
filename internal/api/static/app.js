// Janus Web UI

const API  = window.location.origin + '/api/v1';
const WS   = `ws://${window.location.host}/ws/v1/activity`;

let ws            = null;
let autoScroll    = true;
let activeEncrypt = null; // scenario id awaiting encrypt submission

// ─── Init ────────────────────────────────────────────────────────────────────

window.addEventListener('DOMContentLoaded', () => {
  showTab('generate');
  connectWebSocket();
});

// ─── Tabs ─────────────────────────────────────────────────────────────────────

function showTab(name) {
  document.querySelectorAll('.tab-content').forEach(el => el.classList.add('hidden'));
  document.getElementById(`content-${name}`).classList.remove('hidden');

  document.querySelectorAll('[id^="tab-"]').forEach(btn => {
    btn.classList.remove('tab-active');
    btn.classList.add('tab-inactive');
  });
  const btn = document.getElementById(`tab-${name}`);
  btn.classList.remove('tab-inactive');
  btn.classList.add('tab-active');

  if (name === 'generate')  loadProfiles();
  if (name === 'scenarios') loadScenarios();
  if (name === 'jobs')      loadJobs();
}

// ─── Toast ────────────────────────────────────────────────────────────────────

function toast(message, type = 'info') {
  const colors = {
    info:    'bg-gray-800 text-white',
    success: 'bg-green-600 text-white',
    error:   'bg-red-600 text-white',
    warning: 'bg-yellow-500 text-white',
  };
  const el = document.createElement('div');
  el.className = `toast rounded-lg shadow-lg px-4 py-3 text-sm ${colors[type] || colors.info} fade-in`;
  el.textContent = message;
  document.getElementById('toast-container').appendChild(el);
  setTimeout(() => el.remove(), 4000);
}

// ─── Modal ────────────────────────────────────────────────────────────────────

function openModal(id)  { document.getElementById(id).classList.remove('hidden'); }
function closeModal(id) { document.getElementById(id).classList.add('hidden'); }

// ─── Profiles ─────────────────────────────────────────────────────────────────

async function loadProfiles() {
  try {
    const res  = await fetch(`${API}/profiles`);
    const data = await res.json();
    renderProfiles(data.profiles || []);
  } catch (e) {
    document.getElementById('profiles-grid').innerHTML =
      '<p class="text-red-500 col-span-3">Failed to load profiles.</p>';
  }
}

function renderProfiles(profiles) {
  const grid = document.getElementById('profiles-grid');
  if (!profiles.length) {
    grid.innerHTML = '<p class="text-gray-400 col-span-3 text-sm">No profiles yet. Create one to get started.</p>';
    return;
  }

  const piiTypeColor = { standard: 'bg-blue-100 text-blue-700', healthcare: 'bg-green-100 text-green-700', financial: 'bg-yellow-100 text-yellow-800' };

  grid.innerHTML = profiles.map(p => {
    const opts       = p.options || {};
    const mode       = opts.total_size ? 'size' : 'count';
    const constraint = opts.total_size  ? opts.total_size : `${(opts.file_count || 0).toLocaleString()} files`;
    const piiType    = opts.pii_type || 'standard';
    const piiColor   = piiTypeColor[piiType] || 'bg-gray-100 text-gray-700';

    return `
      <div class="bg-white rounded-xl shadow-sm border border-gray-100 p-5 flex flex-col justify-between">
        <div>
          <div class="flex items-start justify-between mb-2">
            <h4 class="font-semibold text-gray-800 leading-tight">${esc(p.name)}</h4>
            <span class="text-xs px-2 py-0.5 rounded-full ${piiColor} ml-2 whitespace-nowrap">${esc(piiType)}</span>
          </div>
          ${p.description ? `<p class="text-xs text-gray-500 mb-3">${esc(p.description)}</p>` : '<p class="mb-3"></p>'}
          <div class="grid grid-cols-2 gap-x-4 gap-y-1 text-xs text-gray-600 mb-4">
            <span class="text-gray-400">Mode</span>      <span class="font-medium">${mode}</span>
            <span class="text-gray-400">Size/Count</span><span class="font-medium">${esc(constraint)}</span>
            <span class="text-gray-400">PII</span>       <span class="font-medium">${opts.pii_percent || 0}%</span>
            <span class="text-gray-400">Files</span>     <span class="font-medium">${esc(opts.file_size_min || '—')} – ${esc(opts.file_size_max || '—')}</span>
          </div>
        </div>
        <div class="flex gap-2 mt-auto">
          <button onclick="generateFromProfile('${p.id}')"
            class="flex-1 bg-indigo-600 text-white text-sm px-3 py-2 rounded-lg hover:bg-indigo-700 font-medium">
            Generate
          </button>
          <button onclick="deleteProfile('${p.id}', '${esc(p.name)}')"
            class="border text-gray-500 text-sm px-3 py-2 rounded-lg hover:bg-gray-50">
            Delete
          </button>
        </div>
      </div>
    `;
  }).join('');
}

function toggleProfileMode() {
  const mode = document.getElementById('cp-mode').value;
  document.getElementById('cp-count-wrap').classList.toggle('hidden', mode !== 'count');
  document.getElementById('cp-size-wrap').classList.toggle('hidden',  mode !== 'size');
}

async function submitCreateProfile() {
  const name        = document.getElementById('cp-name').value.trim();
  const description = document.getElementById('cp-description').value.trim();
  const mode        = document.getElementById('cp-mode').value;
  const output      = document.getElementById('cp-output').value.trim() || './payloads';
  const piiPercent  = parseFloat(document.getElementById('cp-pii-percent').value);
  const fillerPct   = 100 - piiPercent;
  const piiType     = document.getElementById('cp-pii-type').value;
  const sizeMin     = document.getElementById('cp-size-min').value.trim() || '1KB';
  const sizeMax     = document.getElementById('cp-size-max').value.trim() || '10MB';

  if (!name) { toast('Name is required', 'error'); return; }

  const options = {
    name,
    output_path:    output,
    file_size_min:  sizeMin,
    file_size_max:  sizeMax,
    pii_percent:    piiPercent,
    filler_percent: fillerPct,
    pii_type:       piiType,
    formats:        ['csv', 'json', 'txt'],
  };

  if (mode === 'count') {
    options.file_count = parseInt(document.getElementById('cp-file-count').value) || 1000;
  } else {
    const ts = document.getElementById('cp-total-size').value.trim();
    if (!ts) { toast('Total size is required for size mode', 'error'); return; }
    options.total_size = ts;
  }

  try {
    const res = await fetch(`${API}/profiles`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name, description, options }),
    });
    const data = await res.json();
    if (!res.ok) { toast(data.error || 'Failed to create profile', 'error'); return; }
    closeModal('modal-create-profile');
    toast(`Profile "${data.name}" created`, 'success');
    loadProfiles();
  } catch (e) {
    toast('Network error', 'error');
  }
}

async function deleteProfile(id, name) {
  if (!confirm(`Delete profile "${name}"?`)) return;
  try {
    const res = await fetch(`${API}/profiles/${id}`, { method: 'DELETE' });
    if (!res.ok) { toast('Failed to delete profile', 'error'); return; }
    toast(`Profile "${name}" deleted`);
    loadProfiles();
  } catch (e) {
    toast('Network error', 'error');
  }
}

async function generateFromProfile(id) {
  try {
    const res  = await fetch(`${API}/profiles/${id}/generate`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: '{}' });
    const data = await res.json();
    if (!res.ok) { toast(data.error || 'Failed to start generation', 'error'); return; }
    openProgressModal('Generating from profile...', data.scenario_id);
    toast('Generation started', 'success');
  } catch (e) {
    toast('Network error', 'error');
  }
}

// ─── Progress modal ───────────────────────────────────────────────────────────

let _activeScenarioId = null;

function openProgressModal(title, scenarioId) {
  _activeScenarioId = scenarioId || null;
  document.getElementById('progress-title').textContent       = title;
  document.getElementById('progress-scenario').textContent    = `scenario: ${(scenarioId || '').substring(0, 8)}`;
  document.getElementById('progress-files').textContent       = '0 / ? files';
  document.getElementById('progress-pct').textContent         = '0%';
  document.getElementById('progress-bar').style.width         = '0%';
  document.getElementById('progress-current-file').textContent = 'Starting...';
  document.getElementById('progress-bytes').textContent       = '0 MB written';
  document.getElementById('progress-dismiss-btn').classList.add('hidden');
  document.getElementById('progress-status-badge').textContent = 'Running';
  document.getElementById('progress-status-badge').className  = 'text-xs px-2 py-1 rounded-full badge-generating';
  // Reset controls to running state
  document.getElementById('progress-controls').classList.remove('hidden');
  document.getElementById('progress-pause-btn').classList.remove('hidden');
  document.getElementById('progress-resume-btn').classList.add('hidden');
  document.getElementById('progress-cancel-btn').classList.remove('hidden');
  openModal('modal-progress');
}

async function pauseGeneration() {
  if (!_activeScenarioId) return;
  try {
    await fetch(`${API}/scenarios/${_activeScenarioId}/generation/pause`, { method: 'POST' });
  } catch (e) { toast('Pause failed', 'error'); }
}

async function resumeGeneration() {
  if (!_activeScenarioId) return;
  try {
    await fetch(`${API}/scenarios/${_activeScenarioId}/generation/resume`, { method: 'POST' });
  } catch (e) { toast('Resume failed', 'error'); }
}

async function cancelGeneration() {
  if (!_activeScenarioId) return;
  try {
    await fetch(`${API}/scenarios/${_activeScenarioId}/generation/cancel`, { method: 'POST' });
  } catch (e) { toast('Cancel failed', 'error'); }
}

function updateProgressModal(event) {
  const current = event.current || 0;
  const total   = event.total   || 0;
  const pct     = (event.percent || 0).toFixed(1);
  const mb      = ((event.bytes_written || 0) / 1024 / 1024).toFixed(1);

  document.getElementById('progress-files').textContent        = `${current.toLocaleString()} / ${total.toLocaleString()} files`;
  document.getElementById('progress-pct').textContent          = `${pct}%`;
  document.getElementById('progress-bar').style.width          = `${pct}%`;
  document.getElementById('progress-current-file').textContent = event.current_file || '';
  document.getElementById('progress-bytes').textContent        = `${mb} MB written`;
}

function completeProgressModal(success, message, cancelled) {
  document.getElementById('progress-bar').style.width = '100%';
  const badge = document.getElementById('progress-status-badge');
  if (cancelled) {
    badge.textContent  = 'Cancelled';
    badge.className    = 'text-xs px-2 py-1 rounded-full badge-destroyed';
    document.getElementById('progress-current-file').textContent = 'Cancelled';
    document.getElementById('progress-bar').classList.replace('bg-indigo-500', 'bg-gray-400');
  } else if (success) {
    badge.textContent  = 'Complete';
    badge.className    = 'text-xs px-2 py-1 rounded-full badge-ready';
    document.getElementById('progress-current-file').textContent = message || 'Done';
    document.getElementById('progress-pct').textContent = '100%';
  } else {
    badge.textContent  = 'Failed';
    badge.className    = 'text-xs px-2 py-1 rounded-full badge-failed';
    document.getElementById('progress-current-file').textContent = message || 'Error';
    document.getElementById('progress-bar').classList.replace('bg-indigo-500', 'bg-red-400');
  }
  // Hide job control buttons once job finishes
  document.getElementById('progress-controls').classList.add('hidden');
  document.getElementById('progress-dismiss-btn').classList.remove('hidden');
}

function dismissProgress() {
  _activeScenarioId = null;
  closeModal('modal-progress');
  document.getElementById('progress-bar').classList.add('bg-indigo-500');
  document.getElementById('progress-bar').classList.remove('bg-red-400', 'bg-gray-400');
  loadScenarios();
}

// ─── Scenarios ────────────────────────────────────────────────────────────────

async function loadScenarios() {
  try {
    const res  = await fetch(`${API}/scenarios`);
    const data = await res.json();
    renderScenarios(data.scenarios || []);
  } catch (e) {
    document.getElementById('scenarios-list').innerHTML = '<p class="text-red-500 text-sm">Failed to load scenarios.</p>';
  }
}

function renderScenarios(scenarios) {
  const el = document.getElementById('scenarios-list');
  if (!scenarios.length) {
    el.innerHTML = '<p class="text-gray-500 text-sm p-1">No scenarios yet.</p>';
    return;
  }

  el.innerHTML = `
    <table class="w-full text-sm">
      <thead>
        <tr class="text-gray-500 text-xs uppercase border-b border-gray-100">
          <th class="text-left py-2 px-3">Name</th>
          <th class="text-left py-2 px-3">Status</th>
          <th class="text-left py-2 px-3">Type</th>
          <th class="text-left py-2 px-3">Created</th>
          <th class="text-left py-2 px-3">Actions</th>
        </tr>
      </thead>
      <tbody>
        ${scenarios.map(s => `
          <tr class="border-b border-gray-50 hover:bg-gray-50">
            <td class="py-2 px-3 font-medium text-gray-800">${esc(s.name)}</td>
            <td class="py-2 px-3"><span class="px-2 py-0.5 rounded-full text-xs ${statusBadge(s.status)}">${s.status}</span></td>
            <td class="py-2 px-3 text-gray-500">${s.type || 'local'}</td>
            <td class="py-2 px-3 text-gray-500">${fmtDate(s.created_at)}</td>
            <td class="py-2 px-3">
              <div class="flex gap-1 flex-wrap">
                <button onclick="loadScenarioStats('${s.id}')" class="text-xs px-2 py-1 rounded bg-gray-100 hover:bg-gray-200">Stats</button>
                <button onclick="showEncryptDialog('${s.id}')"  class="text-xs px-2 py-1 rounded bg-purple-100 text-purple-700 hover:bg-purple-200">Encrypt</button>
                <button onclick="exportManifest('${s.id}')"     class="text-xs px-2 py-1 rounded bg-blue-100 text-blue-700 hover:bg-blue-200">Export</button>
                <button onclick="destroyScenario('${s.id}', '${esc(s.name)}')" class="text-xs px-2 py-1 rounded bg-red-100 text-red-700 hover:bg-red-200">Destroy</button>
              </div>
            </td>
          </tr>
        `).join('')}
      </tbody>
    </table>
  `;
}

async function createScenario(event) {
  event.preventDefault();
  const name     = document.getElementById('scenario-name').value.trim();
  const template = document.getElementById('scenario-template').value;
  if (!name || !template) return;

  try {
    const res  = await fetch(`${API}/scenarios`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name, template }),
    });
    const data = await res.json();
    if (!res.ok) { toast(data.error || 'Failed to create scenario', 'error'); return; }
    toast(`Scenario "${data.name}" created`, 'success');
    document.getElementById('create-scenario-form').reset();
    loadScenarios();
  } catch (e) {
    toast('Network error', 'error');
  }
}

async function loadScenarioStats(id) {
  try {
    const res   = await fetch(`${API}/scenarios/${id}/stats`);
    const stats = await res.json();
    document.getElementById('stats-content').innerHTML = `
      <div class="space-y-2 text-sm">
        <div class="flex justify-between"><span class="text-gray-500">Total files</span><span class="font-semibold">${(stats.total_files||0).toLocaleString()}</span></div>
        <div class="flex justify-between"><span class="text-gray-500">Encrypted</span><span class="font-semibold text-purple-600">${(stats.encrypted_files||0).toLocaleString()} (${(stats.encrypted_percent||0).toFixed(1)}%)</span></div>
        <div class="flex justify-between"><span class="text-gray-500">Pending</span><span class="font-semibold text-blue-600">${(stats.pending_files||0).toLocaleString()}</span></div>
        <div class="flex justify-between"><span class="text-gray-500">Total size</span><span class="font-semibold">${fmtBytes(stats.total_size||0)}</span></div>
        <div class="mt-3">
          <div class="w-full bg-gray-200 rounded-full h-2">
            <div class="bg-purple-500 h-2 rounded-full" style="width:${stats.encrypted_percent||0}%"></div>
          </div>
        </div>
        ${renderDataTypeBadges(stats.by_data_type)}
      </div>
    `;
  } catch (e) {
    document.getElementById('stats-content').innerHTML = '<p class="text-red-500 text-xs">Failed to load stats.</p>';
  }
}

function renderDataTypeBadges(byType) {
  if (!byType || !Object.keys(byType).length) return '';
  const colors = { pii: 'bg-blue-100 text-blue-700', healthcare: 'bg-green-100 text-green-700', financial: 'bg-yellow-100 text-yellow-800', filler: 'bg-gray-100 text-gray-600' };
  return `<div class="flex flex-wrap gap-1 mt-2">` +
    Object.entries(byType).map(([t, n]) =>
      `<span class="text-xs px-2 py-0.5 rounded-full ${colors[t]||'bg-gray-100 text-gray-600'}">${t}: ${n.toLocaleString()}</span>`
    ).join('') + '</div>';
}

function showEncryptDialog(id) {
  activeEncrypt = id;
  document.getElementById('enc-password').value   = '';
  document.getElementById('enc-percentage').value = 25;
  document.getElementById('enc-pct-label').textContent = '25%';
  document.getElementById('enc-mode').value = 'partial';
  openModal('modal-encrypt');
}

async function submitEncrypt() {
  const password   = document.getElementById('enc-password').value;
  const percentage = parseFloat(document.getElementById('enc-percentage').value);
  const mode       = document.getElementById('enc-mode').value;

  if (!password) { toast('Password is required', 'error'); return; }

  try {
    const res = await fetch(`${API}/scenarios/${activeEncrypt}/encrypt`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ password, percentage, mode }),
    });
    const data = await res.json();
    if (!res.ok) { toast(data.error || 'Failed to start encryption', 'error'); return; }
    closeModal('modal-encrypt');
    toast(`Encryption started (${percentage}%)`, 'success');
    showTab('activity');
  } catch (e) {
    toast('Network error', 'error');
  }
}

async function exportManifest(id) {
  window.location.href = `${API}/scenarios/${id}/export`;
}

async function destroyScenario(id, name) {
  if (!confirm(`Destroy all payload files for "${name}"?\n\nThis permanently deletes all generated files. This cannot be undone.`)) return;
  try {
    const res  = await fetch(`${API}/scenarios/${id}/destroy`, { method: 'POST' });
    const data = await res.json();
    if (!res.ok) { toast(data.error || 'Failed to destroy', 'error'); return; }
    toast(`Destroy started for "${name}"`, 'warning');
    showTab('activity');
  } catch (e) {
    toast('Network error', 'error');
  }
}

// ─── Jobs ─────────────────────────────────────────────────────────────────────

async function loadJobs() {
  try {
    const res  = await fetch(`${API}/jobs`);
    const data = await res.json();
    renderJobs(data.jobs || []);
  } catch (e) {
    document.getElementById('jobs-list').innerHTML = '<p class="text-red-500 text-sm">Failed to load jobs.</p>';
  }
}

function renderJobs(jobs) {
  const el = document.getElementById('jobs-list');
  if (!jobs.length) {
    el.innerHTML = '<p class="text-gray-500 text-sm p-1">No jobs yet.</p>';
    return;
  }

  el.innerHTML = `
    <table class="w-full text-sm">
      <thead>
        <tr class="text-gray-500 text-xs uppercase border-b border-gray-100">
          <th class="text-left py-2 px-3">ID</th>
          <th class="text-left py-2 px-3">Scenario</th>
          <th class="text-left py-2 px-3">Target %</th>
          <th class="text-left py-2 px-3">Encrypted</th>
          <th class="text-left py-2 px-3">Status</th>
          <th class="text-left py-2 px-3">Scheduled</th>
        </tr>
      </thead>
      <tbody>
        ${jobs.map(j => `
          <tr class="border-b border-gray-50 hover:bg-gray-50">
            <td class="py-2 px-3 text-gray-600">${j.id}</td>
            <td class="py-2 px-3 font-mono text-xs text-gray-500">${(j.scenario_id||'').substring(0,8)}</td>
            <td class="py-2 px-3">${(j.target_percentage||0).toFixed(1)}%</td>
            <td class="py-2 px-3">${j.files_encrypted||0}</td>
            <td class="py-2 px-3"><span class="px-2 py-0.5 rounded-full text-xs ${statusBadge(j.status)}">${j.status}</span></td>
            <td class="py-2 px-3 text-gray-500">${fmtDate(j.scheduled_at)}</td>
          </tr>
        `).join('')}
      </tbody>
    </table>
  `;
}

// ─── WebSocket ────────────────────────────────────────────────────────────────

function connectWebSocket() {
  ws = new WebSocket(WS);

  ws.onopen = () => {
    setWsStatus(true);
    addLog('system', 'Connected to activity stream');
  };

  ws.onclose = () => {
    setWsStatus(false);
    addLog('warning', 'Disconnected. Reconnecting in 5s...');
    setTimeout(connectWebSocket, 5000);
  };

  ws.onerror = () => {};

  ws.onmessage = (e) => {
    try { handleEvent(JSON.parse(e.data)); }
    catch (_) {}
  };
}

function setWsStatus(connected) {
  const dot  = document.getElementById('ws-dot');
  const text = document.getElementById('ws-status-text');
  dot.className  = `w-2 h-2 rounded-full ${connected ? 'bg-green-400' : 'bg-red-400'}`;
  text.textContent = connected ? 'Live' : 'Disconnected';
}

function handleEvent(ev) {
  const sid = (ev.scenario_id || '').substring(0, 8);

  switch (ev.type) {
    // Enhanced generation
    case 'enhanced_generation_started':
      addLog('info', `[${sid}] Generation started — ${ev.name || ''}`);
      openProgressModal(ev.name || 'Generating...', ev.scenario_id);
      break;

    case 'enhanced_generation_progress':
      updateProgressModal(ev);
      // Only log every 5% to avoid flooding
      if (Math.round(ev.percent || 0) % 5 === 0) {
        addLog('info', `[${sid}] ${(ev.percent||0).toFixed(1)}%  ${(ev.current||0).toLocaleString()}/${(ev.total||0).toLocaleString()} files`);
      }
      break;

    case 'enhanced_generation_completed':
      completeProgressModal(true, `${(ev.files_created||0).toLocaleString()} files  ·  ${fmtBytes(ev.bytes_written||0)}`);
      addLog('success', `[${sid}] Generation complete — ${(ev.files_created||0).toLocaleString()} files, ${fmtBytes(ev.bytes_written||0)}, ${((ev.duration_ms||0)/1000).toFixed(1)}s`);
      loadScenarios();
      break;

    case 'enhanced_generation_failed':
      completeProgressModal(false, ev.error || 'Unknown error', ev.cancelled);
      addLog(ev.cancelled ? 'warn' : 'error', `[${sid}] Generation ${ev.cancelled ? 'cancelled' : 'failed'} — ${ev.error || ''}`);
      break;

    // Job control
    case 'generation_paused':
      document.getElementById('progress-pause-btn').classList.add('hidden');
      document.getElementById('progress-resume-btn').classList.remove('hidden');
      document.getElementById('progress-status-badge').textContent = 'Paused';
      document.getElementById('progress-status-badge').className   = 'text-xs px-2 py-1 rounded-full badge-pending';
      addLog('warn', `[${sid}] Generation paused`);
      break;

    case 'generation_resumed':
      document.getElementById('progress-pause-btn').classList.remove('hidden');
      document.getElementById('progress-resume-btn').classList.add('hidden');
      document.getElementById('progress-status-badge').textContent = 'Running';
      document.getElementById('progress-status-badge').className   = 'text-xs px-2 py-1 rounded-full badge-generating';
      addLog('info', `[${sid}] Generation resumed`);
      break;

    case 'generation_cancelled':
      addLog('warn', `[${sid}] Generation cancelled`);
      break;

    // Legacy generation
    case 'generation_started':
      addLog('info', `[${sid}] Legacy generation started`);
      break;
    case 'generation_progress':
      addLog('info', `[${sid}] ${ev.current}/${ev.total} — ${ev.message||''}`);
      break;
    case 'generation_completed':
      addLog('success', `[${sid}] Legacy generation complete`);
      loadScenarios();
      break;
    case 'generation_failed':
      addLog('error', `[${sid}] Legacy generation failed — ${ev.error||''}`);
      break;

    // Encryption
    case 'encryption_started':
      addLog('info', `[${sid}] Encryption started — ${ev.percentage}%`);
      break;
    case 'file_encrypted':
      // Don't log every file — too noisy
      break;
    case 'encryption_completed':
      addLog('success', `[${sid}] Encryption complete`);
      loadScenarios();
      break;
    case 'encryption_failed':
      addLog('error', `[${sid}] Encryption failed — ${ev.error||''}`);
      break;

    // Destroy
    case 'destroy_started':
      addLog('warning', `[${sid}] Destroy started — ${ev.total||0} files`);
      break;
    case 'destroy_completed':
      addLog('warning', `[${sid}] Destroy complete — deleted ${ev.deleted||0}, failed ${ev.failed||0}`);
      loadScenarios();
      break;

    // Scenario lifecycle
    case 'scenario_created':
      addLog('info', `Scenario created — ${ev.name||''}`);
      break;
    case 'scenario_deleted':
      addLog('info', `Scenario deleted — ${sid}`);
      break;

    default:
      addLog('system', JSON.stringify(ev));
  }
}

// ─── Activity log ─────────────────────────────────────────────────────────────

function addLog(level, message) {
  const log  = document.getElementById('activity-log');
  const time = new Date().toLocaleTimeString();

  const colors = {
    info:    'text-blue-400',
    success: 'text-green-400',
    warning: 'text-yellow-400',
    error:   'text-red-400',
    system:  'text-gray-500',
  };

  const line = document.createElement('div');
  line.className = `${colors[level] || 'text-gray-400'}`;
  line.textContent = `${time}  ${message}`;
  log.appendChild(line);

  // Cap log at 500 lines
  while (log.children.length > 500) log.removeChild(log.firstChild);

  if (autoScroll) log.scrollTop = log.scrollHeight;
}

function toggleAutoScroll() {
  autoScroll = !autoScroll;
  document.getElementById('autoscroll-btn').textContent = `Auto-scroll: ${autoScroll ? 'ON' : 'OFF'}`;
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

function esc(str) {
  return String(str || '').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');
}

function fmtDate(iso) {
  if (!iso) return '—';
  return new Date(iso).toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' });
}

function fmtBytes(bytes) {
  if (bytes < 1024)          return `${bytes} B`;
  if (bytes < 1024**2)       return `${(bytes/1024).toFixed(1)} KB`;
  if (bytes < 1024**3)       return `${(bytes/1024**2).toFixed(1)} MB`;
  return `${(bytes/1024**3).toFixed(2)} GB`;
}

function statusBadge(status) {
  const map = {
    pending:    'badge-pending',
    generating: 'badge-generating',
    ready:      'badge-ready',
    failed:     'badge-failed',
    destroyed:  'badge-destroyed',
    encrypted:  'badge-encrypted',
    running:    'badge-generating',
    completed:  'badge-ready',
    cancelled:  'badge-destroyed',
  };
  return map[status] || 'badge-pending';
}
