/* jshint esversion: 11, browser: true */
/* globals MapperEvents, AdminAPI, symbolForRoom, colorForSymbol, bgColorForBiome, contrastColor, escapeHtml, isDirectionalExit, DIRECTION_DELTAS, isExitConstraintSatisfied, buildDragConstraints, breakExitLocally, BIOME_SYMBOLS, BIOME_COLORS, BIOME_BG_COLORS, BIOME_SYMBOL_OVERRIDES, biomeEnvMap, invalidateZoneBoundsCache */
'use strict';

/**
 * MapperState — single source of truth for the admin map editor.
 *
 * Owns all mutable state: room/zone data, camera, selection, dirty-change
 * tracking, and the various modal interaction modes (exit-draw, quick-build,
 * room-drag).  Renderer and input modules read from and write to these
 * objects; MapperState never touches the DOM beyond the changelog panel.
 *
 * Public API (returned IIFE object):
 *   State objects  — data, selected, dirty, camera, roomDrag, exitDrawMode,
 *                    quickBuildMode, hoveredRoomId, hoveredGridCell, selRect,
 *                    mouseState
 *   Mutations      — isDirty, updateSaveButtons, logChange, roomLabel,
 *                    clearDirty, moveRoomLocally, breakExitLocally,
 *                    addExitLocally, deleteRoomLocally, createRoomLocally,
 *                    applyGroupMove, selectRoom, toggleRoomSelection
 *   Data loading   — loadBiomes, loadAllRooms,
 *                    applyZoneLayout, centerOnZone
 *   Registration   — setDom, setRenderFn, setUpdateFns
 */
