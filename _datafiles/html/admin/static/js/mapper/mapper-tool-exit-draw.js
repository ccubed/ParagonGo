/**
 * mapper-tool-exit-draw.js -- Rubber-band line drawing mode for wiring exits.
 *
 * Activated from the context menu "Add Exit" item. A dashed line stretches
 * from the source room to the cursor; clicking on a different room opens a
 * modal to confirm the exit name and optional return exit. Directional exit
 * names are validated against the spatial relationship between the two rooms
 * so "north" cannot point south, etc.
 *
 * A separate "Add Exit (By Room Number)" path skips the visual rubber-band
 * and just opens the same modal after prompting for a target room ID.
 */
/* jshint esversion: 11, browser: true */
/* globals MapperTools, MapperCtxMenu, MapperState, MapperRender,
   ROOM_SIZE_2D, CONNECTION_WIDTH_2D,
   EXIT_DRAW_TARGET_HIGHLIGHT, EXIT_DRAW_LINE_COLOR,
   DIRECTION_DELTAS, DIRECTIONAL_EXITS, sign, escapeHtml */
'use strict';

(function() {

    // =====================================================================
    //  Spatial direction inference
    // =====================================================================

    var OPPOSITES = {
        north: 'south', south: 'north', east: 'west', west: 'east',
        northeast: 'southwest', southwest: 'northeast',
        northwest: 'southeast', southeast: 'northwest',
        up: 'down', down: 'up'
    };

    /**
     * Infer a cardinal/intercardinal exit name from the grid delta between
     * two rooms. Returns null when the relationship is ambiguous (neither
     * perfectly orthogonal nor perfectly diagonal on the XY plane, or when
     * either room lacks coordinates).
     *
     * Z-only connections (pure up/down) are also resolved.
     */
    function inferExitName(srcRoom, tgtRoom) {
        if (!srcRoom || !tgtRoom) return null;
        if (!srcRoom.HasCoordinates || !tgtRoom.HasCoordinates) return null;

        var dx = tgtRoom.MapX - srcRoom.MapX;
        var dy = tgtRoom.MapY - srcRoom.MapY;
        var dz = tgtRoom.MapZ - srcRoom.MapZ;

        // Pure Z movement
        if (dx === 0 && dy === 0 && dz !== 0) {
            return dz > 0 ? 'up' : 'down';
        }

        // Must be on the same Z level for XY directional names
        if (dz !== 0) return null;

        // Perfectly orthogonal
        if (dx === 0 && dy !== 0) return dy < 0 ? 'north' : 'south';
        if (dy === 0 && dx !== 0) return dx > 0 ? 'east' : 'west';

        // Perfectly diagonal (|dx| == |dy|)
        if (Math.abs(dx) === Math.abs(dy)) {
            if (dx > 0 && dy < 0) return 'northeast';
            if (dx < 0 && dy < 0) return 'northwest';
            if (dx > 0 && dy > 0) return 'southeast';
            if (dx < 0 && dy > 0) return 'southwest';
        }

        return null;
    }

    // =====================================================================
    //  Exit-name validation
    // =====================================================================

    /**
     * Reject directional exit names that contradict the spatial relationship.
     * Returns an error string, or null when the name is acceptable.
     */
    function validateExitName(dir, sourceRoom, targetRoom) {
        if (!dir) return null;
        if (DIRECTIONAL_EXITS[dir]) {
            var delta = DIRECTION_DELTAS[dir];
            if (delta && sourceRoom && targetRoom && sourceRoom.HasCoordinates && targetRoom.HasCoordinates) {
                var dx = targetRoom.MapX - sourceRoom.MapX;
                var dy = targetRoom.MapY - sourceRoom.MapY;
                var dz = targetRoom.MapZ - sourceRoom.MapZ;
                if (sign(dx) !== sign(delta[0]) || sign(dy) !== sign(delta[1]) || sign(dz) !== sign(delta[2])) {
                    return 'Exit "' + dir + '" does not match the spatial direction to the target room. Use a non-directional name instead.';
                }
            }
        }
        return null;
    }

    // =====================================================================
    //  Exit draw modal
    // =====================================================================

    /**
     * Open the exit-draw modal.
     *
     * @param {number} sourceRoomId
     * @param {number} targetRoomId
     * @param {function} onConfirm  Called with (dir, returnDir) on confirm.
     *                              returnDir may be empty string.
     * @param {function} onCancel   Called when the modal is dismissed without
     *                              confirming.
     */
    function openExitModal(sourceRoomId, targetRoomId, onConfirm, onCancel) {
        var backdrop   = document.getElementById('exit-draw-backdrop');
        var srcEl      = document.getElementById('exit-draw-src');
        var tgtEl      = document.getElementById('exit-draw-tgt');
        var dirEl      = document.getElementById('exit-draw-dir');
        var retEl      = document.getElementById('exit-draw-ret');
        var errorEl    = document.getElementById('exit-draw-error');
        var confirmBtn = document.getElementById('exit-draw-confirm');
        var cancelBtn  = document.getElementById('exit-draw-cancel');
        var closeBtn   = document.getElementById('exit-draw-close');
        if (!backdrop) { onCancel(); return; }

        var srcRoom = MapperState.data.rooms.get(sourceRoomId);
        var tgtRoom = MapperState.data.rooms.get(targetRoomId);

        srcEl.textContent = srcRoom ? srcRoom.Title + ' (#' + sourceRoomId + ')' : '#' + sourceRoomId;
        tgtEl.textContent = tgtRoom ? tgtRoom.Title + ' (#' + targetRoomId + ')' : '#' + targetRoomId;
        errorEl.textContent = '';
        confirmBtn.disabled = false;

        // Pre-populate exit names when the spatial relationship is unambiguous.
        var inferredDir = inferExitName(srcRoom, tgtRoom);
        var inferredRet = inferredDir ? (OPPOSITES[inferredDir] || '') : '';
        dirEl.value = inferredDir || '';
        retEl.value = inferredRet;

        backdrop.classList.add('visible');
        setTimeout(function() {
            if (inferredDir) {
                // Names are already filled; put focus on the confirm button so
                // the user can just press Enter to accept without tabbing.
                confirmBtn.focus();
            } else {
                dirEl.focus();
            }
        }, 40);

        function close() {
            backdrop.classList.remove('visible');
            backdrop._keyHandler && document.removeEventListener('keydown', backdrop._keyHandler);
        }

        function cancel() {
            close();
            onCancel();
        }

        function confirm() {
            var dir = dirEl.value.trim();
            var ret = retEl.value.trim();
            errorEl.textContent = '';

            if (!dir) {
                errorEl.textContent = 'Exit name is required.';
                dirEl.focus();
                return;
            }

            var err = validateExitName(dir, srcRoom, tgtRoom);
            if (err) { errorEl.textContent = err; dirEl.focus(); return; }

            if (ret) {
                var err2 = validateExitName(ret, tgtRoom, srcRoom);
                if (err2) { errorEl.textContent = 'Return exit: ' + err2; retEl.focus(); return; }
            }

            close();
            onConfirm(dir, ret);
        }

        // Replace listeners to avoid stacking
        function rewire(el, handler) {
            var clone = el.cloneNode(true);
            el.parentNode.replaceChild(clone, el);
            clone.addEventListener('click', handler);
            return clone;
        }
        rewire(document.getElementById('exit-draw-confirm'), confirm);
        rewire(document.getElementById('exit-draw-cancel'),  cancel);
        rewire(document.getElementById('exit-draw-close'),   cancel);

        backdrop._keyHandler && document.removeEventListener('keydown', backdrop._keyHandler);
        backdrop._keyHandler = function(e) {
            if (!backdrop.classList.contains('visible')) return;
            if (e.key === 'Escape') { e.stopPropagation(); cancel(); }
            if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') { e.preventDefault(); confirm(); }
            if (e.key === 'Enter' && document.activeElement !== retEl) { e.preventDefault(); confirm(); }
        };
        document.addEventListener('keydown', backdrop._keyHandler);

        backdrop.onclick = function(e) { if (e.target === backdrop) cancel(); };
    }

    // =====================================================================
    //  Finish / cancel
    // =====================================================================

    /** Complete the rubber-band draw: open modal to confirm exit names. */
    function finishExitDraw(targetRoomId) {
        var edm = MapperState.exitDrawMode;
        if (targetRoomId === edm.sourceRoomId) {
            MapperTools.activate('pan');
            return;
        }
        var sourceRoomId = edm.sourceRoomId;

        // Deactivate the rubber-band before opening the modal so the overlay
        // is not drawn while the modal is visible.
        MapperTools.activate('pan');

        openExitModal(sourceRoomId, targetRoomId,
            function onConfirm(dir, ret) {
                MapperState.addExitLocally(sourceRoomId, dir, targetRoomId);
                if (ret) {
                    MapperState.addExitLocally(targetRoomId, ret, sourceRoomId);
                }
                MapperRender.render();
            },
            function onCancel() {
                MapperRender.render();
            }
        );
    }

    /** Prompt-only path: ask for a room ID then open the modal. */
    function addExitByRoomNumber(sourceRoomId) {
        var targetIdStr = prompt('Target room number:');
        if (!targetIdStr || !targetIdStr.trim()) return;
        var targetRoomId = parseInt(targetIdStr.trim(), 10);
        if (isNaN(targetRoomId)) { alert('Invalid room number.'); return; }
        if (targetRoomId === sourceRoomId) { alert('Cannot connect a room to itself.'); return; }
        if (!MapperState.data.rooms.has(targetRoomId)) { alert('Room #' + targetRoomId + ' not found.'); return; }

        openExitModal(sourceRoomId, targetRoomId,
            function onConfirm(dir, ret) {
                MapperState.addExitLocally(sourceRoomId, dir, targetRoomId);
                if (ret) {
                    MapperState.addExitLocally(targetRoomId, ret, sourceRoomId);
                }
                MapperRender.render();
            },
            function onCancel() {}
        );
    }

    // =====================================================================
    //  Tool definition
    // =====================================================================

    var tool = {
        name: 'exit-draw',
        cursor: 'crosshair',

        // --- Mode activation ---

        onActivate: function(context) {
            var sourceRoomId = context.sourceRoomId;
            var room = MapperState.data.rooms.get(sourceRoomId);
            if (!room || !room.HasCoordinates) {
                MapperTools.activate('pan');
                return;
            }
            var edm = MapperState.exitDrawMode;
            edm.active = true;
            edm.sourceRoomId = sourceRoomId;
            edm.sourceGx = room.MapX;
            edm.sourceGy = room.MapY;
            edm.sourceGz = room.MapZ;
            edm.hoveredTargetId = null;
            MapperRender.render();
        },

        onDeactivate: function() {
            var edm = MapperState.exitDrawMode;
            edm.active = false;
            edm.sourceRoomId = null;
            edm.hoveredTargetId = null;
            MapperRender.render();
        },

        onMouseDown: function(e, cx, cy, roomId) {
            var edm = MapperState.exitDrawMode;
            if (!edm.active) return false;
            if (roomId !== null && roomId !== edm.sourceRoomId) {
                finishExitDraw(roomId);
            } else {
                MapperTools.activate('pan');
            }
            return true; // claim
        },

        onMouseMove: function(e, cx, cy, roomId) {
            var edm = MapperState.exitDrawMode;
            if (!edm.active) return;
            edm.hoveredTargetId = (roomId !== null && roomId !== edm.sourceRoomId) ? roomId : null;
            edm._mouseCx = cx;
            edm._mouseCy = cy;
            MapperRender.scheduleRender();
        },

        onMouseUp: function() {},

        onKeyDown: function(e) {
            if (e.key === 'Escape') {
                MapperTools.activate('pan');
            }
        },

        // -----------------------------------------------------------------
        //  2D overlay -- rubber-band line and target highlight
        // -----------------------------------------------------------------

        renderOverlay2d: function(ctx, rs) {
            var edm = MapperState.exitDrawMode;
            if (!edm.active) return;

            var srcP = rs.gridToCanvas2d(edm.sourceGx, edm.sourceGy);
            var endX, endY;

            if (edm.hoveredTargetId !== null) {
                var tgt = MapperState.data.rooms.get(edm.hoveredTargetId);
                if (tgt && tgt.HasCoordinates) {
                    var tgtP = rs.gridToCanvas2d(tgt.MapX, tgt.MapY);
                    endX = tgtP.px;
                    endY = tgtP.py;
                    var hlHalf = rs.scaledSize / 2 + 3 * rs.zoomScale;
                    ctx.strokeStyle = EXIT_DRAW_TARGET_HIGHLIGHT;
                    ctx.lineWidth = 2 * rs.zoomScale;
                    ctx.strokeRect(tgtP.px - hlHalf, tgtP.py - hlHalf, hlHalf * 2, hlHalf * 2);
                }
            } else {
                endX = edm._mouseCx;
                endY = edm._mouseCy;
            }

            if (endX !== undefined) {
                ctx.strokeStyle = EXIT_DRAW_LINE_COLOR;
                ctx.lineWidth = Math.max(2, CONNECTION_WIDTH_2D * rs.zoomScale);
                ctx.setLineDash([6 * rs.zoomScale, 4 * rs.zoomScale]);
                ctx.beginPath();
                ctx.moveTo(srcP.px, srcP.py);
                ctx.lineTo(endX, endY);
                ctx.stroke();
                ctx.setLineDash([]);
            }
        }
    };

    MapperTools.register(tool);

    // =====================================================================
    //  Context menu items
    // =====================================================================

    MapperCtxMenu.registerProvider(function(target) {
        if (target.type !== 'room') return null;
        return [
            {
                label: 'Add Exit',
                icon: '→',
                action: function() {
                    MapperTools.activate('exit-draw', { sourceRoomId: target.roomId });
                }
            },
            {
                label: 'Add Exit (By Room Number)',
                icon: '⌨',
                action: function() {
                    addExitByRoomNumber(target.roomId);
                }
            },
            {
                label: 'Delete All Exits',
                icon: '✕',
                style: 'color:#ff6b6b',
                action: function() {
                    MapperState.deleteAllExitsLocally(target.roomId);
                    MapperRender.render();
                }
            }
        ];
    });

})();
