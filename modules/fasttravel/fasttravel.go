package fasttravel

import (
	"embed"
	"fmt"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/term"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

var (
	//go:embed files/*
	files embed.FS
)

const (
	fastTravelTag = "fast travel"
	dataKeyFmt    = "fasttravel-user-%d"
)

func init() {
	m := &FastTravelModule{
		plug:    plugins.New(`fasttravel`, `1.0`),
		players: make(map[int]FastTravelData),
	}

	if err := m.plug.AttachFileSystem(files); err != nil {
		panic(err)
	}

	m.plug.AddUserCommand(`fasttravel`, m.fastTravelCommand, false, false)

	m.plug.Web.AdminPage("Config", "fasttravel-config", "html/admin/fasttravel-config.html", true, "Modules", "Fast Travel", nil)

	m.plug.ReserveTags(fastTravelTag)

	m.plug.Callbacks.SetOnSave(m.onSave)

	events.RegisterListener(events.PlayerSpawn{}, m.onPlayerSpawn)
	events.RegisterListener(events.PlayerDespawn{}, m.onPlayerDespawn)
	events.RegisterListener(events.Input{}, m.onLookInput, events.First)

	rooms.OnRoomLook.Register(m.onRoomLook)
}

// FastTravelData holds the unlocked fast travel room IDs for a single user.
type FastTravelData struct {
	UnlockedRoomIds []int `yaml:"unlockedroomids,omitempty"`
}

// FastTravelModule owns all fast travel state.
type FastTravelModule struct {
	plug    *plugins.Plugin
	players map[int]FastTravelData // keyed by userId; loaded on PlayerSpawn
}

func dataKey(userId int) string {
	return fmt.Sprintf(dataKeyFmt, userId)
}

func (m *FastTravelModule) load(userId int) FastTravelData {
	var data FastTravelData
	m.plug.ReadIntoStruct(dataKey(userId), &data)
	return data
}

func (m *FastTravelModule) save(userId int, data FastTravelData) {
	m.plug.WriteStruct(dataKey(userId), data)
}

func (m *FastTravelModule) onSave() {
	for userId, data := range m.players {
		m.save(userId, data)
	}
}

func (m *FastTravelModule) onPlayerSpawn(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.PlayerSpawn)
	if !ok {
		return events.Continue
	}
	m.players[evt.UserId] = m.load(evt.UserId)
	return events.Continue
}

func (m *FastTravelModule) onPlayerDespawn(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.PlayerDespawn)
	if !ok {
		return events.Continue
	}
	if data, exists := m.players[evt.UserId]; exists {
		m.save(evt.UserId, data)
		delete(m.players, evt.UserId)
	}
	return events.Continue
}

// onRoomLook injects a fast travel alert when the room has the fast travel tag.
func (m *FastTravelModule) onRoomLook(d rooms.RoomTemplateDetails) rooms.RoomTemplateDetails {
	for _, t := range d.Tags {
		if strings.EqualFold(t, fastTravelTag) {
			d.RoomAlerts = append(d.RoomAlerts,
				`<ansi fg="yellow-bold">This is a fast travel station!</ansi>`+
					"\n"+
					`      <ansi fg="command">fasttravel</ansi> lists destinations, or <ansi fg="command">look fast travel</ansi> to examine it.`,
			)
			return d
		}
	}
	return d
}

// parseLookTarget extracts the target from a look command, stripping "at " and "the " prefixes.
// Returns an empty string if the input is not a look command or has no target.
func parseLookTarget(inputText string) string {
	lower := strings.ToLower(strings.TrimSpace(inputText))
	cmd, rest, _ := strings.Cut(lower, " ")
	if cmd != `look` && cmd != `l` {
		return ""
	}
	rest = strings.TrimSpace(rest)
	if strings.HasPrefix(rest, `at `) {
		rest = rest[3:]
	}
	if strings.HasPrefix(rest, `the `) {
		rest = rest[4:]
	}
	return strings.TrimSpace(rest)
}

