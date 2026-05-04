/* jshint esversion: 11, browser: true */
/* globals MapperTools, MapperState, MapperRender, MapperEvents,
   BASE_STEP_2D, ROOM_SIZE_2D, SYMBOL_FONT_SIZE_2D, ROOM_BORDER_WIDTH_2D,
   CONNECTION_WIDTH_2D,
   TILE_HW_3D, TILE_HH_3D, TILE_DEPTH_3D, TILE_BORDER_WIDTH_3D,
   TILE_BORDER_COLOR_3D, GRID_STEP_XY_3D, SYMBOL_FONT_SIZE_3D, SIDE_DARKEN_3D,
   SELECTED_ROOM_COLOR, SELECTED_ROOM_TEXT_COLOR, SYMBOL_TEXT_COLOR,
   buildDragConstraints, isExitConstraintSatisfied, darkenColor */
'use strict';

/**
 * Room-drag tool -- threshold-promoted from mousedown on a room.
 *
 * This tool is NOT pre-activated via the toolbar. Instead it provides an
 * `interceptMouseDown` that the init module calls. When the user drags a
 * room past 4 px the tool promotes itself to the active tool.
 *
 * State lives on MapperState.roomDrag so the renderers can read it.
 */
(function() {

    // Private armed state (before threshold promotion)
    var armed = false;
    var armedRoomId = null;
    var armedPxX = 0;
    var armedPxY = 0;

    function clearArmed() {
        armed = false;
        armedRoomId = null;
        armedPxX = 0;
        armedPxY = 0;
    }

    var tool = {
        name: 'room-drag',
        cursor: 'move',

        // -----------------------------------------------------------------
        // Lifecycle
        // -----------------------------------------------------------------

        onActivate: function() {},

        onDeactivate: function() {
            var rd = MapperState.roomDrag;
            rd.active = false;
            rd.anchorId = null;
            rd.group = new Map();
            rd.brokenExits = [];
            rd.allConstraints = [];
            clearArmed();
        },

        // -----------------------------------------------------------------
        // Intercept -- called by init before normal mousedown dispatch
        // Returns true if we claimed/armed the event.
        // -----------------------------------------------------------------

        interceptMouseDown: function(e, cx, cy, roomId) {
            if (roomId === null) return false;
            if (e.shiftKey) return false; // shift-click = toggle selection
            armed = true;
            armedRoomId = roomId;
            armedPxX = e.clientX;
            armedPxY = e.clientY;
            return true; // claimed -- caller should NOT start a pan
        },

        // -----------------------------------------------------------------
        // Mouse handlers (active after promotion)
        // -----------------------------------------------------------------

        onMouseDown: function() { return false; },

        onMouseMove: function(e, cx, cy, roomId, gridCell) {
            var rd = MapperState.roomDrag;

            // Phase 1: armed but not yet promoted -- check threshold
            if (armed && !rd.active) {
                var movedPx = Math.abs(e.clientX - armedPxX) + Math.abs(e.clientY - armedPxY);
                if (movedPx > 4) {
                    var anchorRoom = MapperState.data.rooms.get(armedRoomId);
                    if (anchorRoom && anchorRoom.HasCoordinates) {
                        var groupIds = MapperState.selected.has(armedRoomId) && MapperState.selected.size > 1
                            ? new Set(MapperState.selected) : new Set([armedRoomId]);
                        var groupMap = new Map();
                        var groupSet = new Set();
                        groupIds.forEach(function(rid) {
                            var r = MapperState.data.rooms.get(rid);
                            if (r && r.HasCoordinates) {
                                groupMap.set(rid, { startGx: r.MapX, startGy: r.MapY });
                                groupSet.add(rid);
                            }
                        });

                        var allC = [];
                        groupSet.forEach(function(rid) {
                            allC = allC.concat(buildDragConstraints(rid, groupSet));
                        });

                        var anchorP;
                        if (MapperState.camera.activeTab === '2d') {
                            var ap2 = MapperRender.gridToCanvas2d(anchorRoom.MapX, anchorRoom.MapY);
                            anchorP = { px: ap2.px, py: ap2.py };
                        } else {
                            var ap3 = MapperRender.isoProject3d(anchorRoom.MapX, anchorRoom.MapY, anchorRoom.MapZ, MapperRender.currentZ());
                            anchorP = { px: ap3.sx, py: ap3.sy };
                        }

                        rd.active = true;
                        rd.anchorId = armedRoomId;
                        rd.group = groupMap;
                        rd.deltaGx = 0;
                        rd.deltaGy = 0;
                        rd.pixelDx = 0;
                        rd.pixelDy = 0;
                        rd.anchorCanvasPx = anchorP.px;
                        rd.anchorCanvasPy = anchorP.py;
                        rd.droppable = true;
                        rd.brokenExits = [];
                        rd.allConstraints = allC;

                        // Activate ourselves as the tool
                        MapperTools.activate('room-drag');
                        MapperRender.render();
                        return;
                    }
                }
                return; // still below threshold
            }

            // Phase 2: actively dragging
            if (!rd.active) return;

            rd.pixelDx = cx - rd.anchorCanvasPx;
            rd.pixelDy = cy - rd.anchorCanvasPy;

            var gc = MapperRender.canvasToGrid(cx, cy);
            var anchorStart = rd.group.get(rd.anchorId);
            var newDx = gc.gx - anchorStart.startGx;
            var newDy = gc.gy - anchorStart.startGy;

            if (newDx !== rd.deltaGx || newDy !== rd.deltaGy) {
                rd.deltaGx = newDx;
                rd.deltaGy = newDy;
                var cZ = MapperRender.currentZ();
                var canDrop = true;
                rd.group.forEach(function(start, rid) {
                    var pGx = start.startGx + newDx;
                    var pGy = start.startGy + newDy;
                    var coordKey = pGx + ',' + pGy + ',' + cZ;
                    var occupant = MapperState.data.roomsByCoord.get(coordKey);
                    if (occupant !== undefined && !rd.group.has(occupant)) canDrop = false;
                });
                rd.droppable = canDrop;
                if (canDrop && (newDx !== 0 || newDy !== 0)) {
                    var broken = [];
                    rd.allConstraints.forEach(function(c) {
                        if (!isExitConstraintSatisfied(c, c.ownerGx + newDx, c.ownerGy + newDy)) broken.push(c);
                    });
                    rd.brokenExits = broken;
                } else {
                    rd.brokenExits = [];
                }
            }
            MapperRender.render();
        },

        onMouseUp: function(e, cx, cy) {
            var rd = MapperState.roomDrag;

            // If armed but never promoted, clear and let click through
            if (armed && !rd.active) {
                clearArmed();
                return;
            }

            if (!rd.active) return;

            var wasDroppable = rd.droppable;
            var dGx = rd.deltaGx;
            var dGy = rd.deltaGy;
            var groupCopy = new Map(rd.group);

            rd.active = false;
            rd.anchorId = null;
            rd.group = new Map();
            rd.brokenExits = [];
            rd.allConstraints = [];
            clearArmed();

            MapperEvents.emit('pan:suppressClick');

            if (wasDroppable && (dGx !== 0 || dGy !== 0)) {
                MapperState.applyGroupMove(groupCopy, dGx, dGy);
            }

            // Deactivate ourselves, return to default tool
            MapperTools.activate('pan');
            MapperRender.render();
        },

        onKeyDown: function() {},

        // -----------------------------------------------------------------
        // 2D overlay: origin markers, snap indicators, connection lines, drag ghosts
        // -----------------------------------------------------------------

        renderOverlay2d: function(ctx, rs) {
            var rd = MapperState.roomDrag;
            if (!rd.active) return;

            var scaledSize = rs.scaledSize;
            var half = rs.half;
            var hasBroken = rd.brokenExits.length > 0;
            var hasMoved = rd.deltaGx !== 0 || rd.deltaGy !== 0;

            // Origin marker (faint outline where the room was)
            if (hasMoved) {
                rd.group.forEach(function(start) {
                    var origP = rs.gridToCanvas2d(start.startGx, start.startGy);
                    ctx.strokeStyle = 'rgba(255,255,255,0.15)';
                    ctx.lineWidth = Math.max(1, 1 * rs.zoomScale);
                    ctx.setLineDash([Math.max(2, 3 * rs.zoomScale), Math.max(2, 3 * rs.zoomScale)]);
                    ctx.strokeRect(origP.px - half, origP.py - half, scaledSize, scaledSize);
                    ctx.setLineDash([]);
                });
            }

            // Grid-snap indicator (dashed outline at the target cell)
            if (hasMoved) {
                rd.group.forEach(function(start) {
                    var snapP = rs.gridToCanvas2d(start.startGx + rd.deltaGx, start.startGy + rd.deltaGy);
                    var snapColor = !rd.droppable ? 'rgba(255,80,80,0.5)' :
                                    hasBroken ? 'rgba(255,180,60,0.5)' : 'rgba(100,200,255,0.5)';
                    ctx.strokeStyle = snapColor;
                    ctx.lineWidth = Math.max(1, 1.5 * rs.zoomScale);
                    ctx.setLineDash([Math.max(2, 4 * rs.zoomScale), Math.max(2, 4 * rs.zoomScale)]);
                    ctx.strokeRect(snapP.px - half, snapP.py - half, scaledSize, scaledSize);
                    ctx.setLineDash([]);
                });
            }

            // Predicted connection lines from snap positions to neighbors
            if (rd.droppable && hasMoved) {
                ctx.lineWidth = CONNECTION_WIDTH_2D * rs.zoomScale * 0.7;
                ctx.lineCap = 'round';
                ctx.setLineDash([Math.max(2, 3 * rs.zoomScale), Math.max(2, 3 * rs.zoomScale)]);
                rd.allConstraints.forEach(function(c) {
                    var isBroken = !isExitConstraintSatisfied(c, c.ownerGx + rd.deltaGx, c.ownerGy + rd.deltaGy);
                    var fromP = rs.gridToCanvas2d(c.ownerGx + rd.deltaGx, c.ownerGy + rd.deltaGy);
                    var toP = rs.gridToCanvas2d(c.refX, c.refY);
                    ctx.strokeStyle = isBroken ? 'rgba(255,60,60,0.7)' : 'rgba(100,200,100,0.6)';
                    ctx.beginPath();
                    ctx.moveTo(fromP.px, fromP.py);
                    ctx.lineTo(toP.px, toP.py);
                    ctx.stroke();
                });
                ctx.setLineDash([]);
            }

            // Rooms rendered solid, following cursor at pixel offset
            rd.group.forEach(function(start, rid) {
                var origP = rs.gridToCanvas2d(start.startGx, start.startGy);
                var dragP = { px: origP.px + rd.pixelDx, py: origP.py + rd.pixelDy };
                var dragRoom = MapperState.data.rooms.get(rid);
                if (!dragRoom) return;

                var blocked = !rd.droppable && hasMoved;
                if (blocked) { ctx.globalAlpha = 0.5; }
                rs.drawRoom2d(dragP, dragRoom, rid);
                if (blocked) { ctx.globalAlpha = 1.0; }

                if (rd.droppable && hasBroken) {
                    ctx.fillStyle = 'rgba(255,180,60,0.2)';
                    ctx.fillRect(dragP.px - half, dragP.py - half, scaledSize, scaledSize);
                }
            });
        },

        // -----------------------------------------------------------------
        // 3D overlay: snap indicators and drag ghost tiles
        // -----------------------------------------------------------------

        renderOverlay3d: function(ctx, rs) {
            var rd = MapperState.roomDrag;
            if (!rd.active) return;

            var hasBroken3 = rd.brokenExits.length > 0;
            var hasMoved3 = rd.deltaGx !== 0 || rd.deltaGy !== 0;
            var drawZ = rs.activeZ3d !== null ? rs.activeZ3d : 0;

            // Grid-snap indicator at target cell
            if (hasMoved3) {
                var ghw3 = TILE_HW_3D * rs.zoomScale;
                var ghh3 = TILE_HH_3D * rs.zoomScale;
                var snapColor3 = !rd.droppable ? 'rgba(255,80,80,0.5)' :
                                 hasBroken3 ? 'rgba(255,180,60,0.5)' : 'rgba(100,200,255,0.5)';
                ctx.strokeStyle = snapColor3;
                ctx.lineWidth = Math.max(1, 1.5 * rs.zoomScale);
                ctx.setLineDash([Math.max(2, 4 * rs.zoomScale), Math.max(2, 4 * rs.zoomScale)]);
                rd.group.forEach(function(start) {
                    var sp = rs.isoProject3d(start.startGx + rd.deltaGx, start.startGy + rd.deltaGy, drawZ, drawZ);
                    ctx.beginPath();
                    ctx.moveTo(sp.sx, sp.sy - ghh3);
                    ctx.lineTo(sp.sx + ghw3, sp.sy);
                    ctx.lineTo(sp.sx, sp.sy + ghh3);
                    ctx.lineTo(sp.sx - ghw3, sp.sy);
                    ctx.closePath();
                    ctx.stroke();
                });
                ctx.setLineDash([]);
            }

            // Rooms following cursor at pixel offset
            rd.group.forEach(function(start, rid) {
                var origP3 = rs.isoProject3d(start.startGx, start.startGy, drawZ, drawZ);
                var dragSx = origP3.sx + rd.pixelDx;
                var dragSy = origP3.sy + rd.pixelDy;
                var dragRoom3 = MapperState.data.rooms.get(rid);
                if (!dragRoom3) return;

                var isSelected3 = rs.selectedRoomIds.has(rid);
                var topColor3 = isSelected3 ? SELECTED_ROOM_COLOR : dragRoom3._color;

                var blocked3 = !rd.droppable && hasMoved3;
                if (blocked3) ctx.globalAlpha = 0.5;

                // Draw tile at pixel offset (manual inline to use custom sx/sy)
                var hw3  = TILE_HW_3D * rs.zoomScale;
                var hh3  = TILE_HH_3D * rs.zoomScale;
                var dep3 = TILE_DEPTH_3D * rs.zoomScale;
                var bw3  = TILE_BORDER_WIDTH_3D * rs.zoomScale;
                var leftC3  = darkenColor(topColor3, SIDE_DARKEN_3D * 0.8);
                var rightC3 = darkenColor(topColor3, SIDE_DARKEN_3D);

                // Top face
                ctx.beginPath();
                ctx.moveTo(dragSx, dragSy - hh3);
                ctx.lineTo(dragSx + hw3, dragSy);
                ctx.lineTo(dragSx, dragSy + hh3);
                ctx.lineTo(dragSx - hw3, dragSy);
                ctx.closePath();
                ctx.fillStyle = topColor3; ctx.fill();
                ctx.strokeStyle = TILE_BORDER_COLOR_3D; ctx.lineWidth = bw3; ctx.stroke();

                // Left face
                ctx.beginPath();
                ctx.moveTo(dragSx - hw3, dragSy);
                ctx.lineTo(dragSx, dragSy + hh3);
                ctx.lineTo(dragSx, dragSy + hh3 + dep3);
                ctx.lineTo(dragSx - hw3, dragSy + dep3);
                ctx.closePath();
                ctx.fillStyle = leftC3; ctx.fill();
                ctx.strokeStyle = TILE_BORDER_COLOR_3D; ctx.lineWidth = bw3; ctx.stroke();

                // Right face
                ctx.beginPath();
                ctx.moveTo(dragSx, dragSy + hh3);
                ctx.lineTo(dragSx + hw3, dragSy);
                ctx.lineTo(dragSx + hw3, dragSy + dep3);
                ctx.lineTo(dragSx, dragSy + hh3 + dep3);
                ctx.closePath();
                ctx.fillStyle = rightC3; ctx.fill();
                ctx.strokeStyle = TILE_BORDER_COLOR_3D; ctx.lineWidth = bw3; ctx.stroke();

                // Symbol
                ctx.fillStyle = isSelected3 ? SELECTED_ROOM_TEXT_COLOR : SYMBOL_TEXT_COLOR;
                ctx.font = 'bold ' + (SYMBOL_FONT_SIZE_3D * rs.zoomScale) + 'px monospace';
                ctx.textAlign = 'center'; ctx.textBaseline = 'middle';
                ctx.fillText(dragRoom3._symbol || '•', dragSx, dragSy);

                if (blocked3) ctx.globalAlpha = 1.0;
            });
        }
    };

    MapperTools.register(tool);

})();
