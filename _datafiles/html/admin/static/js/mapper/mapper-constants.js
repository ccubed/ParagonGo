/**
 * mapper-constants.js
 *
 * Shared constants for the admin map editor. Every rendering mode (2D and 3D),
 * color palette, environment symbol mapping, and directional geometry table
 * lives here so the rest of the mapper modules can stay logic-only.
 */
/* jshint esversion: 11, browser: true */
'use strict';

// --- Zoom & Animation ---

var ZOOM_STEP           = 1.25;
var ZOOM_MIN            = 0.15;
var ZOOM_MAX            = 5.0;
var CENTER_EASE_DURATION = 0.2;   // seconds for the smooth-scroll to a room

// --- 2D Rendering Geometry ---

var ROOM_SIZE_2D         = 28;
var ROOM_GAP_2D          = 14;
var BASE_STEP_2D         = ROOM_SIZE_2D + ROOM_GAP_2D;  // grid pitch in px
var CONNECTION_WIDTH_2D  = 14;
var ROOM_BORDER_WIDTH_2D = 3;
var SYMBOL_FONT_SIZE_2D  = 14;
var MAP_BG_2D            = '#111';
var ROOM_BORDER_COLOR_2D = '#d9d9d9';

var ZONE_BOX_PADDING      = 0.6;                      // extra grid cells of padding around the zone bounding box
var ZONE_BOX_COLOR        = 'rgba(180,180,220,0.06)';  // fill for non-hovered zones
var ZONE_BOX_COLOR_HOV    = 'rgba(180,180,255,0.20)';  // fill for hovered zone
var ZONE_BOX_BORDER       = 'rgba(180,180,220,0.20)';  // border for non-hovered zones
var ZONE_BOX_BORDER_HOV   = 'rgba(180,180,255,0.7)';   // border for hovered zone

// --- Shared Color Palette ---

var CONNECTION_COLOR          = '#ffffff';
var ABNORMAL_CONNECTION_COLOR = '#d4c050';  // visually flags non-standard exits
var SELECTED_ROOM_COLOR       = '#1a6abf';
var SELECTED_ROOM_TEXT_COLOR  = '#ffffff';
var SYMBOL_TEXT_COLOR          = '#e0e0e0';
var DEFAULT_ROOM_COLOR        = '#3a3a4a';

// Room border indicators
var ROOM_BORDER_MOB_SPAWN     = 'rgb(255, 90, 90)';  // border color when room has a mob spawn
var ROOM_BORDER_SCRIPT_GLOW   = '#d4a843';  // script glow border color
var ROOM_ARROW_COLOR          = '#ff00ff';  // up/down Z-arrow color
var ROOM_ARROW_STROKE_COLOR   = 'rgba(0,0,0,0.75)';  // stroke/outline color for Z-arrows (set to '' to disable)
var ROOM_ARROW_STROKE_WIDTH   = 3;                   // stroke line width in px (before zoom scaling)

// Line badge colors
var BADGE_SECRET_COLOR        = '#d4a843';  // secret exit badge
var BADGE_LOCK_COLOR          = '#9ab0d4';  // locked exit badge

// Ghost / hover cell (pan tool)
var GHOST_CELL_BORDER         = 'rgba(255,255,255,0.35)';
var GHOST_CELL_FILL           = 'rgba(255,255,255,0.08)';
var GHOST_CELL_SYMBOL         = 'rgba(255,255,255,0.25)';

// Exit-draw tool
var EXIT_DRAW_TARGET_HIGHLIGHT = 'rgba(100,255,100,0.8)';
var EXIT_DRAW_LINE_COLOR       = 'rgba(100,200,255,0.8)';

// Room drag tool
var DRAG_ORIGIN_MARKER        = 'rgba(255,255,255,0.15)';
var DRAG_SNAP_BLOCKED         = 'rgba(255,80,80,0.5)';
var DRAG_SNAP_BROKEN          = 'rgba(255,180,60,0.5)';
var DRAG_SNAP_CLEAN           = 'rgba(100,200,255,0.5)';
var DRAG_CONSTRAINT_BROKEN    = 'rgba(255,60,60,0.7)';
var DRAG_CONSTRAINT_OK        = 'rgba(100,200,100,0.6)';
var DRAG_GHOST_BROKEN_FILL    = 'rgba(255,180,60,0.2)';

// Quick-build tool
var QB_COLOR                  = '95,183,122';  // RGB components for quick-build green
var QB_OCCUPIED_COLOR         = '255,255,255'; // RGB components for occupied slot

