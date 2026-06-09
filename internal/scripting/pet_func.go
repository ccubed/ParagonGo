package scripting

import (
	"github.com/GoMudEngine/GoMud/internal/pets"
)

func setPetFunctions(vm registrar) {
}

// ScriptPet wraps a live pet record and exposes it to the scripting engine.
// The underlying record is a pointer into the owner's character data, so
// mutations are reflected immediately without any extra save call.
type ScriptPet struct {
	petRecord *pets.Pet
	userId    int
}

// Type returns the pet's type identifier (e.g. "dog", "cat", "owl").
func (p ScriptPet) Type() string {
	if p.petRecord != nil {
		return p.petRecord.Type
	}
	return ``
}

// Name returns the pet's display name, including ANSI colour tags and any
// hunger indicator. Use NameSimple() when you only need the plain text name.
func (p ScriptPet) Name() string {
	if p.petRecord != nil {
		return p.petRecord.DisplayName()
	}
	return ``
}

// NameSimple returns the plain text name of the pet with no colour tags or
// hunger indicator. Falls back to the type identifier if no name has been set.
func (p ScriptPet) NameSimple() string {
	if p.petRecord == nil {
		return ``
	}
	if p.petRecord.Name != `` {
		return p.petRecord.Name
	}
	return p.petRecord.Type
}

// SetName renames the pet. Pass an empty string to clear a custom name and
// revert to the type identifier.
func (p ScriptPet) SetName(name string) {
	if p.petRecord != nil {
		p.petRecord.Name = name
	}
}

// Level returns the pet's current level (1–10).
func (p ScriptPet) Level() int {
	if p.petRecord != nil {
		return p.petRecord.Level
	}
	return 0
}

// Food returns the pet's current hunger level as a string:
// "Starving" (0), "Hungry" (1), "Satisfied" (2), or "Full" (3).
func (p ScriptPet) Food() string {
	if p.petRecord != nil {
		return p.petRecord.Food.String()
	}
	return ``
}

// FoodLevel returns the pet's raw hunger value: 0 (starving) through 3 (full).
func (p ScriptPet) FoodLevel() int {
	if p.petRecord != nil {
		return int(p.petRecord.Food)
	}
	return 0
}

// Feed increases the pet's hunger level by one step, up to a maximum of 3
// (Full). Has no effect if the pet is already full.
func (p ScriptPet) Feed() {
	if p.petRecord != nil {
		p.petRecord.Food.Add()
	}
}

// Starve decreases the pet's hunger level by one step, down to a minimum of 0
// (Starving). Has no effect if the pet is already starving.
func (p ScriptPet) Starve() {
	if p.petRecord != nil {
		p.petRecord.Food.Remove()
	}
}

// GetStatMod returns the effective stat modifier the pet currently grants its
// owner for the named stat (e.g. "strength", "speed", "smarts"). Returns 0 if
// the pet has no modifier for that stat at its current level.
func (p ScriptPet) GetStatMod(statName string) int {
	if p.petRecord != nil {
		return p.petRecord.StatMod(statName)
	}
	return 0
}

// GetCapacity returns the number of items the pet can carry at its current
// level. Returns 0 if the pet type has no carry ability.
func (p ScriptPet) GetCapacity() int {
	if p.petRecord != nil {
		return p.petRecord.GetEffectiveCapacity()
	}
	return 0
}

// ItemCount returns the number of items the pet is currently carrying.
func (p ScriptPet) ItemCount() int {
	if p.petRecord != nil {
		return len(p.petRecord.Items)
	}
	return 0
}

// IsMissing returns true if the pet is currently absent (MissingCountdown > 0).
func (p ScriptPet) IsMissing() bool {
	if p.petRecord != nil {
		return p.petRecord.IsMissing()
	}
	return false
}

// GoMissing causes the pet to go absent for the given number of rounds.
// Pass 0 to return the pet immediately, firing PetReturn instead of PetLeave.
// Any positive value fires PetLeave and begins the countdown.
func (p ScriptPet) GoMissing(rounds int) {
	if p.petRecord == nil {
		return
	}
	if rounds <= 0 {
		if !p.petRecord.IsMissing() {
			return
		}
		p.petRecord.GoMissing(0)
		if p.userId > 0 {
			TryPetScriptEvent(`PetReturn`, p.userId)
		}
		return
	}
	if !p.petRecord.IsMissing() {
		p.petRecord.GoMissing(rounds)
		if p.userId > 0 {
			TryPetScriptEvent(`PetLeave`, p.userId)
		}
	} else {
		p.petRecord.GoMissing(rounds)
	}
}

// HasScript returns true if this pet type has a script file on disk.
func (p ScriptPet) HasScript() bool {
	if p.petRecord != nil {
		return p.petRecord.HasScript()
	}
	return false
}

// ////////////////////////////////////////////////////////
//
// Package-internal helpers
//
// ////////////////////////////////////////////////////////

func (p ScriptPet) getScript() string {
	if p.petRecord != nil {
		return p.petRecord.GetScript()
	}
	return ``
}

// GetPet returns a ScriptPet wrapping the given pet pointer.
// Returns nil if pet is nil or does not exist.
// Pass the owner's userId as the optional second argument to enable
// script-triggered events such as GoMissing.
func GetPet(pet *pets.Pet, userId ...int) *ScriptPet {
	if pet == nil || !pet.Exists() {
		return nil
	}
	sp := &ScriptPet{petRecord: pet}
	if len(userId) > 0 {
		sp.userId = userId[0]
	}
	return sp
}

// ////////////////////////////////////////////////////////
//
// New ScriptPet methods
//
// ////////////////////////////////////////////////////////

func (p ScriptPet) GetItems() []ScriptItem {
	if p.petRecord == nil {
		return []ScriptItem{}
	}
	itms := make([]ScriptItem, 0, len(p.petRecord.Items))
	for _, itm := range p.petRecord.Items {
		itms = append(itms, newScriptItem(itm))
	}
	return itms
}

func (p ScriptPet) FindItem(itemName string) *ScriptItem {
	if p.petRecord == nil {
		return nil
	}
	itm, found := p.petRecord.FindItem(itemName)
	if !found {
		return nil
	}
	si := newScriptItem(itm)
	return &si
}

func (p ScriptPet) StoreItem(itm ScriptItem) bool {
	if p.petRecord == nil || itm.itemRecord == nil {
		return false
	}
	return p.petRecord.StoreItem(*itm.itemRecord)
}

func (p ScriptPet) RemoveItem(itm ScriptItem) bool {
	if p.petRecord == nil || itm.itemRecord == nil {
		return false
	}
	return p.petRecord.RemoveItem(*itm.itemRecord)
}

func (p ScriptPet) GetBuffIds() []int {
	if p.petRecord == nil {
		return []int{}
	}
	return p.petRecord.GetBuffs()
}
