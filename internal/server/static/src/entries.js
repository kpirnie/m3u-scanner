'use strict';

const state = {
    type: '',
    query: '',
    offset: 0,
    limit: 50,
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

function thumb(url, label) {
    if (!url) return '';
    return '<img src="' + escapeHtml(url) + '" alt="' + escapeHtml(label) + '" loading="lazy" ' +
        'class="w-10 h-14 object-cover rounded border border-kptv-border cursor-pointer hover:border-kptv-blue-light transition-colors" ' +
        'data-art="' + escapeHtml(url) + '">';
}

function renderEntry(item) {
    const nfoDot = item.has_nfo ? 'status-active' : 'status-warning';

    return '<div class="entry-item flex items-center gap-4">' +
        '<span class="status-indicator ' + nfoDot + '" title="Sidecar metadata"></span>' +
        '<div class="flex gap-2 shrink-0">' +
        thumb(item.poster, 'Poster') +
        thumb(item.fanart, 'Fanart') +
        '</div>' +
        '<div class="flex-1 min-w-0">' +
        '<div class="font-medium text-truncate">' + escapeHtml(item.title || item.display) + '</div>' +
        '<div class="text-sm text-gray-400 text-truncate">' + escapeHtml(subtitleFor(item)) + '</div>' +
        '</div>' +
        '<span class="stat-badge">' + escapeHtml(item.media_type) + '</span>' +
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
            list.querySelectorAll('[data-art]').forEach(img => {
                img.addEventListener('click', () => {
                    document.getElementById('art-modal-img').src = img.dataset.art;
                    showModal('art-modal');
                });
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