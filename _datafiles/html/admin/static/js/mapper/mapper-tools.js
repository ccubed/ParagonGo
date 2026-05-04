/**
 * mapper-tools.js
 *
 * Registry and lifecycle manager for mapper editing tools. Each tool is a
 * plain object that follows this interface contract:
 *
 *   {
 *     name:          {string}   - Unique identifier (used as the registry key).
 *     onActivate:    {function} - Called with a context object when the tool
 *                                  becomes active. Optional.
 *     onDeactivate:  {function} - Called with no args when the tool is
 *                                  switched away from. Optional.
 *     onMouseDown:   {function} - Canvas mouse-down handler. Optional.
 *     onMouseMove:   {function} - Canvas mouse-move handler. Optional.
 *     onMouseUp:     {function} - Canvas mouse-up handler. Optional.
 *     ...any additional tool-specific methods
 *   }
 *
 * Only one tool may be active at a time. Activation emits "tool:activated"
 * on MapperEvents; deactivation emits "tool:deactivated".
 */
/* jshint esversion: 11, browser: true */
/* globals MapperEvents */
'use strict';

var MapperTools = (function() {
    var tools      = {};
    var activeTool = null;

    // --- Registration ---

    function register(tool) {
        tools[tool.name] = tool;
    }

    // --- Lifecycle ---

    function activate(name, context) {
        if (activeTool && activeTool.onDeactivate) activeTool.onDeactivate();
        activeTool = tools[name] || null;
        if (activeTool && activeTool.onActivate) activeTool.onActivate(context || {});
        MapperEvents.emit('tool:activated', { name: name });
    }

    function deactivate() {
        if (activeTool && activeTool.onDeactivate) activeTool.onDeactivate();
        activeTool = null;
        MapperEvents.emit('tool:deactivated');
    }

    // --- Accessors ---

    function getActive() { return activeTool; }
    function get(name)   { return tools[name] || null; }
    function all()       { return tools; }

    return {
        register:   register,
        activate:   activate,
        deactivate: deactivate,
        getActive:  getActive,
        get:        get,
        all:        all
    };
})();
