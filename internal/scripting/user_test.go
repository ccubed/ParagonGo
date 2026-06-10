package scripting

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/require"
)

// withUserScript points the global user-script path at a temp file containing
// src (written with the given extension), resets VM state, and restores
// everything when the test finishes. An empty src writes no file.
func withUserScript(t *testing.T, ext string, src string) {
	t.Helper()

	// Dispatchers log via mudlog.Debug; ensure a logger exists under test.
	mudlog.SetupLogger(nil, "", "", false)

	prevFn := userScriptPathFn
	dir := t.TempDir()
	path := filepath.Join(dir, "user."+ext)

	if src != "" {
		if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
			t.Fatalf("write user script: %v", err)
		}
	}

	userScriptPathFn = func() string {
		// Mimic ResolveScriptPath: prefer .js, else .lua, else .js default.
		if _, err := os.Stat(filepath.Join(dir, "user.js")); err == nil {
			return filepath.Join(dir, "user.js")
		}
		if _, err := os.Stat(filepath.Join(dir, "user.lua")); err == nil {
			return filepath.Join(dir, "user.lua")
		}
		return filepath.Join(dir, "user.js")
	}
	ClearUserVM()

	t.Cleanup(func() {
		userScriptPathFn = prevFn
		ClearUserVM()
	})
}

func newTestUser(userId int) *users.UserRecord {
	return &users.UserRecord{
		UserId:   userId,
		Username: "tester",
		Character: &characters.Character{
			RoomId: 999999, // non-existent room; dispatch must tolerate nil room
		},
	}
}

func TestTryUserCommand_NoScript_NoOp(t *testing.T) {
	users.ResetActiveUsers()
	defer users.ResetActiveUsers()
	withUserScript(t, "js", "")

	users.SetTestUser(newTestUser(1))

	handled, err := TryUserCommand("look", "", 1)
	require.False(t, handled)
	require.ErrorIs(t, err, errNoScript)
}

func TestTryUserCommand_BoolReturn(t *testing.T) {
	cases := []struct {
		name string
		ext  string
		src  string
	}{
		{"js", "js", `function onCommand(cmd, rest, user, room) { return cmd === "halt"; }`},
		{"lua", "lua", `function onCommand(cmd, rest, user, room) return cmd == "halt" end`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			users.ResetActiveUsers()
			defer users.ResetActiveUsers()
			withUserScript(t, tc.ext, tc.src)
			users.SetTestUser(newTestUser(1))

			handled, err := TryUserCommand("halt", "", 1)
			require.NoError(t, err)
			require.True(t, handled)

			handled, err = TryUserCommand("look", "", 1)
			require.NoError(t, err)
			require.False(t, handled)
		})
	}
}

func TestTryUserDieEvent_AbortReturn(t *testing.T) {
	users.ResetActiveUsers()
	defer users.ResetActiveUsers()
	withUserScript(t, "js", `function onDie(user, room) { return true; }`)
	users.SetTestUser(newTestUser(1))

	handled, err := TryUserDieEvent(1)
	require.NoError(t, err)
	require.True(t, handled)
}

func TestTryUserScriptEvent_MissingHandler(t *testing.T) {
	users.ResetActiveUsers()
	defer users.ResetActiveUsers()
	withUserScript(t, "js", `function onDie(user, room) { return true; }`)
	users.SetTestUser(newTestUser(1))

	// onLogin is not defined in the script; should report event-not-found.
	handled, err := TryUserScriptEvent("onLogin", 1)
	require.False(t, handled)
	require.ErrorIs(t, err, ErrEventNotFound)
}

func TestTryUserLevelEvent_ReceivesDetails(t *testing.T) {
	users.ResetActiveUsers()
	defer users.ResetActiveUsers()
	withUserScript(t, "js", `function onLevel(user, room, d) { return d.newLevel === 5; }`)
	users.SetTestUser(newTestUser(1))

	handled, err := TryUserLevelEvent(1, map[string]any{"newLevel": 5})
	require.NoError(t, err)
	require.True(t, handled)
}

func TestInvalidateUserVM_ReloadsScript(t *testing.T) {
	users.ResetActiveUsers()
	defer users.ResetActiveUsers()
	withUserScript(t, "js", `function onCommand(cmd, rest, user, room) { return false; }`)
	users.SetTestUser(newTestUser(1))

	handled, err := TryUserCommand("x", "", 1)
	require.NoError(t, err)
	require.False(t, handled)

	// Overwrite the script on disk to now halt, then invalidate the cache.
	if err := os.WriteFile(userScriptPathFn(), []byte(`function onCommand(cmd, rest, user, room) { return true; }`), 0o644); err != nil {
		t.Fatalf("rewrite script: %v", err)
	}
	InvalidateUserVM()

	handled, err = TryUserCommand("x", "", 1)
	require.NoError(t, err)
	require.True(t, handled)
}
