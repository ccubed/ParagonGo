package scripting

import (
	"errors"
	"fmt"
	"time"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/pets"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/dop251/goja"
)

var (
	petVMCache       = make(map[string]*VMWrapper)
	scriptPetTimeout = 50 * time.Millisecond
)

func ClearPetVMs() {
	clear(petVMCache)
}

func PrunePetVMs(petTypes ...string) {
	if len(petTypes) == 0 {
		return
	}
	for _, pt := range petTypes {
		delete(petVMCache, pt)
	}
}

// TryPetScriptEvent fires a named event on the pet script (e.g. "onCommand", "PetAct").
// userId is the owner of the pet.
func TryPetScriptEvent(eventName string, userId int) (bool, error) {

	user := users.GetByUserId(userId)
	if user == nil {
		return false, errors.New("user not found")
	}

	if !user.Character.Pet.Exists() {
		return false, errors.New("user has no pet")
	}

	sPet := GetPet(&user.Character.Pet)
	if sPet == nil {
		return false, errors.New("pet not found")
	}

	vmw, err := getPetVM(sPet)
	if err != nil {
		return false, err
	}

	timestart := time.Now()
	defer func() {
		mudlog.Debug("TryPetScriptEvent()", "eventName", eventName, "petType", sPet.Type(), "time", time.Since(timestart))
	}()

	if onFunc, ok := vmw.GetFunction(eventName); ok {

		sActor := GetActor(userId, 0)
		sRoom := GetRoom(sActor.GetRoomId())

		userTextWrap.Set(`script-text`, ``, ``)
		roomTextWrap.Set(`script-text`, ``, ``)

		tmr := time.AfterFunc(scriptPetTimeout, func() {
			vmw.VM.Interrupt(errTimeout)
		})
		res, err := onFunc(goja.Undefined(),
			vmw.VM.ToValue(sPet),
			vmw.VM.ToValue(sActor),
			vmw.VM.ToValue(sRoom),
		)
		vmw.VM.ClearInterrupt()
		tmr.Stop()

		userTextWrap.Reset()
		roomTextWrap.Reset()

		if err != nil {
			finalErr := fmt.Errorf("%s(): %w", eventName, err)
			if _, ok := finalErr.(*goja.Exception); ok {
				mudlog.Error("JSVM", "exception", finalErr)
				return false, finalErr
			} else if errors.Is(finalErr, errTimeout) {
				mudlog.Error("JSVM", "interrupted", finalErr)
				return false, finalErr
			}
			mudlog.Error("JSVM", "error", finalErr)
			return false, finalErr
		}

		if boolVal, ok := res.Export().(bool); ok {
			return boolVal, nil
		}
	}

	return false, ErrEventNotFound
}

// TryPetCommand checks whether the pet script intercepts a command typed by its owner.
func TryPetCommand(cmd string, rest string, userId int) (bool, error) {

	user := users.GetByUserId(userId)
	if user == nil {
		return false, errors.New("user not found")
	}

	if !user.Character.Pet.Exists() {
		return false, ErrEventNotFound
	}

	sPet := GetPet(&user.Character.Pet)
	if sPet == nil {
		return false, ErrEventNotFound
	}

	vmw, err := getPetVM(sPet)
	if err != nil {
		return false, err
	}

	timestart := time.Now()
	defer func() {
		mudlog.Debug("TryPetCommand()", "cmd", cmd, "petType", sPet.Type(), "userId", userId, "time", time.Since(timestart))
	}()

	sActor := GetActor(userId, 0)
	sRoom := GetRoom(sActor.GetRoomId())

	if onCommandFunc, ok := vmw.GetFunction(`onCommand_` + cmd); ok {

		userTextWrap.Set(`script-text`, ``, ``)
		roomTextWrap.Set(`script-text`, ``, ``)

		tmr := time.AfterFunc(scriptPetTimeout, func() {
			vmw.VM.Interrupt(errTimeout)
		})
		res, err := onCommandFunc(goja.Undefined(),
			vmw.VM.ToValue(rest),
			vmw.VM.ToValue(sPet),
			vmw.VM.ToValue(sActor),
			vmw.VM.ToValue(sRoom),
		)
		vmw.VM.ClearInterrupt()
		tmr.Stop()

		userTextWrap.Reset()
		roomTextWrap.Reset()

		if err != nil {
			finalErr := fmt.Errorf("onCommand_%s(): %w", cmd, err)
			if _, ok := finalErr.(*goja.Exception); ok {
				mudlog.Error("JSVM", "exception", finalErr)
				return false, finalErr
			} else if errors.Is(finalErr, errTimeout) {
				mudlog.Error("JSVM", "interrupted", finalErr)
				return false, finalErr
			}
			mudlog.Error("JSVM", "error", finalErr)
			return false, finalErr
		}

		if boolVal, ok := res.Export().(bool); ok {
			return boolVal, nil
		}

	} else if onCommandFunc, ok := vmw.GetFunction(`onCommand`); ok {

		userTextWrap.Set(`script-text`, ``, ``)
		roomTextWrap.Set(`script-text`, ``, ``)

		tmr := time.AfterFunc(scriptPetTimeout, func() {
			vmw.VM.Interrupt(errTimeout)
		})
		res, err := onCommandFunc(goja.Undefined(),
			vmw.VM.ToValue(cmd),
			vmw.VM.ToValue(rest),
			vmw.VM.ToValue(sPet),
			vmw.VM.ToValue(sActor),
			vmw.VM.ToValue(sRoom),
		)
		vmw.VM.ClearInterrupt()
		tmr.Stop()

		userTextWrap.Reset()
		roomTextWrap.Reset()

		if err != nil {
			finalErr := fmt.Errorf("onCommand(): %w", err)
			if _, ok := finalErr.(*goja.Exception); ok {
				mudlog.Error("JSVM", "exception", finalErr)
				return false, finalErr
			} else if errors.Is(finalErr, errTimeout) {
				mudlog.Error("JSVM", "interrupted", finalErr)
				return false, finalErr
			}
			mudlog.Error("JSVM", "error", finalErr)
			return false, finalErr
		}

		if boolVal, ok := res.Export().(bool); ok {
			return boolVal, nil
		}
	}

	return false, ErrEventNotFound
}

func getPetVM(sPet *ScriptPet) (*VMWrapper, error) {

	scriptId := sPet.petRecord.Type

	if vm, ok := petVMCache[scriptId]; ok {
		if vm == nil {
			return nil, errNoScript
		}
		return vm, nil
	}

	script := sPet.getScript()
	if len(script) == 0 {
		petVMCache[scriptId] = nil
		return nil, errNoScript
	}

	vmw, err := loadVM(fmt.Sprintf(`pet-%s`, scriptId), script, nil)
	if err != nil {
		return nil, err
	}

	petVMCache[scriptId] = vmw
	return vmw, nil
}

// InvalidatePetVM removes the cached VM for a given pet type so the next call
// reloads the script from disk. Called after SavePetScript.
func InvalidatePetVM(petType string) {
	delete(petVMCache, petType)
}

// GetPetSpec returns the definition for a pet type by name (for scripting helpers).
func GetPetSpec(petType string) *pets.Pet {
	cp := pets.GetPetCopy(petType)
	if !cp.Exists() {
		return nil
	}
	return &cp
}
