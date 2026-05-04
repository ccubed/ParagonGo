/* jshint esversion: 11, browser: true */
/* globals MapperEvents */
'use strict';

var MapperTools = (function() {
    var tools = {};
    var activeTool = null;

    function register(tool) {
        tools[tool.name] = tool;
    }

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

    function getActive() { return activeTool; }
    function get(name) { return tools[name] || null; }
    function all() { return tools; }

    return { register: register, activate: activate, deactivate: deactivate,
             getActive: getActive, get: get, all: all };
})();
