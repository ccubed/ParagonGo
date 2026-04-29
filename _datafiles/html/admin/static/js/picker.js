// Picker - reusable search-and-select modal for admin pages.
//
// Supersedes SelectDialog for new work. SelectDialog remains for backward
// compatibility with existing color-pattern selection.
//
// Single-select usage:
//   Picker.open({
//     title:   'Select Buff',
//     source:  '/admin/api/v1/buffs',   // API path OR async function
//     columns: [
//       { key: 'BuffId', label: '#',    width: '4rem', mono: true },
//       { key: 'Name',   label: 'Name', flex: true },
//     ],
//     idKey:   'BuffId',
//     onSelect: (buff) => { /* buff is the full item object */ },
//   });
//
// Multi-select usage:
//   Picker.open({
//     ...PickerConfigs.buffs,
//     multi:    true,
//     selected: [101, 205],   // pre-checked IDs
//     onSelect: (buffs) => { /* buffs is an array of full item objects */ },
//   });
//
// source accepts:
//   - A string API path: Picker calls AdminAPI.get(path) and expects res.data.data
//     to be an array or object (Object.values() applied to objects).
//   - An async function: async () => itemArray
//
// columns: array of column descriptors:
//   { key, label, flex?, width?, mono?, render? }
//   render(value, item) returns a string (HTML-escaped) or a DOM node.
//
// Optional options:
//   idKey      - property used as the unique ID (default: 'id')
//   searchKeys - array of property names to search (default: all string/number columns)
//   filter     - predicate applied to items before display: item => bool
//   sort       - comparator applied after loading: (a, b) => number
//   selected   - array of ID values to pre-check (multi only)

