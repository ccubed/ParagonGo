/* jshint esversion: 11, browser: true */
'use strict';

var MapperEvents = (function() {
    var listeners = {};

    function on(event, fn) {
        if (!listeners[event]) listeners[event] = [];
        listeners[event].push(fn);
    }

    function off(event, fn) {
        if (!listeners[event]) return;
        listeners[event] = listeners[event].filter(function(f) { return f !== fn; });
    }

    function emit(event, data) {
        if (!listeners[event]) return;
        listeners[event].forEach(function(fn) { fn(data); });
    }

    return { on: on, off: off, emit: emit };
})();
