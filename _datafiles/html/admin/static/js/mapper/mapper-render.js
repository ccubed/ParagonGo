/* jshint esversion: 11, browser: true */
/* globals MapperState, MapperTools, MapperEvents,
   ZOOM_STEP, ZOOM_MIN, ZOOM_MAX, CENTER_EASE_DURATION,
   ROOM_SIZE_2D, ROOM_GAP_2D, BASE_STEP_2D, CONNECTION_WIDTH_2D, ROOM_BORDER_WIDTH_2D, SYMBOL_FONT_SIZE_2D, MAP_BG_2D, ROOM_BORDER_COLOR_2D,
   TILE_HW_3D, TILE_HH_3D, TILE_DEPTH_3D, GRID_STEP_XY_3D, Z_STEP_3D, Z_SPACING_EXP_3D, CONNECTION_WIDTH_3D, MAP_BG_3D, TILE_BORDER_COLOR_3D, TILE_BORDER_WIDTH_3D, SIDE_DARKEN_3D, SYMBOL_FONT_SIZE_3D, SPACING_MIN_3D, SPACING_MAX_3D,
   ALPHA_INACTIVE_3D, ALPHA_CONNECTED_3D, CONN_COLOR_SAME_Z, CONN_COLOR_CROSS_Z, CROSS_Z_OFFSET_X, CROSS_Z_ARROW_SIZE,
   CONNECTION_COLOR, ABNORMAL_CONNECTION_COLOR, SELECTED_ROOM_COLOR, SELECTED_ROOM_TEXT_COLOR, SYMBOL_TEXT_COLOR,
   exitDelta, isDirectionalExit, darkenColor, smoothstep, isExitConstraintSatisfied */

