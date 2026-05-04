/* jshint esversion: 11, browser: true */
/* globals MapperState, MapperRender, MapperEvents, MapperCtxMenu, AdminAPI,
   symbolForRoom, colorForSymbol, escapeHtml, smoothstep,
   ZOOM_STEP, ZOOM_MIN, ZOOM_MAX, CENTER_EASE_DURATION,
   SPACING_STEP_3D, SPACING_MIN_3D, SPACING_MAX_3D, ROOM_SIZE_2D */
'use strict';

var MapperUI = (function() {

    // =====================================================================
    // DOM references (set via init)
    // =====================================================================

    var dom = {
        viewport: null,
        spacingCtrlEl: null,
        zButtonsEl: null,
        saveCtrlEl: null,
        dirtyCountEl: null,
        changelogEl: null,
        clEntriesEl: null,
        statsEl: null,
        zoneSelect: null,
        loadingEl: null,
        infoEmptyEl: null,
        infoContentEl: null,
        tooltip: null
    };

    function init(domRefs) {
        if (domRefs.viewport) dom.viewport = domRefs.viewport;
        if (domRefs.spacingCtrlEl) dom.spacingCtrlEl = domRefs.spacingCtrlEl;
        if (domRefs.zButtonsEl) dom.zButtonsEl = domRefs.zButtonsEl;
        if (domRefs.saveCtrlEl) dom.saveCtrlEl = domRefs.saveCtrlEl;
        if (domRefs.dirtyCountEl) dom.dirtyCountEl = domRefs.dirtyCountEl;
        if (domRefs.changelogEl) dom.changelogEl = domRefs.changelogEl;
        if (domRefs.clEntriesEl) dom.clEntriesEl = domRefs.clEntriesEl;
        if (domRefs.statsEl) dom.statsEl = domRefs.statsEl;
        if (domRefs.zoneSelect) dom.zoneSelect = domRefs.zoneSelect;
        if (domRefs.loadingEl) dom.loadingEl = domRefs.loadingEl;
        if (domRefs.infoEmptyEl) dom.infoEmptyEl = domRefs.infoEmptyEl;
        if (domRefs.infoContentEl) dom.infoContentEl = domRefs.infoContentEl;
        if (domRefs.tooltip) dom.tooltip = domRefs.tooltip;
    }

    // =====================================================================
    // Tab switching
    // =====================================================================

    function switchTab(tab) {
        MapperState.camera.activeTab = tab;
        localStorage.setItem('mapper.activeTab', tab);
        document.querySelectorAll('.mapper-tabs .tab-btn').forEach(function(b) {
            b.classList.toggle('active', b.dataset.tab === tab);
        });
        if (dom.spacingCtrlEl) dom.spacingCtrlEl.style.display = tab === '3d' ? '' : 'none';
        MapperRender.resizeCanvas();
        updateZButtons();
        MapperRender.render();
    }

    // =====================================================================
    // Zoom / controls
    // =====================================================================

    function zoomIn() {
        MapperState.camera.zoomScale = Math.min(ZOOM_MAX, MapperState.camera.zoomScale * ZOOM_STEP);
        MapperRender.render();
    }

    function zoomOut() {
        MapperState.camera.zoomScale = Math.max(ZOOM_MIN, MapperState.camera.zoomScale / ZOOM_STEP);
        MapperRender.render();
    }

    function spacingDown() {
        MapperState.camera.spacingScale3d = Math.max(SPACING_MIN_3D, MapperState.camera.spacingScale3d / SPACING_STEP_3D);
        localStorage.setItem('mapper.spacing3d', MapperState.camera.spacingScale3d);
        MapperRender.render();
    }

    function spacingUp() {
        MapperState.camera.spacingScale3d = Math.min(SPACING_MAX_3D, MapperState.camera.spacingScale3d * SPACING_STEP_3D);
        localStorage.setItem('mapper.spacing3d', MapperState.camera.spacingScale3d);
        MapperRender.render();
    }

    function centerCamera() {
        MapperState.camera.panOffsetX = 0;
        MapperState.camera.panOffsetY = 0;
        if (MapperState.data.currentZone) {
            MapperState.centerOnZone(MapperState.data.currentZone);
        } else if (MapperState.data.zLevels.length > 0) {
            MapperState.camera.activeZ2d = MapperState.data.zLevels[0];
            MapperState.camera.activeZ3d = MapperState.data.zLevels[0];
            updateZButtons();
            MapperRender.render();
        }
    }

    function setCameraTarget(tx, ty, tz) {
        var cam = MapperState.camera;
        cam.panOffsetX = 0;
        cam.panOffsetY = 0;
        if (typeof tz === 'undefined') tz = cam.cameraZ;
        if (CENTER_EASE_DURATION <= 0) {
            cam.cameraX = tx; cam.cameraY = ty; cam.cameraZ = tz;
            MapperRender.render();
            return;
        }
        if (cam.easeRafId !== null) { cancelAnimationFrame(cam.easeRafId); cam.easeRafId = null; }
        cam.easeStartX = cam.cameraX; cam.easeStartY = cam.cameraY; cam.easeStartZ = cam.cameraZ;
        cam.easeTargetX = tx; cam.easeTargetY = ty; cam.easeTargetZ = tz;
        cam.easeStartTime = null;
        function step(ts) {
            if (cam.easeStartTime === null) cam.easeStartTime = ts;
            var t = Math.min((ts - cam.easeStartTime) / 1000 / CENTER_EASE_DURATION, 1);
            var s = smoothstep(t);
            cam.cameraX = cam.easeStartX + (cam.easeTargetX - cam.easeStartX) * s;
            cam.cameraY = cam.easeStartY + (cam.easeTargetY - cam.easeStartY) * s;
            cam.cameraZ = cam.easeStartZ + (cam.easeTargetZ - cam.easeStartZ) * s;
            MapperRender.render();
            cam.easeRafId = t < 1 ? requestAnimationFrame(step) : null;
        }
        cam.easeRafId = requestAnimationFrame(step);
    }

    // =====================================================================
    // Z-level controls
    // =====================================================================

    function updateZButtons() {
        if (!dom.zButtonsEl) return;
        dom.zButtonsEl.innerHTML = '';
        if (MapperState.data.zLevels.length <= 1) return;
        var cam = MapperState.camera;
        var current = cam.activeTab === '3d' ? cam.activeZ3d : cam.activeZ2d;
        MapperState.data.zLevels.slice().reverse().forEach(function(z) {
            var btn = document.createElement('button');
            btn.textContent = z;
            btn.title = 'Z level ' + z;
            btn.style.cssText = 'width:24px;height:20px;padding:0;font-size:0.7rem;font-family:monospace;' +
                'border:1px solid var(--color-border);border-radius:3px;cursor:pointer;';
            if (z === current) {
                btn.style.background = 'var(--color-primary)';
                btn.style.color = '#fff';
                btn.style.borderColor = 'var(--color-primary)';
            } else {
                btn.style.background = 'var(--color-surface-white)';
                btn.style.color = 'var(--color-text-dim)';
            }
            btn.addEventListener('click', function() {
                if (cam.activeTab === '3d') cam.activeZ3d = z; else cam.activeZ2d = z;
                updateZButtons();
                MapperRender.render();
            });
            dom.zButtonsEl.appendChild(btn);
        });
    }

    // =====================================================================
    // Info panel
    // =====================================================================

    function updateInfoPanel() {
        if (!dom.infoEmptyEl || !dom.infoContentEl) return;

        if (MapperState.selected.size === 0) {
            dom.infoEmptyEl.style.display = '';
            dom.infoContentEl.style.display = 'none';
            return;
        }

        dom.infoEmptyEl.style.display = 'none';
        dom.infoContentEl.style.display = '';

        var html = '';

        // Multi-select summary
        if (MapperState.selected.size > 1) {
            var ids = Array.from(MapperState.selected).sort(function(a, b) { return a - b; });
            html = '<div class="info-row"><span class="info-label">Selected</span><span class="info-value">' + ids.length + ' rooms</span></div>';
            html += '<hr class="info-divider">';
            ids.forEach(function(rid) {
                var r = MapperState.data.rooms.get(rid);
                var title = r ? escapeHtml(r.Title) : '?';
                html += '<div class="info-exit-row"><span class="info-exit-dir">#' + rid + '</span><span class="info-exit-id">' + title + '</span></div>';
            });
            dom.infoContentEl.innerHTML = html;
            return;
        }

        // Single room detail
        var selectedRoomId = Array.from(MapperState.selected)[0];
        var room = MapperState.data.rooms.get(selectedRoomId);
        if (!room) {
            dom.infoEmptyEl.style.display = '';
            dom.infoContentEl.style.display = 'none';
            return;
        }

        html = '';
        html += '<div class="info-row"><span class="info-label">ID</span><span class="info-value">' + selectedRoomId + '</span></div>';
        html += '<div class="info-row"><span class="info-label">Title</span><span class="info-value">' + escapeHtml(room.Title) + '</span></div>';
        html += '<div class="info-row"><span class="info-label">Biome</span><span class="info-value">' + escapeHtml(room.Biome || '-') + '</span></div>';
        html += '<div class="info-row"><span class="info-label">Symbol</span><span class="info-value">' + escapeHtml(room._symbol || '-') + '</span></div>';
        if (room.MapLegend) {
            html += '<div class="info-row"><span class="info-label">Legend</span><span class="info-value">' + escapeHtml(room.MapLegend) + '</span></div>';
        }
        html += '<div class="info-row"><span class="info-label">Coords</span><span class="info-value">' + room.MapX + ', ' + room.MapY + ', ' + room.MapZ + '</span></div>';
        html += '<div class="info-row"><span class="info-label">Has Coords</span><span class="info-value">' + (room.HasCoordinates ? 'yes' : 'no') + '</span></div>';

        if (room.Exits && Object.keys(room.Exits).length > 0) {
            html += '<hr class="info-divider">';
            html += '<div class="info-exits">';
            var dirs = Object.keys(room.Exits).sort();
            dirs.forEach(function(dir) {
                var ex = room.Exits[dir];
                var badges = '';
                if (ex.Secret) badges += '<span class="info-badge secret">secret</span> ';
                if (ex.HasLock) badges += '<span class="info-badge locked">locked</span> ';
                html += '<div class="info-exit-row">';
                html += '<span class="info-exit-dir">' + escapeHtml(dir) + '</span>';
                html += '<span class="info-exit-id">' + badges + '#' + ex.RoomId + '</span>';
                html += '</div>';
            });
            html += '</div>';
        }

        html += '<hr class="info-divider">';
        html += '<a class="info-link" href="/admin/rooms#' + selectedRoomId + '">Edit Room &rarr;</a>';

        dom.infoContentEl.innerHTML = html;
    }

    // =====================================================================
    // Tooltip
    // =====================================================================

    function showTooltip(mx, my, room, id) {
        if (!dom.tooltip) return;
        var html = '<div class="tt-title">' + escapeHtml(room.Title || 'Unknown') + '</div>';
        html += '<div class="tt-id">Room #' + id + '</div>';
        html += '<hr class="tt-divider">';
        if (room.Biome) html += '<div class="tt-row"><span class="tt-label">Biome</span><span class="tt-value">' + escapeHtml(room.Biome) + '</span></div>';
        if (room._symbol) html += '<div class="tt-row"><span class="tt-label">Symbol</span><span class="tt-value">' + escapeHtml(room._symbol) + '</span></div>';
        html += '<div class="tt-row"><span class="tt-label">Coords</span><span class="tt-value">' + room.MapX + ', ' + room.MapY + ', ' + room.MapZ + '</span></div>';
        if (room.Exits) {
            var dirs = Object.keys(room.Exits).sort();
            if (dirs.length > 0) {
                html += '<hr class="tt-divider">';
                html += '<div class="tt-row"><span class="tt-label">Exits</span><span class="tt-value">' + dirs.join(', ') + '</span></div>';
            }
        }
        dom.tooltip.innerHTML = html;
        dom.tooltip.style.display = 'block';
        positionTooltip(mx, my);
    }

    function positionTooltip(mx, my) {
        if (!dom.tooltip) return;
        var ttW = dom.tooltip.offsetWidth, ttH = dom.tooltip.offsetHeight;
        var vw = window.innerWidth, vh = window.innerHeight;
        var left = mx + 14;
        if (left + ttW > vw - 8) left = mx - ttW - 14;
        left = Math.max(8, left);
        var top = my - Math.floor(ttH / 2);
        if (top + ttH > vh - 8) top = vh - ttH - 8;
        top = Math.max(8, top);
        dom.tooltip.style.left = left + 'px';
        dom.tooltip.style.top  = top + 'px';
    }

    function hideTooltip() {
        if (!dom.tooltip) return;
        var cam = MapperState.camera;
        cam.tooltipHideTimer = setTimeout(function() {
            dom.tooltip.style.display = 'none';
        }, 80);
    }

    // =====================================================================
    // Loading indicator
    // =====================================================================

    function showLoading(show) {
        if (!dom.loadingEl) return;
        dom.loadingEl.classList.toggle('hidden', !show);
    }

    // =====================================================================
    // Stats display
    // =====================================================================

    function updateStats() {
        if (!dom.statsEl) return;
        var data = MapperState.data;
        var total = data.rooms.size;
        var coordinated = 0;
        data.rooms.forEach(function(r) { if (r.HasCoordinates) coordinated++; });
        var unmapped = total - coordinated;
        var txt = total + ' rooms';
        if (unmapped > 0) txt += ' (' + unmapped + ' unmapped)';
        txt += ' | ' + data.zLevels.length + ' z-level' + (data.zLevels.length !== 1 ? 's' : '');
        txt += ' | ' + data.visibleZones.size + '/' + data.allZones.length + ' zones';
        dom.statsEl.textContent = txt;
    }

    // =====================================================================
    // Zone dropdown
    // =====================================================================

    function populateZoneDropdown() {
        if (!dom.zoneSelect) return;
        var current = dom.zoneSelect.value;
        while (dom.zoneSelect.options.length > 1) dom.zoneSelect.remove(1);
        MapperState.data.allZones.forEach(function(z) {
            var opt = document.createElement('option');
            opt.value = z.Name;
            opt.textContent = z.Name + ' (' + z.RoomCount + ' rooms)';
            dom.zoneSelect.appendChild(opt);
        });
        if (current) dom.zoneSelect.value = current;
    }

    // =====================================================================
    // Local room creation (deferred save)
    // =====================================================================

    function createRoomAt(gx, gy, gz) {
        if (!MapperState.data.currentZone) {
            alert('Select a zone from the dropdown first to set which zone new rooms are created in.');
            return;
        }
        var tempId = MapperState.createRoomLocally(gx, gy, gz);
        MapperState.selectRoom(tempId);
        MapperRender.render();
    }

    // =====================================================================
    // Coordinate conversion for saving
    // =====================================================================

    function toServerCoords(room) {
        if (!room) return { x: 0, y: 0, z: 0 };
        var data = MapperState.data;
        var zone = room.Zone || data.currentZone;
        var off = zone ? data.zoneOffsets.get(zone) : null;
        if (!off) return { x: room.MapX, y: room.MapY, z: room.MapZ };
        return { x: room.MapX - off.dx, y: room.MapY - off.dy, z: room.MapZ };
    }

    // =====================================================================
    // Save / Discard
    // =====================================================================

    async function saveAllChanges() {
        var data = MapperState.data;
        var dirty = MapperState.dirty;
        showLoading(true);

        // 1. Delete rooms on server
        for (var i = 0; i < dirty.deletedRooms.length; i++) {
            await AdminAPI.delete('/admin/api/v1/rooms/' + dirty.deletedRooms[i]);
        }

        // 2. Create new rooms on server
        var tempToReal = new Map();
        for (var entry of dirty.createdRooms) {
            var tempId = entry[0], info = entry[1];
            var zone = info.zone || data.currentZone || '';
            var off = data.zoneOffsets.get(zone) || { dx: 0, dy: 0 };
            var serverX = info.gx - off.dx, serverY = info.gy - off.dy;
            var tempRoom = data.rooms.get(tempId);
            var createRes = await AdminAPI.post('/admin/api/v1/rooms', { Zone: zone });
            if (createRes.ok && createRes.data && createRes.data.data) {
                var realId = createRes.data.data.RoomId;
                tempToReal.set(tempId, realId);
                var patchData = {
                    MapX: serverX, MapY: serverY, MapZ: info.gz, HasCoordinates: true
                };
                if (tempRoom) {
                    if (tempRoom.Title && tempRoom.Title !== 'New Room') patchData.Title = tempRoom.Title;
                    if (tempRoom.Biome) patchData.Biome = tempRoom.Biome;
                    if (tempRoom.MapSymbol) patchData.MapSymbol = tempRoom.MapSymbol;
                    if (tempRoom.MapLegend) patchData.MapLegend = tempRoom.MapLegend;
                }
                await AdminAPI.patch('/admin/api/v1/rooms/' + realId, patchData);
            }
        }

        // 3. Move existing rooms on server
        for (var entry2 of dirty.movedRooms) {
            var roomId = entry2[0];
            if (roomId < 0) continue;
            var room = data.rooms.get(roomId);
            if (room) {
                var sc = toServerCoords(room);
                await AdminAPI.patch('/admin/api/v1/rooms/' + roomId, {
                    MapX: sc.x, MapY: sc.y, MapZ: sc.z, HasCoordinates: true
                });
            }
        }

        // 4. Handle exit changes (grouped by room: removals and additions)
        var exitChangesByRoom = new Map();
        dirty.exitRemovals.forEach(function(er) {
            var rId = er.roomId < 0 ? (tempToReal.get(er.roomId) || er.roomId) : er.roomId;
            if (rId < 0) return;
            if (!exitChangesByRoom.has(rId)) exitChangesByRoom.set(rId, { remove: new Set(), add: [] });
            exitChangesByRoom.get(rId).remove.add(er.dir);
        });
        dirty.exitAdditions.forEach(function(ea) {
            var rId = ea.roomId < 0 ? (tempToReal.get(ea.roomId) || ea.roomId) : ea.roomId;
            var tId = ea.targetRoomId < 0 ? (tempToReal.get(ea.targetRoomId) || ea.targetRoomId) : ea.targetRoomId;
            if (rId < 0 || tId < 0) return;
            if (!exitChangesByRoom.has(rId)) exitChangesByRoom.set(rId, { remove: new Set(), add: [] });
            exitChangesByRoom.get(rId).add.push({ dir: ea.dir, targetRoomId: tId });
        });
        for (var entry3 of exitChangesByRoom) {
            var rId = entry3[0], changes = entry3[1];
            var res = await AdminAPI.get('/admin/api/v1/rooms/' + rId);
            if (res.ok && res.data && res.data.data) {
                var fullRoom = res.data.data;
                var mergedExits = {};
                for (var d in (fullRoom.Exits || {})) {
                    if (!changes.remove.has(d)) mergedExits[d] = fullRoom.Exits[d];
                }
                changes.add.forEach(function(a) { mergedExits[a.dir] = { RoomId: a.targetRoomId }; });
                await AdminAPI.patch('/admin/api/v1/rooms/' + rId, { Exits: mergedExits });
            }
        }

        MapperState.clearDirty();
        await MapperState.loadAllRooms();
        showLoading(false);
        MapperState.selected.clear();
        updateInfoPanel();
        MapperRender.render();
    }

    async function discardChanges() {
        MapperState.clearDirty();
        showLoading(true);
        await MapperState.loadAllRooms();
        showLoading(false);
        MapperState.selected.clear();
        updateInfoPanel();
        MapperRender.render();
    }

    // =====================================================================
    // Public API
    // =====================================================================

    return {
        init: init,
        switchTab: switchTab, zoomIn: zoomIn, zoomOut: zoomOut,
        spacingDown: spacingDown, spacingUp: spacingUp,
        centerCamera: centerCamera, setCameraTarget: setCameraTarget,
        updateZButtons: updateZButtons, updateInfoPanel: updateInfoPanel,
        showTooltip: showTooltip, positionTooltip: positionTooltip, hideTooltip: hideTooltip,
        showLoading: showLoading, updateStats: updateStats,
        populateZoneDropdown: populateZoneDropdown,
        createRoomAt: createRoomAt,
        toServerCoords: toServerCoords,
        saveAllChanges: saveAllChanges, discardChanges: discardChanges
    };

})();
