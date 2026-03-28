// Janus Web UI JavaScript

const API_BASE = window.location.origin + '/api/v1';
const WS_URL = `ws://${window.location.host}/ws/v1/activity`;

let ws = null;
let autoScroll = true;
let selectedScenario = null;

// Initialize on page load
window.addEventListener('DOMContentLoaded', () => {
    loadScenarios();
    connectWebSocket();
    
    // Refresh data periodically
    setInterval(refreshCurrentTab, 5000);
});

// Tab management
function showTab(tabName) {
    // Hide all tabs
    document.querySelectorAll('.tab-content').forEach(el => {
        el.classList.add('hidden');
    });
    
    // Show selected tab
    document.getElementById(`content-${tabName}`).classList.remove('hidden');
    
    // Update button styles
    document.querySelectorAll('[id^="tab-"]').forEach(btn => {
        btn.classList.remove('bg-blue-500', 'text-white');
        btn.classList.add('bg-gray-200', 'text-gray-700');
    });
    document.getElementById(`tab-${tabName}`).classList.remove('bg-gray-200', 'text-gray-700');
    document.getElementById(`tab-${tabName}`).classList.add('bg-blue-500', 'text-white');
    
    // Load data for tab
    if (tabName === 'scenarios') loadScenarios();
    if (tabName === 'jobs') loadJobs();
}

function refreshCurrentTab() {
    const visibleTab = Array.from(document.querySelectorAll('.tab-content')).find(el => !el.classList.contains('hidden'));
    if (visibleTab) {
        const tabId = visibleTab.id.replace('content-', '');
        if (tabId === 'scenarios' && selectedScenario) loadScenarioStats(selectedScenario);
        if (tabId === 'jobs') loadJobs();
    }
}

// Create scenario
async function createScenario(event) {
    event.preventDefault();
    
    const name = document.getElementById('scenario-name').value;
    const template = document.getElementById('scenario-template').value;
    
    try {
        const response = await fetch(`${API_BASE}/scenarios`, {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({name, template})
        });
        
        if (!response.ok) throw new Error('Failed to create scenario');
        
        const data = await response.json();
        alert(`Scenario created: ${data.name} (ID: ${data.id})`);
        
        document.getElementById('create-scenario-form').reset();
        loadScenarios();
    } catch (error) {
        alert('Error: ' + error.message);
    }
}

// Load scenarios
async function loadScenarios() {
    try {
        const response = await fetch(`${API_BASE}/scenarios`);
        const data = await response.json();
        
        const list = document.getElementById('scenarios-list');
        
        if (data.scenarios.length === 0) {
            list.innerHTML = '<p class="text-gray-600">No scenarios yet. Create one above!</p>';
            return;
        }
        
        list.innerHTML = `
            <table class="w-full">
                <thead>
                    <tr class="bg-gray-100">
                        <th class="text-left p-2">Name</th>
                        <th class="text-left p-2">Status</th>
                        <th class="text-left p-2">Created</th>
                        <th class="text-left p-2">Actions</th>
                    </tr>
                </thead>
                <tbody>
                    ${data.scenarios.map(s => `
                        <tr class="border-t">
                            <td class="p-2">${s.name}</td>
                            <td class="p-2"><span class="px-2 py-1 bg-blue-100 text-blue-800 rounded text-sm">${s.status}</span></td>
                            <td class="p-2">${new Date(s.created_at).toLocaleDateString()}</td>
                            <td class="p-2">
                                <button onclick="loadScenarioStats('${s.id}')" class="px-2 py-1 bg-green-500 text-white rounded text-sm mr-1">Stats</button>
                                <button onclick="generateFiles('${s.id}')" class="px-2 py-1 bg-blue-500 text-white rounded text-sm mr-1">Generate</button>
                                <button onclick="showEncryptDialog('${s.id}')" class="px-2 py-1 bg-purple-500 text-white rounded text-sm mr-1">Encrypt</button>
                                <button onclick="deleteScenario('${s.id}')" class="px-2 py-1 bg-red-500 text-white rounded text-sm">Delete</button>
                            </td>
                        </tr>
                    `).join('')}
                </tbody>
            </table>
        `;
    } catch (error) {
        console.error('Error loading scenarios:', error);
    }
}

