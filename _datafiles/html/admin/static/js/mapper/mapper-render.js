/* jshint esversion: 11, browser: true */
/* globals MapperState, MapperTools, MapperEvents,
   ZOOM_STEP, ZOOM_MIN, ZOOM_MAX, CENTER_EASE_DURATION,
   ROOM_SIZE_2D, ROOM_GAP_2D, BASE_STEP_2D, CONNECTION_WIDTH_2D, ROOM_BORDER_WIDTH_2D, SYMBOL_FONT_SIZE_2D, MAP_BG_2D, ROOM_BORDER_COLOR_2D,
   CONNECTION_COLOR, ABNORMAL_CONNECTION_COLOR, SELECTED_ROOM_COLOR, SELECTED_ROOM_TEXT_COLOR, SYMBOL_TEXT_COLOR,
   ZONE_BOX_PADDING, ZONE_BOX_COLOR, ZONE_BOX_COLOR_HOV, ZONE_BOX_BORDER, ZONE_BOX_BORDER_HOV,
   ROOM_ARROW_COLOR, ROOM_ARROW_STROKE_COLOR, ROOM_ARROW_STROKE_WIDTH,
   ROOM_BORDER_MOB_SPAWN, ROOM_BORDER_SCRIPT_GLOW, ROOM_BORDER_TAGS, ROOM_BORDER_UNSAVED, BADGE_SECRET_COLOR, BADGE_LOCK_COLOR,
   GHOST_CELL_BORDER, GHOST_CELL_FILL, GHOST_CELL_SYMBOL,
   EXIT_DRAW_TARGET_HIGHLIGHT, EXIT_DRAW_LINE_COLOR,
   DRAG_ORIGIN_MARKER, DRAG_SNAP_BLOCKED, DRAG_SNAP_BROKEN, DRAG_SNAP_CLEAN, DRAG_CONSTRAINT_BROKEN, DRAG_CONSTRAINT_OK, DRAG_GHOST_BROKEN_FILL,
   QB_COLOR, QB_OCCUPIED_COLOR, SELECT_RECT_FILL, SELECT_RECT_BORDER,
   bgColorForBiome,
   computeZonePaddedBounds,
   exitDelta, isDirectionalExit, darkenColor, smoothstep, isExitConstraintSatisfied */

/**
 * MapperRender — canvas drawing engine for the admin map editor (2D only).
 *
 * Public API (returned IIFE object):
 *   Setup          — setCanvas, initResizeObserver, resizeCanvas
 *   Rendering      — render, getRenderState
 *   Coord helpers  — gridToCanvas2d, canvasToGrid2d, canvasToGrid
 *   Hit testing    — roomAtPoint, roomAtPoint2d, currentZ, gridCellOccupied
 *   Drawing prims  — drawRoom2d, drawLineBadge2d
 */
