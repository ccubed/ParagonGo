// ScriptWizard - modal dialog for selecting script event handler stubs and
// inserting them into the Monaco editor managed by ScriptEditor.
//
// Usage:
//   ScriptWizard.open({ scriptType: 'room', textareaId: 'f-script' });

const ScriptWizard = (() => {
    'use strict';

    let overlay = null;
    let schemaCache = null;

    function injectStyles() {
        if (document.getElementById('script-wizard-styles')) return;
        const s = document.createElement('style');
        s.id = 'script-wizard-styles';
        s.textContent = `
            .sw-overlay {
                position: fixed; inset: 0; background: var(--color-overlay);
                display: flex; align-items: center; justify-content: center;
                z-index: 9999;
            }
            .sw-modal {
                background: var(--color-surface); border-radius: 8px; box-shadow: 0 8px 32px var(--color-shadow);
                width: 640px; max-width: 95vw; max-height: 85vh;
                display: flex; flex-direction: column; overflow: hidden;
            }
            .sw-header {
                padding: 0.85rem 1rem 0.7rem; border-bottom: 1px solid var(--color-border);
                display: flex; align-items: center; justify-content: space-between;
            }
            .sw-title { font-size: 1rem; font-weight: 700; color: var(--color-text); }
            .sw-close {
                background: none; border: none; font-size: 1.25rem; cursor: pointer;
                color: var(--color-text-faint); line-height: 1; padding: 0 0.2rem;
            }
            .sw-close:hover { color: var(--color-text); }
            .sw-body { overflow-y: auto; flex: 1; padding: 0; }
            .sw-fn-list { list-style: none; margin: 0; padding: 0.35rem 0; }
            .sw-fn-item {
                padding: 0.55rem 1rem; cursor: pointer; font-size: 0.875rem;
                border-bottom: 1px solid var(--color-border-light);
                color: var(--color-text);
            }
            .sw-fn-item:last-child { border-bottom: none; }
            .sw-fn-item:hover { background: var(--color-row-hover); }
            .sw-fn-item.sw-selected { background: var(--color-active-bg); }
            .sw-fn-name { font-family: monospace; font-weight: 700; font-size: 0.9rem; color: var(--color-code-keyword); }
            .sw-fn-desc { font-size: 0.8rem; color: var(--color-text-secondary); margin-top: 0.2rem; }

            .sw-detail { padding: 1rem; display: none; flex-direction: column; gap: 0.75rem; }
            .sw-detail.sw-active { display: flex; }
            .sw-detail-title { font-family: monospace; font-weight: 700; font-size: 1rem; color: var(--color-code-keyword); }
            .sw-detail-desc { font-size: 0.85rem; color: var(--color-text); line-height: 1.5; }
            .sw-params-title { font-size: 0.8rem; font-weight: 700; color: var(--color-text-secondary); text-transform: uppercase; letter-spacing: 0.05em; }
            .sw-param-row { display: flex; gap: 0.5rem; align-items: baseline; font-size: 0.82rem; padding: 0.15rem 0; }
            .sw-param-name { font-family: monospace; font-weight: 600; color: var(--color-code-symbol); min-width: 100px; }
            .sw-param-type { font-family: monospace; color: var(--color-api-flag); min-width: 90px; }
            .sw-param-desc { color: var(--color-text-secondary); }
            .sw-return { font-size: 0.82rem; color: var(--color-text-secondary); font-style: italic; }
            .sw-dynamic-input {
                display: flex; align-items: center; gap: 0.5rem; margin-top: 0.25rem;
            }
            .sw-dynamic-input label { font-size: 0.82rem; font-weight: 600; color: var(--color-text); white-space: nowrap; }
            .sw-dynamic-input input {
                padding: 0.3rem 0.5rem; border: 1px solid var(--color-border-medium);
                border-radius: 4px; font-size: 0.82rem; font-family: monospace;
                background: var(--color-surface-raised); color: var(--color-text);
                width: 180px;
            }
            .sw-dynamic-input input:focus { outline: 2px solid var(--color-focus); outline-offset: 1px; border-color: transparent; }
            .sw-dynamic-hint { font-size: 0.75rem; color: var(--color-text-faint); }

            .sw-variants { font-size: 0.8rem; margin-top: 0.25rem; }
            .sw-variants summary { cursor: pointer; color: var(--color-text-secondary); font-weight: 600; }
            .sw-variant-row { display: flex; gap: 0.5rem; align-items: baseline; padding: 0.15rem 0 0.15rem 1rem; }
            .sw-variant-key { font-family: monospace; font-weight: 600; color: var(--color-api-kw); min-width: 80px; }
            .sw-variant-type { font-family: monospace; color: var(--color-api-flag); min-width: 90px; }
            .sw-variant-desc { color: var(--color-text-secondary); }

            .sw-footer {
                padding: 0.65rem 1rem; border-top: 1px solid var(--color-border-light);
                display: flex; justify-content: flex-end; gap: 0.5rem;
                background: var(--color-surface);
            }
            .sw-btn {
                padding: 0.4rem 1rem; border: none; border-radius: 4px; cursor: pointer;
                font-size: 0.82rem; font-weight: 600;
            }
            .sw-btn-primary { background: var(--color-primary); color: var(--color-surface-white); }
            .sw-btn-primary:hover { background: var(--color-primary-hover); }
            .sw-btn-primary:disabled { background: var(--color-text-secondary); cursor: default; }
            .sw-btn-cancel { background: var(--color-border-light); color: var(--color-text-strong); }
            .sw-btn-cancel:hover { background: var(--color-border); }
            .sw-loading { padding: 2rem 1rem; text-align: center; color: var(--color-text-faint); font-size: 0.85rem; }
            .sw-back {
                background: none; border: none; font-size: 0.82rem; cursor: pointer;
                color: var(--color-accent-link); padding: 0; margin-bottom: 0.25rem;
            }
            .sw-back:hover { text-decoration: underline; }
        `;
        document.head.appendChild(s);
    }

    async function fetchSchema() {
        if (schemaCache) return schemaCache;
        const res = await AdminAPI.get('/admin/api/v1/scripting/functions');
        if (!res.ok) throw new Error('Failed to load script schema: ' + res.error);
        schemaCache = res.data.data;
        return schemaCache;
    }

    function close() {
        if (overlay) { overlay.remove(); overlay = null; }
    }

    function open(opts) {
        const { scriptType, textareaId } = opts;
        injectStyles();
        close();

        overlay = document.createElement('div');
        overlay.className = 'sw-overlay';
        overlay.addEventListener('mousedown', (e) => { if (e.target === overlay) close(); });

        const modal = document.createElement('div');
        modal.className = 'sw-modal';
        modal.innerHTML = '<div class="sw-loading">Loading script functions...</div>';
        overlay.appendChild(modal);
        document.body.appendChild(overlay);

        fetchSchema().then((schema) => {
            const typeDef = schema.scriptTypes[scriptType];
            if (!typeDef) {
                modal.innerHTML = '<div class="sw-loading">No functions found for script type: ' + scriptType + '</div>';
                return;
            }
            renderModal(modal, typeDef, textareaId);
        }).catch((err) => {
            modal.innerHTML = '<div class="sw-loading">Error: ' + err.message + '</div>';
        });
    }

    function renderModal(modal, typeDef, textareaId) {
        modal.innerHTML = '';

        const header = document.createElement('div');
        header.className = 'sw-header';
        header.innerHTML = '<span class="sw-title">Add Event Handler</span>';
        const closeBtn = document.createElement('button');
        closeBtn.className = 'sw-close';
        closeBtn.textContent = '×';
        closeBtn.onclick = close;
        header.appendChild(closeBtn);
        modal.appendChild(header);

        const body = document.createElement('div');
        body.className = 'sw-body';
        modal.appendChild(body);

        const footer = document.createElement('div');
        footer.className = 'sw-footer';
        const cancelBtn = document.createElement('button');
        cancelBtn.className = 'sw-btn sw-btn-cancel';
        cancelBtn.textContent = 'Cancel';
        cancelBtn.onclick = close;
        const insertBtn = document.createElement('button');
        insertBtn.className = 'sw-btn sw-btn-primary';
        insertBtn.textContent = 'Insert';
        insertBtn.disabled = true;
        footer.appendChild(cancelBtn);
        footer.appendChild(insertBtn);
        modal.appendChild(footer);

        let selectedFn = null;
        let dynamicValue = '';

        function showList() {
            body.innerHTML = '';
            selectedFn = null;
            insertBtn.disabled = true;

            const ul = document.createElement('ul');
            ul.className = 'sw-fn-list';

            typeDef.functions.forEach((fn) => {
                const li = document.createElement('li');
                li.className = 'sw-fn-item';
                li.innerHTML =
                    '<div class="sw-fn-name">' + escHtml(fn.name) + '</div>' +
                    '<div class="sw-fn-desc">' + escHtml(fn.description) + '</div>';
                li.addEventListener('click', () => showDetail(fn));
                ul.appendChild(li);
            });
            body.appendChild(ul);
        }

        function showDetail(fn) {
            selectedFn = fn;
            dynamicValue = '';
            body.innerHTML = '';
            insertBtn.disabled = fn.dynamic ? true : false;

            const detail = document.createElement('div');
            detail.className = 'sw-detail sw-active';

            const backBtn = document.createElement('button');
            backBtn.className = 'sw-back';
            backBtn.textContent = '← Back to list';
            backBtn.onclick = showList;
            detail.appendChild(backBtn);

            const title = document.createElement('div');
            title.className = 'sw-detail-title';
            title.textContent = fn.name;
            detail.appendChild(title);

            const desc = document.createElement('div');
            desc.className = 'sw-detail-desc';
            desc.textContent = fn.description;
            detail.appendChild(desc);

            if (fn.dynamic) {
                const dynWrap = document.createElement('div');
                dynWrap.className = 'sw-dynamic-input';
                const lbl = document.createElement('label');
                lbl.textContent = fn.dynamic.label + ':';
                const inp = document.createElement('input');
                inp.type = fn.dynamic.inputType || 'text';
                inp.placeholder = 'e.g. pull, push';
                inp.addEventListener('input', () => {
                    dynamicValue = inp.value.trim().toLowerCase().replace(/[^a-z0-9_]/g, '');
                    inp.value = dynamicValue;
                    insertBtn.disabled = dynamicValue.length === 0;
                    title.textContent = fn.name.replace(fn.dynamic.placeholder, dynamicValue || fn.dynamic.placeholder);
                });
                dynWrap.appendChild(lbl);
                dynWrap.appendChild(inp);
                detail.appendChild(dynWrap);

                const hint = document.createElement('div');
                hint.className = 'sw-dynamic-hint';
                hint.textContent = fn.dynamic.description;
                detail.appendChild(hint);

                setTimeout(() => inp.focus(), 50);
            }

            if (fn.params && fn.params.length > 0) {
                const pTitle = document.createElement('div');
                pTitle.className = 'sw-params-title';
                pTitle.textContent = 'Parameters';
                detail.appendChild(pTitle);

                fn.params.forEach((p) => {
                    const row = document.createElement('div');
                    row.className = 'sw-param-row';
                    row.innerHTML =
                        '<span class="sw-param-name">' + escHtml(p.name) + '</span>' +
                        '<span class="sw-param-type">' + escHtml(p.type) + '</span>' +
                        '<span class="sw-param-desc">' + escHtml(p.description) + '</span>';
                    detail.appendChild(row);

                    if (p.typeVariants) {
                        const variants = document.createElement('details');
                        variants.className = 'sw-variants';
                        const summary = document.createElement('summary');
                        summary.textContent = 'Type varies by spell type';
                        variants.appendChild(summary);

                        Object.keys(p.typeVariants).forEach((key) => {
                            const v = p.typeVariants[key];
                            const vRow = document.createElement('div');
                            vRow.className = 'sw-variant-row';
                            vRow.innerHTML =
                                '<span class="sw-variant-key">' + escHtml(key) + '</span>' +
                                '<span class="sw-variant-type">' + escHtml(v.type) + '</span>' +
                                '<span class="sw-variant-desc">' + escHtml(v.description) + '</span>';
                            variants.appendChild(vRow);
                        });
                        detail.appendChild(variants);
                    }
                });
            }

            if (fn.returnSemantics) {
                const ret = document.createElement('div');
                ret.className = 'sw-return';
                ret.textContent = fn.returnSemantics;
                detail.appendChild(ret);
            }

            body.appendChild(detail);
        }

        insertBtn.addEventListener('click', () => {
            if (!selectedFn) return;
            let fnName = selectedFn.name;
            if (selectedFn.dynamic && dynamicValue) {
                fnName = fnName.split(selectedFn.dynamic.placeholder).join(dynamicValue);
            }
            const lang = (typeof ScriptEditor !== 'undefined' && ScriptEditor.getLang)
                ? ScriptEditor.getLang(textareaId) : 'js';
            const stub = lang === 'lua'
                ? buildLuaStub(selectedFn, dynamicValue)
                : buildStub(selectedFn, dynamicValue);
            const iframeWin = ScriptEditor.getEditor(textareaId);
            if (iframeWin) {
                // Modal is open: ask the iframe to jump-to or insert
                iframeWin.postMessage({ type: 'monaco-jump-or-insert', fnName: fnName, stub: stub }, '*');
                close();
            } else {
                // Modal not open: check textarea for existing function, then open
                const ta = document.getElementById(textareaId);
                if (ta) {
                    const escaped = fnName.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
                    const exists = new RegExp('(^|\n)[ \t]*function\\s+' + escaped + '\\s*\\(').test(ta.value);
                    if (!exists) {
                        let current = ta.value;
                        if (current.length > 0 && !current.endsWith('\n')) current += '\n';
                        if (current.length > 0 && !current.endsWith('\n\n')) current += '\n';
                        ta.value = current + stub;
                    }
                }
                ScriptEditor.open(textareaId);
                close();
            }
        });

        showList();
    }

    function buildStub(fn, dynValue) {
        const lines = [];

        // JSDoc block - Monaco's JS type checker reads @param tags to infer
        // parameter types inside the function body, enabling member completions.
        lines.push('/**');
        lines.push(' * ' + fn.description);
        if (fn.params && fn.params.length > 0) {
            fn.params.forEach(function (p) {
                // Strip optional/variadic markers from the param name for JSDoc
                const paramName = p.name.replace(/^\.\.\./, '').replace(/\?$/, '');
                lines.push(' * @param {' + paramTypeString(p) + '} ' + paramName + ' ' + p.description);
            });
        }
        if (fn.returnType && fn.returnType !== 'void') {
            lines.push(' * @returns {' + fn.returnType + '} ' + (fn.returnSemantics || ''));
        } else if (fn.returnSemantics && fn.returnSemantics !== 'Return value is ignored.') {
            lines.push(' * ' + fn.returnSemantics);
        }
        lines.push(' */');

        let code = fn.stub;
        if (fn.dynamic && dynValue) {
            code = code.split(fn.dynamic.placeholder).join(dynValue);
        }
        return lines.join('\n') + '\n' + code;
    }

    // Resolve a parameter's declared type, collapsing typeVariants into a union.
    function paramTypeString(p) {
        if (!p.typeVariants) return p.type;
        const seen = {};
        const types = [];
        Object.keys(p.typeVariants).forEach(function (k) {
            const t = p.typeVariants[k].type;
            if (!seen[t]) { seen[t] = true; types.push(t); }
        });
        return types.join('|');
    }

    // Build a Lua event-handler stub. The JavaScript schema is the single source
    // of truth, so the signature (parameter names/order) and any return value are
    // derived from fn.stub, while the doc block is rebuilt as Lua comments.
    function buildLuaStub(fn, dynValue) {
        let code = fn.stub;
        if (fn.dynamic && dynValue) {
            code = code.split(fn.dynamic.placeholder).join(dynValue);
        }

        // Pull "function NAME(args)" and any "return <expr>;" out of the JS stub.
        const sig = /function\s+([A-Za-z0-9_]+)\s*\(([^)]*)\)/.exec(code);
        const fnName = sig ? sig[1] : fn.name;
        const params = sig ? sig[2].trim() : '';
        const ret = /return\s+([^;]+);/.exec(code);

        const lines = [];
        lines.push('-- ' + fn.description);
        if (fn.params && fn.params.length > 0) {
            fn.params.forEach(function (p) {
                const paramName = p.name.replace(/^\.\.\./, '').replace(/\?$/, '');
                lines.push('-- @param ' + paramName + ' (' + paramTypeString(p) + ') ' + p.description);
            });
        }
        if (fn.returnType && fn.returnType !== 'void') {
            lines.push('-- @return (' + fn.returnType + ') ' + (fn.returnSemantics || ''));
        } else if (fn.returnSemantics && fn.returnSemantics !== 'Return value is ignored.') {
            lines.push('-- ' + fn.returnSemantics);
        }

        lines.push('function ' + fnName + '(' + params + ')');
        lines.push('');
        if (ret) {
            lines.push('    return ' + ret[1].trim());
        }
        lines.push('end');
        return lines.join('\n') + '\n';
    }

    function escHtml(str) {
        const d = document.createElement('div');
        d.textContent = str;
        return d.innerHTML;
    }

    return { open, close };
})();
