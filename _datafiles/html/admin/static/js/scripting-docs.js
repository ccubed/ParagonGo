// scripting-docs.js
// Shared behavior for the /admin/scripting documentation pages.
//
// Adds a persistent, floating JavaScript/Lua language toggle. Every runnable
// code example on a docs page ships both a JavaScript and a Lua variant, each a
//   <pre class="scriptlang-block" data-lang="js|lua">...</pre>
// inside a `.code-wrap`. A class on <html> (scriptlang-js / scriptlang-lua)
// decides which variant is shown; the toggle flips that class and remembers the
// choice in localStorage so it sticks across pages and reloads.
//
// The class is applied during head parsing (before <body> renders) to avoid a
// flash of the wrong language. The floating control and copy handler are wired
// up once the DOM is ready.
(function () {
    'use strict';

    var STORAGE_KEY = 'gomud-scripting-lang';

    function readLang() {
        try {
            var v = window.localStorage.getItem(STORAGE_KEY);
            if (v === 'lua' || v === 'js') return v;
        } catch (e) { /* private mode / disabled storage */ }
        return 'js';
    }

    function writeLang(lang) {
        try { window.localStorage.setItem(STORAGE_KEY, lang); } catch (e) {}
    }

    function applyLang(lang) {
        var root = document.documentElement;
        root.classList.remove('scriptlang-js', 'scriptlang-lua');
        root.classList.add(lang === 'lua' ? 'scriptlang-lua' : 'scriptlang-js');
    }

    // Inline <code> references in prose are written JS-style. In Lua mode:
    //   * a method-call reference ("room.SetTempData()") uses a colon
    //     ("room:SetTempData()"), and
    //   * the literal "null" becomes "nil".
    // Only inline code is touched — the syntax-highlighted example blocks
    // (<pre><code class="language-*">) are handled by the block toggle and are
    // skipped here. Original text is cached so toggling stays exact.
    var METHOD_CALL_RE = /^[A-Za-z_]\w*\.[A-Za-z_]\w*\(/;
    function applyInlineTokens(lang) {
        var isLua = (lang === 'lua');
        var codes = document.querySelectorAll('code');
        for (var i = 0; i < codes.length; i++) {
            var el = codes[i];
            if (el.className.indexOf('language-') !== -1) continue; // example block
            if (el.closest('pre')) continue;                        // any preformatted block
            var base = el.getAttribute('data-code-base');
            if (base === null) {
                base = el.textContent;
                el.setAttribute('data-code-base', base);
            }
            var out = base;
            if (base === 'null') {
                out = isLua ? 'nil' : 'null';
            } else if (/\.(js|lua)$/.test(base)) {
                // A script file extension reference (e.g. ".js" or "heal.js").
                out = base.replace(/\.(js|lua)$/, isLua ? '.lua' : '.js');
            } else if (isLua && METHOD_CALL_RE.test(base)) {
                out = base.replace('.', ':');
            }
            if (el.textContent !== out) el.textContent = out;
        }
    }

    // Script-path callouts (".path-box") spell out on-disk file paths ending in
    // .js, with `<span class="ph">` placeholders inside. Swap every .js/.lua file
    // extension to match the active language. The pristine markup is cached so the
    // placeholders survive repeated toggles.
    function applyScriptPaths(lang) {
        var ext = (lang === 'lua') ? '.lua' : '.js';
        var boxes = document.querySelectorAll('.path-box');
        for (var i = 0; i < boxes.length; i++) {
            var box = boxes[i];
            var base = box.getAttribute('data-path-base');
            if (base === null) {
                base = box.innerHTML;
                box.setAttribute('data-path-base', base);
            }
            box.innerHTML = base.replace(/\.(js|lua)\b/g, ext);
        }
    }

    // Reference-page method titles are written JS-style ("ActorObject.GetLevel").
    // In Lua mode the member separator becomes a colon ("ActorObject:GetLevel"),
    // matching the colon method-call syntax used throughout the Lua examples.
    // Global functions have no separator and are left alone. The original text is
    // cached so toggling back and forth stays exact.
    function applyMethodSeparators(lang) {
        var sep = (lang === 'lua') ? ':' : '.';
        var els = document.querySelectorAll('.fn-n');
        for (var i = 0; i < els.length; i++) {
            var el = els[i];
            var base = el.getAttribute('data-fn-base');
            if (base === null) {
                base = el.textContent;
                el.setAttribute('data-fn-base', base);
            }
            if (base.indexOf('.') !== -1) {
                el.textContent = base.replace('.', sep);
            }
        }
    }

    // Apply immediately so hidden variants never flash before the toggle loads.
    var currentLang = readLang();
    applyLang(currentLang);

    // The show/hide rules and the floating control styling. Injected synchronously
    // (this script is a blocking <head> include) so the rules exist before paint.
    function injectStyles() {
        if (document.getElementById('scriptlang-styles')) return;
        var s = document.createElement('style');
        s.id = 'scriptlang-styles';
        s.textContent = [
            'html.scriptlang-js .scriptlang-block[data-lang="lua"]{display:none;}',
            'html.scriptlang-lua .scriptlang-block[data-lang="js"]{display:none;}',
            '.scriptlang-toggle{',
            '  position:fixed;bottom:1.25rem;right:1.25rem;z-index:1000;',
            '  display:flex;align-items:center;gap:0;',
            '  background:var(--color-surface-white,#fff);border:1px solid var(--color-border,#d0d0d0);',
            '  border-radius:999px;box-shadow:0 4px 16px rgba(0,0,0,0.18);overflow:hidden;',
            '  font-size:0.8rem;font-family:system-ui,sans-serif;',
            '}',
            '.scriptlang-toggle .scriptlang-label{',
            '  padding:0 0.6rem 0 0.85rem;color:var(--color-text-faint,#888);font-weight:600;',
            '  text-transform:uppercase;letter-spacing:0.05em;font-size:0.65rem;',
            '}',
            '.scriptlang-toggle button{',
            '  border:none;background:none;cursor:pointer;padding:0.5rem 0.95rem;',
            '  font-size:0.82rem;font-weight:600;color:var(--color-text-muted,#666);',
            '  font-family:inherit;line-height:1;transition:background 0.12s,color 0.12s;',
            '}',
            '.scriptlang-toggle button:hover{color:var(--color-text,#222);}',
            '.scriptlang-toggle button.active{background:var(--color-primary,#2563eb);color:#fff;}',
            '@media (max-width:860px){.scriptlang-toggle{bottom:0.75rem;right:0.75rem;}}'
        ].join('');
        (document.head || document.documentElement).appendChild(s);
    }
    injectStyles();

    function buildToggle() {
        // Only show the control on pages that actually carry switchable examples.
        if (!document.querySelector('.scriptlang-block')) return;
        if (document.querySelector('.scriptlang-toggle')) return;

        var wrap = document.createElement('div');
        wrap.className = 'scriptlang-toggle';
        wrap.setAttribute('role', 'group');
        wrap.setAttribute('aria-label', 'Code example language');

        var label = document.createElement('span');
        label.className = 'scriptlang-label';
        label.textContent = 'Lang';
        wrap.appendChild(label);

        var buttons = {};
        [['js', 'JavaScript'], ['lua', 'Lua']].forEach(function (opt) {
            var b = document.createElement('button');
            b.type = 'button';
            b.textContent = opt[1];
            b.addEventListener('click', function () { select(opt[0]); });
            wrap.appendChild(b);
            buttons[opt[0]] = b;
        });

        function select(lang) {
            currentLang = (lang === 'lua') ? 'lua' : 'js';
            applyLang(currentLang);
            applyMethodSeparators(currentLang);
            applyInlineTokens(currentLang);
            applyScriptPaths(currentLang);
            writeLang(currentLang);
            buttons.js.classList.toggle('active', currentLang === 'js');
            buttons.lua.classList.toggle('active', currentLang === 'lua');
        }

        select(currentLang);
        document.body.appendChild(wrap);
    }

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', buildToggle);
    } else {
        buildToggle();
    }

    // Copy the currently-visible code variant within a `.code-wrap`. Shared by all
    // docs pages (each page used to define its own identical copyCode).
    window.copyCode = function (btn) {
        var wrap = btn.closest('.code-wrap');
        if (!wrap) return;
        var pres = wrap.querySelectorAll('pre');
        var pre = null;
        for (var i = 0; i < pres.length; i++) {
            // offsetParent is null for display:none elements — pick the visible one.
            if (pres[i].offsetParent !== null) { pre = pres[i]; break; }
        }
        if (!pre) pre = pres[0];
        var text = pre ? pre.textContent : '';
        navigator.clipboard.writeText(text).then(function () {
            btn.textContent = 'Copied!';
            btn.classList.add('copied');
            setTimeout(function () { btn.textContent = 'Copy'; btn.classList.remove('copied'); }, 1800);
        });
    };
})();
