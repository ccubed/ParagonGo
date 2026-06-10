package scripting

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// There is a single, global user script that applies to every player. It is
// cached as one shared VM (unlike rooms/mobs which cache one VM per owner).
var (
	userVM         scriptVM
	userVMLoaded   bool
	scriptUserTimeout = 50 * time.Millisecond

	// userScriptPathFn resolves the path to the global user script. It is a
	// package variable so tests can point it at a temporary file.
	userScriptPathFn = defaultUserScriptPath
)

// defaultUserScriptPath returns the resolved path to the global user script,
// preferring .js and falling back to .lua (default .js when neither exists).
func defaultUserScriptPath() string {
	return util.ResolveScriptPath(configs.GetFilePathsConfig().DataFiles.String() + `/scripts/user.yaml`)
}

// UserScriptPath returns the resolved on-disk path of the global user script.
func UserScriptPath() string {
	return userScriptPathFn()
}

// GetUserScript returns the source of the global user script, or an empty
// string when no script file exists.
func GetUserScript() string {
	scriptPath := userScriptPathFn()
	if _, err := os.Stat(scriptPath); err == nil {
		if bytes, err := util.ReadFile(scriptPath); err == nil {
			return string(bytes)
		}
	}
	return ``
}

// UserScriptLang returns the editor language identifier ("js" or "lua") for the
// global user script based on its resolved path.
func UserScriptLang() string {
	if LangFromPath(userScriptPathFn()) == LangLua {
		return `lua`
	}
	return `js`
}

// SaveUserScript writes (or overwrites) the global user script file. When
// content is empty the existing script file is deleted instead. The cached VM
// is invalidated so the next dispatch reloads from disk.
func SaveUserScript(content string, lang string) error {
	scriptPath := userScriptPathFn()

	if content == `` {
		if err := os.Remove(scriptPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing user script: %w", err)
		}
		InvalidateUserVM()
		return nil
	}

	scriptPath = util.ApplyScriptLang(scriptPath, lang)
	if dir := filepath.Dir(scriptPath); dir != `` {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating user script directory: %w", err)
		}
	}
	if err := util.WriteFile(scriptPath, []byte(content), 0644); err != nil {
		return err
	}
	InvalidateUserVM()
	return nil
}

// ClearUserVM evicts the cached global user-script VM.
func ClearUserVM() {
	userVM = nil
	userVMLoaded = false
}

// InvalidateUserVM forces the next dispatch to reload the user script from
// disk. Call this after saving the script via the admin API.
func InvalidateUserVM() {
	ClearUserVM()
}

