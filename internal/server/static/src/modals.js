'use strict';

function showModal(id) {
    const el = document.getElementById(id);
    if (el) el.classList.add('active');
}

function hideModal(id) {
    const el = document.getElementById(id);
    if (el) el.classList.remove('active');
}

function bindModals() {
    document.querySelectorAll('[data-close-modal]').forEach(btn => {
        btn.addEventListener('click', () => hideModal(btn.dataset.closeModal));
    });

    document.querySelectorAll('.modal').forEach(modal => {
        modal.addEventListener('click', ev => {
            if (ev.target === modal) modal.classList.remove('active');
        });
    });

    document.addEventListener('keydown', ev => {
        if (ev.key === 'Escape') {
            document.querySelectorAll('.modal.active').forEach(m => m.classList.remove('active'));
        }
    });
}