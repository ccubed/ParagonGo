package scripting

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

var (
	roomVMCache       = make(map[int]scriptVM)
	scriptLoadTimeout = 1000 * time.Millisecond
	scriptRoomTimeout = 50 * time.Millisecond
)

func ClearRoomVMs() {
	clear(roomVMCache)
}

// InvalidateRoomVM removes the cached VM for a room so the next call reloads
// the script from disk. Call this after saving a room script via the admin API.
func InvalidateRoomVM(roomId int) {
	delete(roomVMCache, roomId)
}

func PruneRoomVMs(roomIds ...int) {
	pruneCt := 0
	defer func() {
		if pruneCt > 0 {
			mudlog.Info("PruneRoomVMs", "Removed VM Count", pruneCt)
		}
	}()

	if len(roomIds) > 0 {
		for _, roomId := range roomIds {
			if _, ok := roomVMCache[roomId]; ok {
				pruneCt++
				delete(roomVMCache, roomId)
			}
		}
		return
	}
	for roomId, _ := range roomVMCache {
		if !rooms.IsRoomLoaded(roomId) {
			pruneCt++
			delete(roomVMCache, roomId)
		}
	}
}

func TryRoomScriptEvent(eventName string, userId int, roomId int) (bool, error) {

	vmw, err := getRoomVM(roomId)
	if err != nil {
		return false, err
	}

	timestart := time.Now()
	defer func() {
		mudlog.Debug("TryRoomScriptEvent()", "eventName", eventName, "roomId", roomId, "time", time.Since(timestart))
	}()

	if onCommandFunc, ok := vmw.GetFunction(eventName); ok {

		// Set forced ansi tag wrappers
		userTextWrap.Set(`script-text`, ``, ``)
		roomTextWrap.Set(`script-text`, ``, ``)

		sUser := GetActor(userId, 0)
		sRoom := GetRoom(roomId)

		res, err := runCallable(vmw, scriptRoomTimeout, onCommandFunc,
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
	}

	return false, ErrEventNotFound
}

func TryRoomTryEnterEvent(userId int, destRoomId int) (bool, error) {

	vmw, err := getRoomVM(destRoomId)
	if err != nil {
		return false, err
	}

	timestart := time.Now()
	defer func() {
		mudlog.Debug("TryRoomTryEnterEvent()", "destRoomId", destRoomId, "time", time.Since(timestart))
	}()

	if onTryEnterFunc, ok := vmw.GetFunction(`onTryEnter`); ok {

		// Set forced ansi tag wrappers
		userTextWrap.Set(`script-text`, ``, ``)
		roomTextWrap.Set(`script-text`, ``, ``)

		sUser := GetActor(userId, 0)
		sRoom := GetRoom(destRoomId)

		res, err := runCallable(vmw, scriptRoomTimeout, onTryEnterFunc,
			vmw.ToValue(sUser),
			vmw.ToValue(sRoom),
		)

		userTextWrap.Reset()
		roomTextWrap.Reset()

		if err != nil {
			return false, fmt.Errorf("onTryEnter(): %w", err)
		}

		if boolVal, ok := res.Export().(bool); ok && !boolVal {
			return true, nil
		}

		return false, nil
	}

	return false, ErrEventNotFound
}

func TryRoomTryExitEvent(exitName string, userId int, roomId int) (bool, error) {

	vmw, err := getRoomVM(roomId)
	if err != nil {
		return false, err
	}

	timestart := time.Now()
	defer func() {
		mudlog.Debug("TryRoomTryExitEvent()", "exitName", exitName, "roomId", roomId, "time", time.Since(timestart))
	}()

	if onTryExitFunc, ok := vmw.GetFunction(`onTryExit`); ok {

		// Set forced ansi tag wrappers
		userTextWrap.Set(`script-text`, ``, ``)
		roomTextWrap.Set(`script-text`, ``, ``)

		sUser := GetActor(userId, 0)
		sRoom := GetRoom(roomId)

		res, err := runCallable(vmw, scriptRoomTimeout, onTryExitFunc,
			vmw.ToValue(exitName),
			vmw.ToValue(sUser),
			vmw.ToValue(sRoom),
		)

		userTextWrap.Reset()
		roomTextWrap.Reset()

		if err != nil {
			return false, fmt.Errorf("onTryExit(): %w", err)
		}

		if boolVal, ok := res.Export().(bool); ok && !boolVal {
			return true, nil
		}

		return false, nil
	}

	return false, ErrEventNotFound
}

func TryRoomIdleEvent(roomId int) (bool, error) {

	vmw, err := getRoomVM(roomId)
	if err != nil {
		return false, err
	}

	timestart := time.Now()
	defer func() {
		mudlog.Debug("TryRoomIdleEvent()", "roomId", roomId, "time", time.Since(timestart))
	}()

	if onCommandFunc, ok := vmw.GetFunction(`onIdle`); ok {

		// Set forced ansi tag wrappers
		userTextWrap.Set(`script-text`, ``, ``)
		roomTextWrap.Set(`script-text`, ``, ``)

		sRoom := GetRoom(roomId)

		res, err := runCallable(vmw, scriptRoomTimeout, onCommandFunc,
			vmw.ToValue(sRoom),
		)

		userTextWrap.Reset()
		roomTextWrap.Reset()

		if err != nil {
			return false, fmt.Errorf("TryRoomIdleEvent(): %w", err)
		}

		if boolVal, ok := res.Export().(bool); ok {
			return boolVal, nil
		}
	}

	return false, ErrEventNotFound
}

func TryRoomCommand(cmd string, rest string, userId int) (bool, error) {

	user := users.GetByUserId(userId)
	if user == nil {
		return false, errors.New("user not found")
	}

	room := rooms.LoadRoom(user.Character.RoomId)
	if room == nil {
		return false, fmt.Errorf("room %d not found", user.Character.RoomId)
	}

	altCmd, _ := room.FindExitByName(cmd)

	if room != nil {

		for _, mobInstanceId := range room.GetMobs() {
			if handled, err := TryMobCommand(cmd, rest, mobInstanceId, userId, `user`); err == nil {
				if handled {
					return true, nil
				}
			}

		}
	}

	vmw, err := getRoomVM(user.Character.RoomId)
	if err != nil {
		return false, err
	}

	timestart := time.Now()
	defer func() {
		mudlog.Debug("TryRoomCommand()", "cmd", cmd, "roomId", user.Character.RoomId, "time", time.Since(timestart))
	}()

	onCommandFunc, cmdFound := vmw.GetFunction(`onCommand_` + cmd)
	if !cmdFound && altCmd != `` {
		onCommandFunc, cmdFound = vmw.GetFunction(`onCommand_` + altCmd)
	}

	if cmdFound {

		// Set forced ansi tag wrappers
		userTextWrap.Set(`script-text`, ``, ``)
		roomTextWrap.Set(`script-text`, ``, ``)

		sUser := GetUser(userId)
		sRoom := GetRoom(user.Character.RoomId)

		res, err := runCallable(vmw, scriptRoomTimeout, onCommandFunc,
			vmw.ToValue(rest),
			vmw.ToValue(sUser),
			vmw.ToValue(sRoom),
		)

		userTextWrap.Reset()
		roomTextWrap.Reset()

		if err != nil {
			return false, fmt.Errorf("onCommand_%s(): %w", cmd, err)
		}

		if boolVal, ok := res.Export().(bool); ok {
			return boolVal, nil
		}

	} else if onCommandFunc, ok := vmw.GetFunction(`onCommand`); ok {

		// Set forced ansi tag wrappers
		userTextWrap.Set(`script-text`, ``, ``)
		roomTextWrap.Set(`script-text`, ``, ``)

		sUser := GetUser(userId)
		sRoom := GetRoom(user.Character.RoomId)

		res, err := runCallable(vmw, scriptRoomTimeout, onCommandFunc,
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
	}

	return false, ErrEventNotFound
}

func getRoomVM(roomId int) (scriptVM, error) {

	if vmw, ok := roomVMCache[roomId]; ok {
		if vmw == nil {
			return nil, errNoScript
		}
		if scriptHotReload {
			room := rooms.LoadRoom(roomId)
			if room != nil {
				if info, err := os.Stat(room.GetScriptPath()); err == nil {
					if info.ModTime().After(vmw.LoadedAt()) {
						delete(roomVMCache, roomId)
						// fall through to reload
					} else {
						return vmw, nil
					}
				} else {
					return vmw, nil
				}
			} else {
				return vmw, nil
			}
		} else {
			return vmw, nil
		}
	}

	room := rooms.LoadRoom(roomId)
	if room == nil {
		return nil, fmt.Errorf("room not found: %d", roomId)
	}

	script := room.GetScript()
	if len(script) == 0 {
		roomVMCache[roomId] = nil
		return nil, errNoScript
	}

	src := sourceFromPath(room.GetScriptPath(), script)
	vmw, err := loadVM(fmt.Sprintf(`room-%d`, roomId), src, func(vm scriptVM) error {
		if fn, ok := vm.GetFunction(`onLoad`); ok {
			sRoom := GetRoom(roomId)
			_, err := vm.Call(scriptLoadTimeout, fn, vm.ToValue(sRoom))
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	roomVMCache[roomId] = vmw
	return vmw, nil
}
