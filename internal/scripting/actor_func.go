package scripting

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/dop251/goja"
)

func setActorFunctions(vm *goja.Runtime) {
	vm.Set(`GetUser`, GetUser)
	vm.Set(`GetMob`, GetMob)
	vm.Set(`ActorNames`, ActorNames)
}

type ScriptActor struct {
	userId          int
	mobInstanceId   int
	userRecord      *users.UserRecord
	mobRecord       *mobs.Mob
	characterRecord *characters.Character // Lets us bypass the user/mob check in many cases
}

func (a ScriptActor) UserId() int {
	return a.userId
}

func (a ScriptActor) InstanceId() int {
	return a.mobInstanceId
}

func (a ScriptActor) MobTypeId() int {
	if a.mobRecord != nil {
		return int(a.mobRecord.MobId)
	}
	return 0
}

func (a ScriptActor) GetRace() string {
	return a.characterRecord.Race()
}

func (a ScriptActor) GetTrueRace() string {
	if r := races.GetRace(a.characterRecord.RaceId); r != nil {
		return r.Name
	}
	return `Ghostly Spirit`
}

func (a ScriptActor) IsFormChanged() bool {
	return a.characterRecord.IsFormChanged()
}

func (a ScriptActor) ApplyFormChange(raceId int) bool {
	if races.GetRace(raceId) == nil {
		return false
	}
	a.characterRecord.ApplyFormChange(raceId)
	return true
}

func (a ScriptActor) RevertFormChange() bool {
	if !a.characterRecord.IsFormChanged() {
		return false
	}
	a.characterRecord.RevertFormChange()
	return true
}

func (a ScriptActor) GetSize() string {
	if r := races.GetRace(a.characterRecord.GetRaceId()); r != nil {
		return string(r.Size)
	}
	return string(races.Medium)
}

func (a ScriptActor) SendText(msg string) {
	if a.userRecord == nil {
		return
	}

	msg = userTextWrap.Wrap(msg)

	a.userRecord.SendText(msg)
}

func (a ScriptActor) GetLevel() int {
	return a.characterRecord.Level
}

func (a ScriptActor) GetStat(statName string) int {

	statName = strings.ToLower(statName)

	if strings.HasPrefix(statName, "st") {
		return a.characterRecord.Stats.Strength.ValueAdj
	}

	if strings.HasPrefix(statName, "sp") {
		return a.characterRecord.Stats.Speed.ValueAdj
	}

	if strings.HasPrefix(statName, "sm") {
		return a.characterRecord.Stats.Smarts.ValueAdj
	}

	if strings.HasPrefix(statName, "vi") {
		return a.characterRecord.Stats.Vitality.ValueAdj
	}

	if strings.HasPrefix(statName, "my") {
		return a.characterRecord.Stats.Mysticism.ValueAdj
	}

	if strings.HasPrefix(statName, "pe") {
		return a.characterRecord.Stats.Perception.ValueAdj
	}

	return 0
}

func (a ScriptActor) SetResetRoomId(roomId int) {
	if a.userRecord == nil {
		return
	}
	a.userRecord.Character.RoomIdOnReset = roomId
}

func (a ScriptActor) SetTempData(key string, value any) {

	if a.userRecord != nil {
		if userValue, ok := value.(ScriptActor); ok { // Don't store pointer to user data.
			userValue.userRecord = nil
			value = userValue
		}
		a.userRecord.SetTempData(key, value)
		return
	}

	if a.mobRecord != nil {
		if userValue, ok := value.(ScriptActor); ok { // Don't store pointer to user data.
			userValue.mobRecord = nil
			value = userValue
		}
		a.mobRecord.SetTempData(key, value)
		return
	}
}

