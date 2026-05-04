/* jshint esversion: 11, browser: true */
/* globals MapperTools, MapperState, MapperRender, MapperEvents,
   BASE_STEP_2D, TILE_HW_3D, GRID_STEP_XY_3D,
   ROOM_SIZE_2D, SYMBOL_FONT_SIZE_2D, SYMBOL_FONT_SIZE_3D,
   TILE_HH_3D, ZOOM_STEP, ZOOM_MIN, ZOOM_MAX */
'use strict';

/**
 * Pan tool -- the default tool.
 *
 * Handles:
 *   - Mousedown on empty space  -> starts pan drag
 *   - Mousemove during pan      -> updates panOffsetX/Y (2D step or 3D iso inverse)
 *   - Mouseup                   -> ends pan, sets suppressClick when moved > 4 px
 *   - Ghost-cell rendering      -> dashed outline with "+" on empty hovered cells
 *   - Hover cursor              -> pointer over room, grab over empty space
 */
(function() {

    var tool = {
        name: 'pan',

        // -----------------------------------------------------------------
        // Lifecycle
        // -----------------------------------------------------------------

        onActivate: function() {},
        onDeactivate: function() {
            var cam = MapperState.camera;
            cam.dragActive = false;
        },

        // -----------------------------------------------------------------
        // Mouse handlers
        // -----------------------------------------------------------------

        onMouseDown: function(e, cx, cy, roomId, gridCell) {
            // Only claim empty-space clicks (rooms handled elsewhere)
            if (roomId !== null) return false;
            if (e.shiftKey) return false; // shift+empty = selection rect

            var cam = MapperState.camera;
            cam.dragActive = true;
            cam.dragStartPxX = e.clientX;
            cam.dragStartPxY = e.clientY;
            cam.dragStartPanX = cam.panOffsetX;
            cam.dragStartPanY = cam.panOffsetY;
            return true; // claim
        },

        onMouseMove: function(e, cx, cy, roomId, gridCell) {
            var cam = MapperState.camera;

            // Active pan drag
            if (cam.dragActive) {
                if (cam.activeTab === '2d') {
                    var step = BASE_STEP_2D * cam.zoomScale;
                    cam.panOffsetX = cam.dragStartPanX - (e.clientX - cam.dragStartPxX) / step;
                    cam.panOffsetY = cam.dragStartPanY - (e.clientY - cam.dragStartPxY) / step;
                } else {
                    var step3 = TILE_HW_3D * GRID_STEP_XY_3D * cam.spacingScale3d * cam.zoomScale;
                    var dsx = e.clientX - cam.dragStartPxX;
                    var dsy = e.clientY - cam.dragStartPxY;
                    cam.panOffsetX = cam.dragStartPanX - (dsx / step3 + dsy * 2 / step3) / 2;
                    cam.panOffsetY = cam.dragStartPanY - (dsy * 2 / step3 - dsx / step3) / 2;
                }
                MapperRender.render();
                return;
            }

            // Hover cursor (only when no special mode active)
            if (!MapperState.exitDrawMode.active && !MapperState.quickBuildMode.active) {
                // cursor is set by init; we just help with ghost-cell tracking
            }
        },

        onMouseUp: function(e, cx, cy) {
            var cam = MapperState.camera;
            if (!cam.dragActive) return;
            var dx = e.clientX - cam.dragStartPxX;
            var dy = e.clientY - cam.dragStartPxY;
            cam.dragActive = false;
            if (Math.abs(dx) > 4 || Math.abs(dy) > 4) {
                MapperEvents.emit('pan:suppressClick');
            }
        },

        onKeyDown: function() {},

        // -----------------------------------------------------------------
        // Overlay rendering -- 2D ghost cell
        // -----------------------------------------------------------------

        renderOverlay2d: function(ctx, rs) {
            // Only render when no room drag and no quick build are active
            if (MapperState.roomDrag.active) return;
            if (MapperState.quickBuildMode.active) return;
            if (MapperState.exitDrawMode.active) return;

            var hoveredGridCell = rs.hoveredGridCell;
            if (!hoveredGridCell) return;
            if (rs.gridCellOccupied(hoveredGridCell.gx, hoveredGridCell.gy, rs.activeZ2d)) return;

            var gp = rs.gridToCanvas2d(hoveredGridCell.gx, hoveredGridCell.gy);
            var scaledSize = rs.scaledSize;
            var ghalf = scaledSize / 2;

            ctx.strokeStyle = 'rgba(255,255,255,0.35)';
            ctx.lineWidth = Math.max(1, 1.5 * rs.zoomScale);
            ctx.setLineDash([Math.max(2, 4 * rs.zoomScale), Math.max(2, 4 * rs.zoomScale)]);
            ctx.strokeRect(gp.px - ghalf, gp.py - ghalf, scaledSize, scaledSize);
            ctx.setLineDash([]);

            ctx.fillStyle = 'rgba(255,255,255,0.08)';
            ctx.fillRect(gp.px - ghalf, gp.py - ghalf, scaledSize, scaledSize);

            ctx.fillStyle = 'rgba(255,255,255,0.25)';
            ctx.font = 'bold ' + Math.max(10, rs.scaledFont * 0.8) + 'px monospace';
            ctx.textAlign = 'center';
            ctx.textBaseline = 'middle';
            ctx.fillText('+', gp.px, gp.py);
        },

        // -----------------------------------------------------------------
        // Overlay rendering -- 3D ghost cell
        // -----------------------------------------------------------------

        renderOverlay3d: function(ctx, rs) {
            if (MapperState.roomDrag.active) return;
            if (MapperState.quickBuildMode.active) return;
            if (MapperState.exitDrawMode.active) return;

            var hoveredGridCell = rs.hoveredGridCell;
            if (!hoveredGridCell) return;

            var drawZ = rs.activeZ3d !== null ? rs.activeZ3d : 0;
            if (rs.gridCellOccupied(hoveredGridCell.gx, hoveredGridCell.gy, drawZ)) return;

            var gp3 = rs.isoProject3d(hoveredGridCell.gx, hoveredGridCell.gy, drawZ, drawZ);
            var ghw = TILE_HW_3D * rs.zoomScale;
            var ghh = TILE_HH_3D * rs.zoomScale;

            ctx.strokeStyle = 'rgba(255,255,255,0.35)';
            ctx.lineWidth = Math.max(1, 1.5 * rs.zoomScale);
            ctx.setLineDash([Math.max(2, 4 * rs.zoomScale), Math.max(2, 4 * rs.zoomScale)]);
            ctx.beginPath();
            ctx.moveTo(gp3.sx, gp3.sy - ghh);
            ctx.lineTo(gp3.sx + ghw, gp3.sy);
            ctx.lineTo(gp3.sx, gp3.sy + ghh);
            ctx.lineTo(gp3.sx - ghw, gp3.sy);
            ctx.closePath();
            ctx.stroke();
            ctx.setLineDash([]);

            ctx.fillStyle = 'rgba(255,255,255,0.08)';
            ctx.fill();

            ctx.fillStyle = 'rgba(255,255,255,0.25)';
            ctx.font = 'bold ' + Math.max(8, SYMBOL_FONT_SIZE_3D * rs.zoomScale * 0.8) + 'px monospace';
            ctx.textAlign = 'center';
            ctx.textBaseline = 'middle';
            ctx.fillText('+', gp3.sx, gp3.sy);
        }
    };

    MapperTools.register(tool);

})();
