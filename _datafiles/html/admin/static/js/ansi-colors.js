// AnsiColors - global utility for converting ANSI 256-color codes to CSS hex
// and rendering color-swatch HTML. Loaded by _header.html and available on all
// admin pages.

const AnsiColors = (() => {
    'use strict';

    const _base16 = [
        '#000000','#800000','#008000','#808000','#000080','#800080','#008080','#c0c0c0',
        '#808080','#ff0000','#00ff00','#ffff00','#0000ff','#ff00ff','#00ffff','#ffffff',
    ];
    const _cubeSteps = [0, 95, 135, 175, 215, 255];

    function toHex(n) {
        n = parseInt(n, 10);
        if (n < 16) return _base16[n] || 'var(--color-text-faint)';
        if (n > 231) {
            const v = 8 + 10 * (n - 232);
            const h = v.toString(16).padStart(2, '0');
            return '#' + h + h + h;
        }
        n -= 16;
        const r = _cubeSteps[Math.floor(n / 36)];
        const g = _cubeSteps[Math.floor((n % 36) / 6)];
        const b = _cubeSteps[n % 6];
        const hx = v => v.toString(16).padStart(2, '0');
        return '#' + hx(r) + hx(g) + hx(b);
    }

    function swatchHtml(colors, maxColors) {
        const list = maxColors ? colors.slice(0, maxColors) : colors;
        return list.map(c => '<span style="background:' + toHex(c) + '"></span>').join('');
    }

    function _esc(ch) {
        return ch.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
    }

    // Default: per-character coloring, pattern reverses at each end (spaces don't advance)
    function previewDefault(text, colors) {
        if (!colors || colors.length === 0) return _esc(text);
        let html = '';
        let dir = 1, pos = 0;
        for (const ch of text) {
            const hex = toHex(colors[pos]);
            html += '<span style="color:' + hex + '">' + _esc(ch) + '</span>';
            if (ch !== ' ') {
                if (pos === colors.length - 1) dir = -1;
                else if (pos === 0) dir = 1;
                pos += dir;
            }
        }
        return html;
    }

    // Words: color changes on each word boundary
    function previewWords(text, colors) {
        if (!colors || colors.length === 0) return _esc(text);
        let html = '';
        let pos = 0;
        let inWord = false;
        for (let i = 0; i < text.length; i++) {
            const ch = text[i];
            if (ch === ' ') {
                if (inWord) {
                    pos = (pos + 1) % colors.length;
                    inWord = false;
                }
                html += '<span style="color:' + toHex(colors[pos]) + '">' + _esc(ch) + '</span>';
            } else {
                inWord = true;
                html += '<span style="color:' + toHex(colors[pos]) + '">' + _esc(ch) + '</span>';
            }
        }
        return html;
    }

    // Once: advances through pattern once, stays on final color
    function previewOnce(text, colors) {
        if (!colors || colors.length === 0) return _esc(text);
        let html = '';
        let pos = 0;
        for (const ch of text) {
            const hex = toHex(colors[pos]);
            html += '<span style="color:' + hex + '">' + _esc(ch) + '</span>';
            if (ch !== ' ' && pos < colors.length - 1) {
                pos++;
            }
        }
        return html;
    }

    // Stretch: spreads pattern evenly across the full string length
    function previewStretch(text, colors) {
        if (!colors || colors.length === 0) return _esc(text);
        const len = text.length;
        const stretchAmount = Math.max(1, Math.floor(len / colors.length));
        let html = '';
        let pos = 0;
        let subCounter = 0;
        for (const ch of text) {
            const hex = toHex(colors[pos]);
            html += '<span style="color:' + hex + '">' + _esc(ch) + '</span>';
            subCounter++;
            if (pos < colors.length - 1) {
                if (subCounter % stretchAmount === 0) {
                    pos++;
                }
            }
        }
        return html;
    }

    // Legacy single-mode entry point (Default) kept for backward compatibility
    function previewTextHtml(text, colors) {
        return previewDefault(text, colors);
    }

    return { toHex, swatchHtml, previewTextHtml, previewDefault, previewWords, previewOnce, previewStretch };
})();
