/**
 * mapper-ctx-menu.js
 *
 * Context menu for the mapper canvas. Uses a provider pattern: any module
 * can register a provider function via registerProvider(). When the menu
 * opens, every provider is called with a target descriptor and may return
 * an array of menu items (or null to contribute nothing).
 *
 * Menu item shape:
 *   { label: string, action: function, style?: string, disabled?: boolean }
 */
/* jshint esversion: 11, browser: true */
/* globals MapperState, MapperRender, escapeHtml, gridCellOccupied */
'use strict';

var MapperCtxMenu = (function() {
    var el        = null;
    var providers = [];

    // --- Initialization ---

    function init(menuEl) { el = menuEl; }

    // --- Provider Registration ---

    /** Register a function(target) that returns menu items or null. */
    function registerProvider(fn) { providers.push(fn); }

    // --- Display ---

    /**
     * Build and show the context menu at viewport coordinates (mx, my).
     *
     * @param {number} mx     - Mouse X in viewport pixels.
     * @param {number} my     - Mouse Y in viewport pixels.
     * @param {Object} target - Describes what was right-clicked:
     *   {
     *     type:   'room' | 'empty',
     *     roomId: number|undefined,  // present when type === 'room'
     *     room:   Object|undefined,  // the room data object
     *     gx:     number,            // grid X of the click
     *     gy:     number,            // grid Y of the click
     *     gz:     number             // grid Z (current Z level)
     *   }
     */
    function show(mx, my, target) {
        var items = [];
        providers.forEach(function(fn) {
            var provided = fn(target);
            if (provided) items = items.concat(provided);
        });
        if (items.length === 0) return;

        // Build header: show room title for rooms, grid coords for empty cells
        var html = '';
        if (target.type === 'room') {
            var label = target.room ? escapeHtml(target.room.Title) : 'Room';
            html += '<div class="ctx-header">#' + target.roomId + ' ' + label + '</div>';
        } else {
            html += '<div class="ctx-header">Cell (' + target.gx + ', ' + target.gy + ', ' + target.gz + ')</div>';
        }

        items.forEach(function(item) {
            var style    = item.style ? ' style="' + item.style + '"' : '';
            var disabled = item.disabled ? ' disabled' : '';
            html += '<button class="ctx-item"' + style + disabled + '>' + item.label + '</button>';
        });

        el.innerHTML = html;
        el.style.display = 'block';
        position(mx, my);

        // Bind click actions via closure to preserve correct item index
        var buttons = el.querySelectorAll('.ctx-item');
        for (var i = 0; i < buttons.length; i++) {
            (function(idx) {
                buttons[idx].addEventListener('click', function() {
                    hide();
                    if (items[idx].action && !items[idx].disabled) items[idx].action();
                });
            })(i);
        }
    }

    // --- Positioning ---

    /** Clamp the menu within the viewport so it never overflows off-screen. */
    function position(mx, my) {
        var mw = el.offsetWidth,  mh = el.offsetHeight;
        var vw = window.innerWidth, vh = window.innerHeight;

        var left = mx;
        if (left + mw > vw - 8) left = mx - mw;
        left = Math.max(8, left);

        var top = my;
        if (top + mh > vh - 8) top = my - mh;
        top = Math.max(8, top);

        el.style.left = left + 'px';
        el.style.top  = top + 'px';
    }

    // --- Hide ---

    function hide() { if (el) el.style.display = 'none'; }

    return { init: init, registerProvider: registerProvider, show: show, hide: hide };
})();
