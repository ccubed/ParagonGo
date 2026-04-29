// AnsiComposerPicker - modal composer for building <ansi fg="..." bg="..."> tags.
//
// The composer has two levels:
//   1. A main editor with a textarea (supports <ansi> tags directly) and a
//      live terminal preview. The user can select any portion of the text and
//      click "Colorize Selection" to open the color picker for that range.
//   2. A color picker sub-modal (FG/BG numeric input, alias list, 256-color
//      palette) that wraps the selected text in an <ansi> tag and returns to
//      the editor.
//
// Usage:
//   AnsiComposerPicker.open({
//     initial: '<ansi fg="username">Hello</ansi>',  // optional pre-fill
//     onApply: (result) => { myInput.value = result; },
//   });

const AnsiComposerPicker = (() => {
    'use strict';

    let overlay     = null;
    let _triggerEl  = null;
    let _aliasData  = null;
    let _aliasReady = false;

    // ── Styles ────────────────────────────────────────────────────────────────
    function injectStyles() {
        if (document.getElementById('acp-styles')) return;
        const s = document.createElement('style');
        s.id = 'acp-styles';
        s.textContent = `
            .acp-overlay {
                position: fixed; inset: 0; background: var(--color-overlay);
                display: flex; align-items: center; justify-content: center;
                z-index: 9999;
            }
            /* ── Main composer modal ── */
            .acp-modal {
                background: var(--color-surface); border-radius: 8px; box-shadow: 0 8px 40px var(--color-shadow);
                width: 640px; max-width: 96vw; max-height: 92vh;
                display: flex; flex-direction: column; overflow: hidden;
            }
            .acp-header {
                padding: 0.85rem 1.1rem 0.7rem; border-bottom: 1px solid var(--color-border);
                display: flex; align-items: center; justify-content: space-between; flex-shrink: 0;
            }
            .acp-title { font-size: 1rem; font-weight: 700; color: var(--color-text); }
            .acp-close { background: none; border: none; font-size: 1.3rem; cursor: pointer; color: var(--color-text-faint); line-height: 1; padding: 0 0.2rem; }
            .acp-close:hover { color: var(--color-text); }
            .acp-body { overflow-y: auto; padding: 1rem 1.1rem; flex: 1; display: flex; flex-direction: column; gap: 0.75rem; background: var(--color-surface); }
            .acp-section-label {
                font-size: 0.72rem; font-weight: 700; text-transform: uppercase;
                letter-spacing: 0.04em; color: var(--color-text-faint); margin-bottom: 0.3rem;
            }
            .acp-toolbar { display: flex; gap: 0.5rem; align-items: center; flex-wrap: wrap; }
            .acp-btn-colorize {
                padding: 0.38rem 0.9rem; background: var(--color-btn-primary-bg); color: var(--color-btn-primary-text); border: none;
                border-radius: 4px; font-size: 0.82rem; font-weight: 600; cursor: pointer;
            }
            .acp-btn-colorize:hover { background: var(--color-btn-primary-hover); }
            .acp-btn-strip {
                padding: 0.38rem 0.9rem; background: var(--color-btn-cancel-bg); color: var(--color-btn-cancel-text); border: 1px solid var(--color-border-medium);
                border-radius: 4px; font-size: 0.82rem; font-weight: 600; cursor: pointer;
            }
            .acp-btn-strip:hover { background: var(--color-btn-cancel-hover); }
            .acp-toolbar-hint { font-size: 0.75rem; color: var(--color-text-secondary); margin-left: auto; }
            .acp-textarea {
                width: 100%; min-height: 90px; padding: 0.5rem 0.65rem;
                border: 1px solid var(--color-border-medium); border-radius: 4px; font-family: monospace;
                font-size: 0.85rem; line-height: 1.55; resize: vertical;
                background: var(--color-surface-alt); color: var(--color-text);
            }
            .acp-textarea:focus { outline: 2px solid var(--color-focus); outline-offset: 1px; border-color: transparent; background: var(--color-surface); }
            .acp-preview {
                min-height: 60px; padding: 0.55rem 0.75rem;
                background: var(--color-code-bg); border-radius: 4px; color: var(--color-code-text);
                font-family: monospace; font-size: 0.9rem; line-height: 1.6;
                white-space: pre-wrap; word-break: break-word;
            }
            .acp-preview-empty { color: var(--color-text-subtle); font-style: italic; }
            .acp-footer {
                padding: 0.7rem 1.1rem; border-top: 1px solid var(--color-border-light);
                display: flex; justify-content: flex-end; gap: 0.5rem; flex-shrink: 0;
                background: var(--color-surface);
            }
            .acp-btn {
                padding: 0.42rem 1.1rem; border-radius: 4px; font-size: 0.85rem;
                font-weight: 600; cursor: pointer; border: 1px solid transparent;
            }
            .acp-btn-cancel { background: var(--color-btn-cancel-bg); color: var(--color-btn-cancel-text); border-color: var(--color-border-medium); }
            .acp-btn-cancel:hover { background: var(--color-btn-cancel-hover); }
            .acp-btn-apply { background: var(--color-btn-primary-bg); color: var(--color-btn-primary-text); }
            .acp-btn-apply:hover { background: var(--color-btn-primary-hover); }

            /* ── Color picker sub-modal ── */
            .acp-sub-overlay {
                position: fixed; inset: 0; background: var(--color-overlay);
                display: flex; align-items: center; justify-content: center;
                z-index: 10000;
            }
            .acp-sub-modal {
                background: var(--color-surface); border-radius: 8px; box-shadow: 0 8px 40px var(--color-shadow);
                width: 520px; max-width: 96vw; max-height: 88vh;
                display: flex; flex-direction: column; overflow: hidden;
            }
            .acp-sub-header {
                padding: 0.75rem 1rem 0.6rem; border-bottom: 1px solid var(--color-border);
                display: flex; align-items: center; justify-content: space-between; flex-shrink: 0;
            }
            .acp-sub-title { font-size: 0.95rem; font-weight: 700; color: var(--color-text); }
            .acp-sub-body { overflow-y: auto; padding: 0.9rem 1rem; flex: 1; background: var(--color-surface); }
            .acp-sub-footer {
                padding: 0.65rem 1rem; border-top: 1px solid var(--color-border-light);
                display: flex; justify-content: flex-end; gap: 0.5rem; flex-shrink: 0;
                background: var(--color-surface);
            }

            /* Selection strip */
            .acp-sel-strip {
                background: var(--color-code-bg); border-radius: 4px; padding: 0.4rem 0.65rem;
                font-family: monospace; font-size: 0.88rem; color: var(--color-code-text);
                margin-bottom: 0.85rem; min-height: 1.8rem; word-break: break-all;
            }

            /* FG / BG channels */
            .acp-channels { display: grid; grid-template-columns: 1fr 1fr; gap: 0.7rem; margin-bottom: 0.85rem; }
            .acp-channel {
                border: 2px solid transparent; border-radius: 6px;
                padding: 0.5rem 0.6rem; cursor: pointer; transition: border-color 0.12s, background 0.12s;
            }
            .acp-channel:hover { background: var(--color-row-hover); }
            .acp-channel.acp-active-fg { border-color: var(--color-primary); background: var(--color-surface-alt); cursor: default; }
            .acp-channel.acp-active-bg { border-color: var(--color-text-subtle); background: var(--color-surface-raised); cursor: default; }
            .acp-channel-label {
                font-size: 0.75rem; font-weight: 700; text-transform: uppercase;
                letter-spacing: 0.05em; color: var(--color-text-muted); margin-bottom: 0.4rem;
                display: flex; align-items: center; gap: 0.4rem; pointer-events: none;
            }
            .acp-badge { font-size: 0.62rem; padding: 0.1rem 0.32rem; border-radius: 3px; font-weight: 700; text-transform: uppercase; }
            .acp-badge-fg { background: var(--color-btn-primary-bg); color: var(--color-btn-primary-text); }
            .acp-badge-bg { background: var(--color-text-subtle); color: var(--color-surface-white); }
            .acp-ch-row { display: flex; gap: 0.4rem; align-items: center; }
            .acp-ch-row input[type="number"] {
                width: 64px; padding: 0.3rem 0.4rem; border: 1px solid var(--color-border-medium);
                border-radius: 4px; font-size: 0.85rem; font-family: monospace;
                background: var(--color-surface-raised); color: var(--color-text);
            }
            .acp-ch-row input:focus { outline: 2px solid var(--color-focus); outline-offset: 1px; border-color: transparent; }
            .acp-swatch { width: 26px; height: 26px; border-radius: 4px; border: 1px solid var(--color-border-medium); background: var(--color-code-bg); flex-shrink: 0; }
            .acp-clear-btn { font-size: 0.72rem; padding: 0.2rem 0.4rem; border: 1px solid var(--color-border-medium); background: var(--color-surface-raised); border-radius: 3px; cursor: pointer; color: var(--color-text-muted); }
            .acp-clear-btn:hover { background: var(--color-surface-alt); }

            /* Alias list */
            .acp-alias-search {
                width: 100%; padding: 0.3rem 0.5rem; border: 1px solid var(--color-border-medium);
                border-radius: 4px; font-size: 0.82rem; margin-bottom: 0.3rem;
                background: var(--color-surface-raised); color: var(--color-text);
            }
            .acp-alias-search:focus { outline: 2px solid var(--color-focus); outline-offset: 1px; border-color: transparent; }
            .acp-alias-list { max-height: 150px; overflow-y: auto; border: 1px solid var(--color-border); border-radius: 4px; margin-bottom: 0.85rem; background: var(--color-surface); }
            .acp-alias-row {
                display: flex; align-items: center; gap: 0.5rem;
                padding: 0.26rem 0.5rem; cursor: pointer;
                border-bottom: 1px solid var(--color-border-faint); user-select: none;
                color: var(--color-text);
            }
            .acp-alias-row:last-child { border-bottom: none; }
            .acp-alias-row:hover { background: var(--color-row-hover); }
            .acp-alias-row.acp-sel-fg  { background: var(--color-active-bg); color: var(--color-active-text); }
            .acp-alias-row.acp-sel-bg  { background: var(--color-text-subtle); color: var(--color-surface-white); }
            .acp-alias-row.acp-sel-both { background: linear-gradient(90deg, var(--color-active-bg) 50%, var(--color-text-subtle) 50%); color: var(--color-surface-white); }
            .acp-alias-swatch { width: 22px; height: 16px; border-radius: 3px; flex-shrink: 0; border: 1px solid var(--color-border-medium); }
            .acp-alias-name { flex: 1; font-size: 0.82rem; font-family: monospace; }
            .acp-alias-code { font-size: 0.7rem; font-family: monospace; color: var(--color-text-placeholder); flex-shrink: 0; }
            .acp-alias-row.acp-sel-fg .acp-alias-code,
            .acp-alias-row.acp-sel-bg .acp-alias-code,
            .acp-alias-row.acp-sel-both .acp-alias-code { color: var(--color-text-placeholder); }
            .acp-alias-sel-badges { display: flex; gap: 0.2rem; flex-shrink: 0; }
            .acp-sel-badge { font-size: 0.58rem; font-weight: 700; padding: 0.08rem 0.26rem; border-radius: 2px; text-transform: uppercase; background: rgba(255,255,255,0.25); color: var(--color-surface-white); }
            .acp-alias-empty { padding: 0.65rem; text-align: center; color: var(--color-text-placeholder); font-size: 0.82rem; font-style: italic; }

            /* 256-color palette */
            .acp-palette { display: flex; flex-wrap: wrap; gap: 2px; margin-bottom: 0.85rem; }
            .acp-palette-cell {
                width: 18px; height: 18px; border-radius: 2px; cursor: pointer;
                border: 2px solid transparent; flex-shrink: 0; transition: transform 0.08s;
            }
            .acp-palette-cell:hover { transform: scale(1.35); border-color: rgba(255,255,255,0.6); z-index: 1; position: relative; }
            .acp-palette-cell.acp-ring-fg { border-color: var(--color-surface-white); box-shadow: 0 0 0 2px var(--color-primary); }
            .acp-palette-cell.acp-ring-bg { border-color: var(--color-surface-white); box-shadow: 0 0 0 2px var(--color-text-faint); }

            /* Sub-modal result preview */
            .acp-result-preview {
                background: var(--color-code-bg); border-radius: 4px; padding: 0.4rem 0.65rem;
                font-family: monospace; font-size: 0.88rem; color: var(--color-code-text);
                min-height: 1.8rem; word-break: break-all;
            }
            .acp-btn-sub-apply { background: var(--color-btn-primary-bg); color: var(--color-btn-primary-text); }
            .acp-btn-sub-apply:hover { background: var(--color-btn-primary-hover); }
            .acp-btn-sub-apply:disabled { background: var(--color-btn-disabled); cursor: not-allowed; }
        `;
        document.head.appendChild(s);
    }

    // ── Utilities ─────────────────────────────────────────────────────────────
    function escHtml(s) {
        return String(s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;');
    }

    function pendingToHex(val) {
        if (val === null) return null;
        const n = typeof val === 'string'
            ? (_aliasData && _aliasData[val] !== undefined ? _aliasData[val] : null)
            : val;
        return n !== null ? AnsiColors.toHex(n) : 'var(--color-text-faint)';
    }

    function pendingToCode(val) {
        if (val === null) return null;
        if (typeof val === 'string') return (_aliasData && _aliasData[val] !== undefined) ? _aliasData[val] : null;
        return val;
    }

    // ── Alias loading ─────────────────────────────────────────────────────────
    async function ensureAliases() {
        if (_aliasReady) return;
        const res = await AdminAPI.get('/admin/api/v1/color-aliases');
        _aliasData = {};
        if (res.ok) {
            const raw = (res.data && res.data.data) || {};
            for (const [k, v] of Object.entries(raw)) {
                const n = parseInt(v, 10);
                if (!isNaN(n)) _aliasData[k] = n;
            }
        }
        ansitags.loadAliases(_aliasData);
        _aliasReady = true;
    }

    // ── Close helpers ─────────────────────────────────────────────────────────
    function closeAll() {
        closeSubModal();
        if (overlay) { overlay.remove(); overlay = null; }
        if (_triggerEl) { _triggerEl.focus(); _triggerEl = null; }
    }

    let subOverlay = null;
    function closeSubModal() {
        if (subOverlay) { subOverlay.remove(); subOverlay = null; }
    }

    // ── Color picker sub-modal ────────────────────────────────────────────────
    // Opens on top of the main composer. On confirm, wraps selText in an <ansi>
    // tag and splices it back into the textarea, then closes itself.
    function openColorPicker({ selText, selStart, selEnd, textarea, onDone }) {
        closeSubModal();

        let pendingFg    = null;
        let pendingBg    = null;
        let activeTarget = 'fg';

        subOverlay = document.createElement('div');
        subOverlay.className = 'acp-sub-overlay';
        subOverlay.addEventListener('click', e => { if (e.target === subOverlay) closeSubModal(); });

        const modal = document.createElement('div');
        modal.className = 'acp-sub-modal';

        // Header
        const header = document.createElement('div');
        header.className = 'acp-sub-header';
        const title = document.createElement('span');
        title.className = 'acp-sub-title';
        title.textContent = 'Colorize Selection';
        const closeBtn = document.createElement('button');
        closeBtn.className = 'acp-close';
        closeBtn.textContent = '\u00d7';
        closeBtn.addEventListener('click', closeSubModal);
        header.append(title, closeBtn);

        // Body
        const body = document.createElement('div');
        body.className = 'acp-sub-body';

        // Selection preview
        const selLabel = document.createElement('div');
        selLabel.className = 'acp-section-label';
        selLabel.textContent = 'Selected text';
        const selStrip = document.createElement('div');
        selStrip.className = 'acp-sel-strip';
        selStrip.innerHTML = selText
            ? ansitags.parse(selText)
            : '<span style="color:var(--color-text-muted);font-style:italic">No selection \u2014 entire input will be wrapped</span>';

        // Channels
        const channels = document.createElement('div');
        channels.className = 'acp-channels';

        function makeChannel(which) {
            const sec = document.createElement('div');
            sec.className = 'acp-channel' + (which === 'fg' ? ' acp-active-fg' : '');
            sec.id = 'acp-sub-' + which + '-sec';
            sec.addEventListener('click', () => setTarget(which));

            const lbl = document.createElement('div');
            lbl.className = 'acp-channel-label';
            const badge = document.createElement('span');
            badge.className = 'acp-badge acp-badge-' + which;
            badge.textContent = which.toUpperCase();
            lbl.append(badge, document.createTextNode(which === 'fg' ? ' Foreground' : ' Background'));

            const row = document.createElement('div');
            row.className = 'acp-ch-row';

            const numInput = document.createElement('input');
            numInput.type = 'number';
            numInput.id = 'acp-sub-' + which + '-num';
            numInput.min = '0'; numInput.max = '255';
            numInput.placeholder = '0\u2013255';
            numInput.addEventListener('click', e => e.stopPropagation());
            numInput.addEventListener('input', () => {
                const v = parseInt(numInput.value, 10);
                if (which === 'fg') pendingFg = (!isNaN(v) && v >= 0 && v <= 255) ? v : null;
                else                pendingBg = (!isNaN(v) && v >= 0 && v <= 255) ? v : null;
                refreshSwatch(which);
                syncRings();
                refreshAliasList();
                refreshResultPreview();
            });

            const swatch = document.createElement('div');
            swatch.className = 'acp-swatch';
            swatch.id = 'acp-sub-' + which + '-sw';

            const clearBtn = document.createElement('button');
            clearBtn.type = 'button';
            clearBtn.className = 'acp-clear-btn';
            clearBtn.textContent = 'None';
            clearBtn.addEventListener('click', e => {
                e.stopPropagation();
                if (which === 'fg') pendingFg = null;
                else                pendingBg = null;
                numInput.value = '';
                refreshSwatch(which);
                syncRings();
                refreshAliasList();
                refreshResultPreview();
            });

            row.append(numInput, swatch, clearBtn);
            sec.append(lbl, row);
            return sec;
        }

        channels.append(makeChannel('fg'), makeChannel('bg'));

        // Alias list
        const aliasLabel = document.createElement('div');
        aliasLabel.className = 'acp-section-label';
        aliasLabel.textContent = 'Color Aliases';
        const aliasSearch = document.createElement('input');
        aliasSearch.type = 'search';
        aliasSearch.className = 'acp-alias-search';
        aliasSearch.id = 'acp-sub-alias-search';
        aliasSearch.placeholder = 'Search aliases\u2026';
        aliasSearch.addEventListener('input', refreshAliasList);
        const aliasList = document.createElement('div');
        aliasList.className = 'acp-alias-list';
        aliasList.id = 'acp-sub-alias-list';
        aliasList.innerHTML = '<div class="acp-alias-empty">Loading\u2026</div>';

        // Palette
        const paletteLabel = document.createElement('div');
        paletteLabel.className = 'acp-section-label';
        paletteLabel.textContent = '256-Color Palette \u2014 click to apply to active channel';
        const palette = document.createElement('div');
        palette.className = 'acp-palette';
        palette.id = 'acp-sub-palette';

        // Result preview
        const resultLabel = document.createElement('div');
        resultLabel.className = 'acp-section-label';
        resultLabel.style.marginTop = '0.2rem';
        resultLabel.textContent = 'Result preview';
        const resultPreview = document.createElement('div');
        resultPreview.className = 'acp-result-preview';
        resultPreview.id = 'acp-sub-result';

        body.append(selLabel, selStrip, channels, aliasLabel, aliasSearch, aliasList, paletteLabel, palette, resultLabel, resultPreview);

        // Footer
        const footer = document.createElement('div');
        footer.className = 'acp-sub-footer';
        const cancelBtn = document.createElement('button');
        cancelBtn.className = 'acp-btn acp-btn-cancel';
        cancelBtn.textContent = 'Cancel';
        cancelBtn.addEventListener('click', closeSubModal);
        const applyBtn = document.createElement('button');
        applyBtn.className = 'acp-btn acp-btn-sub-apply';
        applyBtn.id = 'acp-sub-apply';
        applyBtn.textContent = 'Apply to Selection';
        applyBtn.disabled = true;
        applyBtn.addEventListener('click', () => {
            if (pendingFg === null && pendingBg === null) return;
            let attrs = '';
            if (pendingFg !== null) attrs += ' fg="' + pendingFg + '"';
            if (pendingBg !== null) attrs += ' bg="' + pendingBg + '"';
            const target  = selText || textarea.value;
            const start   = selText ? selStart : 0;
            const end     = selText ? selEnd   : textarea.value.length;
            const wrapped = '<ansi' + attrs + '>' + target + '</ansi>';
            textarea.value = textarea.value.slice(0, start) + wrapped + textarea.value.slice(end);
            const newPos = start + wrapped.length;
            textarea.setSelectionRange(newPos, newPos);
            textarea.focus();
            closeSubModal();
            onDone();
        });
        footer.append(cancelBtn, applyBtn);

        modal.append(header, body, footer);
        subOverlay.appendChild(modal);
        document.body.appendChild(subOverlay);

        subOverlay.addEventListener('keydown', e => {
            if (e.key === 'Escape') { e.preventDefault(); closeSubModal(); }
        });

        // ── Sub-modal internal helpers ──

        function setTarget(which) {
            activeTarget = which;
            const fg = document.getElementById('acp-sub-fg-sec');
            const bg = document.getElementById('acp-sub-bg-sec');
            if (fg) { fg.classList.toggle('acp-active-fg', which === 'fg'); fg.classList.remove('acp-active-bg'); }
            if (bg) { bg.classList.toggle('acp-active-bg', which === 'bg'); bg.classList.remove('acp-active-fg'); }
        }

        function refreshSwatch(which) {
            const val = which === 'fg' ? pendingFg : pendingBg;
            const sw  = document.getElementById('acp-sub-' + which + '-sw');
            if (!sw) return;
            const hex = pendingToHex(val);
            sw.style.background  = hex || 'var(--color-code-bg)';
            sw.style.borderColor = val !== null ? 'var(--color-empty-text)' : 'var(--color-border-medium)';
        }

        function syncRings() {
            const fgCode = pendingToCode(pendingFg);
            const bgCode = pendingToCode(pendingBg);
            document.querySelectorAll('#acp-sub-palette .acp-palette-cell').forEach(cell => {
                const c = parseInt(cell.dataset.code, 10);
                cell.classList.toggle('acp-ring-fg', fgCode === c && typeof pendingFg === 'number');
                cell.classList.toggle('acp-ring-bg', bgCode === c && typeof pendingBg === 'number');
            });
        }

        function refreshAliasList() {
            const list = document.getElementById('acp-sub-alias-list');
            const searchEl = document.getElementById('acp-sub-alias-search');
            if (!list || !_aliasData) return;
            const q = (searchEl ? searchEl.value : '').toLowerCase();
            const names = Object.keys(_aliasData).sort().filter(n => !q || n.includes(q));
            if (!names.length) {
                list.innerHTML = '<div class="acp-alias-empty">No aliases match.</div>';
                return;
            }
            list.innerHTML = names.map(name => {
                const code = _aliasData[name];
                const hex  = AnsiColors.toHex(code);
                const isFg = pendingFg === name;
                const isBg = pendingBg === name;
                let cls = 'acp-alias-row';
                if (isFg && isBg) cls += ' acp-sel-both';
                else if (isFg)   cls += ' acp-sel-fg';
                else if (isBg)   cls += ' acp-sel-bg';
                let badges = '';
                if (isFg) badges += '<span class="acp-sel-badge">FG</span>';
                if (isBg) badges += '<span class="acp-sel-badge">BG</span>';
                return '<div class="' + cls + '" data-name="' + escHtml(name) + '">' +
                    '<div class="acp-alias-swatch" style="background:' + hex + '"></div>' +
                    '<span class="acp-alias-name">' + escHtml(name) + '</span>' +
                    (badges ? '<span class="acp-alias-sel-badges">' + badges + '</span>' : '') +
                    '<span class="acp-alias-code">' + code + '</span>' +
                    '</div>';
            }).join('');

            list.querySelectorAll('.acp-alias-row').forEach(row => {
                row.addEventListener('click', () => {
                    const name = row.dataset.name;
                    if (activeTarget === 'bg') {
                        pendingBg = (pendingBg === name) ? null : name;
                        const inp = document.getElementById('acp-sub-bg-num');
                        if (inp) inp.value = '';
                        refreshSwatch('bg');
                    } else {
                        pendingFg = (pendingFg === name) ? null : name;
                        const inp = document.getElementById('acp-sub-fg-num');
                        if (inp) inp.value = '';
                        refreshSwatch('fg');
                    }
                    syncRings();
                    refreshAliasList();
                    refreshResultPreview();
                });
            });
        }

        function buildPalette() {
            const p = document.getElementById('acp-sub-palette');
            if (!p) return;
            let html = '';
            for (let i = 0; i < 256; i++) {
                html += '<div class="acp-palette-cell" data-code="' + i + '" style="background:' + AnsiColors.toHex(i) + '" title="' + i + '"></div>';
            }
            p.innerHTML = html;
            p.querySelectorAll('.acp-palette-cell').forEach(cell => {
                cell.addEventListener('click', () => {
                    const code = parseInt(cell.dataset.code, 10);
                    if (activeTarget === 'bg') {
                        pendingBg = (pendingBg === code) ? null : code;
                        const inp = document.getElementById('acp-sub-bg-num');
                        if (inp) inp.value = pendingBg !== null ? pendingBg : '';
                        refreshSwatch('bg');
                    } else {
                        pendingFg = (pendingFg === code) ? null : code;
                        const inp = document.getElementById('acp-sub-fg-num');
                        if (inp) inp.value = pendingFg !== null ? pendingFg : '';
                        refreshSwatch('fg');
                    }
                    syncRings();
                    refreshAliasList();
                    refreshResultPreview();
                });
            });
        }

        function refreshResultPreview() {
            const target = selText || textarea.value || '(entire input)';
            let attrs = '';
            if (pendingFg !== null) attrs += ' fg="' + pendingFg + '"';
            if (pendingBg !== null) attrs += ' bg="' + pendingBg + '"';
            const tag = attrs ? '<ansi' + attrs + '>' + target + '</ansi>' : escHtml(target);
            const prev = document.getElementById('acp-sub-result');
            if (prev) prev.innerHTML = ansitags.parse(tag);
            const btn = document.getElementById('acp-sub-apply');
            if (btn) btn.disabled = (pendingFg === null && pendingBg === null);
        }

        // Boot
        buildPalette();
        refreshResultPreview();
        ensureAliases().then(() => refreshAliasList());
    }

    // ── Main composer ─────────────────────────────────────────────────────────
    function open({ initial = '', onApply }) {
        injectStyles();
        closeAll();

        _triggerEl = document.activeElement || null;

        overlay = document.createElement('div');
        overlay.className = 'acp-overlay';
        overlay.setAttribute('role', 'dialog');
        overlay.setAttribute('aria-modal', 'true');
        overlay.setAttribute('aria-label', 'ANSI Tag Composer');
        overlay.addEventListener('click', e => { if (e.target === overlay) { closeSubModal(); } });

        const modal = document.createElement('div');
        modal.className = 'acp-modal';

        // Header
        const header = document.createElement('div');
        header.className = 'acp-header';
        const titleEl = document.createElement('span');
        titleEl.className = 'acp-title';
        titleEl.textContent = 'ANSI Tag Composer';
        const closeBtn = document.createElement('button');
        closeBtn.className = 'acp-close';
        closeBtn.textContent = '\u00d7';
        closeBtn.setAttribute('aria-label', 'Close');
        closeBtn.addEventListener('click', closeAll);
        header.append(titleEl, closeBtn);

        // Body
        const body = document.createElement('div');
        body.className = 'acp-body';

        // Toolbar
        const toolbarLabel = document.createElement('div');
        toolbarLabel.className = 'acp-section-label';
        toolbarLabel.textContent = 'Source';
        const toolbar = document.createElement('div');
        toolbar.className = 'acp-toolbar';
        const colorizeBtn = document.createElement('button');
        colorizeBtn.type = 'button';
        colorizeBtn.className = 'acp-btn-colorize';
        colorizeBtn.innerHTML = '<span style="font-size:0.95rem">\u270e</span> Colorize Selection';
        const stripBtn = document.createElement('button');
        stripBtn.type = 'button';
        stripBtn.className = 'acp-btn-strip';
        stripBtn.innerHTML = '\u2716 Strip Tags';
        const hint = document.createElement('span');
        hint.className = 'acp-toolbar-hint';
        hint.textContent = 'Select text, then click Colorize';
        toolbar.append(colorizeBtn, stripBtn, hint);

        // Textarea
        const textarea = document.createElement('textarea');
        textarea.className = 'acp-textarea';
        textarea.spellcheck = false;
        textarea.placeholder = 'Type your text here\u2026';
        textarea.value = initial;
        textarea.addEventListener('input', renderPreview);

        // Preview
        const previewLabel = document.createElement('div');
        previewLabel.className = 'acp-section-label';
        previewLabel.textContent = 'Preview';
        const previewEl = document.createElement('div');
        previewEl.className = 'acp-preview';
        previewEl.id = 'acp-main-preview';

        body.append(toolbarLabel, toolbar, textarea, previewLabel, previewEl);

        // Footer
        const footer = document.createElement('div');
        footer.className = 'acp-footer';
        const cancelBtn = document.createElement('button');
        cancelBtn.className = 'acp-btn acp-btn-cancel';
        cancelBtn.textContent = 'Cancel';
        cancelBtn.addEventListener('click', closeAll);
        const applyBtn = document.createElement('button');
        applyBtn.className = 'acp-btn acp-btn-apply';
        applyBtn.textContent = 'Apply';
        applyBtn.addEventListener('click', () => {
            const result = textarea.value;
            closeAll();
            onApply(result);
        });
        footer.append(cancelBtn, applyBtn);

        modal.append(header, body, footer);
        overlay.appendChild(modal);
        document.body.appendChild(overlay);

        overlay.addEventListener('keydown', e => {
            if (e.key === 'Escape' && !subOverlay) { e.preventDefault(); closeAll(); }
        });

        function renderPreview() {
            const src = textarea.value;
            const out = document.getElementById('acp-main-preview');
            if (!out) return;
            if (!src.trim()) {
                out.innerHTML = '<span class="acp-preview-empty">Preview will appear here\u2026</span>';
                return;
            }
            out.innerHTML = ansitags.parse(src);
        }

        colorizeBtn.addEventListener('click', () => {
            const start   = textarea.selectionStart;
            const end     = textarea.selectionEnd;
            const selText = textarea.value.slice(start, end);
            openColorPicker({
                selText,
                selStart: start,
                selEnd:   end,
                textarea,
                onDone:   renderPreview,
            });
        });

        stripBtn.addEventListener('click', () => {
            textarea.value = ansitags.parse(textarea.value, { stripTags: true });
            renderPreview();
        });

        renderPreview();
        textarea.focus();
    }

    return { open, close: closeAll };
})();