func (a ScriptActor) GetTempData(key string) any {

	if a.userRecord != nil {
		if value := a.userRecord.GetTempData(key); value != nil {
			if userValue, ok := value.(ScriptActor); ok { // If it was userdata we need to reload the whole thing in case the user isn't around anymore.
				value = GetActor(userValue.userId, 0)
			}
			return value
		}
	} else if a.mobRecord != nil {
		if value := a.mobRecord.GetTempData(key); value != nil {
			if mobValue, ok := value.(ScriptActor); ok { // If it was userdata we need to reload the whole thing in case the user isn't around anymore.
				value = GetActor(0, mobValue.mobInstanceId)
			}
			return value
		}
	}
	return nil
}

func (a ScriptActor) GetTameMastery() map[int]int {
	return a.characterRecord.MobMastery.GetAllTame()
}

func (a ScriptActor) SetTameMastery(mobId int, newSkillLevel int) {
	a.characterRecord.MobMastery.SetTame(mobId, newSkillLevel)
}

func (a ScriptActor) GetChanceToTame(target ScriptActor) int {
	return combat.ChanceToTame(a.userRecord, target.mobRecord)
}

func (a ScriptActor) SetMiscCharacterData(key string, value any) {

	if _, ok := value.(ScriptActor); ok { // Don't store actor data.
		return
	}
	a.characterRecord.SetMiscData(key, value)
}

func (a ScriptActor) GetMiscCharacterData(key string) any {
	if value := a.characterRecord.GetMiscData(key); value != nil {
		return value
	}
	return nil
}

func (a ScriptActor) GetMiscCharacterDataKeys(prefixMatches ...string) []string {
	return a.characterRecord.GetMiscDataKeys(prefixMatches...)
}

func (a ScriptActor) GetCharacterName(wrapInTags bool) string {

	if wrapInTags {
		if a.userRecord != nil {
			return `<ansi fg="username">` + a.characterRecord.Name + `</ansi>`
		} else if a.mobRecord != nil {
			return `<ansi fg="mobname">` + a.characterRecord.Name + `</ansi>`
		}
	}

	return a.characterRecord.Name
}

func (a ScriptActor) SetCharacterName(newName string) {
	a.characterRecord.Name = newName
}

func (a ScriptActor) GetRoomId() int {
	return a.characterRecord.RoomId
}

func (a ScriptActor) HasQuest(questId string) bool {
	return a.characterRecord.HasQuest(questId)
}

func (a ScriptActor) GiveQuest(questId string) {

	events.AddToQueue(events.Quest{
		UserId:     a.UserId(),
		QuestToken: questId,
	})

}

// Gets a party object that indexes ALL members of the party
func (a ScriptActor) GetParty(excludeSelf ...bool) ScriptParty {
	return ScriptParty{
		actor:          &a,
		includePresent: true,
		includeMissing: true,
		includeSelf:    len(excludeSelf) == 0 || !excludeSelf[0],
	}
}

// Gets a party object that indexes only party members in the same room
func (a ScriptActor) GetPartyPresent(excludeSelf ...bool) ScriptParty {
	return ScriptParty{
		actor:          &a,
		includePresent: true,
		includeMissing: false,
		includeSelf:    len(excludeSelf) == 0 || !excludeSelf[0],
	}
}

// Gets a party object that indexes only members NOT in the same room.
func (a ScriptActor) GetPartyMissing() ScriptParty {
	return ScriptParty{
		actor:          &a,
		includePresent: false,
		includeMissing: true,
		includeSelf:    false,
	}
}

func (a ScriptActor) AddGold(amt int, bankAmt ...int) {
	a.characterRecord.Gold += amt
	if a.characterRecord.Gold < 0 {
		a.characterRecord.Gold = 0
	}
	if len(bankAmt) > 0 {
		a.characterRecord.Bank += bankAmt[0]
		if a.characterRecord.Bank < 0 {
			a.characterRecord.Bank = 0
		}
	}
}

func (a ScriptActor) AddHealth(amt int) int {
	ret := a.characterRecord.ApplyHealthChange(amt)

	if ret != 0 && a.userId > 0 {
		events.AddToQueue(events.CharacterVitalsChanged{UserId: a.userId})
	}

	return ret
}

