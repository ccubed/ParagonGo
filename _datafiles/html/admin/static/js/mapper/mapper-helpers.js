/**
 * mapper-helpers.js
 *
 * Pure utility functions shared across every mapper module. Nothing here
 * touches the DOM or carries state -- each function takes explicit inputs
 * and returns a deterministic result (except escapeHtml, which uses a
 * throwaway DOM node for correctness).
 */
/* jshint esversion: 11, browser: true */
/* globals DIRECTION_DELTAS, DIRECTIONAL_EXITS, ENVIRONMENT_SYMBOLS, ENVIRONMENT_COLORS, SYMBOL_COLORS, BIOME_SYMBOLS, BIOME_COLORS, DEFAULT_ROOM_COLOR, mapperData */
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

// --- Symbol & Color Resolution ---

/**
 * Picks the display symbol for a room, preferring the room's explicit
 * MapSymbol, then the biome's environment symbol, then a fallback dot.
 * @param  {Object} room - Room data object.
 * @return {string} Single-character symbol.
 */
function symbolForRoom(room) {
    if (room.MapSymbol) return room.MapSymbol;
    var env = biomeEnvName(room.Biome);
    if (env && ENVIRONMENT_SYMBOLS[env]) return ENVIRONMENT_SYMBOLS[env];
    if (room.Biome && BIOME_SYMBOLS[room.Biome]) return BIOME_SYMBOLS[room.Biome];
    return '•';
}

/**
 * Determines the fill color for a symbol, cascading through symbol-specific,
 * environment-level, biome-level, and finally the default room color.
 * @param  {string} sym   - Map symbol character.
 * @param  {string} biome - Biome identifier.
 * @return {string} CSS hex color.
 */
function colorForSymbol(sym, biome) {
    if (sym && SYMBOL_COLORS[sym]) return SYMBOL_COLORS[sym];
    var env = biomeEnvName(biome);
    if (env && ENVIRONMENT_COLORS[env]) return ENVIRONMENT_COLORS[env];
    if (biome && BIOME_COLORS[biome]) return BIOME_COLORS[biome];
    return DEFAULT_ROOM_COLOR;
}

// Cache mapping biome IDs to their canonical environment name
var biomeEnvMap = {};

/**
 * Maps a biome identifier to its canonical environment name. Returns the
 * biomeId itself when it already matches an environment, otherwise falls
 * back to the runtime-populated biomeEnvMap cache.
 * @param  {string} biomeId - Biome identifier from room data.
 * @return {string} Environment name, or empty string if unmapped.
 */
function biomeEnvName(biomeId) {
    if (!biomeId) return '';
    if (ENVIRONMENT_COLORS[biomeId]) return biomeId;
    return biomeEnvMap[biomeId] || '';
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
    var room = mapperData.rooms.get(roomId);
    if (!room) return [];
    var constraints = [];

    // Outgoing exits: from this room to neighbors outside the group
    if (room.Exits) {
        for (var dir in room.Exits) {
            var ex = room.Exits[dir];
            if (groupSet && groupSet.has(ex.RoomId)) continue;
            var delta = exitDelta(dir, room);
            if (!delta || (delta[0] === 0 && delta[1] === 0)) continue;
            var dest = mapperData.rooms.get(ex.RoomId);
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
    mapperData.rooms.forEach(function(other, otherId) {
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
