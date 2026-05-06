/**
 * mapper-ui.js -- UI controls and panels for the map editor.
 *
 * Owns the tab bar, zoom/spacing controls, camera easing, Z-level buttons,
 * room info panel, tooltip, stats line, zone dropdown, local room creation,
 * coordinate conversion, and save/discard workflow. All DOM interaction
 * funnels through a `dom` refs object set at init time so the module stays
 * decoupled from specific element IDs.
 */
/* jshint esversion: 11, browser: true */
/* globals MapperState, MapperRender, MapperEvents, MapperCtxMenu, AdminAPI,
   symbolForRoom, colorForSymbol, escapeHtml, smoothstep,
   getZonesAtPoint, closestZone,
   ZOOM_STEP, ZOOM_MIN, ZOOM_MAX, CENTER_EASE_DURATION, ROOM_SIZE_2D */
'use strict';

var MapperUI = (function() {

    // =====================================================================
    //  DOM references (set via init)
    // =====================================================================

    var dom = {
        viewport: null,
        spacingCtrlEl: null,
        zSelectEl: null,
        zPrevBtn: null,
        zNextBtn: null,
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
        if (domRefs.zSelectEl) dom.zSelectEl = domRefs.zSelectEl;
        if (domRefs.zPrevBtn) dom.zPrevBtn = domRefs.zPrevBtn;
        if (domRefs.zNextBtn) dom.zNextBtn = domRefs.zNextBtn;
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
    //  Zoom / spacing controls
    // =====================================================================

    function zoomIn() {
        MapperState.camera.zoomScale = Math.min(ZOOM_MAX, MapperState.camera.zoomScale * ZOOM_STEP);
        MapperRender.render();
    }

    function zoomOut() {
        MapperState.camera.zoomScale = Math.max(ZOOM_MIN, MapperState.camera.zoomScale / ZOOM_STEP);
        MapperRender.render();
    }

    // =====================================================================
    //  Camera centering and easing
    // =====================================================================

    function centerCamera() {
        MapperState.camera.panOffsetX = 0;
        MapperState.camera.panOffsetY = 0;
        if (MapperState.data.currentZone) {
            MapperState.centerOnZone(MapperState.data.currentZone);
        } else if (MapperState.data.zLevels.length > 0) {
            MapperState.camera.activeZ2d = MapperState.data.zLevels[0];
            updateZButtons();
            MapperRender.render();
        }
    }

    /** Smoothly animate the camera to (tx, ty, tz) using requestAnimationFrame. */
    function setCameraTarget(tx, ty, tz) {
        var cam = MapperState.camera;
        cam.panOffsetX = 0;
        cam.panOffsetY = 0;
        if (typeof tz === 'undefined') tz = cam.cameraZ;

        // Skip animation when easing is disabled
        if (CENTER_EASE_DURATION <= 0) {
            cam.cameraX = tx; cam.cameraY = ty; cam.cameraZ = tz;
            MapperRender.render();
            return;
        }

        // Cancel any in-progress ease before starting a new one
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
    //  Z-level buttons
    // =====================================================================

    function updateZButtons() {
        if (!dom.zSelectEl) return;
        var levels = MapperState.data.zLevels;
        var cam = MapperState.camera;
        var current = cam.activeZ2d;

        dom.zSelectEl.innerHTML = '';
        levels.slice().reverse().forEach(function(z) {
            var opt = document.createElement('option');
            opt.value = z;
            opt.textContent = z;
            if (z === current) opt.selected = true;
            dom.zSelectEl.appendChild(opt);
        });

        var idx = levels.indexOf(current);
        if (dom.zPrevBtn) dom.zPrevBtn.disabled = (idx < 0 || idx >= levels.length - 1);
        if (dom.zNextBtn) dom.zNextBtn.disabled = (idx <= 0);
    }

    // =====================================================================
    //  Info panel (single room detail or multi-select summary)
    // =====================================================================

    // Cache for mob names fetched on demand: mobId -> name string
    var _mobNameCache = {};
    // Track the room ID whose detail fetch is currently in flight so stale
    // responses from a previous selection are discarded.
    var _infoPanelFetchId = null;

    function moveSelectedZ(deltaZ) {
        var ids = Array.from(MapperState.selected);
        var ok = MapperState.moveRoomsZLocally(ids, deltaZ);
        if (!ok) {
            MapperState.showToast('Cannot move — room collision detected');
            return;
        }
        var newZ = null;
        for (var i = 0; i < ids.length; i++) {
            var r = MapperState.data.rooms.get(ids[i]);
            if (r && r.HasCoordinates) { newZ = r.MapZ; break; }
        }
        if (newZ !== null) {
            MapperState.camera.activeZ2d = newZ;
            updateZButtons();
        }
        updateInfoPanel();
        MapperRender.render();
    }

    async function _fetchMobName(mobId) {
        if (_mobNameCache[mobId] !== undefined) return _mobNameCache[mobId];
        var res = await AdminAPI.get('/admin/api/v1/mobs/' + mobId);
        var name = 'Mob #' + mobId;
        if (res.ok && res.data && res.data.data) {
            var mob = res.data.data;
            name = (mob.Character && mob.Character.Name) ? mob.Character.Name : name;
        }
        _mobNameCache[mobId] = name;
        return name;
    }

    function updateInfoPanel() {
        if (!dom.infoEmptyEl || !dom.infoContentEl) return;

        if (MapperState.selected.size === 0) {
            _infoPanelFetchId = null;
            dom.infoEmptyEl.style.display = '';
            dom.infoContentEl.style.display = 'none';
            return;
        }

        dom.infoEmptyEl.style.display = 'none';
        dom.infoContentEl.style.display = '';

        var html = '';

        // --- Multi-select summary ---
        if (MapperState.selected.size > 1) {
            _infoPanelFetchId = null;
            var ids = Array.from(MapperState.selected).sort(function(a, b) { return a - b; });
            html = '<div class="info-row"><span class="info-label">Selected</span><span class="info-value">' + ids.length + ' rooms</span></div>';
            html += '<hr class="info-divider">';
            ids.forEach(function(rid) {
                var r = MapperState.data.rooms.get(rid);
                var title = r ? escapeHtml(r.Title) : '?';
                html += '<div class="info-exit-row"><span class="info-exit-dir">#' + rid + '</span><span class="info-exit-id">' + title + '</span></div>';
            });
            html += '<hr class="info-divider">';
            html += '<div style="display:flex;gap:4px;margin-top:2px">';
            html += '<button class="info-zlevel-btn" id="info-move-down">&#9660; Move Down</button>';
            html += '<button class="info-zlevel-btn" id="info-move-up">&#9650; Move Up</button>';
            html += '</div>';
            dom.infoContentEl.innerHTML = html;
            document.getElementById('info-move-up').addEventListener('click', function() { moveSelectedZ(1); });
            document.getElementById('info-move-down').addEventListener('click', function() { moveSelectedZ(-1); });
            return;
        }

        // --- Single room detail: fetch full room data from server ---
        var selectedRoomId = Array.from(MapperState.selected)[0];
        var mapRoom = MapperState.data.rooms.get(selectedRoomId);
        if (!mapRoom) {
            _infoPanelFetchId = null;
            dom.infoEmptyEl.style.display = '';
            dom.infoContentEl.style.display = 'none';
            return;
        }

        // Render a skeleton immediately so the panel doesn't look empty
        dom.infoContentEl.innerHTML =
            '<div class="info-row"><span class="info-label">ID</span><span class="info-value">' + selectedRoomId + '</span></div>' +
            '<div class="info-row"><span class="info-label">Title</span><span class="info-value">' + escapeHtml(mapRoom.Title) + '</span></div>' +
            '<div class="info-empty" style="margin-top:8px">Loading details…</div>';

        // Tag this fetch so we can discard the result if the selection changes
        var fetchId = selectedRoomId;
        _infoPanelFetchId = fetchId;

        AdminAPI.get('/admin/api/v1/rooms/' + selectedRoomId).then(async function(res) {
            // Selection changed while we were waiting
            if (_infoPanelFetchId !== fetchId) return;

            var room = res.ok && res.data && res.data.data ? res.data.data : null;
            if (!room) {
                dom.infoContentEl.innerHTML += '<div style="color:#ff6b6b;font-size:0.75rem">Failed to load room details.</div>';
                return;
            }

            var html = '';
            html += '<div class="info-row"><span class="info-label">ID</span><span class="info-value">' + room.RoomId + '</span></div>';
            html += '<div class="info-row"><span class="info-label">Title</span><span class="info-value info-desc-snippet" id="info-title-link">' + escapeHtml(room.Title || '') + '</span></div>';
            html += '<div class="info-row"><span class="info-label">Zone</span><span class="info-value">' + escapeHtml(room.Zone || '-') + '</span></div>';
            html += '<div class="info-row"><span class="info-label">Biome</span><span class="info-value">' + escapeHtml(room.Biome || '-') + '</span></div>';
            if (mapRoom._symbol) {
                html += '<div class="info-row"><span class="info-label">Symbol</span><span class="info-value">' + escapeHtml(mapRoom._symbol) + '</span></div>';
            }
            if (room.MapLegend) {
                html += '<div class="info-row"><span class="info-label">Legend</span><span class="info-value">' + escapeHtml(room.MapLegend) + '</span></div>';
            }
            html += '<div class="info-row"><span class="info-label">Coords</span><span class="info-value">' + room.MapX + ', ' + room.MapY + ', ' + room.MapZ + '</span></div>';

            // Description snippet — click to open editor
            var descText = (room.Description || '').trim();
            var descSnippet = descText.length > 120 ? descText.slice(0, 120).replace(/\s+\S*$/, '') + '…' : descText;
            html += '<hr class="info-divider">';
            html += '<div class="info-row"><span class="info-label">Description</span></div>';
            html += '<div class="info-desc-snippet" id="info-desc-snippet">' + escapeHtml(descSnippet || '(none)') + '</div>';

            // Tags
            if (room.Tags && room.Tags.length > 0) {
                var tagBadges = room.Tags.map(function(t) {
                    var mod = MapperState.tagDescriptions[t];
                    var tipText = mod ? 'module: ' + mod : t;
                    return '<span class="info-badge" style="cursor:default" title="' + escapeHtml(tipText) + '">' + escapeHtml(t) + '</span>';
                }).join(' ');
                html += '<div class="info-row"><span class="info-label">Tags</span><span class="info-value info-badges">' + tagBadges + '</span></div>';
            }

            // Mob spawns: rendered as a placeholder first, then mob names filled in
            var mobSpawns = [];
            if (Array.isArray(room.SpawnInfo)) {
                room.SpawnInfo.forEach(function(s, idx) { if (s.MobId > 0) mobSpawns.push({ mid: s.MobId, idx: idx }); });
            }

            if (mobSpawns.length > 0) {
                html += '<hr class="info-divider">';
                html += '<div class="info-row info-row-block"><span class="info-label">Mob Spawns</span>';
                html += '<table class="info-spawn-table"><tbody>';
                mobSpawns.forEach(function(entry) {
                    html += '<tr><td class="info-desc-snippet" style="font-style:normal" ' +
                        'data-spawn-idx="' + entry.idx + '" ' +
                        'id="mob-name-' + entry.idx + '">' +
                        escapeHtml(_mobNameCache[entry.mid] || ('Mob #' + entry.mid)) + '</td></tr>';
                });
                html += '</tbody></table></div>';
            }

            // Exits
            if (room.Exits && Object.keys(room.Exits).length > 0) {
                html += '<hr class="info-divider">';
                html += '<div class="info-exits">';
                Object.keys(room.Exits).sort().forEach(function(dir) {
                    var ex = room.Exits[dir];
                    var badges = '';
                    if (ex.Secret) badges += '<span class="info-badge secret">secret</span> ';
                    if (ex.Lock && ex.Lock.Difficulty > 0) badges += '<span class="info-badge locked">locked</span> ';
                    html += '<div class="info-exit-row">';
                    html += '<span class="info-exit-dir">' + escapeHtml(dir) + '</span>';
                    html += '<span class="info-exit-id">' + badges + '#' + ex.RoomId + '</span>';
                    html += '</div>';
                });
                html += '</div>';
            }

            html += '<hr class="info-divider">';
            html += '<a class="info-link" href="/admin/rooms#' + room.RoomId + '">Edit Room &rarr;</a>';
            if (room.HasCoordinates) {
                html += '<hr class="info-divider">';
                html += '<div style="display:flex;gap:4px">';
                html += '<button class="info-zlevel-btn" id="info-move-down">&#9660; Move Down</button>';
                html += '<button class="info-zlevel-btn" id="info-move-up">&#9650; Move Up</button>';
                html += '</div>';
            }



            // Guard again: selection may have changed while we built the string
            if (_infoPanelFetchId !== fetchId) return;
            dom.infoContentEl.innerHTML = html;

            var upBtn = document.getElementById('info-move-up');
            var downBtn = document.getElementById('info-move-down');
            if (upBtn) upBtn.addEventListener('click', function() { moveSelectedZ(1); });
            if (downBtn) downBtn.addEventListener('click', function() { moveSelectedZ(-1); });

            // Wire title and description snippet clicks -> room editor
            var snippetEl = document.getElementById('info-desc-snippet');
            var titleLinkEl = document.getElementById('info-title-link');
            var openEditor = function() { openRoomEditor(room.RoomId, room); };
            if (snippetEl) {
                (function(fn) { snippetEl.addEventListener('click', fn); })(openEditor);
            }
            if (titleLinkEl) {
                (function(fn) { titleLinkEl.addEventListener('click', fn); })(openEditor);
            }

            // Back-fill mob names asynchronously
            if (mobSpawns.length > 0) {
                mobSpawns.forEach(function(entry) {
                    _fetchMobName(entry.mid).then(function(name) {
                        if (_infoPanelFetchId !== fetchId) return;
                        var cell = document.getElementById('mob-name-' + entry.idx);
                        if (cell) cell.textContent = name;
                    });
                });

                // Wire click on each row -> open room editor on that spawn card
                mobSpawns.forEach(function(entry) {
                    var cell = document.getElementById('mob-name-' + entry.idx);
                    if (!cell) return;
                    (function(spawnIdx, capturedRoom) {
                        cell.addEventListener('click', function() {
                            openRoomEditor(capturedRoom.RoomId, capturedRoom, spawnIdx);
                        });
                    })(entry.idx, room);
                });
            }
        });
    }

    // =====================================================================
    //  Tooltip
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

    /** Keep the tooltip on-screen by flipping sides when it would overflow. */
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

    /** Brief delay before hiding prevents flicker when moving between rooms. */
    function hideTooltip() {
        if (!dom.tooltip) return;
        var cam = MapperState.camera;
        cam.tooltipHideTimer = setTimeout(function() {
            dom.tooltip.style.display = 'none';
        }, 80);
    }

    // =====================================================================
    //  Loading indicator
    // =====================================================================

    function showLoading(show) {
        if (!dom.loadingEl) return;
        dom.loadingEl.classList.toggle('hidden', !show);
    }

    // =====================================================================
    //  Stats display
    // =====================================================================

    function updateStats() {
        if (!dom.statsEl) return;
        var data = MapperState.data;
        var total = data.rooms.size;
        var coordinated = 0;
        data.rooms.forEach(function(r) { if (r.HasCoordinates) coordinated++; });
        var unmapped = total - coordinated;
        dom.statsEl.innerHTML = '';
        var prefix = document.createTextNode(total + ' rooms');
        dom.statsEl.appendChild(prefix);
        if (unmapped > 0) {
            var link = document.createElement('span');
            link.className = 'unmapped-link';
            link.textContent = ' (' + unmapped + ' unmapped)';
            link.title = 'Click to see unmapped rooms';
            link.addEventListener('click', showUnmappedModal);
            dom.statsEl.appendChild(link);
        }
        var suffix = document.createTextNode(
            ' | ' + data.zLevels.length + ' z-level' + (data.zLevels.length !== 1 ? 's' : '') +
            ' | ' + data.allZones.length + ' zones'
        );
        dom.statsEl.appendChild(suffix);
    }

    // =====================================================================
    //  Unmapped rooms modal
    // =====================================================================

    function showUnmappedModal() {
        var backdrop = document.getElementById('unmapped-backdrop');
        var listEl = document.getElementById('unmapped-list');
        var closeBtn = document.getElementById('unmapped-close');
        if (!backdrop || !listEl) return;

        var unmapped = [];
        MapperState.data.rooms.forEach(function(room, id) {
            if (!room.HasCoordinates) {
                unmapped.push({ id: id, title: room.Title || '', zone: room.Zone || '' });
            }
        });
        unmapped.sort(function(a, b) {
            if (a.zone !== b.zone) return a.zone.localeCompare(b.zone);
            return a.id - b.id;
        });

        var html = '<table><tr><th>ID</th><th>Title</th><th>Zone</th><th></th></tr>';
        unmapped.forEach(function(r) {
            html += '<tr>' +
                '<td>' + r.id + '</td>' +
                '<td>' + escapeHtml(r.title) + '</td>' +
                '<td>' + escapeHtml(r.zone) + '</td>' +
                '<td><a href="/admin/rooms#' + r.id + '">edit</a></td>' +
                '</tr>';
        });
        html += '</table>';
        listEl.innerHTML = html;

        backdrop.classList.add('visible');

        function close() {
            backdrop.classList.remove('visible');
            backdrop.removeEventListener('click', onBackdropClick);
            closeBtn.removeEventListener('click', close);
        }
        function onBackdropClick(e) {
            if (e.target === backdrop) close();
        }
        backdrop.addEventListener('click', onBackdropClick);
        closeBtn.addEventListener('click', close);
    }

    // =====================================================================
    //  Zone dropdown
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
    //  Zone picker modal
    // =====================================================================

    var zonePickerCallback = null;

    function showZonePicker(zones, callback) {
        var backdrop = document.getElementById('zone-picker-backdrop');
        var list = document.getElementById('zone-picker-list');
        var cancelBtn = document.getElementById('zone-picker-cancel');
        if (!backdrop || !list) { callback(zones[0]); return; }

        list.innerHTML = '';
        zones.forEach(function(zone) {
            var btn = document.createElement('button');
            btn.textContent = zone;
            btn.addEventListener('click', function() {
                hideZonePicker();
                callback(zone);
            });
            list.appendChild(btn);
        });

        zonePickerCallback = null;
        cancelBtn.onclick = hideZonePicker;
        backdrop.classList.add('visible');
    }

    function hideZonePicker() {
        var backdrop = document.getElementById('zone-picker-backdrop');
        if (backdrop) backdrop.classList.remove('visible');
        zonePickerCallback = null;
    }

    /**
     * Resolves which zone a new room at (gx, gy, gz) should belong to.
     * hintZone is used as a fallback (e.g. the source room's zone in quick-build).
     * Calls callback(zoneName) asynchronously (immediately or after user picks).
     */
    function resolveZoneForRoom(gx, gy, gz, hintZone, callback) {
        var candidates = getZonesAtPoint(gx, gy, gz);
        if (candidates.length === 0) {
            // Outside all bounding boxes — use hint or closest room
            callback(hintZone || closestZone(gx, gy, gz, MapperState.data.currentZone || ''));
        } else if (candidates.length === 1) {
            callback(candidates[0]);
        } else {
            showZonePicker(candidates, callback);
        }
    }

    // =====================================================================
    //  Zone creation
    // =====================================================================

    function createZoneAt(gx, gy, gz) {
        var backdrop  = document.getElementById('zone-name-backdrop');
        var input     = document.getElementById('zone-name-input');
        var errorEl   = document.getElementById('zone-name-error');
        var confirmBtn = document.getElementById('zone-name-confirm');
        var cancelBtn  = document.getElementById('zone-name-cancel');
        if (!backdrop || !input) return;

        input.value = '';
        errorEl.textContent = '';
        backdrop.classList.add('visible');
        setTimeout(function() { input.focus(); }, 50);

        function close() {
            backdrop.classList.remove('visible');
            confirmBtn.onclick = null;
            cancelBtn.onclick = null;
            input.onkeydown = null;
        }

        function submit() {
            var name = input.value.trim();
            if (!name) { errorEl.textContent = 'Zone name is required.'; return; }
            if (name.length < 2) { errorEl.textContent = 'Zone name must be at least 2 characters.'; return; }
            var nameLower = name.toLowerCase();
            var exists = MapperState.data.allZones.some(function(z) { return z.Name.toLowerCase() === nameLower; });
            if (exists) { errorEl.textContent = 'A zone with that name already exists.'; return; }

            errorEl.textContent = '';
            confirmBtn.disabled = true;
            confirmBtn.textContent = 'Creating…';

            AdminAPI.post('/admin/api/v1/zones', { Name: name })
                .then(function(res) {
                    confirmBtn.disabled = false;
                    confirmBtn.textContent = 'Create Zone';
                    if (!res.ok || !res.data || !res.data.data) {
                        errorEl.textContent = (res.data && res.data.error) || 'Failed to create zone.';
                        return;
                    }
                    var serverRootId = res.data.data.RoomId;

                    // Immediately patch the root room to the clicked coordinates
                    // so the server state matches local state without requiring a save.
                    AdminAPI.patch('/admin/api/v1/rooms/' + serverRootId, {
                        MapX: gx, MapY: gy, MapZ: gz, HasCoordinates: true
                    }).then(function() {
                        close();

                        // Zone now exists on the server — register it in local state
                        MapperState.data.allZones.push({ Name: name, RoomCount: 1, RoomId: serverRootId, DefaultBiome: '' });
                        populateZoneDropdown();

                        // Create a local room at the chosen position assigned to the new zone.
                        // Replace its temp ID with the server root room ID so saves patch
                        // the real room rather than creating a duplicate.
                        var tempId = MapperState.createRoomLocally(gx, gy, gz, name);
                        MapperState.replaceRoomId(tempId, serverRootId);
                        // The room is already saved at the correct position — remove it
                        // from dirty.createdRooms so save doesn't patch it again.
                        MapperState.dirty.createdRooms.delete(serverRootId);
                        MapperState.selectRoom(serverRootId);

                        if (dom.zoneSelect) {
                            dom.zoneSelect.value = name;
                            dom.zoneSelect.dispatchEvent(new Event('change'));
                        }
                        MapperRender.render();
                    }).catch(function() {
                        close();
                        errorEl.textContent = 'Zone created but failed to set room coordinates.';
                    });
                })
                .catch(function() {
                    confirmBtn.disabled = false;
                    confirmBtn.textContent = 'Create Zone';
                    errorEl.textContent = 'Request failed.';
                });
        }

        confirmBtn.onclick = submit;
        cancelBtn.onclick = close;
        input.onkeydown = function(e) {
            if (e.key === 'Enter') submit();
            if (e.key === 'Escape') close();
        };
    }

    // =====================================================================
    //  Local room creation (deferred until save)
    // =====================================================================

    function createRoomAt(gx, gy, gz) {
        if (!MapperState.data.currentZone && MapperState.data.rooms.size === 0) {
            alert('Select a zone from the dropdown first to set which zone new rooms are created in.');
            return;
        }
        resolveZoneForRoom(gx, gy, gz, MapperState.data.currentZone, function(zone) {
            var tempId = MapperState.createRoomLocally(gx, gy, gz, zone);
            MapperState.selectRoom(tempId);
            MapperRender.render();
        });
    }

    // =====================================================================
    //  Server coordinate conversion
    // =====================================================================

    /**
     * Translate world-space coordinates back to zone-relative server
     * coordinates by subtracting the zone offset applied at load time.
     */
    function toServerCoords(room) {
        if (!room) return { x: 0, y: 0, z: 0 };
        return { x: room.MapX, y: room.MapY, z: room.MapZ };
    }

    // =====================================================================
    //  Save / Discard
    // =====================================================================

    async function saveAllChanges() {
        var data = MapperState.data;
        var dirty = MapperState.dirty;
        showLoading(true);

        // 1. Delete rooms on server
        for (var i = 0; i < dirty.deletedRooms.length; i++) {
            await AdminAPI.delete('/admin/api/v1/rooms/' + dirty.deletedRooms[i]);
        }

        // 2. Create new rooms on server and build a temp-to-real ID map.
        //    Rooms whose IDs are already positive were pre-created (e.g. zone root
        //    rooms) and only need a position patch, not a new POST.
        var tempToReal = new Map();
        for (var entry of dirty.createdRooms) {
            var tempId = entry[0], info = entry[1];
            var zone = info.zone || data.currentZone || '';
            var serverX = info.gx, serverY = info.gy;
            var tempRoom = data.rooms.get(tempId);
            var realId = tempId > 0 ? tempId : null;
            if (realId === null) {
                var createRes = await AdminAPI.post('/admin/api/v1/rooms', { Zone: zone });
                if (createRes.ok && createRes.data && createRes.data.data) {
                    realId = createRes.data.data.RoomId;
                }
            }
            if (realId !== null) {
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
                var moveData = {
                    MapX: sc.x, MapY: sc.y, MapZ: sc.z, HasCoordinates: true,
                    Exits: room.Exits || {}
                };
                await AdminAPI.patch('/admin/api/v1/rooms/' + roomId, moveData);
            }
        }

        // 4. Apply exit changes (grouped by room: removals then additions)
        //    We merge against the server's current exit map so we don't
        //    clobber exits that were not touched locally.
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
        var cam = MapperState.camera;
        var savedCamX = cam.cameraX, savedCamY = cam.cameraY;
        var savedPanX = cam.panOffsetX, savedPanY = cam.panOffsetY;
        var savedZ = cam.activeZ2d;
        await MapperState.loadAllRooms();
        cam.cameraX = savedCamX; cam.cameraY = savedCamY;
        cam.panOffsetX = savedPanX; cam.panOffsetY = savedPanY;
        cam.activeZ2d = savedZ;
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
    //  Spawn card helpers (room editor)
    // =====================================================================

    function _reEsc(s) {
        return String(s || '').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');
    }

    function _reSpawnSummaryFromData(type, s) {
        if (type === 'mob') {
            var displayName = (s.Name && s.Name.trim()) || s._mobName || (s.MobId ? 'Mob #' + s.MobId : 'No mob selected');
            return displayName + (s.RespawnRate ? '  —  ' + s.RespawnRate : '');
        }
        var parts = [];
        if (s.ItemId) parts.push('Item #' + s.ItemId);
        if (s.Gold)   parts.push(s.Gold + ' gold');
        if (!parts.length) parts.push('Empty spawn');
        if (s.RespawnRate) parts.push(s.RespawnRate);
        return parts.join('  —  ');
    }

    function _reSpawnBuildHeaderSummary(card) {
        var type = card.dataset.spawnType;
        if (type === 'mob') {
            var hiddenInput = card.querySelector('.re-spawn-mobid');
            var mobId = hiddenInput ? parseInt(hiddenInput.value, 10) : 0;
            var mobName = hiddenInput ? (hiddenInput.dataset.mobName || '') : '';
            var nameOverride = card.querySelector('.re-spawn-name');
            var displayName = (nameOverride && nameOverride.value.trim()) || mobName || (mobId ? 'Mob #' + mobId : 'No mob selected');
            var rate = card.querySelector('.re-spawn-respawnrate');
            return displayName + (rate && rate.value.trim() ? '  —  ' + rate.value.trim() : '');
        }
        var parts = [];
        var itemId = parseInt((card.querySelector('.re-spawn-itemid') || {}).value || 0, 10);
        var gold   = parseInt((card.querySelector('.re-spawn-gold')   || {}).value || 0, 10);
        if (itemId) parts.push('Item #' + itemId);
        if (gold)   parts.push(gold + ' gold');
        if (!parts.length) parts.push('Empty spawn');
        var rateEl = card.querySelector('.re-spawn-respawnrate');
        if (rateEl && rateEl.value.trim()) parts.push(rateEl.value.trim());
        return parts.join('  —  ');
    }

    function _reSpawnUpdateHeader(card) {
        var el = card.querySelector('.re-spawn-summary');
        if (el) el.textContent = _reSpawnBuildHeaderSummary(card);
    }

    function _reBuffChipHtml(buffId) {
        var name = PickerConfigs.buffName(buffId);
        return '<span class="re-buff-chip" data-buff-id="' + buffId + '">' + _reEsc(name) +
            ' <button type="button" title="Remove" onclick="this.closest(\'[data-buff-id]\').remove()">&times;</button></span>';
    }

    function _reQuestChipHtml(token) {
        return '<span class="re-buff-chip" data-quest-token="' + _reEsc(token) + '">' + _reEsc(token) +
            ' <button type="button" title="Remove" onclick="this.closest(\'[data-quest-token]\').remove()">&times;</button></span>';
    }

    function _reSpawnAppendCard(list, s, openOnAdd) {
        s = s || {};
        var type = s._type || (s.MobId ? 'mob' : 'item');
        var isMob = type === 'mob';
        var shown = '', hidden = 'display:none;';

        var card = document.createElement('div');
        card.className = 're-spawn-card';
        card.dataset.spawnType = type;

        var summary = _reSpawnSummaryFromData(type, s);
        // Pre-compute buff/quest chip html
        var buffChips = (s.BuffIds || []).map(_reBuffChipHtml).join('');
        var questChips = (s.QuestFlags || []).map(_reQuestChipHtml).join('');

        card.innerHTML =
            '<div class="re-spawn-header" onclick="this.classList.toggle(\'open\');this.nextElementSibling.classList.toggle(\'open\')">'+
                '<span class="re-spawn-badge ' + (isMob ? 're-spawn-badge-mob' : 're-spawn-badge-item') + '">' + (isMob ? 'Mob' : 'Item / Gold') + '</span>'+
                '<span class="re-spawn-summary">' + _reEsc(summary) + '</span>'+
                '<button class="re-spawn-del" type="button" title="Remove spawn" onclick="this.closest(\'.re-spawn-card\').remove();event.stopPropagation()">&times;</button>'+
            '</div>'+
            '<div class="re-spawn-body">'+
            '<div class="re-spawn-grid">'+

            // --- Mob picker ---
            '<div class="re-spawn-field span2" style="' + (isMob ? shown : hidden) + '">'+
                '<label>Mob</label>'+
                '<div class="re-spawn-picker-wrap">'+
                    '<span class="re-spawn-picker-display ' + (s.MobId ? '' : 'empty') + '">' + (s.MobId ? '#' + s.MobId : 'No mob selected') + '</span>'+
                    '<input type="hidden" class="re-spawn-mobid" value="' + (s.MobId || 0) + '" data-mob-name="">'+
                    '<button type="button" class="room-editor-noun-add" style="width:auto;margin:0" onclick="MapperUI._rePickMob(this)">Pick Mob…</button>'+
                '</div>'+
                '<div class="field-hint">The mob template to spawn in this room.</div>'+
            '</div>'+

            // --- Item picker ---
            '<div class="re-spawn-field span2" style="' + (isMob ? hidden : shown) + '">'+
                '<label>Item</label>'+
                '<div class="re-spawn-picker-wrap">'+
                    '<span class="re-spawn-picker-display ' + (s.ItemId ? '' : 'empty') + '">' + (s.ItemId ? '#' + s.ItemId : 'No item (gold only)') + '</span>'+
                    '<input type="hidden" class="re-spawn-itemid" value="' + (s.ItemId || 0) + '">'+
                    '<button type="button" class="room-editor-noun-add" style="width:auto;margin:0" onclick="MapperUI._rePickItem(this)">Pick Item…</button>'+
                    '<button type="button" class="re-spawn-del" style="margin-left:0" title="Clear item" onclick="MapperUI._reClearItem(this)">&times;</button>'+
                '</div>'+
                '<div class="field-hint">Leave unset to spawn gold only.</div>'+
            '</div>'+

            // --- Gold ---
            '<div class="re-spawn-field" style="' + (isMob ? hidden : shown) + '">'+
                '<label>Gold</label>'+
                '<input type="number" class="re-spawn-gold" min="0" value="' + (s.Gold || 0) + '">'+
                '<div class="field-hint">Gold coins to spawn.</div>'+
            '</div>'+

            // --- Container ---
            '<div class="re-spawn-field" style="' + (isMob ? hidden : shown) + '">'+
                '<label>Container</label>'+
                '<input type="text" class="re-spawn-container" value="' + _reEsc(s.Container || '') + '" placeholder="e.g. chest">'+
                '<div class="field-hint">Leave blank to spawn on floor.</div>'+
            '</div>'+

            // --- Respawn Rate ---
            '<div class="re-spawn-field span2">'+
                '<label>Respawn Rate</label>'+
                '<input type="text" class="re-spawn-respawnrate" value="' + _reEsc(s.RespawnRate || '') + '" placeholder="e.g. 15 real minutes, 1 hour">'+
                '<div class="field-hint">How long before this spawns again. Defaults to 15 real minutes if blank.</div>'+
            '</div>'+

            // --- Name Override (mob) ---
            '<div class="re-spawn-field" style="' + (isMob ? shown : hidden) + '">'+
                '<label>Name Override</label>'+
                '<input type="text" class="re-spawn-name" value="' + _reEsc(s.Name || '') + '" placeholder="Leave blank for mob default">'+
                '<div class="field-hint">Replaces the mob\'s default name.</div>'+
            '</div>'+

            // --- Spawn Message (mob) ---
            '<div class="re-spawn-field" style="' + (isMob ? shown : hidden) + '">'+
                '<label>Spawn Message</label>'+
                '<input type="text" class="re-spawn-message" value="' + _reEsc(s.Message || '') + '" placeholder="Shown to room on spawn">'+
                '<div class="field-hint">Replaces the default spawn announcement.</div>'+
            '</div>'+

            // --- Force Hostile (mob) ---
            '<div class="re-spawn-field" style="' + (isMob ? shown : hidden) + '">'+
                '<label>Force Hostile</label>'+
                '<div class="re-spawn-toggle-wrap">'+
                    '<label class="re-spawn-toggle">'+
                        '<input type="checkbox" class="re-spawn-forcehostile"' + (s.ForceHostile ? ' checked' : '') + '>'+
                        '<span class="re-spawn-toggle-track"></span>'+
                    '</label>'+
                    '<span class="re-spawn-toggle-label">Mob attacks on sight</span>'+
                '</div>'+
            '</div>'+

            // --- Max Wander (mob) ---
            '<div class="re-spawn-field" style="' + (isMob ? shown : hidden) + '">'+
                '<label>Max Wander</label>'+
                '<input type="number" class="re-spawn-maxwander" value="' + (s.MaxWander != null && s.MaxWander !== 0 ? s.MaxWander : '') + '" placeholder="blank = mob default">'+
                '<div class="field-hint">Rooms mob can wander from spawn. 0 = stays put, -1 = unlimited.</div>'+
            '</div>'+

            // --- Force Level (mob) ---
            '<div class="re-spawn-field" style="' + (isMob ? shown : hidden) + '">'+
                '<label>Force Level</label>'+
                '<input type="number" class="re-spawn-level" min="0" value="' + (s.Level || '') + '" placeholder="0 = mob default">'+
                '<div class="field-hint">Force to a specific level. 0 = use mob template level.</div>'+
            '</div>'+

            // --- Level Modifier (mob) ---
            '<div class="re-spawn-field" style="' + (isMob ? shown : hidden) + '">'+
                '<label>Level Modifier</label>'+
                '<input type="number" class="re-spawn-levelmod" value="' + (s.LevelMod || 0) + '" placeholder="0 = no change">'+
                '<div class="field-hint">Added to resolved level. Can be negative.</div>'+
            '</div>'+

            // --- Script Tag (mob) ---
            '<div class="re-spawn-field" style="' + (isMob ? shown : hidden) + '">'+
                '<label>Script Tag</label>'+
                '<input type="text" class="re-spawn-scripttag" value="' + _reEsc(s.ScriptTag || '') + '" placeholder="Overrides mob script tag">'+
                '<div class="field-hint">Leave blank for mob template default.</div>'+
            '</div>'+

            // --- Quest Flags (mob) ---
            '<div class="re-spawn-field span2" style="' + (isMob ? shown : hidden) + '">'+
                '<label>Quest Flags</label>'+
                '<div class="re-buff-chips-wrap re-spawn-questflags">' + questChips + '</div>'+
                '<button type="button" class="room-editor-noun-add" style="margin-top:4px" onclick="MapperUI._rePickQuestFlag(this)">+ Add Quest Flag</button>'+
                '<div class="field-hint">Quest flag tokens assigned to this mob instance.</div>'+
            '</div>'+

            // --- Buff IDs (mob) ---
            '<div class="re-spawn-field span2" style="' + (isMob ? shown : hidden) + '">'+
                '<label>Permanent Buffs</label>'+
                '<div class="re-buff-chips-wrap re-spawn-buffids">' + buffChips + '</div>'+
                '<button type="button" class="room-editor-noun-add" style="margin-top:4px" onclick="MapperUI._rePickBuff(this)">+ Pick Buffs…</button>'+
                '<div class="field-hint">Buffs always active on this mob instance.</div>'+
            '</div>'+

            // --- Idle Commands (mob) ---
            '<div class="re-spawn-field span2" style="' + (isMob ? shown : hidden) + '">'+
                '<label>Idle Commands</label>'+
                '<textarea class="re-spawn-idlecommands" rows="3" style="width:100%;resize:vertical">' +
                    _reEsc((s.IdleCommands || []).join('\n')) +
                '</textarea>'+
                '<div class="field-hint">One command per line. Executed randomly when mob is idle.</div>'+
            '</div>'+

            '</div>'+ // end grid
            '</div>';  // end body

        // Auto-open when freshly added
        if (openOnAdd) {
            card.querySelector('.re-spawn-header').classList.add('open');
            card.querySelector('.re-spawn-body').classList.add('open');
        }

        // Wire respawn-rate and name inputs to update header summary
        ['re-spawn-respawnrate', 're-spawn-name'].forEach(function(cls) {
            var el = card.querySelector('.' + cls);
            if (el) el.addEventListener('input', function() { _reSpawnUpdateHeader(card); });
        });

        // Resolve mob name asynchronously for existing spawns
        if (!openOnAdd && s.MobId) {
            AdminAPI.get('/admin/api/v1/mobs/' + s.MobId).then(function(res) {
                if (!res.ok || !res.data || !res.data.data) return;
                var mob = res.data.data;
                var name = (mob.Character && mob.Character.Name) || '';
                if (!name) return;
                var hi = card.querySelector('.re-spawn-mobid');
                if (hi) hi.dataset.mobName = name;
                var disp = card.querySelector('.re-spawn-picker-display');
                if (disp) disp.textContent = '#' + s.MobId + ' ' + name;
                _reSpawnUpdateHeader(card);
            });
        }

        list.appendChild(card);
        return card;
    }

    function _reCollectSpawns(list) {
        var spawns = [];
        list.querySelectorAll('.re-spawn-card').forEach(function(card) {
            var type = card.dataset.spawnType;
            var s = {};
            if (type === 'mob') {
                var mobId = parseInt(card.querySelector('.re-spawn-mobid').value, 10);
                if (!mobId) return;
                s.MobId = mobId;
                var name = card.querySelector('.re-spawn-name').value.trim();
                if (name) s.Name = name;
                var msg = card.querySelector('.re-spawn-message').value.trim();
                if (msg) s.Message = msg;
                if (card.querySelector('.re-spawn-forcehostile').checked) s.ForceHostile = true;
                var mwRaw = card.querySelector('.re-spawn-maxwander').value.trim();
                if (mwRaw !== '') { var mw = parseInt(mwRaw, 10); if (!isNaN(mw)) s.MaxWander = mw; }
                var level = parseInt(card.querySelector('.re-spawn-level').value, 10);
                if (level) s.Level = level;
                var lmod = parseInt(card.querySelector('.re-spawn-levelmod').value, 10);
                if (lmod) s.LevelMod = lmod;
                var stag = card.querySelector('.re-spawn-scripttag').value.trim();
                if (stag) s.ScriptTag = stag;
                var qf = Array.from(card.querySelectorAll('.re-spawn-questflags [data-quest-token]')).map(function(c) { return c.dataset.questToken; }).filter(Boolean);
                if (qf.length) s.QuestFlags = qf;
                var bi = Array.from(card.querySelectorAll('.re-spawn-buffids [data-buff-id]')).map(function(c) { return parseInt(c.dataset.buffId, 10); }).filter(function(n) { return !isNaN(n); });
                if (bi.length) s.BuffIds = bi;
                var ic = card.querySelector('.re-spawn-idlecommands').value.split('\n').map(function(l) { return l.trim(); }).filter(Boolean);
                if (ic.length) s.IdleCommands = ic;
            } else {
                var itemId = parseInt(card.querySelector('.re-spawn-itemid').value, 10);
                var gold   = parseInt(card.querySelector('.re-spawn-gold').value, 10);
                if (!itemId && !gold) return;
                if (itemId) s.ItemId = itemId;
                if (gold)   s.Gold = gold;
                var container = card.querySelector('.re-spawn-container').value.trim();
                if (container) s.Container = container;
            }
            var rate = card.querySelector('.re-spawn-respawnrate').value.trim();
            if (rate) s.RespawnRate = rate;
            spawns.push(s);
        });
        return spawns;
    }

    // Picker callbacks exposed on MapperUI so inline onclick attributes can reach them
    // (assigned to the return object at the bottom of this module)
    function _rePickMob(btn) {
        var card = btn.closest('.re-spawn-card');
        Picker.open(Object.assign({}, PickerConfigs.mobs, {
            onSelect: function(mob) {
                var name = (mob.Character && mob.Character.Name) || 'Mob #' + mob.MobId;
                card.querySelector('.re-spawn-mobid').value = mob.MobId;
                card.querySelector('.re-spawn-mobid').dataset.mobName = name;
                var disp = card.querySelector('.re-spawn-picker-display');
                disp.textContent = '#' + mob.MobId + ' ' + name;
                disp.classList.remove('empty');
                _reSpawnUpdateHeader(card);
            }
        }));
    }

    function _rePickItem(btn) {
        var card = btn.closest('.re-spawn-card');
        Picker.open(Object.assign({}, PickerConfigs.items, {
            onSelect: function(item) {
                card.querySelector('.re-spawn-itemid').value = item.ItemId;
                var disp = card.querySelector('.re-spawn-picker-display');
                disp.textContent = '#' + item.ItemId + ' ' + item.Name;
                disp.classList.remove('empty');
                _reSpawnUpdateHeader(card);
            }
        }));
    }

    function _reClearItem(btn) {
        var card = btn.closest('.re-spawn-card');
        card.querySelector('.re-spawn-itemid').value = 0;
        var disp = card.querySelector('.re-spawn-picker-display');
        disp.textContent = 'No item (gold only)';
        disp.classList.add('empty');
        _reSpawnUpdateHeader(card);
    }

    function _rePickQuestFlag(btn) {
        var card = btn.closest('.re-spawn-card');
        var wrap = card.querySelector('.re-spawn-questflags');
        QuestTokenPicker.pick(function(token) {
            var existing = Array.from(wrap.querySelectorAll('[data-quest-token]')).map(function(c) { return c.dataset.questToken; });
            if (existing.indexOf(token) !== -1) return;
            wrap.insertAdjacentHTML('beforeend', _reQuestChipHtml(token));
        });
    }

    function _rePickBuff(btn) {
        var card = btn.closest('.re-spawn-card');
        var wrap = card.querySelector('.re-spawn-buffids');
        var existing = Array.from(wrap.querySelectorAll('[data-buff-id]')).map(function(c) { return parseInt(c.dataset.buffId, 10); });
        Picker.open(Object.assign({}, PickerConfigs.buffs, {
            multi:    true,
            selected: existing,
            onSelect: function(buffs) {
                wrap.innerHTML = buffs.map(function(b) { return _reBuffChipHtml(b.BuffId); }).join('');
            }
        }));
    }

    // =====================================================================
    //  Room editor modal
    // =====================================================================

    var _roomEditorRoomId = null;  // room currently loaded in the editor
    var _roomEditorData   = null;  // full room object fetched for the editor

    function _roomEditorNounRowHtml(keyword, description) {
        // Returns an HTML string for one noun row (used for initial render).
        // Rows are built as DOM nodes when adding dynamically.
        return '<div class="room-editor-noun-row">' +
            '<input class="room-editor-noun-input noun-key" type="text" placeholder="keyword" value="' + escapeHtml(keyword) + '" autocomplete="off">' +
            '<input class="room-editor-noun-input noun-desc" type="text" placeholder="description" value="' + escapeHtml(description) + '" autocomplete="off">' +
            '<button class="room-editor-noun-del" title="Remove noun">&times;</button>' +
            '</div>';
    }

    function _roomEditorAddNounRow(container, keyword, desc) {
        var row = document.createElement('div');
        row.className = 'room-editor-noun-row';
        row.innerHTML =
            '<input class="room-editor-noun-input noun-key" type="text" placeholder="keyword" value="' + escapeHtml(keyword || '') + '" autocomplete="off">' +
            '<input class="room-editor-noun-input noun-desc" type="text" placeholder="description" value="' + escapeHtml(desc || '') + '" autocomplete="off">' +
            '<button class="room-editor-noun-del" title="Remove noun">&times;</button>';
        row.querySelector('.room-editor-noun-del').addEventListener('click', function() {
            row.remove();
        });
        container.appendChild(row);
        return row;
    }

    function openRoomEditor(roomId, roomData, focusSpawnIdx) {
        _roomEditorRoomId = roomId;
        _roomEditorData   = roomData;

        var backdrop   = document.getElementById('room-editor-backdrop');
        var idEl       = document.getElementById('room-editor-id');
        var titleEl    = document.getElementById('room-editor-title');
        var descEl     = document.getElementById('room-editor-desc');
        var nounRows   = document.getElementById('room-editor-noun-rows');
        var spawnList  = document.getElementById('room-editor-spawn-list');
        var errorEl    = document.getElementById('room-editor-error');
        var saveBtn    = document.getElementById('room-editor-save');
        var cancelBtn  = document.getElementById('room-editor-cancel');
        var closeBtn   = document.getElementById('room-editor-close');
        var addNounBtn = document.getElementById('room-editor-noun-add');
        if (!backdrop) return;

        // Populate fields
        idEl.textContent  = '#' + roomId;
        titleEl.value     = roomData.Title || '';
        descEl.value      = roomData.Description || '';
        errorEl.textContent = '';
        saveBtn.disabled  = false;
        saveBtn.textContent = 'Save Changes';

        // Populate noun rows
        nounRows.innerHTML = '';
        var nouns = roomData.Nouns || {};
        Object.keys(nouns).sort().forEach(function(k) {
            _roomEditorAddNounRow(nounRows, k, nouns[k]);
        });

        // Populate spawn cards
        spawnList.innerHTML = '';
        (roomData.SpawnInfo || []).forEach(function(s) {
            _reSpawnAppendCard(spawnList, s, false);
        });

        backdrop.classList.add('visible');
        setTimeout(function() {
            if (focusSpawnIdx !== undefined && focusSpawnIdx !== null) {
                // Open and scroll to the targeted spawn card
                var cards = spawnList.querySelectorAll('.re-spawn-card');
                var target = cards[focusSpawnIdx];
                if (target) {
                    var header = target.querySelector('.re-spawn-header');
                    var body   = target.querySelector('.re-spawn-body');
                    if (header && !header.classList.contains('open')) {
                        header.classList.add('open');
                        body.classList.add('open');
                    }
                    target.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
                }
            } else {
                titleEl.focus();
                titleEl.select();
            }
        }, 60);

        // Wire add-noun button (replace listener to avoid duplicates)
        var newAddBtn = addNounBtn.cloneNode(true);
        addNounBtn.parentNode.replaceChild(newAddBtn, addNounBtn);
        newAddBtn.addEventListener('click', function() {
            var row = _roomEditorAddNounRow(nounRows, '', '');
            row.querySelector('.noun-key').focus();
        });

        // Wire add-spawn buttons (replace to avoid stacking)
        function rewireSpawnBtn(id, type) {
            var el = document.getElementById(id);
            if (!el) return;
            var clone = el.cloneNode(true);
            el.parentNode.replaceChild(clone, el);
            clone.addEventListener('click', function() {
                _reSpawnAppendCard(spawnList, { _type: type }, true);
            });
        }
        rewireSpawnBtn('re-spawn-add-mob',  'mob');
        rewireSpawnBtn('re-spawn-add-item', 'item');

        function close() {
            backdrop.classList.remove('visible');
            _roomEditorRoomId = null;
            _roomEditorData   = null;
        }

        async function save() {
            errorEl.textContent = '';
            var newTitle = titleEl.value.trim();
            var newDesc  = descEl.value.trim();
            if (!newTitle) { errorEl.textContent = 'Title cannot be empty.'; titleEl.focus(); return; }
            if (!newDesc)  { errorEl.textContent = 'Description cannot be empty.'; descEl.focus(); return; }

            // Collect nouns from rows
            var newNouns = {};
            var rows = nounRows.querySelectorAll('.room-editor-noun-row');
            var nounError = false;
            rows.forEach(function(r) {
                var k = r.querySelector('.noun-key').value.trim();
                var v = r.querySelector('.noun-desc').value.trim();
                if (!k) return;
                if (newNouns[k] !== undefined) { errorEl.textContent = 'Duplicate noun keyword: "' + k + '".'; nounError = true; }
                newNouns[k] = v;
            });
            if (nounError) return;

            // Collect spawn cards
            var newSpawns = _reCollectSpawns(spawnList);

            saveBtn.disabled = true;
            saveBtn.textContent = 'Saving…';

            var patch = Object.assign({}, _roomEditorData, {
                Title:       newTitle,
                Description: newDesc,
                Nouns:       Object.keys(newNouns).length > 0 ? newNouns : null,
                SpawnInfo:   newSpawns.length > 0 ? newSpawns : null
            });

            var res = await AdminAPI.patch('/admin/api/v1/rooms/' + _roomEditorRoomId, patch);
            saveBtn.disabled = false;
            saveBtn.textContent = 'Save Changes';

            if (!res.ok) {
                errorEl.textContent = (res.data && res.data.error) || 'Save failed.';
                return;
            }

            // Update the in-memory map room title so the panel and changelog reflect it
            var mapRoom = MapperState.data.rooms.get(_roomEditorRoomId);
            if (mapRoom) mapRoom.Title = newTitle;

            // Update the cached room data with the server's response
            if (res.data && res.data.data) _roomEditorData = res.data.data;

            close();
            // Refresh the info panel with fresh data
            updateInfoPanel();
            MapperRender.render();
        }

        // Replace save/cancel/close listeners to avoid stacking handlers
        function rewire(el, handler) {
            var clone = el.cloneNode(true);
            el.parentNode.replaceChild(clone, el);
            clone.addEventListener('click', handler);
            return clone;
        }
        rewire(document.getElementById('room-editor-save'),   save);
        rewire(document.getElementById('room-editor-cancel'), close);
        rewire(document.getElementById('room-editor-close'),  close);

        // Close on backdrop click
        backdrop.onclick = function(e) { if (e.target === backdrop) close(); };

        // Keyboard: Escape closes, Ctrl+Enter saves
        backdrop._keyHandler && document.removeEventListener('keydown', backdrop._keyHandler);
        backdrop._keyHandler = function(e) {
            if (!backdrop.classList.contains('visible')) return;
            if (e.key === 'Escape') { e.stopPropagation(); close(); }
            if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') { e.preventDefault(); save(); }
        };
        document.addEventListener('keydown', backdrop._keyHandler);
    }

    // =====================================================================
    //  Public API
    // =====================================================================

    return {
        init: init,
        zoomIn: zoomIn, zoomOut: zoomOut,
        centerCamera: centerCamera, setCameraTarget: setCameraTarget,
        updateZButtons: updateZButtons, updateInfoPanel: updateInfoPanel,
        showTooltip: showTooltip, positionTooltip: positionTooltip, hideTooltip: hideTooltip,
        showLoading: showLoading, updateStats: updateStats,
        populateZoneDropdown: populateZoneDropdown,
        createRoomAt: createRoomAt,
        createZoneAt: createZoneAt,
        resolveZoneForRoom: resolveZoneForRoom,
        hideZonePicker: hideZonePicker,
        toServerCoords: toServerCoords,
        openRoomEditor: openRoomEditor,
        // Spawn picker callbacks (called from inline onclick in dynamically built cards)
        _rePickMob:       _rePickMob,
        _rePickItem:      _rePickItem,
        _reClearItem:     _reClearItem,
        _rePickQuestFlag: _rePickQuestFlag,
        _rePickBuff:      _rePickBuff,
        saveAllChanges: saveAllChanges, discardChanges: discardChanges
    };

})();
