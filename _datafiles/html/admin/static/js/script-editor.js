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

        // ---- iframe mount inside modal ----
        // A plain div that the iframe will be moved into.
        // No overflow:hidden, no border-radius — avoids compositing layer
        // issues that misalign Monaco's pointer-event hit-testing.
        const mount = document.createElement('div');
        mount.style.cssText = 'flex:1;min-height:0;min-width:0;position:relative;';

        modal.appendChild(header);
        modal.appendChild(mount);
        overlay.appendChild(modal);
        document.body.appendChild(overlay);

        // Move the iframe from inline container into the modal mount.
        // Reset its size to fill the modal.
        const iframe = rec.iframe;
        iframe.style.cssText = 'width:100%;height:100%;border:none;display:block;border-radius:0 0 6px 6px;';
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

    return { init, getEditor, open, setLang, getLang, lockLang };
})();
