/**
 * mapper-helpers.js
 *
 * Pure utility functions shared across every mapper module. Nothing here
 * touches the DOM or carries state -- each function takes explicit inputs
 * and returns a deterministic result (except escapeHtml, which uses a
 * throwaway DOM node for correctness).
 */
/* jshint esversion: 11, browser: true */
/* globals DIRECTION_DELTAS, DIRECTIONAL_EXITS, BIOME_SYMBOLS, BIOME_COLORS, BIOME_BG_COLORS, BIOME_SYMBOL_OVERRIDES, DEFAULT_ROOM_COLOR, ZONE_BOX_PADDING, MapperState */
'use strict';

// --- Exit & Direction Helpers ---

/**
 * Returns true when `dir` is a standard directional exit (cardinal,
 * intercardinal, up/down) rather than a named portal like "enter".
 * @param  {string}  dir - Exit name to test.
 * @return {boolean}
 */
function isDirectionalExit(dir) {
    return DIRECTIONAL_EXITS[dir] === true || DIRECTION_DELTAS[dir] !== undefined;
}

/**
 * Resolves the grid delta [dx, dy, dz] for a given exit, taking into account
 * any per-exit MapDirection override stored on the room.
 * @param  {string} dir  - Exit name.
 * @param  {Object} room - Room object that owns the exit.
 * @return {number[]|null} Three-element delta array, or null if unknown.
 */
function exitDelta(dir, room) {
    var ex = room.Exits && room.Exits[dir];
    if (ex && ex.MapDirection && DIRECTION_DELTAS[ex.MapDirection]) return DIRECTION_DELTAS[ex.MapDirection];
    if (DIRECTION_DELTAS[dir]) return DIRECTION_DELTAS[dir];
    return null;
}

/**
 * Converts an ANSI 256-color index to a CSS hex string.
 * Returns null for index 0 (treated as "no override") or out-of-range values.
 */
function ansi256ToHex(n) {
    if (!n || n <= 0 || n > 255) return null;
    var named = [
        '#000000','#800000','#008000','#808000','#000080','#800080','#008080','#c0c0c0',
        '#808080','#ff0000','#00ff00','#ffff00','#0000ff','#ff00ff','#00ffff','#ffffff'
    ];
    if (n < 16) return named[n];
    if (n < 232) {
        var idx = n - 16;
        var bv = idx % 6, gv = Math.floor(idx / 6) % 6, rv = Math.floor(idx / 36);
        var cv = function(i) { return i === 0 ? 0 : 55 + i * 40; };
        var toH = function(i) { return ('0' + cv(i).toString(16)).slice(-2); };
        return '#' + toH(rv) + toH(gv) + toH(bv);
    }
    var gray = 8 + (n - 232) * 10;
    var h = ('0' + gray.toString(16)).slice(-2);
    return '#' + h + h + h;
}

/**
 * Returns the bg hex color for a biome, or null if none is set.
 * Checks per-symbol overrides first when a symbol is provided.
 */
function bgColorForBiome(biome, sym) {
    if (!biome) return null;
    if (sym && BIOME_SYMBOL_OVERRIDES[biome]) {
        var ov = BIOME_SYMBOL_OVERRIDES[biome][sym];
        if (ov && ov.bg) return ov.bg;
    }
    return BIOME_BG_COLORS[biome] || null;
}

// --- Symbol & Color Resolution ---

/**
 * Picks the display symbol for a room, preferring the room's explicit
 * MapSymbol, then the biome's environment symbol, then a fallback dot.
 * @param  {Object} room - Room data object.
 * @return {string} Single-character symbol.
 */
function symbolForRoom(room) {
    if (room.MapSymbol) return room.MapSymbol;
    var biome = room._effectiveBiome || room.Biome || '';
    var env = biomeEnvName(biome);
    if (biome && BIOME_SYMBOLS[biome]) return BIOME_SYMBOLS[biome];
    return '•';
}

/**
 * Determines the fill color for a symbol, cascading through:
 * 1. Per-symbol biome override (SymbolOverrides[sym].fg)
 * 2. Biome default FG color (BIOME_COLORS)
 * 3. Default room color
 */