func (a ScriptActor) AddMana(amt int) int {
	ret := a.characterRecord.ApplyManaChange(amt)

	if ret != 0 && a.userId > 0 {
		events.AddToQueue(events.CharacterVitalsChanged{UserId: a.userId})
	}

	return ret
}

func (a ScriptActor) Sleep(seconds int) {
	if a.userId == 0 {
		a.mobRecord.Sleep(seconds)
	}
}

func (a ScriptActor) Command(cmd string, waitSeconds ...float64) {
	if len(waitSeconds) < 1 {
		waitSeconds = append(waitSeconds, 0)
	}
	if a.userId > 0 {
		a.userRecord.Command(cmd, waitSeconds[0])
	} else {
		a.mobRecord.Command(cmd, waitSeconds[0])
	}
}

func (a ScriptActor) CommandFlagged(cmd string, flags events.EventFlag, waitSeconds ...float64) {
	if len(waitSeconds) < 1 {
		waitSeconds = append(waitSeconds, 0)
	}
	if a.userId > 0 {
		a.userRecord.CommandFlagged(cmd, flags, waitSeconds[0])
	} else {
		a.mobRecord.Command(cmd, waitSeconds[0])
	}
}

func (a ScriptActor) TrainSkill(skillName string, skillLevel int) bool {

	if a.userRecord == nil {
		return false
	}

	skillName = strings.ToLower(skillName)
	currentLevel := a.characterRecord.GetSkillLevel(skills.SkillTag(skillName))

	if currentLevel < skillLevel {
		newLevel := a.characterRecord.TrainSkill(skillName, skillLevel)

		skillData := struct {
			SkillName  string
			SkillLevel int
		}{
			SkillName:  skillName,
			SkillLevel: newLevel,
		}
		skillUpTxt, _ := templates.Process("character/skillup", skillData, a.userRecord.UserId)
		a.SendText(skillUpTxt)

		return true

	}
	return false
}

func (a ScriptActor) GetSkillLevel(skillName string) int {
	return a.characterRecord.GetSkillLevel(skills.SkillTag(skillName))
}

func (a ScriptActor) MoveRoom(destRoomId int) {

	if a.userRecord != nil {

		rmNow := rooms.LoadRoom(a.characterRecord.RoomId)

		if rmNext := rooms.LoadRoom(destRoomId); rmNext != nil {

			rooms.MoveToRoom(a.userId, destRoomId)

			for _, mobInstId := range a.characterRecord.GetCharmIds() {
				rmNow.RemoveMob(mobInstId)
				rmNext.AddMob(mobInstId)
			}

			if doLook, err := TryRoomScriptEvent(`onEnter`, a.userRecord.UserId, destRoomId); err != nil || doLook {
				a.userRecord.CommandFlagged(`look`, events.CmdSecretly) // Do a secret look.
			}
		}

	} else if a.mobRecord != nil {

		if mobRoom := rooms.LoadRoom(a.characterRecord.RoomId); mobRoom != nil {
			if destRoom := rooms.LoadRoom(destRoomId); destRoom != nil {
				mobRoom.RemoveMob(a.mobInstanceId)
				destRoom.AddMob(a.mobInstanceId)
			}
		}

	}
}

func (a ScriptActor) UpdateItem(itm ScriptItem) {
	a.userRecord.Character.UpdateItem(itm.originalItem, *itm.itemRecord)
}

func (a ScriptActor) AddEventLog(category string, message string) {
	if a.userRecord != nil {
		a.userRecord.EventLog.Add(category, message)
	}
}

// MarkVisitedRoom marks one or more rooms as visited for this actor.
// Only applies to user actors; mobs do not track room visits.
// Each roomId argument is recorded under the zone that room belongs to.
func (a ScriptActor) MarkVisitedRoom(roomIds ...int) {
	if a.userRecord == nil {
		return
	}
	for _, roomId := range roomIds {
		room := rooms.LoadRoom(roomId)
		if room == nil {
			continue
		}
		zCfg := rooms.GetZoneConfig(room.Zone)
		var validRoomIds map[int]struct{}
		if zCfg != nil {
			validRoomIds = zCfg.RoomIds
		}
		a.characterRecord.MarkVisitedRoom(roomId, room.Zone, validRoomIds)
	}
}