/**
 * MapperRender — canvas drawing engine for the admin map editor.
 *
 * Handles all coordinate transforms (2D orthogonal, 3D isometric), hit
 * testing, and the two core render paths (render2d / render3d).  Tool
 * modules contribute visual overlays via the renderOverlay2d/3d protocol
 * dispatched at the end of each frame.
 *
 * Public API (returned IIFE object):
 *   Setup          — setCanvas, initResizeObserver, resizeCanvas
 *   Rendering      — render, getRenderState
 *   Coord helpers  — gridToCanvas2d, canvasToGrid2d, canvasToGrid3d,
 *                    canvasToGrid, isoProject3d
 *   Hit testing    — roomAtPoint, roomAtPoint2d, roomAtPoint3d,
 *                    currentZ, gridCellOccupied
 *   Drawing prims  — drawRoom2d, drawTile3d, drawLineBadge2d
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

    // --- Coordinate Transforms: 2D ---

    function gridToCanvas2d(gx, gy) {
        var cam = MapperState.camera;
        var midX = Math.floor(canvas.width / 2);
        var midY = Math.floor(canvas.height / 2);
        var step = BASE_STEP_2D * cam.zoomScale;
        return {
            px: midX + (gx - cam.cameraX - cam.panOffsetX) * step,
            py: midY + (gy - cam.cameraY - cam.panOffsetY) * step
        };
    }

    function canvasToGrid2d(cx, cy) {
        var cam = MapperState.camera;
        var midX = Math.floor(canvas.width / 2);
        var midY = Math.floor(canvas.height / 2);
        var step = BASE_STEP_2D * cam.zoomScale;
        return {
            gx: Math.round((cx - midX) / step + cam.cameraX + cam.panOffsetX),
            gy: Math.round((cy - midY) / step + cam.cameraY + cam.panOffsetY)
        };
    }

    // --- Coordinate Transforms: 3D (Isometric) ---

    function canvasToGrid3d(cx, cy) {
        var cam = MapperState.camera;
        var step = TILE_HW_3D * GRID_STEP_XY_3D * cam.spacingScale3d * cam.zoomScale;
        var zs = Z_STEP_3D * Math.pow(cam.spacingScale3d, Z_SPACING_EXP_3D) * cam.zoomScale;
        var midX = Math.floor(canvas.width / 2);
        var midY = Math.floor(canvas.height / 2);
        var drawZ = cam.activeZ3d !== null ? cam.activeZ3d : (MapperState.data.zLevels.length > 0 ? MapperState.data.zLevels[0] : 0);

        // Reverse the isometric projection: undo z-offset then solve the
        // 2x2 system for grid-x and grid-y.
        var relZ = drawZ - cam.cameraZ;
        var sxOff = cx - midX;
        var syAdj = cy - midY + relZ * zs;
        var halfStep = step / 2;
        var relX = (sxOff / step + syAdj / halfStep) / 2;
        var relY = (syAdj / halfStep - sxOff / step) / 2;
        return {
            gx: Math.round(relX + cam.cameraX + cam.panOffsetX),
            gy: Math.round(relY + cam.cameraY + cam.panOffsetY)
        };
    }

    function canvasToGrid(cx, cy) {
        return MapperState.camera.activeTab === '2d' ? canvasToGrid2d(cx, cy) : canvasToGrid3d(cx, cy);
    }

    function isoProject3d(gx, gy, gz, drawZ) {
        var cam = MapperState.camera;
        var step = TILE_HW_3D * GRID_STEP_XY_3D * cam.spacingScale3d * cam.zoomScale;
        var zs = Z_STEP_3D * Math.pow(cam.spacingScale3d, Z_SPACING_EXP_3D) * cam.zoomScale;
        var midX = Math.floor(canvas.width / 2);
        var midY = Math.floor(canvas.height / 2);
        var relX = gx - cam.cameraX - cam.panOffsetX;
        var relY = gy - cam.cameraY - cam.panOffsetY;
        var relZ = gz - cam.cameraZ;
        return {
            sx: midX + (relX - relY) * step,
            sy: midY + (relX + relY) * (step / 2) - relZ * zs
        };
    }

    // --- Hit Testing ---

    function roomAtPoint(cx, cy) {
        return MapperState.camera.activeTab === '2d' ? roomAtPoint2d(cx, cy) : roomAtPoint3d(cx, cy);
    }

    function roomAtPoint2d(cx, cy) {
        var cam = MapperState.camera;
        var half = (ROOM_SIZE_2D * cam.zoomScale) / 2;
        var found = null;
        MapperState.data.rooms.forEach(function(room, id) {
            if (found !== null) return;
            if (!room.HasCoordinates || room.MapZ !== cam.activeZ2d) return;
            var p = gridToCanvas2d(room.MapX, room.MapY);
            if (cx >= p.px - half && cx <= p.px + half && cy >= p.py - half && cy <= p.py + half) {
                found = id;
            }
        });
        return found;
    }

    function roomAtPoint3d(cx, cy) {
        var cam = MapperState.camera;
        var targetZ = cam.activeZ3d !== null ? cam.activeZ3d : (MapperState.data.zLevels.length > 0 ? MapperState.data.zLevels[0] : 0);
        var step = TILE_HW_3D * GRID_STEP_XY_3D * cam.spacingScale3d * cam.zoomScale;
        var hw = step, hh = step / 2;

        // Collect active-z rooms and sort back-to-front so the topmost
        // (visually closest) tile wins the hit test.
        var list = [];
        MapperState.data.rooms.forEach(function(room, id) {
            if (room.HasCoordinates && room.MapZ === targetZ) {
                list.push({ id: id, x: room.MapX, y: room.MapY, z: room.MapZ });
            }
        });
        list.sort(function(a, b) { return (b.x + b.y) - (a.x + a.y); });

        // Diamond point-in-rhombus test
        for (var i = 0; i < list.length; i++) {
            var item = list[i];
            var p = isoProject3d(item.x, item.y, item.z);
            if (Math.abs(cx - p.sx) / hw + Math.abs(cy - p.sy) / hh <= 1) return item.id;
        }
        return null;
    }

    // --- Current Z / Grid Occupancy ---

    function currentZ() {
        var cam = MapperState.camera;
        return cam.activeTab === '3d' ? (cam.activeZ3d !== null ? cam.activeZ3d : 0) : cam.activeZ2d;
    }

    function gridCellOccupied(gx, gy, gz) {
        return MapperState.data.roomsByCoord.has(gx + ',' + gy + ',' + gz);
    }

    // --- 2D Drawing Primitives ---

    /** Draws a small badge (secret "?" or key icon) at the midpoint of a connection line. */
    function drawLineBadge2d(mx, my, type) {
        var cam = MapperState.camera;
        var sz = Math.max(7, Math.round(CONNECTION_WIDTH_2D * cam.zoomScale * 2.5));
        var half = sz / 2;

        ctx.save();
        ctx.fillStyle = MAP_BG_2D;
        ctx.fillRect(mx - half, my - half, sz, sz);

        if (type === 'secret') {
            ctx.fillStyle = '#d4a843';
            ctx.font = 'bold ' + Math.round(sz * 0.85) + 'px monospace';
            ctx.textAlign = 'center'; ctx.textBaseline = 'middle';
            ctx.fillText('?', mx, my);
        } else {
            // Tiny key icon: circle bow + shaft + two teeth
            var kc = '#9ab0d4', lw = Math.max(1, sz * 0.14);
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

    /** Draws a single room tile in 2D: filled square, border, symbol, and Z-arrow indicators. */
    function drawRoom2d(p, room, id) {
        var cam = MapperState.camera;
        var scaledSize = ROOM_SIZE_2D * cam.zoomScale;
        var scaledBorder = ROOM_BORDER_WIDTH_2D * cam.zoomScale;
        var scaledFont = SYMBOL_FONT_SIZE_2D * cam.zoomScale;
        var half = scaledSize / 2;

        var isSelected = MapperState.selected.has(id);
        var fill = isSelected ? SELECTED_ROOM_COLOR : room._color;
        var rx = p.px - half, ry = p.py - half;

        ctx.fillStyle = fill;
        ctx.fillRect(rx, ry, scaledSize, scaledSize);
        ctx.strokeStyle = ROOM_BORDER_COLOR_2D;
        ctx.lineWidth = scaledBorder;
        ctx.strokeRect(rx, ry, scaledSize, scaledSize);

        ctx.fillStyle = isSelected ? SELECTED_ROOM_TEXT_COLOR : SYMBOL_TEXT_COLOR;
        ctx.font = 'bold ' + scaledFont + 'px monospace';
        ctx.textAlign = 'center'; ctx.textBaseline = 'middle';
        ctx.fillText(room._symbol || '•', p.px, p.py);

        // Up/down arrows indicate exits to other Z levels
        var hasUp = false, hasDown = false;
        if (room.Exits) {
            for (var dir in room.Exits) {
                var delta = exitDelta(dir, room);
                if (delta && delta[2] > 0) hasUp = true;
                if (delta && delta[2] < 0) hasDown = true;
            }
        }
        if (hasUp || hasDown) {
            var arrowSize = Math.max(5, scaledSize * 0.28);
            var margin = Math.max(2, scaledSize * 0.1);
            ctx.font = 'bold ' + arrowSize + 'px monospace';
            ctx.fillStyle = isSelected ? SELECTED_ROOM_TEXT_COLOR : SYMBOL_TEXT_COLOR;
            if (hasDown) {
                ctx.textAlign = 'left'; ctx.textBaseline = 'alphabetic';
                ctx.fillText('▾', rx + margin, ry + scaledSize - margin);
            }
            if (hasUp) {
                ctx.textAlign = 'right'; ctx.textBaseline = 'top';
                ctx.fillText('▴', rx + scaledSize - margin, ry + margin);
            }
        }
    }

    // --- 3D Drawing Primitives ---

    /** Draws an isometric tile: diamond top face + two darkened side faces + symbol. */
    function drawTile3d(gx, gy, gz, topColor, isSelected, symbol, drawZ) {
        var cam = MapperState.camera;
        var hw = TILE_HW_3D * cam.zoomScale;
        var hh = TILE_HH_3D * cam.zoomScale;
        var dep = TILE_DEPTH_3D * cam.zoomScale;
        var bw = TILE_BORDER_WIDTH_3D * cam.zoomScale;
        var p = isoProject3d(gx, gy, gz, drawZ);
        var sx = p.sx, sy = p.sy;
        var leftColor = darkenColor(topColor, SIDE_DARKEN_3D * 0.8);
        var rightColor = darkenColor(topColor, SIDE_DARKEN_3D);

        // Top face
        ctx.beginPath();
        ctx.moveTo(sx, sy - hh); ctx.lineTo(sx + hw, sy);
        ctx.lineTo(sx, sy + hh); ctx.lineTo(sx - hw, sy);
        ctx.closePath();
        ctx.fillStyle = topColor; ctx.fill();
        ctx.strokeStyle = TILE_BORDER_COLOR_3D; ctx.lineWidth = bw; ctx.stroke();

        // Left face
        ctx.beginPath();
        ctx.moveTo(sx - hw, sy); ctx.lineTo(sx, sy + hh);
        ctx.lineTo(sx, sy + hh + dep); ctx.lineTo(sx - hw, sy + dep);
        ctx.closePath();
        ctx.fillStyle = leftColor; ctx.fill();
        ctx.strokeStyle = TILE_BORDER_COLOR_3D; ctx.lineWidth = bw; ctx.stroke();

        // Right face
        ctx.beginPath();
        ctx.moveTo(sx, sy + hh); ctx.lineTo(sx + hw, sy);
        ctx.lineTo(sx + hw, sy + dep); ctx.lineTo(sx, sy + hh + dep);
        ctx.closePath();
        ctx.fillStyle = rightColor; ctx.fill();
        ctx.strokeStyle = TILE_BORDER_COLOR_3D; ctx.lineWidth = bw; ctx.stroke();

        // Symbol
        ctx.fillStyle = isSelected ? SELECTED_ROOM_TEXT_COLOR : SYMBOL_TEXT_COLOR;
        ctx.font = 'bold ' + (SYMBOL_FONT_SIZE_3D * cam.zoomScale) + 'px monospace';
        ctx.textAlign = 'center'; ctx.textBaseline = 'middle';
        ctx.fillText(symbol || '•', sx, sy);
    }

    // --- 3D Bucket Cache ---
    // Pre-sorts rooms and edges by Z level so the render loop can draw
    // them in correct painter's order without re-scanning every frame.

    var bucketCache3d = null;

    function buildBucketCache3d() {
        var roomsByZ = {};
        var sameZEdges = {};
        var crossZEdges = {};

        MapperState.data.zLevels.forEach(function(z) {
            roomsByZ[z] = [];
            sameZEdges[z] = [];
            crossZEdges[z] = [];
        });

        MapperState.data.rooms.forEach(function(room, id) {
            if (!room.HasCoordinates) return;
            roomsByZ[room.MapZ].push({ id: id, x: room.MapX, y: room.MapY, z: room.MapZ,
                                       symbol: room._symbol, color: room._color });
        });

        MapperState.data.zLevels.forEach(function(z) {
            roomsByZ[z].sort(function(a, b) { return (a.x + a.y) - (b.x + b.y); });
        });

        // Deduplicate edges: only process each room-pair+direction once
        var edgeSeen = new Set();
        MapperState.data.rooms.forEach(function(room, id) {
            if (!room.HasCoordinates || !room.Exits) return;
            for (var dir in room.Exits) {
                var ex = room.Exits[dir];
                var dest = MapperState.data.rooms.get(ex.RoomId);
                if (!dest || !dest.HasCoordinates) continue;
                var key = Math.min(id, ex.RoomId) + '-' + Math.max(id, ex.RoomId) + ':' + dir;
                if (edgeSeen.has(key)) continue;
                edgeSeen.add(key);
                var delta = exitDelta(dir, room);
                var abnormal = !isDirectionalExit(dir) || !delta;
                var dz = delta ? delta[2] : (dest.MapZ - room.MapZ);
                var entry = { rA: room, rB: dest, dz: dz,
                              dx: dest.MapX - room.MapX, dy: dest.MapY - room.MapY,
                              abnormal: abnormal };
                if (abnormal || dz === 0) {
                    var z = room.MapZ;
                    if (sameZEdges[z]) sameZEdges[z].push(entry);
                    else { sameZEdges[z] = [entry]; }
                } else {
                    var lowerZ = Math.min(room.MapZ, dest.MapZ);
                    if (crossZEdges[lowerZ]) crossZEdges[lowerZ].push(entry);
                    else { crossZEdges[lowerZ] = [entry]; }
                }
            }
        });

        bucketCache3d = { roomsByZ: roomsByZ, sameZEdges: sameZEdges, crossZEdges: crossZEdges };
    }

    // --- 3D Edge Drawing ---

    /** Returns the canvas-space attach point on a tile diamond for a given exit direction. */
    function tileAttachPoint3d(sx, sy, dx, dy) {
        var cam = MapperState.camera;
        var hw = TILE_HW_3D * cam.zoomScale, hh = TILE_HH_3D * cam.zoomScale;

        // Diagonal exits attach at diamond corners
        if (dx !== 0 && dy !== 0) {
            if (dx > 0 && dy > 0) return { sx: sx, sy: sy + hh };
            if (dx > 0 && dy < 0) return { sx: sx + hw, sy: sy };
            if (dx < 0 && dy > 0) return { sx: sx - hw, sy: sy };
            return { sx: sx, sy: sy - hh };
        }

        // Cardinal exits attach at diamond edge midpoints
        if (dx > 0) return { sx: sx + hw / 2, sy: sy + hh / 2 };
        if (dx < 0) return { sx: sx - hw / 2, sy: sy - hh / 2 };
        if (dy > 0) return { sx: sx - hw / 2, sy: sy + hh / 2 };
        if (dy < 0) return { sx: sx + hw / 2, sy: sy - hh / 2 };
        return { sx: sx, sy: sy };
    }

    function drawEdge3d(e, drawZ) {
        var cam = MapperState.camera;
        var pA = isoProject3d(e.rA.MapX, e.rA.MapY, e.rA.MapZ, drawZ);
        var pB = isoProject3d(e.rB.MapX, e.rB.MapY, e.rB.MapZ, drawZ);
        var startPt, endPt;

        if (e.dz !== 0) {
            // Cross-z edges are offset horizontally to avoid overlapping tiles
            startPt = { sx: pA.sx + CROSS_Z_OFFSET_X * cam.zoomScale, sy: pA.sy };
            endPt = { sx: pB.sx + CROSS_Z_OFFSET_X * cam.zoomScale, sy: pB.sy };
        } else {
            startPt = tileAttachPoint3d(pA.sx, pA.sy, e.dx, e.dy);
            endPt = tileAttachPoint3d(pB.sx, pB.sy, -e.dx, -e.dy);
        }

        ctx.beginPath(); ctx.moveTo(startPt.sx, startPt.sy);
        ctx.lineTo(endPt.sx, endPt.sy); ctx.stroke();

        // Draw double arrowheads on cross-z connections
        if (e.dz !== 0) {
            var ddx = endPt.sx - startPt.sx;
            var ddy = endPt.sy - startPt.sy;
            var len = Math.sqrt(ddx * ddx + ddy * ddy);
            if (len > 0) {
                var ux = ddx / len, uy = ddy / len;
                var as = CROSS_Z_ARROW_SIZE * cam.zoomScale;
                ctx.fillStyle = ctx.strokeStyle;
                ctx.beginPath();
                ctx.moveTo(endPt.sx, endPt.sy);
                ctx.lineTo(endPt.sx - ux * as - uy * as, endPt.sy - uy * as + ux * as);
                ctx.lineTo(endPt.sx - ux * as + uy * as, endPt.sy - uy * as - ux * as);
                ctx.closePath(); ctx.fill();
                ctx.beginPath();
                ctx.moveTo(startPt.sx, startPt.sy);
                ctx.lineTo(startPt.sx + ux * as - uy * as, startPt.sy + uy * as + ux * as);
                ctx.lineTo(startPt.sx + ux * as + uy * as, startPt.sy + uy * as - ux * as);
                ctx.closePath(); ctx.fill();
            }
        }
    }

    // --- Tool Overlay Dispatch ---
    // After core rendering, each registered tool gets a chance to paint its
    // own overlay (selection rectangles, drag ghosts, exit-draw lines, etc.).

    function getRenderState() {
        var cam = MapperState.camera;
        return {
            ctx: ctx, canvas: canvas, zoomScale: cam.zoomScale,
            activeTab: cam.activeTab, activeZ2d: cam.activeZ2d, activeZ3d: cam.activeZ3d,
            spacingScale3d: cam.spacingScale3d,
            selectedRoomIds: MapperState.selected,
            hoveredRoomId: MapperState.hoveredRoomId,
            hoveredGridCell: MapperState.hoveredGridCell,
            gridToCanvas2d: gridToCanvas2d,
            canvasToGrid2d: canvasToGrid2d,
            isoProject3d: isoProject3d,
            canvasToGrid: canvasToGrid,
            gridCellOccupied: gridCellOccupied,
            drawRoom2d: drawRoom2d,
            drawTile3d: drawTile3d,
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

    function renderToolOverlays3d() {
        var rs = getRenderState();
        var tools = MapperTools.all();
        for (var name in tools) {
            if (tools[name] && typeof tools[name].renderOverlay3d === 'function') {
                tools[name].renderOverlay3d(ctx, rs);
            }
        }
    }

    // --- 2D Core Renderer ---

    function render2d() {
        var cam = MapperState.camera;
        var data = MapperState.data;

        ctx.clearRect(0, 0, canvas.width, canvas.height);
        ctx.fillStyle = MAP_BG_2D;
        ctx.fillRect(0, 0, canvas.width, canvas.height);

        var rooms = data.rooms;
        if (rooms.size === 0) { renderToolOverlays2d(); return; }

        // -- Edges (two passes: normal directional first, then abnormal) --
        ctx.lineCap = 'round';
        var drawnEdges = new Set();
        var abnormalEdges = [];

        // Pass 1: normal directional edges (solid lines)
        ctx.strokeStyle = CONNECTION_COLOR;
        ctx.lineWidth = CONNECTION_WIDTH_2D * cam.zoomScale;
        rooms.forEach(function(room, id) {
            if (!room.HasCoordinates || room.MapZ !== cam.activeZ2d) return;
            if (!room.Exits) return;
            for (var dir in room.Exits) {
                var ex = room.Exits[dir];
                var dest = rooms.get(ex.RoomId);
                if (!dest || !dest.HasCoordinates) continue;
                var key = Math.min(id, ex.RoomId) + '-' + Math.max(id, ex.RoomId) + ':' + dir;
                if (drawnEdges.has(key)) continue;
                drawnEdges.add(key);
                var delta = exitDelta(dir, room);
                var directional = isDirectionalExit(dir);
                if (!directional || !delta) {
                    abnormalEdges.push({ room: room, dest: dest, dir: dir, ex: ex });
                    continue;
                }
                if (delta[2] !== 0) continue;
                if (dest.MapZ !== cam.activeZ2d) continue;
                var pA = gridToCanvas2d(room.MapX, room.MapY);
                var pB = gridToCanvas2d(dest.MapX, dest.MapY);
                ctx.beginPath(); ctx.moveTo(pA.px, pA.py); ctx.lineTo(pB.px, pB.py); ctx.stroke();
                if (ex.Secret || ex.HasLock) {
                    drawLineBadge2d((pA.px + pB.px) / 2, (pA.py + pB.py) / 2, ex.Secret ? 'secret' : 'key');
                }
            }
        });

        // Pass 2: abnormal edges (yellow dotted arcs)
        if (abnormalEdges.length > 0) {
            ctx.strokeStyle = ABNORMAL_CONNECTION_COLOR;
            ctx.lineWidth = Math.max(1, CONNECTION_WIDTH_2D * cam.zoomScale * 0.7);
            ctx.setLineDash([Math.max(3, 8 * cam.zoomScale), Math.max(4, 10 * cam.zoomScale)]);
            abnormalEdges.forEach(function(ae) {
                var pA = gridToCanvas2d(ae.room.MapX, ae.room.MapY);
                var pB = gridToCanvas2d(ae.dest.MapX, ae.dest.MapY);
                var mx = (pA.px + pB.px) / 2, my = (pA.py + pB.py) / 2;
                var dx = pB.px - pA.px, dy = pB.py - pA.py;
                var dist = Math.sqrt(dx * dx + dy * dy);
                var bulge = Math.max(15, dist * 0.25);
                var cpx = mx + (-dy / dist) * bulge;
                var cpy = my + (dx / dist) * bulge;
                ctx.beginPath(); ctx.moveTo(pA.px, pA.py);
                ctx.quadraticCurveTo(cpx, cpy, pB.px, pB.py); ctx.stroke();
            });
            ctx.setLineDash([]);
        }

        // -- Rooms --
        var dragGroup = MapperState.roomDrag;
        var dragGroupSet = dragGroup.active ? new Set(dragGroup.group.keys()) : new Set();

        rooms.forEach(function(room, id) {
            if (!room.HasCoordinates || room.MapZ !== cam.activeZ2d) return;
            if (dragGroupSet.has(id)) return;
            drawRoom2d(gridToCanvas2d(room.MapX, room.MapY), room, id);
        });

        // -- Tool Overlays --
        renderToolOverlays2d();
    }

    // --- 3D Core Renderer ---

    function render3d() {
        var cam = MapperState.camera;
        var data = MapperState.data;

        ctx.clearRect(0, 0, canvas.width, canvas.height);
        ctx.fillStyle = MAP_BG_3D;
        ctx.fillRect(0, 0, canvas.width, canvas.height);
        if (data.rooms.size === 0) { renderToolOverlays3d(); return; }

        var drawZ = cam.activeZ3d !== null ? cam.activeZ3d : (data.zLevels.length > 0 ? data.zLevels[0] : 0);

        // Build the set of rooms on inactive Z levels that have a direct
        // cross-z exit to the active level -- they render at a higher alpha
        // than other inactive rooms so the spatial relationship is visible.
        var connectedToActive = new Set();
        if (!bucketCache3d) buildBucketCache3d();
        var bc = bucketCache3d;

        data.zLevels.forEach(function(z) {
            (bc.crossZEdges[z] || []).forEach(function(e) {
                if (e.rA.MapZ === drawZ && e.rB.MapZ !== drawZ)
                    connectedToActive.add(data.roomsByCoord.get(e.rB.MapX + ',' + e.rB.MapY + ',' + e.rB.MapZ) || -1);
                if (e.rB.MapZ === drawZ && e.rA.MapZ !== drawZ)
                    connectedToActive.add(data.roomsByCoord.get(e.rA.MapX + ',' + e.rA.MapY + ',' + e.rA.MapZ) || -1);
            });
        });

        ctx.lineWidth = CONNECTION_WIDTH_3D * cam.zoomScale;
        ctx.lineCap = 'round';

        var dragGroup = MapperState.roomDrag;
        var dragGroupSet3d = dragGroup.active ? new Set(dragGroup.group.keys()) : new Set();

        data.zLevels.forEach(function(z) {
            var zDiff = Math.abs(z - drawZ);
            var baseAlpha = zDiff === 0 ? 1.0 : ALPHA_INACTIVE_3D;

            // Same-z edges
            (bc.sameZEdges[z] || []).forEach(function(e) {
                ctx.globalAlpha = baseAlpha;
                if (e.abnormal) {
                    ctx.strokeStyle = ABNORMAL_CONNECTION_COLOR;
                    ctx.setLineDash([Math.max(3, 8 * cam.zoomScale), Math.max(4, 10 * cam.zoomScale)]);
                    var pA3a = isoProject3d(e.rA.MapX, e.rA.MapY, e.rA.MapZ, drawZ);
                    var pB3a = isoProject3d(e.rB.MapX, e.rB.MapY, e.rB.MapZ, drawZ);
                    var mx3 = (pA3a.sx + pB3a.sx) / 2, my3 = (pA3a.sy + pB3a.sy) / 2;
                    var dx3 = pB3a.sx - pA3a.sx, dy3 = pB3a.sy - pA3a.sy;
                    var dist3 = Math.sqrt(dx3 * dx3 + dy3 * dy3) || 1;
                    var bulge3 = Math.max(15, dist3 * 0.25);
                    ctx.beginPath(); ctx.moveTo(pA3a.sx, pA3a.sy);
                    ctx.quadraticCurveTo(mx3 + (-dy3 / dist3) * bulge3, my3 + (dx3 / dist3) * bulge3, pB3a.sx, pB3a.sy);
                    ctx.stroke();
                    ctx.setLineDash([]);
                } else {
                    ctx.strokeStyle = CONN_COLOR_SAME_Z;
                    drawEdge3d(e, drawZ);
                }
            });

            // Tiles (skip rooms being dragged -- they are drawn by the drag tool overlay)
            (bc.roomsByZ[z] || []).forEach(function(item) {
                if (dragGroupSet3d.has(item.id)) return;
                var isSelected = MapperState.selected.has(item.id);
                var topColor = isSelected ? SELECTED_ROOM_COLOR : item.color;
                var onActive = zDiff === 0;
                ctx.globalAlpha = onActive ? 1.0 : (connectedToActive.has(item.id) ? ALPHA_CONNECTED_3D : ALPHA_INACTIVE_3D);
                drawTile3d(item.x, item.y, item.z, topColor, isSelected, item.symbol, drawZ);
            });

            // Cross-z edges
            ctx.strokeStyle = CONN_COLOR_CROSS_Z;
            (bc.crossZEdges[z] || []).forEach(function(e) {
                var zd = Math.min(Math.abs(e.rA.MapZ - drawZ), Math.abs(e.rB.MapZ - drawZ));
                ctx.globalAlpha = zd === 0 ? 1.0 : ALPHA_INACTIVE_3D;
                drawEdge3d(e, drawZ);
            });
        });

        ctx.globalAlpha = 1.0;

        // -- Tool Overlays --
        renderToolOverlays3d();
    }

    // --- Render Dispatch ---

    function render() {
        if (MapperState.camera.activeTab === '3d') { bucketCache3d = null; render3d(); }
        else render2d();
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
        resizeObserver = new ResizeObserver(function() { resizeCanvas(); render(); });
        resizeObserver.observe(viewport);
    }

    // --- Public API ---
    // Setup, rendering, coordinate helpers, hit testing, and drawing
    // primitives used by tool overlay modules.

    return {
        // Setup
        setCanvas: setCanvas,
        initResizeObserver: initResizeObserver,
        resizeCanvas: resizeCanvas,

        // Rendering
        render: render,
        getRenderState: getRenderState,

        // Coordinate transforms
        gridToCanvas2d: gridToCanvas2d,
        canvasToGrid2d: canvasToGrid2d,
        canvasToGrid3d: canvasToGrid3d,
        canvasToGrid: canvasToGrid,
        isoProject3d: isoProject3d,

        // Hit testing
        roomAtPoint: roomAtPoint,
        roomAtPoint2d: roomAtPoint2d,
        roomAtPoint3d: roomAtPoint3d,
        currentZ: currentZ,
        gridCellOccupied: gridCellOccupied,

        // Drawing primitives (used by tool overlays)
        drawRoom2d: drawRoom2d,
        drawTile3d: drawTile3d,
        drawLineBadge2d: drawLineBadge2d
    };

})();