// Load scenario statistics
async function loadScenarioStats(scenarioId) {
    selectedScenario = scenarioId;
    
    try {
        const response = await fetch(`${API_BASE}/scenarios/${scenarioId}/stats`);
        const stats = await response.json();
        
        document.getElementById('stats-content').innerHTML = `
            <div class="space-y-2">
                <div class="flex justify-between">
                    <span class="text-gray-700">Total Files:</span>
                    <span class="font-bold">${stats.total_files}</span>
                </div>
                <div class="flex justify-between">
                    <span class="text-gray-700">Encrypted:</span>
                    <span class="font-bold text-green-600">${stats.encrypted_files} (${stats.encrypted_percent.toFixed(1)}%)</span>
                </div>
                <div class="flex justify-between">
                    <span class="text-gray-700">Pending:</span>
                    <span class="font-bold text-blue-600">${stats.pending_files}</span>
                </div>
                <div class="flex justify-between">
                    <span class="text-gray-700">Total Size:</span>
                    <span class="font-bold">${(stats.total_size / 1024 / 1024).toFixed(2)} MB</span>
                </div>
                <div class="mt-4">
                    <div class="w-full bg-gray-200 rounded-full h-4">
                        <div class="bg-green-500 h-4 rounded-full" style="width: ${stats.encrypted_percent}%"></div>
                    </div>
                    <p class="text-sm text-gray-600 mt-1 text-center">${stats.encrypted_percent.toFixed(1)}% Encrypted</p>
                </div>
            </div>
        `;
    } catch (error) {
        console.error('Error loading stats:', error);
    }
}

// Generate files
async function generateFiles(scenarioId) {
    if (!confirm('Start file generation? This may take several minutes.')) return;
    
    try {
        const response = await fetch(`${API_BASE}/scenarios/${scenarioId}/generate`, {
            method: 'POST'
        });
        
        if (!response.ok) throw new Error('Failed to start generation');
        
        alert('File generation started. Check the Activity tab for progress.');
        showTab('activity');
    } catch (error) {
        alert('Error: ' + error.message);
    }
}

// Show encrypt dialog
function showEncryptDialog(scenarioId) {
    const password = prompt('Enter encryption password:');
    if (!password) return;
    
    const percentage = prompt('Enter percentage to encrypt (0-100):', '25');
    if (!percentage) return;
    
    encryptFiles(scenarioId, password, parseFloat(percentage));
}

// Encrypt files
async function encryptFiles(scenarioId, password, percentage) {
    try {
        const response = await fetch(`${API_BASE}/scenarios/${scenarioId}/encrypt`, {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({
                password,
                percentage,
                mode: 'partial'
            })
        });
        
        if (!response.ok) throw new Error('Failed to start encryption');
        
        alert('Encryption started. Check the Activity tab for progress.');
        showTab('activity');
    } catch (error) {
        alert('Error: ' + error.message);
    }
}

// Delete scenario
async function deleteScenario(scenarioId) {
    if (!confirm('Delete this scenario? This cannot be undone.')) return;
    
    try {
        const response = await fetch(`${API_BASE}/scenarios/${scenarioId}`, {
            method: 'DELETE'
        });
        
        if (!response.ok) throw new Error('Failed to delete scenario');
        
        alert('Scenario deleted');
        loadScenarios();
    } catch (error) {
        alert('Error: ' + error.message);
    }
}

