/**
 * mapper-tool-select.js -- Selection rectangle and base context menu items.
 *
 * Triggered by shift+drag on empty canvas space (NOT activated via the
 * toolbar). The tool draws a translucent selection rectangle while
 * dragging and enumerates all rooms whose 2D bounding boxes intersect the
 * rectangle on mouseup.
 *
 * Also registers the "base" context menu providers for both room targets
 * and empty-cell targets:
 *   Room items:   Select, Add Room Up/Down, Edit Room, Delete Room
 *   Empty items:  Create Room Here
 */
/* jshint esversion: 11, browser: true */
/* globals MapperTools, MapperCtxMenu, MapperState, MapperRender, MapperEvents,
   ROOM_SIZE_2D, escapeHtml */
'use strict';

(function() {

    var tool = {
        name: 'select',

        onActivate: function() {},
        onDeactivate: function() {
            MapperState.selRect.active = false;
        },

        // -----------------------------------------------------------------
        //  Intercept -- detect shift+empty mousedown to start a rect
        // -----------------------------------------------------------------

        interceptMouseDown: function(e, cx, cy, roomId) {
            if (!e.shiftKey) return false;
            if (roomId !== null) return false;

            var sr = MapperState.selRect;
            sr.active = true;
            sr.startCx = cx;
            sr.startCy = cy;
            sr.endCx = cx;
            sr.endCy = cy;

            MapperTools.activate('select');
            return true; // claimed
        },

        onMouseDown: function() { return false; },

        // -----------------------------------------------------------------
        //  Rect tracking
        // -----------------------------------------------------------------

        onMouseMove: function(e, cx, cy) {
            var sr = MapperState.selRect;
            if (!sr.active) return;
            sr.endCx = cx;
            sr.endCy = cy;
            MapperRender.render();
        },

        // -----------------------------------------------------------------
        //  Room enumeration on release
        // -----------------------------------------------------------------

        onMouseUp: function(e, cx, cy) {
            var sr = MapperState.selRect;
            if (!sr.active) return;

            sr.active = false;
            var minCx = Math.min(sr.startCx, sr.endCx);
            var maxCx = Math.max(sr.startCx, sr.endCx);
            var minCy = Math.min(sr.startCy, sr.endCy);
            var maxCy = Math.max(sr.startCy, sr.endCy);

            // Only commit a selection when the rect is large enough to be intentional
            if (maxCx - minCx > 4 || maxCy - minCy > 4) {
                var half = (ROOM_SIZE_2D * MapperState.camera.zoomScale) / 2;
                if (!e.shiftKey && !e.ctrlKey) MapperState.selected.clear();
                MapperState.data.rooms.forEach(function(room, id) {
                    if (!room.HasCoordinates || room.MapZ !== MapperState.camera.activeZ2d) return;
                    var p = MapperRender.gridToCanvas2d(room.MapX, room.MapY);
                    if (p.px + half >= minCx && p.px - half <= maxCx &&
                        p.py + half >= minCy && p.py - half <= maxCy) {
                        MapperState.selected.add(id);
                    }
                });
            }

            // Suppress the click that would otherwise fire on the same mouseup
            document.getElementById('mapper-canvas').dataset.suppressClick = '1';
            MapperTools.activate('pan');
            MapperRender.render();
        },

        onKeyDown: function() {},

        // -----------------------------------------------------------------
        //  2D overlay -- selection rectangle
        // -----------------------------------------------------------------

        renderOverlay2d: function(ctx, rs) {
            var sr = MapperState.selRect;
            if (!sr.active) return;

            var sx = Math.min(sr.startCx, sr.endCx);
            var sy = Math.min(sr.startCy, sr.endCy);
            var sw = Math.abs(sr.endCx - sr.startCx);
            var sh = Math.abs(sr.endCy - sr.startCy);

            ctx.fillStyle = 'rgba(100,160,255,0.12)';
            ctx.fillRect(sx, sy, sw, sh);
            ctx.strokeStyle = 'rgba(100,160,255,0.6)';
            ctx.lineWidth = 1;
            ctx.setLineDash([4, 3]);
            ctx.strokeRect(sx, sy, sw, sh);
            ctx.setLineDash([]);
        },

        // Selection rectangle is 2D-only in the current implementation
        renderOverlay3d: function() {}
    };

    MapperTools.register(tool);

    // =====================================================================
    //  Context menu providers
    // =====================================================================

    MapperCtxMenu.registerProvider(function(target) {

        // --- Room context menu ---
        if (target.type === 'room') {
            var items = [];
            var room = target.room;
            var roomId = target.roomId;

            items.push({
                label: 'Select',
                action: function() {
                    MapperState.selectRoom(roomId);
                }
            });

            // Add Room Up/Down (only for rooms with coordinates)
            if (room && room.HasCoordinates) {
                var upZ = room.MapZ + 1;
                var downZ = room.MapZ - 1;
                var upOccupied = MapperRender.gridCellOccupied(room.MapX, room.MapY, upZ);
                var downOccupied = MapperRender.gridCellOccupied(room.MapX, room.MapY, downZ);

                items.push({
                    label: 'Add Room Up (z:' + upZ + ')' + (upOccupied ? ' — occupied' : ''),
                    disabled: upOccupied,
                    action: function() {
                        MapperEvents.emit('room:createAt', { gx: room.MapX, gy: room.MapY, gz: upZ });
                    }
                });
                items.push({
                    label: 'Add Room Down (z:' + downZ + ')' + (downOccupied ? ' — occupied' : ''),
                    disabled: downOccupied,
                    action: function() {
                        MapperEvents.emit('room:createAt', { gx: room.MapX, gy: room.MapY, gz: downZ });
                    }
                });
            }

            // Edit Room (only for persisted rooms, not temp negative IDs)
            if (roomId > 0) {
                items.push({
                    label: 'Edit Room',
                    action: function() {
                        window.location.href = '/admin/rooms#' + roomId;
                    }
                });
            }

            items.push({
                label: 'Delete Room',
                style: 'color:#ff6b6b',
                action: function() {
                    var lbl = room ? room.Title : 'Room #' + roomId;
                    if (confirm('Delete "' + lbl + '"? All exits to/from it will be removed.')) {
                        MapperState.deleteRoomLocally(roomId);
                        MapperRender.render();
                    }
                }
            });

            return items;
        }

        // --- Empty cell context menu ---
        if (target.type === 'empty') {
            return [
                {
                    label: 'Create Room Here',
                    action: function() {
                        MapperEvents.emit('room:createAt', { gx: target.gx, gy: target.gy, gz: target.gz });
                    }
                }
            ];
        }

        return null;
    });

})();
