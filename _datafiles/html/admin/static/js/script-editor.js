// ScriptEditor - wraps a <textarea> with a highlight.js overlay for live
// syntax highlighting. Requires highlight.js to be loaded globally (window.hljs).
//
// Usage:
//   const sync = ScriptEditor.init('f-script');
//   // Call sync() after programmatically changing the textarea's value:
//   document.getElementById('f-script').value = newCode;
//   sync();

const ScriptEditor = (() => {
    'use strict';

    let stylesInjected = false;

    function injectStyles() {
        if (stylesInjected) return;
        stylesInjected = true;
        const s = document.createElement('style');
        s.textContent = `
            .script-editor { position: relative; }
            .script-editor pre {
                position: absolute; top: 0; left: 0; right: 0; bottom: 0;
                margin: 0; pointer-events: none; overflow: hidden;
                border: 1px solid transparent; border-radius: 4px;
                padding: 0.6rem 0.75rem;
                font-family: monospace; font-size: 0.82rem; line-height: 1.55;
            }
            .script-editor pre code.hljs {
                padding: 0; background: transparent;
                font-family: inherit; font-size: inherit; line-height: inherit;
                display: block; white-space: pre;
                overflow: visible;
            }
            .script-editor textarea {
                -webkit-text-fill-color: transparent;
                color: transparent !important;
                caret-color: var(--color-text);
                white-space: pre; overflow-x: auto;
            }
            .script-editor textarea::placeholder {
                -webkit-text-fill-color: var(--color-text-subtle);
                color: var(--color-text-subtle);
            }
            /* Dark syntax colors (GitHub-dark inspired) */
            .script-editor .hljs { background: transparent; color: var(--color-code-text); }
            .script-editor .hljs-comment,
            .script-editor .hljs-quote { color: var(--color-code-comment); font-style: italic; }
            .script-editor .hljs-keyword,
            .script-editor .hljs-selector-tag { color: var(--color-code-keyword); font-weight: 700; }
            .script-editor .hljs-string,
            .script-editor .hljs-template-tag,
            .script-editor .hljs-deletion { color: var(--color-api-str); }
            .script-editor .hljs-number,
            .script-editor .hljs-literal { color: var(--color-api-kw); }
            .script-editor .hljs-built_in,
            .script-editor .hljs-title,
            .script-editor .hljs-section { color: var(--color-api-flag); }
            .script-editor .hljs-variable,
            .script-editor .hljs-template-variable,
            .script-editor .hljs-symbol { color: var(--color-code-symbol); }
            .script-editor .hljs-name,
            .script-editor .hljs-attribute,
            .script-editor .hljs-attr { color: var(--color-api-kw); }
            .script-editor .hljs-regexp,
            .script-editor .hljs-link { color: var(--color-code-link); }
            .script-editor .hljs-meta { color: var(--color-api-kw); }
            .script-editor .hljs-punctuation { color: var(--color-code-text); }
            .script-editor .hljs-addition { color: var(--color-code-addition); }
            .script-editor .hljs-operator { color: var(--color-code-keyword); }
            .script-editor .hljs-params { color: var(--color-code-text); }
            .script-editor .hljs-property { color: var(--color-api-kw); }
            .script-editor .hljs-type,
            .script-editor .hljs-selector-class,
            .script-editor .hljs-selector-id { color: var(--color-code-symbol); }
        `;
        document.head.appendChild(s);
    }

    function init(textareaId) {
        const textarea = document.getElementById(textareaId);
        if (!textarea || !window.hljs) return function () {};

        injectStyles();
        textarea.setAttribute('wrap', 'off');

        const wrapper = document.createElement('div');
        wrapper.className = 'script-editor';
        textarea.parentNode.insertBefore(wrapper, textarea);
        wrapper.appendChild(textarea);

        const pre = document.createElement('pre');
        pre.setAttribute('aria-hidden', 'true');
        const code = document.createElement('code');
        code.className = 'language-javascript hljs';
        pre.appendChild(code);
        wrapper.insertBefore(pre, textarea);

        function sync() {
            if (!textarea.value) {
                code.innerHTML = '';
            } else {
                code.innerHTML = hljs.highlight(textarea.value, { language: 'javascript' }).value + '\n';
            }
            pre.scrollTop = textarea.scrollTop;
            pre.scrollLeft = textarea.scrollLeft;
        }

        textarea.addEventListener('input', sync);
        textarea.addEventListener('scroll', function () {
            pre.scrollTop = textarea.scrollTop;
            pre.scrollLeft = textarea.scrollLeft;
        });

        textarea.addEventListener('keydown', function (e) {
            if (e.key === 'Tab') {
                e.preventDefault();
                var start = this.selectionStart;
                var end = this.selectionEnd;
                this.value = this.value.substring(0, start) + '    ' + this.value.substring(end);
                this.selectionStart = this.selectionEnd = start + 4;
                sync();
            }
        });

        sync();
        return sync;
    }

    return { init };
})();
