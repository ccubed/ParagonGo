package scripting

import (
	"fmt"
	"os"
	"time"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/pets"
	"github.com/GoMudEngine/GoMud/internal/users"
)

var (
	petVMCache       = make(map[string]scriptVM)
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
		return false, fmt.Errorf("user not found")
	}

	if !user.Character.Pet.Exists() {
		return false, fmt.Errorf("user has no pet")
	}

	sPet := GetPet(&user.Character.Pet, userId)
	if sPet == nil {
		return false, fmt.Errorf("pet not found")
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

		res, err := runCallable(vmw, scriptPetTimeout, onFunc,
			vmw.ToValue(sPet),
			vmw.ToValue(sActor),
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

// TryPetCommand checks whether the pet script intercepts a command typed by its owner.
func TryPetCommand(cmd string, rest string, userId int) (bool, error) {

	user := users.GetByUserId(userId)
	if user == nil {
		return false, fmt.Errorf("user not found")
	}

	if !user.Character.Pet.Exists() {
		return false, ErrEventNotFound
	}

	sPet := GetPet(&user.Character.Pet, userId)
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

		res, err := runCallable(vmw, scriptPetTimeout, onCommandFunc,
			vmw.ToValue(rest),
			vmw.ToValue(sPet),
			vmw.ToValue(sActor),
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

		userTextWrap.Set(`script-text`, ``, ``)
		roomTextWrap.Set(`script-text`, ``, ``)

		res, err := runCallable(vmw, scriptPetTimeout, onCommandFunc,
			vmw.ToValue(cmd),
			vmw.ToValue(rest),
			vmw.ToValue(sPet),
			vmw.ToValue(sActor),
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

func getPetVM(sPet *ScriptPet) (scriptVM, error) {

	scriptId := sPet.petRecord.Type

	if vmw, ok := petVMCache[scriptId]; ok {
		if vmw == nil {
			return nil, errNoScript
		}
		if scriptHotReload {
			spec := pets.GetPetCopy(scriptId)
			if spec.Exists() {
				if info, err := os.Stat(spec.GetScriptPath()); err == nil {
					if info.ModTime().After(vmw.LoadedAt()) {
						delete(petVMCache, scriptId)
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

	script := sPet.getScript()
	if len(script) == 0 {
		petVMCache[scriptId] = nil
		return nil, errNoScript
	}

	src := sourceFromPath(sPet.petRecord.GetScriptPath(), script)
	vmw, err := loadVM(fmt.Sprintf(`pet-%s`, scriptId), src, nil)
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