// MarkVisitedZone marks every room in the named zone as visited for this actor.
// Only applies to user actors; mobs do not track room visits.
// Uses FindZoneName for partial/best-match zone name resolution.
func (a ScriptActor) MarkVisitedZone(zoneName string) {
	if a.userRecord == nil {
		return
	}
	resolvedZone := rooms.FindZoneName(zoneName)
	if resolvedZone == `` {
		return
	}
	zCfg := rooms.GetZoneConfig(resolvedZone)
	if zCfg == nil {
		return
	}
	for roomId := range zCfg.RoomIds {
		a.characterRecord.MarkVisitedRoom(roomId, resolvedZone, nil)
	}
}

func (a ScriptActor) GiveItem(itm any) {

	var sItem *ScriptItem

	if itmScriptItem, ok := itm.(*ScriptItem); ok {
		sItem = itmScriptItem
	} else if itmInt, ok := itm.(int); ok {
		sItem = CreateItem(itmInt)
	} else if itmInt64, ok := itm.(int64); ok {
		sItem = CreateItem(int(itmInt64))
	} else if itmInt32, ok := itm.(int32); ok {
		sItem = CreateItem(int(itmInt32))
	} else if itmFloat64, ok := itm.(float64); ok {
		sItem = CreateItem(int(itmFloat64))
	}

	if sItem != nil {
		iRecord := sItem.itemRecord
		if a.characterRecord.StoreItem(*iRecord) {
			if a.userId > 0 {

				events.AddToQueue(events.ItemOwnership{
					UserId: a.userId,
					Item:   *iRecord,
					Gained: true,
				})

			}
		}
	}

}

func (a ScriptActor) TakeItem(itm ScriptItem) {
	if a.characterRecord.RemoveItem(*itm.itemRecord) {
		if a.userId > 0 {

			events.AddToQueue(events.ItemOwnership{
				UserId: a.userId,
				Item:   *itm.itemRecord,
				Gained: false,
			})

		}
	}
}

func (a ScriptActor) IsTameable() bool {
	if a.mobRecord == nil {
		return false
	}
	return a.mobRecord.IsTameable()
}

func (a ScriptActor) HasBuff(buffId int) bool {
	return a.characterRecord.HasBuff(buffId)
}

func (a ScriptActor) GiveBuff(buffId int, source string) {

	events.AddToQueue(events.Buff{
		UserId:        a.userId,
		MobInstanceId: a.mobInstanceId,
		BuffId:        buffId,
		Source:        source,
	})

}

func (a ScriptActor) GetStatMod(statModName string) int {
	return a.characterRecord.StatMod(statModName)
}

func (a ScriptActor) HasBuffFlag(buffFlag string) bool {
	return a.characterRecord.HasBuffFlag(buffs.Flag(buffFlag))
}

func (a ScriptActor) CancelBuffWithFlag(buffFlag string) bool {

	found := false

	for _, buffId := range a.characterRecord.Buffs.GetBuffIdsWithFlag(buffs.Flag(strings.ToLower(buffFlag))) {
		found = found || a.RemoveBuff(buffId)
	}

	return found
}

// Remove a buff silently
func (a ScriptActor) RemoveBuff(buffId int) bool {

	if !configs.GetGamePlayConfig().AllowItemBuffRemoval {
		buffList := a.characterRecord.GetBuffs(buffId)
		if len(buffList) > 0 {
			if buffList[0].PermaBuff {
				return false
			}
		}
	}

	return a.characterRecord.Buffs.RemoveBuff(buffId)

}

