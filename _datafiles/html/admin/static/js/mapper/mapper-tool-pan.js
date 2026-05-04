/**
 * mapper-tool-pan.js -- Default tool for panning the map canvas.
 *
 * Dragging on empty space pans the camera. When no drag is active the tool
 * renders a dashed "ghost cell" outline with a "+" on whichever empty grid
 * cell the cursor hovers, hinting that clicking will open the context menu
 * for room creation.
 *
 * Pan math differs between 2D (simple pixel-to-grid ratio) and 3D (inverse
 * isometric projection), so both paths are handled in onMouseMove.
 */
/* jshint esversion: 11, browser: true */
/* globals MapperTools, MapperState, MapperRender, MapperEvents,
   BASE_STEP_2D, TILE_HW_3D, GRID_STEP_XY_3D,
   ROOM_SIZE_2D, SYMBOL_FONT_SIZE_2D, SYMBOL_FONT_SIZE_3D,
   TILE_HH_3D, ZOOM_STEP, ZOOM_MIN, ZOOM_MAX */
'use strict';

(function() {

    var tool = {
        name: 'pan',

        // -----------------------------------------------------------------
        //  Lifecycle
        // -----------------------------------------------------------------

        onActivate: function() {},
        onDeactivate: function() {
            var cam = MapperState.camera;
            cam.dragActive = false;
        },

        // -----------------------------------------------------------------
        //  Mouse handlers
        // -----------------------------------------------------------------

        onMouseDown: function(e, cx, cy, roomId, gridCell) {
            if (roomId !== null) return false;
            if (e.shiftKey) return false; // shift+empty starts a selection rect

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

            if (cam.dragActive) {
                if (cam.activeTab === '2d') {
                    var step = BASE_STEP_2D * cam.zoomScale;
                    cam.panOffsetX = cam.dragStartPanX - (e.clientX - cam.dragStartPxX) / step;
                    cam.panOffsetY = cam.dragStartPanY - (e.clientY - cam.dragStartPxY) / step;
                } else {
                    // Invert the iso projection so pixel deltas map to grid units
                    var step3 = TILE_HW_3D * GRID_STEP_XY_3D * cam.spacingScale3d * cam.zoomScale;
                    var dsx = e.clientX - cam.dragStartPxX;
                    var dsy = e.clientY - cam.dragStartPxY;
                    cam.panOffsetX = cam.dragStartPanX - (dsx / step3 + dsy * 2 / step3) / 2;
                    cam.panOffsetY = cam.dragStartPanY - (dsy * 2 / step3 - dsx / step3) / 2;
                }
                MapperRender.render();
                return;
            }

            // Hover cursor logic is handled by init; nothing extra needed here
        },

        onMouseUp: function(e, cx, cy) {
            var cam = MapperState.camera;
            if (!cam.dragActive) return;
            var dx = e.clientX - cam.dragStartPxX;
            var dy = e.clientY - cam.dragStartPxY;
            cam.dragActive = false;
            // Suppress the click event when the user clearly intended a drag
            if (Math.abs(dx) > 4 || Math.abs(dy) > 4) {
                MapperEvents.emit('pan:suppressClick');
            }
        },

        onKeyDown: function() {},

        // -----------------------------------------------------------------
        //  2D overlay -- ghost cell on empty hovered grid position
        // -----------------------------------------------------------------

        renderOverlay2d: function(ctx, rs) {
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
        //  3D overlay -- ghost cell (iso diamond on empty hovered cell)
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
