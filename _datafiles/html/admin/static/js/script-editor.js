// ScriptEditor - Monaco editor in an isolated iframe.
//
// The editor is embedded inline below the script section buttons. Clicking
// "Pop Out" moves the same iframe into a fullscreen modal. Closing the modal
// moves it back inline. One iframe instance is created per textareaId and
// reused for the lifetime of the page.
//
// The <textarea> is always the form's source of truth.
//
// Public API:
//   ScriptEditor.init(textareaId, scriptType) -> syncFn
//     syncFn()        - push textarea.value into the editor
//     syncFn(value)   - set textarea.value and push into editor
//
//   ScriptEditor.open(textareaId)       - pop out to fullscreen modal
//   ScriptEditor.getEditor(textareaId)  - returns iframe contentWindow or null
//
//   ScriptEditor.setLang(textareaId, lang, locked)
//     Set the active language ('js' | 'lua') and whether the language selector
//     is locked. Pages call this on load: locked=true for an existing script
//     file (language is fixed until the file is deleted), false for a new one.
//   ScriptEditor.getLang(textareaId)    - returns active language ('js' | 'lua')
//   ScriptEditor.lockLang(textareaId, locked) - lock/unlock the selector only

const ScriptEditor = (() => {
    'use strict';

    const MONACO_BASE = '/admin/static/js/monaco/vs';
    const FRAME_URL   = '/admin/static/js/monaco-editor-frame.html';
    const INLINE_HEIGHT = '400px';

    // textareaId -> { scriptType, iframe, iframeWin, inlineContainer, ready }
    const registry = {};

    // Cached scripting-functions schema (shared by every pop-out reference panel).
    let _schemaCache = null;
    let _schemaPromise = null;

    // Reference-panel width, remembered across pop-outs for the page session.
    let _refWidth = null;

    // -------------------------------------------------------------------------
    // Public API
    // -------------------------------------------------------------------------

    function init(textareaId, scriptType) {
        const textarea = document.getElementById(textareaId);
        if (!textarea) return function () {};

        textarea.style.display = 'none';

        const rec = {
            scriptType: scriptType || null,
            lang: 'js',
            iframe: null,
            iframeWin: null,
            inlineContainer: null,
            langSelect: null,
            langMount: null,
            ready: false,
        };
        registry[textareaId] = rec;

        // Build the inline container that sits in the page below the button row.
        const container = document.createElement('div');
        container.style.cssText = [
            'width:100%',
            'height:' + INLINE_HEIGHT,
            'margin-top:0.5rem',
            'position:relative',
        ].join(';');
        rec.inlineContainer = container;

        // Language selector. Lives in the page toolbar (the slot where the
        // "has script" badge used to be); it is moved into the modal header on
        // pop-out and back to the toolbar on close (same lifecycle as the iframe).
        rec.langSelect = _buildLangSelect(textareaId, rec);
        rec.langMount = (textarea.closest('.script-section') || document)
            .querySelector('.script-lang-mount');
        if (rec.langMount) {
            rec.langMount.appendChild(rec.langSelect);
        } else {
            // Fallback: keep the selector visible above the editor if a page has
            // no toolbar slot.
            container.appendChild(rec.langSelect);
        }

        // Insert after the textarea (which is hidden, sitting inside script-section).
        textarea.parentNode.insertBefore(container, textarea.nextSibling);

        // Build the iframe and place it in the inline container.
        const iframe = _buildIframe(textareaId, textarea, rec);
        rec.iframe = iframe;
        iframe.style.cssText = 'width:100%;height:100%;border:none;display:block;';
        container.appendChild(iframe);

        return function syncFn(newValue, lang) {
            if (lang !== undefined && lang !== null) {
                rec.lang = (lang === 'lua') ? 'lua' : 'js';
                if (rec.langSelect) rec.langSelect.value = rec.lang;
            }
            if (newValue !== undefined) {
                textarea.value = newValue;
            }
            if (rec.iframeWin) {
                rec.iframeWin.postMessage({
                    type: 'monaco-set',
                    value: textarea.value,
                    lang: rec.lang,
                }, '*');
            }
        };
    }

    // Set the active language and (optionally) the locked state of the selector.
    // locked === undefined leaves the current lock state untouched.
    function setLang(textareaId, lang, locked) {
        const rec = registry[textareaId];
        if (!rec) return;
        rec.lang = (lang === 'lua') ? 'lua' : 'js';
        if (rec.langSelect) {
            rec.langSelect.value = rec.lang;
            if (locked !== undefined) rec.langSelect.disabled = !!locked;
        }
        _pushToEditor(textareaId, rec);
    }

    function getLang(textareaId) {
        const rec = registry[textareaId];
        return (rec && rec.lang) || 'js';
    }

    function lockLang(textareaId, locked) {
        const rec = registry[textareaId];
        if (rec && rec.langSelect) rec.langSelect.disabled = !!locked;
    }

    // Push the textarea's current value plus the active language into the editor.
    function _pushToEditor(textareaId, rec) {
        if (!rec.iframeWin) return;
        const textarea = document.getElementById(textareaId);
        rec.iframeWin.postMessage({
            type: 'monaco-set',
            value: textarea ? textarea.value : '',
            lang: rec.lang,
        }, '*');
    }

    // Build the language <select>. It drives the editor language live and is
    // disabled when editing an existing script file (the language is fixed until
    // the file is deleted).
    function _buildLangSelect(textareaId, rec) {
        const select = document.createElement('select');
        select.style.cssText = 'font-size:0.8rem;padding:0.2rem 0.4rem;border:1px solid var(--color-border-medium);border-radius:4px;';
        [['js', 'JavaScript'], ['lua', 'Lua']].forEach(function (opt) {
            const o = document.createElement('option');
            o.value = opt[0];
            o.textContent = opt[1];
            select.appendChild(o);
        });
        select.value = rec.lang;
        select.title = 'Language is fixed once a script file exists. Clear and save to delete the file, then choose a different language.';
        select.addEventListener('change', function () {
            rec.lang = (select.value === 'lua') ? 'lua' : 'js';
            _pushToEditor(textareaId, rec);
        });
        return select;
    }

    function getEditor(textareaId) {
        const rec = registry[textareaId];
        return (rec && rec.iframeWin) || null;
    }

    function open(textareaId) {
        if (document.getElementById('monaco-modal-overlay')) return;
        const rec = registry[textareaId];
        if (!rec || !rec.iframe) return;

        const textarea = document.getElementById(textareaId);

        // ---- Overlay ----
        const overlay = document.createElement('div');
        overlay.id = 'monaco-modal-overlay';
        overlay.style.cssText = [
            'position:fixed', 'inset:0', 'z-index:9998',
            'background:rgba(0,0,0,0.75)',
            'display:flex', 'align-items:stretch', 'justify-content:stretch',
        ].join(';');

        // ---- Modal shell ----
        const modal = document.createElement('div');
        modal.style.cssText = [
            'display:flex', 'flex-direction:column',
            'flex:1', 'margin:1.5rem',
            'background:#1e1e1e',
            'box-shadow:0 8px 40px rgba(0,0,0,0.7)',
            'min-height:0', 'min-width:0',
        ].join(';');

        // ---- Header ----
        const header = document.createElement('div');
        header.style.cssText = [
            'display:flex', 'align-items:center', 'gap:0.75rem',
            'padding:0.45rem 0.75rem',
            'background:#2d2d2d', 'border-bottom:1px solid #444',
            'flex-shrink:0',
            'border-radius:6px 6px 0 0',
        ].join(';');

        const titleEl = document.createElement('span');
        titleEl.style.cssText = 'flex:1;font-size:0.8rem;color:#aaa;font-family:monospace;';
        titleEl.textContent = 'Script Editor';

        const addHandlerBtn = document.createElement('button');
        addHandlerBtn.type = 'button';
        addHandlerBtn.textContent = '+ Add Event Handler';
        addHandlerBtn.style.cssText = [
            'font-size:0.78rem', 'padding:0.25rem 0.7rem',
            'border:1px solid #666', 'border-radius:3px',
            'background:#3a3a3a', 'color:#ccc', 'cursor:pointer', 'flex-shrink:0',
        ].join(';');
        addHandlerBtn.addEventListener('mouseenter', function () {
            addHandlerBtn.style.background = '#555'; addHandlerBtn.style.color = '#fff';
        });
        addHandlerBtn.addEventListener('mouseleave', function () {
            addHandlerBtn.style.background = '#3a3a3a'; addHandlerBtn.style.color = '#ccc';
        });
        addHandlerBtn.addEventListener('click', function () {
            const rec = registry[textareaId];
            ScriptWizard.open({ scriptType: rec.scriptType, textareaId: textareaId });
        });

        const hintEl = document.createElement('span');
        hintEl.style.cssText = 'font-size:0.75rem;color:#666;';
        hintEl.textContent = 'Esc to close';

        const closeBtn = document.createElement('button');
        closeBtn.type = 'button';
        closeBtn.textContent = '\u2715 Close';
        closeBtn.style.cssText = [
            'font-size:0.78rem', 'padding:0.25rem 0.7rem',
            'border:1px solid #666', 'border-radius:3px',
            'background:#3a3a3a', 'color:#ccc', 'cursor:pointer', 'flex-shrink:0',
        ].join(';');
        closeBtn.addEventListener('mouseenter', function () {
            closeBtn.style.background = '#555'; closeBtn.style.color = '#fff';
        });
        closeBtn.addEventListener('mouseleave', function () {
            closeBtn.style.background = '#3a3a3a'; closeBtn.style.color = '#ccc';
        });

        header.appendChild(titleEl);
        if (rec.langSelect) header.appendChild(rec.langSelect);
        header.appendChild(addHandlerBtn);
        header.appendChild(hintEl);
        header.appendChild(closeBtn);

        // ---- Body: editor mount (left ~67%) + function reference (right 33%) ----
        const body = document.createElement('div');
        // position:relative anchors the reference panel when it switches to
        // absolute (auto-hide); overflow:hidden clips the off-screen portion.
        body.style.cssText = 'display:flex;flex-direction:row;flex:1;min-height:0;min-width:0;position:relative;overflow:hidden;';

        // A plain div that the iframe will be moved into.
        // No overflow:hidden, no border-radius — avoids compositing layer
        // issues that misalign Monaco's pointer-event hit-testing.
        const mount = document.createElement('div');
        mount.style.cssText = 'flex:1;min-height:0;min-width:0;position:relative;';

        // Reserve the right third of the modal for a searchable function reference.
        const reference = _buildReferencePanel(rec);

        body.appendChild(mount);
        body.appendChild(reference);

        modal.appendChild(header);
        modal.appendChild(body);
        overlay.appendChild(modal);
        document.body.appendChild(overlay);

        // Move the iframe from inline container into the modal mount.
        // Reset its size to fill the modal.
        const iframe = rec.iframe;
        iframe.style.cssText = 'width:100%;height:100%;border:none;display:block;';
        mount.appendChild(iframe);

        // Tell Monaco to re-measure now that it's in a new (larger) container.
        if (rec.iframeWin) {
            rec.iframeWin.postMessage({ type: 'monaco-layout' }, '*');
        }

        // ---- Close: move iframe back inline ----
        function close() {
            // Restore inline sizing and move back. The language selector was
            // moved into the modal header, so return it to the toolbar slot.
            iframe.style.cssText = 'width:100%;height:100%;border:none;display:block;';
            if (rec.langSelect && rec.langMount) {
                rec.langMount.appendChild(rec.langSelect);
            }
            rec.inlineContainer.appendChild(iframe);

            if (rec.iframeWin) {
                rec.iframeWin.postMessage({ type: 'monaco-layout' }, '*');
            }

            overlay.remove();
            document.removeEventListener('keydown', onKeyDown);
        }

        function onKeyDown(e) {
            if (e.key === 'Escape') close();
        }

        closeBtn.addEventListener('click', close);
        document.addEventListener('keydown', onKeyDown);
        overlay.addEventListener('mousedown', function (e) {
            if (e.target === overlay) close();
        });
    }

    // -------------------------------------------------------------------------
    // Internal: build the iframe
    // -------------------------------------------------------------------------

    function _buildIframe(textareaId, textarea, rec) {
        const iframe = document.createElement('iframe');
        iframe.src = FRAME_URL;
        iframe.style.cssText = 'width:100%;height:100%;border:none;display:block;';
        iframe.setAttribute('sandbox', 'allow-scripts allow-same-origin');

        // Message handler for this iframe.
        function onMessage(e) {
            if (e.source !== iframe.contentWindow) return;
            const msg = e.data;
            if (!msg || !msg.type) return;

            if (msg.type === 'monaco-ready') {
                rec.iframeWin = iframe.contentWindow;
                rec.ready = true;
                // The page may have loaded a script (and its language) while the
                // editor was still initializing, in which case the earlier
                // monaco-set was skipped because iframeWin was null. Reconcile
                // now so the editor reflects the current value and language.
                rec.iframeWin.postMessage({
                    type: 'monaco-set',
                    value: textarea.value,
                    lang: rec.lang,
                }, '*');
            } else if (msg.type === 'monaco-change') {
                textarea.value = msg.value;
            } else if (msg.type === 'monaco-value') {
                textarea.value = msg.value;
            } else if (msg.type === 'monaco-open-wizard') {
                ScriptWizard.open({ scriptType: rec.scriptType, textareaId: textareaId });
            }
        }
        window.addEventListener('message', onMessage);

        // Once the iframe's HTML has loaded, send the init config.
        iframe.addEventListener('load', function () {
            iframe.contentWindow.postMessage({
                type: 'monaco-init',
                monacoBase: MONACO_BASE,
                scriptType: rec.scriptType,
                lang: rec.lang,
                initialValue: textarea.value,
            }, '*');
        });

        return iframe;
    }

    // -------------------------------------------------------------------------
    // Internal: function reference side panel (right third of the pop-out modal)
    // -------------------------------------------------------------------------

    // Fetch (once) the same scripting-functions schema that backs the
    // /admin/scripting-functions reference page and the editor intellisense.
    // Uses the shared AdminAPI helper when present (cached in sessionStorage),
    // falling back to a plain fetch if it has not loaded on this page.
    function _loadSchema() {
        if (_schemaCache) return Promise.resolve(_schemaCache);
        if (_schemaPromise) return _schemaPromise;
        const path = '/admin/api/v1/scripting/functions';
        let req;
        if (typeof AdminAPI !== 'undefined') {
            req = AdminAPI.get(path).then(function (res) {
                return (res && res.ok && res.data) ? res.data.data : null;
            });
        } else {
            req = fetch(path, { credentials: 'include' })
                .then(function (r) { return r.json(); })
                .then(function (j) { return (j && j.data) || null; });
        }
        _schemaPromise = req
            .then(function (schema) { _schemaCache = schema; return schema; })
            .catch(function () { return null; });
        return _schemaPromise;
    }

    const _RET_COLORS = {
        boolean: '#d19a66', number: '#61afef', string: '#98c379',
        void: '#777', object: '#c678dd',
    };

    function _retBadgeColor(t) {
        if (!t) return _RET_COLORS.void;
        const base = t.replace(/\[\]/g, '').split('|')[0].trim();
        return _RET_COLORS[base] || '#c678dd';
    }

    function _esc(s) {
        return String(s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
    }

    // Strip variadic ("...") and optional ("?") markers from a param name.
    function _paramName(name) {
        return name.replace(/^\.\.\./, '').replace(/\?$/, '');
    }

    // Render a function's name + typed parameter list into syntax-coloured HTML.
    function _signatureHtml(fn) {
        const parts = (fn.params || []).map(function (p) {
            const variadic = /^\.\.\./.test(p.name);
            const optional = /\?$/.test(p.name);
            let s = '<span style="color:#98c379">' + (variadic ? '...' : '') + _esc(_paramName(p.name)) + (optional ? '?' : '') + '</span>';
            if (p.type) s += '<span style="color:#777">: ' + _esc(p.type) + '</span>';
            return s;
        });
        return '<span style="color:#61afef;font-weight:600">' + _esc(fn.name) + '</span>' +
            '<span style="color:#777">(</span>' +
            parts.join('<span style="color:#777">, </span>') +
            '<span style="color:#777">)</span>';
    }

    // Build a call stub (required params only) for insertion into the editor.
    function _callStub(fn) {
        const args = (fn.params || [])
            .filter(function (p) { return !/\?$/.test(p.name) && !/^\.\.\./.test(p.name); })
            .map(function (p) { return _paramName(p.name); });
        return fn.name + '(' + args.join(', ') + ')';
    }

    // Build one collapsible reference card for a function. When allowInsert is
    // true (global engine functions, which are called identically in JS and
    // Lua) an "Insert" button drops a call stub at the editor's cursor.
    function _buildFnCard(fn, rec, allowInsert) {
        const card = document.createElement('div');
        card.style.cssText = 'border-bottom:1px solid #333;';
        const searchText = (fn.name + ' ' + (fn.description || '') + ' ' +
            (fn.params || []).map(function (p) { return p.name + ' ' + (p.type || ''); }).join(' ')).toLowerCase();
        card.setAttribute('data-search', searchText);

        const head = document.createElement('div');
        head.style.cssText = 'display:flex;align-items:center;gap:0.4rem;padding:0.4rem 0.6rem;cursor:pointer;font-family:monospace;font-size:0.76rem;';
        head.innerHTML = '<span style="flex:1;min-width:0;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">' + _signatureHtml(fn) + '</span>';

        const rt = fn.returnType || 'void';
        const badge = document.createElement('span');
        badge.textContent = rt;
        badge.style.cssText = 'flex-shrink:0;font-size:0.66rem;padding:0.05rem 0.35rem;border-radius:3px;color:#1e1e1e;font-weight:700;background:' + _retBadgeColor(rt) + ';';
        head.appendChild(badge);

        const bodyEl = document.createElement('div');
        bodyEl.style.cssText = 'display:none;padding:0.1rem 0.6rem 0.7rem;';

        if (fn.description) {
            const d = document.createElement('p');
            d.textContent = fn.description;
            d.style.cssText = 'margin:0.3rem 0 0.5rem;font-size:0.76rem;line-height:1.5;color:#bbb;';
            bodyEl.appendChild(d);
        }
        if ((fn.params || []).length) {
            const tbl = document.createElement('table');
            tbl.style.cssText = 'width:100%;border-collapse:collapse;font-size:0.72rem;margin-bottom:0.5rem;';
            (fn.params || []).forEach(function (p) {
                const tr = document.createElement('tr');
                tr.innerHTML =
                    '<td style="padding:0.15rem 0.4rem 0.15rem 0;color:#98c379;font-family:monospace;white-space:nowrap;vertical-align:top">' + _esc(_paramName(p.name)) + '</td>' +
                    '<td style="padding:0.15rem 0.4rem;color:#777;font-family:monospace;white-space:nowrap;vertical-align:top">' + _esc(p.type || '') + '</td>' +
                    '<td style="padding:0.15rem 0;color:#aaa;vertical-align:top">' + _esc(p.description || '') + '</td>';
                tbl.appendChild(tr);
            });
            bodyEl.appendChild(tbl);
        }
        if (fn.returnSemantics) {
            const r = document.createElement('p');
            r.innerHTML = '<span style="color:#777">returns:</span> ' + _esc(fn.returnSemantics);
            r.style.cssText = 'margin:0 0 0.5rem;font-size:0.72rem;color:#aaa;';
            bodyEl.appendChild(r);
        }
        if (allowInsert) {
            const insertBtn = document.createElement('button');
            insertBtn.type = 'button';
            insertBtn.textContent = '+ Insert';
            insertBtn.style.cssText = 'font-size:0.7rem;padding:0.2rem 0.55rem;border:1px solid #555;border-radius:3px;background:#3a3a3a;color:#ccc;cursor:pointer;';
            insertBtn.addEventListener('click', function (e) {
                e.stopPropagation();
                if (rec.iframeWin) {
                    rec.iframeWin.postMessage({ type: 'monaco-insert', stub: _callStub(fn) }, '*');
                }
            });
            bodyEl.appendChild(insertBtn);
        }

        head.addEventListener('click', function () {
            bodyEl.style.display = (bodyEl.style.display === 'none') ? 'block' : 'none';
        });

        card.appendChild(head);
        card.appendChild(bodyEl);
        return card;
    }

    function _buildReferencePanel(rec) {
        const HIDDEN_STRIP = 28; // px of the panel left visible when auto-hidden

        const panel = document.createElement('div');
        panel.style.cssText = [
            'width:' + (_refWidth || '33%'), 'flex-shrink:0', 'min-width:0',
            'display:flex', 'flex-direction:column',
            'background:#252526', 'border-left:1px solid #444', 'color:#ccc',
            'position:relative', 'transition:transform 0.25s ease', 'will-change:transform',
        ].join(';');

        // ---- Drag handle (left edge) to resize the panel width ----
        let dragging = false;
        const handle = document.createElement('div');
        handle.title = 'Drag to resize';
        handle.style.cssText = 'position:absolute;left:0;top:0;bottom:0;width:6px;cursor:col-resize;background:transparent;z-index:3;';
        handle.addEventListener('mouseenter', function () { handle.style.background = '#555'; });
        handle.addEventListener('mouseleave', function () { if (!dragging) handle.style.background = 'transparent'; });
        handle.addEventListener('mousedown', function (e) {
            e.preventDefault();
            dragging = true;
            handle.style.background = '#61afef';
            const row = panel.parentNode; // the flex body row
            // Block iframe pointer events so the editor doesn't swallow mousemove.
            if (rec.iframe) rec.iframe.style.pointerEvents = 'none';
            document.body.style.cursor = 'col-resize';
            function onMove(ev) {
                const rect = row.getBoundingClientRect();
                let w = rect.right - ev.clientX;
                const min = 200;
                const max = rect.width - 220;
                if (w < min) w = min;
                if (max > min && w > max) w = max;
                _refWidth = w + 'px';
                panel.style.width = _refWidth;
            }
            function onUp() {
                dragging = false;
                handle.style.background = 'transparent';
                if (rec.iframe) rec.iframe.style.pointerEvents = '';
                document.body.style.cursor = '';
                document.removeEventListener('mousemove', onMove);
                document.removeEventListener('mouseup', onUp);
                if (autoHide) flyClosed(); // re-hide if the cursor left the panel
            }
            document.addEventListener('mousemove', onMove);
            document.addEventListener('mouseup', onUp);
        });

        // ---- Auto-hide state ----
        let autoHide = false;
        function flyOpen() { panel.style.transform = 'translateX(0)'; }
        function flyClosed() { panel.style.transform = 'translateX(calc(100% - ' + HIDDEN_STRIP + 'px))'; }
        function applyAutoHide(on) {
            autoHide = on;
            hideBtn.textContent = on ? '⇤ Pinned off' : '⇥ Auto-hide';
            hideBtn.style.color = on ? '#61afef' : '#aaa';
            hideBtn.style.borderColor = on ? '#61afef' : '#555';
            if (on) {
                panel.style.position = 'absolute';
                panel.style.top = '0';
                panel.style.right = '0';
                panel.style.bottom = '0';
                panel.style.boxShadow = '-8px 0 24px rgba(0,0,0,0.5)';
                flyClosed();
            } else {
                panel.style.position = 'relative';
                panel.style.top = '';
                panel.style.right = '';
                panel.style.bottom = '';
                panel.style.boxShadow = '';
                panel.style.transform = 'none';
            }
        }
        panel.addEventListener('mouseenter', function () { if (autoHide) flyOpen(); });
        panel.addEventListener('mouseleave', function () { if (autoHide && !dragging) flyClosed(); });

        // Header with the auto-hide toggle and a link to the full reference page.
        const head = document.createElement('div');
        head.style.cssText = 'display:flex;align-items:center;gap:0.5rem;padding:0.45rem 0.6rem 0.45rem 0.7rem;border-bottom:1px solid #444;flex-shrink:0;';
        const title = document.createElement('span');
        title.textContent = 'Function Reference';
        title.style.cssText = 'flex:1;font-size:0.78rem;color:#aaa;min-width:0;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;';
        const hideBtn = document.createElement('button');
        hideBtn.type = 'button';
        hideBtn.textContent = '⇥ Auto-hide';
        hideBtn.title = 'Auto-hide: slide the panel off-screen until you hover the right edge.';
        hideBtn.style.cssText = 'font-size:0.68rem;padding:0.15rem 0.4rem;border:1px solid #555;border-radius:3px;background:#3a3a3a;color:#aaa;cursor:pointer;flex-shrink:0;';
        hideBtn.addEventListener('click', function () { applyAutoHide(!autoHide); });
        const docsLink = document.createElement('a');
        docsLink.href = '/admin/scripting-functions';
        docsLink.target = '_blank';
        docsLink.rel = 'noopener';
        docsLink.textContent = 'Full docs ↗';
        docsLink.style.cssText = 'font-size:0.72rem;color:#61afef;text-decoration:none;flex-shrink:0;';
        head.appendChild(title);
        head.appendChild(hideBtn);
        head.appendChild(docsLink);

        const search = document.createElement('input');
        search.type = 'search';
        search.placeholder = 'Search functions…';
        search.style.cssText = 'margin:0.5rem 0.6rem;padding:0.35rem 0.5rem;background:#1e1e1e;border:1px solid #444;border-radius:4px;color:#ddd;font-size:0.78rem;flex-shrink:0;';

        const list = document.createElement('div');
        list.style.cssText = 'flex:1;overflow-y:auto;min-height:0;';

        panel.appendChild(handle);
        panel.appendChild(head);
        panel.appendChild(search);
        panel.appendChild(list);

        function groupHeader(text) {
            const h = document.createElement('div');
            h.textContent = text;
            h.setAttribute('data-group', '1');
            h.style.cssText = 'position:sticky;top:0;background:#2d2d2d;color:#888;font-size:0.66rem;font-weight:700;text-transform:uppercase;letter-spacing:0.05em;padding:0.35rem 0.6rem;border-bottom:1px solid #333;z-index:1;';
            return h;
        }

        _loadSchema().then(function (schema) {
            if (!schema) {
                list.innerHTML = '<div style="padding:1rem;font-size:0.8rem;color:#888">Could not load function reference. <a href="/admin/scripting-functions" target="_blank" rel="noopener" style="color:#61afef">Open full docs ↗</a></div>';
                return;
            }
            // Global engine functions — the same set documented on the
            // /admin/scripting-functions page. Callable identically in JS & Lua.
            const engine = schema.engineFunctions || [];
            if (engine.length) {
                list.appendChild(groupHeader('Global Functions'));
                engine.forEach(function (fn) { list.appendChild(_buildFnCard(fn, rec, true)); });
            }
            // Event handlers for the active script type (reference only — use the
            // "+ Add Event Handler" button to scaffold one in the correct language).
            const typeDef = schema.scriptTypes && rec.scriptType && schema.scriptTypes[rec.scriptType];
            if (typeDef && typeDef.functions && typeDef.functions.length) {
                list.appendChild(groupHeader((typeDef.label || rec.scriptType) + ' Event Handlers'));
                typeDef.functions.forEach(function (fn) { list.appendChild(_buildFnCard(fn, rec, false)); });
            }
        });

        let searchTimer;
        search.addEventListener('input', function () {
            clearTimeout(searchTimer);
            searchTimer = setTimeout(function () {
                const terms = search.value.trim().toLowerCase().split(/\s+/).filter(Boolean);
                list.querySelectorAll('[data-search]').forEach(function (card) {
                    const txt = card.getAttribute('data-search');
                    const match = terms.every(function (t) { return txt.indexOf(t) !== -1; });
                    card.style.display = match ? '' : 'none';
                });
                // Hide a group header when every card under it is filtered out.
                list.querySelectorAll('[data-group]').forEach(function (g) {
                    let anyVisible = false;
                    let n = g.nextElementSibling;
                    while (n && !n.hasAttribute('data-group')) {
                        if (n.style.display !== 'none') { anyVisible = true; break; }
                        n = n.nextElementSibling;
                    }
                    g.style.display = anyVisible ? '' : 'none';
                });
            }, 80);
        });

        return panel;
    }

    // -------------------------------------------------------------------------

    return { init, getEditor, open, setLang, getLang, lockLang };
})();