func (a ScriptActor) HasItemId(itemId int, excludeWorn ...bool) bool {
	for _, itm := range a.characterRecord.GetAllBackpackItems() {
		if itm.ItemId == itemId {
			return true
		}
	}
	if len(excludeWorn) == 0 || !excludeWorn[0] {
		for _, itm := range a.characterRecord.GetAllWornItems() {
			if itm.ItemId == itemId {
				return true
			}
		}
	}
	return false
}

func (a ScriptActor) GetBackpackItems() []ScriptItem {
	itms := make([]ScriptItem, 0, 5)
	for _, item := range a.characterRecord.GetAllBackpackItems() {
		itms = append(itms, newScriptItem(item))
	}
	return itms
}

func (a ScriptActor) GetAlignment() int {
	return int(a.characterRecord.Alignment)
}

func (a ScriptActor) GetAlignmentName() string {
	return a.characterRecord.AlignmentName()
}

func (a ScriptActor) ChangeAlignment(alignmentChange int) {
	a.characterRecord.UpdateAlignment(alignmentChange)
}

func (a ScriptActor) HasSpell(spellId string) bool {
	return a.characterRecord.HasSpell(spellId)
}

func (a ScriptActor) LearnSpell(spellId string) bool {
	return a.characterRecord.LearnSpell(spellId)
}

func (a ScriptActor) IsAggro(actor ScriptActor) bool {
	return a.characterRecord.IsAggro(actor.UserId(), actor.InstanceId())
}

func (a ScriptActor) GetMobKills(mobId int) int {
	return a.characterRecord.KD.GetMobKills(mobId)
}

func (a ScriptActor) GetRaceKills(race string) int {

	raceKills := map[string]int{}

	for mid, kCt := range a.characterRecord.KD.Kills {
		if mobSpec := mobs.GetMobSpec(mobs.MobId(mid)); mobSpec != nil {
			if raceInfo := races.GetRace(mobSpec.Character.RaceId); raceInfo != nil {
				raceKills[raceInfo.Name] = raceKills[raceInfo.Name] + kCt
			}
		}
	}

	return raceKills[race]
}

func (a ScriptActor) SetHealth(amt int) {
	a.characterRecord.Health = amt
	if a.characterRecord.Health > a.characterRecord.HealthMax.Value {
		a.characterRecord.Health = a.characterRecord.HealthMax.Value
	}
}

func (a ScriptActor) GetHealth() int {
	return a.characterRecord.Health
}

func (a ScriptActor) GetHealthMax() int {
	return a.characterRecord.HealthMax.Value
}

func (a ScriptActor) GetHealthPct() float64 {
	return float64(a.characterRecord.Health) / float64(a.characterRecord.HealthMax.Value)
}

func (a ScriptActor) GetMana() int {
	return a.characterRecord.Mana
}

func (a ScriptActor) GetManaMax() int {
	return a.characterRecord.ManaMax.Value
}

func (a ScriptActor) GetManaPct() float64 {
	return float64(a.characterRecord.Mana) / float64(a.characterRecord.ManaMax.Value)
}

func (a ScriptActor) SetAdjective(adj string, addIt bool) {
	a.characterRecord.SetAdjective(adj, addIt)
}

func (a ScriptActor) IsHome() bool {
	if a.mobRecord != nil {
		return a.mobRecord.HomeRoomId == a.characterRecord.RoomId
	}
	return false
}

func (a ScriptActor) GetCharmCount() int {
	return len(a.characterRecord.GetCharmIds())
}

func (a ScriptActor) GetMaxCharmCount() int {
	return a.characterRecord.GetMaxCharmedCreatures()
}

func (a ScriptActor) GetTrainingPoints() int {
	return a.characterRecord.TrainingPoints
}

func (a ScriptActor) GiveTrainingPoints(ct int) {
	if ct < 1 {
		return
	}
	a.characterRecord.TrainingPoints += ct
}

func (a ScriptActor) GetStatPoints() int {
	return a.characterRecord.StatPoints
}

func (a ScriptActor) GiveStatPoints(ct int) {
	if ct < 1 {
		return
	}
	a.characterRecord.StatPoints += ct
}

