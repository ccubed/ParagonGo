/* jshint esversion: 11, browser: true */
/* globals MapperTools, MapperCtxMenu, MapperState, MapperRender,
   ROOM_SIZE_2D, CONNECTION_WIDTH_2D, CONNECTION_WIDTH_3D,
   TILE_HW_3D, TILE_HH_3D,
   DIRECTION_DELTAS, DIRECTIONAL_EXITS, sign, escapeHtml */
'use strict';

/**
 * Exit-draw tool -- activated via context menu "Add Exit".
 *
 * Draws a rubber-band line from source room to cursor / target room,
 * then prompts for exit name on click.
 */
(function() {

    // -----------------------------------------------------------------
    // Exit-name validation
    // -----------------------------------------------------------------

    function validateExitName(dir, sourceRoom, targetRoom) {
        if (!dir) return null;
        if (DIRECTIONAL_EXITS[dir]) {
            var delta = DIRECTION_DELTAS[dir];
            if (delta && sourceRoom && targetRoom && sourceRoom.HasCoordinates && targetRoom.HasCoordinates) {
                var dx = targetRoom.MapX - sourceRoom.MapX;
                var dy = targetRoom.MapY - sourceRoom.MapY;
                var dz = targetRoom.MapZ - sourceRoom.MapZ;
                if (sign(dx) !== sign(delta[0]) || sign(dy) !== sign(delta[1]) || sign(dz) !== sign(delta[2])) {
                    return 'Exit "' + dir + '" does not match the spatial direction to the target room. Use a non-directional name instead.';
                }
            }
        }
        return null;
    }

    // -----------------------------------------------------------------
    // Exit draw finish / prompts
    // -----------------------------------------------------------------

    function finishExitDraw(targetRoomId) {
        var edm = MapperState.exitDrawMode;
        if (targetRoomId === edm.sourceRoomId) {
            MapperTools.activate('pan');
            return;
        }
        var sourceRoom = MapperState.data.rooms.get(edm.sourceRoomId);
        var targetRoom = MapperState.data.rooms.get(targetRoomId);

        var dir = null;
        while (true) {
            dir = prompt('Exit name (e.g. "north", "portal", "trapdoor"):');
            if (!dir || !dir.trim()) { MapperTools.activate('pan'); return; }
            dir = dir.trim();
            var err = validateExitName(dir, sourceRoom, targetRoom);
            if (!err) break;
            alert(err);
        }

        MapperState.addExitLocally(edm.sourceRoomId, dir, targetRoomId);

        var returnDir = prompt('Return exit name (leave blank for no return exit):');
        if (returnDir && returnDir.trim()) {
            returnDir = returnDir.trim();
            var err2 = validateExitName(returnDir, targetRoom, sourceRoom);
            if (err2) {
                alert(err2 + '\nReturn exit not created.');
            } else {
                MapperState.addExitLocally(targetRoomId, returnDir, edm.sourceRoomId);
            }
        }

        MapperTools.activate('pan');
        MapperRender.render();
    }

    // -----------------------------------------------------------------
    // Add exit by room number (prompt-only, does not use tool activation)
    // -----------------------------------------------------------------

    function addExitByRoomNumber(sourceRoomId) {
        var targetIdStr = prompt('Target room number:');
        if (!targetIdStr || !targetIdStr.trim()) return;
        var targetRoomId = parseInt(targetIdStr.trim(), 10);
        if (isNaN(targetRoomId)) { alert('Invalid room number.'); return; }
        if (targetRoomId === sourceRoomId) { alert('Cannot connect a room to itself.'); return; }

        var sourceRoom = MapperState.data.rooms.get(sourceRoomId);
        var targetRoom = MapperState.data.rooms.get(targetRoomId);

        var dir = null;
        while (true) {
            dir = prompt('Exit name (e.g. "north", "portal", "trapdoor"):');
            if (!dir || !dir.trim()) return;
            dir = dir.trim();
            if (sourceRoom && targetRoom) {
                var err = validateExitName(dir, sourceRoom, targetRoom);
                if (err) { alert(err); continue; }
            }
            break;
        }

        MapperState.addExitLocally(sourceRoomId, dir, targetRoomId);

        var returnDir = prompt('Return exit name (leave blank for no return exit):');
        if (returnDir && returnDir.trim()) {
            returnDir = returnDir.trim();
            if (sourceRoom && targetRoom) {
                var err2 = validateExitName(returnDir, targetRoom, sourceRoom);
                if (err2) {
                    alert(err2 + '\nReturn exit not created.');
                } else {
                    MapperState.addExitLocally(targetRoomId, returnDir, sourceRoomId);
                }
            } else {
                MapperState.addExitLocally(targetRoomId, returnDir, sourceRoomId);
            }
        }

        MapperRender.render();
    }

    // -----------------------------------------------------------------
    // Tool definition
    // -----------------------------------------------------------------

    var tool = {
        name: 'exit-draw',
        cursor: 'crosshair',

        onActivate: function(context) {
            var sourceRoomId = context.sourceRoomId;
            var room = MapperState.data.rooms.get(sourceRoomId);
            if (!room || !room.HasCoordinates) {
                MapperTools.activate('pan');
                return;
            }
            var edm = MapperState.exitDrawMode;
            edm.active = true;
            edm.sourceRoomId = sourceRoomId;
            edm.sourceGx = room.MapX;
            edm.sourceGy = room.MapY;
            edm.sourceGz = room.MapZ;
            edm.hoveredTargetId = null;
            MapperRender.render();
        },

        onDeactivate: function() {
            var edm = MapperState.exitDrawMode;
            edm.active = false;
            edm.sourceRoomId = null;
            edm.hoveredTargetId = null;
            MapperRender.render();
        },

        onMouseDown: function(e, cx, cy, roomId) {
            var edm = MapperState.exitDrawMode;
            if (!edm.active) return false;
            if (roomId !== null && roomId !== edm.sourceRoomId) {
                finishExitDraw(roomId);
            } else {
                MapperTools.activate('pan');
            }
            return true; // claim
        },

        onMouseMove: function(e, cx, cy, roomId) {
            var edm = MapperState.exitDrawMode;
            if (!edm.active) return;
            edm.hoveredTargetId = (roomId !== null && roomId !== edm.sourceRoomId) ? roomId : null;
            edm._mouseCx = cx;
            edm._mouseCy = cy;
            MapperRender.render();
        },

        onMouseUp: function() {},

        onKeyDown: function(e) {
            if (e.key === 'Escape') {
                MapperTools.activate('pan');
            }
        },

        // -----------------------------------------------------------------
        // 2D overlay: rubber-band line + target highlight
        // -----------------------------------------------------------------

        renderOverlay2d: function(ctx, rs) {
            var edm = MapperState.exitDrawMode;
            if (!edm.active) return;

            var srcP = rs.gridToCanvas2d(edm.sourceGx, edm.sourceGy);
            var endX, endY;

            if (edm.hoveredTargetId !== null) {
                var tgt = MapperState.data.rooms.get(edm.hoveredTargetId);
                if (tgt && tgt.HasCoordinates) {
                    var tgtP = rs.gridToCanvas2d(tgt.MapX, tgt.MapY);
                    endX = tgtP.px;
                    endY = tgtP.py;
                    var hlHalf = rs.scaledSize / 2 + 3 * rs.zoomScale;
                    ctx.strokeStyle = 'rgba(100,255,100,0.8)';
                    ctx.lineWidth = 2 * rs.zoomScale;
                    ctx.strokeRect(tgtP.px - hlHalf, tgtP.py - hlHalf, hlHalf * 2, hlHalf * 2);
                }
            } else {
                endX = edm._mouseCx;
                endY = edm._mouseCy;
            }

            if (endX !== undefined) {
                ctx.strokeStyle = 'rgba(100,200,255,0.8)';
                ctx.lineWidth = Math.max(2, CONNECTION_WIDTH_2D * rs.zoomScale);
                ctx.setLineDash([6 * rs.zoomScale, 4 * rs.zoomScale]);
                ctx.beginPath();
                ctx.moveTo(srcP.px, srcP.py);
                ctx.lineTo(endX, endY);
                ctx.stroke();
                ctx.setLineDash([]);
            }
        },

        // -----------------------------------------------------------------
        // 3D overlay: rubber-band line
        // -----------------------------------------------------------------

        renderOverlay3d: function(ctx, rs) {
            var edm = MapperState.exitDrawMode;
            if (!edm.active) return;

            var drawZ = rs.activeZ3d !== null ? rs.activeZ3d : 0;
            var srcP3 = rs.isoProject3d(edm.sourceGx, edm.sourceGy, edm.sourceGz, drawZ);
            var endX3, endY3;

            if (edm.hoveredTargetId !== null) {
                var tgt3 = MapperState.data.rooms.get(edm.hoveredTargetId);
                if (tgt3 && tgt3.HasCoordinates) {
                    var tgtP3 = rs.isoProject3d(tgt3.MapX, tgt3.MapY, tgt3.MapZ, drawZ);
                    endX3 = tgtP3.sx;
                    endY3 = tgtP3.sy;
                }
            } else {
                endX3 = edm._mouseCx;
                endY3 = edm._mouseCy;
            }

            if (endX3 !== undefined) {
                ctx.strokeStyle = 'rgba(100,200,255,0.8)';
                ctx.lineWidth = Math.max(2, CONNECTION_WIDTH_3D * rs.zoomScale * 1.5);
                ctx.setLineDash([6 * rs.zoomScale, 4 * rs.zoomScale]);
                ctx.beginPath();
                ctx.moveTo(srcP3.sx, srcP3.sy);
                ctx.lineTo(endX3, endY3);
                ctx.stroke();
                ctx.setLineDash([]);
            }
        }
    };

    MapperTools.register(tool);

    // -----------------------------------------------------------------
    // Context menu items
    // -----------------------------------------------------------------

    MapperCtxMenu.registerProvider(function(target) {
        if (target.type !== 'room') return null;
        return [
            {
                label: 'Add Exit',
                action: function() {
                    MapperTools.activate('exit-draw', { sourceRoomId: target.roomId });
                }
            },
            {
                label: 'Add Exit (By Room Number)',
                action: function() {
                    addExitByRoomNumber(target.roomId);
                }
            }
        ];
    });

})();
