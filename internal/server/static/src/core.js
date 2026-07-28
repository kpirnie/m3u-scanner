'use strict';

async function loadStats() {
    try {
        const h = await api('/api/health');
        document.getElementById('stat-entries').textContent = h.entry_count;
        document.getElementById('stat-lastscan').textContent = formatTime(h.last_scan);
        document.getElementById('stat-ffprobe').textContent = h.ffprobe_available ? 'Yes' : 'No';
    } catch (err) {
        notify('Could not load status: ' + err.message, 'error');
    }
}

async function triggerRescan(type) {
    const url = type ? '/api/rescan?type=' + encodeURIComponent(type) : '/api/rescan';
    try {
        await api(url, { method: 'POST' });
        notify('Rescan started' + (type ? ' — ' + type : ''), 'info');
        setTimeout(() => { loadStats(); loadEntries(); }, 2500);
    } catch (err) {
        notify(err.message, 'error');
    }
}

function on(id, event, handler) {
    const el = document.getElementById(id);
    if (el) {
        el.addEventListener(event, handler);
    } else {
        console.warn('missing element: #' + id);
    }
}

function bindToolbar() {
    document.querySelectorAll('#type-tabs .tab-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            document.querySelectorAll('#type-tabs .tab-btn').forEach(b => {
                b.classList.remove('bg-kptv-blue', 'text-white');
                b.classList.add('text-gray-400');
            });
            btn.classList.add('bg-kptv-blue', 'text-white');
            btn.classList.remove('text-gray-400');

            state.type = btn.dataset.type;
            state.offset = 0;
            loadEntries();
        });
    });

    on('search', 'input', debounce(ev => {
        state.query = ev.target.value;
        state.offset = 0;
        loadEntries();
    }, 300));

    on('page-size', 'change', ev => {
        state.limit = parseInt(ev.target.value, 10);
        state.offset = 0;
        loadEntries();
    });

    on('btn-prev', 'click', () => {
        state.offset = Math.max(0, state.offset - state.limit);
        loadEntries();
    });

    on('btn-next', 'click', () => {
        state.offset += state.limit;
        loadEntries();
    });

    on('btn-rescan-all', 'click', () => triggerRescan(''));
    on('btn-rescan-type', 'click', () => triggerRescan(state.type || ''));
    on('btn-save', 'click', saveEntry);
    on('btn-entry-rescan', 'click', rescanEntry);
    on('btn-write-tags', 'click', writeFileTags);
}

document.addEventListener('DOMContentLoaded', () => {
    bindModals();
    bindToolbar();
    loadStats();
    loadEntries();
    setInterval(loadStats, 30000);
});