func (a ScriptActor) GiveExtraLife() {
	c := configs.GetGamePlayConfig()
	a.characterRecord.ExtraLives += 1
	if a.characterRecord.ExtraLives > int(c.LivesMax) {
		a.characterRecord.ExtraLives = int(c.LivesMax)
	}
}

func (a ScriptActor) Uncurse() []*ScriptItem {

	retList := []*ScriptItem{}

	for _, itm := range a.characterRecord.Uncurse() {
		retList = append(retList, GetItem(itm))
	}

	return retList
}

func (a ScriptActor) GetPet() *ScriptPet {
	if !a.characterRecord.Pet.Exists() {
		return nil
	}
	return GetPet(&a.characterRecord.Pet, a.userId)
}

func (a ScriptActor) GrantXP(xpAmt int, reason string) {
	if a.mobInstanceId > 0 {
		return
	}
	a.userRecord.GrantXP(xpAmt, reason)
}

func (a ScriptActor) TimerSet(name string, period string) {
	a.characterRecord.TimerSet(name, period)
}

func (a ScriptActor) TimerExpired(name string) bool {
	return a.characterRecord.TimerExpired(name)
}

func (a ScriptActor) TimerExists(name string) bool {
	return a.characterRecord.TimerExists(name)
}

// ////////////////////////////////////////////////////////
//
// Functions only really useful for mobs
//
// ////////////////////////////////////////////////////////

// Returns true if a mob is charmed by/friendly to a player.
// If userId is ommitted, it will return true if the mob is charmed by any player.
func (a ScriptActor) IsCharmed(userId ...int) bool {
	if len(userId) < 1 {
		return a.characterRecord.IsCharmed()
	}
	return a.characterRecord.IsCharmed(userId[0])
}

func (a ScriptActor) GetCharmedUserId() int {
	return a.characterRecord.GetCharmedUserId()
}

func (a ScriptActor) CharmSet(userId int, charmRounds int, onRevertCommand ...string) {

	// If the player is in a party, add the mob to their party
	if a.mobInstanceId < 1 {
		return
	}

	if len(onRevertCommand) < 1 {
		onRevertCommand = append(onRevertCommand, ``)
	}
	a.characterRecord.Charm(userId, charmRounds, onRevertCommand[0])

	if user := users.GetByUserId(userId); user != nil {
		user.Character.TrackCharmed(a.mobInstanceId, true)
	}

}

func (a ScriptActor) CharmRemove() {
	if a.characterRecord.Charmed == nil {
		return
	}
	charmUserId := a.characterRecord.RemoveCharm()

	if user := users.GetByUserId(charmUserId); user != nil {
		user.Character.TrackCharmed(a.mobInstanceId, false)
	}
}

func (a ScriptActor) CharmExpire() {
	a.characterRecord.Charmed.Expire()
}

func (a ScriptActor) getScript() string {
	if a.mobRecord != nil {
		return a.mobRecord.GetScript()
	}
	return ""
}

func (a ScriptActor) getScriptTag() string {
	if a.mobRecord != nil {
		return a.mobRecord.ScriptTag
	}
	return ""
}

func (a ScriptActor) ShorthandId() string {

	if a.userRecord != nil {

		return a.userRecord.ShorthandId()

	} else if a.mobRecord != nil {

		return a.mobRecord.ShorthandId()

	}

	return ``
}

func (a ScriptActor) GetLastInputRound() uint64 {
	if a.userRecord != nil {
		return a.userRecord.GetLastInputRound()
	}
	return 0
}

// PlaySound sends a sound event to this actor. Only affects user actors.
// soundId is the identifier defined in audio.yaml (e.g. "levelup").
// category groups related sounds for client-side volume control (e.g. "combat", "other").
func (a ScriptActor) PlaySound(soundId string, category string) {
	if a.userRecord == nil {
		return
	}
	a.userRecord.PlaySound(soundId, category)
}

