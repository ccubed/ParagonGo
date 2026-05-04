/* jshint esversion: 11, browser: true */
/* globals MapperEvents, AdminAPI, symbolForRoom, colorForSymbol, escapeHtml, isDirectionalExit, DIRECTION_DELTAS, isExitConstraintSatisfied, buildDragConstraints, breakExitLocally, BIOME_SYMBOLS, biomeEnvMap */
'use strict';

var MapperState = (function() {

    // =====================================================================
    // Callback registrations (break circular dependencies)
    // =====================================================================

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

    // =====================================================================
    // DOM references (set via setDom)
    // =====================================================================

    var dom = {
        saveCtrlEl: null,
        dirtyCountEl: null,
        changelogEl: null,
        clEntriesEl: null
    };

    function setDom(domRefs) {
        if (domRefs.saveCtrlEl) dom.saveCtrlEl = domRefs.saveCtrlEl;
        if (domRefs.dirtyCountEl) dom.dirtyCountEl = domRefs.dirtyCountEl;
        if (domRefs.changelogEl) dom.changelogEl = domRefs.changelogEl;
        if (domRefs.clEntriesEl) dom.clEntriesEl = domRefs.clEntriesEl;
    }

    // =====================================================================
    // Data layer
    // =====================================================================

    var mapperData = {
        allZones: [],
        currentZone: null,
        rawRooms: [],
        rooms: new Map(),
        roomsByCoord: new Map(),
        zLevels: [],
        zoneRootRooms: new Map(),
        zoneOffsets: new Map(),
        visibleZones: new Set()
    };

    // =====================================================================
    // Selection
    // =====================================================================

    var selectedRoomIds = new Set();
    var hoveredRoomId   = null;
    var hoveredGridCell  = null; // { gx, gy } when hovering an empty 2D cell
    var selRect = { active: false, startCx: 0, startCy: 0, endCx: 0, endCy: 0 };

    // =====================================================================
    // Deferred save tracking
    // =====================================================================

    var dirty = {
        movedRooms: new Map(),      // roomId -> { origGx, origGy, origGz }
        exitRemovals: [],           // [{ roomId, dir }]
        exitAdditions: [],          // [{ roomId, dir, targetRoomId }]
        deletedRooms: [],           // [roomId]
        createdRooms: new Map(),    // tempId -> { gx, gy, gz, zone }
        nextTempId: -1
    };

    // =====================================================================
    // Exit drawing mode
    // =====================================================================

    var exitDrawMode = {
        active: false,
        sourceRoomId: null,
        sourceGx: 0, sourceGy: 0, sourceGz: 0,
        hoveredTargetId: null,
        _mouseCx: 0, _mouseCy: 0
    };

    // =====================================================================
    // Quick build mode
    // =====================================================================

    var quickBuildMode = {
        active: false,
        sourceRoomId: null,
        sourceGx: 0, sourceGy: 0, sourceGz: 0
    };

    // =====================================================================
    // Room drag state
    // =====================================================================

    var roomDrag = {
        active: false,
        anchorId: null,
        group: new Map(),       // roomId -> { startGx, startGy }
        deltaGx: 0, deltaGy: 0,
        pixelDx: 0, pixelDy: 0,   // raw pixel offset from anchor's canvas position at drag start
        anchorCanvasPx: 0, anchorCanvasPy: 0,
        droppable: false,
        brokenExits: [],
        allConstraints: []
    };

    // =====================================================================
    // Camera state
    // =====================================================================

    var activeTab = localStorage.getItem('mapper.activeTab') === '3d' ? '3d' : '2d';
    var zoomScale = 1.0;
    var cameraX = 0, cameraY = 0, cameraZ = 0;
    var panOffsetX = 0, panOffsetY = 0;
    var easeStartX = 0, easeStartY = 0, easeStartZ = 0;
    var easeTargetX = 0, easeTargetY = 0, easeTargetZ = 0;
    var easeStartTime = null, easeRafId = null;
    var dragActive = false, dragStartPxX = 0, dragStartPxY = 0, dragStartPanX = 0, dragStartPanY = 0;

    var activeZ2d = 0;
    var activeZ3d = null;
    var spacingScale3d = (function() {
        var s = parseFloat(localStorage.getItem('mapper.spacing3d'));
        return (isFinite(s) && s >= 0.6 && s <= 4.0) ? s : 0.6;
    })();
    var tooltipHideTimer = null;

    // =====================================================================
    // Mouse state
    // =====================================================================

    var mousedownRoomId = null;
    var mousedownPxX = 0, mousedownPxY = 0;
    var mousedownShift = false;

    // =====================================================================
    // Camera object (exposed)
    // =====================================================================

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
        get activeZ3d() { return activeZ3d; },
        set activeZ3d(v) { activeZ3d = v; },
        get spacingScale3d() { return spacingScale3d; },
        set spacingScale3d(v) { spacingScale3d = v; },
        get tooltipHideTimer() { return tooltipHideTimer; },
        set tooltipHideTimer(v) { tooltipHideTimer = v; }
    };

    // =====================================================================
    // Mouse state object (exposed)
    // =====================================================================

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

    // =====================================================================
    // Dirty tracking mutations
    // =====================================================================

    function isDirty() {
        return dirty.movedRooms.size > 0 || dirty.exitRemovals.length > 0 ||
               dirty.exitAdditions.length > 0 || dirty.deletedRooms.length > 0 ||
               dirty.createdRooms.size > 0;
    }

    function updateSaveButtons() {
        var d = isDirty();
        if (dom.saveCtrlEl) dom.saveCtrlEl.classList.toggle('visible', d);
        if (d && dom.dirtyCountEl) {
            var parts = [];
            if (dirty.movedRooms.size > 0) parts.push(dirty.movedRooms.size + ' moved');
            if (dirty.createdRooms.size > 0) parts.push(dirty.createdRooms.size + ' new');
            if (dirty.deletedRooms.length > 0) parts.push(dirty.deletedRooms.length + ' deleted');
            if (dirty.exitAdditions.length > 0) parts.push(dirty.exitAdditions.length + ' exits added');
            if (dirty.exitRemovals.length > 0) parts.push(dirty.exitRemovals.length + ' exits removed');
            dom.dirtyCountEl.textContent = parts.join(', ');
        }
    }

    function logChange(cssClass, text) {
        if (!dom.clEntriesEl || !dom.changelogEl) return;
        var entry = document.createElement('div');
        entry.className = 'cl-entry ' + cssClass;
        entry.innerHTML = text;
        dom.clEntriesEl.appendChild(entry);
        dom.changelogEl.classList.add('visible');
        entry.scrollIntoView({ block: 'nearest' });
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
        updateSaveButtons();
    }

    // =====================================================================
    // Local mutations
    // =====================================================================

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
        mapperData.rooms.forEach(function(other, otherId) {
            if (!other.Exits) return;
            for (var dir in other.Exits) {
                if (other.Exits[dir].RoomId === roomId) breakExitLocally(otherId, dir);
            }
        });
        if (room.HasCoordinates) {
            mapperData.roomsByCoord.delete(room.MapX + ',' + room.MapY + ',' + room.MapZ);
        }
        if (roomId > 0) {
            dirty.deletedRooms.push(roomId);
        } else {
            dirty.createdRooms.delete(roomId);
        }
        mapperData.rooms.delete(roomId);
        selectedRoomIds.delete(roomId);
        logChange('cl-delete', '<span class="cl-action">DELETE</span> ' + label);
        updateSaveButtons();
        _updateInfoPanel();
        _updateStats();
    }

    function createRoomLocally(gx, gy, gz) {
        var tempId = dirty.nextTempId--;
        var zoneInfo = mapperData.allZones.find(function(z) { return z.Name === mapperData.currentZone; });
        var biome = zoneInfo ? zoneInfo.DefaultBiome : '';
        var tempRoom = {
            RoomId: tempId, Zone: mapperData.currentZone || '', Title: 'New Room',
            MapX: gx, MapY: gy, MapZ: gz,
            HasCoordinates: true, MapSymbol: '', MapLegend: '', Biome: biome, Exits: {}
        };
        tempRoom._symbol = symbolForRoom(tempRoom);
        tempRoom._color = colorForSymbol(tempRoom._symbol, tempRoom.Biome);
        mapperData.rooms.set(tempId, tempRoom);
        mapperData.roomsByCoord.set(gx + ',' + gy + ',' + gz, tempId);
        if (!mapperData.zLevels.includes(gz)) {
            mapperData.zLevels.push(gz);
            mapperData.zLevels.sort(function(a, b) { return a - b; });
        }
        dirty.createdRooms.set(tempId, { gx: gx, gy: gy, gz: gz, zone: mapperData.currentZone || '' });
        logChange('cl-create', '<span class="cl-action">CREATE</span> New Room at (' + gx + ', ' + gy + ', ' + gz + ') in zone ' + escapeHtml(mapperData.currentZone || '?'));
        updateSaveButtons();
        _updateStats();
        return tempId;
    }

    function applyGroupMove(group, deltaGx, deltaGy) {
        var brokenConstraints = [];
        group.forEach(function(start, roomId) {
            var newGx = start.startGx + deltaGx;
            var newGy = start.startGy + deltaGy;
            moveRoomLocally(roomId, newGx, newGy);
        });

        // Compute and break invalid exits for each room in the group
        group.forEach(function(start, roomId) {
            var room = mapperData.rooms.get(roomId);
            if (!room) return;
            var constraints = buildDragConstraints(roomId);
            constraints.forEach(function(c) {
                if (!isExitConstraintSatisfied(c, room.MapX, room.MapY)) {
                    brokenConstraints.push(c);
                    if (c.outgoing) {
                        var neighborId = mapperData.roomsByCoord.get(c.refX + ',' + c.refY + ',' + room.MapZ);
                        if (room.Exits) {
                            for (var dir in room.Exits) {
                                if (room.Exits[dir].RoomId === neighborId) {
                                    breakExitLocally(roomId, dir);
                                }
                            }
                        }
                    } else {
                        var neighborId2 = mapperData.roomsByCoord.get(c.refX + ',' + c.refY + ',' + room.MapZ);
                        if (neighborId2 !== undefined) {
                            var neighbor = mapperData.rooms.get(neighborId2);
                            if (neighbor && neighbor.Exits) {
                                for (var dir2 in neighbor.Exits) {
                                    if (neighbor.Exits[dir2].RoomId === roomId) {
                                        breakExitLocally(neighborId2, dir2);
                                    }
                                }
                            }
                        }
                    }
                }
            });
        });
    }

    // =====================================================================
    // Selection
    // =====================================================================

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

    // =====================================================================
    // Data loading
    // =====================================================================

    async function loadBiomes() {
        var res = await AdminAPI.get('/admin/api/v1/biomes');
        var biomeList = res.ok && res.data ? res.data.data : null;
        if (Array.isArray(biomeList)) {
            biomeList.forEach(function(b) {
                if (b.Name) biomeEnvMap[b.BiomeId || b.Name] = b.Name;
                if (b.Symbol) BIOME_SYMBOLS[b.BiomeId || b.Name] = b.Symbol;
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
        mapperData.allZones.forEach(function(z) {
            mapperData.zoneRootRooms.set(z.Name, z.RoomId);
        });

        mapperData.rawRooms = (data.Rooms || []).map(function(r) {
            r._symbol = symbolForRoom(r);
            r._color  = colorForSymbol(r._symbol, r.Biome);
            return r;
        });

        if (mapperData.currentZone) {
            applyZoneLayout(mapperData.currentZone);
        }
    }

    function buildCrossZoneGraph(rawRooms) {
        var roomById = new Map();
        rawRooms.forEach(function(r) { roomById.set(r.RoomId, r); });

        var edges = new Map();
        rawRooms.forEach(function(r) {
            if (!r.HasCoordinates || !r.Zone || !r.Exits) return;
            for (var dir in r.Exits) {
                if (!isDirectionalExit(dir)) continue;
                var delta = DIRECTION_DELTAS[dir];
                if (!delta || (delta[0] === 0 && delta[1] === 0 && delta[2] === 0)) continue;
                var dest = roomById.get(r.Exits[dir].RoomId);
                if (!dest || !dest.HasCoordinates || !dest.Zone) continue;
                if (dest.Zone === r.Zone) continue;
                var key = r.Zone + '>' + dest.Zone;
                if (!edges.has(key)) {
                    edges.set(key, {
                        fromZone: r.Zone, toZone: dest.Zone,
                        fromX: r.MapX, fromY: r.MapY,
                        toX: dest.MapX, toY: dest.MapY,
                        dx: delta[0], dy: delta[1]
                    });
                }
            }
        });
        return edges;
    }

    function applyZoneLayout(centerZone) {
        mapperData.currentZone = centerZone;
        mapperData.rooms.clear();
        mapperData.roomsByCoord.clear();
        mapperData.zoneOffsets.clear();
        mapperData.visibleZones.clear();

        var graph = buildCrossZoneGraph(mapperData.rawRooms);

        // BFS from centerZone through directional cross-zone exits
        var adj = new Map();
        graph.forEach(function(edge) {
            if (!adj.has(edge.fromZone)) adj.set(edge.fromZone, []);
            adj.get(edge.fromZone).push(edge);
        });

        mapperData.zoneOffsets.set(centerZone, { dx: 0, dy: 0 });
        mapperData.visibleZones.add(centerZone);
        var queue = [centerZone];

        while (queue.length > 0) {
            var zone = queue.shift();
            var myOff = mapperData.zoneOffsets.get(zone);
            var neighbors = adj.get(zone) || [];
            neighbors.forEach(function(edge) {
                if (mapperData.visibleZones.has(edge.toZone)) return;
                var dx = myOff.dx + (edge.fromX + edge.dx) - edge.toX;
                var dy = myOff.dy + (edge.fromY + edge.dy) - edge.toY;
                mapperData.zoneOffsets.set(edge.toZone, { dx: dx, dy: dy });
                mapperData.visibleZones.add(edge.toZone);
                queue.push(edge.toZone);
            });
        }

        // Populate rooms for visible zones only, applying offsets
        var zSet = new Set();
        mapperData.rawRooms.forEach(function(r) {
            if (!r.Zone || !mapperData.visibleZones.has(r.Zone)) return;
            var copy = Object.assign({}, r);
            copy.Exits = r.Exits;
            copy._symbol = r._symbol;
            copy._color = r._color;
            copy._serverX = r.MapX;
            copy._serverY = r.MapY;
            if (copy.HasCoordinates) {
                var off = mapperData.zoneOffsets.get(copy.Zone);
                if (off) { copy.MapX = r.MapX + off.dx; copy.MapY = r.MapY + off.dy; }
            }
            mapperData.rooms.set(copy.RoomId, copy);
            if (copy.HasCoordinates) {
                mapperData.roomsByCoord.set(copy.MapX + ',' + copy.MapY + ',' + copy.MapZ, copy.RoomId);
                zSet.add(copy.MapZ);
            }
        });
        mapperData.zLevels = Array.from(zSet).sort(function(a, b) { return a - b; });

        _updateStats();
        _updateZButtons();
        _render();
    }

    function centerOnZone(zoneName) {
        applyZoneLayout(zoneName);
        var rootId = mapperData.zoneRootRooms.get(zoneName);
        if (rootId) {
            var root = mapperData.rooms.get(rootId);
            if (root && root.HasCoordinates) {
                // setCameraTarget will be called by the init module via the camera object
                // For now, directly set camera values and trigger render
                cameraX = root.MapX;
                cameraY = root.MapY;
                cameraZ = root.MapZ;
                panOffsetX = 0;
                panOffsetY = 0;
                activeZ2d = root.MapZ;
                activeZ3d = root.MapZ;
                _updateZButtons();
                _render();
                return;
            }
        }
        if (mapperData.zLevels.length > 0) {
            activeZ2d = mapperData.zLevels[0];
            activeZ3d = mapperData.zLevels[0];
            _updateZButtons();
            _render();
        }
    }

    // =====================================================================
    // Public API
    // =====================================================================

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
        breakExitLocally: breakExitLocally,
        addExitLocally: addExitLocally,
        deleteRoomLocally: deleteRoomLocally,
        createRoomLocally: createRoomLocally,
        applyGroupMove: applyGroupMove,
        selectRoom: selectRoom,
        toggleRoomSelection: toggleRoomSelection,
        loadBiomes: loadBiomes,
        loadAllRooms: loadAllRooms,
        buildCrossZoneGraph: buildCrossZoneGraph,
        applyZoneLayout: applyZoneLayout,
        centerOnZone: centerOnZone,

        // Registration functions
        setDom: setDom,
        setRenderFn: setRenderFn,
        setUpdateFns: setUpdateFns
    };

})();
