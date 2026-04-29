/**
 * vwin.js
 *
 * Lightweight floating-window library, purpose-built for the GoMud web client.
 *
 * Public API (mirrors Winbox subset):
 *
 *   new VWin(opts)
 *     opts.title      string    — title bar text
 *     opts.mount      Element   — content element to place in the body area
 *     opts.x          number|'right'|'center'  — initial left position
 *     opts.y          number|'center'           — initial top position
 *     opts.width      number    — initial width in px
 *     opts.height     number    — initial height in px
 *     opts.header     number    — header bar height in px (default 35)
 *     opts.bottom     number    — reserved space at the bottom of the viewport
 *     opts.background string    — CSS background for the title bar
 *     opts.border     number    — body margin in px
 *     opts.class      string    — extra class(es) added to the root element
 *     opts.onclose    function  — called when the X button is clicked;
 *                                 return true to cancel the close
 *     opts.onmove     function(x, y)    — called after every drag
 *     opts.onresize   function(w, h)    — called after every resize
 *     opts.oncreate   function(opts)    — called synchronously at end of constructor
 *
 *   Instance properties:
 *     .window   Element  — the root VWin element (used by window-comm.js)
 *     .body     Element  — the scrollable content area
 *     .id       string
 *     .title    string
 *     .x, .y, .width, .height   numbers (current geometry)
 *     .onclose, .onmove, .onresize   (reassignable callbacks)
 *
 *   Instance methods:
 *     .close()
 *     .move(x, y)
 *     .resize(w, h)
 *     .setTitle(text)
 *     .setBackground(css)
 *     .focus()
 *     .addClass(name)
 *     .removeClass(name)
 *     .addControl({ index, class, image, click })
 *     .removeControl(className)
 */

