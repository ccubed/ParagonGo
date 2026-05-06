/**
 * window-map.js
 *
 * Virtual window: Map with two tabs - "2D" (flat grid) and "3D" (isometric).
 *
 * Both views share a single data pipeline:
 *   - One roomInfoStore, one roomCache, one World.Map request per session.
 *   - Shared color/symbol lookup tables.
 *
 * Responds to GMCP namespaces:
 *   Room      - incremental update as the player moves room-to-room
 *   World.Map - bulk snapshot of all visited rooms, requested once on connect
 */

'use strict';

(function () {

    // =========================================================================
    // Shared constants
    // =========================================================================

    var ZOOM_STEP = 1.25;  // How much each zoom button click scales the view; higher = bigger jumps per click
    var ZOOM_MIN  = 0.25;  // Furthest out the user can zoom; lower = more of the map visible but smaller rooms
    var ZOOM_MAX  = 4.0;   // Closest in the user can zoom; higher = larger rooms but less of the map visible

    var CENTER_EASE_DURATION = 0.2; // How long (seconds) the camera takes to pan to a new room; 0 = instant snap, higher = slower slide

    var CONNECTION_COLOR        = '#7a4a1a'; // Color of the lines drawn between connected rooms; change to make corridors more or less visible
    var CURRENT_ROOM_COLOR      = '#c20000'; // Fill color of the room the player is currently in; change to adjust how much it stands out
    var CURRENT_ROOM_TEXT_COLOR = '#ffffff'; // Symbol character color inside the current room; should contrast with CURRENT_ROOM_COLOR
    var SYMBOL_TEXT_COLOR       = '#e0e0e0'; // Symbol character color inside all non-current rooms; lower contrast = subtler symbols

    var DEFAULT_ROOM_COLOR = '#3a3a4a';


    // =========================================================================
    // Shared helpers
    // =========================================================================

    function symbolForRoom(info) {
        if (info.mapsymbol) { return info.mapsymbol; }
        return '\u2022';
    }

    function colorForSymbol(sym, env) {
        return DEFAULT_ROOM_COLOR;
    }

    function smoothstep(t) {
        return t * t * (3 - 2 * t);
    }

    // =========================================================================
    // Shared data pipeline
    // =========================================================================

    /** Full GMCP info objects keyed by roomId - used by both views for tooltips. */
    var roomInfoStore = new Map();

    /**
     * roomCache: keyed by roomId.
     * { RoomId, zoneName, x, y, z, symbol, env, exits, stubs, hasUp, hasDown }
     * exits includes cross-z exits (for 3D); same-z only used by 2D view.
     */
    var roomCache = {};

    var worldMapRequested = false;

    function upsertRoomCache(id, zoneName, gx, gy, gz, sym, env, exitsv2) {
        var exitIds   = [];
        var exitStubs = [];
        var hasUp     = false;
        var hasDown   = false;

        if (exitsv2) {
            for (var dir in exitsv2) {
                var exitInfo = exitsv2[dir];

                if (exitInfo.dz > 0) { hasUp   = true; }
                if (exitInfo.dz < 0) { hasDown = true; }

                if (exitInfo.dx === 0 && exitInfo.dy === 0 && exitInfo.dz === 0) { continue; }

                var isSecret    = Array.isArray(exitInfo.details) && exitInfo.details.indexOf('secret') !== -1;
                var isLocked    = Array.isArray(exitInfo.details) && exitInfo.details.indexOf('locked') !== -1;
                var destVisited = roomInfoStore.has(exitInfo.num);

                if (isSecret && !destVisited) { continue; }

                if (destVisited) {
                    exitIds.push({ num: exitInfo.num, locked: isLocked, secret: isSecret, dz: exitInfo.dz });
                } else {
                    exitStubs.push({ dx: exitInfo.dx, dy: exitInfo.dy, dz: exitInfo.dz, locked: isLocked, secret: isSecret });
                }
            }
        }

        roomCache[id] = {
            RoomId:   id,
            zoneName: zoneName,
            x: gx, y: gy, z: gz,
            symbol:   sym,
            env:      env,
            exits:    exitIds,
            stubs:    exitStubs,
            hasUp:    hasUp,
            hasDown:  hasDown,
        };
    }

    function ingestWorldMap(entries) {
        if (!Array.isArray(entries) || entries.length === 0) { return; }

        entries.forEach(function (info) {
            if (info.num) { roomInfoStore.set(info.num, info); }
        });

        entries.forEach(function (info) {
            var id = info.num;
            if (!id) { return; }
            var coords = info.coords ? info.coords.split(',').map(function (s) { return s.trim(); }) : null;
            if (!coords || coords.length < 4) { return; }
            var zoneName = coords[0];
            var gx = parseInt(coords[1], 10);
            var gy = parseInt(coords[2], 10);
            var gz = parseInt(coords[3], 10);
            var isZoneRoot = Array.isArray(info.details) && info.details.indexOf('root') !== -1;
            if (gx === 0 && gy === 0 && !isZoneRoot) { return; }
            upsertRoomCache(id, zoneName, gx, gy, gz, symbolForRoom(info), info.environment || '', info.exitsv2);
        });

        view2d.onWorldMap();
        view3d.onWorldMap();
    }

    // =========================================================================
    // Shared tooltip (one element, used by whichever view is active)
    // =========================================================================

    var tooltip          = null;
    var tooltipHideTimer = null;

    function ensureTooltip(idSuffix) {
        if (tooltip) { return; }
        tooltip = document.createElement('div');
        tooltip.id = 'map-tooltip';
        document.body.appendChild(tooltip);
    }

    function showTooltip(mouseX, mouseY, info, showZ) {
        ensureTooltip();
        clearTimeout(tooltipHideTimer);

        var html = '<div class="tt-name">' + (info.name || 'Unknown') + '</div>';
        var rows = [];
        if (info.environment) { rows.push({ label: 'Env',    value: info.environment }); }
        if (info.maplegend)   { rows.push({ label: 'Type',   value: info.maplegend   }); }
        if (info.mapsymbol)   { rows.push({ label: 'Symbol', value: info.mapsymbol   }); }
        if (info.area)        { rows.push({ label: 'Area',   value: info.area        }); }
        if (showZ && info.coords) {
            var c = info.coords.split(',').map(function (s) { return s.trim(); });
            if (c.length >= 4) { rows.push({ label: 'Z', value: c[3] }); }
        }
        if (rows.length > 0) {
            html += '<hr class="tt-divider">';
            rows.forEach(function (r) {
                html += '<div class="tt-row"><span class="tt-label">' + r.label +
                        '</span><span class="tt-value">' + r.value + '</span></div>';
            });
        }

        var details    = info.details || [];
        var badgeOrder = ['pvp', 'bank', 'trainer', 'storage', 'character', 'ephemeral'];
        var badges     = badgeOrder.filter(function (d) { return details.indexOf(d) !== -1; });
        if (badges.length > 0) {
            html += '<hr class="tt-divider"><div class="tt-badges">';
            badges.forEach(function (b) { html += '<span class="tt-badge ' + b + '">' + b + '</span>'; });
            html += '</div>';
        }

        if (info.exitsv2) {
            var exitNames = Object.keys(info.exitsv2).filter(function (dir) {
                var e = info.exitsv2[dir];
                return !(Array.isArray(e.details) && e.details.indexOf('secret') !== -1) ||
                       roomInfoStore.has(e.num);
            }).sort();
            if (exitNames.length > 0) {
                html += '<hr class="tt-divider"><div class="tt-row">' +
                        '<span class="tt-label">Exits</span>' +
                        '<span class="tt-value">' + exitNames.join(', ') + '</span></div>';
            }
        }

        tooltip.innerHTML     = html;
        tooltip.style.display = 'block';
        positionTooltip(mouseX, mouseY);
    }

    function positionTooltip(mouseX, mouseY) {
        if (!tooltip) { return; }
        var ttW  = tooltip.offsetWidth;
        var ttH  = tooltip.offsetHeight;
        var vw   = window.innerWidth;
        var vh   = window.innerHeight;
        var left = mouseX + 14;
        if (left + ttW > vw - 8) { left = mouseX - ttW - 14; }
        left = Math.max(8, left);
        var top = mouseY - Math.floor(ttH / 2);
        if (top + ttH > vh - 8) { top = vh - ttH - 8; }
        top = Math.max(8, top);
        tooltip.style.left = left + 'px';
        tooltip.style.top  = top  + 'px';
    }

    function hideTooltip() {
        tooltipHideTimer = setTimeout(function () {
            if (tooltip) { tooltip.style.display = 'none'; }
        }, 80);
    }

    // =========================================================================
    // Styles
    // =========================================================================

    injectStyles([
        '#map-window {',
        '    display: flex;',
        '    flex-direction: column;',
        '    width: 100%;',
        '    height: 100%;',
        '    background: var(--t-bg-panel);',
        '}',
        '#map-tab-bar {',
        '    display: flex;',
        '    flex-shrink: 0;',
        '    background: var(--t-map-tab-bg);',
        '    border-bottom: 1px solid var(--t-map-tab-border);',
        '}',
        '#map-tab-bar button {',
        '    flex: 1;',
        '    padding: 4px 0;',
        '    background: none;',
        '    border: none;',
        '    border-bottom: 2px solid transparent;',
        '    color: var(--t-map-tab-text);',
        '    font-size: 11px;',
        '    font-family: monospace;',
        '    cursor: pointer;',
        '    letter-spacing: 0.05em;',
        '}',
        '#map-tab-bar button:hover { color: var(--t-map-tab-hover); }',
        '#map-tab-bar button.active {',
        '    color: var(--t-map-tab-active-text);',
        '    border-bottom-color: var(--t-map-tab-active-border);',
        '}',
        '#map-panels {',
        '    flex: 1;',
        '    position: relative;',
        '    overflow: hidden;',
        '}',
        '.map-panel {',
        '    position: absolute;',
        '    inset: 0;',
        '    display: none;',
        '}',
        '.map-panel.active { display: block; }',
        '.map-canvas-wrap {',
        '    width: 100%;',
        '    height: 100%;',
        '    position: relative;',
        '    overflow: hidden;',
        '}',
        '.map-canvas-wrap canvas {',
        '    display: block;',
        '    position: absolute;',
        '    top: 0; left: 0;',
        '    cursor: grab;',
        '}',
        '#map-tooltip {',
        '    position: fixed;',
        '    z-index: 99999;',
        '    pointer-events: none;',
        '    background: var(--t-bg-surface);',
        '    border: 1px solid var(--t-accent-dim);',
        '    border-radius: 6px;',
        '    box-shadow: 0 4px 16px rgba(0,0,0,0.7);',
        '    padding: 8px 10px;',
        '    min-width: 140px;',
        '    max-width: 240px;',
        '    display: none;',
        '    font-family: monospace;',
        '}',
        '#map-tooltip .tt-name { font-size:0.85em; font-weight:bold; color:var(--t-text); margin-bottom:4px; line-height:1.3; }',
        '#map-tooltip .tt-divider { border:none; border-top:1px solid var(--t-accent-dim); margin:5px 0; }',
        '#map-tooltip .tt-row { display:flex; justify-content:space-between; align-items:baseline; gap:8px; font-size:0.75em; line-height:1.6; }',
        '#map-tooltip .tt-label { color:var(--t-text-secondary); text-transform:uppercase; letter-spacing:0.04em; font-size:0.88em; flex-shrink:0; }',
        '#map-tooltip .tt-value { color:var(--t-text); text-align:right; }',
        '#map-tooltip .tt-badges { display:flex; flex-wrap:wrap; gap:3px; margin-top:4px; }',
        '#map-tooltip .tt-badge { font-size:0.62em; padding:1px 4px; border-radius:3px; background:var(--t-map-badge-bg); color:var(--t-text-secondary); border:1px solid var(--t-accent-dim); }',
        '#map-tooltip .tt-badge.pvp     { background:var(--t-badge-pvp-bg); color:var(--t-badge-pvp-text); border-color:var(--t-badge-pvp-border); }',
        '#map-tooltip .tt-badge.bank    { background:var(--t-badge-bank-bg); color:var(--t-badge-bank-text); border-color:var(--t-badge-bank-border); }',
        '#map-tooltip .tt-badge.trainer { background:var(--t-badge-trainer-bg); color:var(--t-badge-trainer-text); border-color:var(--t-badge-trainer-border); }',
        '#map-tooltip .tt-badge.storage { background:var(--t-badge-storage-bg); color:var(--t-badge-storage-text); border-color:var(--t-badge-storage-border); }',
        '.map-controls {',
        '    position: absolute;',
        '    top: 6px;',
        '    right: 6px;',
        '    display: flex;',
        '    align-items: center;',
        '    gap: 2px;',
        '    z-index: 10;',
        '}',
        '.map-controls .ctrl-sep {',
        '    width: 1px; height: 16px;',
        '    background: var(--t-map-controls-border);',
        '    margin: 0 3px;',
        '}',
        '.map-controls button {',
        '    width: 22px; height: 22px;',
        '    padding: 0;',
        '    font-size: 14px;',
        '    line-height: 1;',
        '    background: var(--t-map-controls-bg);',
        '    color: var(--t-map-controls-text);',
        '    border: 1px solid var(--t-map-controls-border);',
        '    border-radius: 3px;',
        '    cursor: pointer;',
        '}',
        '.map-controls button:hover { background: var(--t-map-controls-hover); color: var(--t-text-white); }',
        '.map-controls button.active { background: var(--t-map-controls-active); color: var(--t-map-tab-active-text); border-color: var(--t-map-tab-active-border); }',
        '.map-z-levels {',
        '    position: absolute;',
        '    top: 34px;',
        '    right: 6px;',
        '    display: flex;',
        '    flex-direction: column;',
        '    align-items: center;',
        '    gap: 2px;',
        '    z-index: 10;',
        '}',
        '.map-z-levels button {',
        '    width: 22px; height: 22px;',
        '    padding: 0;',
        '    font-size: 10px;',
        '    line-height: 1;',
        '    background: var(--t-map-controls-bg);',
        '    color: var(--t-map-controls-text);',
        '    border: 1px solid var(--t-map-controls-border);',
        '    border-radius: 3px;',
        '    cursor: pointer;',
        '    font-family: monospace;',
        '}',
        '.map-z-levels button:hover { background: var(--t-map-controls-hover); color: var(--t-text-white); }',
        '.map-z-levels button.active { background: var(--t-map-controls-active); color: var(--t-map-tab-active-text); border-color: var(--t-map-tab-active-border); }',
    ].join('\n'));

    // =========================================================================
    // 2D view
    // =========================================================================

    var view2d = (function () {

        // -- Constants ---------------------------------------------------------
        var ROOM_SIZE        = 28;   // Pixel side length of each room square; larger = bigger rooms, fewer fit on screen
        var ROOM_GAP         = 14;   // Pixel gap between adjacent room squares; larger = more space between rooms, map spreads out
        var BASE_STEP        = ROOM_SIZE + ROOM_GAP; // Derived: center-to-center grid distance; do not edit directly
        var CONNECTION_WIDTH = 4;    // Stroke width of corridor lines; thicker = more visible connections, can obscure small rooms
        var ROOM_BORDER_WIDTH = 1.5; // Stroke width of the outline drawn around each room square; higher = bolder room edges
        var SYMBOL_FONT_SIZE  = 14;  // Font size of the symbol character drawn inside each room; larger = more readable but may overflow small rooms
        var MAP_BACKGROUND    = '#111';    // Canvas fill color behind all rooms; change to adjust overall map contrast
        var ROOM_BORDER_COLOR = '#000000'; // Outline color drawn around each room square; darker = crisper separation between rooms

        // -- State -------------------------------------------------------------
        var canvas        = null;
        var ctx           = null;
        var container     = null;
        var rooms         = new Map();
        var edges         = new Map();
        var zoneExitStubs = [];
        var currentRoomId = null;
        var cameraX = 0, cameraY = 0;
        var easeStartX = 0, easeStartY = 0;
        var easeTargetX = 0, easeTargetY = 0;
        var easeStartTime = null, easeRafId = null;
        var panOffsetX = 0, panOffsetY = 0;
        var dragActive = false;
        var dragStartPxX = 0, dragStartPxY = 0;
        var dragStartPanX = 0, dragStartPanY = 0;
        var zoomScale     = 1.0;
        var currentZoneKey = '';

        // -- Helpers -----------------------------------------------------------
        function resizeCanvas() {
            if (!canvas || !container) { return; }
            canvas.width  = container.clientWidth  || 1;
            canvas.height = container.clientHeight || 1;
        }

        function gridToCanvas(gx, gy) {
            var midX = Math.floor(canvas.width  / 2);
            var midY = Math.floor(canvas.height / 2);
            var step = BASE_STEP * zoomScale;
            return {
                px: midX + (gx - cameraX - panOffsetX) * step,
                py: midY + (gy - cameraY - panOffsetY) * step,
            };
        }

        function setCameraTarget(tx, ty) {
            panOffsetX = 0; panOffsetY = 0;
            if (CENTER_EASE_DURATION <= 0) { cameraX = tx; cameraY = ty; render(); return; }
            if (easeRafId !== null) { cancelAnimationFrame(easeRafId); easeRafId = null; }
            easeStartX = cameraX; easeStartY = cameraY;
            easeTargetX = tx; easeTargetY = ty;
            easeStartTime = null;
            function step(ts) {
                if (easeStartTime === null) { easeStartTime = ts; }
                var t = Math.min((ts - easeStartTime) / 1000 / CENTER_EASE_DURATION, 1);
                var s = smoothstep(t);
                cameraX = easeStartX + (easeTargetX - easeStartX) * s;
                cameraY = easeStartY + (easeTargetY - easeStartY) * s;
                render();
                easeRafId = t < 1 ? requestAnimationFrame(step) : null;
            }
            easeRafId = requestAnimationFrame(step);
        }

        function addOrUpdateRoom(id, gx, gy, symbol, env) {
            var rc = roomCache[id];
            rooms.set(id, { x: gx, y: gy, symbol: symbol || '\u2022', env: env || '',
                            hasUp: rc ? rc.hasUp : false, hasDown: rc ? rc.hasDown : false });
        }

        function addEdge(idA, idB, locked, secret) {
            var key = idA < idB ? (idA + '-' + idB) : (idB + '-' + idA);
            if (!edges.has(key)) {
                edges.set(key, { locked: !!locked, secret: !!secret });
            } else {
                var ex = edges.get(key);
                ex.locked = ex.locked || !!locked;
                ex.secret = ex.secret || !!secret;
            }
        }

        function resetMap() {
            rooms.clear(); edges.clear(); zoneExitStubs = [];
            currentRoomId = null;
            cameraX = 0; cameraY = 0; panOffsetX = 0; panOffsetY = 0;
            dragActive = false;
            if (easeRafId !== null) { cancelAnimationFrame(easeRafId); easeRafId = null; }
        }

        function replayZone(zoneKey) {
            resetMap();
            var zMatch = zoneKey.match(/\/z:(-?\d+)$/);
            if (!zMatch) { return; }
            var targetZ = parseInt(zMatch[1], 10);
            var visited = {}, queue = [];
            for (var rid in roomCache) {
                var rc = roomCache[rid];
                if (rc.z === targetZ && rc.zoneName + '/z:' + rc.z === zoneKey) {
                    queue.push(parseInt(rid, 10));
                }
            }
            while (queue.length > 0) {
                var id = queue.shift();
                if (visited[id]) { continue; }
                visited[id] = true;
                var r = roomCache[id];
                if (!r) { continue; }
                addOrUpdateRoom(r.RoomId, r.x, r.y, r.symbol, r.env);
                if (Array.isArray(r.exits)) {
                    r.exits.forEach(function (exit) {
                        if (exit.dz === 0 && !visited[exit.num] && roomCache[exit.num]) {
                            queue.push(exit.num);
                        }
                    });
                }
            }
            rooms.forEach(function (room, id) {
                var r = roomCache[id];
                if (!r) { return; }
                if (Array.isArray(r.exits)) {
                    r.exits.forEach(function (exit) {
                        if (exit.dz === 0 && rooms.has(exit.num)) {
                            addEdge(id, exit.num, exit.locked, exit.secret);
                        }
                    });
                }
                if (Array.isArray(r.stubs)) {
                    r.stubs.forEach(function (stub) {
                        if (stub.dz === 0) {
                            zoneExitStubs.push({ roomId: id, dx: stub.dx, dy: stub.dy,
                                                 locked: stub.locked, secret: stub.secret });
                        }
                    });
                }
            });
        }

        // -- Rendering ---------------------------------------------------------
        function drawLineBadge(mx, my, type) {
            var sz = Math.max(7, Math.round(CONNECTION_WIDTH * zoomScale * 2.5));
            var half = sz / 2;
            ctx.save();
            ctx.fillStyle = MAP_BACKGROUND;
            ctx.fillRect(mx - half, my - half, sz, sz);
            if (type === 'secret') {
                ctx.fillStyle = '#d4a843';
                ctx.font = 'bold ' + Math.round(sz * 0.85) + 'px monospace';
                ctx.textAlign = 'center'; ctx.textBaseline = 'middle';
                ctx.fillText('?', mx, my);
            } else {
                var kc = '#9ab0d4', lw = Math.max(1, sz * 0.14);
                ctx.strokeStyle = kc; ctx.fillStyle = kc;
                ctx.lineWidth = lw; ctx.lineCap = 'round';
                var bowR = sz * 0.22, bowCx = mx - sz * 0.14, bowCy = my;
                ctx.beginPath(); ctx.arc(bowCx, bowCy, bowR, 0, Math.PI * 2); ctx.stroke();
                var shaftX1 = bowCx + bowR, shaftX2 = mx + half * 0.82;
                ctx.beginPath(); ctx.moveTo(shaftX1, bowCy); ctx.lineTo(shaftX2, bowCy); ctx.stroke();
                var toothH = sz * 0.18;
                var t1x = shaftX1 + (shaftX2 - shaftX1) * 0.45;
                var t2x = shaftX1 + (shaftX2 - shaftX1) * 0.72;
                ctx.beginPath();
                ctx.moveTo(t1x, bowCy); ctx.lineTo(t1x, bowCy + toothH);
                ctx.moveTo(t2x, bowCy); ctx.lineTo(t2x, bowCy + toothH);
                ctx.stroke();
            }
            ctx.restore();
        }

        function render() {
            if (!ctx || !canvas) { return; }
            ctx.clearRect(0, 0, canvas.width, canvas.height);
            ctx.fillStyle = MAP_BACKGROUND;
            ctx.fillRect(0, 0, canvas.width, canvas.height);

            ctx.strokeStyle = CONNECTION_COLOR;
            ctx.lineWidth   = CONNECTION_WIDTH * zoomScale;
            ctx.lineCap     = 'round';

            edges.forEach(function (flags, key) {
                var parts = key.split('-');
                var rA = rooms.get(parseInt(parts[0], 10));
                var rB = rooms.get(parseInt(parts[1], 10));
                if (!rA || !rB) { return; }
                var pA = gridToCanvas(rA.x, rA.y), pB = gridToCanvas(rB.x, rB.y);
                ctx.beginPath(); ctx.moveTo(pA.px, pA.py); ctx.lineTo(pB.px, pB.py); ctx.stroke();
                if (flags.locked || flags.secret) {
                    drawLineBadge((pA.px + pB.px) / 2, (pA.py + pB.py) / 2,
                                  flags.secret ? 'secret' : 'key');
                }
            });

            var stubLen = BASE_STEP * zoomScale * 0.55;
            zoneExitStubs.forEach(function (stub) {
                var r = rooms.get(stub.roomId);
                if (!r) { return; }
                var p = gridToCanvas(r.x, r.y);
                var len = Math.sqrt(stub.dx * stub.dx + stub.dy * stub.dy);
                if (len === 0) { return; }
                var ex = p.px + (stub.dx / len) * stubLen;
                var ey = p.py + (stub.dy / len) * stubLen;
                ctx.beginPath(); ctx.moveTo(p.px, p.py); ctx.lineTo(ex, ey); ctx.stroke();
                if (stub.locked || stub.secret) {
                    drawLineBadge((p.px + ex) / 2, (p.py + ey) / 2, stub.secret ? 'secret' : 'key');
                }
            });

            var scaledSize   = ROOM_SIZE        * zoomScale;
            var scaledBorder = ROOM_BORDER_WIDTH * zoomScale;
            var scaledFont   = SYMBOL_FONT_SIZE  * zoomScale;
            var half         = scaledSize / 2;

            rooms.forEach(function (room, id) {
                var p         = gridToCanvas(room.x, room.y);
                var isCurrent = (id === currentRoomId);
                var fill      = isCurrent ? CURRENT_ROOM_COLOR : colorForSymbol(room.symbol, room.env);
                var rx = p.px - half, ry = p.py - half;
                ctx.fillStyle = fill; ctx.fillRect(rx, ry, scaledSize, scaledSize);
                ctx.strokeStyle = ROOM_BORDER_COLOR; ctx.lineWidth = scaledBorder;
                ctx.strokeRect(rx, ry, scaledSize, scaledSize);
                ctx.fillStyle    = isCurrent ? CURRENT_ROOM_TEXT_COLOR : SYMBOL_TEXT_COLOR;
                ctx.font         = 'bold ' + scaledFont + 'px monospace';
                ctx.textAlign    = 'center'; ctx.textBaseline = 'middle';
                ctx.fillText(room.symbol || '\u2022', p.px, p.py);
                if (room.hasUp || room.hasDown) {
                    var arrowSize = Math.max(5, scaledSize * 0.28);
                    var margin    = Math.max(2, scaledSize * 0.1);
                    ctx.font      = 'bold ' + arrowSize + 'px monospace';
                    ctx.fillStyle = isCurrent ? CURRENT_ROOM_TEXT_COLOR : SYMBOL_TEXT_COLOR;
                    if (room.hasDown) {
                        ctx.textAlign = 'left'; ctx.textBaseline = 'alphabetic';
                        ctx.fillText('\u25be', rx + margin, ry + scaledSize - margin);
                    }
                    if (room.hasUp) {
                        ctx.textAlign = 'right'; ctx.textBaseline = 'top';
                        ctx.fillText('\u25b4', rx + scaledSize - margin, ry + margin);
                    }
                }
            });
        }

        function roomAtPoint(cx, cy) {
            var half = (ROOM_SIZE * zoomScale) / 2;
            for (var [id, room] of rooms) {
                var p = gridToCanvas(room.x, room.y);
                if (cx >= p.px - half && cx <= p.px + half &&
                    cy >= p.py - half && cy <= p.py + half) { return id; }
            }
            return null;
        }

        // -- DOM ---------------------------------------------------------------
        function createPanel() {
            var wrap = document.createElement('div');
            wrap.className = 'map-canvas-wrap';

            canvas = document.createElement('canvas');
            canvas.id = 'map-2d-canvas';
            wrap.appendChild(canvas);
            ctx = canvas.getContext('2d');

            canvas.addEventListener('mouseleave', function () {
                hideTooltip();
                if (dragActive) { dragActive = false; canvas.style.cursor = ''; }
            });
            canvas.addEventListener('mousedown', function (e) {
                if (e.button !== 0) { return; }
                dragActive = true;
                dragStartPxX = e.clientX; dragStartPxY = e.clientY;
                dragStartPanX = panOffsetX; dragStartPanY = panOffsetY;
                canvas.style.cursor = 'grabbing'; e.preventDefault();
            });
            canvas.addEventListener('mousemove', function (e) {
                var rect = canvas.getBoundingClientRect();
                if (dragActive) {
                    var step = BASE_STEP * zoomScale;
                    panOffsetX = dragStartPanX - (e.clientX - dragStartPxX) / step;
                    panOffsetY = dragStartPanY - (e.clientY - dragStartPxY) / step;
                    render(); return;
                }
                var id   = roomAtPoint(e.clientX - rect.left, e.clientY - rect.top);
                var info = id !== null ? roomInfoStore.get(id) : null;
                if (info) { clearTimeout(tooltipHideTimer); showTooltip(e.clientX, e.clientY, info, false); }
                else      { hideTooltip(); }
            });
            canvas.addEventListener('mouseup', function (e) {
                if (!dragActive) { return; }
                var dx = e.clientX - dragStartPxX, dy = e.clientY - dragStartPxY;
                dragActive = false; canvas.style.cursor = '';
                if (Math.abs(dx) > 4 || Math.abs(dy) > 4) { canvas.dataset.suppressClick = '1'; }
            });
            canvas.addEventListener('click', function (e) {
                if (canvas.dataset.suppressClick) { delete canvas.dataset.suppressClick; return; }
                var charInfo = Client.GMCPStructs.Char && Client.GMCPStructs.Char.Info;
                if (!charInfo || charInfo.role !== 'admin') { return; }
                var rect = canvas.getBoundingClientRect();
                var id   = roomAtPoint(e.clientX - rect.left, e.clientY - rect.top);
                if (id === null) { return; }
                e.stopPropagation();
                uiMenu(e, [{ label: 'teleport ' + id, cmd: 'teleport ' + id },
                            { label: 'room info ' + id, cmd: 'room info ' + id }]);
            });
            canvas.addEventListener('wheel', function (e) {
                e.preventDefault();
                var factor = Math.pow(ZOOM_STEP, e.deltaY * 0.002);
                zoomScale = Math.min(ZOOM_MAX, Math.max(ZOOM_MIN, zoomScale / factor));
                render();
            }, { passive: false });

            var controls = document.createElement('div');
            controls.className = 'map-controls';
            var btnOut = document.createElement('button');
            btnOut.textContent = '\u2212'; btnOut.title = 'Zoom out';
            btnOut.addEventListener('click', function () {
                zoomScale = Math.max(ZOOM_MIN, zoomScale / ZOOM_STEP); render();
            });
            var btnIn = document.createElement('button');
            btnIn.textContent = '+'; btnIn.title = 'Zoom in';
            btnIn.addEventListener('click', function () {
                zoomScale = Math.min(ZOOM_MAX, zoomScale * ZOOM_STEP); render();
            });
            controls.appendChild(btnOut);
            controls.appendChild(btnIn);
            wrap.appendChild(controls);

            container = wrap;
            return wrap;
        }

        function onActivate() {
            resizeCanvas();
            render();
        }

        function onWorldMap() {
            resizeCanvas();
            if (currentZoneKey) {
                var savedId = currentRoomId;
                replayZone(currentZoneKey);
                currentRoomId = savedId;
            }
            render();
        }

        function onRoomUpdate(info, gx, gy, gz, sym, env) {
            resizeCanvas();
            var zoneKey = info.coords.split(',').map(function (s) { return s.trim(); })[0] + '/z:' + gz;
            if (currentZoneKey !== zoneKey) {
                currentZoneKey = zoneKey;
                replayZone(zoneKey);
            } else {
                addOrUpdateRoom(info.num, gx, gy, sym, env);
                var rc = roomCache[info.num];
                if (rc) {
                    if (Array.isArray(rc.exits)) {
                        rc.exits.forEach(function (exit) {
                            if (exit.dz !== 0) { return; }
                            var destRc = roomCache[exit.num];
                            if (destRc) {
                                if (!rooms.has(exit.num)) {
                                    addOrUpdateRoom(exit.num, destRc.x, destRc.y, destRc.symbol, destRc.env);
                                }
                                addEdge(info.num, exit.num, exit.locked, exit.secret);
                            }
                        });
                    }
                    if (Array.isArray(rc.stubs)) {
                        rc.stubs.forEach(function (stub) {
                            if (stub.dz === 0) {
                                zoneExitStubs.push({ roomId: info.num, dx: stub.dx, dy: stub.dy,
                                                     locked: stub.locked, secret: stub.secret });
                            }
                        });
                    }
                }
            }
            currentRoomId = info.num;
            setCameraTarget(gx, gy);
        }

        function setupResizeObserver(win) {
            if (typeof ResizeObserver === 'undefined') { return; }
            var ro = new ResizeObserver(function () { resizeCanvas(); render(); });
            var orig = win.open.bind(win);
            win.open = function () { orig(); if (container) { ro.observe(container); } };
        }

        return {
            createPanel:        createPanel,
            onActivate:         onActivate,
            onWorldMap:         onWorldMap,
            onRoomUpdate:       onRoomUpdate,
            setupResizeObserver: setupResizeObserver,
            getCurrentRoomId:   function () { return currentRoomId; },
        };

    }());

    // =========================================================================
    // 3D view
    // =========================================================================

    var view3d = (function () {

        // -- Constants ---------------------------------------------------------
        var TILE_HW          = 20;   // Half-width of a tile diamond in px; larger = wider tiles, more horizontal spread
        var TILE_HH          = 10;   // Half-height of a tile diamond in px; larger = taller diamonds, more vertical compression
        var TILE_DEPTH       = 7;    // Height of the visible block side below the tile face; larger = taller 3D extrusion
        var GRID_STEP_XY     = 1.6;  // Spacing multiplier between adjacent XY grid positions relative to TILE_HW; increase to spread rooms apart horizontally, decrease to compress them
        var Z_STEP           = 50;   // Screen pixels of vertical separation per z-level at default spacing; increase to push layers further apart vertically
        var Z_SPACING_EXPONENT = 1.5; // Controls how aggressively z-layer separation grows as spacing increases; 1.0 = linear with spacing, 2.0 = quadratic (very dramatic), lower = subtler
        var CONNECTION_WIDTH = 2;    // Stroke width of lines connecting rooms; thicker = more visible corridors
        var MAP_BG           = '#000000';    // Canvas fill color behind all tiles; change to adjust overall contrast
        var TILE_BORDER_COLOR = '#000000'; // Outline color drawn around each tile face; darker = crisper tile edges
        var TILE_BORDER_WIDTH = 0.8;  // Stroke width of tile outlines; higher = bolder edges, can obscure small tiles at low zoom
        var SIDE_DARKEN       = 0.55; // Brightness multiplier for the left and right block faces; lower = darker sides, stronger 3D illusion
        var SYMBOL_FONT_SIZE  = 11;   // Font size of the symbol character on each tile; larger = more readable but may overflow the tile face
        var SPACING_STEP = 1.25; // Multiplier applied per spacing button click; higher = bigger jumps between spacing levels
        var SPACING_MIN  = 0.6;  // Minimum spacing scale; lower values compress rooms closer together until tiles overlap
        var SPACING_MAX  = 4.0;  // Maximum spacing scale; higher values spread rooms and z-layers further apart
        var ALPHA_INACTIVE  = 0.0; // Opacity of rooms and edges on z-levels other than the active one; lower = more faded
        var ALPHA_CONNECTED = 0.30; // Opacity of rooms on inactive z-levels that have a direct connection to the active layer; higher = more visible cross-z neighbours
        var LAYER_OFFSET_X  = 0;   // Screen pixels to shift each z-level horizontally per level away from the active layer; increase to spread layers apart left/right
        var LAYER_OFFSET_Y  = 0;   // Screen pixels to shift each z-level vertically per level away from the active layer; increase to spread layers apart up/down
        var CONNECTION_COLOR_SAME_Z  = '#ffffff'; // Color of lines connecting rooms on the same z-level; change to distinguish same-floor corridors
        var CONNECTION_COLOR_CROSS_Z = '#3a6b8a'; // Color of lines connecting rooms on different z-levels; change to make vertical connections stand out
        var CROSS_Z_OFFSET_X = 8;   // Horizontal pixel offset from tile center where cross-z connection lines attach; shift left/right to reposition the vertical passage indicator
        var CROSS_Z_OFFSET_Y = 0;   // Vertical pixel offset from tile center where cross-z connection lines attach; shift up/down to reposition the vertical passage indicator
        var CROSS_Z_ARROW_SIZE = 6; // Size of the arrowhead drawn on cross-z connection lines in px at zoom 1; larger = more prominent direction indicator

        // -- State -------------------------------------------------------------
        var canvas    = null;
        var ctx       = null;
        var container = null;
        var rooms3d   = new Map();
        var edges3d   = new Map();
        var bucketCache = null;
        var currentRoomId  = null;
        var camX = 0, camY = 0, camZ = 0;
        var easeStartX = 0, easeStartY = 0, easeStartZ = 0;
        var easeTargetX = 0, easeTargetY = 0, easeTargetZ = 0;
        var easeStartTime = null, easeRafId = null;
        var panOffsetX = 0, panOffsetY = 0;
        var dragActive = false;
        var dragStartPxX = 0, dragStartPxY = 0;
        var dragStartPanX = 0, dragStartPanY = 0;
        var zoomScale    = 1.0;
        var activeZ       = null;
        var currentRoomKey = '';
        var zLevelsEl    = null;

        var spacingScale = (function () {
            var saved = parseFloat(localStorage.getItem('map3d.spacingScale'));
            return (isFinite(saved) && saved >= SPACING_MIN && saved <= SPACING_MAX) ? saved : SPACING_MIN;
        }());

        // -- Helpers -----------------------------------------------------------
        function resizeCanvas() {
            if (!canvas || !container) { return; }
            canvas.width  = container.clientWidth  || 1;
            canvas.height = container.clientHeight || 1;
        }

        function darkenColor(hex, factor) {
            var r = Math.min(255, Math.round(parseInt(hex.slice(1, 3), 16) * factor));
            var g = Math.min(255, Math.round(parseInt(hex.slice(3, 5), 16) * factor));
            var b = Math.min(255, Math.round(parseInt(hex.slice(5, 7), 16) * factor));
            return '#' + ('0' + r.toString(16)).slice(-2) +
                         ('0' + g.toString(16)).slice(-2) +
                         ('0' + b.toString(16)).slice(-2);
        }

        function isoProject(gx, gy, gz, drawZ) {
            var step = TILE_HW * GRID_STEP_XY * spacingScale * zoomScale;
            var zs   = Z_STEP  * Math.pow(spacingScale, Z_SPACING_EXPONENT) * zoomScale;
            var midX = Math.floor(canvas.width  / 2);
            var midY = Math.floor(canvas.height / 2);
            var relX = gx - camX - panOffsetX;
            var relY = gy - camY - panOffsetY;
            var relZ = gz - camZ;
            var layerDiff = (drawZ !== undefined) ? (gz - drawZ) : 0;
            return {
                sx: midX + (relX - relY) * step + layerDiff * LAYER_OFFSET_X * zoomScale,
                sy: midY + (relX + relY) * (step / 2) - relZ * zs + layerDiff * LAYER_OFFSET_Y * zoomScale,
            };
        }

        function setCameraTarget(tx, ty, tz) {
            panOffsetX = 0; panOffsetY = 0;
            if (CENTER_EASE_DURATION <= 0) { camX = tx; camY = ty; camZ = tz; render(); return; }
            if (easeRafId !== null) { cancelAnimationFrame(easeRafId); easeRafId = null; }
            easeStartX = camX; easeStartY = camY; easeStartZ = camZ;
            easeTargetX = tx; easeTargetY = ty; easeTargetZ = tz;
            easeStartTime = null;
            function step(ts) {
                if (easeStartTime === null) { easeStartTime = ts; }
                var t = Math.min((ts - easeStartTime) / 1000 / CENTER_EASE_DURATION, 1);
                var s = smoothstep(t);
                camX = easeStartX + (easeTargetX - easeStartX) * s;
                camY = easeStartY + (easeTargetY - easeStartY) * s;
                camZ = easeStartZ + (easeTargetZ - easeStartZ) * s;
                render();
                easeRafId = t < 1 ? requestAnimationFrame(step) : null;
            }
            easeRafId = requestAnimationFrame(step);
        }

        function tileAttachPoint(sx, sy, dx, dy) {
            var hw = TILE_HW * zoomScale, hh = TILE_HH * zoomScale;
            if (dx !== 0 && dy !== 0) {
                if (dx > 0 && dy > 0) { return { sx: sx,      sy: sy + hh }; }
                if (dx > 0 && dy < 0) { return { sx: sx + hw, sy: sy      }; }
                if (dx < 0 && dy > 0) { return { sx: sx - hw, sy: sy      }; }
                return { sx: sx, sy: sy - hh };
            }
            if (dx > 0) { return { sx: sx + hw / 2, sy: sy + hh / 2 }; }
            if (dx < 0) { return { sx: sx - hw / 2, sy: sy - hh / 2 }; }
            if (dy > 0) { return { sx: sx - hw / 2, sy: sy + hh / 2 }; }
            if (dy < 0) { return { sx: sx + hw / 2, sy: sy - hh / 2 }; }
            return { sx: sx, sy: sy };
        }

        function addRoom3d(id, gx, gy, gz, symbol, env) {
            rooms3d.set(id, { x: gx, y: gy, z: gz, symbol: symbol || '\u2022', env: env || '' });
            bucketCache = null;
        }

        function addEdge3d(idA, idB, rA, rB) {
            var key, dx, dy, dz;
            if (idA < idB) {
                key = idA + '-' + idB;
                dx = rB.x - rA.x; dy = rB.y - rA.y; dz = rB.z - rA.z;
            } else {
                key = idB + '-' + idA;
                dx = rA.x - rB.x; dy = rA.y - rB.y; dz = rA.z - rB.z;
            }
            if (!edges3d.has(key)) { edges3d.set(key, { dx: dx, dy: dy, dz: dz }); bucketCache = null; }
        }

        function resetMap3d() {
            rooms3d.clear(); edges3d.clear();
            bucketCache = null;
            currentRoomId = null;
            panOffsetX = 0; panOffsetY = 0;
            dragActive = false;
            if (easeRafId !== null) { cancelAnimationFrame(easeRafId); easeRafId = null; }
        }

        function buildBucketCache() {
            var zSet = new Set();
            rooms3d.forEach(function (room) { zSet.add(room.z); });
            var zLevels = Array.from(zSet).sort(function (a, b) { return a - b; });
            var roomsByZ = {};
            var sameZEdgesByZ = {};
            var crossZUpEdgesByZ = {};
            zLevels.forEach(function (z) {
                roomsByZ[z] = [];
                sameZEdgesByZ[z] = [];
                crossZUpEdgesByZ[z] = [];
            });
            rooms3d.forEach(function (room, id) {
                roomsByZ[room.z].push({ id: id, x: room.x, y: room.y, z: room.z,
                                        symbol: room.symbol, env: room.env });
            });
            zLevels.forEach(function (z) {
                roomsByZ[z].sort(function (a, b) { return (a.x + a.y) - (b.x + b.y); });
            });
            edges3d.forEach(function (edge, key) {
                var parts = key.split('-');
                var rA = rooms3d.get(parseInt(parts[0], 10));
                var rB = rooms3d.get(parseInt(parts[1], 10));
                if (!rA || !rB) { return; }
                if (edge.dz === 0) {
                    if (sameZEdgesByZ[rA.z]) { sameZEdgesByZ[rA.z].push({ edge: edge, rA: rA, rB: rB }); }
                } else {
                    var lowerZ = Math.min(rA.z, rB.z);
                    if (crossZUpEdgesByZ[lowerZ]) { crossZUpEdgesByZ[lowerZ].push({ edge: edge, rA: rA, rB: rB }); }
                }
            });
            bucketCache = { zLevels: zLevels, roomsByZ: roomsByZ,
                            sameZEdgesByZ: sameZEdgesByZ, crossZUpEdgesByZ: crossZUpEdgesByZ };
        }

        function replayZone3d(startId) {
            resetMap3d();
            if (!roomCache[startId]) { return; }
            var startRoom = roomCache[startId];
            var zoneKey = startRoom.zoneName;
            var visited = {}, queue = [];
            for (var rid in roomCache) {
                var rc = roomCache[rid];
                if (rc.zoneName === zoneKey) { queue.push(parseInt(rid, 10)); }
            }
            while (queue.length > 0) {
                var id = queue.shift();
                if (visited[id]) { continue; }
                visited[id] = true;
                var r = roomCache[id];
                if (!r) { continue; }
                addRoom3d(r.RoomId, r.x, r.y, r.z, r.symbol, r.env);
                if (Array.isArray(r.exits)) {
                    r.exits.forEach(function (exit) {
                        if (!visited[exit.num] && roomCache[exit.num]) { queue.push(exit.num); }
                    });
                }
            }
            rooms3d.forEach(function (room, id) {
                var r = roomCache[id];
                if (!r || !Array.isArray(r.exits)) { return; }
                r.exits.forEach(function (exit) {
                    if (rooms3d.has(exit.num)) { addEdge3d(id, exit.num, r, roomCache[exit.num]); }
                });
            });
        }

        // -- Rendering ---------------------------------------------------------
        function drawTile(gx, gy, gz, topColor, isCurrent, symbol, drawZ) {
            var hw  = TILE_HW    * zoomScale;
            var hh  = TILE_HH    * zoomScale;
            var dep = TILE_DEPTH * zoomScale;
            var bw  = TILE_BORDER_WIDTH * zoomScale;
            var p   = isoProject(gx, gy, gz, drawZ);
            var sx  = p.sx, sy = p.sy;
            var leftColor  = darkenColor(topColor, SIDE_DARKEN * 0.8);
            var rightColor = darkenColor(topColor, SIDE_DARKEN);

            ctx.beginPath();
            ctx.moveTo(sx, sy - hh); ctx.lineTo(sx + hw, sy);
            ctx.lineTo(sx, sy + hh); ctx.lineTo(sx - hw, sy);
            ctx.closePath();
            ctx.fillStyle = topColor; ctx.fill();
            ctx.strokeStyle = TILE_BORDER_COLOR; ctx.lineWidth = bw; ctx.stroke();

            ctx.beginPath();
            ctx.moveTo(sx - hw, sy); ctx.lineTo(sx, sy + hh);
            ctx.lineTo(sx, sy + hh + dep); ctx.lineTo(sx - hw, sy + dep);
            ctx.closePath();
            ctx.fillStyle = leftColor; ctx.fill();
            ctx.strokeStyle = TILE_BORDER_COLOR; ctx.lineWidth = bw; ctx.stroke();

            ctx.beginPath();
            ctx.moveTo(sx, sy + hh); ctx.lineTo(sx + hw, sy);
            ctx.lineTo(sx + hw, sy + dep); ctx.lineTo(sx, sy + hh + dep);
            ctx.closePath();
            ctx.fillStyle = rightColor; ctx.fill();
            ctx.strokeStyle = TILE_BORDER_COLOR; ctx.lineWidth = bw; ctx.stroke();

            ctx.fillStyle    = isCurrent ? CURRENT_ROOM_TEXT_COLOR : SYMBOL_TEXT_COLOR;
            ctx.font         = 'bold ' + (SYMBOL_FONT_SIZE * zoomScale) + 'px monospace';
            ctx.textAlign    = 'center'; ctx.textBaseline = 'middle';
            ctx.fillText(symbol || '\u2022', sx, sy);
        }

        function render() {
            if (!ctx || !canvas) { return; }
            ctx.clearRect(0, 0, canvas.width, canvas.height);
            ctx.fillStyle = MAP_BG; ctx.fillRect(0, 0, canvas.width, canvas.height);
            if (rooms3d.size === 0) { return; }

            var playerZ = (currentRoomId !== null && rooms3d.has(currentRoomId))
                ? rooms3d.get(currentRoomId).z : (camZ | 0);
            var drawZ = (activeZ !== null) ? activeZ : playerZ;

            // Build a set of room IDs on other z-levels that connect directly to the active layer.
            var connectedToActive = new Set();
            edges3d.forEach(function (edge, key) {
                if (edge.dz === 0) { return; }
                var parts = key.split('-');
                var rA = rooms3d.get(parseInt(parts[0], 10));
                var rB = rooms3d.get(parseInt(parts[1], 10));
                if (!rA || !rB) { return; }
                if (rA.z === drawZ && rB.z !== drawZ) { connectedToActive.add(parseInt(parts[1], 10)); }
                if (rB.z === drawZ && rA.z !== drawZ) { connectedToActive.add(parseInt(parts[0], 10)); }
            });

            if (!bucketCache) { buildBucketCache(); }
            var zLevels         = bucketCache.zLevels;
            var roomsByZ        = bucketCache.roomsByZ;
            var sameZEdgesByZ   = bucketCache.sameZEdgesByZ;
            var crossZUpEdgesByZ = bucketCache.crossZUpEdgesByZ;

            ctx.lineWidth = CONNECTION_WIDTH * zoomScale;
            ctx.lineCap   = 'round';

            function drawEdge(e) {
                var pA = isoProject(e.rA.x, e.rA.y, e.rA.z, drawZ);
                var pB = isoProject(e.rB.x, e.rB.y, e.rB.z, drawZ);
                var startPt, endPt;
                if (e.edge.dz !== 0) {
                    startPt = { sx: pA.sx + CROSS_Z_OFFSET_X * zoomScale, sy: pA.sy + CROSS_Z_OFFSET_Y * zoomScale };
                    endPt   = { sx: pB.sx + CROSS_Z_OFFSET_X * zoomScale, sy: pB.sy + CROSS_Z_OFFSET_Y * zoomScale };
                } else {
                    startPt = tileAttachPoint(pA.sx, pA.sy,  e.edge.dx,  e.edge.dy);
                    endPt   = tileAttachPoint(pB.sx, pB.sy, -e.edge.dx, -e.edge.dy);
                }
                ctx.beginPath(); ctx.moveTo(startPt.sx, startPt.sy);
                ctx.lineTo(endPt.sx, endPt.sy); ctx.stroke();

                if (e.edge.dz !== 0) {
                    var dx = endPt.sx - startPt.sx;
                    var dy = endPt.sy - startPt.sy;
                    var len = Math.sqrt(dx * dx + dy * dy);
                    if (len > 0) {
                        var ux = dx / len, uy = dy / len;
                        var as = CROSS_Z_ARROW_SIZE * zoomScale;
                        ctx.fillStyle = ctx.strokeStyle;
                        // Arrowhead at endPt pointing away from startPt.
                        ctx.beginPath();
                        ctx.moveTo(endPt.sx, endPt.sy);
                        ctx.lineTo(endPt.sx - ux * as - uy * as, endPt.sy - uy * as + ux * as);
                        ctx.lineTo(endPt.sx - ux * as + uy * as, endPt.sy - uy * as - ux * as);
                        ctx.closePath(); ctx.fill();
                        // Arrowhead at startPt pointing away from endPt.
                        ctx.beginPath();
                        ctx.moveTo(startPt.sx, startPt.sy);
                        ctx.lineTo(startPt.sx + ux * as - uy * as, startPt.sy + uy * as + ux * as);
                        ctx.lineTo(startPt.sx + ux * as + uy * as, startPt.sy + uy * as - ux * as);
                        ctx.closePath(); ctx.fill();
                    }
                }
            }

            // Draw each z-level in order: same-z edges, then tiles, then cross-z-up edges.
            zLevels.forEach(function (z) {
                var zDiffFromActive = Math.abs(z - drawZ);
                var baseAlpha = zDiffFromActive === 0 ? 1.0 : ALPHA_INACTIVE;

                // 1. Same-z edges for this layer.
                ctx.strokeStyle = CONNECTION_COLOR_SAME_Z;
                sameZEdgesByZ[z].forEach(function (e) {
                    ctx.globalAlpha = baseAlpha;
                    drawEdge(e);
                });

                // 2. Tiles for this layer.
                roomsByZ[z].forEach(function (item) {
                    var isCurrent = (item.id === currentRoomId);
                    var topColor  = isCurrent ? CURRENT_ROOM_COLOR : colorForSymbol(item.symbol, item.env);
                    var onActive  = zDiffFromActive === 0;
                    ctx.globalAlpha = onActive ? 1.0 : (connectedToActive.has(item.id) ? ALPHA_CONNECTED : ALPHA_INACTIVE);
                    drawTile(item.x, item.y, item.z, topColor, isCurrent, item.symbol, drawZ);
                });

                // 3. Cross-z-up edges leaving this layer (drawn over this layer's tiles, under the next).
                ctx.strokeStyle = CONNECTION_COLOR_CROSS_Z;
                crossZUpEdgesByZ[z].forEach(function (e) {
                    var zDiff = Math.min(Math.abs(e.rA.z - drawZ), Math.abs(e.rB.z - drawZ));
                    ctx.globalAlpha = zDiff === 0 ? 1.0 : ALPHA_INACTIVE;
                    drawEdge(e);
                });
            });

            ctx.globalAlpha = 1.0;
        }

        function getActiveZ() {
            if (activeZ !== null) { return activeZ; }
            return (currentRoomId !== null && rooms3d.has(currentRoomId))
                ? rooms3d.get(currentRoomId).z : (camZ | 0);
        }

        function roomAtPoint(cx, cy) {
            var step = TILE_HW * GRID_STEP_XY * spacingScale * zoomScale;
            var hw = step, hh = step / 2, found = null;
            var targetZ = getActiveZ();
            var list = [];
            rooms3d.forEach(function (room, id) {
                if (room.z === targetZ) { list.push({ id: id, x: room.x, y: room.y, z: room.z }); }
            });
            list.sort(function (a, b) { return (b.x + b.y - b.z * 2) - (a.x + a.y - a.z * 2); });
            for (var i = 0; i < list.length; i++) {
                var item = list[i];
                var p = isoProject(item.x, item.y, item.z);
                if (Math.abs(cx - p.sx) / hw + Math.abs(cy - p.sy) / hh <= 1) {
                    found = item.id; break;
                }
            }
            return found;
        }

        // -- DOM ---------------------------------------------------------------
        function createPanel() {
            var wrap = document.createElement('div');
            wrap.className = 'map-canvas-wrap';

            canvas = document.createElement('canvas');
            canvas.id = 'map-3d-canvas';
            wrap.appendChild(canvas);
            ctx = canvas.getContext('2d');

            canvas.addEventListener('mouseleave', function () {
                hideTooltip();
                if (dragActive) { dragActive = false; canvas.style.cursor = ''; }
            });
            canvas.addEventListener('mousedown', function (e) {
                if (e.button !== 0) { return; }
                dragActive = true;
                dragStartPxX = e.clientX; dragStartPxY = e.clientY;
                dragStartPanX = panOffsetX; dragStartPanY = panOffsetY;
                canvas.style.cursor = 'grabbing'; e.preventDefault();
            });
            canvas.addEventListener('mousemove', function (e) {
                var rect = canvas.getBoundingClientRect();
                if (dragActive) {
                    var step = TILE_HW * GRID_STEP_XY * spacingScale * zoomScale;
                    var dsx = e.clientX - dragStartPxX, dsy = e.clientY - dragStartPxY;
                    panOffsetX = dragStartPanX - (dsx / step + dsy * 2 / step) / 2;
                    panOffsetY = dragStartPanY - (dsy * 2 / step - dsx / step) / 2;
                    render(); return;
                }
                var id   = roomAtPoint(e.clientX - rect.left, e.clientY - rect.top);
                var info = id !== null ? roomInfoStore.get(id) : null;
                if (info) {
                    clearTimeout(tooltipHideTimer);
                    showTooltip(e.clientX, e.clientY, info, true);
                } else {
                    hideTooltip();
                }
            });
            canvas.addEventListener('mouseup', function (e) {
                if (!dragActive) { return; }
                var dx = e.clientX - dragStartPxX, dy = e.clientY - dragStartPxY;
                dragActive = false; canvas.style.cursor = '';
                if (Math.abs(dx) > 4 || Math.abs(dy) > 4) { canvas.dataset.suppressClick = '1'; }
            });
            canvas.addEventListener('click', function (e) {
                if (canvas.dataset.suppressClick) { delete canvas.dataset.suppressClick; return; }
                var charInfo = Client.GMCPStructs.Char && Client.GMCPStructs.Char.Info;
                if (!charInfo || charInfo.role !== 'admin') { return; }
                var rect = canvas.getBoundingClientRect();
                var id   = roomAtPoint(e.clientX - rect.left, e.clientY - rect.top);
                if (id === null) { return; }
                e.stopPropagation();
                uiMenu(e, [{ label: 'teleport ' + id, cmd: 'teleport ' + id },
                            { label: 'room info ' + id, cmd: 'room info ' + id }]);
            });
            canvas.addEventListener('wheel', function (e) {
                e.preventDefault();
                var factor = Math.pow(ZOOM_STEP, e.deltaY * 0.002);
                zoomScale = Math.min(ZOOM_MAX, Math.max(ZOOM_MIN, zoomScale / factor));
                render();
            }, { passive: false });

            var controls = document.createElement('div');
            controls.className = 'map-controls';

            var btnSpacingDown = document.createElement('button');
            btnSpacingDown.textContent = '\u2193'; btnSpacingDown.title = 'Decrease spacing';
            btnSpacingDown.addEventListener('click', function () {
                spacingScale = Math.max(SPACING_MIN, spacingScale / SPACING_STEP);
                localStorage.setItem('map3d.spacingScale', spacingScale); render();
            });
            var btnSpacingUp = document.createElement('button');
            btnSpacingUp.textContent = '\u2191'; btnSpacingUp.title = 'Increase spacing';
            btnSpacingUp.addEventListener('click', function () {
                spacingScale = Math.min(SPACING_MAX, spacingScale * SPACING_STEP);
                localStorage.setItem('map3d.spacingScale', spacingScale); render();
            });
            var sep = document.createElement('span');
            sep.className = 'ctrl-sep';
            var btnZoomOut = document.createElement('button');
            btnZoomOut.textContent = '\u2212'; btnZoomOut.title = 'Zoom out';
            btnZoomOut.addEventListener('click', function () {
                zoomScale = Math.max(ZOOM_MIN, zoomScale / ZOOM_STEP); render();
            });
            var btnZoomIn = document.createElement('button');
            btnZoomIn.textContent = '+'; btnZoomIn.title = 'Zoom in';
            btnZoomIn.addEventListener('click', function () {
                zoomScale = Math.min(ZOOM_MAX, zoomScale * ZOOM_STEP); render();
            });

            controls.appendChild(btnSpacingDown);
            controls.appendChild(btnSpacingUp);
            controls.appendChild(sep);
            controls.appendChild(btnZoomOut);
            controls.appendChild(btnZoomIn);
            wrap.appendChild(controls);

            zLevelsEl = document.createElement('div');
            zLevelsEl.className = 'map-z-levels';
            zLevelsEl.style.display = 'none';
            wrap.appendChild(zLevelsEl);

            container = wrap;
            return wrap;
        }

        function updateZButtons() {
            if (!zLevelsEl) { return; }
            var levels = Array.from(new Set(Array.from(rooms3d.values()).map(function (r) { return r.z; }))).sort(function (a, b) { return b - a; });

            zLevelsEl.innerHTML = '';
            if (levels.length <= 1) { zLevelsEl.style.display = 'none'; return; }

            var playerZ = (currentRoomId !== null && rooms3d.has(currentRoomId))
                ? rooms3d.get(currentRoomId).z : null;
            var currentActiveZ = (activeZ !== null) ? activeZ : playerZ;

            levels.forEach(function (z) {
                var btn = document.createElement('button');
                btn.textContent = z;
                btn.title = 'Z level ' + z;
                if (z === currentActiveZ) { btn.classList.add('active'); }
                btn.addEventListener('click', function () {
                    activeZ = z;
                    updateZButtons();
                    render();
                });
                zLevelsEl.appendChild(btn);
            });

            zLevelsEl.style.display = 'flex';
        }

        function onActivate() {
            resizeCanvas();
            render();
        }

        function onWorldMap() {
            resizeCanvas();
            if (currentRoomId && roomCache[currentRoomId]) {
                var savedId = currentRoomId;
                replayZone3d(savedId);
                currentRoomId = savedId;
            }
            updateZButtons();
            render();
        }

        function onRoomUpdate(info, gx, gy, gz, sym, env) {
            resizeCanvas();
            var roomKey = info.num + '';
            if (currentRoomKey !== roomKey) {
                currentRoomKey = roomKey;
                currentRoomId  = info.num;
                replayZone3d(info.num);
            } else {
                addRoom3d(info.num, gx, gy, gz, sym, env);
                var rc = roomCache[info.num];
                if (rc && Array.isArray(rc.exits)) {
                    rc.exits.forEach(function (exit) {
                        var destRc = roomCache[exit.num];
                        if (destRc) {
                            if (!rooms3d.has(exit.num)) {
                                addRoom3d(exit.num, destRc.x, destRc.y, destRc.z, destRc.symbol, destRc.env);
                            }
                            addEdge3d(info.num, exit.num, roomCache[info.num], destRc);
                        }
                    });
                }
            }
            currentRoomId = info.num;
            activeZ = gz;
            updateZButtons();
            setCameraTarget(gx, gy, gz);
        }

        function setupResizeObserver(win) {
            if (typeof ResizeObserver === 'undefined') { return; }
            var ro = new ResizeObserver(function () { resizeCanvas(); render(); });
            var orig = win.open.bind(win);
            win.open = function () { orig(); if (container) { ro.observe(container); } };
        }

        return {
            createPanel:         createPanel,
            onActivate:          onActivate,
            onWorldMap:          onWorldMap,
            onRoomUpdate:        onRoomUpdate,
            setupResizeObserver: setupResizeObserver,
            updateZButtons:      updateZButtons,
        };

    }());

    // =========================================================================
    // Window DOM - tab bar + two panels
    // =========================================================================

    var activeTab   = localStorage.getItem('map.activeTab') === '3d' ? '3d' : '2d';
    var panel2d     = null;
    var panel3d     = null;
    var tabBtn2d    = null;
    var tabBtn3d    = null;

    function switchTab(tab) {
        activeTab = tab;
        localStorage.setItem('map.activeTab', tab);
        panel2d.classList.toggle('active', tab === '2d');
        panel3d.classList.toggle('active', tab === '3d');
        tabBtn2d.classList.toggle('active', tab === '2d');
        tabBtn3d.classList.toggle('active', tab === '3d');
        if (tab === '2d') { view2d.onActivate(); }
        else              { view3d.onActivate(); }
    }

    function createDOM() {
        var root = document.createElement('div');
        root.id = 'map-window';

        var tabBar = document.createElement('div');
        tabBar.id = 'map-tab-bar';

        tabBtn2d = document.createElement('button');
        tabBtn2d.textContent = '2D';
        tabBtn2d.addEventListener('click', function () { switchTab('2d'); });

        tabBtn3d = document.createElement('button');
        tabBtn3d.textContent = '3D';
        tabBtn3d.addEventListener('click', function () { switchTab('3d'); });

        tabBar.appendChild(tabBtn2d);
        tabBar.appendChild(tabBtn3d);
        root.appendChild(tabBar);

        var panels = document.createElement('div');
        panels.id = 'map-panels';

        panel2d = document.createElement('div');
        panel2d.className = 'map-panel';
        panel2d.appendChild(view2d.createPanel());

        panel3d = document.createElement('div');
        panel3d.className = 'map-panel';
        panel3d.appendChild(view3d.createPanel());

        panels.appendChild(panel2d);
        panels.appendChild(panel3d);
        root.appendChild(panels);

        document.body.appendChild(root);

        // Apply initial active tab without triggering resize before DOM is ready
        panel2d.classList.toggle('active', activeTab === '2d');
        panel3d.classList.toggle('active', activeTab === '3d');
        tabBtn2d.classList.toggle('active', activeTab === '2d');
        tabBtn3d.classList.toggle('active', activeTab === '3d');

        return root;
    }

    // =========================================================================
    // VirtualWindow
    // =========================================================================

    var win = new VirtualWindow('Map', {
        dock:          'right',
        defaultDocked: true,
        dockedHeight:  363,
        factory: function () {
            var el = createDOM();
            return {
                title:      'Map',
                mount:      el,
                background: 'var(--t-bg-panel)',
                border:     1,
                x:          'right',
                y:          66,
                width:      363,
                height:     20 + 363,
                header:     20,
                bottom:     60,
            };
        },
    });

    view2d.setupResizeObserver(win);
    view3d.setupResizeObserver(win);

    // =========================================================================
    // GMCP update logic
    // =========================================================================

    function updateWorldMap() {
        var worldData = Client.GMCPStructs.World;
        if (!worldData || !Array.isArray(worldData.Map)) { return; }
        win.open();
        if (!win.isOpen()) { return; }
        ingestWorldMap(worldData.Map);
    }

    function updateMap() {
        var obj = Client.GMCPStructs.Room;
        if (!obj || !obj.Info) { return; }
        win.open();
        if (!win.isOpen()) { return; }

        if (!worldMapRequested) {
            worldMapRequested = true;
            Client.GMCPRequest('World.Map');
        }

        var info     = obj.Info;
        var coords   = info.coords.split(',').map(function (s) { return s.trim(); });
        var gx       = parseInt(coords[1], 10);
        var gy       = parseInt(coords[2], 10);
        var gz       = parseInt(coords[3], 10);
        var sym      = symbolForRoom(info);
        var env      = info.environment || '';

        roomInfoStore.set(info.num, info);
        upsertRoomCache(info.num, coords[0], gx, gy, gz, sym, env, info.exitsv2);

        var winBox = win.get();
        if (winBox) { winBox.setTitle('map (' + info.area + ')'); }

        view2d.onRoomUpdate(info, gx, gy, gz, sym, env);
        view3d.onRoomUpdate(info, gx, gy, gz, sym, env);
    }

    // =========================================================================
    // Registration
    // =========================================================================

    VirtualWindows.register({
        window:       win,
        gmcpHandlers: ['Room', 'World'],
        onGMCP: function (namespace) {
            if (namespace === 'World.Map') {
                updateWorldMap();
            } else if (namespace === 'Room.Info' || namespace === 'Room') {
                updateMap();
            }
        },
    });

}());
