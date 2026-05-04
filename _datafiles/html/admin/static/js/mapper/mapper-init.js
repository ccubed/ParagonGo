/**
 * mapper-init.js -- Boot module that wires every mapper subsystem together.
 *
 * Loaded last (after state, render, tools, UI, and all tool modules).
 * Responsibilities:
 *   1. Collect DOM references and pass them to each module's init.
 *   2. Wire cross-module callbacks (render, UI updates).
 *   3. Attach canvas event listeners and dispatch to the tool chain.
 *      The dispatch order for mousedown is: select intercept, room-drag
 *      intercept, then the active tool -- so the select and drag tools
 *      can claim events before the default pan tool sees them.
 *   4. Register keyboard shortcuts (Escape, Delete/Backspace).
 *   5. Populate the zone dropdown and restore the last-used zone.
 *   6. Expose global handler references for inline HTML onclick attributes.
 *   7. Run the async boot sequence (load biomes, load rooms, center camera).
 */
/* jshint esversion: 11, browser: true */
/* globals MapperState, MapperRender, MapperTools, MapperCtxMenu, MapperUI, MapperEvents, AdminAPI */
'use strict';

(function() {

    // =====================================================================
    //  DOM setup
    // =====================================================================

    var canvas     = document.getElementById('mapper-canvas');
    var viewport   = document.getElementById('mapper-viewport');

    var domRefs = {
        canvas:       canvas,
        viewport:     viewport,
        spacingCtrlEl: document.getElementById('spacing-controls'),
        zButtonsEl:    document.getElementById('z-buttons'),
        saveCtrlEl:    document.getElementById('save-controls'),
        dirtyCountEl:  document.getElementById('dirty-count'),
        changelogEl:   document.getElementById('mapper-changelog'),
        clEntriesEl:   document.getElementById('changelog-entries'),
        statsEl:       document.getElementById('mapper-stats'),
        zoneSelect:    document.getElementById('zone-select'),
        loadingEl:     document.getElementById('mapper-loading'),
        infoEmptyEl:   document.getElementById('info-empty'),
        infoContentEl: document.getElementById('info-content'),
        tooltip:       document.getElementById('mapper-tooltip'),
        ctxMenuEl:     document.getElementById('mapper-ctx-menu')
    };

    // =====================================================================
    //  Module wiring
    // =====================================================================

    MapperState.setDom(domRefs);
    MapperState.setRenderFn(function() { MapperRender.render(); });
    MapperState.setUpdateFns({
        updateInfoPanel:  function() { MapperUI.updateInfoPanel(); },
        updateStats:      function() { MapperUI.updateStats(); },
        updateZButtons:   function() { MapperUI.updateZButtons(); }
    });

    MapperRender.setCanvas(canvas);
    MapperCtxMenu.init(domRefs.ctxMenuEl);
    MapperUI.init(domRefs);

    MapperEvents.on('room:createAt', function(data) {
        MapperUI.createRoomAt(data.gx, data.gy, data.gz);
    });

    // =====================================================================
    //  Tool references for interceptors
    // =====================================================================

    var roomDragTool = MapperTools.get('room-drag');
    var selectTool   = MapperTools.get('select');

    MapperTools.activate('pan');

    // =====================================================================
    //  Canvas event dispatch
    // =====================================================================

    canvas.addEventListener('mousedown', function(e) {
        if (e.button !== 0) return;
        e.preventDefault();
        MapperCtxMenu.hide();

        var rect = canvas.getBoundingClientRect();
        var cx = e.clientX - rect.left, cy = e.clientY - rect.top;
        var roomId = MapperRender.roomAtPoint(cx, cy);
        var gridCell = roomId === null ? MapperRender.canvasToGrid(cx, cy) : null;

        // Intercept order matters: selection rect first, then room-drag
        if (selectTool && selectTool.interceptMouseDown && selectTool.interceptMouseDown(e, cx, cy, roomId)) {
            return;
        }
        if (roomDragTool && roomDragTool.interceptMouseDown && roomDragTool.interceptMouseDown(e, cx, cy, roomId)) {
            return;
        }

        var tool = MapperTools.getActive();
        if (tool && tool.onMouseDown) {
            tool.onMouseDown(e, cx, cy, roomId, gridCell);
        }
    });

    canvas.addEventListener('mousemove', function(e) {
        var rect = canvas.getBoundingClientRect();
        var cx = e.clientX - rect.left, cy = e.clientY - rect.top;
        var roomId = MapperRender.roomAtPoint(cx, cy);
        var gridCell = roomId === null ? MapperRender.canvasToGrid(cx, cy) : null;

        // Room-drag gets first look so threshold promotion works even when
        // the active tool is still "pan"
        if (roomDragTool && roomDragTool.onMouseMove) {
            roomDragTool.onMouseMove(e, cx, cy, roomId, gridCell);
            if (MapperState.roomDrag.active) return;
        }

        var tool = MapperTools.getActive();
        if (tool && tool.onMouseMove) {
            tool.onMouseMove(e, cx, cy, roomId, gridCell);
        }

        // Update hover state for ghost cells and cursor style
        MapperState.hoveredRoomId = roomId;
        var prevGhost = MapperState.hoveredGridCell;
        if (roomId === null && MapperState.data.rooms.size > 0) {
            MapperState.hoveredGridCell = MapperRender.canvasToGrid(cx, cy);
        } else {
            MapperState.hoveredGridCell = null;
        }
        var ghostChanged = (prevGhost === null) !== (MapperState.hoveredGridCell === null) ||
            (prevGhost && MapperState.hoveredGridCell &&
             (prevGhost.gx !== MapperState.hoveredGridCell.gx || prevGhost.gy !== MapperState.hoveredGridCell.gy));
        if (ghostChanged) MapperRender.render();

        // Cursor: tools with a custom cursor override the default room/empty logic
        var activeTool = MapperTools.getActive();
        var specialCursor = (activeTool && activeTool.cursor) ? activeTool.cursor : null;
        if (roomId !== null) {
            canvas.style.cursor = specialCursor || 'pointer';
        } else {
            canvas.style.cursor = specialCursor || 'grab';
        }
    });

    canvas.addEventListener('mouseup', function(e) {
        var rect = canvas.getBoundingClientRect();
        var cx = e.clientX - rect.left, cy = e.clientY - rect.top;

        // Room-drag may still be armed even though it is not the active tool
        if (roomDragTool && roomDragTool.onMouseUp) {
            roomDragTool.onMouseUp(e, cx, cy);
        }

        var tool = MapperTools.getActive();
        if (tool && tool.onMouseUp) {
            tool.onMouseUp(e, cx, cy);
        }
    });

    canvas.addEventListener('mouseleave', function() {
        // Clean up any in-progress interaction so state does not leak
        var tool = MapperTools.getActive();
        if (tool && tool.name === 'select' && MapperState.selRect.active) {
            MapperState.selRect.active = false;
            canvas.style.cursor = '';
            MapperRender.render();
        }
        if (MapperState.roomDrag.active) {
            MapperState.roomDrag.active = false;
            MapperState.roomDrag.anchorId = null;
            MapperState.roomDrag.group = new Map();
            MapperState.roomDrag.brokenExits = [];
            MapperState.roomDrag.allConstraints = [];
            canvas.style.cursor = '';
            MapperRender.render();
        }
        if (MapperState.camera.dragActive) {
            MapperState.camera.dragActive = false;
            canvas.style.cursor = '';
        }
        if (MapperState.hoveredGridCell) {
            MapperState.hoveredGridCell = null;
            MapperRender.render();
        }
        MapperState.mouseState.mousedownRoomId = null;
    });

    canvas.addEventListener('click', function(e) {
        // Suppress click when a drag or selection rect just finished
        if (canvas.dataset.suppressClick) { delete canvas.dataset.suppressClick; return; }

        var activeTool = MapperTools.getActive();
        if (activeTool && (activeTool.name === 'exit-draw' || activeTool.name === 'quick-build')) return;

        MapperCtxMenu.hide();
        var rect = canvas.getBoundingClientRect();
        var cx = e.clientX - rect.left, cy = e.clientY - rect.top;
        var id = MapperRender.roomAtPoint(cx, cy);

        if (id !== null) {
            if (e.shiftKey || e.ctrlKey) {
                MapperState.toggleRoomSelection(id);
            } else {
                MapperState.selectRoom(id);
                var room = MapperState.data.rooms.get(id);
                var target = { type: 'room', roomId: id, room: room, gx: room ? room.MapX : 0, gy: room ? room.MapY : 0, gz: room ? room.MapZ : 0 };
                MapperCtxMenu.show(e.clientX, e.clientY, target);
                e.stopPropagation();
            }
        } else {
            MapperState.selectRoom(null);
            if (MapperState.data.rooms.size > 0) {
                var gc = MapperRender.canvasToGrid(cx, cy);
                var cZ = MapperRender.currentZ();
                if (!MapperRender.gridCellOccupied(gc.gx, gc.gy, cZ)) {
                    var target2 = { type: 'empty', roomId: null, room: null, gx: gc.gx, gy: gc.gy, gz: cZ };
                    MapperCtxMenu.show(e.clientX, e.clientY, target2);
                    e.stopPropagation();
                }
            }
        }
    });

    // Right-click cancels special tool modes instead of opening browser menu
    canvas.addEventListener('contextmenu', function(e) {
        e.preventDefault();
        e.stopPropagation();
        var activeTool = MapperTools.getActive();
        if (activeTool && activeTool.name === 'quick-build') { MapperTools.activate('pan'); return; }
        if (activeTool && activeTool.name === 'exit-draw') { MapperTools.activate('pan'); return; }
    });

    canvas.addEventListener('wheel', function(e) {
        e.preventDefault();
        var factor = Math.pow(1.25, e.deltaY * 0.002);
        var cam = MapperState.camera;
        cam.zoomScale = Math.min(5.0, Math.max(0.15, cam.zoomScale / factor));
        MapperRender.render();
    }, { passive: false });

    // Close context menu when clicking outside it
    document.addEventListener('click', function(e) {
        var ctxEl = domRefs.ctxMenuEl;
        if (!ctxEl.contains(e.target)) MapperCtxMenu.hide();
    });

    // =====================================================================
    //  Keyboard shortcuts
    // =====================================================================

    document.addEventListener('keydown', function(e) {
        if (e.key === 'Escape') {
            var activeTool = MapperTools.getActive();
            if (activeTool && activeTool.name === 'quick-build') { MapperTools.activate('pan'); return; }
            if (activeTool && activeTool.name === 'exit-draw') { MapperTools.activate('pan'); return; }
            MapperCtxMenu.hide();
        }
        if (e.key === 'Delete' || e.key === 'Backspace') {
            if (MapperState.selected.size > 0) {
                var ids = Array.from(MapperState.selected);
                var msg = ids.length === 1
                    ? 'Delete this room? All exits to/from it will be removed.'
                    : 'Delete ' + ids.length + ' selected rooms? All exits to/from them will be removed.';
                if (confirm(msg)) {
                    ids.forEach(function(rid) { MapperState.deleteRoomLocally(rid); });
                    MapperRender.render();
                }
                e.preventDefault();
                return;
            }
        }
        var tool = MapperTools.getActive();
        if (tool && tool.onKeyDown) tool.onKeyDown(e);
    });

    // =====================================================================
    //  Zone dropdown
    // =====================================================================

    domRefs.zoneSelect.addEventListener('change', function() {
        var zone = domRefs.zoneSelect.value;
        if (!zone) return;
        if (MapperState.isDirty() && !confirm('You have unsaved changes. Discard and switch zones?')) {
            domRefs.zoneSelect.value = MapperState.data.currentZone || '';
            return;
        }
        if (MapperState.isDirty()) { MapperState.clearDirty(); }
        localStorage.setItem('mapper.lastZone', zone);
        MapperState.centerOnZone(zone);
    });

    // =====================================================================
    //  Global handler exposure (for inline HTML onclick attributes)
    // =====================================================================

    window.switchTab      = MapperUI.switchTab;
    window.zoomIn         = MapperUI.zoomIn;
    window.zoomOut        = MapperUI.zoomOut;
    window.centerCamera   = MapperUI.centerCamera;
    window.spacingDown    = MapperUI.spacingDown;
    window.spacingUp      = MapperUI.spacingUp;
    window.saveAllChanges = MapperUI.saveAllChanges;
    window.discardChanges = MapperUI.discardChanges;

    // =====================================================================
    //  Boot sequence
    // =====================================================================

    (async function() {
        await MapperState.loadBiomes();
        await MapperState.loadAllRooms();
        MapperUI.populateZoneDropdown();
        MapperUI.switchTab(MapperState.camera.activeTab);
        MapperRender.initResizeObserver();

        // Restore last-used zone, falling back to the first available zone
        var lastZone = localStorage.getItem('mapper.lastZone');
        if (!lastZone || !MapperState.data.allZones.some(function(z) { return z.Name === lastZone; })) {
            lastZone = MapperState.data.allZones.length > 0 ? MapperState.data.allZones[0].Name : null;
        }
        if (lastZone) {
            domRefs.zoneSelect.value = lastZone;
            MapperState.centerOnZone(lastZone);
        }

        MapperRender.resizeCanvas();
        MapperRender.render();
    })();

})();