// onLookInput intercepts look commands directed at the fast travel station.
// It sends a description and unlocks the station for the player, then cancels
// the event so the engine does not attempt to resolve it as a room object.
func (m *FastTravelModule) onLookInput(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.Input)
	if !ok || evt.UserId == 0 {
		return events.Continue
	}

	target := parseLookTarget(evt.InputText)
	if target == "" {
		return events.Continue
	}

	_, match := util.FindMatchIn(target, `fast travel`, `fast travel station`, `fasttravel`)
	if match == "" {
		return events.Continue
	}

	user := users.GetByUserId(evt.UserId)
	if user == nil {
		return events.Continue
	}

	room := rooms.LoadRoom(user.Character.RoomId)
	if room == nil || !roomIsFastTravel(room) {
		return events.Continue
	}

	// Unlock the station for the player when they look at it.
	if m.unlock(user.UserId, room.RoomId) {
		user.SendText(`<ansi fg="yellow-bold">You have discovered a new fast travel station: </ansi><ansi fg="room-title">` + room.Title + `</ansi><ansi fg="yellow-bold">!</ansi>` + term.CRLFStr)
	}

	m.sendFastTravelLookDescription(user, room)
	return events.Cancel
}

// sendFastTravelLookDescription sends the formatted look description for the fast travel station.
func (m *FastTravelModule) sendFastTravelLookDescription(user *users.UserRecord, room *rooms.Room) {
	goldCost, requiredItemId := m.travelCost()

	var costLine string
	if goldCost > 0 {
		costLine = fmt.Sprintf(`Each journey costs <ansi fg="gold">%d gold</ansi>.`, goldCost)
	} else {
		costLine = `Travel through this network is free of charge.`
	}

	lines := []string{
		`A shimmering portal of woven light stands here, humming with arcane energy.`,
		`Runes etched into the surrounding stonework pulse in a slow, rhythmic pattern,`,
		`connecting this station to others scattered across the world.`,
		``,
		costLine,
	}

	if requiredItemId > 0 {
		itemName := fmt.Sprintf(`item #%d`, requiredItemId)
		if iSpec := items.GetItemSpec(requiredItemId); iSpec != nil {
			itemName = iSpec.Name
		}
		lines = append(lines, fmt.Sprintf(`A <ansi fg="itemname">%s</ansi> is consumed with each use.`, itemName))
	}

	disallowTypes := m.disallowedTypes()
	disallowSubtypes := m.disallowedSubtypes()
	if len(disallowTypes) > 0 || len(disallowSubtypes) > 0 {
		all := make([]string, 0, len(disallowTypes)+len(disallowSubtypes))
		for t := range disallowTypes {
			all = append(all, t)
		}
		for s := range disallowSubtypes {
			all = append(all, s)
		}
		sort.Strings(all)
		tagged := make([]string, len(all))
		for i, v := range all {
			tagged[i] = `<ansi fg="itemname">` + v + `</ansi>`
		}
		lines = append(lines, fmt.Sprintf(`Items of type %s cannot be carried through.`, strings.Join(tagged, `, `)))
	}

	lines = append(lines,
		`Use <ansi fg="command">fasttravel</ansi> to list known destinations, or`,
		`<ansi fg="command">fasttravel <destination></ansi> to travel instantly.`,
	)

	desc := strings.Join(lines, "\n")

	user.SendText(``)
	user.SendText(fmt.Sprintf(`You look at the <ansi fg="noun">fast travel station</ansi>:`))
	user.SendText(``)
	for _, line := range strings.Split(desc, "\n") {
		user.SendText(line)
	}
	user.SendText(``)

	room.SendText(
		fmt.Sprintf(`<ansi fg="username">%s</ansi> is examining the <ansi fg="noun">fast travel station</ansi>.`, user.Character.Name),
		user.UserId,
	)
}

// roomIsFastTravel returns true if the room has the fast travel tag.
func roomIsFastTravel(room *rooms.Room) bool {
	return room.HasTag(fastTravelTag)
}

// hasUnlocked returns true if the user has unlocked the given room.
func (m *FastTravelModule) hasUnlocked(userId, roomId int) bool {
	data := m.players[userId]
	for _, id := range data.UnlockedRoomIds {
		if id == roomId {
			return true
		}
	}
	return false
}

// unlock adds the roomId to the user's unlocked list if not already present.
// Returns true if the room was newly unlocked, false if it was already known.
func (m *FastTravelModule) unlock(userId, roomId int) bool {
	if m.hasUnlocked(userId, roomId) {
		return false
	}
	data := m.players[userId]
	data.UnlockedRoomIds = append(data.UnlockedRoomIds, roomId)
	m.players[userId] = data
	return true
}

