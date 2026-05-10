package telemetry

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// RegisterListeners wires all telemetry event listeners. Called once at startup.
func RegisterListeners() {
	events.RegisterListener(events.MobItemDrop{}, onMobItemDrop)
	events.RegisterListener(events.ItemOwnership{}, onItemOwnership)
	events.RegisterListener(events.MobDeath{}, onMobDeath)
	events.RegisterListener(events.PlayerDeath{}, onPlayerDeath)
	events.RegisterListener(events.Purchase{}, onPurchase)
}

func onMobItemDrop(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.MobItemDrop)
	if !ok {
		return events.Continue
	}
	Track(CatItemDrop, evt.Zone, evt.ItemId, evt.MobId, evt.RoomId)
	return events.Continue
}

func onItemOwnership(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.ItemOwnership)
	if !ok {
		return events.Continue
	}
	if !evt.Gained {
		return events.Continue
	}
	if evt.UserId == 0 {
		return events.Continue
	}

	zone := ""
	roomId := 0
	if user := users.GetByUserId(evt.UserId); user != nil {
		roomId = user.Character.RoomId
		if r := rooms.LoadRoom(roomId); r != nil {
			zone = r.Zone
		}
	}

	Track(CatItemPickup, zone, evt.Item.ItemId, 0, roomId)
	return events.Continue
}

func onMobDeath(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.MobDeath)
	if !ok {
		return events.Continue
	}

	zone := ""
	if r := rooms.LoadRoom(evt.RoomId); r != nil {
		zone = r.Zone
	}

	Track(CatMobKill, zone, 0, evt.MobId, evt.RoomId)
	return events.Continue
}

func onPlayerDeath(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.PlayerDeath)
	if !ok {
		return events.Continue
	}

	zone := ""
	if r := rooms.LoadRoom(evt.RoomId); r != nil {
		zone = r.Zone
	}

	Track(CatPlayerDeath, zone, 0, evt.KillerMobId, evt.RoomId)
	return events.Continue
}

func onPurchase(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.Purchase)
	if !ok {
		return events.Continue
	}
	if evt.ItemId == 0 {
		return events.Continue
	}

	Track(CatItemPurchase, "", evt.ItemId, evt.SellerMobId, 0)
	return events.Continue
}