function colorForSymbol(sym, biome) {
    if (biome && sym && BIOME_SYMBOL_OVERRIDES[biome]) {
        var ov = BIOME_SYMBOL_OVERRIDES[biome][sym];
        if (ov && ov.fg) return ov.fg;
    }
    if (biome && BIOME_COLORS[biome]) return BIOME_COLORS[biome];
    return DEFAULT_ROOM_COLOR;
}

/** Returns '#ffffff' or '#000000' whichever contrasts better against the given hex fill color. */
function contrastColor(hex) {
    var r = parseInt(hex.slice(1, 3), 16);
    var g = parseInt(hex.slice(3, 5), 16);
    var b = parseInt(hex.slice(5, 7), 16);
    // Perceived luminance (sRGB)
    var lum = (0.299 * r + 0.587 * g + 0.114 * b) / 255;
    return lum > 0.45 ? '#000000' : '#ffffff';
}

// Cache mapping biome IDs to their canonical environment name
var biomeEnvMap = {};

// --- Zone Bounds Cache ---
// computeZonePaddedBounds is O(N*zones) and is called every render frame when
// showBounds is enabled. Cache the result and invalidate it only when rooms
// are added, removed, or moved -- camera pan/zoom does not affect grid bounds.

var _zoneBoundsCache = null;
var _zoneBoundsCacheZ = null;

function invalidateZoneBoundsCache() {
    _zoneBoundsCache = null;
    _zoneBoundsCacheZ = null;
}

/**
 * Maps a biome identifier to its canonical environment name. Returns the
 * biomeId itself when it already matches an environment, otherwise falls
 * back to the runtime-populated biomeEnvMap cache.
 * @param  {string} biomeId - Biome identifier from room data.
 * @return {string} Environment name, or empty string if unmapped.
 */
function biomeEnvName(biomeId) {
    if (!biomeId) return '';
    return biomeEnvMap[biomeId] || '';
}

// --- Zone Helpers ---

/**
 * Computes per-zone padded bounding boxes in grid space for all rooms on
 * the given Z level. Padding on each edge is capped to half the gap to the
 * nearest room from a different zone, so adjacent zone boxes never overlap.
 *
 * Returns an object keyed by zone name:
 *   { minX, maxX, minY, maxY }  (already padded, in grid units)
 */
function computeZonePaddedBounds(rooms, activeZ) {
    if (_zoneBoundsCache !== null && _zoneBoundsCacheZ === activeZ) {
        return _zoneBoundsCache;
    }
    _zoneBoundsCacheZ = activeZ;
    _zoneBoundsCache = _computeZonePaddedBounds(rooms, activeZ);
    return _zoneBoundsCache;
}

function _computeZonePaddedBounds(rooms, activeZ) {
    var maxPad = ZONE_BOX_PADDING;

    // Collect raw (unpadded) grid bounds per zone
    var raw = {};
    rooms.forEach(function(room) {
        if (!room.HasCoordinates || room.MapZ !== activeZ) return;
        var z = room.Zone || '';
        if (!raw[z]) {
            raw[z] = { minX: room.MapX, maxX: room.MapX, minY: room.MapY, maxY: room.MapY };
        } else {
            var b = raw[z];
            if (room.MapX < b.minX) b.minX = room.MapX;
            if (room.MapX > b.maxX) b.maxX = room.MapX;
            if (room.MapY < b.minY) b.minY = room.MapY;
            if (room.MapY > b.maxY) b.maxY = room.MapY;
        }
    });

    var zones = Object.keys(raw);
    if (zones.length === 0) return {};

    // For each zone edge, find the minimum gap to any room from a different zone.
    // gap[zone] = { left, right, top, bottom } in grid units
    var gap = {};
    zones.forEach(function(z) {
        gap[z] = { left: Infinity, right: Infinity, top: Infinity, bottom: Infinity };
    });

    rooms.forEach(function(room) {
        if (!room.HasCoordinates || room.MapZ !== activeZ) return;
        var rz = room.Zone || '';
        zones.forEach(function(z) {
            if (z === rz) return;
            var b = raw[z];
            // Distance from this foreign room to each edge of zone z's raw box
            var dRight  = room.MapX - b.maxX;  // positive = room is to the right of zone
            var dLeft   = b.minX - room.MapX;  // positive = room is to the left of zone
            var dBottom = room.MapY - b.maxY;  // positive = room is below zone
            var dTop    = b.minY - room.MapY;  // positive = room is above zone
            if (dRight  > 0 && dRight  < gap[z].right)  gap[z].right  = dRight;
            if (dLeft   > 0 && dLeft   < gap[z].left)   gap[z].left   = dLeft;
            if (dBottom > 0 && dBottom < gap[z].bottom) gap[z].bottom = dBottom;
            if (dTop    > 0 && dTop    < gap[z].top)    gap[z].top    = dTop;
        });
    });

    // Build padded bounds: pad each edge by min(maxPad, gap/2)
    var result = {};
    zones.forEach(function(z) {
        var b = raw[z], g = gap[z];
        var padLeft   = Math.min(maxPad, g.left   === Infinity ? maxPad : g.left   / 2);
        var padRight  = Math.min(maxPad, g.right  === Infinity ? maxPad : g.right  / 2);
        var padTop    = Math.min(maxPad, g.top    === Infinity ? maxPad : g.top    / 2);
        var padBottom = Math.min(maxPad, g.bottom === Infinity ? maxPad : g.bottom / 2);
        result[z] = {
            minX: b.minX - padLeft,
            maxX: b.maxX + padRight,
            minY: b.minY - padTop,
            maxY: b.maxY + padBottom
        };
    });
    return result;
}

