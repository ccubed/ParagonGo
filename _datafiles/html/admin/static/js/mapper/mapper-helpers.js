/* jshint esversion: 11, browser: true */
/* globals DIRECTION_DELTAS, DIRECTIONAL_EXITS, ENVIRONMENT_SYMBOLS, ENVIRONMENT_COLORS, SYMBOL_COLORS, BIOME_SYMBOLS, BIOME_COLORS, DEFAULT_ROOM_COLOR, mapperData */
'use strict';

// =========================================================================
// Helpers
// =========================================================================

function isDirectionalExit(dir) {
    return DIRECTIONAL_EXITS[dir] === true || DIRECTION_DELTAS[dir] !== undefined;
}

function symbolForRoom(room) {
    if (room.MapSymbol) return room.MapSymbol;
    var env = biomeEnvName(room.Biome);
    if (env && ENVIRONMENT_SYMBOLS[env]) return ENVIRONMENT_SYMBOLS[env];
    if (room.Biome && BIOME_SYMBOLS[room.Biome]) return BIOME_SYMBOLS[room.Biome];
    return '•';
}

function colorForSymbol(sym, biome) {
    if (sym && SYMBOL_COLORS[sym]) return SYMBOL_COLORS[sym];
    var env = biomeEnvName(biome);
    if (env && ENVIRONMENT_COLORS[env]) return ENVIRONMENT_COLORS[env];
    if (biome && BIOME_COLORS[biome]) return BIOME_COLORS[biome];
    return DEFAULT_ROOM_COLOR;
}

var biomeEnvMap = {};

function biomeEnvName(biomeId) {
    if (!biomeId) return '';
    if (ENVIRONMENT_COLORS[biomeId]) return biomeId;
    return biomeEnvMap[biomeId] || '';
}

function smoothstep(t) { return t * t * (3 - 2 * t); }

function darkenColor(hex, factor) {
    var r = Math.min(255, Math.round(parseInt(hex.slice(1,3), 16) * factor));
    var g = Math.min(255, Math.round(parseInt(hex.slice(3,5), 16) * factor));
    var b = Math.min(255, Math.round(parseInt(hex.slice(5,7), 16) * factor));
    return '#' + ('0'+r.toString(16)).slice(-2) + ('0'+g.toString(16)).slice(-2) + ('0'+b.toString(16)).slice(-2);
}

function exitDelta(dir, room) {
    var ex = room.Exits && room.Exits[dir];
    if (ex && ex.MapDirection && DIRECTION_DELTAS[ex.MapDirection]) return DIRECTION_DELTAS[ex.MapDirection];
    if (DIRECTION_DELTAS[dir]) return DIRECTION_DELTAS[dir];
    return null;
}

function sign(v) { return v > 0 ? 1 : (v < 0 ? -1 : 0); }

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

function isExitConstraintSatisfied(c, gx, gy) {
    var dx, dy;
    if (c.outgoing) {
        dx = c.refX - gx;
        dy = c.refY - gy;
    } else {
        dx = gx - c.refX;
        dy = gy - c.refY;
    }
    // Sign must match
    if (c.signDx !== 0 && sign(dx) !== c.signDx) return false;
    if (c.signDy !== 0 && sign(dy) !== c.signDy) return false;
    // Cardinal: axis-aligned (zero component must stay zero)
    if (c.signDx === 0 && dx !== 0) return false;
    if (c.signDy === 0 && dy !== 0) return false;
    // Intercardinal: must stay on the exact diagonal (|dx| == |dy|)
    if (c.signDx !== 0 && c.signDy !== 0 && Math.abs(dx) !== Math.abs(dy)) return false;
    return true;
}

function computeBrokenExits(gx, gy, constraints) {
    var broken = [];
    for (var i = 0; i < constraints.length; i++) {
        if (!isExitConstraintSatisfied(constraints[i], gx, gy)) {
            broken.push(constraints[i]);
        }
    }
    return broken;
}

function escapeHtml(s) {
    var d = document.createElement('div');
    d.textContent = s;
    return d.innerHTML;
}
