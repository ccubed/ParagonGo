/* site-settings.js - Public site settings modal (theme picker, settings stubs).
 * Independent of webclient code. Uses localStorage key 'gomud-site-theme'.
 */
(function () {
    'use strict';

    var DEFAULT_THEME = 'brooding';
    var STORAGE_KEY   = 'gomud-site-theme';

    /* ── Theme switching ──────────────────────────────────────── */
    function setSiteTheme(name, keepDropdownOpen) {
        var link = document.getElementById('site-theme-css');
        if (!link) { return; }
        link.href = link.href.replace(/theme-[\w-]+\.css/, 'theme-' + name + '.css');
        localStorage.setItem(STORAGE_KEY, name);

        document.querySelectorAll('.site-theme-option').forEach(function (el) {
            el.classList.toggle('selected', el.dataset.theme === name);
        });

        _updateDropdownHeader(name);

        if (!keepDropdownOpen) {
            var dd = document.getElementById('site-theme-dropdown');
            if (dd) { dd.classList.remove('open'); }
        }
    }

    function _updateDropdownHeader(name) {
        var header = document.getElementById('site-theme-dropdown-header');
        if (!header) { return; }

        var option = document.querySelector('.site-theme-option[data-theme="' + name + '"]');
        if (!option) { return; }

        var labelEl = header.querySelector('.site-theme-dropdown-label');
        if (labelEl) {
            var srcLabel = option.querySelector('.site-theme-label');
            labelEl.textContent = srcLabel ? srcLabel.textContent : name;
        }

        var swatchContainer = header.querySelector('.site-theme-dropdown-swatches');
        if (swatchContainer) {
            var srcSwatches = option.querySelectorAll('.site-theme-swatch');
            swatchContainer.innerHTML = '';
            srcSwatches.forEach(function (s) {
                var clone = s.cloneNode(true);
                swatchContainer.appendChild(clone);
            });
        }
    }

    /* ── Modal open / close ───────────────────────────────────── */
    function openSiteSettings() {
        var backdrop = document.getElementById('site-settings-backdrop');
        if (backdrop) { backdrop.classList.add('open'); }
    }

    function closeSiteSettings() {
        var backdrop = document.getElementById('site-settings-backdrop');
        if (backdrop) { backdrop.classList.remove('open'); }
    }

    /* ── Tab switching ────────────────────────────────────────── */
    function _initTabs() {
        var tabBar = document.getElementById('site-settings-tab-bar');
        if (!tabBar) { return; }

        tabBar.addEventListener('click', function (e) {
            var btn = e.target.closest('.site-settings-tab-btn');
            if (!btn) { return; }

            tabBar.querySelectorAll('.site-settings-tab-btn').forEach(function (b) {
                b.classList.remove('active');
            });
            document.querySelectorAll('.site-settings-tab-panel').forEach(function (p) {
                p.classList.remove('active');
            });

            btn.classList.add('active');
            var panel = document.getElementById(btn.dataset.tab);
            if (panel) { panel.classList.add('active'); }
        });
    }

    /* ── Theme dropdown ───────────────────────────────────────── */
    function _initThemeDropdown() {
        var saved    = localStorage.getItem(STORAGE_KEY) || DEFAULT_THEME;
        var themeList = document.getElementById('site-theme-list');
        var dropdown  = document.getElementById('site-theme-dropdown');
        var ddHeader  = document.getElementById('site-theme-dropdown-header');

        if (!themeList || !dropdown) { return; }

        /* Restore saved selection highlight */
        themeList.querySelectorAll('.site-theme-option').forEach(function (el) {
            el.classList.toggle('selected', el.dataset.theme === saved);
        });
        _updateDropdownHeader(saved);

        /* Click on an option */
        themeList.addEventListener('click', function (e) {
            var option = e.target.closest('.site-theme-option');
            if (option) { setSiteTheme(option.dataset.theme); }
        });

        /* Toggle dropdown open/close */
        if (ddHeader) {
            ddHeader.setAttribute('tabindex', '0');
            ddHeader.addEventListener('click', function () {
                dropdown.classList.toggle('open');
            });

            /* Keyboard nav on the dropdown header */
            ddHeader.addEventListener('keydown', function (e) {
                var options = Array.from(themeList.querySelectorAll('.site-theme-option'));
                if (!options.length) { return; }
                var current = options.findIndex(function (el) {
                    return el.classList.contains('selected');
                });
                if (e.key === 'ArrowDown') {
                    e.preventDefault();
                    setSiteTheme(options[(current + 1) % options.length].dataset.theme, true);
                } else if (e.key === 'ArrowUp') {
                    e.preventDefault();
                    setSiteTheme(options[(current - 1 + options.length) % options.length].dataset.theme, true);
                } else if (e.key === 'Enter' || e.key === ' ') {
                    e.preventDefault();
                    dropdown.classList.toggle('open');
                } else if (e.key === 'Escape') {
                    dropdown.classList.remove('open');
                }
            });
        }

        /* Close dropdown when clicking outside it */
        document.addEventListener('click', function (e) {
            if (dropdown && !dropdown.contains(e.target)) {
                dropdown.classList.remove('open');
            }
        });
    }

    /* ── Auto-login stub ──────────────────────────────────────── */
    function _initAutoLogin() {
        var toggle = document.getElementById('site-autologin-toggle');
        if (!toggle) { return; }

        /* Restore saved state */
        toggle.checked = localStorage.getItem('gomud-autologin') === '1';

        toggle.addEventListener('change', function () {
            localStorage.setItem('gomud-autologin', toggle.checked ? '1' : '0');
        });
    }

    /* ── Backdrop click-to-close + Escape ─────────────────────── */
    function _initBackdrop() {
        var backdrop = document.getElementById('site-settings-backdrop');
        if (!backdrop) { return; }

        backdrop.addEventListener('click', function (e) {
            if (e.target === backdrop) { closeSiteSettings(); }
        });

        document.addEventListener('keydown', function (e) {
            if (e.key === 'Escape') { closeSiteSettings(); }
        });

        var closeBtn = document.getElementById('site-settings-close');
        if (closeBtn) {
            closeBtn.addEventListener('click', closeSiteSettings);
        }
    }

    /* ── Gear button ──────────────────────────────────────────── */
    function _initGearButton() {
        var btn = document.getElementById('site-gear-btn');
        if (btn) {
            btn.addEventListener('click', openSiteSettings);
        }
    }

    /* ── Boot ─────────────────────────────────────────────────── */
    document.addEventListener('DOMContentLoaded', function () {
        _initBackdrop();
        _initGearButton();
        _initTabs();
        _initThemeDropdown();
        _initAutoLogin();
    });

    /* Expose for inline onclick if ever needed */
    window.siteSettings = {
        open:     openSiteSettings,
        close:    closeSiteSettings,
        setTheme: setSiteTheme
    };
}());