/**
 * Returns the names of all visible zones whose padded grid bounding box
 * (on the given Z level) contains the point (gx, gy).
 */
function getZonesAtPoint(gx, gy, gz) {
    var bounds = computeZonePaddedBounds(MapperState.data.rooms, gz);
    var result = [];
    for (var zone in bounds) {
        var b = bounds[zone];
        if (gx >= b.minX && gx <= b.maxX && gy >= b.minY && gy <= b.maxY) {
            result.push(zone);
        }
    }
    return result;
}

/**
 * Returns the zone of the closest room to (gx, gy) on the given Z level.
 * Falls back to hintZone if no rooms exist.
 */
function closestZone(gx, gy, gz, hintZone) {
    var rooms = MapperState.data.rooms;
    var bestZone = hintZone || '';
    var bestDist = Infinity;
    rooms.forEach(function(room) {
        if (!room.HasCoordinates || room.MapZ !== gz) return;
        var dx = room.MapX - gx, dy = room.MapY - gy;
        var d = dx * dx + dy * dy;
        if (d < bestDist) { bestDist = d; bestZone = room.Zone || bestZone; }
    });
    return bestZone;
}

// --- Math Utilities ---

/** Hermite smoothstep for eased animations: 0 at t=0, 1 at t=1. */
function smoothstep(t) { return t * t * (3 - 2 * t); }

/** Returns -1, 0, or 1 matching the sign of v. */
function sign(v) { return v > 0 ? 1 : (v < 0 ? -1 : 0); }

/**
 * Darkens a hex color by a multiplicative factor (0-1).
 * @param  {string} hex    - 7-char hex color string (#rrggbb).
 * @param  {number} factor - Multiplier (e.g., 0.55 = 55% brightness).
 * @return {string} Darkened hex color.
 */
function darkenColor(hex, factor) {
    var r = Math.min(255, Math.round(parseInt(hex.slice(1,3), 16) * factor));
    var g = Math.min(255, Math.round(parseInt(hex.slice(3,5), 16) * factor));
    var b = Math.min(255, Math.round(parseInt(hex.slice(5,7), 16) * factor));
    return '#' + ('0'+r.toString(16)).slice(-2) + ('0'+g.toString(16)).slice(-2) + ('0'+b.toString(16)).slice(-2);
}

// --- Drag Constraint Helpers ---
// When the user drags a room to a new grid position, these functions enforce
// that connected exits still make geometric sense (e.g., a "north" exit must
// actually point northward on the grid).

/**
 * Builds a list of directional constraints for a room being dragged.
 * Each constraint captures the expected sign of the dx/dy delta between the
 * dragged room and an anchored neighbor so that exit directions stay valid.
 *
 * @param  {number}  roomId   - ID of the room being moved.
 * @param  {Set}     groupSet - Optional set of room IDs moving together
 *                               (connections within the group are ignored).
 * @return {Array}   Constraint objects used by isExitConstraintSatisfied().
 */
