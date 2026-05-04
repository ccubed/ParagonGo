/**
 * mapper-tool-quick-build.js -- Stamping mode for rapid room creation.
 *
 * Activated via the context menu "Quick Build From Here". Shows cardinal
 * and intercardinal build slots radiating from a source room. Clicking an
 * unoccupied slot stamps a new room (copying traits from the source),
 * wires bidirectional exits, and advances the source to the new room so
 * the builder can chain rooms in a straight line without reopening the
 * menu each time.
 *
 * Slots are organized by distance tier (1, 2, 3) and fade progressively
 * so the nearest directions are the most prominent.
 */
/* jshint esversion: 11, browser: true */
/* globals MapperTools, MapperCtxMenu, MapperState, MapperRender,
   ROOM_SIZE_2D, SYMBOL_FONT_SIZE_2D, SYMBOL_FONT_SIZE_3D,
   TILE_HW_3D, TILE_HH_3D,
   CARDINAL_OFFSETS, symbolForRoom, colorForSymbol */
'use strict';

(function() {

    // =====================================================================
    //  Slot computation
    // =====================================================================

    /** Build the list of cardinal/intercardinal slots around the current source. */
    function getQuickBuildSlots() {
        var qb = MapperState.quickBuildMode;
        var slots = [];
        var gz = qb.sourceGz;
        var blockedDirs = {};
        CARDINAL_OFFSETS.forEach(function(co) {
            var gx = qb.sourceGx + co.dx;
            var gy = qb.sourceGy + co.dy;
            var occupied = MapperRender.gridCellOccupied(gx, gy, gz);
            var blocked = !!blockedDirs[co.label];
            // Mark farther slots along the same direction as blocked when
            // a nearer slot in that direction is already occupied.
            if (occupied) blockedDirs[co.label] = true;
            slots.push({
                gx: gx, gy: gy, gz: gz,
                dir: co.dir, ret: co.ret, label: co.label,
                dist: co.dist, occupied: occupied || blocked
            });
        });
        return slots;
    }

    function isQuickBuildSlot(gx, gy) {
        var gz = MapperState.quickBuildMode.sourceGz;
        for (var i = 0; i < CARDINAL_OFFSETS.length; i++) {
            var co = CARDINAL_OFFSETS[i];
            if (MapperState.quickBuildMode.sourceGx + co.dx === gx &&
                MapperState.quickBuildMode.sourceGy + co.dy === gy &&
                !MapperRender.gridCellOccupied(gx, gy, gz)) {
                return true;
            }
        }
        return false;
    }

    // =====================================================================
    //  Stamp-and-advance
    // =====================================================================

    /** Create a room at (gx,gy), wire exits, copy traits, and advance the source. */
    function quickBuildAt(gx, gy) {
        var qb = MapperState.quickBuildMode;
        var gz = qb.sourceGz;
        var match = null;
        CARDINAL_OFFSETS.forEach(function(co) {
            if (qb.sourceGx + co.dx === gx && qb.sourceGy + co.dy === gy) {
                match = co;
            }
        });
        if (!match) return false;
        if (MapperRender.gridCellOccupied(gx, gy, gz)) return false;

        var newId = MapperState.createRoomLocally(gx, gy, gz);

        // Copy visual traits from the source so the new room blends in
        var srcRoom = MapperState.data.rooms.get(qb.sourceRoomId);
        var newRoom = MapperState.data.rooms.get(newId);
        if (srcRoom && newRoom) {
            newRoom.Title = srcRoom.Title;
            newRoom.Biome = srcRoom.Biome;
            newRoom.MapSymbol = srcRoom.MapSymbol;
            newRoom.MapLegend = srcRoom.MapLegend;
            newRoom._symbol = symbolForRoom(newRoom);
            newRoom._color = colorForSymbol(newRoom._symbol, newRoom.Biome);
        }

        MapperState.addExitLocally(qb.sourceRoomId, match.dir, newId);
        MapperState.addExitLocally(newId, match.ret, qb.sourceRoomId);

        // Advance source to the newly created room
        qb.sourceRoomId = newId;
        qb.sourceGx = gx;
        qb.sourceGy = gy;
        MapperState.selectRoom(newId);
        MapperRender.render();
        return true;
    }

    // =====================================================================
    //  Tool definition
    // =====================================================================

    var tool = {
        name: 'quick-build',
        cursor: 'crosshair',

        onActivate: function(context) {
            var sourceRoomId = context.sourceRoomId;
            var room = MapperState.data.rooms.get(sourceRoomId);
            if (!room || !room.HasCoordinates) {
                MapperTools.activate('pan');
                return;
            }
            if (!MapperState.data.currentZone) {
                alert('Select a zone from the dropdown first.');
                MapperTools.activate('pan');
                return;
            }
            var qb = MapperState.quickBuildMode;
            qb.active = true;
            qb.sourceRoomId = sourceRoomId;
            qb.sourceGx = room.MapX;
            qb.sourceGy = room.MapY;
            qb.sourceGz = room.MapZ;
            MapperRender.render();
        },

        onDeactivate: function() {
            var qb = MapperState.quickBuildMode;
            qb.active = false;
            qb.sourceRoomId = null;
            MapperRender.render();
        },

        onMouseDown: function(e, cx, cy, roomId, gridCell) {
            var qb = MapperState.quickBuildMode;
            if (!qb.active) return false;

            if (roomId === null) {
                var gc = MapperRender.canvasToGrid(cx, cy);
                if (!quickBuildAt(gc.gx, gc.gy)) {
                    MapperTools.activate('pan');
                }
            } else {
                MapperTools.activate('pan');
            }
            return true; // claim
        },

        onMouseMove: function() {},

        onMouseUp: function() {},

        onKeyDown: function(e) {
            if (e.key === 'Escape') {
                MapperTools.activate('pan');
            }
        },

        // -----------------------------------------------------------------
        //  2D overlay -- cardinal slots with labels and hover highlighting
        // -----------------------------------------------------------------

        renderOverlay2d: function(ctx, rs) {
            var qb = MapperState.quickBuildMode;
            if (!qb.active) return;
            if (qb.sourceGz !== rs.activeZ2d) return;

            var scaledSize = rs.scaledSize;
            var half = rs.half;
            var hoveredGridCell = rs.hoveredGridCell;

            var srcP = rs.gridToCanvas2d(qb.sourceGx, qb.sourceGy);

            // Highlight the current source room
            ctx.strokeStyle = 'rgba(95,183,122,0.8)';
            ctx.lineWidth = Math.max(2, 2.5 * rs.zoomScale);
            ctx.strokeRect(
                srcP.px - half - 2 * rs.zoomScale,
                srcP.py - half - 2 * rs.zoomScale,
                scaledSize + 4 * rs.zoomScale,
                scaledSize + 4 * rs.zoomScale
            );

            var slots = getQuickBuildSlots();
            // Distance tiers fade so nearer slots draw more attention
            var fadeByDist = [1.0, 0.7, 0.45];

            slots.forEach(function(slot) {
                var sp = rs.gridToCanvas2d(slot.gx, slot.gy);
                var isHovered = hoveredGridCell && hoveredGridCell.gx === slot.gx && hoveredGridCell.gy === slot.gy;
                var fade = fadeByDist[slot.dist - 1] || 0.3;

                if (slot.occupied) {
                    ctx.strokeStyle = 'rgba(255,255,255,' + (0.1 * fade) + ')';
                    ctx.lineWidth = Math.max(1, 1 * rs.zoomScale);
                    ctx.strokeRect(sp.px - half, sp.py - half, scaledSize, scaledSize);
                } else {
                    var alpha = isHovered ? 0.9 : 0.4 * fade;
                    ctx.strokeStyle = 'rgba(95,183,122,' + alpha + ')';
                    ctx.lineWidth = Math.max(1, 1.5 * rs.zoomScale);
                    ctx.setLineDash(isHovered ? [] : [Math.max(2, 4 * rs.zoomScale), Math.max(2, 4 * rs.zoomScale)]);
                    ctx.strokeRect(sp.px - half, sp.py - half, scaledSize, scaledSize);
                    ctx.setLineDash([]);
                    ctx.fillStyle = 'rgba(95,183,122,' + (isHovered ? 0.15 : 0.05 * fade) + ')';
                    ctx.fillRect(sp.px - half, sp.py - half, scaledSize, scaledSize);
                    ctx.fillStyle = 'rgba(95,183,122,' + alpha + ')';
                    ctx.font = 'bold ' + Math.max(8, rs.scaledFont * 0.6) + 'px monospace';
                    ctx.textAlign = 'center';
                    ctx.textBaseline = 'middle';
                    var lbl = slot.label.substring(0, 2).toUpperCase();
                    if (slot.dist > 1) lbl += slot.dist;
                    ctx.fillText(lbl, sp.px, sp.py);
                }

                // Connection line from source to slot
                var lineAlpha = slot.occupied ? 0.05 * fade : (isHovered ? 0.6 : 0.15 * fade);
                ctx.strokeStyle = 'rgba(95,183,122,' + lineAlpha + ')';
                ctx.lineWidth = Math.max(1, 2 * rs.zoomScale);
                ctx.beginPath();
                ctx.moveTo(srcP.px, srcP.py);
                ctx.lineTo(sp.px, sp.py);
                ctx.stroke();
            });
        },

        // -----------------------------------------------------------------
        //  3D overlay -- cardinal slots as iso diamonds
        // -----------------------------------------------------------------

        renderOverlay3d: function(ctx, rs) {
            var qb = MapperState.quickBuildMode;
            if (!qb.active) return;

            var drawZ = rs.activeZ3d !== null ? rs.activeZ3d : 0;
            if (qb.sourceGz !== drawZ) return;

            var ghw3q = TILE_HW_3D * rs.zoomScale;
            var ghh3q = TILE_HH_3D * rs.zoomScale;
            var hoveredGridCell = rs.hoveredGridCell;
            var slots3 = getQuickBuildSlots();
            var fade3 = [1.0, 0.7, 0.45];

            slots3.forEach(function(slot) {
                var sp3 = rs.isoProject3d(slot.gx, slot.gy, drawZ, drawZ);
                var isH3 = hoveredGridCell && hoveredGridCell.gx === slot.gx && hoveredGridCell.gy === slot.gy;
                var f3 = fade3[slot.dist - 1] || 0.3;

                if (!slot.occupied) {
                    var a3 = isH3 ? 0.9 : 0.4 * f3;
                    ctx.strokeStyle = 'rgba(95,183,122,' + a3 + ')';
                    ctx.lineWidth = Math.max(1, 1.5 * rs.zoomScale);
                    if (!isH3) ctx.setLineDash([Math.max(2, 4 * rs.zoomScale), Math.max(2, 4 * rs.zoomScale)]);
                    ctx.beginPath();
                    ctx.moveTo(sp3.sx, sp3.sy - ghh3q);
                    ctx.lineTo(sp3.sx + ghw3q, sp3.sy);
                    ctx.lineTo(sp3.sx, sp3.sy + ghh3q);
                    ctx.lineTo(sp3.sx - ghw3q, sp3.sy);
                    ctx.closePath();
                    ctx.stroke();
                    ctx.setLineDash([]);
                    ctx.fillStyle = 'rgba(95,183,122,' + (isH3 ? 0.15 : 0.05 * f3) + ')';
                    ctx.fill();
                    ctx.fillStyle = 'rgba(95,183,122,' + a3 + ')';
                    ctx.font = 'bold ' + Math.max(6, SYMBOL_FONT_SIZE_3D * rs.zoomScale * 0.6) + 'px monospace';
                    ctx.textAlign = 'center';
                    ctx.textBaseline = 'middle';
                    var lbl3 = slot.label.substring(0, 2).toUpperCase();
                    if (slot.dist > 1) lbl3 += slot.dist;
                    ctx.fillText(lbl3, sp3.sx, sp3.sy);
                }
            });
        }
    };

    MapperTools.register(tool);

    // =====================================================================
    //  Context menu item
    // =====================================================================

    MapperCtxMenu.registerProvider(function(target) {
        if (target.type !== 'room') return null;
        var room = target.room;
        if (!room || !room.HasCoordinates) return null;
        return [
            {
                label: 'Quick Build From Here',
                style: 'color:#5fb77a',
                action: function() {
                    MapperTools.activate('quick-build', { sourceRoomId: target.roomId });
                }
            }
        ];
    });

})();