// PlayMusic sends a music-change event to this actor. Only affects user actors.
// musicFileOrId may be a filepath (e.g. "static/audio/music/intro.mp3") or a sound
// identifier from audio.yaml whose filepath resolves to a music file.
// Pass "Off" to stop music.
func (a ScriptActor) PlayMusic(musicFileOrId string) {
	if a.userRecord == nil {
		return
	}
	a.userRecord.PlayMusic(musicFileOrId)
}

func (a ScriptActor) Pathing() bool {
	if a.mobRecord == nil {
		return false
	}
	return a.mobRecord.Path.Current() != nil || a.mobRecord.Path.Len() > 0
}

func (a ScriptActor) PathingAtWaypoint() bool {
	if a.mobRecord == nil {
		return false
	}
	pathStep := a.mobRecord.Path.Current()
	return pathStep != nil && pathStep.Waypoint()
}

// ////////////////////////////////////////////////////////
//
// # These functions get exported to the scripting engine
//
// ////////////////////////////////////////////////////////
func GetActor(userId int, mobInstanceId int) *ScriptActor {

	if userId > 0 {
		if user := users.GetByUserId(userId); user != nil {
			return &ScriptActor{
				userId:          userId,
				userRecord:      user,
				characterRecord: user.Character,
			}
		}
	} else if mobInstanceId > 0 {
		if mob := mobs.GetInstance(mobInstanceId); mob != nil {
			return &ScriptActor{
				mobInstanceId:   mobInstanceId,
				mobRecord:       mob,
				characterRecord: &mob.Character,
			}
		}
	}

	return nil
}

func GetUser(userId int) *ScriptActor {
	return GetActor(userId, 0)
}

func GetMob(mobInstanceId int) *ScriptActor {
	return GetActor(0, mobInstanceId)
}

func ActorNames(actorList []*ScriptActor) string {

	sBuilder := strings.Builder{}
	listSize := len(actorList)

	for i := 0; i < listSize; i++ {

		sBuilder.WriteString(actorList[i].GetCharacterName(true))

		if i < listSize-2 {
			sBuilder.WriteString(`, `)
		} else if i == listSize-2 {
			sBuilder.WriteString(`and `)
		}
	}

	return sBuilder.String()
}

// ////////////////////////////////////////////////////////
//
// New ScriptActor methods
//
// ////////////////////////////////////////////////////////

func (a ScriptActor) GetDescription() string {
	return a.characterRecord.GetDescription()
}

func (a ScriptActor) GetGold() int {
	return a.characterRecord.Gold
}

func (a ScriptActor) GetBank() int {
	return a.characterRecord.Bank
}

func (a ScriptActor) GetWornItems() []ScriptItem {
	itms := make([]ScriptItem, 0)
	for _, itm := range a.characterRecord.GetAllWornItems() {
		itms = append(itms, newScriptItem(itm))
	}
	return itms
}

func (a ScriptActor) GetWornItem(slot string) *ScriptItem {
	itm := a.characterRecord.Equipment.Get(items.ItemType(slot))
	if itm == nil || itm.ItemId <= 0 || itm.IsDisabled() {
		return nil
	}
	si := newScriptItem(*itm)
	return &si
}

func (a ScriptActor) FindInBackpack(itemName string) *ScriptItem {
	itm, found := a.characterRecord.FindInBackpack(itemName)
	if !found {
		return nil
	}
	si := newScriptItem(itm)
	return &si
}

func (a ScriptActor) FindOnBody(itemName string) *ScriptItem {
	itm, found := a.characterRecord.FindOnBody(itemName)
	if !found {
		return nil
	}
	si := newScriptItem(itm)
	return &si
}

func (a ScriptActor) IsQuestDone(questToken string) bool {
	return a.characterRecord.IsQuestDone(questToken)
}

func (a ScriptActor) ClearQuestToken(questToken string) {
	a.characterRecord.ClearQuestToken(questToken)
}

func (a ScriptActor) GetAllSkills() map[string]int {
	return a.characterRecord.GetAllSkillRanks()
}

func (a ScriptActor) GetSpells() map[string]int {
	return a.characterRecord.GetSpells()
}