func getUserVM() (scriptVM, error) {

	if userVMLoaded {
		if userVM == nil {
			return nil, errNoScript
		}
		if scriptHotReload {
			scriptPath := userScriptPathFn()
			if info, err := os.Stat(scriptPath); err == nil {
				if info.ModTime().After(userVM.LoadedAt()) {
					ClearUserVM()
					// fall through to reload
				} else {
					return userVM, nil
				}
			} else {
				return userVM, nil
			}
		} else {
			return userVM, nil
		}
	}

	scriptPath := userScriptPathFn()

	script := ``
	if _, err := os.Stat(scriptPath); err == nil {
		if bytes, err := util.ReadFile(scriptPath); err == nil {
			script = string(bytes)
		}
	}

	if len(script) == 0 {
		userVM = nil
		userVMLoaded = true
		return nil, errNoScript
	}

	src := sourceFromPath(scriptPath, script)
	vmw, err := loadVM(`user`, src, func(vm scriptVM) error {
		if fn, ok := vm.GetFunction(`onLoad`); ok {
			_, err := vm.Call(scriptLoadTimeout, fn)
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	userVM = vmw
	userVMLoaded = true
	return vmw, nil
}

// TryUserCommand fires the global user script's onCommand handler before room
// and mob scripts. Returning true halts all further command processing.
func TryUserCommand(cmd string, rest string, userId int) (bool, error) {

	vmw, err := getUserVM()
	if err != nil {
		return false, err
	}

	sUser := GetUser(userId)
	if sUser == nil {
		return false, fmt.Errorf("user not found")
	}

	timestart := time.Now()
	defer func() {
		mudlog.Debug("TryUserCommand()", "cmd", cmd, "userId", userId, "time", time.Since(timestart))
	}()

	onCommandFunc, ok := vmw.GetFunction(`onCommand`)
	if !ok {
		return false, ErrEventNotFound
	}

	userTextWrap.Set(`script-text`, ``, ``)
	roomTextWrap.Set(`script-text`, ``, ``)

	sRoom := GetRoom(sUser.GetRoomId())

	res, err := runCallable(vmw, scriptUserTimeout, onCommandFunc,
		vmw.ToValue(cmd),
		vmw.ToValue(rest),
		vmw.ToValue(sUser),
		vmw.ToValue(sRoom),
	)

	userTextWrap.Reset()
	roomTextWrap.Reset()

	if err != nil {
		return false, fmt.Errorf("onCommand(): %w", err)
	}

	if boolVal, ok := res.Export().(bool); ok {
		return boolVal, nil
	}

	return false, nil
}

// TryUserScriptEvent fires a simple (user, room) global user-script handler,
// used for onDying, onLogin, and onLogout. The room may be nil-safe (e.g. at
// logout) and is passed through unchanged.
func TryUserScriptEvent(eventName string, userId int) (bool, error) {

	vmw, err := getUserVM()
	if err != nil {
		return false, err
	}

	sUser := GetUser(userId)
	if sUser == nil {
		return false, fmt.Errorf("user not found")
	}

	timestart := time.Now()
	defer func() {
		mudlog.Debug("TryUserScriptEvent()", "eventName", eventName, "userId", userId, "time", time.Since(timestart))
	}()

	eventFunc, ok := vmw.GetFunction(eventName)
	if !ok {
		return false, ErrEventNotFound
	}

	userTextWrap.Set(`script-text`, ``, ``)
	roomTextWrap.Set(`script-text`, ``, ``)

	sRoom := GetRoom(sUser.GetRoomId())

	res, err := runCallable(vmw, scriptUserTimeout, eventFunc,
		vmw.ToValue(sUser),
		vmw.ToValue(sRoom),
	)

	userTextWrap.Reset()
	roomTextWrap.Reset()

	if err != nil {
		return false, fmt.Errorf("%s(): %w", eventName, err)
	}

	if boolVal, ok := res.Export().(bool); ok {
		return boolVal, nil
	}

	return false, nil
}

// TryUserDieEvent fires the global user script's onDie handler when a player
// fully dies. Returning true aborts the default death handling; the script is
// responsible for whatever happens instead.
func TryUserDieEvent(userId int) (bool, error) {
	return TryUserScriptEvent(`onDie`, userId)
}

// TryUserLevelEvent fires the global user script's onLevel handler when a
// player levels up. The return value is informational and ignored by callers.
func TryUserLevelEvent(userId int, details map[string]any) (bool, error) {

	vmw, err := getUserVM()
	if err != nil {
		return false, err
	}

	sUser := GetUser(userId)
	if sUser == nil {
		return false, fmt.Errorf("user not found")
	}

	timestart := time.Now()
	defer func() {
		mudlog.Debug("TryUserLevelEvent()", "userId", userId, "time", time.Since(timestart))
	}()

	eventFunc, ok := vmw.GetFunction(`onLevel`)
	if !ok {
		return false, ErrEventNotFound
	}

	if details == nil {
		details = make(map[string]any)
	}

	userTextWrap.Set(`script-text`, ``, ``)
	roomTextWrap.Set(`script-text`, ``, ``)

	sRoom := GetRoom(sUser.GetRoomId())

	res, err := runCallable(vmw, scriptUserTimeout, eventFunc,
		vmw.ToValue(sUser),
		vmw.ToValue(sRoom),
		vmw.ToValue(details),
	)

	userTextWrap.Reset()
	roomTextWrap.Reset()

	if err != nil {
		return false, fmt.Errorf("onLevel(): %w", err)
	}

	if boolVal, ok := res.Export().(bool); ok {
		return boolVal, nil
	}

	return false, nil
}
