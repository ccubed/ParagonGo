(function () {
    'use strict';

    var COOKIE_NAME = 'gomud_admin_theme';
    var COOKIE_DAYS = 365;

    var THEMES = [
        {
            id: 'default',
            label: 'Default',
            swatches: ['#1a1a2e', '#f5f5f5', '#ffffff', '#1e6e34', '#8a0000']
        },
        {
            id: 'slate',
            label: 'Slate',
            swatches: ['#2c3e50', '#ecf0f1', '#ffffff', '#1e8449', '#eaf2f8']
        },
        {
            id: 'arctic',
            label: 'Arctic',
            swatches: ['#0d4f6e', '#eaf6fd', '#ffffff', '#1e7a40', '#c03040']
        },
        {
            id: 'ember',
            label: 'Ember',
            swatches: ['#1c1208', '#f7f2ec', '#fffdf9', '#2a6e30', '#8a2000']
        },
        {
            id: 'parchment',
            label: 'Parchment',
            swatches: ['#3a2410', '#f5ead0', '#fdf6e3', '#3a7028', '#8a2010']
        },
        {
            id: 'forest',
            label: 'Forest',
            swatches: ['#1a3020', '#f0f5f1', '#fafcfa', '#2e7d32', '#c62828']
        },
        {
            id: 'bubblegum',
            label: 'Bubblegum',
            swatches: ['#e91e8c', '#fff0f8', '#ffffff', '#2a8a40', '#e00030']
        },
        {
            id: 'dusk',
            label: 'Dusk',
            swatches: ['#6c5ce7', '#141220', '#1e1b2e', '#4ade80', '#ff6b6b']
        },
        {
            id: 'crypt',
            label: 'Crypt',
            swatches: ['#1a1008', '#0e0a06', '#2a1c0c', '#68a848', '#c84040']
        },
        {
            id: 'toxic',
            label: 'Toxic',
            swatches: ['#39ff14', '#060a00', '#0c1400', '#39ff14', '#ff5030']
        },
        {
            id: 'synthwave',
            label: 'Synthwave',
            swatches: ['#ff2d78', '#08001a', '#0d0020', '#00f0ff', '#ffd020']
        }
    ];

    function getCookie(name) {
        var pairs = document.cookie.split(';');
        for (var i = 0; i < pairs.length; i++) {
            var pair = pairs[i].trim().split('=');
            if (pair[0] === name) return decodeURIComponent(pair[1] || '');
        }
        return '';
    }

    function setCookie(name, value, days) {
        var expires = new Date(Date.now() + days * 864e5).toUTCString();
        document.cookie = name + '=' + encodeURIComponent(value) +
            '; expires=' + expires + '; path=/; SameSite=Lax';
    }

    function applyTheme(id) {
        var link = document.getElementById('theme-stylesheet');
        if (!link) return;
        var href = '/admin/static/css/theme-' + id + '.css';
        if (link.getAttribute('href') !== href) link.setAttribute('href', href);
        THEMES.forEach(function (t) {
            document.documentElement.classList.toggle('theme-' + t.id, t.id === id);
        });
    }

    function buildPicker(currentId) {
        var wrap = document.createElement('div');
        wrap.id = 'theme-picker-wrap';
        wrap.style.cssText = 'position:relative;display:inline-block;';

        var btn = document.createElement('button');
        btn.id = 'theme-picker-btn';
        btn.title = 'Switch theme';
        btn.style.cssText =
            'background:none;border:1px solid var(--color-text-tertiary);border-radius:3px;' +
            'color:var(--color-nav-link);font-size:0.75rem;cursor:pointer;' +
            'padding:0.2rem 0.55rem;white-space:nowrap;display:flex;align-items:center;gap:0.35rem;';
        btn.innerHTML = '<span style="font-size:0.85rem;">&#9680;</span> Theme';

        var panel = document.createElement('div');
        panel.id = 'theme-picker-panel';
        panel.style.cssText =
            'display:none;position:absolute;top:calc(100% + 6px);right:0;' +
            'background:var(--color-nav-dropdown-bg);border:1px solid var(--color-nav-border);' +
            'border-radius:6px;padding:0.5rem;z-index:200;min-width:170px;' +
            'box-shadow:0 4px 16px rgba(0,0,0,0.35);';

        THEMES.forEach(function (theme) {
            var row = document.createElement('button');
            row.dataset.themeId = theme.id;
            row.style.cssText =
                'display:flex;align-items:center;gap:0.5rem;width:100%;' +
                'background:var(--color-nav-dropdown-bg);border:none;cursor:pointer;padding:0.35rem 0.5rem;' +
                'border-radius:4px;color:var(--color-nav-link);font-size:0.82rem;text-align:left;';
            if (theme.id === currentId) {
                row.style.background = 'var(--color-nav-dropdown-hover)';
                row.style.color = 'var(--color-nav-link-hover)';
            }

            var swatchWrap = document.createElement('span');
            swatchWrap.style.cssText =
                'display:inline-flex;gap:2px;flex-shrink:0;border-radius:3px;overflow:hidden;' +
                'border:1px solid rgba(255,255,255,0.15);';
            theme.swatches.forEach(function (color) {
                var s = document.createElement('span');
                s.style.cssText = 'display:inline-block;width:10px;height:16px;background:' + color + ';';
                swatchWrap.appendChild(s);
            });

            var label = document.createElement('span');
            label.textContent = theme.label;

            row.appendChild(swatchWrap);
            row.appendChild(label);

            row.addEventListener('mouseover', function () {
                row.style.background = 'var(--color-nav-dropdown-hover)';
                row.style.color = 'var(--color-nav-link-hover)';
            });
            row.addEventListener('mouseout', function () {
                if (row.dataset.themeId !== getCurrentId()) {
                    row.style.background = 'var(--color-nav-dropdown-bg)';
                    row.style.color = 'var(--color-nav-link)';
                }
            });
            row.addEventListener('click', function () {
                var id = row.dataset.themeId;
                setCookie(COOKIE_NAME, id, COOKIE_DAYS);
                applyTheme(id);
                closePanel();
                refreshActiveState(id);
            });

            panel.appendChild(row);
        });

        function getCurrentId() {
            return getCookie(COOKIE_NAME) || 'default';
        }

        function refreshActiveState(activeId) {
            var rows = panel.querySelectorAll('[data-theme-id]');
            rows.forEach(function (r) {
                var isActive = r.dataset.themeId === activeId;
                r.style.background = isActive ? 'var(--color-nav-dropdown-hover)' : 'var(--color-nav-dropdown-bg)';
                r.style.color = isActive ? 'var(--color-nav-link-hover)' : 'var(--color-nav-link)';
            });
        }

        var open = false;
        function openPanel() {
            panel.style.display = 'block';
            open = true;
        }
        function closePanel() {
            panel.style.display = 'none';
            open = false;
        }

        btn.addEventListener('click', function (e) {
            e.stopPropagation();
            if (open) closePanel(); else openPanel();
        });

        document.addEventListener('click', function (e) {
            if (open && !wrap.contains(e.target)) closePanel();
        });

        wrap.appendChild(btn);
        wrap.appendChild(panel);
        return wrap;
    }

    function init() {
        var savedTheme = getCookie(COOKIE_NAME) || 'default';
        applyTheme(savedTheme);

        var nav = document.querySelector('nav');
        if (!nav) return;

        var cacheBtn = document.getElementById('nav-clear-cache-btn');
        var picker = buildPicker(savedTheme);
        if (cacheBtn) {
            nav.insertBefore(picker, cacheBtn);
        } else {
            nav.appendChild(picker);
        }
    }

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }
})();
