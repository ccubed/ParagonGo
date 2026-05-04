/* jshint esversion: 11, browser: true */
/* globals MapperState, MapperRender, MapperTools, MapperCtxMenu, MapperUI, MapperEvents, AdminAPI */
'use strict';

(function() {

    // =====================================================================
    // DOM references
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
    // Wire up modules
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

    // =====================================================================
    // Event wiring
    // =====================================================================

    MapperEvents.on('room:createAt', function(data) {
        MapperUI.createRoomAt(data.gx, data.gy, data.gz);
    });

    // =====================================================================
    // Tool references for interceptors
    // =====================================================================

    var roomDragTool = MapperTools.get('room-drag');
    var selectTool   = MapperTools.get('select');

    // Activate pan as the default tool
    MapperTools.activate('pan');

    // =====================================================================
    // Canvas event dispatch
    // =====================================================================

    canvas.addEventListener('mousedown', function(e) {
        if (e.button !== 0) return;
        e.preventDefault();
        MapperCtxMenu.hide();

        var rect = canvas.getBoundingClientRect();
        var cx = e.clientX - rect.left, cy = e.clientY - rect.top;
        var roomId = MapperRender.roomAtPoint(cx, cy);
        var gridCell = roomId === null ? MapperRender.canvasToGrid(cx, cy) : null;

        // Let select tool intercept shift+drag
        if (selectTool && selectTool.interceptMouseDown && selectTool.interceptMouseDown(e, cx, cy, roomId)) {
            return;
        }

        // Let room-drag tool intercept room clicks (arms the drag)
        if (roomDragTool && roomDragTool.interceptMouseDown && roomDragTool.interceptMouseDown(e, cx, cy, roomId)) {
            return;
        }

        // Forward to active tool
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

        // Let room-drag tool handle armed-state mousemove (threshold promotion)
        if (roomDragTool && roomDragTool.onMouseMove) {
            roomDragTool.onMouseMove(e, cx, cy, roomId, gridCell);
            if (MapperState.roomDrag.active) return;
        }

        var tool = MapperTools.getActive();
        if (tool && tool.onMouseMove) {
            tool.onMouseMove(e, cx, cy, roomId, gridCell);
        }

        // Update hover state for ghost cells and cursor
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

        // Cursor
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

        // Let room-drag tool handle mouseup (even if not the "active" tool, it may be armed)
        if (roomDragTool && roomDragTool.onMouseUp) {
            roomDragTool.onMouseUp(e, cx, cy);
        }

        var tool = MapperTools.getActive();
        if (tool && tool.onMouseUp) {
            tool.onMouseUp(e, cx, cy);
        }
    });

    canvas.addEventListener('mouseleave', function() {
        // Cancel any active mode
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

    document.addEventListener('click', function(e) {
        var ctxEl = domRefs.ctxMenuEl;
        if (!ctxEl.contains(e.target)) MapperCtxMenu.hide();
    });

    document.addEventListener('keydown', function(e) {
        if (e.key === 'Escape') {
            var activeTool = MapperTools.getActive();
            if (activeTool && activeTool.name === 'quick-build') { MapperTools.activate('pan'); return; }
            if (activeTool && activeTool.name === 'exit-draw') { MapperTools.activate('pan'); return; }
            MapperCtxMenu.hide();
        }
        var tool = MapperTools.getActive();
        if (tool && tool.onKeyDown) tool.onKeyDown(e);
    });

    // =====================================================================
    // Zone select
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
    // Expose global handlers for inline HTML onclick attributes
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
    // Boot
    // =====================================================================

    (async function() {
        await MapperState.loadBiomes();
        await MapperState.loadAllRooms();
        MapperUI.populateZoneDropdown();
        MapperUI.switchTab(MapperState.camera.activeTab);
        MapperRender.initResizeObserver();

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
