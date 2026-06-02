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
/* globals MapperState, MapperRender, MapperTools, MapperCtxMenu, MapperUI, MapperEvents, AdminAPI,
   BASE_STEP_2D, ROOM_SIZE_2D, ROOM_BORDER_COLOR_2D, ROOM_BORDER_WIDTH_2D, SYMBOL_FONT_SIZE_2D,
   MAP_BG_2D, ABNORMAL_CONNECTION_COLOR, SELECTED_ROOM_COLOR, SELECTED_ROOM_TEXT_COLOR,
   SYMBOL_TEXT_COLOR, ROOM_BORDER_MOB_SPAWN, ROOM_BORDER_SCRIPT_GLOW, ROOM_ARROW_COLOR,
   BADGE_SECRET_COLOR, BADGE_LOCK_COLOR, ZONE_BOX_COLOR, ZONE_BOX_COLOR_HOV, ZONE_BOX_BORDER,
   ZONE_BOX_BORDER_HOV, ZONE_BOX_PADDING, computeZonePaddedBounds */
/* globals CONNECTION_COLOR: true, CONNECTION_WIDTH_2D: true */
'use strict';

(function() {

    // =====================================================================
    //  DOM setup
    // =====================================================================

    var canvas     = document.getElementById('mapper-canvas');
    var viewport   = document.getElementById('mapper-viewport');

    var domRefs = {
        canvas:            canvas,
        viewport:          viewport,
        spacingCtrlEl:     document.getElementById('spacing-controls'),
        zSelectEl:         document.getElementById('z-select'),
        zPrevBtn:          document.getElementById('z-prev'),
        zNextBtn:          document.getElementById('z-next'),
        saveCtrlEl:        document.getElementById('save-controls'),
        toastContainerEl:  document.getElementById('mapper-toast-container'),
        changelogEl:       document.getElementById('mapper-changelog'),
        clEntriesEl:       document.getElementById('changelog-entries'),
        changelogBtnEl:    document.getElementById('changelog-btn'),
        changelogBadgeEl:  document.getElementById('changelog-badge'),
        statsEl:           document.getElementById('mapper-stats'),
        zoneSelect:        document.getElementById('zone-select'),
        loadingEl:         document.getElementById('mapper-loading'),
        infoEmptyEl:       document.getElementById('info-empty'),
        infoContentEl:     document.getElementById('info-content'),
        tooltip:           document.getElementById('mapper-tooltip'),
        ctxMenuEl:         document.getElementById('mapper-ctx-menu')
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

    // Wire the API spinner to AdminAPI's busy/idle callbacks.
    (function() {
        var spinnerEl = document.getElementById('mapper-api-spinner');
        if (!spinnerEl) return;
        AdminAPI.onBusy(function()  { spinnerEl.classList.add('visible'); });
        AdminAPI.onIdle(function()  { spinnerEl.classList.remove('visible'); });
    })();

    if (domRefs.zSelectEl) {
        domRefs.zSelectEl.addEventListener('change', function() {
            MapperState.camera.activeZ2d = parseInt(this.value, 10);
            MapperUI.updateZButtons();
            MapperRender.render();
        });
    }
    if (domRefs.zPrevBtn) {
        domRefs.zPrevBtn.addEventListener('click', function() {
            var levels = MapperState.data.zLevels;
            var idx = levels.indexOf(MapperState.camera.activeZ2d);
            if (idx < levels.length - 1) {
                MapperState.camera.activeZ2d = levels[idx + 1];
                MapperUI.updateZButtons();
                MapperRender.render();
            }
        });
    }
    if (domRefs.zNextBtn) {
        domRefs.zNextBtn.addEventListener('click', function() {
            var levels = MapperState.data.zLevels;
            var idx = levels.indexOf(MapperState.camera.activeZ2d);
            if (idx > 0) {
                MapperState.camera.activeZ2d = levels[idx - 1];
                MapperUI.updateZButtons();
                MapperRender.render();
            }
        });
    }

    MapperEvents.on('room:createAt', function(data) {
        MapperUI.createRoomAt(data.gx, data.gy, data.gz);
    });

    MapperEvents.on('pan:suppressClick', function() {
        canvas.dataset.suppressClick = '1';
    });

    // =====================================================================
    //  Tool references for interceptors
    // =====================================================================

    var roomDragTool = MapperTools.get('room-drag');
    var selectTool   = MapperTools.get('select');

    MapperTools.activate('pan');

    // =====================================================================
    //  Canvas rect cache
    // Calling getBoundingClientRect() on every mouse event forces a layout
    // reflow. Cache it and invalidate only when the viewport resizes.
    // =====================================================================

    var _canvasRect = null;
    function _getCanvasRect() {
        if (!_canvasRect) _canvasRect = canvas.getBoundingClientRect();
        return _canvasRect;
    }

    // Invalidate the cached rect whenever the canvas is resized or the page
    // is scrolled. getBoundingClientRect() is viewport-relative, so any scroll
    // shifts the canvas position and the cached value becomes stale.
    (function() {
        var _origResizeCanvas = MapperRender.resizeCanvas;
        MapperRender.resizeCanvas = function() {
            _origResizeCanvas();
            _canvasRect = null;
        };
        window.addEventListener('scroll', function() { _canvasRect = null; }, { passive: true, capture: true });
    })();

    // =====================================================================
    //  Canvas event dispatch
    // =====================================================================

    canvas.addEventListener('mousedown', function(e) {
        if (e.button !== 0) return;
        e.preventDefault();
        MapperCtxMenu.hide();

        var rect = _getCanvasRect();
        var cx = e.clientX - rect.left, cy = e.clientY - rect.top;
        var roomId = MapperRender.roomAtPoint(cx, cy);
        var gridCell = roomId === null ? MapperRender.canvasToGrid(cx, cy) : null;

        var tool = MapperTools.getActive();

        // When a special tool (exit-draw, quick-build) is active, skip
        // interceptors and forward directly to the tool
        var specialActive = tool && (tool.name === 'exit-draw' || tool.name === 'quick-build');
        if (!specialActive) {
            if (selectTool && selectTool.interceptMouseDown && selectTool.interceptMouseDown(e, cx, cy, roomId)) {
                return;
            }
            if (roomDragTool && roomDragTool.interceptMouseDown && roomDragTool.interceptMouseDown(e, cx, cy, roomId)) {
                return;
            }
        }

        if (tool && tool.onMouseDown) {
            tool.onMouseDown(e, cx, cy, roomId, gridCell);
        }
    });

    canvas.addEventListener('mousemove', function(e) {
        var rect = _getCanvasRect();
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

        // Update hover state for ghost cells and cursor style.
        // Skip hoveredGridCell tracking while a room drag is active — the
        // ghost cell is not visible during drag and the extra canvasToGrid
        // call is wasted work.
        MapperState.hoveredRoomId = roomId;
        if (!MapperState.roomDrag.active) {
            var prevGhost = MapperState.hoveredGridCell;
            if (roomId === null && MapperState.data.rooms.size > 0) {
                MapperState.hoveredGridCell = MapperRender.canvasToGrid(cx, cy);
            } else {
                MapperState.hoveredGridCell = null;
            }
            var ghostChanged = (prevGhost === null) !== (MapperState.hoveredGridCell === null) ||
                (prevGhost && MapperState.hoveredGridCell &&
                 (prevGhost.gx !== MapperState.hoveredGridCell.gx || prevGhost.gy !== MapperState.hoveredGridCell.gy));
            if (ghostChanged) MapperRender.scheduleRender();
        }

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
        var rect = _getCanvasRect();
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
            MapperRender.scheduleRender();
        }
        if (MapperState.roomDrag.active) {
            MapperState.roomDrag.active = false;
            MapperState.roomDrag.anchorId = null;
            MapperState.roomDrag.group = new Map();
            MapperState.roomDrag.brokenExits = [];
            MapperState.roomDrag.allConstraints = [];
            canvas.style.cursor = '';
            MapperRender.scheduleRender();
        }
        if (MapperState.camera.dragActive) {
            MapperState.camera.dragActive = false;
            canvas.style.cursor = '';
        }
        if (MapperState.hoveredGridCell) {
            MapperState.hoveredGridCell = null;
            MapperRender.scheduleRender();
        }
        MapperState.mouseState.mousedownRoomId = null;
    });

    canvas.addEventListener('click', function(e) {
        // Suppress click when a drag or selection rect just finished
        if (canvas.dataset.suppressClick) { delete canvas.dataset.suppressClick; return; }

        var activeTool = MapperTools.getActive();
        if (activeTool && (activeTool.name === 'exit-draw' || activeTool.name === 'quick-build')) return;

        MapperCtxMenu.hide();
        var rect = _getCanvasRect();
        var cx = e.clientX - rect.left, cy = e.clientY - rect.top;
        var id = MapperRender.roomAtPoint(cx, cy);

        if (id !== null) {
            if (e.shiftKey || e.ctrlKey) {
                MapperState.toggleRoomSelection(id);
            } else {
                MapperState.selectRoom(id);
            }
        } else {
            MapperState.selectRoom(null);
        }
    });

    // Right-click: cancel special tool modes, or open the context menu.
    canvas.addEventListener('contextmenu', function(e) {
        e.preventDefault();
        e.stopPropagation();
        var activeTool = MapperTools.getActive();
        if (activeTool && activeTool.name === 'quick-build') { MapperTools.activate('pan'); return; }
        if (activeTool && activeTool.name === 'exit-draw') { MapperTools.activate('pan'); return; }

        var rect = _getCanvasRect();
        var cx = e.clientX - rect.left, cy = e.clientY - rect.top;
        var id = MapperRender.roomAtPoint(cx, cy);

        if (id !== null) {
            MapperState.selectRoom(id);
            var room = MapperState.data.rooms.get(id);
            var target = { type: 'room', roomId: id, room: room, gx: room ? room.MapX : 0, gy: room ? room.MapY : 0, gz: room ? room.MapZ : 0 };
            MapperCtxMenu.show(e.clientX, e.clientY, target);
        } else if (MapperState.data.rooms.size > 0) {
            var gc = MapperRender.canvasToGrid(cx, cy);
            var cZ = MapperRender.currentZ();
            if (!MapperRender.gridCellOccupied(gc.gx, gc.gy, cZ)) {
                var target2 = { type: 'empty', roomId: null, room: null, gx: gc.gx, gy: gc.gy, gz: cZ };
                MapperCtxMenu.show(e.clientX, e.clientY, target2);
            }
        }
    });

    // Wheel zoom: accumulate delta across rapid scroll events and apply once
    // per animation frame so a fast trackpad doesn't queue dozens of
    // individual Math.pow() calls and render passes.
    var _wheelDelta = 0;
    var _wheelScheduled = false;
    canvas.addEventListener('wheel', function(e) {
        e.preventDefault();
        _wheelDelta += e.deltaY;
        if (!_wheelScheduled) {
            _wheelScheduled = true;
            requestAnimationFrame(function() {
                _wheelScheduled = false;
                var factor = Math.pow(1.25, _wheelDelta * 0.002);
                _wheelDelta = 0;
                var cam = MapperState.camera;
                cam.zoomScale = Math.min(5.0, Math.max(0.15, cam.zoomScale / factor));
                MapperRender.scheduleRender();
            });
        }
    }, { passive: false });

    // Close context menu when clicking outside it
    document.addEventListener('click', function(e) {
        var ctxEl = domRefs.ctxMenuEl;
        if (!ctxEl.contains(e.target)) MapperCtxMenu.hide();
        // Close changelog overlay when clicking outside it and outside the button
        var clEl = domRefs.changelogEl;
        var btnEl = domRefs.changelogBtnEl;
        if (clEl && clEl.classList.contains('visible') &&
            !clEl.contains(e.target) && btnEl && !btnEl.contains(e.target)) {
            clEl.classList.remove('visible');
        }
        // Close settings modal when clicking the backdrop directly
        var settingsBackdrop = document.getElementById('mapper-settings-backdrop');
        if (settingsBackdrop && e.target === settingsBackdrop) {
            settingsBackdrop.classList.remove('visible');
        }
    });

    // =====================================================================
    //  Keyboard shortcuts
    // =====================================================================

    function _isEditingOrModal() {
        var tag = document.activeElement && document.activeElement.tagName;
        if (tag === 'INPUT' || tag === 'SELECT' || tag === 'TEXTAREA') return true;
        var backdrops = ['zone-picker-backdrop', 'zone-name-backdrop', 'unmapped-backdrop',
                         'mapper-settings-backdrop', 'room-editor-backdrop', 'exit-draw-backdrop'];
        for (var i = 0; i < backdrops.length; i++) {
            var el = document.getElementById(backdrops[i]);
            if (el && el.classList.contains('visible')) return true;
        }
        return false;
    }

    document.addEventListener('keydown', function(e) {
        if (e.key === 'Escape') {
            var settingsBackdrop = document.getElementById('mapper-settings-backdrop');
            if (settingsBackdrop && settingsBackdrop.classList.contains('visible')) {
                settingsBackdrop.classList.remove('visible');
                return;
            }
            var activeTool = MapperTools.getActive();
            if (activeTool && activeTool.name === 'quick-build') { MapperTools.activate('pan'); return; }
            if (activeTool && activeTool.name === 'exit-draw') { MapperTools.activate('pan'); return; }
            MapperCtxMenu.hide();
        }
        if (e.key === 'ArrowLeft' || e.key === 'ArrowRight' || e.key === 'ArrowUp' || e.key === 'ArrowDown') {
            if (_isEditingOrModal()) return;
            e.preventDefault();
            var cam = MapperState.camera;
            var step = BASE_STEP_2D * cam.spacingScale2d * cam.zoomScale;
            var gridStep = 1 / step * 40;  // 40px worth of scroll per keypress
            if (e.key === 'ArrowLeft')  cam.panOffsetX -= gridStep;
            if (e.key === 'ArrowRight') cam.panOffsetX += gridStep;
            if (e.key === 'ArrowUp')    cam.panOffsetY -= gridStep;
            if (e.key === 'ArrowDown')  cam.panOffsetY += gridStep;
            MapperRender.scheduleRender();
            return;
        }
        if (e.key === ',' || e.key === '.') {
            if (_isEditingOrModal()) return;
            var levels2 = MapperState.data.zLevels;
            var idx2 = levels2.indexOf(MapperState.camera.activeZ2d);
            if (e.key === '.' && idx2 < levels2.length - 1) {
                MapperState.camera.activeZ2d = levels2[idx2 + 1];
                MapperUI.updateZButtons();
                MapperRender.render();
            } else if (e.key === ',' && idx2 > 0) {
                MapperState.camera.activeZ2d = levels2[idx2 - 1];
                MapperUI.updateZButtons();
                MapperRender.render();
            }
            return;
        }
        if (e.key === 'Delete' || e.key === 'Backspace') {
            if (_isEditingOrModal()) return;
            if (MapperState.selected.size > 0) {
                var ids = Array.from(MapperState.selected);
                var proceed = ids.length === 1 || confirm('Delete ' + ids.length + ' selected rooms? All exits to/from them will be removed.');
                if (proceed) {
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
    //  Toolbar tooltips
    // =====================================================================

    (function() {
        var tip = document.getElementById('toolbar-tooltip');
        var toolbar = document.querySelector('.mapper-toolbar');
        if (!tip || !toolbar) return;

        var showTimer = null;
        var currentTarget = null;

        function show(el, x, y) {
            var text = el.getAttribute('data-tip');
            if (!text) return;
            tip.textContent = text;
            tip.style.display = 'block';
            position(x, y);
        }

        function position(x, y) {
            var tw = tip.offsetWidth;
            var th = tip.offsetHeight;
            var left = x + 12;
            var top  = y + 18;
            if (left + tw + 8 > window.innerWidth)  left = x - tw - 8;
            if (top  + th + 8 > window.innerHeight) top  = y - th - 8;
            tip.style.left = left + 'px';
            tip.style.top  = top  + 'px';
        }

        function hide() {
            clearTimeout(showTimer);
            showTimer = null;
            currentTarget = null;
            tip.style.display = 'none';
        }

        toolbar.addEventListener('mouseover', function(e) {
            var el = e.target.closest('[data-tip]');
            if (!el || el === currentTarget) return;
            hide();
            currentTarget = el;
            showTimer = setTimeout(function() {
                show(el, e.clientX, e.clientY);
            }, 400);
        });

        toolbar.addEventListener('mousemove', function(e) {
            if (tip.style.display === 'block') {
                position(e.clientX, e.clientY);
            }
        });

        toolbar.addEventListener('mouseout', function(e) {
            var el = e.target.closest('[data-tip]');
            if (!el) return;
            var to = e.relatedTarget;
            if (to && el.contains(to)) return;
            hide();
        });

        toolbar.addEventListener('mousedown', hide);
    })();

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

    window.zoomIn           = MapperUI.zoomIn;
    window.zoomOut          = MapperUI.zoomOut;
    window.centerCamera     = MapperUI.centerCamera;
    window.saveAllChanges   = MapperUI.saveAllChanges;
    window.discardChanges   = MapperUI.discardChanges;
    window.toggleChangelog  = function() {
        var clEl = domRefs.changelogEl;
        if (!clEl) return;
        clEl.classList.toggle('visible');
        if (clEl.classList.contains('visible')) {
            var last = clEl.lastElementChild && clEl.lastElementChild.lastElementChild;
            if (last) last.scrollIntoView({ block: 'nearest' });
        }
    };

    window.searchRoom = function() {
        var input = prompt('Enter a Room ID:');
        if (!input) return;
        var roomId = parseInt(input, 10);
        if (isNaN(roomId)) return;
        var room = MapperState.data.rooms.get(roomId);
        if (!room) { alert('Room #' + roomId + ' not found.'); return; }
        if (!room.HasCoordinates) { alert('Room #' + roomId + ' has no map coordinates.'); return; }
        var cam = MapperState.camera;
        cam.cameraX = room.MapX;
        cam.cameraY = room.MapY;
        cam.panOffsetX = 0;
        cam.panOffsetY = 0;
        cam.activeZ2d = room.MapZ;
        MapperState.selected.clear();
        MapperState.selected.add(roomId);
        MapperUI.updateZButtons();
        MapperUI.updateInfoPanel();
        MapperRender.render();
    };

    window.toggleLegend = function() {
        var el = document.getElementById('mapper-legend');
        if (el) el.classList.toggle('visible');
    };

    window.toggleMapperSettings = function() {
        var el = document.getElementById('mapper-settings-backdrop');
        if (el) el.classList.toggle('visible');
    };

    window.addEventListener('beforeunload', function() {
        var cam = MapperState.camera;
        localStorage.setItem('mapper.cameraState', JSON.stringify({
            zone: MapperState.data.currentZone,
            zoomScale: cam.zoomScale,
            cameraX: cam.cameraX,
            cameraY: cam.cameraY,
            panOffsetX: cam.panOffsetX,
            panOffsetY: cam.panOffsetY,
            activeZ2d: cam.activeZ2d
        }));
    });

    window.onSpacingSlider = function(val) {
        var v = Math.max(0.75, parseFloat(val));
        MapperState.camera.spacingScale2d = v;
        localStorage.setItem('mapper.spacing2d', v);
        var lbl = document.getElementById('spacing-val');
        if (lbl) lbl.textContent = v.toFixed(2);
        MapperRender.render();
    };

    window.onConnectionWidthSlider = function(val) {
        var v = Math.min(ROOM_SIZE_2D, Math.max(1, parseInt(val, 10)));
        CONNECTION_WIDTH_2D = v;
        localStorage.setItem('mapper.connectionWidth2d', v);
        var lbl = document.getElementById('conn-width-val');
        if (lbl) lbl.textContent = v;
        MapperRender.render();
    };

    window.onConnectionColorPicker = function(val) {
        CONNECTION_COLOR = val;
        localStorage.setItem('mapper.connectionColor', val);
        MapperRender.render();
    };

    window.onShowBoundsToggle = function(checked) {
        MapperState.camera.showBounds = checked;
        localStorage.setItem('mapper.showBounds', checked);
        MapperRender.render();
    };

    window.onSelectedZoneOnlyToggle = function(checked) {
        MapperState.camera.selectedZoneOnly = checked;
        localStorage.setItem('mapper.selectedZoneOnly', checked);
        MapperRender.render();
    };

    window.onMobBorderToggle = function(checked) {
        MapperState.camera.showMobBorder = checked;
        localStorage.setItem('mapper.showMobBorder', checked);
        MapperRender.render();
    };

    window.onScriptBorderToggle = function(checked) {
        MapperState.camera.showScriptBorder = checked;
        localStorage.setItem('mapper.showScriptBorder', checked);
        MapperRender.render();
    };

    window.onTagsBorderToggle = function(checked) {
        MapperState.camera.showTagsBorder = checked;
        localStorage.setItem('mapper.showTagsBorder', checked);
        MapperRender.render();
    };

    // =====================================================================
    //  Boot sequence
    // =====================================================================

    (async function() {
        await MapperState.loadBiomes();
        await MapperState.loadTags();
        await MapperState.loadAllRooms();
        MapperUI.populateZoneDropdown();
        MapperRender.initResizeObserver();

        // Sync spacing slider to persisted value
        var slider = document.getElementById('spacing-slider');
        var sliderVal = document.getElementById('spacing-val');
        if (slider) {
            slider.value = MapperState.camera.spacingScale2d;
            if (sliderVal) sliderVal.textContent = MapperState.camera.spacingScale2d.toFixed(2);
        }

        // Sync connection width slider to persisted value
        var savedConnWidth = localStorage.getItem('mapper.connectionWidth2d');
        if (savedConnWidth !== null) {
            var cw = Math.min(ROOM_SIZE_2D, Math.max(1, parseInt(savedConnWidth, 10)));
            CONNECTION_WIDTH_2D = cw;
        }
        var connWidthSlider = document.getElementById('conn-width-slider');
        var connWidthVal = document.getElementById('conn-width-val');
        if (connWidthSlider) {
            connWidthSlider.max = ROOM_SIZE_2D;
            connWidthSlider.value = CONNECTION_WIDTH_2D;
            if (connWidthVal) connWidthVal.textContent = CONNECTION_WIDTH_2D;
        }

        // Sync connection color picker to persisted value
        var savedConnColor = localStorage.getItem('mapper.connectionColor');
        if (savedConnColor) {
            CONNECTION_COLOR = savedConnColor;
        }
        var connColorPicker = document.getElementById('conn-color-picker');
        if (connColorPicker) connColorPicker.value = CONNECTION_COLOR;

        // Sync show-bounds toggle to persisted value
        var boundsToggle = document.getElementById('show-bounds-toggle');
        if (boundsToggle) boundsToggle.checked = MapperState.camera.showBounds;

        // Sync selected-zone-only toggle to persisted value
        var zoneOnlyToggle = document.getElementById('selected-zone-only-toggle');
        if (zoneOnlyToggle) zoneOnlyToggle.checked = MapperState.camera.selectedZoneOnly;

        // Sync border indicator toggles to persisted values
        var mobBorderToggle = document.getElementById('mob-border-toggle');
        if (mobBorderToggle) mobBorderToggle.checked = MapperState.camera.showMobBorder;
        var scriptBorderToggle = document.getElementById('script-border-toggle');
        if (scriptBorderToggle) scriptBorderToggle.checked = MapperState.camera.showScriptBorder;
        var tagsBorderToggle = document.getElementById('tags-border-toggle');
        if (tagsBorderToggle) tagsBorderToggle.checked = MapperState.camera.showTagsBorder;

        // Restore last-used zone, falling back to the first available zone
        var lastZone = localStorage.getItem('mapper.lastZone');
        if (!lastZone || !MapperState.data.allZones.some(function(z) { return z.Name === lastZone; })) {
            lastZone = MapperState.data.allZones.length > 0 ? MapperState.data.allZones[0].Name : null;
        }
        if (lastZone) {
            domRefs.zoneSelect.value = lastZone;
            MapperState.centerOnZone(lastZone);
        }

        var savedCam = localStorage.getItem('mapper.cameraState');
        if (savedCam) {
            try {
                var cs = JSON.parse(savedCam);
                if (cs.zone === lastZone) {
                    var cam = MapperState.camera;
                    if (typeof cs.zoomScale === 'number') cam.zoomScale = cs.zoomScale;
                    if (typeof cs.cameraX === 'number') cam.cameraX = cs.cameraX;
                    if (typeof cs.cameraY === 'number') cam.cameraY = cs.cameraY;
                    if (typeof cs.panOffsetX === 'number') cam.panOffsetX = cs.panOffsetX;
                    if (typeof cs.panOffsetY === 'number') cam.panOffsetY = cs.panOffsetY;
                    if (typeof cs.activeZ2d === 'number' && MapperState.data.zLevels.indexOf(cs.activeZ2d) !== -1) {
                        cam.activeZ2d = cs.activeZ2d;
                        MapperUI.updateZButtons();
                    }
                }
            } catch(e) { /* ignore corrupt data */ }
        }

        MapperRender.resizeCanvas();
        MapperRender.render();
    })();

})();
