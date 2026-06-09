-- Luacheck configuration for GoMud world scripts.
--
-- World scripts run on the embedded gopher-lua engine (Lua 5.1 compatible, with
-- the `goto` statement). They behave differently from standalone Lua programs:
--
--   * they call engine-injected globals (GetRoom, GetUser, UtilFindMatchIn, ...)
--     that Luacheck cannot see, and
--   * they define their event handlers as global functions (onCommand, onIdle, ...)
--     that the engine invokes by name.
--
-- Those patterns are expected, so the related "undefined/non-standard global" and
-- "unused" diagnostics are suppressed. Genuine problems such as syntax errors are
-- not warning codes and still fail the lint, which is the point of `make lua-lint`.

ignore = {
    "11.", -- accessing/setting/mutating undefined or non-standard globals
    "21.", -- unused or shadowing local variables
    "212", -- unused argument (event-handler signatures are fixed)
    "23.", -- unused/unset loop variables
    "311", -- value assigned to a local is never used
    "542", -- empty if branch
}
