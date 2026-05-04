/* jshint esversion: 11, browser: true */
/* globals MapperState, MapperRender, escapeHtml, gridCellOccupied */
'use strict';

var MapperCtxMenu = (function() {
    var el = null;
    var providers = [];

    function init(menuEl) { el = menuEl; }

    function registerProvider(fn) { providers.push(fn); }

    function show(mx, my, target) {
        // target: { type: 'room'|'empty', roomId, room, gx, gy, gz }
        var items = [];
        providers.forEach(function(fn) {
            var provided = fn(target);
            if (provided) items = items.concat(provided);
        });
        if (items.length === 0) return;

        var html = '';
        if (target.type === 'room') {
            var label = target.room ? escapeHtml(target.room.Title) : 'Room';
            html += '<div class="ctx-header">#' + target.roomId + ' ' + label + '</div>';
        } else {
            html += '<div class="ctx-header">Cell (' + target.gx + ', ' + target.gy + ', ' + target.gz + ')</div>';
        }
        items.forEach(function(item) {
            var style = item.style ? ' style="' + item.style + '"' : '';
            var disabled = item.disabled ? ' disabled' : '';
            html += '<button class="ctx-item"' + style + disabled + '>' + item.label + '</button>';
        });
        el.innerHTML = html;
        el.style.display = 'block';
        position(mx, my);

        // Bind click actions
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

    function position(mx, my) {
        var mw = el.offsetWidth, mh = el.offsetHeight;
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

    function hide() { if (el) el.style.display = 'none'; }

    return { init: init, registerProvider: registerProvider, show: show, hide: hide };
})();