// travelCost reads the configured gold cost and required item ID from the module config.
// A cost of 0 means no gold is required. An itemId of 0 means no item is required.
func (m *FastTravelModule) travelCost() (goldCost int, itemId int) {
	if v, ok := m.plug.Config.Get(`GoldCost`).(int); ok && v > 0 {
		goldCost = v
	}
	if v, ok := m.plug.Config.Get(`RequiredItemId`).(int); ok && v > 0 {
		itemId = v
	}
	return goldCost, itemId
}

// disallowedTypes returns the set of item types that block fast travel.
// The config value is a comma-separated string, e.g. "weapon,grenade".
func (m *FastTravelModule) disallowedTypes() map[string]struct{} {
	return parseStringSet(m.plug.Config.Get(`DisallowedItemTypes`))
}

// disallowedSubtypes returns the set of item subtypes that block fast travel.
// The config value is a comma-separated string, e.g. "throwable,explosive".
func (m *FastTravelModule) disallowedSubtypes() map[string]struct{} {
	return parseStringSet(m.plug.Config.Get(`DisallowedItemSubtypes`))
}

// parseStringSet converts a config value to a lowercase set of strings.
// Accepts a comma-separated string or a []interface{} of strings.
func parseStringSet(raw any) map[string]struct{} {
	result := map[string]struct{}{}
	switch v := raw.(type) {
	case string:
		for _, part := range strings.Split(v, ",") {
			s := strings.TrimSpace(strings.ToLower(part))
			if s != "" {
				result[s] = struct{}{}
			}
		}
	case []interface{}:
		for _, elem := range v {
			if s, ok := elem.(string); ok {
				s = strings.TrimSpace(strings.ToLower(s))
				if s != "" {
					result[s] = struct{}{}
				}
			}
		}
	}
	return result
}

// blockedByItem checks whether the player carries or wears any item that is
// disallowed by the current config. Returns the blocking item name, whether
// it was found on a pet, the pet's display name, and true if blocked.
func (m *FastTravelModule) blockedByItem(user *users.UserRecord) (itemName string, onPet bool, petName string, blocked bool) {
	disallowTypes := m.disallowedTypes()
	disallowSubtypes := m.disallowedSubtypes()

	if len(disallowTypes) == 0 && len(disallowSubtypes) == 0 {
		return "", false, "", false
	}

	checkItem := func(itm items.Item) (string, bool) {
		if itm.ItemId == 0 {
			return "", false
		}
		spec := itm.GetSpec()
		if _, b := disallowTypes[strings.ToLower(string(spec.Type))]; b {
			return spec.Name, true
		}
		if _, b := disallowSubtypes[strings.ToLower(string(spec.Subtype))]; b {
			return spec.Name, true
		}
		return "", false
	}

	for _, itm := range user.Character.GetAllBackpackItems() {
		if name, b := checkItem(itm); b {
			return name, false, "", true
		}
	}

	for _, itm := range user.Character.GetAllWornItems() {
		if name, b := checkItem(itm); b {
			return name, false, "", true
		}
	}

	if user.Character.Pet.Exists() {
		petDisplayName := user.Character.Pet.DisplayName()
		for _, itm := range user.Character.Pet.Items {
			if name, b := checkItem(itm); b {
				return name, true, petDisplayName, true
			}
		}
	}

	return "", false, "", false
}

// destEntry holds a resolved fast travel destination for display and matching.
type destEntry struct {
	roomId int
	title  string
}

