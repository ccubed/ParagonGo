/* jshint esversion: 11, browser: true */
'use strict';

// =========================================================================
// Constants
// =========================================================================

var ZOOM_STEP = 1.25;
var ZOOM_MIN  = 0.15;
var ZOOM_MAX  = 5.0;
var CENTER_EASE_DURATION = 0.2;

var ROOM_SIZE_2D       = 28;
var ROOM_GAP_2D        = 14;
var BASE_STEP_2D       = ROOM_SIZE_2D + ROOM_GAP_2D;
var CONNECTION_WIDTH_2D = 4;
var ROOM_BORDER_WIDTH_2D = 1.5;
var SYMBOL_FONT_SIZE_2D  = 14;
var MAP_BG_2D            = '#111';
var ROOM_BORDER_COLOR_2D = '#000000';

var TILE_HW_3D          = 20;
var TILE_HH_3D          = 10;
var TILE_DEPTH_3D       = 7;
var GRID_STEP_XY_3D     = 1.6;
var Z_STEP_3D           = 50;
var Z_SPACING_EXP_3D    = 1.5;
var CONNECTION_WIDTH_3D  = 2;
var MAP_BG_3D            = '#000';
var TILE_BORDER_COLOR_3D = '#000';
var TILE_BORDER_WIDTH_3D = 0.8;
var SIDE_DARKEN_3D       = 0.55;
var SYMBOL_FONT_SIZE_3D  = 11;
var SPACING_STEP_3D      = 1.25;
var SPACING_MIN_3D       = 0.6;
var SPACING_MAX_3D       = 4.0;
var ALPHA_INACTIVE_3D    = 0.0;
var ALPHA_CONNECTED_3D   = 0.30;
var CONN_COLOR_SAME_Z    = '#ffffff';
var CONN_COLOR_CROSS_Z   = '#3a6b8a';
var CROSS_Z_OFFSET_X     = 8;
var CROSS_Z_ARROW_SIZE   = 6;

var CONNECTION_COLOR          = '#7a4a1a';
var ABNORMAL_CONNECTION_COLOR = '#d4c050';
var SELECTED_ROOM_COLOR     = '#1a6abf';
var SELECTED_ROOM_TEXT_COLOR = '#ffffff';
var SYMBOL_TEXT_COLOR        = '#e0e0e0';
var DEFAULT_ROOM_COLOR       = '#3a3a4a';

var SYMBOL_COLORS = {
    '~':  '#2a53f7',
    '≈': '#0033cd',
    '♣': '#1a6b1a',
    '♨': '#4a6b20',
    '❄': '#b8d8f0',
    '⌬': '#5a4a38',
    '⩕': '#7a6a50',
    '▼': '#8a7a5a',
    '⌂': '#8a6a3a',
    '*':  '#d4aa55',
    "'":  '#6a8a30',
    '=':  '#a07840',
    '$':  '#2a7a2a',
    '%':  '#2a5a8a',
    '♜': '#4a4a4a',
    '+':  '#5fb7ff',
    '•': '#3a3a4a'
};

var ENVIRONMENT_SYMBOLS = {
    'Forest':    '♣',
    'Swamp':     '♨',
    'Snow':      '❄',
    'Cave':      '⌬',
    'Dungeon':   '⌬',
    'Mountains': '⩕',
    'Cliffs':    '▼',
    'House':     '⌂',
    'Desert':    '*',
    'Farmland':  "'",
    'Road':      '=',
    'Shore':     '~',
    'Water':     '≈'
};

var ENVIRONMENT_COLORS = {
    'Forest':    '#1a6b1a',
    'Swamp':     '#4a6b20',
    'Snow':      '#b8d8f0',
    'Cave':      '#5a4a38',
    'Dungeon':   '#5a4a38',
    'Mountains': '#7a6a50',
    'Cliffs':    '#8a7a5a',
    'House':     '#8a6a3a',
    'Desert':    '#d4aa55',
    'Farmland':  '#6a8a30',
    'Road':      '#a07840',
    'Shore':     '#2a53f7',
    'Water':     '#0033cd',
    'City':      '#5a5a6a',
    'Fort':      '#5a5a6a',
    'Land':      '#3a3a4a'
};

var BIOME_SYMBOLS = {};
var BIOME_COLORS = {};

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

var DIRECTIONAL_EXITS = {
    'north': true, 'south': true, 'east': true, 'west': true,
    'northeast': true, 'northwest': true, 'southeast': true, 'southwest': true,
    'up': true, 'down': true
};

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
