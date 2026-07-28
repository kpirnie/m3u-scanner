'use strict';

const state = {
    type: '',
    query: '',
    offset: 0,
    limit: 100,
    total: 0
};

function subtitleFor(item) {
    if (item.media_type === 'music') {
        return [item.artist, item.album].filter(Boolean).join(' — ');
    }
    if (item.media_type === 'shows') {
        const se = (item.season || item.episode)
            ? 'S' + String(item.season).padStart(2, '0') + 'E' + String(item.episode).padStart(2, '0')
            : '';
        return [item.series, se].filter(Boolean).join(' — ');
    }
    return item.year || '';
}

function renderEntry(item) {
    const nfoDot = item.has_nfo ? 'status-active' : 'status-warning';
    const posterIcon = item.has_poster
        ? '<svg class="w-4 h-4 text-green-500" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"/></svg>'
        : '<svg class="w-4 h-4 text-gray-600" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M3 3l18 18M4 16l4.586-4.586a2 2 0 012.828 0M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"/></svg>';

    return '<div class="entry-item flex items-center gap-4">' +
        '<span class="status-indicator ' + nfoDot + '" title="Sidecar metadata"></span>' +
        '<div class="flex-1 min-w-0">' +
        '<div class="font-medium text-truncate">' + escapeHtml(item.title || item.display) + '</div>' +
        '<div class="text-sm text-gray-400 text-truncate">' + escapeHtml(subtitleFor(item)) + '</div>' +
        '</div>' +
        '<span class="stat-badge">' + escapeHtml(item.media_type) + '</span>' +
        posterIcon +
        '<button class="btn-secondary text-sm" data-edit="' + escapeHtml(item.id) + '">Edit</button>' +
        '</div>';
}

async function loadEntries() {
    const list = document.getElementById('entry-list');
    list.innerHTML = '<div class="flex justify-center py-12"><div class="spinner"></div></div>';

    const params = new URLSearchParams();
    if (state.type) params.set('type', state.type);
    if (state.query) params.set('q', state.query);
    params.set('limit', state.limit);
    params.set('offset', state.offset);

    try {
        const data = await api('/api/entries?' + params.toString());
        state.total = data.total;

        if (!data.items.length) {
            list.innerHTML = '<div class="text-center py-12 text-gray-500">No entries match the current filter.</div>';
        } else {
            list.innerHTML = data.items.map(renderEntry).join('');
            list.querySelectorAll('[data-edit]').forEach(btn => {
                btn.addEventListener('click', () => openEditor(btn.dataset.edit));
            });
        }

        document.getElementById('stat-showing').textContent = data.items.length + ' / ' + data.total;
        renderPagination();
    } catch (err) {
        list.innerHTML = '<div class="text-center py-12 text-red-400">' + escapeHtml(err.message) + '</div>';
    }
}

function renderPagination() {
    const wrap = document.getElementById('pagination');
    if (state.total <= state.limit) {
        wrap.classList.add('hidden');
        return;
    }
    wrap.classList.remove('hidden');

    const page = Math.floor(state.offset / state.limit) + 1;
    const pages = Math.ceil(state.total / state.limit);

    document.getElementById('page-info').textContent = 'Page ' + page + ' of ' + pages;
    document.getElementById('btn-prev').disabled = state.offset === 0;
    document.getElementById('btn-next').disabled = state.offset + state.limit >= state.total;
}