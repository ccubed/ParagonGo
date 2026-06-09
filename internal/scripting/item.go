package scripting

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

var (
	itemVMCache       = make(map[string]scriptVM)
	scriptItemTimeout = 50 * time.Millisecond
)

func ClearItemVMs() {
	clear(itemVMCache)
}

// PruneItemVMs is intentionally a no-op. Item VMs are keyed by item spec ID
// and are not tied to any instance lifecycle.
func PruneItemVMs(instanceIds ...int) {
}

// InvalidateItemVM removes the cached VM for an item spec so the next call
// reloads the script from disk. Call this after saving an item script via the
// admin API.
func InvalidateItemVM(itemId int) {
	delete(itemVMCache, strconv.Itoa(itemId))
}

func TryItemScriptEvent(eventName string, item items.Item, userId int) (bool, error) {

	sItem := GetItem(item)

	timestart := time.Now()
	defer func() {
		mudlog.Debug("TryItemScriptEvent()", "eventName", eventName, "item", item, "time", time.Since(timestart))
	}()

	vmw, err := getItemVM(sItem)
	if err != nil {
		return false, err
	}

	if onCommandFunc, ok := vmw.GetFunction(eventName); ok {

		sUser := GetActor(userId, 0)
		sRoom := GetRoom(sUser.GetRoomId())

		res, err := runCallable(vmw, scriptItemTimeout, onCommandFunc,
			vmw.ToValue(sUser),
			vmw.ToValue(sItem),
			vmw.ToValue(sRoom),
		)
		if err != nil {
			return false, fmt.Errorf("%s(): %w", eventName, err)
		}

		if eventName != `onLost` {
			// Save any changes that might have happened to the item
			sUser.characterRecord.UpdateItem(item, *sItem.itemRecord)
		}

		if boolVal, ok := res.Export().(bool); ok {
			return boolVal, nil
		}

	}

	return false, ErrEventNotFound
}

func TryItemCommand(cmd string, item items.Item, userId int) (bool, error) {

	sItem := GetItem(item)

	timestart := time.Now()
	defer func() {
		mudlog.Debug("TryItemCommand()", "cmd", cmd, "itemId", item.ItemId, "userId", userId, "time", time.Since(timestart))
	}()

	vmw, err := getItemVM(sItem)
	if err != nil {
		return false, err
	}

	if onCommandFunc, ok := vmw.GetFunction(`onCommand_` + cmd); ok {

		sUser := GetActor(userId, 0)
		sRoom := GetRoom(sUser.GetRoomId())

		res, err := runCallable(vmw, scriptItemTimeout, onCommandFunc,
			vmw.ToValue(sUser),
			vmw.ToValue(sItem),
			vmw.ToValue(sRoom),
		)
		if err != nil {
			return false, fmt.Errorf("onCommand_%s(): %w", cmd, err)
		}

		// Save any changes that might have happened to the item
		sUser.characterRecord.UpdateItem(item, *sItem.itemRecord)

		if boolVal, ok := res.Export().(bool); ok {
			return boolVal, nil
		}

	} else if onCommandFunc, ok := vmw.GetFunction(`onCommand`); ok {

		sUser := GetActor(userId, 0)
		sRoom := GetRoom(sUser.GetRoomId())

		res, err := runCallable(vmw, scriptItemTimeout, onCommandFunc,
			vmw.ToValue(cmd),
			vmw.ToValue(sUser),
			vmw.ToValue(sItem),
			vmw.ToValue(sRoom),
		)
		if err != nil {
			return false, fmt.Errorf("onCommand(): %w", err)
		}

		// Save any changes that might have happened to the item
		sUser.characterRecord.UpdateItem(item, *sItem.itemRecord)

		if boolVal, ok := res.Export().(bool); ok {
			return boolVal, nil
		}

	}

	return false, ErrEventNotFound
}

func TryItemTryPurchaseEvent(item items.Item, userId int) (bool, error) {

	sItem := GetItem(item)

	timestart := time.Now()
	defer func() {
		mudlog.Debug("TryItemTryPurchaseEvent()", "itemId", item.ItemId, "userId", userId, "time", time.Since(timestart))
	}()

	vmw, err := getItemVM(sItem)
	if err != nil {
		return false, err
	}

	if onTryPurchaseFunc, ok := vmw.GetFunction(`onTryPurchase`); ok {

		sUser := GetActor(userId, 0)
		sRoom := GetRoom(sUser.GetRoomId())

		res, err := runCallable(vmw, scriptItemTimeout, onTryPurchaseFunc,
			vmw.ToValue(sUser),
			vmw.ToValue(sItem),
			vmw.ToValue(sRoom),
		)
		if err != nil {
			return false, fmt.Errorf("onTryPurchase(): %w", err)
		}

		if boolVal, ok := res.Export().(bool); ok && !boolVal {
			return true, nil
		}

		return false, nil
	}

	return false, ErrEventNotFound
}

func getItemVM(sItem *ScriptItem) (scriptVM, error) {

	scriptId := strconv.Itoa(sItem.ItemId())

	if vmw, ok := itemVMCache[scriptId]; ok {
		if vmw == nil {
			return nil, errNoScript
		}
		if scriptHotReload {
			spec := items.GetItemSpec(sItem.ItemId())
			if spec != nil {
				if info, err := os.Stat(spec.GetScriptPath()); err == nil {
					if info.ModTime().After(vmw.LoadedAt()) {
						delete(itemVMCache, scriptId)
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

	script := sItem.getScript()
	if len(script) == 0 {
		itemVMCache[scriptId] = nil
		return nil, errNoScript
	}

	var scriptPath string
	if spec := items.GetItemSpec(sItem.ItemId()); spec != nil {
		scriptPath = spec.GetScriptPath()
	}
	src := sourceFromPath(scriptPath, script)
	vmw, err := loadVM(fmt.Sprintf(`item-%s`, scriptId), src, nil)
	if err != nil {
		return nil, err
	}

	itemVMCache[scriptId] = vmw
	return vmw, nil
}