var MapperRender = (function() {
    'use strict';

    // --- Canvas References ---

    var canvas = null;
    var ctx = null;

    function setCanvas(c) {
        canvas = c;
        ctx = c.getContext('2d');
    }

    // --- Per-frame transform cache ---
    // Refreshed once at the top of render2d; consumed by gridToCanvas2d,
    // canvasToGrid2d, and all inline callers so the values are never
    // recomputed mid-frame.

    var _step  = 1;
    var _midX  = 0;
    var _midY  = 0;
    var _camX  = 0;  // cam.cameraX + cam.panOffsetX
    var _camY  = 0;  // cam.cameraY + cam.panOffsetY

    function _refreshTransformCache() {
        var cam = MapperState.camera;
        _step = BASE_STEP_2D * cam.spacingScale2d * cam.zoomScale;
        _midX = Math.floor(canvas.width  / 2);
        _midY = Math.floor(canvas.height / 2);
        _camX = cam.cameraX + cam.panOffsetX;
        _camY = cam.cameraY + cam.panOffsetY;
    }

    // --- Coordinate Transforms: 2D ---

    function gridToCanvas2d(gx, gy) {
        return {
            px: _midX + (gx - _camX) * _step,
            py: _midY + (gy - _camY) * _step
        };
    }

    function canvasToGrid2d(cx, cy) {
        return {
            gx: Math.round((cx - _midX) / _step + _camX),
            gy: Math.round((cy - _midY) / _step + _camY)
        };
    }

    function canvasToGrid(cx, cy) {
        // Called from event handlers outside render2d — ensure cache is fresh.
        _refreshTransformCache();
        return canvasToGrid2d(cx, cy);
    }

    // --- Hit Testing ---

    function roomAtPoint(cx, cy) {
        return roomAtPoint2d(cx, cy);
    }

    function roomAtPoint2d(cx, cy) {
        // Refresh the transform cache: this function is called from event
        // handlers outside of render2d, so we cannot rely on render2d having
        // run first.
        _refreshTransformCache();
        var cam = MapperState.camera;
        var half = (ROOM_SIZE_2D * cam.zoomScale) / 2;
        var g = canvasToGrid2d(cx, cy);
        var id = MapperState.data.roomsByCoord.get(g.gx + ',' + g.gy + ',' + cam.activeZ2d);
        if (id === undefined) return null;
        var room = MapperState.data.rooms.get(id);
        if (!room || !room.HasCoordinates) return null;
        var p = gridToCanvas2d(room.MapX, room.MapY);
        if (cx >= p.px - half && cx <= p.px + half && cy >= p.py - half && cy <= p.py + half) {
            return id;
        }
        return null;
    }

    // --- Current Z / Grid Occupancy ---

    function currentZ() {
        return MapperState.camera.activeZ2d;
    }

    function gridCellOccupied(gx, gy, gz) {
        return MapperState.data.roomsByCoord.has(gx + ',' + gy + ',' + gz);
    }

    // --- 2D Drawing Primitives ---

    /** Draws all deferred line badges in two batched passes (secret then key),
     *  setting canvas state once per type instead of once per badge. */
    function drawLineBadges2d(badges) {
        var cam = MapperState.camera;
        var sz = Math.max(7, Math.round(CONNECTION_WIDTH_2D * cam.zoomScale * 2.5));
        var half = sz / 2;

        // Partition by type
        var secrets = [], keys = [];
        for (var i = 0; i < badges.length; i++) {
            if (badges[i].type === 'secret') secrets.push(badges[i]);
            else keys.push(badges[i]);
        }

        // --- Secret badges ---
        if (secrets.length > 0) {
            ctx.save();
            ctx.font = 'bold ' + Math.round(sz * 0.85) + 'px monospace';
            ctx.textAlign = 'center';
            ctx.textBaseline = 'middle';
            for (var si = 0; si < secrets.length; si++) {
                var s = secrets[si];
                ctx.fillStyle = MAP_BG_2D;
                ctx.fillRect(s.mx - half, s.my - half, sz, sz);
                ctx.fillStyle = BADGE_SECRET_COLOR;
                ctx.fillText('?', s.mx, s.my);
            }
            ctx.restore();
        }

        // --- Key (lock) badges ---
        if (keys.length > 0) {
            ctx.save();
            var kc = BADGE_LOCK_COLOR;
            var lw = Math.max(1, sz * 0.14);
            ctx.strokeStyle = kc;
            ctx.fillStyle = kc;
            ctx.lineWidth = lw;
            ctx.lineCap = 'round';
            var bowR = sz * 0.22;
            var bowOffX = -sz * 0.14;
            var toothH = sz * 0.18;
            for (var ki = 0; ki < keys.length; ki++) {
                var k = keys[ki];
                ctx.fillStyle = MAP_BG_2D;
                ctx.fillRect(k.mx - half, k.my - half, sz, sz);
                ctx.fillStyle = kc;
                var bowCx = k.mx + bowOffX;
                ctx.beginPath(); ctx.arc(bowCx, k.my, bowR, 0, Math.PI * 2); ctx.stroke();
                var shaftX1 = bowCx + bowR, shaftX2 = k.mx + half * 0.82;
                ctx.beginPath(); ctx.moveTo(shaftX1, k.my); ctx.lineTo(shaftX2, k.my); ctx.stroke();
                var t1x = shaftX1 + (shaftX2 - shaftX1) * 0.45;
                var t2x = shaftX1 + (shaftX2 - shaftX1) * 0.72;
                ctx.beginPath();
                ctx.moveTo(t1x, k.my); ctx.lineTo(t1x, k.my + toothH);
                ctx.moveTo(t2x, k.my); ctx.lineTo(t2x, k.my + toothH);
                ctx.stroke();
            }
            ctx.restore();
        }
    }

    /** Draws a small badge (secret "?" or key icon) at the midpoint of a connection line. */
    function drawLineBadge2d(mx, my, type) {
        var cam = MapperState.camera;
        var sz = Math.max(7, Math.round(CONNECTION_WIDTH_2D * cam.zoomScale * 2.5));
        var half = sz / 2;

        ctx.save();
        ctx.fillStyle = MAP_BG_2D;
        ctx.fillRect(mx - half, my - half, sz, sz);

        if (type === 'secret') {
            ctx.fillStyle = BADGE_SECRET_COLOR;
            ctx.font = 'bold ' + Math.round(sz * 0.85) + 'px monospace';
            ctx.textAlign = 'center'; ctx.textBaseline = 'middle';
            ctx.fillText('?', mx, my);
        } else {
            var kc = BADGE_LOCK_COLOR, lw = Math.max(1, sz * 0.14);
            ctx.strokeStyle = kc; ctx.fillStyle = kc;
            ctx.lineWidth = lw; ctx.lineCap = 'round';
            var bowR = sz * 0.22, bowCx = mx - sz * 0.14;
            ctx.beginPath(); ctx.arc(bowCx, my, bowR, 0, Math.PI * 2); ctx.stroke();
            var shaftX1 = bowCx + bowR, shaftX2 = mx + half * 0.82;
            ctx.beginPath(); ctx.moveTo(shaftX1, my); ctx.lineTo(shaftX2, my); ctx.stroke();
            var toothH = sz * 0.18;
            var t1x = shaftX1 + (shaftX2 - shaftX1) * 0.45;
            var t2x = shaftX1 + (shaftX2 - shaftX1) * 0.72;
            ctx.beginPath();
            ctx.moveTo(t1x, my); ctx.lineTo(t1x, my + toothH);
            ctx.moveTo(t2x, my); ctx.lineTo(t2x, my + toothH);
            ctx.stroke();
        }
        ctx.restore();
    }

    /**
     * Draws a single room tile in 2D: filled square, border, and symbol.
     * Sets ctx.font, ctx.textAlign, and ctx.textBaseline itself so it is safe
     * to call as a standalone primitive (e.g. from tool overlays).
     * Script-glow and Z-arrows are handled in separate deferred passes by render2d.
     *
     * Returns { hasScript, hasUp, hasDown } so the caller can collect rooms
     * that need the deferred passes without re-iterating.
     */
    function drawRoom2d(p, room, id) {
        var cam = MapperState.camera;
        var scaledSize = ROOM_SIZE_2D * cam.zoomScale;
        var scaledBorder = ROOM_BORDER_WIDTH_2D * cam.zoomScale;
        var scaledFont = SYMBOL_FONT_SIZE_2D * cam.zoomScale;
        var half = scaledSize / 2;

        var isSelected = MapperState.selected.has(id);
        var fill = isSelected ? SELECTED_ROOM_COLOR : (room._bgColor || room._color);
        var rx = p.px - half, ry = p.py - half;

        ctx.fillStyle = fill;
        ctx.fillRect(rx, ry, scaledSize, scaledSize);

        if (!isSelected && room.HasMobSpawn) {
            ctx.strokeStyle = ROOM_BORDER_MOB_SPAWN;
        } else {
            ctx.strokeStyle = isSelected ? SELECTED_ROOM_COLOR : ROOM_BORDER_COLOR_2D;
        }
        ctx.lineWidth = scaledBorder;
        ctx.strokeRect(rx, ry, scaledSize, scaledSize);

        var symColor = isSelected ? SELECTED_ROOM_TEXT_COLOR
            : (room._bgColor ? room._color : (room._symbolColor || SYMBOL_TEXT_COLOR));
        ctx.fillStyle = symColor;
        ctx.font = 'bold ' + scaledFont + 'px monospace';
        ctx.textAlign = 'center';
        ctx.textBaseline = 'middle';
        ctx.fillText(room._symbol || '•', p.px, p.py);

        return {
            hasScript: !isSelected && room.HasScript,
            hasUp: false,
            hasDown: false
        };
    }

    // --- Zone Bounding Boxes ---

    /** Draws faint dashed zone bounding boxes behind all rooms. The zone
     *  containing the currently hovered room is drawn solid. */
    function drawZoneBounds2d(rooms, activeZ, hoveredRoomId) {
        var cam = MapperState.camera;

        // Determine hovered zone
        var hoveredZone = null;
        if (hoveredRoomId !== null) {
            var hr = rooms.get(hoveredRoomId);
            if (hr) hoveredZone = hr.Zone;
        }

        // Compute per-zone padded bounds in grid space (gaps between zones respected)
        var zoneBounds = computeZonePaddedBounds(rooms, activeZ);

        ctx.save();
        ctx.lineJoin = 'round';

        for (var zone in zoneBounds) {
            var b = zoneBounds[zone];
            var isHov = zone === hoveredZone;

            // Convert padded grid bounds to canvas space
            var pMin = gridToCanvas2d(b.minX, b.minY);
            var pMax = gridToCanvas2d(b.maxX, b.maxY);
            var rx = pMin.px;
            var ry = pMin.py;
            var rw = pMax.px - pMin.px;
            var rh = pMax.py - pMin.py;

            // Fill
            ctx.fillStyle = isHov ? ZONE_BOX_COLOR_HOV : ZONE_BOX_COLOR;
            ctx.fillRect(rx, ry, rw, rh);

            // Border
            ctx.strokeStyle = isHov ? ZONE_BOX_BORDER_HOV : ZONE_BOX_BORDER;
            ctx.lineWidth = Math.max(1, cam.zoomScale);
            if (!isHov) {
                ctx.setLineDash([Math.max(3, 6 * cam.zoomScale), Math.max(3, 6 * cam.zoomScale)]);
            } else {
                ctx.setLineDash([]);
            }
            ctx.strokeRect(rx, ry, rw, rh);
            ctx.setLineDash([]);

            // Zone name label in top-left corner
            var fontSize = Math.max(9, 11 * cam.zoomScale);
            ctx.font = fontSize + 'px monospace';
            ctx.fillStyle = isHov ? 'rgba(200,200,255,0.9)' : 'rgba(180,180,220,0.5)';
            ctx.textAlign = 'left';
            ctx.textBaseline = 'top';
            ctx.fillText(zone, rx + 4, ry + 3);
        }

        ctx.restore();
    }

    // --- Tool Overlay Dispatch ---

    function getRenderState() {
        var cam = MapperState.camera;
        return {
            ctx: ctx, canvas: canvas, zoomScale: cam.zoomScale,
            activeTab: cam.activeTab, activeZ2d: cam.activeZ2d,
            selectedRoomIds: MapperState.selected,
            hoveredRoomId: MapperState.hoveredRoomId,
            hoveredGridCell: MapperState.hoveredGridCell,
            gridToCanvas2d: gridToCanvas2d,
            canvasToGrid2d: canvasToGrid2d,
            canvasToGrid: canvasToGrid,
            gridCellOccupied: gridCellOccupied,
            drawRoom2d: drawRoom2d,
            scaledSize: ROOM_SIZE_2D * cam.zoomScale,
            scaledBorder: ROOM_BORDER_WIDTH_2D * cam.zoomScale,
            scaledFont: SYMBOL_FONT_SIZE_2D * cam.zoomScale,
            half: (ROOM_SIZE_2D * cam.zoomScale) / 2
        };
    }

    function renderToolOverlays2d() {
        var rs = getRenderState();
        var tools = MapperTools.all();
        for (var name in tools) {
            if (tools[name] && typeof tools[name].renderOverlay2d === 'function') {
                tools[name].renderOverlay2d(ctx, rs);
            }
        }
    }

    // --- Reusable per-frame collections ---
    // Allocated once; cleared at the start of each render2d call to avoid
    // per-frame GC pressure from repeated new Set() / [] allocations.

    var _drawnEdges    = new Set();
    var _abnormalEdges = [];
    var _deferredBadges = [];
    var _glowRooms     = [];   // { p, room, rx, ry, scaledSize, scaledBorder }
    var _tagRooms      = [];   // { rx, ry, scaledSize, scaledBorder }
    var _arrowRooms    = [];   // { p, room, rx, ry, scaledSize }
    var _unsavedRooms  = [];   // { rx, ry, scaledSize, scaledBorder } — temp (negative-ID) rooms

    // --- 2D Core Renderer ---

    function render2d() {
        var cam = MapperState.camera;
        var data = MapperState.data;

        ctx.clearRect(0, 0, canvas.width, canvas.height);
        ctx.fillStyle = MAP_BG_2D;
        ctx.fillRect(0, 0, canvas.width, canvas.height);

        var rooms = data.rooms;
        if (rooms.size === 0) { renderToolOverlays2d(); return; }

        // Refresh the per-frame transform cache once so gridToCanvas2d /
        // canvasToGrid2d never recompute step/mid/cam values mid-frame.
        _refreshTransformCache();

        var zoneOnly = cam.selectedZoneOnly ? data.currentZone : null;

        // Zone bounding boxes (drawn first, behind everything)
        if (MapperState.camera.showBounds) {
            drawZoneBounds2d(rooms, cam.activeZ2d, MapperState.hoveredRoomId);
        }

        // --- Viewport culling bounds (in grid units) ---
        // Skip rooms whose grid cell centre is outside the visible area plus a
        // 2-cell margin so rooms near the edge always draw their connections.
        var margin = 2;
        var invStep = 1 / _step;
        var vMinGx = Math.floor(( 0             - _midX) * invStep + _camX) - margin;
        var vMaxGx = Math.ceil( ( canvas.width  - _midX) * invStep + _camX) + margin;
        var vMinGy = Math.floor(( 0             - _midY) * invStep + _camY) - margin;
        var vMaxGy = Math.ceil( ( canvas.height - _midY) * invStep + _camY) + margin;

        ctx.lineCap = 'round';

        // Clear reusable collections
        _drawnEdges.clear();
        _abnormalEdges.length = 0;
        _deferredBadges.length = 0;
        _glowRooms.length = 0;
        _tagRooms.length = 0;
        _arrowRooms.length = 0;
        _unsavedRooms.length = 0;

        // Pass 1: normal directional edges
        ctx.strokeStyle = CONNECTION_COLOR;
        ctx.lineWidth = CONNECTION_WIDTH_2D * cam.zoomScale;
        rooms.forEach(function(room, id) {
            if (!room.HasCoordinates || room.MapZ !== cam.activeZ2d) return;
            if (zoneOnly && room.Zone !== zoneOnly) return;
            if (room.MapX < vMinGx || room.MapX > vMaxGx || room.MapY < vMinGy || room.MapY > vMaxGy) return;
            if (!room.Exits) return;
            for (var dir in room.Exits) {
                var ex = room.Exits[dir];
                var dest = rooms.get(ex.RoomId);
                if (!dest || !dest.HasCoordinates) continue;
                if (zoneOnly && dest.Zone !== zoneOnly) continue;
                var key = Math.min(id, ex.RoomId) + '-' + Math.max(id, ex.RoomId) + ':' + dir;
                if (_drawnEdges.has(key)) continue;
                _drawnEdges.add(key);
                var delta = exitDelta(dir, room);
                var directional = isDirectionalExit(dir);
                if (!directional || !delta) {
                    _abnormalEdges.push({ room: room, dest: dest, dir: dir, ex: ex });
                    continue;
                }
                if (delta[2] !== 0) continue;
                if (dest.MapZ !== cam.activeZ2d) continue;
                var pA = gridToCanvas2d(room.MapX, room.MapY);
                var pB = gridToCanvas2d(dest.MapX, dest.MapY);
                ctx.beginPath(); ctx.moveTo(pA.px, pA.py); ctx.lineTo(pB.px, pB.py); ctx.stroke();
                if (ex.Secret || ex.HasLock) {
                    _deferredBadges.push({ mx: (pA.px + pB.px) / 2, my: (pA.py + pB.py) / 2, type: ex.Secret ? 'secret' : 'key' });
                }
            }
        });

        // Pass 2: abnormal edges (yellow dotted arcs)
        if (_abnormalEdges.length > 0) {
            ctx.strokeStyle = ABNORMAL_CONNECTION_COLOR;
            ctx.lineWidth = Math.max(1, CONNECTION_WIDTH_2D * cam.zoomScale * .5);
            ctx.setLineDash([Math.max(3, 4 * cam.zoomScale), Math.max(4, 5 * cam.zoomScale)]);
            for (var ai = 0; ai < _abnormalEdges.length; ai++) {
                var ae = _abnormalEdges[ai];
                var pA2 = gridToCanvas2d(ae.room.MapX, ae.room.MapY);
                var pB2 = gridToCanvas2d(ae.dest.MapX, ae.dest.MapY);
                var mx = (pA2.px + pB2.px) / 2, my = (pA2.py + pB2.py) / 2;
                var dx = pB2.px - pA2.px, dy = pB2.py - pA2.py;
                var dist = Math.sqrt(dx * dx + dy * dy);
                var bulge = Math.max(15, dist * 0.25);
                var cpx = mx + (-dy / dist) * bulge;
                var cpy = my + (dx / dist) * bulge;
                ctx.beginPath(); ctx.moveTo(pA2.px, pA2.py);
                ctx.quadraticCurveTo(cpx, cpy, pB2.px, pB2.py); ctx.stroke();
            }
            ctx.setLineDash([]);
        }

        // Pass 3: rooms
        // Set shared text state once for all room symbols (avoids per-room font assignment).
        var scaledFont = SYMBOL_FONT_SIZE_2D * cam.zoomScale;
        ctx.font = 'bold ' + scaledFont + 'px monospace';
        ctx.textAlign = 'center';
        ctx.textBaseline = 'middle';

        var dragGroup = MapperState.roomDrag;

        rooms.forEach(function(room, id) {
            if (!room.HasCoordinates || room.MapZ !== cam.activeZ2d) return;
            if (zoneOnly && room.Zone !== zoneOnly) return;
            if (room.MapX < vMinGx || room.MapX > vMaxGx || room.MapY < vMinGy || room.MapY > vMaxGy) return;
            if (dragGroup.active && dragGroup.group.has(id)) return;

            var p = gridToCanvas2d(room.MapX, room.MapY);
            var scaledSize = ROOM_SIZE_2D * cam.zoomScale;
            var scaledBorder = ROOM_BORDER_WIDTH_2D * cam.zoomScale;
            var half = scaledSize / 2;
            var rx = p.px - half, ry = p.py - half;

            var isSelected = MapperState.selected.has(id);
            var fill = isSelected ? SELECTED_ROOM_COLOR : (room._bgColor || room._color);

            ctx.fillStyle = fill;
            ctx.fillRect(rx, ry, scaledSize, scaledSize);

            if (!isSelected && room.HasMobSpawn && cam.showMobBorder) {
                ctx.strokeStyle = ROOM_BORDER_MOB_SPAWN;
            } else {
                ctx.strokeStyle = isSelected ? SELECTED_ROOM_COLOR : ROOM_BORDER_COLOR_2D;
            }
            ctx.lineWidth = scaledBorder;
            ctx.strokeRect(rx, ry, scaledSize, scaledSize);

            var symColor = isSelected ? SELECTED_ROOM_TEXT_COLOR
                : (room._bgColor ? room._color : (room._symbolColor || SYMBOL_TEXT_COLOR));
            ctx.fillStyle = symColor;
            ctx.fillText(room._symbol || '•', p.px, p.py);

            // Collect rooms that need deferred passes rather than doing them inline
            if (!isSelected && room.HasScript && cam.showScriptBorder) {
                _glowRooms.push({ p: p, scaledSize: scaledSize, scaledBorder: scaledBorder, rx: rx, ry: ry });
            }

            if (!isSelected && room.Tags && room.Tags.length > 0 && cam.showTagsBorder) {
                _tagRooms.push({ scaledSize: scaledSize, scaledBorder: scaledBorder, rx: rx, ry: ry });
            }

            if (room.Exits) {
                var hasUp = false, hasDown = false;
                for (var dir in room.Exits) {
                    var delta = exitDelta(dir, room);
                    if (delta && delta[2] > 0) hasUp = true;
                    if (delta && delta[2] < 0) hasDown = true;
                }
                if (hasUp || hasDown) {
                    _arrowRooms.push({ p: p, scaledSize: scaledSize, rx: rx, ry: ry, hasUp: hasUp, hasDown: hasDown });
                }
            }

            if (!isSelected && id < 0) {
                _unsavedRooms.push({ rx: rx, ry: ry, scaledSize: scaledSize, scaledBorder: scaledBorder });
            }
        });

        // Pass 4: tags border — one save/restore for all rooms with tags
        if (_tagRooms.length > 0) {
            ctx.save();
            ctx.strokeStyle = ROOM_BORDER_TAGS;
            for (var ti = 0; ti < _tagRooms.length; ti++) {
                var tr = _tagRooms[ti];
                var tagOffset = tr.scaledBorder * 2;
                ctx.lineWidth = tr.scaledBorder;
                ctx.strokeRect(tr.rx - tagOffset, tr.ry - tagOffset, tr.scaledSize + tagOffset * 2, tr.scaledSize + tagOffset * 2);
            }
            ctx.restore();
        }

        // Pass 5: script glow — one save/restore for all glowing rooms
        if (_glowRooms.length > 0 && cam.showScriptBorder) {
            ctx.save();
            ctx.shadowColor = ROOM_BORDER_SCRIPT_GLOW;
            ctx.strokeStyle = ROOM_BORDER_SCRIPT_GLOW;
            for (var gi = 0; gi < _glowRooms.length; gi++) {
                var gr = _glowRooms[gi];
                var offset = gr.scaledBorder * 1;
                ctx.lineWidth = gr.scaledBorder;
                ctx.strokeRect(gr.rx - offset, gr.ry - offset, gr.scaledSize + offset * 2, gr.scaledSize + offset * 2);
            }
            ctx.restore();
        }

        // Pass 6: Z-arrows — one save/restore for all arrow rooms
        if (_arrowRooms.length > 0) {
            var arrowSize = Math.max(10, ROOM_SIZE_2D * cam.zoomScale * 0.56);
            var useStroke = ROOM_ARROW_STROKE_COLOR && ROOM_ARROW_STROKE_WIDTH > 0;
            ctx.save();
            ctx.font = 'bold ' + arrowSize + 'px monospace';
            ctx.fillStyle = ROOM_ARROW_COLOR;
            if (useStroke) {
                ctx.strokeStyle = ROOM_ARROW_STROKE_COLOR;
                ctx.lineWidth = ROOM_ARROW_STROKE_WIDTH * cam.zoomScale;
                ctx.lineJoin = 'round';
            }
            for (var zi = 0; zi < _arrowRooms.length; zi++) {
                var ar = _arrowRooms[zi];
                var margin = Math.max(2, ar.scaledSize * 0.1);
                if (ar.hasDown) {
                    ctx.textAlign = 'left'; ctx.textBaseline = 'alphabetic';
                    if (useStroke) ctx.strokeText('▾', ar.rx + margin, ar.ry + ar.scaledSize - margin);
                    ctx.fillText('▾', ar.rx + margin, ar.ry + ar.scaledSize - margin);
                }
                if (ar.hasUp) {
                    ctx.textAlign = 'right'; ctx.textBaseline = 'top';
                    if (useStroke) ctx.strokeText('▴', ar.rx + ar.scaledSize - margin, ar.ry - margin);
                    ctx.fillText('▴', ar.rx + ar.scaledSize - margin, ar.ry - margin);
                }
            }
            ctx.restore();
        }

        if (_deferredBadges.length > 0) {
            drawLineBadges2d(_deferredBadges);
        }

        // Pass 7: unsaved (pending) rooms -- dashed amber border drawn on top
        if (_unsavedRooms.length > 0) {
            ctx.save();
            ctx.strokeStyle = ROOM_BORDER_UNSAVED;
            ctx.setLineDash([Math.max(3, 3 * cam.zoomScale), Math.max(6, 6 * cam.zoomScale)]);
            for (var ui = 0; ui < _unsavedRooms.length; ui++) {
                var ur = _unsavedRooms[ui];
                ctx.lineWidth = Math.max(1, ur.scaledBorder * 0.85);
                ctx.strokeRect(ur.rx, ur.ry, ur.scaledSize, ur.scaledSize);
            }
            ctx.setLineDash([]);
            ctx.restore();
        }

        renderToolOverlays2d();
    }

    // --- Render Dispatch ---

    var _renderScheduled = false;

    function scheduleRender() {
        if (_renderScheduled) return;
        _renderScheduled = true;
        requestAnimationFrame(function() {
            _renderScheduled = false;
            render();
        });
    }

    function render() {
        render2d();
    }

    // --- Canvas Resize ---

    var viewport = null;

    function resizeCanvas() {
        if (!canvas) return;
        if (!viewport) viewport = canvas.parentElement;
        canvas.width = (viewport ? viewport.clientWidth : window.innerWidth) || 1;
        canvas.height = (viewport ? viewport.clientHeight : window.innerHeight) || 1;
    }

    var resizeObserver = null;
    function initResizeObserver() {
        if (typeof ResizeObserver === 'undefined') return;
        if (!canvas) return;
        viewport = canvas.parentElement;
        if (!viewport) return;
        resizeObserver = new ResizeObserver(function() { resizeCanvas(); scheduleRender(); });
        resizeObserver.observe(viewport);
    }

    return {
        setCanvas: setCanvas,
        initResizeObserver: initResizeObserver,
        resizeCanvas: resizeCanvas,
        render: render,
        scheduleRender: scheduleRender,
        getRenderState: getRenderState,
        gridToCanvas2d: gridToCanvas2d,
        canvasToGrid2d: canvasToGrid2d,
        canvasToGrid: canvasToGrid,
        roomAtPoint: roomAtPoint,
        roomAtPoint2d: roomAtPoint2d,
        currentZ: currentZ,
        gridCellOccupied: gridCellOccupied,
        drawRoom2d: drawRoom2d,
        drawLineBadge2d: drawLineBadge2d
    };

})();