// Load jobs
async function loadJobs() {
    try {
        const response = await fetch(`${API_BASE}/jobs`);
        const data = await response.json();
        
        const list = document.getElementById('jobs-list');
        
        if (data.jobs.length === 0) {
            list.innerHTML = '<p class="text-gray-600">No jobs scheduled</p>';
            return;
        }
        
        list.innerHTML = `
            <table class="w-full">
                <thead>
                    <tr class="bg-gray-100">
                        <th class="text-left p-2">ID</th>
                        <th class="text-left p-2">Scenario</th>
                        <th class="text-left p-2">Target %</th>
                        <th class="text-left p-2">Status</th>
                        <th class="text-left p-2">Scheduled</th>
                    </tr>
                </thead>
                <tbody>
                    ${data.jobs.map(j => `
                        <tr class="border-t">
                            <td class="p-2">${j.id}</td>
                            <td class="p-2">${j.scenario_id.substring(0, 8)}</td>
                            <td class="p-2">${j.target_percentage.toFixed(1)}%</td>
                            <td class="p-2"><span class="px-2 py-1 bg-blue-100 text-blue-800 rounded text-sm">${j.status}</span></td>
                            <td class="p-2">${new Date(j.scheduled_at).toLocaleString()}</td>
                        </tr>
                    `).join('')}
                </tbody>
            </table>
        `;
    } catch (error) {
        console.error('Error loading jobs:', error);
    }
}

// WebSocket connection
function connectWebSocket() {
    ws = new WebSocket(WS_URL);
    
    ws.onopen = () => {
        document.getElementById('ws-status').textContent = 'Connected';
        document.getElementById('ws-status').classList.add('text-green-600');
        addActivityLog('info', 'Connected to activity stream');
    };
    
    ws.onclose = () => {
        document.getElementById('ws-status').textContent = 'Disconnected';
        document.getElementById('ws-status').classList.remove('text-green-600');
        addActivityLog('warning', 'Disconnected from activity stream. Reconnecting...');
        
        // Reconnect after 5 seconds
        setTimeout(connectWebSocket, 5000);
    };
    
    ws.onerror = (error) => {
        console.error('WebSocket error:', error);
    };
    
    ws.onmessage = (event) => {
        try {
            const data = JSON.parse(event.data);
            handleActivityEvent(data);
        } catch (error) {
            console.error('Error parsing WebSocket message:', error);
        }
    };
}

// Handle activity events
function handleActivityEvent(event) {
    let message = '';
    let level = 'info';
    
    switch (event.type) {
        case 'scenario_created':
            message = `Scenario created: ${event.name}`;
            break;
        case 'generation_started':
            message = `File generation started for scenario ${event.scenario_id.substring(0, 8)}`;
            break;
        case 'generation_progress':
            message = `Generation: ${event.current}/${event.total} - ${event.message}`;
            break;
        case 'generation_completed':
            message = `File generation completed for scenario ${event.scenario_id.substring(0, 8)}`;
            level = 'success';
            break;
        case 'encryption_started':
            message = `Encryption started: ${event.percentage}%`;
            break;
        case 'file_encrypted':
            message = `Encrypted: ${event.file_path}`;
            break;
        case 'encryption_completed':
            message = `Encryption completed for scenario ${event.scenario_id.substring(0, 8)}`;
            level = 'success';
            break;
        default:
            message = JSON.stringify(event);
    }
    
    addActivityLog(level, message);
}

// Add to activity log
function addActivityLog(level, message) {
    const log = document.getElementById('activity-log');
    const timestamp = new Date().toLocaleTimeString();
    
    const colors = {
        info: 'text-blue-600',
        success: 'text-green-600',
        warning: 'text-yellow-600',
        error: 'text-red-600'
    };
    
    const entry = document.createElement('div');
    entry.className = `${colors[level] || 'text-gray-600'} mb-1`;
    entry.textContent = `[${timestamp}] ${message}`;
    
    log.appendChild(entry);
    
    // Auto-scroll
    if (autoScroll) {
        log.scrollTop = log.scrollHeight;
    }
}

// Toggle auto-scroll
function toggleAutoScroll() {
    autoScroll = !autoScroll;
    document.getElementById('autoscroll-btn').textContent = `Auto-scroll: ${autoScroll ? 'ON' : 'OFF'}`;
}