func (m *FastTravelModule) fastTravelCommand(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if !roomIsFastTravel(room) {
		user.SendText(`You must be at a fast travel station to use fast travel.` + term.CRLFStr)
		return true, nil
	}

	user.SendText(``)

	// Always unlock the current room when the player uses the command here.
	if m.unlock(user.UserId, room.RoomId) {
		user.SendText(`<ansi fg="yellow-bold">You have discovered a new fast travel station: </ansi><ansi fg="room-title">` + room.Title + `</ansi><ansi fg="yellow-bold">!</ansi>` + term.CRLFStr)
	}

	// Build the list of unlocked destinations, excluding the current room.
	data := m.players[user.UserId]
	destinations := make([]destEntry, 0, len(data.UnlockedRoomIds))
	for _, id := range data.UnlockedRoomIds {
		if id == room.RoomId {
			continue
		}
		r := rooms.LoadRoom(id)
		if r == nil {
			continue
		}
		destinations = append(destinations, destEntry{roomId: id, title: r.Title})
	}
	sort.Slice(destinations, func(i, j int) bool {
		return destinations[i].title < destinations[j].title
	})

	// List mode.
	if rest == `` {
		if len(destinations) == 0 {
			user.SendText(`<ansi fg="yellow">You have not discovered any other fast travel stations yet.</ansi>` + term.CRLFStr)
			return true, nil
		}

		headers := []string{`#`, `Destination`}
		rows := make([][]string, len(destinations))
		for i, dest := range destinations {
			rows[i] = []string{fmt.Sprintf(`%d`, i+1), dest.title}
		}
		formatting := [][]string{
			{`<ansi fg="cyan">%s</ansi>`, `<ansi fg="room-title">%s</ansi>`},
		}
		tbl := templates.GetTable(`Fast Travel Destinations`, headers, rows, formatting...)
		tplTxt, _ := templates.Process("tables/generic", tbl, user.UserId)
		user.SendText(tplTxt)
		return true, nil
	}

	// Travel mode: match the argument against destination titles.
	titles := make([]string, len(destinations))
	for i, dest := range destinations {
		titles[i] = dest.title
	}

	closeMatch, exactMatch := util.FindMatchIn(rest, titles...)

	matchTitle := exactMatch
	if matchTitle == `` {
		matchTitle = closeMatch
	}
	if matchTitle == `` {
		user.SendText(fmt.Sprintf(`No fast travel destination matching "<ansi fg="cyan">%s</ansi>" found.`+term.CRLFStr, rest))
		return true, nil
	}

	var destRoomId int
	for _, dest := range destinations {
		if strings.EqualFold(dest.title, matchTitle) {
			destRoomId = dest.roomId
			break
		}
	}
	if destRoomId == 0 {
		user.SendText(fmt.Sprintf(`No fast travel destination matching "<ansi fg="cyan">%s</ansi>" found.`+term.CRLFStr, rest))
		return true, nil
	}

	destRoom := rooms.LoadRoom(destRoomId)
	if destRoom == nil {
		user.SendText(`That fast travel destination is no longer available.` + term.CRLFStr)
		return true, nil
	}

	// Check for items that block fast travel.
	if itemName, onPet, petName, blocked := m.blockedByItem(user); blocked {
		if onPet {
			user.SendText(fmt.Sprintf(`The <ansi fg="itemname">%s</ansi> %s carries prevents fast travelling!`+term.CRLFStr, itemName, petName))
		} else {
			user.SendText(fmt.Sprintf(`Your <ansi fg="itemname">%s</ansi> prevents fast travelling!`+term.CRLFStr, itemName))
		}
		return true, nil
	}

	// Enforce travel cost before moving.
	goldCost, requiredItemId := m.travelCost()

	if goldCost > 0 {
		if user.Character.Gold < goldCost {
			user.SendText(fmt.Sprintf(`You need at least <ansi fg="gold">%d gold</ansi> to use the fast travel network.`+term.CRLFStr, goldCost))
			return true, nil
		}
	}

	var requiredItem items.Item
	if requiredItemId > 0 {
		itm, found := user.Character.FindInBackpack(fmt.Sprintf(`!%d`, requiredItemId))
		if !found {
			iSpec := items.GetItemSpec(requiredItemId)
			itemName := fmt.Sprintf(`item #%d`, requiredItemId)
			if iSpec != nil {
				itemName = iSpec.Name
			}
			user.SendText(fmt.Sprintf(`You need a <ansi fg="itemname">%s</ansi> to use the fast travel network.`+term.CRLFStr, itemName))
			return true, nil
		}
		requiredItem = itm
	}

	// Deduct gold cost.
	if goldCost > 0 {
		user.Character.Gold -= goldCost
		events.AddToQueue(events.EquipmentChange{
			UserId:     user.UserId,
			GoldChange: -goldCost,
		})
	}

	// Consume required item.
	if requiredItem.ItemId > 0 {
		user.Character.RemoveItem(requiredItem)
		events.AddToQueue(events.ItemOwnership{
			UserId: user.UserId,
			Item:   requiredItem,
			Gained: false,
		})
	}

	user.SendText(fmt.Sprintf(`You step through the fast travel network and arrive at <ansi fg="room-title">%s</ansi>.`+term.CRLFStr, destRoom.Title))
	rooms.MoveToRoom(user.UserId, destRoomId)

	return true, nil
}
