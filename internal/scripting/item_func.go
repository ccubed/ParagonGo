package scripting

import (
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func setItemFunctions(vm registrar) {
	vm.Set(`CreateItem`, CreateItem)
}

func newScriptItem(i items.Item) ScriptItem {
	return ScriptItem{i, &i}
}

type ScriptItem struct {
	originalItem items.Item
	itemRecord   *items.Item
}

func (i ScriptItem) ItemId() int {
	if i.itemRecord != nil {
		return i.itemRecord.ItemId
	}
	return 0
}

func (i ScriptItem) getScript() string {
	if i.itemRecord != nil {
		return i.itemRecord.GetScript()
	}
	return ""
}

func (i ScriptItem) GetUsesLeft() int {
	return i.itemRecord.Uses
}

func (i ScriptItem) SetUsesLeft(amount int) int {
	if i.itemRecord.Uses+amount < 0 {
		i.itemRecord.Uses = 0
	} else {
		i.itemRecord.Uses = amount
	}

	return i.itemRecord.Uses
}

func (i ScriptItem) AddUsesLeft(amount int) int {

	if i.itemRecord.Uses+amount < 0 {
		i.itemRecord.Uses = 0
	} else {
		i.itemRecord.Uses += amount
	}

	return i.itemRecord.Uses
}

func (i ScriptItem) GetLastUsedRound() uint64 {
	return i.itemRecord.LastUsedRound
}

func (i ScriptItem) MarkLastUsed(clear ...bool) uint64 {
	if len(clear) > 0 && clear[0] {
		i.itemRecord.LastUsedRound = 0
	} else {
		i.itemRecord.LastUsedRound = util.GetRoundCount()
	}
	return i.itemRecord.LastUsedRound
}

func (i ScriptItem) Name(simpleVersion ...bool) string {
	if len(simpleVersion) > 0 && simpleVersion[0] {
		return i.NameSimple()
	}
	return i.itemRecord.DisplayName()
}

func (i ScriptItem) NameSimple() string {
	return i.itemRecord.NameSimple()
}

func (i ScriptItem) NameComplex() string {
	return i.itemRecord.NameComplex()
}

func (i ScriptItem) SetTempData(key string, value any) {
	i.itemRecord.SetTempData(key, value)
}

func (i ScriptItem) GetTempData(key string) any {
	return i.itemRecord.GetTempData(key)
}

func (i ScriptItem) ShorthandId() string {
	return i.itemRecord.ShorthandId()
}

func (i ScriptItem) Rename(newName string, displayNameOrStyle ...string) {
	i.itemRecord.Rename(newName, displayNameOrStyle...)
}

func (i ScriptItem) Redescribe(newDescription string) {
	i.itemRecord.Redescribe(newDescription)
}

// Converts an item into a ScriptItem for use in the scripting engine
func GetItem(i items.Item) *ScriptItem {
	sItm := newScriptItem(i)
	return &sItm
}

// ////////////////////////////////////////////////////////
//
// # These functions get exported to the scripting engine
//
// ////////////////////////////////////////////////////////

// CreateItem creates a NEW instance of an item by id
func CreateItem(itemId int) *ScriptItem {
	i := items.New(itemId)
	if i.ItemId == 0 {
		return nil
	}
	sItm := newScriptItem(i)
	return &sItm
}

// ////////////////////////////////////////////////////////
//
// New ScriptItem methods
//
// ////////////////////////////////////////////////////////

func (i ScriptItem) GetType() string {
	if i.itemRecord == nil {
		return ``
	}
	return string(i.itemRecord.GetSpec().Type)
}

func (i ScriptItem) GetSubtype() string {
	if i.itemRecord == nil {
		return ``
	}
	return string(i.itemRecord.GetSpec().Subtype)
}

func (i ScriptItem) GetValue() int {
	if i.itemRecord == nil {
		return 0
	}
	return i.itemRecord.GetSpec().Value
}

func (i ScriptItem) GetDescription() string {
	if i.itemRecord == nil {
		return ``
	}
	return i.itemRecord.GetSpec().Description
}

func (i ScriptItem) GetQuestToken() string {
	if i.itemRecord == nil {
		return ``
	}
	return i.itemRecord.GetSpec().QuestToken
}

func (i ScriptItem) GetElement() string {
	if i.itemRecord == nil {
		return ``
	}
	return string(i.itemRecord.GetSpec().Element)
}

func (i ScriptItem) GetBuffIds() []int {
	if i.itemRecord == nil {
		return []int{}
	}
	src := i.itemRecord.GetSpec().BuffIds
	result := make([]int, len(src))
	copy(result, src)
	return result
}

func (i ScriptItem) GetWornBuffIds() []int {
	if i.itemRecord == nil {
		return []int{}
	}
	src := i.itemRecord.GetSpec().WornBuffIds
	result := make([]int, len(src))
	copy(result, src)
	return result
}

func (i ScriptItem) GetStatMods() map[string]int {
	result := make(map[string]int)
	if i.itemRecord == nil {
		return result
	}
	for k, v := range i.itemRecord.GetSpec().StatMods {
		result[k] = v
	}
	return result
}

func (i ScriptItem) GetDamageReduction() int {
	if i.itemRecord == nil {
		return 0
	}
	return i.itemRecord.GetSpec().DamageReduction
}

func (i ScriptItem) GetDamage() map[string]any {
	if i.itemRecord == nil {
		return map[string]any{`attacks`: 0, `diceCount`: 0, `diceSides`: 0, `bonusDamage`: 0, `diceRoll`: ``}
	}
	d := i.itemRecord.GetSpec().Damage
	return map[string]any{
		`attacks`:     d.Attacks,
		`diceCount`:   d.DiceCount,
		`diceSides`:   d.SideCount,
		`bonusDamage`: d.BonusDamage,
		`diceRoll`:    d.DiceRoll,
	}
}

func (i ScriptItem) GetBreakChance() int {
	if i.itemRecord == nil {
		return 0
	}
	return int(i.itemRecord.GetSpec().BreakChance)
}

func (i ScriptItem) GetKeyLockId() string {
	if i.itemRecord == nil {
		return ``
	}
	return i.itemRecord.GetSpec().KeyLockId
}

func (i ScriptItem) IsCursed() bool {
	if i.itemRecord == nil {
		return false
	}
	return i.itemRecord.IsCursed()
}

func (i ScriptItem) HasUses() bool {
	if i.itemRecord == nil {
		return false
	}
	return i.itemRecord.Uses > 0
}

func (i ScriptItem) IsWearable() bool {
	if i.itemRecord == nil {
		return false
	}
	spec := i.itemRecord.GetSpec()
	return spec.Subtype == items.Wearable
}

func (i ScriptItem) IsWeapon() bool {
	if i.itemRecord == nil {
		return false
	}
	return i.itemRecord.GetSpec().Type == items.Weapon
}
