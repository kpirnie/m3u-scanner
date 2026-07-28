'use strict';

let currentEntry = null;
let currentID = null;

function field(label, name, value, type) {
    return '<div>' +
        '<label class="form-label">' + escapeHtml(label) + '</label>' +
        '<input class="form-input" data-field="' + name + '" type="' + (type || 'text') +
        '" value="' + escapeHtml(value) + '">' +
        '</div>';
}

function textarea(label, name, value) {
    return '<div class="md:col-span-2">' +
        '<label class="form-label">' + escapeHtml(label) + '</label>' +
        '<textarea class="form-textarea" data-field="' + name + '">' + escapeHtml(value) + '</textarea>' +
        '</div>';
}

function listField(label, name, values) {
    return '<div class="md:col-span-2">' +
        '<label class="form-label">' + escapeHtml(label) +
        ' <span class="text-gray-500 font-normal">(comma separated)</span></label>' +
        '<input class="form-input" data-list="' + name + '" type="text" value="' +
        escapeHtml(joinList(values)) + '">' +
        '</div>';
}

function castRow(person) {
    return '<div class="flex gap-2 mb-2" data-cast-row>' +
        '<input class="form-input flex-1" data-cast-name placeholder="Name" value="' +
        escapeHtml(person ? person.name : '') + '">' +
        '<input class="form-input flex-1" data-cast-role placeholder="Role" value="' +
        escapeHtml(person && person.role ? person.role : '') + '">' +
        '<button class="btn-danger px-3" data-cast-remove>' +
        '<svg class="w-4 h-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">' +
        '<path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12"/></svg>' +
        '</button></div>';
}

function buildForm(e) {
    let html = '<div class="grid grid-cols-1 md:grid-cols-2 gap-4">';

    html += field('Title', 'title', e.title || '');
    html += field('Sort Title', 'sort_title', e.sort_title || '');
    html += textarea('Plot', 'plot', e.plot || '');
    html += field('Tagline', 'tagline', e.tagline || '');
    html += field('Year', 'year', e.year || '');
    html += field('Premiered', 'premiered', e.premiered || '');
    html += field('Rating', 'rating', e.rating || '', 'number');
    html += field('Critic Rating', 'critic_rating', e.critic_rating || '', 'number');
    html += field('Content Rating', 'mpaa', e.mpaa || '');
    html += field('Country', 'country', e.country || '');

    if (e.media_type === 'music') {
        html += field('Artist', 'artist', e.artist || '');
        html += field('Album', 'album', e.album || '');
        html += field('Disc', 'disc', e.disc || '', 'number');
        html += field('Track', 'track', e.track || '', 'number');
    }

    if (e.media_type === 'shows') {
        html += field('Series', 'series', e.series || '');
        html += field('Episode Title', 'episode_title', e.episode_title || '');
        html += field('Season', 'season', e.season || '', 'number');
        html += field('Episode', 'episode', e.episode || '', 'number');
    }

    if (e.media_type === 'movies') {
        html += field('Collection', 'collection', e.collection || '');
    }

    html += listField('Genres', 'genres', e.genres);
    html += listField('Studios', 'studios', e.studios);
    html += listField('Tags', 'tags', e.tags);
    html += listField('Directors', 'directors', e.directors);
    html += listField('Writers', 'writers', e.writers);

    html += field('IMDb ID', 'imdb_id', e.imdb_id || '');
    html += field('TMDb ID', 'tmdb_id', e.tmdb_id || '');
    html += field('TVDb ID', 'tvdb_id', e.tvdb_id || '');

    html += '<div class="md:col-span-2">' +
        '<label class="form-label">Cast</label>' +
        '<div id="cast-rows">' + (e.cast || []).map(castRow).join('') + '</div>' +
        '<button id="btn-add-cast" class="btn-secondary text-sm mt-1">Add Cast Member</button>' +
        '</div>';

    html += field('Poster URL or path', 'poster', e.poster || '');
    html += field('Fanart URL or path', 'fanart', e.fanart || '');

    html += '<div class="md:col-span-2 pt-2 border-t border-kptv-border">' +
        '<div class="metric-row"><span class="text-gray-400 text-sm">Path</span>' +
        '<span class="text-xs text-gray-500 text-truncate max-w-lg">' + escapeHtml(e.path) + '</span></div>' +
        '</div>';

    html += '</div>';
    return html;
}

function bindCastControls() {
    const rows = document.getElementById('cast-rows');

    document.getElementById('btn-add-cast').addEventListener('click', () => {
        rows.insertAdjacentHTML('beforeend', castRow(null));
        bindCastRemove();
    });

    bindCastRemove();
}

function bindCastRemove() {
    document.querySelectorAll('[data-cast-remove]').forEach(btn => {
        btn.onclick = () => btn.closest('[data-cast-row]').remove();
    });
}

async function openEditor(id) {
    try {
        const data = await api('/api/entry/' + id);
        currentEntry = data.entry;
        currentID = data.id;

        document.getElementById('edit-title').textContent = currentEntry.display;
        document.getElementById('edit-nfopath').textContent = data.nfo_path || 'no sidecar for this media type';
        document.getElementById('edit-body').innerHTML = buildForm(currentEntry);
        
        const tagsBtn = document.getElementById('btn-write-tags');
        tagsBtn.classList.toggle('hidden', currentEntry.media_type !== 'music');

        bindCastControls();
        showModal('edit-modal');
    } catch (err) {
        notify(err.message, 'error');
    }
}

function collectForm() {
    const payload = {};

    document.querySelectorAll('#edit-body [data-field]').forEach(input => {
        const key = input.dataset.field;
        if (input.type === 'number') {
            const n = parseFloat(input.value);
            payload[key] = isNaN(n) ? 0 : n;
        } else {
            payload[key] = input.value;
        }
    });

    document.querySelectorAll('#edit-body [data-list]').forEach(input => {
        payload[input.dataset.list] = splitList(input.value);
    });

    payload.cast = [];
    document.querySelectorAll('#edit-body [data-cast-row]').forEach(row => {
        const name = row.querySelector('[data-cast-name]').value.trim();
        if (!name) return;
        payload.cast.push({ name: name, role: row.querySelector('[data-cast-role]').value.trim() });
    });

    return payload;
}

async function saveEntry() {
    const btn = document.getElementById('btn-save');
    btn.disabled = true;

    try {
        const res = await api('/api/entry/' + currentID, {
            method: 'PUT',
            body: JSON.stringify(collectForm())
        });
        notify('Saved to ' + res.nfo_path, 'success');
        hideModal('edit-modal');
        loadEntries();
    } catch (err) {
        notify(err.message, 'error');
    } finally {
        btn.disabled = false;
    }
}

async function rescanEntry() {
    const btn = document.getElementById('btn-entry-rescan');
    btn.disabled = true;

    try {
        await api('/api/entry/' + currentID + '/rescan', { method: 'POST' });
        notify('Sidecar re-read', 'success');
        await openEditor(currentID);
    } catch (err) {
        notify(err.message, 'error');
    } finally {
        btn.disabled = false;
    }
}

async function writeFileTags() {
    if (!confirm('Write these values into the audio file itself? This modifies the file on disk.')) {
        return;
    }

    const btn = document.getElementById('btn-write-tags');
    btn.disabled = true;

    try {
        const res = await api('/api/entry/' + currentID + '/tags', { method: 'POST' });
        notify(res.cover_note ? 'Tags written — cover skipped: ' + res.cover_note : 'Tags written to file', 'success');
        await openEditor(currentID);
    } catch (err) {
        notify(err.message, 'error');
    } finally {
        btn.disabled = false;
    }
}