function buildDragConstraints(roomId, groupSet) {
    var room = MapperState.data.rooms.get(roomId);
    if (!room) return [];
    var constraints = [];

    // Outgoing exits: from this room to neighbors outside the group
    if (room.Exits) {
        for (var dir in room.Exits) {
            var ex = room.Exits[dir];
            if (groupSet && groupSet.has(ex.RoomId)) continue;
            var delta = exitDelta(dir, room);
            if (!delta || (delta[0] === 0 && delta[1] === 0)) continue;
            var dest = MapperState.data.rooms.get(ex.RoomId);
            if (!dest || !dest.HasCoordinates) continue;
            constraints.push({
                signDx: sign(delta[0]),
                signDy: sign(delta[1]),
                refX: dest.MapX,
                refY: dest.MapY,
                ownerGx: room.MapX,
                ownerGy: room.MapY,
                ownerId: roomId,
                outgoing: true
            });
        }
    }

    // Incoming exits: from neighbors outside the group pointing at this room
    MapperState.data.rooms.forEach(function(other, otherId) {
        if (otherId === roomId || !other.HasCoordinates || !other.Exits) return;
        if (groupSet && groupSet.has(otherId)) return;
        for (var dir in other.Exits) {
            var ex = other.Exits[dir];
            if (ex.RoomId !== roomId) continue;
            var delta = exitDelta(dir, other);
            if (!delta || (delta[0] === 0 && delta[1] === 0)) continue;
            constraints.push({
                signDx: sign(delta[0]),
                signDy: sign(delta[1]),
                refX: other.MapX,
                refY: other.MapY,
                ownerGx: room.MapX,
                ownerGy: room.MapY,
                ownerId: roomId,
                outgoing: false
            });
        }
    });

    return constraints;
}

/**
 * Tests whether a single exit constraint is satisfied at a candidate grid
 * position. Cardinal exits must stay axis-aligned; intercardinal exits must
 * stay on the exact diagonal (|dx| == |dy|).
 *
 * @param  {Object}  c  - Constraint from buildDragConstraints().
 * @param  {number}  gx - Candidate grid X.
 * @param  {number}  gy - Candidate grid Y.
 * @return {boolean}
 */
function isExitConstraintSatisfied(c, gx, gy) {
    var dx, dy;
    if (c.outgoing) {
        dx = c.refX - gx;
        dy = c.refY - gy;
    } else {
        dx = gx - c.refX;
        dy = gy - c.refY;
    }
    if (c.signDx !== 0 && sign(dx) !== c.signDx) return false;
    if (c.signDy !== 0 && sign(dy) !== c.signDy) return false;
    // Cardinal axis: the zero component must stay zero
    if (c.signDx === 0 && dx !== 0) return false;
    if (c.signDy === 0 && dy !== 0) return false;
    // Intercardinal: must stay on the exact diagonal
    if (c.signDx !== 0 && c.signDy !== 0 && Math.abs(dx) !== Math.abs(dy)) return false;
    return true;
}

/**
 * Returns all constraints that would be violated if the room were placed
 * at the given grid coordinates.
 *
 * @param  {number} gx          - Candidate grid X.
 * @param  {number} gy          - Candidate grid Y.
 * @param  {Array}  constraints - From buildDragConstraints().
 * @return {Array}  Subset of constraints that are broken.
 */
function computeBrokenExits(gx, gy, constraints) {
    var broken = [];
    for (var i = 0; i < constraints.length; i++) {
        if (!isExitConstraintSatisfied(constraints[i], gx, gy)) {
            broken.push(constraints[i]);
        }
    }
    return broken;
}

// --- String Utilities ---

/**
 * Escapes a string for safe insertion into HTML. Uses a throwaway DOM text
 * node so we get browser-native escaping without maintaining our own entity map.
 * @param  {string} s - Raw text.
 * @return {string} HTML-safe string.
 */
function escapeHtml(s) {
    var d = document.createElement('div');
    d.textContent = s;
    return d.innerHTML;
}