var MapperState = (function() {

    // --- Callback Registrations ---
    // Break circular dependencies: other modules hand us their functions at
    // init time so we can trigger renders and UI updates without importing
    // those modules directly.

    var _render = function() {};
    var _updateInfoPanel = function() {};
    var _updateStats = function() {};
    var _updateZButtons = function() {};
    var _showLoading = function() {};

    function setRenderFn(fn) { _render = fn; }
    function setUpdateFns(fns) {
        if (fns.updateInfoPanel) _updateInfoPanel = fns.updateInfoPanel;
        if (fns.updateStats) _updateStats = fns.updateStats;
        if (fns.updateZButtons) _updateZButtons = fns.updateZButtons;
        if (fns.showLoading) _showLoading = fns.showLoading;
    }

    // --- DOM References ---

    var dom = {
        saveCtrlEl: null,
        changelogEl: null,
        clEntriesEl: null,
        changelogBtnEl: null,
        changelogBadgeEl: null,
        toastContainerEl: null
    };

    function setDom(domRefs) {
        if (domRefs.saveCtrlEl) dom.saveCtrlEl = domRefs.saveCtrlEl;
        if (domRefs.changelogEl) dom.changelogEl = domRefs.changelogEl;
        if (domRefs.clEntriesEl) dom.clEntriesEl = domRefs.clEntriesEl;
        if (domRefs.changelogBtnEl) dom.changelogBtnEl = domRefs.changelogBtnEl;
        if (domRefs.changelogBadgeEl) dom.changelogBadgeEl = domRefs.changelogBadgeEl;
        if (domRefs.toastContainerEl) dom.toastContainerEl = domRefs.toastContainerEl;
    }

    // --- Data Layer ---

    var tagDescriptions = {};  // tag -> module

    var mapperData = {
        allZones: [],
        currentZone: null,
        rawRooms: [],
        rooms: new Map(),
        roomsByCoord: new Map(),
        zLevels: [],
        zoneRootRooms: new Map()
    };

    // --- Selection State ---

    var selectedRoomIds = new Set();
    var hoveredRoomId = null;
    var hoveredGridCell = null; // { gx, gy } when hovering an empty 2D cell
    var selRect = { active: false, startCx: 0, startCy: 0, endCx: 0, endCy: 0 };

    // --- Dirty Tracking ---
    // Accumulates local edits that haven't been persisted to the server yet.
    // Each category is tracked separately so the save handler can batch them
    // into the right API calls.

    var dirty = {
        movedRooms: new Map(),      // roomId -> { origGx, origGy, origGz }
        exitRemovals: [],           // [{ roomId, dir }]
        exitAdditions: [],          // [{ roomId, dir, targetRoomId }]
        deletedRooms: [],           // [roomId]
        createdRooms: new Map(),    // tempId -> { gx, gy, gz, zone }
        nextTempId: -1
    };

    // --- Exit Draw Mode ---

    var exitDrawMode = {
        active: false,
        sourceRoomId: null,
        sourceGx: 0, sourceGy: 0, sourceGz: 0,
        hoveredTargetId: null,
        _mouseCx: 0, _mouseCy: 0
    };

    // --- Quick Build Mode ---

    var quickBuildMode = {
        active: false,
        sourceRoomId: null,
        sourceGx: 0, sourceGy: 0, sourceGz: 0
    };

    // --- Room Drag State ---

    var roomDrag = {
        active: false,
        anchorId: null,
        group: new Map(),       // roomId -> { startGx, startGy }
        deltaGx: 0, deltaGy: 0,
        pixelDx: 0, pixelDy: 0,
        anchorCanvasPx: 0, anchorCanvasPy: 0,
        droppable: false,
        brokenExits: [],
        allConstraints: []
    };

    // --- Camera Primitives ---

    var activeTab = '2d';
    var zoomScale = 1.0;
    var cameraX = 0, cameraY = 0, cameraZ = 0;
    var panOffsetX = 0, panOffsetY = 0;
    var easeStartX = 0, easeStartY = 0, easeStartZ = 0;
    var easeTargetX = 0, easeTargetY = 0, easeTargetZ = 0;
    var easeStartTime = null, easeRafId = null;
    var dragActive = false, dragStartPxX = 0, dragStartPxY = 0, dragStartPanX = 0, dragStartPanY = 0;

    var activeZ2d = 0;
    var spacingScale2d = (function() {
        var s = parseFloat(localStorage.getItem('mapper.spacing2d'));
        return (isFinite(s) && s >= 1.0 && s <= 3.0) ? s : 1.30;
    })();
    var showBounds = localStorage.getItem('mapper.showBounds') === 'true';
    var selectedZoneOnly = localStorage.getItem('mapper.selectedZoneOnly') === 'true';
    var tooltipHideTimer = null;

    // --- Mouse State Primitives ---

    var mousedownRoomId = null;
    var mousedownPxX = 0, mousedownPxY = 0;
    var mousedownShift = false;

    // --- Camera Object (Exposed) ---
    // Getter/setter proxy so external modules can read and write camera
    // values while the actual variables stay private to this closure.

    var camera = {
        get activeTab() { return activeTab; },
        set activeTab(v) { activeTab = v; },
        get zoomScale() { return zoomScale; },
        set zoomScale(v) { zoomScale = v; },
        get cameraX() { return cameraX; },
        set cameraX(v) { cameraX = v; },
        get cameraY() { return cameraY; },
        set cameraY(v) { cameraY = v; },
        get cameraZ() { return cameraZ; },
        set cameraZ(v) { cameraZ = v; },
        get panOffsetX() { return panOffsetX; },
        set panOffsetX(v) { panOffsetX = v; },
        get panOffsetY() { return panOffsetY; },
        set panOffsetY(v) { panOffsetY = v; },
        get easeStartX() { return easeStartX; },
        set easeStartX(v) { easeStartX = v; },
        get easeStartY() { return easeStartY; },
        set easeStartY(v) { easeStartY = v; },
        get easeStartZ() { return easeStartZ; },
        set easeStartZ(v) { easeStartZ = v; },
        get easeTargetX() { return easeTargetX; },
        set easeTargetX(v) { easeTargetX = v; },
        get easeTargetY() { return easeTargetY; },
        set easeTargetY(v) { easeTargetY = v; },
        get easeTargetZ() { return easeTargetZ; },
        set easeTargetZ(v) { easeTargetZ = v; },
        get easeStartTime() { return easeStartTime; },
        set easeStartTime(v) { easeStartTime = v; },
        get easeRafId() { return easeRafId; },
        set easeRafId(v) { easeRafId = v; },
        get dragActive() { return dragActive; },
        set dragActive(v) { dragActive = v; },
        get dragStartPxX() { return dragStartPxX; },
        set dragStartPxX(v) { dragStartPxX = v; },
        get dragStartPxY() { return dragStartPxY; },
        set dragStartPxY(v) { dragStartPxY = v; },
        get dragStartPanX() { return dragStartPanX; },
        set dragStartPanX(v) { dragStartPanX = v; },
        get dragStartPanY() { return dragStartPanY; },
        set dragStartPanY(v) { dragStartPanY = v; },
        get activeZ2d() { return activeZ2d; },
        set activeZ2d(v) { activeZ2d = v; },
        get spacingScale2d() { return spacingScale2d; },
        set spacingScale2d(v) { spacingScale2d = v; },
        get showBounds() { return showBounds; },
        set showBounds(v) { showBounds = v; },
        get selectedZoneOnly() { return selectedZoneOnly; },
        set selectedZoneOnly(v) { selectedZoneOnly = v; },
        get tooltipHideTimer() { return tooltipHideTimer; },
        set tooltipHideTimer(v) { tooltipHideTimer = v; }
    };

    // --- Mouse State Object (Exposed) ---

    var mouseState = {
        get mousedownRoomId() { return mousedownRoomId; },
        set mousedownRoomId(v) { mousedownRoomId = v; },
        get mousedownPxX() { return mousedownPxX; },
        set mousedownPxX(v) { mousedownPxX = v; },
        get mousedownPxY() { return mousedownPxY; },
        set mousedownPxY(v) { mousedownPxY = v; },
        get mousedownShift() { return mousedownShift; },
        set mousedownShift(v) { mousedownShift = v; }
    };

    // --- Dirty Tracking Helpers ---

    function isDirty() {
        return dirty.movedRooms.size > 0 || dirty.exitRemovals.length > 0 ||
               dirty.exitAdditions.length > 0 || dirty.deletedRooms.length > 0 ||
               dirty.createdRooms.size > 0;
    }

    var toastTimer = null;

    function showToastDirty() {
        if (!dom.toastContainerEl) return;
        var parts = [];
        if (dirty.movedRooms.size > 0)     parts.push(dirty.movedRooms.size + ' moved');
        if (dirty.createdRooms.size > 0)   parts.push(dirty.createdRooms.size + ' new');
        if (dirty.deletedRooms.length > 0) parts.push(dirty.deletedRooms.length + ' deleted');
        if (dirty.exitAdditions.length > 0) parts.push(dirty.exitAdditions.length + ' exits added');
        if (dirty.exitRemovals.length > 0)  parts.push(dirty.exitRemovals.length + ' exits removed');
        if (parts.length === 0) return;
        var el = document.createElement('div');
        el.className = 'mapper-toast';
        el.textContent = parts.join(', ');
        dom.toastContainerEl.appendChild(el);
        setTimeout(function() {
            if (el.parentNode) el.parentNode.removeChild(el);
        }, 2900);
    }

    function showToast(msg) {
        if (!dom.toastContainerEl) return;
        var el = document.createElement('div');
        el.className = 'mapper-toast';
        el.textContent = msg;
        dom.toastContainerEl.appendChild(el);
        setTimeout(function() {
            if (el.parentNode) el.parentNode.removeChild(el);
        }, 2900);
    }

    function updateSaveButtons() {
        var d = isDirty();
        if (dom.saveCtrlEl) dom.saveCtrlEl.classList.toggle('visible', d);
        if (d) {
            if (toastTimer) clearTimeout(toastTimer);
            toastTimer = setTimeout(function() {
                toastTimer = null;
                showToastDirty();
            }, 400);
        }
    }

    function logChange(cssClass, text) {
        if (!dom.clEntriesEl) return;
        var entry = document.createElement('div');
        entry.className = 'cl-entry ' + cssClass;
        entry.innerHTML = text;
        dom.clEntriesEl.appendChild(entry);
        // Update badge count
        if (dom.changelogBadgeEl) {
            var count = dom.clEntriesEl.childElementCount;
            dom.changelogBadgeEl.textContent = count;
            dom.changelogBadgeEl.classList.add('visible');
        }
        if (dom.changelogBtnEl) dom.changelogBtnEl.classList.add('has-entries');
        // Scroll if overlay is open
        if (dom.changelogEl && dom.changelogEl.classList.contains('visible')) {
            entry.scrollIntoView({ block: 'nearest' });
        }
    }

    function roomLabel(roomId) {
        var r = mapperData.rooms.get(roomId);
        return r ? escapeHtml(r.Title) + ' (#' + roomId + ')' : '#' + roomId;
    }

    function clearDirty() {
        dirty.movedRooms.clear();
        dirty.exitRemovals = [];
        dirty.exitAdditions = [];
        dirty.deletedRooms = [];
        dirty.createdRooms.clear();
        dirty.nextTempId = -1;
        if (dom.clEntriesEl) dom.clEntriesEl.innerHTML = '';
        if (dom.changelogEl) dom.changelogEl.classList.remove('visible');
        if (dom.changelogBadgeEl) { dom.changelogBadgeEl.textContent = ''; dom.changelogBadgeEl.classList.remove('visible'); }
        if (dom.changelogBtnEl) dom.changelogBtnEl.classList.remove('has-entries');
        updateSaveButtons();
    }

    // --- Local Mutations ---
    // These apply edits to the in-memory data immediately so the renderer
    // reflects changes before the server round-trip.  Each mutation also
    // records what changed in the `dirty` tracker so it can be persisted
    // later.

    /**
     * Replaces a temporary (negative) room ID with the real server-assigned ID.
     * Updates rooms map, roomsByCoord, dirty.createdRooms, selection, and any
     * exit references that point at the old temp ID.
     */
    function replaceRoomId(oldId, newId) {
        var room = mapperData.rooms.get(oldId);
        if (!room) return;

        // Update the room object and re-key in rooms map
        room.RoomId = newId;
        mapperData.rooms.delete(oldId);
        mapperData.rooms.set(newId, room);

        // Update coord index
        var coordKey = room.MapX + ',' + room.MapY + ',' + room.MapZ;
        if (mapperData.roomsByCoord.get(coordKey) === oldId) {
            mapperData.roomsByCoord.set(coordKey, newId);
        }

        // Update dirty.createdRooms
        if (dirty.createdRooms.has(oldId)) {
            var info = dirty.createdRooms.get(oldId);
            dirty.createdRooms.delete(oldId);
            dirty.createdRooms.set(newId, info);
        }

        // Update any exit references pointing at oldId
        mapperData.rooms.forEach(function(r) {
            if (!r.Exits) return;
            for (var dir in r.Exits) {
                if (r.Exits[dir].RoomId === oldId) r.Exits[dir].RoomId = newId;
            }
        });

        // Update selection
        if (selected.has(oldId)) { selected.delete(oldId); selected.add(newId); }
    }

    function moveRoomLocally(roomId, newGx, newGy) {
        var room = mapperData.rooms.get(roomId);
        if (!room) return;

        var wasTracked = dirty.movedRooms.has(roomId);
        if (!wasTracked) {
            dirty.movedRooms.set(roomId, { origGx: room.MapX, origGy: room.MapY, origGz: room.MapZ });
        }

        var oldGx = room.MapX, oldGy = room.MapY;
        mapperData.roomsByCoord.delete(room.MapX + ',' + room.MapY + ',' + room.MapZ);
        room.MapX = newGx;
        room.MapY = newGy;
        room.HasCoordinates = true;
        mapperData.roomsByCoord.set(newGx + ',' + newGy + ',' + room.MapZ, roomId);

        invalidateZoneBoundsCache();
        logChange('cl-move', '<span class="cl-action">MOVE</span> ' + roomLabel(roomId) + ' (' + oldGx + ',' + oldGy + ') &rarr; (' + newGx + ',' + newGy + ')');
        updateSaveButtons();
    }

    function breakExitLocally(roomId, dir) {
        var room = mapperData.rooms.get(roomId);
        if (room && room.Exits && room.Exits[dir]) {
            var targetId = room.Exits[dir].RoomId;
            delete room.Exits[dir];
            dirty.exitRemovals.push({ roomId: roomId, dir: dir });
            logChange('cl-exit-remove', '<span class="cl-action">REMOVE EXIT</span> "' + escapeHtml(dir) + '" from ' + roomLabel(roomId) + ' (was &rarr; #' + targetId + ')');
            updateSaveButtons();
        }
    }

    function deleteAllExitsLocally(roomId) {
        var room = mapperData.rooms.get(roomId);
        if (!room) return;
        // Remove all outgoing exits from this room
        if (room.Exits) {
            Object.keys(room.Exits).forEach(function(dir) { breakExitLocally(roomId, dir); });
        }
        // Remove all return exits from other rooms pointing back at this room
        mapperData.rooms.forEach(function(other, otherId) {
            if (otherId === roomId || !other.Exits) return;
            Object.keys(other.Exits).forEach(function(dir) {
                if (other.Exits[dir] && other.Exits[dir].RoomId === roomId) {
                    breakExitLocally(otherId, dir);
                }
            });
        });
    }

    function addExitLocally(sourceRoomId, dir, targetRoomId) {
        var room = mapperData.rooms.get(sourceRoomId);
        if (!room) return;
        if (!room.Exits) room.Exits = {};
        room.Exits[dir] = { RoomId: targetRoomId };
        dirty.exitAdditions.push({ roomId: sourceRoomId, dir: dir, targetRoomId: targetRoomId });
        logChange('cl-exit-add', '<span class="cl-action">ADD EXIT</span> "' + escapeHtml(dir) + '" on ' + roomLabel(sourceRoomId) + ' &rarr; ' + roomLabel(targetRoomId));
        updateSaveButtons();
    }

    function deleteRoomLocally(roomId) {
        var room = mapperData.rooms.get(roomId);
        if (!room) return;
        var label = roomLabel(roomId);

        // Sever every exit that points at this room before removing it
        mapperData.rooms.forEach(function(other, otherId) {
            if (!other.Exits) return;
            for (var dir in other.Exits) {
                if (other.Exits[dir].RoomId === roomId) breakExitLocally(otherId, dir);
            }
        });

        if (room.HasCoordinates) {
            mapperData.roomsByCoord.delete(room.MapX + ',' + room.MapY + ',' + room.MapZ);
        }

        // Temp rooms (negative IDs) are only local -- just drop them from createdRooms
        if (roomId > 0) {
            dirty.deletedRooms.push(roomId);
        } else {
            dirty.createdRooms.delete(roomId);
        }

        mapperData.rooms.delete(roomId);
        selectedRoomIds.delete(roomId);
        invalidateZoneBoundsCache();
        logChange('cl-delete', '<span class="cl-action">DELETE</span> ' + label);
        updateSaveButtons();
        _updateInfoPanel();
        _updateStats();
    }

    function createRoomLocally(gx, gy, gz, zone) {
        var tempId = dirty.nextTempId--;
        var resolvedZone = zone || mapperData.currentZone || '';
        var zoneInfo = mapperData.allZones.find(function(z) { return z.Name === resolvedZone; });
        var biome = zoneInfo ? zoneInfo.DefaultBiome : '';

        var tempRoom = {
            RoomId: tempId, Zone: resolvedZone, Title: 'New Room',
            MapX: gx, MapY: gy, MapZ: gz,
            HasCoordinates: true, MapSymbol: '', MapLegend: '', Biome: biome,
            _effectiveBiome: biome,
            Exits: {}
        };
        tempRoom._symbol = symbolForRoom(tempRoom);
        tempRoom._color = colorForSymbol(tempRoom._symbol, biome);
        tempRoom._bgColor = bgColorForBiome(biome, tempRoom._symbol) || null;
        tempRoom._symbolColor = contrastColor(tempRoom._bgColor || tempRoom._color);

        mapperData.rooms.set(tempId, tempRoom);
        mapperData.roomsByCoord.set(gx + ',' + gy + ',' + gz, tempId);
        invalidateZoneBoundsCache();

        if (!mapperData.zLevels.includes(gz)) {
            mapperData.zLevels.push(gz);
            mapperData.zLevels.sort(function(a, b) { return a - b; });
        }

        dirty.createdRooms.set(tempId, { gx: gx, gy: gy, gz: gz, zone: resolvedZone });
        logChange('cl-create', '<span class="cl-action">CREATE</span> New Room at (' + gx + ', ' + gy + ', ' + gz + ') in zone ' + escapeHtml(resolvedZone || '?'));
        updateSaveButtons();
        _updateStats();
        return tempId;
    }

    // --- Group Move ---
    // Relocates a set of rooms by a grid delta and automatically breaks any
    // directional exits whose spatial constraints are no longer satisfied
    // after the move.

    function applyGroupMove(group, deltaGx, deltaGy) {
        group.forEach(function(start, roomId) {
            var newGx = start.startGx + deltaGx;
            var newGy = start.startGy + deltaGy;
            moveRoomLocally(roomId, newGx, newGy);
        });
    }

    function moveRoomsZLocally(roomIds, deltaZ) {
        var rooms = [];
        for (var i = 0; i < roomIds.length; i++) {
            var room = mapperData.rooms.get(roomIds[i]);
            if (!room || !room.HasCoordinates) continue;
            rooms.push({ id: roomIds[i], room: room });
        }
        var idSet = new Set(roomIds);
        for (var j = 0; j < rooms.length; j++) {
            var r = rooms[j];
            var newZ = r.room.MapZ + deltaZ;
            var key = r.room.MapX + ',' + r.room.MapY + ',' + newZ;
            var occupant = mapperData.roomsByCoord.get(key);
            if (occupant !== undefined && !idSet.has(occupant)) {
                return false;
            }
        }
        for (var k = 0; k < rooms.length; k++) {
            var rm = rooms[k];
            var wasTracked = dirty.movedRooms.has(rm.id);
            if (!wasTracked) {
                dirty.movedRooms.set(rm.id, { origGx: rm.room.MapX, origGy: rm.room.MapY, origGz: rm.room.MapZ });
            }
            var oldZ = rm.room.MapZ;
            mapperData.roomsByCoord.delete(rm.room.MapX + ',' + rm.room.MapY + ',' + oldZ);
            rm.room.MapZ = oldZ + deltaZ;
            mapperData.roomsByCoord.set(rm.room.MapX + ',' + rm.room.MapY + ',' + rm.room.MapZ, rm.id);
            logChange('cl-move', '<span class="cl-action">MOVE Z</span> ' + roomLabel(rm.id) + ' Z ' + oldZ + ' &rarr; ' + rm.room.MapZ);
        }
        invalidateZoneBoundsCache();
        var targetZ = rooms[0].room.MapZ;
        if (!mapperData.zLevels.includes(targetZ)) {
            mapperData.zLevels.push(targetZ);
            mapperData.zLevels.sort(function(a, b) { return a - b; });
        }
        updateSaveButtons();
        return true;
    }

    // --- Selection ---

    function selectRoom(id) {
        selectedRoomIds.clear();
        if (id !== null) selectedRoomIds.add(id);
        _updateInfoPanel();
        _render();
    }

    function toggleRoomSelection(id) {
        if (selectedRoomIds.has(id)) selectedRoomIds.delete(id);
        else selectedRoomIds.add(id);
        _updateInfoPanel();
        _render();
    }

    // --- Data Loading ---

    async function loadTags() {
        var res = await AdminAPI.get('/admin/api/v1/tags');
        var list = res.ok && res.data ? res.data.data : null;
        if (Array.isArray(list)) {
            list.forEach(function(t) {
                tagDescriptions[t.tag] = t.module;
            });
        }
    }

    async function loadBiomes() {
        var res = await AdminAPI.get('/admin/api/v1/biomes');
        var biomeList = res.ok && res.data ? res.data.data : null;
        if (Array.isArray(biomeList)) {
            biomeList.forEach(function(b) {
                var key = b.BiomeId || b.Name;
                if (b.Name) biomeEnvMap[key] = b.Name;
                if (b.Symbol) BIOME_SYMBOLS[key] = b.Symbol;
                var color = b.Color || {};
                if (color.FGColor && color.FGColor > 0) {
                    var hex = ansi256ToHex(color.FGColor);
                    if (hex) BIOME_COLORS[key] = hex;
                }
                if (color.BGColor && color.BGColor > 0) {
                    var bgHex = ansi256ToHex(color.BGColor);
                    if (bgHex) BIOME_BG_COLORS[key] = bgHex;
                }
                // Build per-symbol override map: symbol -> { fg, bg }
                var overrides = b.SymbolOverrides || {};
                var ovMap = {};
                Object.keys(overrides).forEach(function(sym) {
                    var ov = overrides[sym] || {};
                    ovMap[sym] = {
                        fg: (ov.FGColor && ov.FGColor > 0) ? ansi256ToHex(ov.FGColor) : null,
                        bg: (ov.BGColor && ov.BGColor > 0) ? ansi256ToHex(ov.BGColor) : null
                    };
                });
                if (Object.keys(ovMap).length > 0) {
                    BIOME_SYMBOL_OVERRIDES[key] = ovMap;
                }
            });
        }
    }

    async function loadAllRooms() {
        _showLoading(true);
        var res = await AdminAPI.get('/admin/api/v1/mapper/rooms', true);
        _showLoading(false);
        var data = res.ok && res.data ? res.data.data : null;
        if (!data) return;

        mapperData.allZones = data.Zones || [];
        mapperData.zoneRootRooms.clear();

        // Build zone -> defaultBiome lookup for rooms that have no biome set
        var zoneDefaultBiome = {};
        mapperData.allZones.forEach(function(z) {
            mapperData.zoneRootRooms.set(z.Name, z.RoomId);
            if (z.DefaultBiome) zoneDefaultBiome[z.Name] = z.DefaultBiome;
        });

        mapperData.rawRooms = (data.Rooms || []).map(function(r) {
            // Resolve effective biome: room biome -> zone default biome -> ''
            var effectiveBiome = r.Biome || zoneDefaultBiome[r.Zone] || '';
            r._effectiveBiome = effectiveBiome;
            r._symbol = symbolForRoom(r);
            r._color = colorForSymbol(r._symbol, effectiveBiome);
            r._bgColor = bgColorForBiome(effectiveBiome, r._symbol) || null;
            r._symbolColor = contrastColor(r._bgColor || r._color);
            return r;
        });

        // Populate all rooms directly by coordinates — no zone filtering or offsets
        mapperData.rooms.clear();
        mapperData.roomsByCoord.clear();
        invalidateZoneBoundsCache();
        var zSet = new Set();
        mapperData.rawRooms.forEach(function(r) {
            mapperData.rooms.set(r.RoomId, r);
            if (r.HasCoordinates) {
                mapperData.roomsByCoord.set(r.MapX + ',' + r.MapY + ',' + r.MapZ, r.RoomId);
                zSet.add(r.MapZ);
            }
        });
        mapperData.zLevels = Array.from(zSet).sort(function(a, b) { return a - b; });

        _updateStats();
        _updateZButtons();

        if (mapperData.currentZone) {
            centerOnZone(mapperData.currentZone);
        } else {
            _render();
        }
    }

    // --- Zone Layout ---
    // Sets the current zone for default room-creation assignment.
    // All rooms are always visible; no filtering or coordinate offsets are applied.

    function applyZoneLayout(zoneName) {
        mapperData.currentZone = zoneName;
    }

    function centerOnZone(zoneName) {
        applyZoneLayout(zoneName);
        var rootId = mapperData.zoneRootRooms.get(zoneName);
        var root = rootId ? mapperData.rooms.get(rootId) : null;
        var targetZ = null;

        if (root && root.HasCoordinates) {
            cameraX = root.MapX;
            cameraY = root.MapY;
            cameraZ = root.MapZ;
            panOffsetX = 0;
            panOffsetY = 0;
            targetZ = root.MapZ;
        }

        if (mapperData.zLevels.length > 0) {
            if (mapperData.zLevels.includes(0)) {
                activeZ2d = 0;
            } else if (targetZ !== null && mapperData.zLevels.includes(targetZ)) {
                activeZ2d = targetZ;
            } else {
                activeZ2d = mapperData.zLevels.reduce(function(best, z) {
                    return Math.abs(z) < Math.abs(best) ? z : best;
                }, mapperData.zLevels[0]);
            }
        }

        _updateZButtons();
        _render();
    }

    // --- Public API ---
    // State objects, mutation helpers, data-loading functions, and callback
    // registration entry points.

    return {
        // State objects
        data: mapperData,
        selected: selectedRoomIds,
        dirty: dirty,
        camera: camera,
        roomDrag: roomDrag,
        exitDrawMode: exitDrawMode,
        quickBuildMode: quickBuildMode,
        get hoveredRoomId() { return hoveredRoomId; },
        set hoveredRoomId(v) { hoveredRoomId = v; },
        get hoveredGridCell() { return hoveredGridCell; },
        set hoveredGridCell(v) { hoveredGridCell = v; },
        selRect: selRect,
        mouseState: mouseState,

        // Mutation functions
        isDirty: isDirty,
        updateSaveButtons: updateSaveButtons,
        logChange: logChange,
        roomLabel: roomLabel,
        clearDirty: clearDirty,
        moveRoomLocally: moveRoomLocally,
        replaceRoomId: replaceRoomId,
        breakExitLocally: breakExitLocally,
        deleteAllExitsLocally: deleteAllExitsLocally,
        addExitLocally: addExitLocally,
        deleteRoomLocally: deleteRoomLocally,
        createRoomLocally: createRoomLocally,
        applyGroupMove: applyGroupMove,
        moveRoomsZLocally: moveRoomsZLocally,
        showToast: showToast,
        selectRoom: selectRoom,
        toggleRoomSelection: toggleRoomSelection,
        loadBiomes: loadBiomes,
        loadTags: loadTags,
        tagDescriptions: tagDescriptions,
        loadAllRooms: loadAllRooms,
        buildCrossZoneGraph: null,
        applyZoneLayout: applyZoneLayout,
        centerOnZone: centerOnZone,

        // Registration functions
        setDom: setDom,
        setRenderFn: setRenderFn,
        setUpdateFns: setUpdateFns
    };

})();