(function () {
    'use strict';

    // -------------------------------------------------------------------------
    // Stylesheet — injected once
    // -------------------------------------------------------------------------
    var _styleInjected = false;
    function _injectStyle() {
        if (_styleInjected) { return; }
        _styleInjected = true;
        var s = document.createElement('style');
        s.textContent = [
            '.vwin {',  
            '  position: fixed;',
            '  left: 0; top: 0;',
            '  background: #0050ff;',
            '  box-shadow: 0 14px 28px rgba(0,0,0,.25), 0 10px 10px rgba(0,0,0,.22);',
            '  contain: layout size;',
            '  text-align: left;',
            '  touch-action: none;',
            '  box-sizing: border-box;',
            '}',

            /* Header */
            '.vw-header {',  
            '  position: absolute; left: 0; top: 0;',
            '  width: 100%; height: 35px; line-height: 35px;',
            '  color: #fff; overflow: hidden; z-index: 1;',
            '  user-select: none;',
            '}',

            /* Drag area */
            '.vw-drag {',  
            '  height: 100%; padding-left: 10px; cursor: move;',
            '  overflow: hidden;',
            '}',

            /* Title text */
            '.vw-title {',  
            '  font-family: Arial, sans-serif; font-size: 14px;',
            '  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;',
            '}',

            /* Body */
            '.vw-body {',  
            '  position: absolute; left: 0; right: 0; bottom: 0;',
            '  top: 35px;',
            '  overflow: auto;',
            '  -webkit-overflow-scrolling: touch;',
            '  background: #fff;',
            '  contain: strict;',
            '  z-index: 0;',
            '}',

            /* Control strip (right side of header) */
            '.vw-control {',  
            '  float: right; height: 100%;',
            '  display: flex; align-items: center;',
            '}',
            '.vw-control span {',  
            '  display: inline-flex; align-items: center; justify-content: center;',
            '  width: 30px; height: 100%;',
            '  cursor: pointer; background-repeat: no-repeat; background-position: center;',
            '}',

            /* Close button — simple × drawn with CSS */
            '.vw-close::before {',  
            '  content: "\\00d7";',
            '  font-size: 18px; line-height: 1;',
            '  color: rgba(255,255,255,0.8);',
            '}',
            '.vw-close:hover::before { color: #fff; }',  

            /* Dock button added by webclient-core */
            '.vw-dock-btn { background-size: 14px 14px; opacity: 0.7; }',
            '.vw-dock-btn:hover { opacity: 1; }',  

            /* Resize handles */
            '.vw-n, .vw-s { position: absolute; left: 0; right: 0; height: 6px; z-index: 2; }',
            '.vw-n { top: -3px;    cursor: n-resize; }',
            '.vw-s { bottom: -3px; cursor: s-resize; }',
            '.vw-e, .vw-w { position: absolute; top: 0; bottom: 0; width: 6px; z-index: 2; }',
            '.vw-e { right: -3px; cursor: e-resize; }',
            '.vw-w { left: -3px;  cursor: w-resize; }',
            '.vw-nw, .vw-ne, .vw-sw, .vw-se { position: absolute; width: 12px; height: 12px; z-index: 3; }',
            '.vw-nw { top: -3px;    left: -3px;  cursor: nw-resize; }',
            '.vw-ne { top: -3px;    right: -3px; cursor: ne-resize; }',
            '.vw-sw { bottom: -3px; left: -3px;  cursor: sw-resize; }',
            '.vw-se { bottom: -3px; right: -3px; cursor: se-resize; }',  

            /* Focus ring */
            '.vwin.focus { box-shadow: 0 14px 28px rgba(0,0,0,.4), 0 10px 10px rgba(0,0,0,.35); }',  

            /* Drag-lock: disable pointer events on iframes while dragging/resizing */
            '.vw-lock .vw-body { pointer-events: none; }',  
        ].join('\n');
        var head = document.head || document.getElementsByTagName('head')[0];
        if (head.firstChild) {
            head.insertBefore(s, head.firstChild);
        } else {
            head.appendChild(s);
        }
    }

    // -------------------------------------------------------------------------
    // Globals
    // -------------------------------------------------------------------------
    var _zTop   = 10;    // current highest z-index
    var _idSeq  = 0;     // auto-incrementing window id counter
    var _all    = [];    // all live VWin instances

    // -------------------------------------------------------------------------
    // Helpers
    // -------------------------------------------------------------------------

    // Resolve a position/size value that may be a number, 'center', 'right', or 'bottom'.
    function _resolve(val, span, size) {
        if (typeof val === 'number') { return val; }
        if (val === 'center') { return Math.round((span - size) / 2); }
        if (val === 'right' || val === 'bottom') { return span - size; }
        var n = parseFloat(val);
        if (!isNaN(n)) {
            // percentage
            if (typeof val === 'string' && val.indexOf('%') !== -1) {
                return Math.round(span * n / 100);
            }
            return n;
        }
        return 0;
    }

    function _vpWidth()  { return document.documentElement.clientWidth; }
    function _vpHeight() { return document.documentElement.clientHeight; }

    // Apply a CSS property, caching the last value to skip redundant sets.
    function _css(el, prop, val) {
        var s = '' + val;
        if (el['_c_' + prop] !== s) {
            el.style.setProperty(prop, s);
            el['_c_' + prop] = s;
        }
    }

    // -------------------------------------------------------------------------
    // VWin constructor
    // -------------------------------------------------------------------------
    function VWin(opts) {
        if (!(this instanceof VWin)) { return new VWin(opts); }

        _injectStyle();

        // ---- parse opts ----
        var id         = opts.id         || ('vwin-' + (++_idSeq));
        var title      = opts.title      || '';
        var mount      = opts.mount      || null;
        var html       = opts.html       || null;
        var background = opts.background || null;
        var border     = opts.border     || 0;
        var headerH    = opts.header     || 35;
        var extraClass = opts['class']   || '';
        var bottom     = opts.bottom     || 0;
        var root       = opts.root       || document.body;

        // callbacks (reassignable)
        this.onclose  = opts.onclose  || null;
        this.onmove   = opts.onmove   || null;
        this.onresize = opts.onresize || null;

        // ---- viewport ----
        var vpW = _vpWidth();
        var vpH = _vpHeight() - bottom;

        // ---- geometry ----
        var w = opts.width  ? _resolve(opts.width,  vpW, 0) : Math.max(Math.round(vpW / 2), 150);
        var h = opts.height ? _resolve(opts.height, vpH, 0) : Math.max(Math.round(vpH / 2), headerH);
        var x = opts.x !== undefined ? _resolve(opts.x, vpW, w) : 0;
        var y = opts.y !== undefined ? _resolve(opts.y, vpH, h) : 0;

        this.id     = id;
        this.title  = title;
        this.x      = x;
        this.y      = y;
        this.width  = w;
        this.height = h;

        // ---- build DOM ----
        var root_el = document.createElement('div');
        root_el.id        = id;
        root_el.className = 'vwin' + (extraClass ? ' ' + extraClass : '');
        root_el.vwin      = this;

        // Header
        var hdr = document.createElement('div');
        hdr.className = 'vw-header';
        if (headerH !== 35) {
            hdr.style.height     = headerH + 'px';
            hdr.style.lineHeight = headerH + 'px';
        }

        // Control strip (right side of header — holds close button, custom controls)
        var ctrl = document.createElement('div');
        ctrl.className = 'vw-control';

        var closeBtn = document.createElement('span');
        closeBtn.className = 'vw-close';

        ctrl.appendChild(closeBtn);
        hdr.appendChild(ctrl);

        // Drag area + title
        var drag = document.createElement('div');
        drag.className = 'vw-drag';

        var titleEl = document.createElement('div');
        titleEl.className   = 'vw-title';
        titleEl.textContent = title;

        drag.appendChild(titleEl);
        hdr.appendChild(drag);

        // Body
        var body = document.createElement('div');
        body.className = 'vw-body';
        if (headerH !== 35) {
            body.style.top = headerH + 'px';
        }
        if (border) {
            body.style.margin = border + (isNaN(border) ? '' : 'px');
        }

        // Resize handles
        var handles = ['n','s','e','w','nw','ne','sw','se'];
        var handleEls = {};
        handles.forEach(function(d) {
            var el = document.createElement('div');
            el.className = 'vw-' + d;
            handleEls[d] = el;
            root_el.appendChild(el);
        });

        root_el.appendChild(hdr);
        root_el.appendChild(body);

        // Mount content
        if (mount) {
            body.appendChild(mount);
        } else if (html) {
            body.innerHTML = html;
        }

        // Apply background to header
        if (background) {
            root_el.style.background = background;
        }

        // Expose key elements
        this.window = root_el;
        this.body   = body;
        this._hdr   = hdr;
        this._ctrl  = ctrl;
        this._title = titleEl;

        // ---- position & size ----
        _css(root_el, 'width',  w + 'px');
        _css(root_el, 'height', h + 'px');
        _css(root_el, 'left',   x + 'px');
        _css(root_el, 'top',    y + 'px');
        _css(root_el, 'z-index', ++_zTop);
        this.index = _zTop;

        // ---- event wiring ----
        var self = this;

        closeBtn.addEventListener('click', function(e) {
            e.stopPropagation();
            e.preventDefault();
            self.close();
        });

        // Focus on mousedown anywhere on the window
        root_el.addEventListener('mousedown', function() {
            self.focus();
        }, true);

        // Drag
        _wireInteraction(this, drag, 'drag');

        // Resize handles
        handles.forEach(function(d) {
            _wireInteraction(self, handleEls[d], d);
        });

        // ---- add to DOM & registry ----
        root_el.style.position = 'fixed';
        root_el.style.display  = 'block';
        root_el.style.visibility = 'visible';
        root_el.style.opacity  = '1';
        root_el.style.pointerEvents = 'all';

        root.appendChild(root_el);
        _all.push(this);

        // ---- oncreate callback ----
        if (typeof opts.oncreate === 'function') {
            opts.oncreate.call(this, opts);
        }
    }

    // -------------------------------------------------------------------------
    // Drag and resize interaction
    // -------------------------------------------------------------------------
    function _wireInteraction(wb, el, type) {
        if (!el) { return; }

        var startX, startY, startW, startH, startLeft, startTop;
        var active = false;

        function onStart(e) {
            e.stopPropagation();
            e.preventDefault();

            var pt = e.touches ? e.touches[0] : e;
            startX    = pt.clientX;
            startY    = pt.clientY;
            startW    = wb.width;
            startH    = wb.height;
            startLeft = wb.x;
            startTop  = wb.y;
            active    = true;

            document.body.classList.add('vw-lock');

            if (e.touches) {
                document.addEventListener('touchmove', onMove, { capture: true, passive: false });
                document.addEventListener('touchend',  onEnd,  { capture: true, passive: true  });
            } else {
                document.addEventListener('mousemove', onMove, { capture: true, passive: false });
                document.addEventListener('mouseup',   onEnd,  { capture: true, passive: true  });
            }
        }

        function onMove(e) {
            if (!active) { return; }
            e.stopPropagation();
            if (!e.touches) { e.preventDefault(); }

            var pt = e.touches ? e.touches[0] : e;
            var dx = pt.clientX - startX;
            var dy = pt.clientY - startY;

            var vpW    = _vpWidth();
            var vpH    = _vpHeight();
            var newX   = startLeft;
            var newY   = startTop;
            var newW   = startW;
            var newH   = startH;
            var moved  = false;
            var resized = false;

            if (type === 'drag') {
                newX = startLeft + dx;
                newY = startTop  + dy;
                // Clamp so at least a strip remains grabbable
                newX = Math.max(-newW + 30, Math.min(newX, vpW - 30));
                newY = Math.max(0,          Math.min(newY, vpH - 30));
                moved = (newX !== wb.x || newY !== wb.y);
            } else {
                // Horizontal
                if (type === 'e' || type === 'se' || type === 'ne') {
                    newW = Math.max(150, startW + dx);
                    resized = true;
                } else if (type === 'w' || type === 'sw' || type === 'nw') {
                    newW = Math.max(150, startW - dx);
                    newX = startLeft + (startW - newW);
                    resized = true;
                    moved   = true;
                }
                // Vertical
                if (type === 's' || type === 'se' || type === 'sw') {
                    newH = Math.max(50, startH + dy);
                    resized = true;
                } else if (type === 'n' || type === 'ne' || type === 'nw') {
                    newH = Math.max(50, startH - dy);
                    newY = startTop + (startH - newH);
                    resized = true;
                    moved   = true;
                }
            }

            if (resized) {
                wb.width  = newW;
                wb.height = newH;
                _css(wb.window, 'width',  newW + 'px');
                _css(wb.window, 'height', newH + 'px');
                if (wb.onresize) { wb.onresize.call(wb, newW, newH); }
            }
            if (moved || type === 'drag') {
                wb.x = newX;
                wb.y = newY;
                _css(wb.window, 'left', newX + 'px');
                _css(wb.window, 'top',  newY + 'px');
                if (wb.onmove) { wb.onmove.call(wb, newX, newY); }
            }
        }

        function onEnd(e) {
            if (!active) { return; }
            active = false;
            e.stopPropagation();
            document.body.classList.remove('vw-lock');

            document.removeEventListener('mousemove', onMove, { capture: true });
            document.removeEventListener('mouseup',   onEnd,  { capture: true });
            document.removeEventListener('touchmove', onMove, { capture: true });
            document.removeEventListener('touchend',  onEnd,  { capture: true });
        }

        el.addEventListener('mousedown', onStart, { capture: true, passive: false });
        el.addEventListener('touchstart', onStart, { capture: true, passive: false });
    }

    // -------------------------------------------------------------------------
    // Prototype methods
    // -------------------------------------------------------------------------
    var P = VWin.prototype;

    P.close = function () {
        if (this.onclose && this.onclose()) {
            return; // cancelled
        }
        var idx = _all.indexOf(this);
        if (idx !== -1) { _all.splice(idx, 1); }
        if (this.window && this.window.parentNode) {
            this.window.parentNode.removeChild(this.window);
        }
        this.window = this.body = null;
    };

    P.move = function (x, y) {
        if (x !== undefined) { this.x = _resolve(x, _vpWidth(),  this.width);  }
        if (y !== undefined) { this.y = _resolve(y, _vpHeight(), this.height); }
        _css(this.window, 'left', this.x + 'px');
        _css(this.window, 'top',  this.y + 'px');
        if (this.onmove) { this.onmove.call(this, this.x, this.y); }
        return this;
    };

    P.resize = function (w, h) {
        if (w !== undefined) { this.width  = w; }
        if (h !== undefined) { this.height = h; }
        _css(this.window, 'width',  this.width  + 'px');
        _css(this.window, 'height', this.height + 'px');
        if (this.onresize) { this.onresize.call(this, this.width, this.height); }
        return this;
    };

    P.focus = function () {
        _all.forEach(function(wb) { wb.window && wb.window.classList.remove('focus'); });
        if (this.window) {
            _css(this.window, 'z-index', ++_zTop);
            this.index = _zTop;
            this.window.classList.add('focus');
        }
        return this;
    };

    P.setTitle = function (text) {
        this.title = text;
        if (this._title) { this._title.textContent = text; }
        return this;
    };

    P.setBackground = function (css) {
        if (this.window) { this.window.style.background = css; }
        return this;
    };

    P.addClass = function (name) {
        if (this.window) { this.window.classList.add(name); }
        return this;
    };

    P.removeClass = function (name) {
        if (this.window) { this.window.classList.remove(name); }
        return this;
    };

    /**
     * addControl({ index, class, image, click })
     *
     * Inserts a custom button into the control strip.
     * index 0 = leftmost position (prepend); default = append.
     */
    P.addControl = function (spec) {
        var btn = document.createElement('span');
        if (spec['class']) { btn.className = spec['class']; }
        if (spec.image)    { btn.style.backgroundImage = 'url(' + spec.image + ')'; }
        var self = this;
        if (typeof spec.click === 'function') {
            btn.addEventListener('click', function(e) {
                e.stopPropagation();
                spec.click.call(btn, e, self);
            });
        }
        var ctrl = this._ctrl;
        if (!ctrl) { return this; }
        if (spec.index === 0) {
            ctrl.insertBefore(btn, ctrl.firstChild);
        } else if (spec.index !== undefined && spec.index !== null) {
            var ref = ctrl.childNodes[spec.index] || null;
            ctrl.insertBefore(btn, ref);
        } else {
            ctrl.appendChild(btn);
        }
        return this;
    };

    P.removeControl = function (className) {
        if (!this._ctrl) { return this; }
        var el = this._ctrl.querySelector('.' + className);
        if (el) { el.parentNode.removeChild(el); }
        return this;
    };

    // -------------------------------------------------------------------------
    // Expose globally
    // -------------------------------------------------------------------------
    window.VWin = VWin;

})();