func (a ScriptActor) UnLearnSpell(spellId string) bool {
	return a.characterRecord.UnLearnSpell(spellId)
}

func (a ScriptActor) DisableSpell(spellId string) bool {
	return a.characterRecord.DisableSpell(spellId)
}

func (a ScriptActor) EnableSpell(spellId string) bool {
	return a.characterRecord.EnableSpell(spellId)
}

func (a ScriptActor) GetCooldown(tag string) int {
	return a.characterRecord.GetCooldown(tag)
}

func (a ScriptActor) TryCooldown(tag string, period string) bool {
	return a.characterRecord.TryCooldown(tag, period)
}

func (a ScriptActor) GetSetting(name string) string {
	return a.characterRecord.GetSetting(name)
}

func (a ScriptActor) SetSetting(name string, value string) {
	a.characterRecord.SetSetting(name, value)
}

func (a ScriptActor) GetExperience() int {
	return a.characterRecord.Experience
}

func (a ScriptActor) GetExtraLives() int {
	return a.characterRecord.ExtraLives
}

func (a ScriptActor) IsInCombat() bool {
	return a.characterRecord.Aggro != nil
}

func (a ScriptActor) IsDowned() bool {
	return a.characterRecord.Health < 1
}

func (a ScriptActor) GetDefense() int {
	return a.characterRecord.GetDefense()
}

func (a ScriptActor) GetGearValue() int {
	return a.characterRecord.GetGearValue()
}

func (a ScriptActor) GetCarryCapacity() int {
	return a.characterRecord.CarryCapacity()
}

func (a ScriptActor) GetZoneVisitProgress(zoneName string) map[string]any {
	resolvedZone := rooms.FindZoneName(zoneName)
	if resolvedZone == `` {
		return map[string]any{`visited`: 0, `total`: 0, `percent`: 0}
	}
	zCfg := rooms.GetZoneConfig(resolvedZone)
	var validRoomIds map[int]struct{}
	if zCfg != nil {
		validRoomIds = zCfg.RoomIds
	}
	visited, total := a.characterRecord.ZoneVisitProgress(resolvedZone, validRoomIds)
	percent := 0
	if total > 0 {
		percent = int(float64(visited) / float64(total) * 100)
	}
	return map[string]any{`visited`: visited, `total`: total, `percent`: percent}
}

func (a ScriptActor) GetAdjectives() []string {
	return a.characterRecord.GetAdjectives()
}

func (a ScriptActor) HasAdjective(adj string) bool {
	return a.characterRecord.HasAdjective(adj)
}

func (a ScriptActor) GetActionPoints() int {
	return a.characterRecord.ActionPoints
}

func (a ScriptActor) GetActionPointsMax() int {
	return a.characterRecord.ActionPointsMax.Value
}

func (a ScriptActor) GetHealthAppearance() string {
	if a.characterRecord.HealthMax.Value < 1 {
		return a.GetCharacterName(true) + ` is in perfect health.`
	}
	nameTag := `username`
	if a.mobRecord != nil {
		nameTag = `mobname`
	}
	pct := int(float64(a.characterRecord.Health) / float64(a.characterRecord.HealthMax.Value) * 100)
	className := util.HealthClass(a.characterRecord.Health, a.characterRecord.HealthMax.Value)
	name := `<ansi fg="` + nameTag + `">` + a.characterRecord.Name + `</ansi>`
	switch {
	case pct < 15:
		return name + ` looks like they're <ansi fg="` + className + `">about to die!</ansi>`
	case pct < 50:
		return name + ` looks to be in <ansi fg="` + className + `">pretty bad shape.</ansi>`
	case pct < 80:
		return name + ` has some <ansi fg="` + className + `">cuts and bruises.</ansi>`
	case pct < 100:
		return name + ` has <ansi fg="` + className + `">a few scratches.</ansi>`
	}
	return name + ` is in <ansi fg="` + className + `">perfect health.</ansi>`
}
