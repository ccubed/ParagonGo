package scripting

import (
	"fmt"
	"os"
	"time"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/colorpatterns"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/spells"
)

var (
	spellVMCache       = make(map[string]scriptVM)
	scriptSpellTimeout = 50 * time.Millisecond
)

func ClearSpellVMs() {
	clear(spellVMCache)
}

// PruneSpellVMs is intentionally a no-op. Spell VMs are keyed by spell ID
// and are not tied to any instance lifecycle.
func PruneSpellVMs(instanceIds ...int) {
}

// InvalidateSpellVM removes the cached VM for a spell so the next call reloads
// the script from disk. Call this after saving a spell script via the admin API.
func InvalidateSpellVM(spellId string) {
	delete(spellVMCache, spellId)
}

func TrySpellScriptEvent(eventName string, sourceUserId int, sourceMobInstanceId int, spellAggro characters.SpellAggroInfo) (bool, error) {

	spellInfo := spells.GetSpell(spellAggro.SpellId)
	if spellInfo == nil {
		return false, fmt.Errorf("spell %s not found", spellAggro.SpellId)
	}

	timestart := time.Now()
	defer func() {
		mudlog.Debug("TrySpellScriptEvent()", "eventName", eventName, "spellId", spellAggro.SpellId, "spellRest", spellAggro.SpellRest, "TargetUsers", spellAggro.TargetUserIds, "TargetMobs", spellAggro.TargetMobInstanceIds, "time", time.Since(timestart))
	}()

	vmw, err := getSpellVM(spellAggro.SpellId)
	if err != nil {
		mudlog.Debug("TrySpellScriptEvent()", "error", err)
		return false, err
	}

	sourceActor := GetActor(sourceUserId, sourceMobInstanceId)

	if eventName != `onCast` && eventName != `onWait` && eventName != `onMagic` && eventName != `onFail` {
		return false, err
	}

	var stringArg string = ""
	var singleTargetArg *ScriptActor = nil
	var multiTargetArg []*ScriptActor = nil

	if spellInfo.Type == spells.Neutral {

		// arg is just whatever the user entered after the spell casting command
		stringArg = spellAggro.SpellRest

	} else if spellInfo.Type == spells.HelpSingle || spellInfo.Type == spells.HarmSingle {

		// arg is a single actor
		if len(spellAggro.TargetUserIds) > 0 {
			singleTargetArg = GetActor(spellAggro.TargetUserIds[0], 0)
		} else if len(spellAggro.TargetMobInstanceIds) > 0 {
			singleTargetArg = GetActor(0, spellAggro.TargetMobInstanceIds[0])
		}

		// If no longer in the same room, notify the user
		if singleTargetArg == nil || (sourceActor.GetRoomId() != singleTargetArg.GetRoomId()) {
			sourceActor.SendText(`Your target cannot be found.`)
			return true, nil
		}

	} else if spellInfo.Type == spells.HelpMulti || spellInfo.Type == spells.HarmMulti {

		// arg is a list of actors
		multiTargetArg = []*ScriptActor{}
		for _, targetUserId := range spellAggro.TargetUserIds {
			if uActor := GetActor(targetUserId, 0); uActor != nil {
				if uActor.GetRoomId() == sourceActor.GetRoomId() {
					multiTargetArg = append(multiTargetArg, uActor)
				}
			}
		}
		for _, targetMobInstanceId := range spellAggro.TargetMobInstanceIds {
			if mActor := GetActor(0, targetMobInstanceId); mActor != nil {
				if mActor.GetRoomId() == sourceActor.GetRoomId() {
					multiTargetArg = append(multiTargetArg, mActor)
				}
			}
		}

		if len(multiTargetArg) == 0 {
			sourceActor.SendText(`Your target cannot be found.`)
			return true, nil
		}

	}

	if onCommandFunc, ok := vmw.GetFunction(eventName); ok {

		// Set forced ansi tag wrappers
		userTextWrap.Set(`spell-text`, ``, `pink`, colorpatterns.Stretch)
		roomTextWrap.Set(`spell-text`, ``, `pink`, colorpatterns.Stretch)

		var argValue any
		if multiTargetArg != nil {
			argValue = vmw.ToValue(multiTargetArg)
		} else if singleTargetArg != nil {
			argValue = vmw.ToValue(singleTargetArg)
		} else {
			argValue = vmw.ToValue(stringArg)
		}

		res, err := runCallable(vmw, scriptSpellTimeout, onCommandFunc,
			vmw.ToValue(sourceActor),
			argValue,
		)

		// Reset forced ansi tag wrappers
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

func getSpellVM(scriptId string) (scriptVM, error) {

	if vmw, ok := spellVMCache[scriptId]; ok {
		if vmw == nil {
			return nil, errNoScript
		}
		if scriptHotReload {
			spellData := spells.GetSpell(scriptId)
			if spellData != nil {
				if info, err := os.Stat(spellData.GetScriptPath()); err == nil {
					if info.ModTime().After(vmw.LoadedAt()) {
						delete(spellVMCache, scriptId)
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

	spellData := spells.GetSpell(scriptId)
	if spellData == nil {
		return nil, fmt.Errorf("spell %s not found", scriptId)
	}

	script := spellData.GetScript()
	if len(script) == 0 {
		spellVMCache[scriptId] = nil
		return nil, errNoScript
	}

	src := sourceFromPath(spellData.GetScriptPath(), script)
	vmw, err := loadVM(fmt.Sprintf(`spell-%s`, scriptId), src, nil)
	if err != nil {
		return nil, err
	}

	spellVMCache[scriptId] = vmw
	return vmw, nil
}
