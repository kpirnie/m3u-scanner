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

    document.getElementById('search').addEventListener('input', debounce(ev => {
        state.query = ev.target.value;
        state.offset = 0;
        loadEntries();
    }, 300));

    document.getElementById('page-size').addEventListener('change', ev => {
        state.limit = parseInt(ev.target.value, 10);
        state.offset = 0;
        loadEntries();
    });

    document.getElementById('btn-prev').addEventListener('click', () => {
        state.offset = Math.max(0, state.offset - state.limit);
        loadEntries();
    });

    document.getElementById('btn-next').addEventListener('click', () => {
        state.offset += state.limit;
        loadEntries();
    });

    document.getElementById('btn-rescan-all').addEventListener('click', () => triggerRescan(''));
    document.getElementById('btn-rescan-type').addEventListener('click', () => {
        if (!state.type) {
            triggerRescan('');
            return;
        }
        triggerRescan(state.type);
    });

    document.getElementById('btn-save').addEventListener('click', saveEntry);
    document.getElementById('btn-entry-rescan').addEventListener('click', rescanEntry);
    document.getElementById('btn-write-tags').addEventListener('click', writeFileTags);
}

document.addEventListener('DOMContentLoaded', () => {
    bindModals();
    bindToolbar();
    loadStats();
    loadEntries();
    setInterval(loadStats, 30000);
});