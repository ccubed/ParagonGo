package scripting

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

var (
	mobVMCache       = make(map[string]scriptVM)
	scriptMobTimeout = 50 * time.Millisecond
)

func ClearMobVMs() {
	clear(mobVMCache)
}

// PruneMobVMs is intentionally a no-op. Mob VMs are keyed by mob type ID and
// script tag, not by instance ID, so there is no per-instance lifecycle to
// prune. Use ClearMobVMs to evict all mob VMs at once.
func PruneMobVMs(instanceIds ...int) {
}

// InvalidateMobVM removes the cached VM for a specific mob type + script tag
// combination. Use this when a script file has been updated for a mob that
// uses a non-default script tag.
func InvalidateMobVM(mobTypeId int, scriptTag string) {
	key := fmt.Sprintf(`%d-%s`, mobTypeId, scriptTag)
	delete(mobVMCache, key)
}

// InvalidateMobVMById removes all cached VMs for the given mob type ID,
// regardless of script tag. This is the correct call after saving a mob script
// via the admin API because the tag is not known at that point.
func InvalidateMobVMById(mobTypeId int) {
	prefix := fmt.Sprintf(`%d-`, mobTypeId)
	// Also handle the no-tag key: "{mobTypeId}-"
	for key := range mobVMCache {
		if strings.HasPrefix(key, prefix) || key == fmt.Sprintf(`%d-`, mobTypeId) {
			delete(mobVMCache, key)
		}
	}
	// Handle the plain "{mobTypeId}-" key (empty script tag)
	delete(mobVMCache, fmt.Sprintf(`%d-`, mobTypeId))
}

func TryPlayerDownedEvent(mobInstanceId int, downedPlayerId int) (bool, error) {
	sMob := GetActor(0, mobInstanceId)
	if sMob == nil {
		return false, fmt.Errorf("mob not found")
	}

	vmw, err := getMobVM(sMob)
	if err != nil {
		return false, err
	}

	tUser := GetActor(downedPlayerId, 0)
	if tUser == nil {
		return false, fmt.Errorf("player not found")
	}

	timestart := time.Now()
	defer func() {
		mudlog.Debug("TryPlayerDownedEvent()", "mobInstanceId", mobInstanceId, "downedPlayerId", downedPlayerId, "time", time.Since(timestart))
	}()

	if onCommandFunc, ok := vmw.GetFunction(`onPlayerDowned`); ok {

		sRoom := GetRoom(sMob.GetRoomId())

		res, err := runCallable(vmw, scriptRoomTimeout, onCommandFunc,
			vmw.ToValue(sMob),
			vmw.ToValue(tUser),
			vmw.ToValue(sRoom),
		)
		if err != nil {
			return false, fmt.Errorf("onPlayerDowned(): %w", err)
		}

		if boolVal, ok := res.Export().(bool); ok {
			return boolVal, nil
		}
	}

	return false, ErrEventNotFound
}

func TryMobScriptEvent(eventName string, mobInstanceId int, sourceId int, sourceType string, details map[string]any) (bool, error) {

	sMob := GetActor(0, mobInstanceId)
	if sMob == nil {
		return false, fmt.Errorf("mob not found")
	}

	vmw, err := getMobVM(sMob)
	if err != nil {
		return false, err
	}

	timestart := time.Now()
	defer func() {
		if eventName != "onIdle" {
			mudlog.Debug("TryMobScriptEvent()", "eventName", eventName, "MobId", sMob.MobTypeId(), "time", time.Since(timestart))
		}
	}()
	if onCommandFunc, ok := vmw.GetFunction(eventName); ok {

		if details == nil {
			details = make(map[string]any)
		}

		sRoom := GetRoom(sMob.GetRoomId())

		details["sourceId"] = sourceId
		details["sourceType"] = sourceType

		res, err := runCallable(vmw, scriptRoomTimeout, onCommandFunc,
			vmw.ToValue(sMob),
			vmw.ToValue(sRoom),
			vmw.ToValue(details),
		)
		if err != nil {
			return false, fmt.Errorf("%s(): %w", eventName, err)
		}

		if boolVal, ok := res.Export().(bool); ok {
			return boolVal, nil
		}
	}

	return false, ErrEventNotFound
}

func TryMobCommand(cmd string, rest string, mobInstanceId int, sourceId int, sourceType string) (bool, error) {

	sMob := GetActor(0, mobInstanceId)
	if sMob == nil {
		PruneMobVMs(mobInstanceId)
		return false, fmt.Errorf("mob not found")
	}

	vmw, err := getMobVM(sMob)
	if err != nil {
		return false, err
	}

	timestart := time.Now()
	defer func() {
		mudlog.Debug("TryMobCommand()", "cmd", cmd, "MobId", sMob.MobTypeId(), "time", time.Since(timestart))
	}()

	if onCommandFunc, ok := vmw.GetFunction(`onCommand_` + cmd); ok {

		details := map[string]interface{}{
			`sourceId`:   sourceId,
			`sourceType`: sourceType,
		}

		sRoom := GetRoom(sMob.mobRecord.Character.RoomId)

		res, err := runCallable(vmw, scriptRoomTimeout, onCommandFunc,
			vmw.ToValue(rest),
			vmw.ToValue(sMob),
			vmw.ToValue(sRoom),
			vmw.ToValue(details),
		)
		if err != nil {
			return false, fmt.Errorf("onCommand_%s(): %w", cmd, err)
		}

		if boolVal, ok := res.Export().(bool); ok {
			return boolVal, nil
		}

	} else if onCommandFunc, ok := vmw.GetFunction(`onCommand`); ok {

		details := map[string]interface{}{
			`sourceId`:   sourceId,
			`sourceType`: sourceType,
		}

		sRoom := GetRoom(sMob.GetRoomId())

		res, err := runCallable(vmw, scriptRoomTimeout, onCommandFunc,
			vmw.ToValue(cmd),
			vmw.ToValue(rest),
			vmw.ToValue(sMob),
			vmw.ToValue(sRoom),
			vmw.ToValue(details),
		)
		if err != nil {
			return false, fmt.Errorf("onCommand(): %w", err)
		}

		if boolVal, ok := res.Export().(bool); ok {
			return boolVal, nil
		}
	}

	return false, ErrEventNotFound
}

func getMobVM(mobActor *ScriptActor) (scriptVM, error) {

	scriptId := fmt.Sprintf(`%d-%s`, mobActor.MobTypeId(), mobActor.getScriptTag())

	if vmw, ok := mobVMCache[scriptId]; ok {
		if vmw == nil {
			return nil, errNoScript
		}
		if scriptHotReload {
			scriptPath := mobActor.mobRecord.GetScriptPath()
			if info, err := os.Stat(scriptPath); err == nil {
				if info.ModTime().After(vmw.LoadedAt()) {
					delete(mobVMCache, scriptId)
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
	}

	script := mobActor.getScript()
	if len(script) == 0 {
		mobVMCache[scriptId] = nil
		return nil, errNoScript
	}

	src := sourceFromPath(mobActor.mobRecord.GetScriptPath(), script)
	vmw, err := loadVM(fmt.Sprintf(`mob-%s`, scriptId), src, func(vm scriptVM) error {
		if fn, ok := vm.GetFunction(`onLoad`); ok {
			_, err := vm.Call(scriptLoadTimeout, fn, vm.ToValue(mobActor))
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	mobVMCache[scriptId] = vmw
	return vmw, nil
}