const Picker = (() => {
    'use strict';

    let overlay = null;
    let _triggerEl = null;

    function injectStyles() {
        if (document.getElementById('picker-styles')) return;
        const style = document.createElement('style');
        style.id = 'picker-styles';
        style.textContent = `
            .pk-overlay {
                position: fixed; inset: 0; background: var(--color-overlay);
                display: flex; align-items: center; justify-content: center;
                z-index: 9999;
            }
            .pk-modal {
                background: var(--color-surface); border-radius: 8px; box-shadow: 0 8px 32px var(--color-shadow);
                width: 640px; max-width: 96vw; max-height: 82vh;
                display: flex; flex-direction: column; overflow: hidden;
            }
            .pk-header {
                padding: 0.85rem 1rem 0.7rem; border-bottom: 1px solid var(--color-border);
                display: flex; align-items: center; justify-content: space-between;
                flex-shrink: 0;
            }
            .pk-title { font-size: 1rem; font-weight: 700; color: var(--color-text); }
            .pk-close {
                background: none; border: none; font-size: 1.25rem; cursor: pointer;
                color: var(--color-text-faint); line-height: 1; padding: 0 0.2rem;
            }
            .pk-close:hover { color: var(--color-text); }
            .pk-search-wrap { padding: 0.6rem 1rem; border-bottom: 1px solid var(--color-border-light); flex-shrink: 0; }
            .pk-search {
                width: 100%; padding: 0.4rem 0.65rem; border: 1px solid var(--color-border-medium);
                border-radius: 4px; font-size: 0.875rem;
                background: var(--color-surface-raised); color: var(--color-text);
            }
            .pk-search:focus { outline: 2px solid var(--color-focus); outline-offset: 1px; border-color: transparent; }
            .pk-table-wrap { overflow-y: auto; flex: 1; background: var(--color-surface); }
            .pk-table { width: 100%; border-collapse: collapse; font-size: 0.875rem; }
            .pk-table thead { position: sticky; top: 0; background: var(--color-page-bg); z-index: 1; }
            .pk-table th {
                padding: 0.4rem 0.75rem; text-align: left; font-size: 0.72rem;
                font-weight: 700; text-transform: uppercase; letter-spacing: 0.04em;
                color: var(--color-text-subtle); border-bottom: 1px solid var(--color-border-strong); white-space: nowrap;
            }
            .pk-table th.pk-col-check { width: 2rem; }
            .pk-table td { padding: 0.45rem 0.75rem; border-bottom: 1px solid var(--color-border-faint); vertical-align: middle; color: var(--color-text); }
            .pk-table tr:last-child td { border-bottom: none; }
            .pk-table tbody tr { cursor: pointer; }
            .pk-table tbody tr:hover { background: var(--color-row-hover); }
            .pk-table tbody tr.pk-highlighted { background: var(--color-row-hover); }
            .pk-table tbody tr.pk-row-selected { background: var(--color-active-bg); color: var(--color-active-text); }
            .pk-table tbody tr.pk-row-selected td { color: var(--color-active-text); }
            .pk-table tbody tr.pk-row-selected input[type="checkbox"] { accent-color: var(--color-active-text); }
            .pk-cell-mono { font-family: monospace; }
            .pk-empty { padding: 2.5rem 1rem; text-align: center; color: var(--color-text-placeholder); font-size: 0.875rem; }
            .pk-loading { padding: 2.5rem 1rem; text-align: center; color: var(--color-text-faint); font-size: 0.875rem; }
            .pk-footer {
                padding: 0.65rem 1rem; border-top: 1px solid var(--color-border-light);
                display: flex; justify-content: flex-end; gap: 0.5rem; flex-shrink: 0;
                background: var(--color-surface);
            }
            .pk-btn {
                padding: 0.4rem 1rem; border-radius: 4px; font-size: 0.85rem;
                font-weight: 600; cursor: pointer; border: 1px solid transparent;
            }
            .pk-btn-cancel { background: var(--color-btn-cancel-bg); color: var(--color-btn-cancel-text); border-color: var(--color-border-medium); }
            .pk-btn-cancel:hover { background: var(--color-btn-cancel-hover); }
            .pk-btn-confirm { background: var(--color-btn-primary-bg); color: var(--color-btn-primary-text); }
            .pk-btn-confirm:hover { background: var(--color-btn-primary-hover); }
        `;
        document.head.appendChild(style);
    }

    function close() {
        if (overlay) {
            overlay.remove();
            overlay = null;
        }
        if (_triggerEl) {
            _triggerEl.focus();
            _triggerEl = null;
        }
    }

    function escHtml(s) {
        return String(s)
            .replace(/&/g, '&amp;')
            .replace(/</g, '&lt;')
            .replace(/>/g, '&gt;')
            .replace(/"/g, '&quot;');
    }

    function cellContent(col, item) {
        const raw = item[col.key];
        if (col.render) {
            const result = col.render(raw, item);
            if (result instanceof Node) return result;
            // render returned a string - treat as plain text (escape it)
            const span = document.createElement('span');
            span.textContent = String(result);
            return span;
        }
        const span = document.createElement('span');
        if (col.mono) span.className = 'pk-cell-mono';
        span.textContent = raw == null ? '' : String(raw);
        return span;
    }

    function open(opts) {
        const {
            title,
            source,
            columns,
            onSelect,
            multi      = false,
            selected   = [],
            idKey      = 'id',
            searchKeys = null,
            filter     = null,
            sort       = null,
        } = opts;

        injectStyles();
        close();

        _triggerEl = document.activeElement || null;

        overlay = document.createElement('div');
        overlay.className = 'pk-overlay';
        overlay.setAttribute('role', 'dialog');
        overlay.setAttribute('aria-modal', 'true');
        overlay.setAttribute('aria-label', title);
        overlay.addEventListener('click', e => { if (e.target === overlay) close(); });

        const modal = document.createElement('div');
        modal.className = 'pk-modal';

        // Header
        const header = document.createElement('div');
        header.className = 'pk-header';
        const titleEl = document.createElement('span');
        titleEl.className = 'pk-title';
        titleEl.textContent = title;
        const closeBtn = document.createElement('button');
        closeBtn.className = 'pk-close';
        closeBtn.textContent = '\u00d7';
        closeBtn.setAttribute('aria-label', 'Close');
        closeBtn.addEventListener('click', close);
        header.appendChild(titleEl);
        header.appendChild(closeBtn);

        // Search
        const searchWrap = document.createElement('div');
        searchWrap.className = 'pk-search-wrap';
        const searchInput = document.createElement('input');
        searchInput.type = 'search';
        searchInput.className = 'pk-search';
        searchInput.placeholder = 'Search\u2026';
        searchInput.setAttribute('aria-label', 'Search');
        searchWrap.appendChild(searchInput);

        // Table wrapper
        const tableWrap = document.createElement('div');
        tableWrap.className = 'pk-table-wrap';
        tableWrap.setAttribute('role', 'listbox');
        tableWrap.innerHTML = '<div class="pk-loading">Loading\u2026</div>';

        // Footer
        const footer = document.createElement('div');
        footer.className = 'pk-footer';
        const cancelBtn = document.createElement('button');
        cancelBtn.className = 'pk-btn pk-btn-cancel';
        cancelBtn.textContent = 'Cancel';
        cancelBtn.addEventListener('click', close);
        footer.appendChild(cancelBtn);

        let confirmBtn = null;
        if (multi) {
            confirmBtn = document.createElement('button');
            confirmBtn.className = 'pk-btn pk-btn-confirm';
            confirmBtn.textContent = 'Select';
            footer.appendChild(confirmBtn);
        }

        modal.append(header, searchWrap, tableWrap, footer);
        overlay.appendChild(modal);
        document.body.appendChild(overlay);
        searchInput.focus();

        // State
        let allItems = [];
        let visibleItems = [];
        let highlightIdx = -1;
        const selectedIds = new Set(selected.map(v => String(v)));

        function getSearchKeys() {
            if (searchKeys) return searchKeys;
            return columns
                .filter(c => {
                    // include columns whose key is a string or number on the first item
                    if (!allItems.length) return true;
                    const v = allItems[0][c.key];
                    return typeof v === 'string' || typeof v === 'number';
                })
                .map(c => c.key);
        }

        function matchesSearch(item, q) {
            if (!q) return true;
            const lower = q.toLowerCase();
            for (const k of getSearchKeys()) {
                const v = item[k];
                if (v != null && String(v).toLowerCase().includes(lower)) return true;
            }
            return false;
        }

        function renderTable(q) {
            visibleItems = allItems.filter(item => matchesSearch(item, q));
            highlightIdx = -1;

            if (visibleItems.length === 0) {
                tableWrap.innerHTML = '<div class="pk-empty">No results</div>';
                return;
            }

            const table = document.createElement('table');
            table.className = 'pk-table';

            // thead
            const thead = document.createElement('thead');
            const headerRow = document.createElement('tr');
            if (multi) {
                const th = document.createElement('th');
                th.className = 'pk-col-check';
                headerRow.appendChild(th);
            }
            for (const col of columns) {
                const th = document.createElement('th');
                th.textContent = col.label;
                if (col.width) th.style.width = col.width;
                else if (col.flex) th.style.width = 'auto';
                headerRow.appendChild(th);
            }
            thead.appendChild(headerRow);
            table.appendChild(thead);

            // tbody
            const tbody = document.createElement('tbody');
            for (let i = 0; i < visibleItems.length; i++) {
                const item = visibleItems[i];
                const itemId = String(item[idKey]);
                const tr = document.createElement('tr');
                tr.setAttribute('role', 'option');
                tr.dataset.idx = i;
                if (!multi && selectedIds.has(itemId)) tr.classList.add('pk-row-selected');

                if (multi) {
                    const td = document.createElement('td');
                    const cb = document.createElement('input');
                    cb.type = 'checkbox';
                    cb.checked = selectedIds.has(itemId);
                    cb.addEventListener('change', () => {
                        if (cb.checked) selectedIds.add(itemId);
                        else selectedIds.delete(itemId);
                    });
                    td.appendChild(cb);
                    tr.appendChild(td);
                }

                for (const col of columns) {
                    const td = document.createElement('td');
                    if (col.width) td.style.width = col.width;
                    td.appendChild(cellContent(col, item));
                    tr.appendChild(td);
                }

                if (!multi) {
                    tr.addEventListener('click', () => {
                        close();
                        onSelect(item);
                    });
                } else {
                    tr.addEventListener('click', e => {
                        if (e.target.type === 'checkbox') return;
                        const cb = tr.querySelector('input[type="checkbox"]');
                        cb.checked = !cb.checked;
                        if (cb.checked) selectedIds.add(itemId);
                        else selectedIds.delete(itemId);
                    });
                }

                tbody.appendChild(tr);
            }
            table.appendChild(tbody);

            tableWrap.innerHTML = '';
            tableWrap.appendChild(table);
        }

        function setHighlight(idx) {
            const rows = tableWrap.querySelectorAll('tbody tr');
            if (highlightIdx >= 0 && rows[highlightIdx]) {
                rows[highlightIdx].classList.remove('pk-highlighted');
            }
            highlightIdx = Math.max(0, Math.min(idx, visibleItems.length - 1));
            if (rows[highlightIdx]) {
                rows[highlightIdx].classList.add('pk-highlighted');
                rows[highlightIdx].scrollIntoView({ block: 'nearest' });
            }
        }

        // Keyboard navigation
        overlay.addEventListener('keydown', e => {
            if (e.key === 'Escape') {
                e.preventDefault();
                close();
                return;
            }
            if (e.key === 'ArrowDown') {
                e.preventDefault();
                setHighlight(highlightIdx < 0 ? 0 : highlightIdx + 1);
                return;
            }
            if (e.key === 'ArrowUp') {
                e.preventDefault();
                setHighlight(highlightIdx <= 0 ? 0 : highlightIdx - 1);
                return;
            }
            if (e.key === 'Enter') {
                e.preventDefault();
                if (highlightIdx < 0 || !visibleItems[highlightIdx]) return;
                const item = visibleItems[highlightIdx];
                const itemId = String(item[idKey]);
                if (multi) {
                    if (selectedIds.has(itemId)) selectedIds.delete(itemId);
                    else selectedIds.add(itemId);
                    // sync checkbox
                    const rows = tableWrap.querySelectorAll('tbody tr');
                    const row = rows[highlightIdx];
                    if (row) {
                        const cb = row.querySelector('input[type="checkbox"]');
                        if (cb) cb.checked = selectedIds.has(itemId);
                    }
                } else {
                    close();
                    onSelect(item);
                }
            }
        });

        searchInput.addEventListener('input', () => renderTable(searchInput.value));

        if (confirmBtn) {
            confirmBtn.addEventListener('click', () => {
                const result = allItems.filter(item => selectedIds.has(String(item[idKey])));
                close();
                onSelect(result);
            });
        }

        // Load data
        const load = typeof source === 'function'
            ? source()
            : AdminAPI.get(source).then(res => {
                if (!res.ok) throw new Error(res.error || 'Failed to load');
                const d = res.data && res.data.data;
                return Array.isArray(d) ? d : Object.values(d || {});
            });

        load.then(items => {
            let result = items;
            if (filter) result = result.filter(filter);
            if (sort)   result = result.slice().sort(sort);
            allItems = result;
            renderTable('');
        }).catch(err => {
            tableWrap.innerHTML = `<div class="pk-empty">Error: ${escHtml(err.message || String(err))}</div>`;
        });
    }

    return { open, close };
})();