// Selection rectangle
var SELECT_RECT_FILL          = 'rgba(100,160,255,0.12)';
var SELECT_RECT_BORDER        = 'rgba(100,160,255,0.6)';

// Populated at runtime when custom biome data is loaded from the server
var BIOME_SYMBOLS          = {};
var BIOME_COLORS           = {};  // biomeId -> hex fg color
var BIOME_BG_COLORS        = {};  // biomeId -> hex bg color
var BIOME_SYMBOL_OVERRIDES = {};  // biomeId -> { symbol -> { fg: hex|null, bg: hex|null } }

// --- Direction & Grid Geometry ---
// Deltas are [dx, dy, dz]. Suffixes like -x2/-x3 represent multi-cell jumps;
// -gap/-gap2/-gap3 represent exits that skip intervening cells visually.

var DIRECTION_DELTAS = {
    'north': [0,-1,0], 'south': [0,1,0], 'west': [-1,0,0], 'east': [1,0,0],
    'northwest': [-1,-1,0], 'northeast': [1,-1,0], 'southwest': [-1,1,0], 'southeast': [1,1,0],
    'down': [0,0,-1], 'up': [0,0,1],
    'north-x2': [0,-2,0], 'south-x2': [0,2,0], 'west-x2': [-2,0,0], 'east-x2': [2,0,0],
    'northwest-x2': [-2,-2,0], 'northeast-x2': [2,-2,0], 'southwest-x2': [-2,2,0], 'southeast-x2': [2,2,0],
    'north-x3': [0,-3,0], 'south-x3': [0,3,0], 'west-x3': [-3,0,0], 'east-x3': [3,0,0],
    'northwest-x3': [-3,-3,0], 'northeast-x3': [3,-3,0], 'southwest-x3': [-3,3,0], 'southeast-x3': [3,3,0],
    'north-gap': [0,-1,0], 'south-gap': [0,1,0], 'west-gap': [-1,0,0], 'east-gap': [1,0,0],
    'northwest-gap': [-1,-1,0], 'northeast-gap': [1,-1,0], 'southwest-gap': [-1,1,0], 'southeast-gap': [1,1,0],
    'north-gap2': [0,-2,0], 'south-gap2': [0,2,0], 'west-gap2': [-2,0,0], 'east-gap2': [2,0,0],
    'northwest-gap2': [-2,-2,0], 'northeast-gap2': [2,-2,0], 'southwest-gap2': [-2,2,0], 'southeast-gap2': [2,2,0],
    'north-gap3': [0,-3,0], 'south-gap3': [0,3,0], 'west-gap3': [-3,0,0], 'east-gap3': [3,0,0],
    'northwest-gap3': [-3,-3,0], 'northeast-gap3': [3,-3,0], 'southwest-gap3': [-3,3,0], 'southeast-gap3': [3,3,0]
};

// Fast lookup: which exit names are "normal" directional exits (vs. named portals)
var DIRECTIONAL_EXITS = {
    'north': true, 'south': true, 'east': true, 'west': true,
    'northeast': true, 'northwest': true, 'southeast': true, 'southwest': true,
    'up': true, 'down': true
};

// --- Quick-Build Geometry ---
// Base 8 cardinal/intercardinal directions with their return counterparts.
// CARDINAL_OFFSETS expands each base direction to x1, x2, x3 distances.

var CARDINAL_BASE = [
    { dx:  0, dy: -1, dir: 'north',     ret: 'south'     },
    { dx:  0, dy:  1, dir: 'south',     ret: 'north'     },
    { dx:  1, dy:  0, dir: 'east',      ret: 'west'      },
    { dx: -1, dy:  0, dir: 'west',      ret: 'east'      },
    { dx:  1, dy: -1, dir: 'northeast', ret: 'southwest'  },
    { dx: -1, dy: -1, dir: 'northwest', ret: 'southeast'  },
    { dx:  1, dy:  1, dir: 'southeast', ret: 'northwest'  },
    { dx: -1, dy:  1, dir: 'southwest', ret: 'northeast'  }
];

var CARDINAL_OFFSETS = [];
CARDINAL_BASE.forEach(function(b) {
    for (var m = 1; m <= 3; m++) {
        var suffix = m === 1 ? '' : '-x' + m;
        CARDINAL_OFFSETS.push({
            dx: b.dx * m, dy: b.dy * m, dist: m,
            dir: b.dir + suffix, ret: b.ret + suffix,
            label: b.dir
        });
    }
});
