// monaco-editor-frame.js
// Runs inside the isolated iframe. Receives configuration and content from
// the parent page via postMessage, hosts the Monaco editor, and posts changes
// back to the parent.

(function () {
    'use strict';

    var editor = null;
    var monacoBase = null;
    var scriptType = null;
    var currentLang = 'javascript';
    var intellisenseApplied = false;
    var luaProvidersRegistered = false;
    var pendingMessages = [];

    // Map the language identifier sent by the parent ('js' / 'lua') to the
    // Monaco language id. Defaults to javascript for unknown values.
    function monacoLang(lang) {
        return lang === 'lua' ? 'lua' : 'javascript';
    }

    // -------------------------------------------------------------------------
    // Bootstrap: wait for the parent to send config before loading Monaco
    // -------------------------------------------------------------------------

    window.addEventListener('message', function (e) {
        if (!e.data || !e.data.type) return;

        if (e.data.type === 'monaco-init') {
            monacoBase = e.data.monacoBase;
            scriptType = e.data.scriptType || '';
            currentLang = monacoLang(e.data.lang);
            loadMonaco(e.data.initialValue || '');
            return;
        }

        // Queue messages that arrive before the editor is ready
        if (!editor) {
            pendingMessages.push(e.data);
            return;
        }

        handleMessage(e.data);
    });

    function handleMessage(msg) {
        if (msg.type === 'monaco-set') {
            if (msg.lang) {
                applyLanguage(monacoLang(msg.lang));
            }
            if (editor.getValue() !== msg.value) {
                editor.setValue(msg.value);
            }

        } else if (msg.type === 'monaco-get') {
            parent.postMessage({ type: 'monaco-value', value: editor.getValue() }, '*');

        } else if (msg.type === 'monaco-layout') {
            editor.layout();

        } else if (msg.type === 'monaco-insert') {
            insertAtEnd(msg.stub);

        } else if (msg.type === 'monaco-jump-or-insert') {
            var model = editor.getModel();
            var text = model.getValue();
            var escaped = msg.fnName.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
            var m = new RegExp('(^|\n)[ \t]*function\\s+' + escaped + '\\s*\\(').exec(text);
            if (m) {
                var lineNum = text.substring(0, m.index + m[0].indexOf('function')).split('\n').length;
                editor.revealLineInCenter(lineNum);
                editor.setPosition({ lineNumber: lineNum, column: 1 });
                editor.focus();
            } else {
                insertAtEnd(msg.stub);
            }
        }
    }

    function insertAtEnd(stub) {
        var model = editor.getModel();
        var lineCount = model.getLineCount();
        var lastLine = model.getLineContent(lineCount);
        var prefix = (lastLine.trim().length > 0 ? '\n' : '') + '\n';
        editor.executeEdits('script-wizard', [{
            range: new monaco.Range(lineCount, lastLine.length + 1, lineCount, lastLine.length + 1),
            text: prefix + stub,
            forceMoveMarkers: true
        }]);
        editor.revealLine(model.getLineCount());
        editor.focus();
    }

    // -------------------------------------------------------------------------
    // Monaco loading
    // -------------------------------------------------------------------------

    function loadMonaco(initialValue) {
        var loaderScript = document.createElement('script');
        loaderScript.src = monacoBase + '/loader.js';
        loaderScript.onload = function () {
            require.config({
                paths: { vs: monacoBase },
                'vs/css': { disabled: true }
            });
            window.MonacoEnvironment = {
                getWorkerUrl: function () {
                    return window.location.origin + '/admin/static/js/monaco/vs/base/worker/workerMain.js';
                }
            };
            require(['vs/editor/editor.main', 'vs/basic-languages/lua/lua'], function (editorMain, luaLang) {
                registerLuaLanguage(luaLang);
                createEditor(initialValue);
            });
        };
        document.head.appendChild(loaderScript);
    }

    // Register the Lua language using the grammar supplied by the vendored
    // basic-language module (vs/basic-languages/lua/lua). The vendored Monaco
    // distribution has no basic-languages contribution manifest, so the module
    // does not self-register; we register it explicitly here. Registration is
    // idempotent and a no-op if Monaco already knows the language.
    function registerLuaLanguage(luaLang) {
        var langs = monaco.languages.getLanguages();
        for (var i = 0; i < langs.length; i++) {
            if (langs[i].id === 'lua') return;
        }
        if (!luaLang || !luaLang.language) return;

        monaco.languages.register({ id: 'lua', extensions: ['.lua'], aliases: ['Lua', 'lua'] });
        if (luaLang.conf) {
            monaco.languages.setLanguageConfiguration('lua', luaLang.conf);
        }
        monaco.languages.setMonarchTokensProvider('lua', luaLang.language);
    }

    function createEditor(initialValue) {
        editor = monaco.editor.create(document.getElementById('editor'), {
            value: initialValue,
            language: currentLang,
            theme: 'vs-dark',
            automaticLayout: true,
            minimap: { enabled: false },
            fontSize: 13,
            lineNumbers: 'on',
            scrollBeyondLastLine: false,
            tabSize: 4,
            insertSpaces: true,
            wordWrap: 'off',
            renderWhitespace: 'none',
            folding: true,
            glyphMargin: false,
            overviewRulerLanes: 0,
            fixedOverflowWidgets: true
        });

        // Register a custom action so "Add Event Handler" appears in the
        // right-click context menu and F1 Command Palette.
        editor.addAction({
            id: 'gomud-add-event-handler',
            label: 'Add Event Handler',
            contextMenuGroupId: 'navigation',
            contextMenuOrder: 1,
            run: function () {
                parent.postMessage({ type: 'monaco-open-wizard' }, '*');
            }
        });

        // Notify parent of content changes
        editor.onDidChangeModelContent(function () {
            parent.postMessage({ type: 'monaco-change', value: editor.getValue() }, '*');
        });

        // Apply intellisense for the active language.
        if (currentLang === 'lua') {
            applyLuaIntellisense();
        } else if (currentLang === 'javascript' && scriptType) {
            applyIntellisense(scriptType);
        }

        // Drain any messages that arrived before the editor was ready
        pendingMessages.forEach(function (msg) { handleMessage(msg); });
        pendingMessages = [];

        editor.focus();
        parent.postMessage({ type: 'monaco-ready' }, '*');
    }

    // Switch the editor's language at runtime (e.g. when the parent loads a
    // different script). Updates the model language and lazily wires up the
    // intellisense providers appropriate to the selected language.
    function applyLanguage(lang) {
        if (lang === currentLang) return;
        currentLang = lang;

        var model = editor.getModel();
        if (model) {
            monaco.editor.setModelLanguage(model, currentLang);
        }

        if (currentLang === 'lua') {
            applyLuaIntellisense();
        } else if (currentLang === 'javascript' && scriptType && !intellisenseApplied) {
            applyIntellisense(scriptType);
        }
    }

    function applyIntellisense(type) {
        intellisenseApplied = true;
        // Configure the JavaScript language service for a clean engine-only
        // environment. We want full semantic analysis (for hover, signature help,
        // and go-to-definition) but without browser/DOM globals polluting
        // completions. Setting lib:[] gives an empty standard lib; our .d.ts
        // then provides the only globals Monaco knows about.
        monaco.languages.typescript.javascriptDefaults.setCompilerOptions({
            allowNonTsExtensions: true,
            allowJs: true,
            checkJs: true,
            target: monaco.languages.typescript.ScriptTarget.ES2015,
            lib: []
        });
        // Keep all diagnostics on so hover info and signature help work.
        // Only suppress semantic errors that would fire on unknown globals
        // (the engine injects many globals at runtime that TypeScript can't see).
        monaco.languages.typescript.javascriptDefaults.setDiagnosticsOptions({
            noSemanticValidation: false,
            noSyntaxValidation: false,
            diagnosticCodesToIgnore: [2304, 2339, 2349, 2540]
        });
        fetch('/admin/api/v1/scripting/types.d.ts?type=' + encodeURIComponent(type), { credentials: 'include' })
            .then(function (r) { return r.text(); })
            .then(function (dts) {
                monaco.languages.typescript.javascriptDefaults.addExtraLib(dts, 'file:///gomud-engine.d.ts');
            })
            .catch(function () {});
    }

    // Lua intellisense is type-aware. Monaco has no Lua language service, so we
    // build a lightweight type model from the engine schema and object-type
    // definitions (the same data that drives the JavaScript .d.ts) and register
    // completion + hover providers that resolve the type of the value before
    // the cursor. This means obj:Method() completions show only that object's
    // methods, instead of dumping every engine function on every variable.
    function applyLuaIntellisense() {
        if (luaProvidersRegistered) return;
        luaProvidersRegistered = true;

        Promise.all([
            fetch('/admin/api/v1/scripting/functions', { credentials: 'include' }).then(function (r) { return r.json(); }),
            fetch('/admin/api/v1/scripting/objecttypes', { credentials: 'include' }).then(function (r) { return r.json(); })
        ]).then(function (results) {
            var schema = results[0] && results[0].data;
            var objTypes = results[1] && results[1].data;
            if (!schema || !objTypes) return;
            registerLuaProviders(buildLuaModel(schema, objTypes));
        }).catch(function () {});
    }

    function luaSignature(name, params) {
        var parts = (params || []).map(function (p) {
            var nm = p.name.replace(/^\.\.\./, '').replace(/\?$/, '');
            return nm;
        });
        return name + '(' + parts.join(', ') + ')';
    }

    function luaDoc(fn) {
        var lines = [];
        if (fn.description) lines.push(fn.description);
        (fn.params || []).forEach(function (p) {
            lines.push('@param `' + p.name + '` *' + (p.type || 'any') + '* ' + (p.description || ''));
        });
        if (fn.returnType && fn.returnType !== 'void') {
            lines.push('@return *' + fn.returnType + '* ' + (fn.returnSemantics || ''));
        }
        return lines.join('\n\n');
    }

    // Normalize a schema/object return type to a base object-type name, or null
    // if it is not one of the known engine object types. Strips array suffixes
    // and union noise so e.g. "ItemObject[]" and "ActorObject | null" resolve.
    function baseObjectType(t, knownTypes) {
        if (!t) return null;
        var cleaned = t.replace(/\[\]/g, '').split('|')[0].trim();
        return knownTypes[cleaned] ? cleaned : null;
    }

    // Build the full type model used by the providers.
    function buildLuaModel(schema, objTypes) {
        var types = objTypes.types || {};

        // globals: name -> {signature, doc, params, returnType}
        var globals = {};
        (schema.engineFunctions || []).forEach(function (fn) {
            globals[fn.name] = {
                signature: luaSignature(fn.name, fn.params),
                doc: luaDoc(fn),
                params: fn.params || [],
                returnType: fn.returnType || ''
            };
        });

        // methodsByType: typeName -> { methodName -> {signature, doc, params, returnType} }
        var methodsByType = {};
        Object.keys(types).forEach(function (typeName) {
            var def = types[typeName];
            var methods = {};
            (def.methods || []).forEach(function (mt) {
                methods[mt.name] = {
                    signature: luaSignature(mt.name, mt.params),
                    doc: luaDoc(mt),
                    params: mt.params || [],
                    returnType: mt.returnType || ''
                };
            });
            methodsByType[typeName] = methods;
        });

        // entrypointVars: variable name -> object type, seeded from the current
        // script type's event-handler parameters (e.g. mob -> ActorObject).
        var entrypointVars = {};
        var typeDef = schema.scriptTypes && scriptType && schema.scriptTypes[scriptType];
        if (typeDef && typeDef.functions) {
            typeDef.functions.forEach(function (fn) {
                (fn.params || []).forEach(function (pr) {
                    var nm = pr.name.replace(/^\.\.\./, '').replace(/\?$/, '');
                    var bt = baseObjectType(pr.type, types);
                    if (bt && !entrypointVars[nm]) {
                        entrypointVars[nm] = bt;
                    }
                });
            });
        }

        return {
            types: types,
            globals: globals,
            methodsByType: methodsByType,
            entrypointVars: entrypointVars
        };
    }

    // Scan the document for `local <name> = ...` assignments and infer the
    // object type of <name> from the right-hand side: a global call
    // (GetRoom(...) -> RoomObject) or a typed method call (room:GetMob(...) ->
    // ActorObject). Returns a map of variable name -> object type, layered on
    // top of the entrypoint variables.
    function inferLocalTypes(model, upToLine, mdl) {
        var vars = {};
        Object.keys(mdl.entrypointVars).forEach(function (k) { vars[k] = mdl.entrypointVars[k]; });

        var lineCount = Math.min(upToLine, model.getLineCount());
        var globalCallRe = /\blocal\s+([A-Za-z_]\w*)\s*=\s*([A-Za-z_]\w*)\s*\(/;
        var methodCallRe = /\blocal\s+([A-Za-z_]\w*)\s*=\s*([A-Za-z_]\w*)\s*:\s*([A-Za-z_]\w*)\s*\(/;

        for (var ln = 1; ln <= lineCount; ln++) {
            var text = model.getLineContent(ln);

            var mm = methodCallRe.exec(text);
            if (mm) {
                var recvType = vars[mm[2]];
                if (recvType && mdl.methodsByType[recvType] && mdl.methodsByType[recvType][mm[3]]) {
                    var rt = baseObjectType(mdl.methodsByType[recvType][mm[3]].returnType, mdl.types);
                    if (rt) vars[mm[1]] = rt;
                }
                continue;
            }

            var gm = globalCallRe.exec(text);
            if (gm && mdl.globals[gm[2]]) {
                var grt = baseObjectType(mdl.globals[gm[2]].returnType, mdl.types);
                if (grt) vars[gm[1]] = grt;
            }
        }
        return vars;
    }

    // Build a tab-stop snippet body: Name(${1:p1}, ${2:p2})
    function luaSnippet(name, params) {
        var idx = 0;
        var ph = [];
        (params || []).forEach(function (p) {
            var nm = p.name.replace(/^\.\.\./, '').replace(/\?$/, '');
            idx++;
            ph.push('${' + idx + ':' + nm + '}');
        });
        return name + '(' + ph.join(', ') + ')';
    }

    var LUA_KEYWORDS = [
        'and', 'break', 'do', 'else', 'elseif', 'end', 'false', 'for',
        'function', 'goto', 'if', 'in', 'local', 'nil', 'not', 'or',
        'repeat', 'return', 'then', 'true', 'until', 'while'
    ];

    function registerLuaProviders(mdl) {
        function completionRange(word, position) {
            return {
                startLineNumber: position.lineNumber,
                endLineNumber: position.lineNumber,
                startColumn: word.startColumn,
                endColumn: word.endColumn
            };
        }

        function methodItems(typeName, range) {
            var methods = mdl.methodsByType[typeName] || {};
            return Object.keys(methods).map(function (name) {
                var mt = methods[name];
                return {
                    label: mt.signature,
                    kind: monaco.languages.CompletionItemKind.Method,
                    insertText: luaSnippet(name, mt.params),
                    insertTextRules: monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
                    documentation: { value: mt.doc },
                    detail: typeName,
                    filterText: name,
                    range: range
                };
            });
        }

        function globalItems(range) {
            var items = Object.keys(mdl.globals).map(function (name) {
                var g = mdl.globals[name];
                return {
                    label: g.signature,
                    kind: monaco.languages.CompletionItemKind.Function,
                    insertText: luaSnippet(name, g.params),
                    insertTextRules: monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
                    documentation: { value: g.doc },
                    detail: 'engine function',
                    filterText: name,
                    range: range
                };
            });
            return items;
        }

        function variableItems(vars, range) {
            return Object.keys(vars).map(function (name) {
                return {
                    label: name,
                    kind: monaco.languages.CompletionItemKind.Variable,
                    insertText: name,
                    detail: vars[name],
                    filterText: name,
                    range: range
                };
            });
        }

        function keywordItems(range) {
            return LUA_KEYWORDS.map(function (kw) {
                return {
                    label: kw,
                    kind: monaco.languages.CompletionItemKind.Keyword,
                    insertText: kw,
                    filterText: kw,
                    range: range
                };
            });
        }

        monaco.languages.registerCompletionItemProvider('lua', {
            triggerCharacters: ['.', ':'],
            provideCompletionItems: function (model, position) {
                var word = model.getWordUntilPosition(position);
                var range = completionRange(word, position);

                // Text on the current line up to the cursor.
                var prefix = model.getValueInRange({
                    startLineNumber: position.lineNumber,
                    startColumn: 1,
                    endLineNumber: position.lineNumber,
                    endColumn: position.column
                });

                // Member access: <receiver>:<partial> or <receiver>.<partial>
                var memberMatch = /([A-Za-z_]\w*)\s*[:.]\s*\w*$/.exec(prefix);
                if (memberMatch) {
                    var vars = inferLocalTypes(model, position.lineNumber, mdl);
                    var recvType = vars[memberMatch[1]];
                    if (recvType) {
                        return { suggestions: methodItems(recvType, range) };
                    }
                    // Unknown receiver: offer no noisy global dump for member access.
                    return { suggestions: [] };
                }

                // Top level: typed variables, engine globals, and keywords.
                var localVars = inferLocalTypes(model, position.lineNumber, mdl);
                var suggestions = variableItems(localVars, range)
                    .concat(globalItems(range))
                    .concat(keywordItems(range));
                return { suggestions: suggestions };
            }
        });

        monaco.languages.registerHoverProvider('lua', {
            provideHover: function (model, position) {
                var word = model.getWordAtPosition(position);
                if (!word) return null;

                var lineToWord = model.getValueInRange({
                    startLineNumber: position.lineNumber,
                    startColumn: 1,
                    endLineNumber: position.lineNumber,
                    endColumn: word.startColumn
                });

                var hoverRange = new monaco.Range(
                    position.lineNumber, word.startColumn,
                    position.lineNumber, word.endColumn
                );

                // Member access hover: resolve the receiver's type.
                var memberMatch = /([A-Za-z_]\w*)\s*[:.]\s*$/.exec(lineToWord);
                if (memberMatch) {
                    var vars = inferLocalTypes(model, position.lineNumber, mdl);
                    var recvType = vars[memberMatch[1]];
                    if (recvType && mdl.methodsByType[recvType] && mdl.methodsByType[recvType][word.word]) {
                        var mt = mdl.methodsByType[recvType][word.word];
                        return {
                            range: hoverRange,
                            contents: [
                                { value: '```lua\n' + recvType + ':' + mt.signature + '\n```' },
                                { value: mt.doc }
                            ]
                        };
                    }
                    return null;
                }

                // Global function hover.
                if (mdl.globals[word.word]) {
                    var g = mdl.globals[word.word];
                    return {
                        range: hoverRange,
                        contents: [
                            { value: '```lua\n' + g.signature + '\n```' },
                            { value: g.doc }
                        ]
                    };
                }

                // Typed variable hover.
                var allVars = inferLocalTypes(model, position.lineNumber, mdl);
                if (allVars[word.word]) {
                    return {
                        range: hoverRange,
                        contents: [
                            { value: '```lua\n' + word.word + ' : ' + allVars[word.word] + '\n```' }
                        ]
                    };
                }

                return null;
            }
        });
    }

})();
