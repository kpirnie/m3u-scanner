'use strict';

function escapeHtml(value) {
    if (value === null || value === undefined) return '';
    return String(value)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;');
}

function notify(message, kind) {
    const palette = {
        success: 'bg-green-700 border-green-500',
        error: 'bg-red-700 border-red-500',
        info: 'bg-kptv-blue border-kptv-blue-light'
    };
    const el = document.createElement('div');
    el.className = 'notification ' + (palette[kind] || palette.info) +
        ' border-l-4 text-white px-4 py-3 rounded shadow-lg mb-2';
    el.textContent = message;
    document.getElementById('notifications').appendChild(el);
    setTimeout(() => el.remove(), 4000);
}

async function api(path, options) {
    const res = await fetch(path, Object.assign({
        headers: { 'Content-Type': 'application/json' }
    }, options || {}));

    let payload = null;
    try {
        payload = await res.json();
    } catch (e) {
        payload = null;
    }

    if (!res.ok) {
        throw new Error((payload && payload.error) || ('HTTP ' + res.status));
    }
    return payload;
}

function debounce(fn, wait) {
    let timer = null;
    return function () {
        const args = arguments;
        clearTimeout(timer);
        timer = setTimeout(() => fn.apply(null, args), wait);
    };
}

function splitList(value) {
    return String(value || '')
        .split(',')
        .map(v => v.trim())
        .filter(v => v.length > 0);
}

function joinList(values) {
    return Array.isArray(values) ? values.join(', ') : '';
}

function formatTime(iso) {
    if (!iso) return '—';
    const d = new Date(iso);
    if (isNaN(d.getTime()) || d.getFullYear() < 2000) return '—';
    return d.toLocaleString();
}