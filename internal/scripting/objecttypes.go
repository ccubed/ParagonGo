package scripting

// ObjectMethod describes a single method on an engine object type, mirroring
// the TypeScript interface members declared for the JavaScript intellisense.
// It is the structured source the Lua editor intellisense consumes (the
// JavaScript editor uses the equivalent hand-authored .d.ts interfaces).
type ObjectMethod struct {
	Name        string            `json:"name"`
	Params      []ScriptFuncParam `json:"params"`
	ReturnType  string            `json:"returnType,omitempty"`
	Description string            `json:"description,omitempty"`
}

// ObjectTypeDef describes an engine object type (the runtime value bound to a
// script variable such as `mob`, `room`, `item`, `pet`) and the methods that
// can be called on it.
type ObjectTypeDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Methods     []ObjectMethod `json:"methods"`
}

// ScriptObjectTypes is the response served to the Lua editor. It carries the
// object type definitions plus a mapping of each script type's entrypoint
// parameter names to their object types, so the editor can seed variable types
// (e.g. for a mob script, `mob` is an ActorObject and `room` is a RoomObject).
type ScriptObjectTypes struct {
	Version int                      `json:"version"`
	Types   map[string]ObjectTypeDef `json:"types"`
}

// GetScriptObjectTypes returns the structured object-type model used to drive
// type-aware Lua intellisense. The method lists are kept in sync with the
// hand-authored TypeScript interfaces in the web package by a test
// (objecttypes_sync_test) that compares method names.
func GetScriptObjectTypes() *ScriptObjectTypes {
	return &ScriptObjectTypes{
		Version: 1,
		Types: map[string]ObjectTypeDef{
			"ActorObject":       actorObjectType(),
			"RoomObject":        roomObjectType(),
			"ItemObject":        itemObjectType(),
			"PetObject":         petObjectType(),
			"PartyObject":       partyObjectType(),
			"ContainerObject":   containerScriptObjectType(),
			"PanelLayoutObject": panelLayoutObjectType(),
			"SlotObject":        slotObjectType(),
			"PanelObject":       panelObjectType(),
		},
	}
}

func m(name, ret, desc string, params ...ScriptFuncParam) ObjectMethod {
	return ObjectMethod{Name: name, ReturnType: ret, Description: desc, Params: params}
}

func p(name, typ string) ScriptFuncParam {
	return ScriptFuncParam{Name: name, Type: typ}
}

func actorObjectType() ObjectTypeDef {
	return ObjectTypeDef{
		Name:        "ActorObject",
		Description: "A player or mob actor.",
		Methods: []ObjectMethod{
			m("UserId", "number", "Returns the user ID, or 0 for mobs."),
			m("InstanceId", "number", "Returns the mob instance ID, or 0 for players."),
			m("MobTypeId", "number", "Returns the mob type ID, or 0 for players."),
			m("ShorthandId", "string", "Returns the shorthand actor ID (e.g. @123 or #45)."),
			m("GetLevel", "number", "Returns the actor's level."),
			m("GetStat", "number", "Returns the value of a named stat.", p("statName", "string")),
			m("GetStatMod", "number", "Returns the value of a named stat modifier.", p("statModName", "string")),
			m("GetRace", "string", "Returns the actor's current race name."),
			m("GetTrueRace", "string", "Returns the actor's true (unchanged) race name."),
			m("GetSize", "string", "Returns the actor's size category."),
			m("IsFormChanged", "boolean", "Returns true if the actor's form has been changed."),
			m("ApplyFormChange", "boolean", "Changes the actor's form to another race.", p("raceId", "number")),
			m("RevertFormChange", "boolean", "Reverts a previous form change."),
			m("SendText", "void", "Sends a text message to the actor.", p("msg", "string")),
			m("GetCharacterName", "string", "Returns the actor's character name.", p("wrapInTags", "boolean")),
			m("SetCharacterName", "void", "Sets the actor's character name.", p("newName", "string")),
			m("GetDescription", "string", "Returns the actor's description."),
			m("SetTempData", "void", "Stores temporary (non-persisted) data on the actor.", p("key", "string"), p("value", "any")),
			m("GetTempData", "any", "Retrieves temporary data stored on the actor.", p("key", "string")),
			m("SetMiscCharacterData", "void", "Stores persisted misc data on the character.", p("key", "string"), p("value", "any")),
			m("GetMiscCharacterData", "any", "Retrieves persisted misc data from the character.", p("key", "string")),
			m("GetMiscCharacterDataKeys", "string[]", "Returns misc data keys, optionally filtered by prefix.", p("...prefixMatches", "string")),
			m("GetRoomId", "number", "Returns the room ID the actor is currently in."),
			m("MoveRoom", "void", "Moves the actor to another room.", p("roomId", "number")),
			m("GetHealth", "number", "Returns current health."),
			m("GetHealthMax", "number", "Returns maximum health."),
			m("GetHealthPct", "number", "Returns current health as a percentage."),
			m("GetHealthAppearance", "string", "Returns a textual description of health state."),
			m("SetHealth", "void", "Sets current health.", p("amt", "number")),
			m("AddHealth", "number", "Adds (or subtracts) health and returns the new value.", p("amt", "number")),
			m("GetMana", "number", "Returns current mana."),
			m("GetManaMax", "number", "Returns maximum mana."),
			m("GetManaPct", "number", "Returns current mana as a percentage."),
			m("AddMana", "number", "Adds (or subtracts) mana and returns the new value.", p("amt", "number")),
			m("GetActionPoints", "number", "Returns current action points."),
			m("GetActionPointsMax", "number", "Returns maximum action points."),
			m("GetGold", "number", "Returns gold carried."),
			m("GetBank", "number", "Returns gold in the bank."),
			m("AddGold", "void", "Adds gold to the actor (and optionally the bank).", p("amount", "number"), p("bankAmount?", "number")),
			m("GetAlignment", "number", "Returns the numeric alignment value."),
			m("GetAlignmentName", "string", "Returns the alignment name."),
			m("ChangeAlignment", "void", "Adjusts alignment by the given amount.", p("alignmentChange", "number")),
			m("HasBuff", "boolean", "Returns true if the actor has the given buff.", p("buffId", "number")),
			m("GiveBuff", "void", "Applies a buff to the actor.", p("buffId", "number"), p("source", "string")),
			m("RemoveBuff", "boolean", "Removes a buff from the actor.", p("buffId", "number")),
			m("HasBuffFlag", "boolean", "Returns true if the actor has any buff with the given flag.", p("buffFlag", "string")),
			m("CancelBuffWithFlag", "boolean", "Cancels buffs carrying the given flag.", p("buffFlag", "string")),
			m("GiveItem", "void", "Gives an item to the actor.", p("item", "ItemObject")),
			m("TakeItem", "void", "Takes an item from the actor.", p("item", "ItemObject")),
			m("GetWornItems", "ItemObject[]", "Returns all worn/equipped items."),
			m("GetWornItem", "ItemObject", "Returns the item worn in a slot, or null.", p("slot", "string")),
			m("GetBackpackItems", "ItemObject[]", "Returns all items in the backpack."),
			m("FindInBackpack", "ItemObject", "Finds a backpack item by name (fuzzy).", p("itemName", "string")),
			m("FindOnBody", "ItemObject", "Finds a worn item by name (fuzzy).", p("itemName", "string")),
			m("HasItemId", "boolean", "Returns true if the actor has an item with the given spec ID.", p("itemId", "number"), p("excludeWorn?", "boolean")),
			m("UpdateItem", "void", "Persists changes made to an item the actor holds.", p("item", "ItemObject")),
			m("IsTameable", "boolean", "Returns true if the actor can be tamed."),
			m("IsCharmed", "boolean", "Returns true if the actor is charmed.", p("userId?", "number")),
			m("IsInCombat", "boolean", "Returns true if the actor is in combat."),
			m("IsHome", "boolean", "Returns true if a mob is in its home room."),
			m("IsDowned", "boolean", "Returns true if the actor is downed."),
			m("IsAggro", "boolean", "Returns true if aggressive toward the given actor.", p("actor", "ActorObject")),
			m("Command", "void", "Queues a command for the actor to perform.", p("cmd", "string"), p("waitSeconds?", "number")),
			m("CommandFlagged", "void", "Queues a command with event flags.", p("cmd", "string"), p("flags", "number"), p("waitSeconds?", "number")),
			m("GetSkillLevel", "number", "Returns the level of a named skill.", p("skillName", "string")),
			m("GetAllSkills", "object", "Returns a map of skill name to level."),
			m("TrainSkill", "boolean", "Sets a skill to a given level.", p("skillName", "string"), p("level", "number")),
			m("HasSpell", "boolean", "Returns true if the actor knows a spell.", p("spellId", "string")),
			m("LearnSpell", "boolean", "Teaches a spell to the actor.", p("spellId", "string")),
			m("UnLearnSpell", "boolean", "Removes a spell from the actor.", p("spellId", "string")),
			m("DisableSpell", "boolean", "Disables a known spell.", p("spellId", "string")),
			m("EnableSpell", "boolean", "Re-enables a disabled spell.", p("spellId", "string")),
			m("GetSpells", "object", "Returns a map of spell ID to level."),
			m("HasQuest", "boolean", "Returns true if the actor has progress on a quest.", p("questId", "string")),
			m("GiveQuest", "void", "Grants a quest token to the actor.", p("questId", "string")),
			m("IsQuestDone", "boolean", "Returns true if a quest token is complete.", p("questToken", "string")),
			m("ClearQuestToken", "void", "Clears a quest token from the actor.", p("questToken", "string")),
			m("GetParty", "PartyObject", "Returns the actor's party (members present and missing).", p("excludeSelf?", "boolean")),
			m("GetPartyPresent", "PartyObject", "Returns the actor's party limited to members in the same room.", p("excludeSelf?", "boolean")),
			m("GetPartyMissing", "PartyObject", "Returns the actor's party limited to members not in the same room."),
			m("GetMobKills", "number", "Returns how many of a mob type the actor has killed.", p("mobId", "number")),
			m("GetRaceKills", "number", "Returns how many of a race the actor has killed.", p("race", "string")),
			m("GetCharmCount", "number", "Returns the number of currently charmed creatures."),
			m("GetMaxCharmCount", "number", "Returns the maximum number of charmable creatures."),
			m("GetCharmedUserId", "number", "Returns the user ID that charmed this mob, or 0."),
			m("CharmSet", "void", "Charms the actor for the given user.", p("userId", "number"), p("charmRounds", "number"), p("onRevertCommand?", "string")),
			m("CharmRemove", "void", "Removes charm from the actor."),
			m("CharmExpire", "void", "Expires the actor's charm immediately."),
			m("GetTameMastery", "object", "Returns a map of mob ID to tame mastery level."),
			m("SetTameMastery", "void", "Sets tame mastery for a mob ID.", p("mobId", "number"), p("skillLevel", "number")),
			m("GetChanceToTame", "number", "Returns the chance to tame a target.", p("target", "ActorObject")),
			m("GetTrainingPoints", "number", "Returns available training points."),
			m("GiveTrainingPoints", "void", "Grants training points.", p("count", "number")),
			m("GetStatPoints", "number", "Returns available stat points."),
			m("GiveStatPoints", "void", "Grants stat points.", p("count", "number")),
			m("GetExperience", "number", "Returns the actor's experience."),
			m("GrantXP", "void", "Grants experience to the actor.", p("amount", "number"), p("reason", "string")),
			m("GetExtraLives", "number", "Returns the number of extra lives."),
			m("GiveExtraLife", "void", "Grants an extra life."),
			m("GetDefense", "number", "Returns the actor's defense value."),
			m("GetGearValue", "number", "Returns the total value of worn gear."),
			m("GetCarryCapacity", "number", "Returns the actor's carry capacity."),
			m("GetAdjectives", "string[]", "Returns the actor's adjectives."),
			m("HasAdjective", "boolean", "Returns true if the actor has an adjective.", p("adj", "string")),
			m("SetAdjective", "void", "Adds or removes an adjective.", p("adj", "string"), p("addIt", "boolean")),
			m("GetCooldown", "number", "Returns rounds left on a named cooldown.", p("tag", "string")),
			m("TryCooldown", "boolean", "Starts a cooldown if not active; returns true if started.", p("tag", "string"), p("period", "string")),
			m("GetSetting", "string", "Returns a named character setting.", p("name", "string")),
			m("SetSetting", "void", "Sets a named character setting.", p("name", "string"), p("value", "string")),
			m("TimerSet", "void", "Starts a named timer for a period.", p("name", "string"), p("period", "string")),
			m("TimerExpired", "boolean", "Returns true if a named timer has expired.", p("name", "string")),
			m("TimerExists", "boolean", "Returns true if a named timer exists.", p("name", "string")),
			m("SetResetRoomId", "void", "Sets the room the mob resets to.", p("roomId", "number")),
			m("AddEventLog", "void", "Adds an entry to the actor's event log.", p("category", "string"), p("message", "string")),
			m("MarkVisitedRoom", "void", "Marks rooms as visited by the actor.", p("...roomIds", "number")),
			m("MarkVisitedZone", "void", "Marks all rooms in a zone as visited.", p("zoneName", "string")),
			m("GetZoneVisitProgress", "object", "Returns visited/total/percent for a zone.", p("zoneName", "string")),
			m("Pathing", "boolean", "Returns true if the actor is pathing."),
			m("PathingAtWaypoint", "boolean", "Returns true if the actor is at a path waypoint."),
			m("Sleep", "void", "Pauses the actor for a number of seconds.", p("seconds", "number")),
			m("GetLastInputRound", "number", "Returns the round of the actor's last input."),
			m("PlaySound", "void", "Plays a sound for the actor.", p("soundId", "string"), p("category", "string")),
			m("PlayMusic", "void", "Plays music for the actor.", p("musicFileOrId", "string")),
			m("Uncurse", "ItemObject[]", "Uncurses worn items and returns them."),
			m("GetPet", "PetObject", "Returns the actor's pet, or null."),
		},
	}
}

func roomObjectType() ObjectTypeDef {
	return ObjectTypeDef{
		Name:        "RoomObject",
		Description: "A room in the world.",
		Methods: []ObjectMethod{
			m("RoomId", "number", "Returns the room ID."),
			m("RoomIdSource", "number", "Returns the source room ID for ephemeral rooms."),
			m("GetTitle", "string", "Returns the room title."),
			m("SetTitle", "void", "Sets the room title.", p("title", "string")),
			m("GetDescription", "string", "Returns the room description."),
			m("SetDescription", "void", "Sets the room description.", p("desc", "string")),
			m("GetZone", "string", "Returns the room's zone name."),
			m("SetTempData", "void", "Stores temporary (non-persisted) data on the room.", p("key", "string"), p("value", "any")),
			m("GetTempData", "any", "Retrieves temporary data stored on the room.", p("key", "string")),
			m("SetPermData", "void", "Stores persisted data on the room.", p("key", "string"), p("value", "any")),
			m("GetPermData", "any", "Retrieves persisted data from the room.", p("key", "string")),
			m("GetItems", "ItemObject[]", "Returns items on the floor."),
			m("GetStashItems", "ItemObject[]", "Returns items in the room stash."),
			m("DestroyItem", "void", "Removes an item from the room.", p("item", "ItemObject")),
			m("SpawnItem", "void", "Spawns an item in the room.", p("itemId", "number"), p("inStash", "boolean")),
			m("RepeatSpawnItem", "boolean", "Periodically respawns an item.", p("itemId", "number"), p("roundFrequency", "number"), p("containerName?", "string")),
			m("GetMobs", "ActorObject[]", "Returns mobs in the room, optionally filtered by mob ID.", p("mobId?", "number")),
			m("GetMob", "ActorObject", "Returns a specific mob in the room, or null.", p("mobId", "number"), p("createIfMissing?", "boolean")),
			m("GetPlayers", "ActorObject[]", "Returns players in the room."),
			m("GetAllActors", "ActorObject[]", "Returns all actors (players and mobs) in the room."),
			m("GetContainers", "ContainerObject[]", "Returns all containers in the room."),
			m("GetContainer", "ContainerObject", "Returns a named container, or null.", p("name", "string")),
			m("SpawnMob", "ActorObject", "Spawns a mob in the room.", p("mobId", "number")),
			m("SpawnTempContainer", "string", "Spawns a temporary container.", p("name", "string"), p("duration", "string"), p("lockDifficulty", "number"), p("...trapBuffIds", "number")),
			m("AddTemporaryExit", "boolean", "Adds a temporary exit.", p("exitNameSimple", "string"), p("exitNameFancy", "string"), p("exitRoomId", "number"), p("expiresTimeString", "string")),
			m("RemoveTemporaryExit", "boolean", "Removes a temporary exit.", p("exitNameSimple", "string"), p("exitNameFancy", "string"), p("exitRoomId", "number")),
			m("IsLocked", "boolean", "Returns true if an exit is locked.", p("exitName", "string")),
			m("SetLocked", "void", "Locks or unlocks an exit.", p("exitName", "string"), p("lockIt", "boolean")),
			m("HasTag", "boolean", "Returns true if the room has a tag.", p("tag", "string")),
			m("SetTag", "void", "Adds a tag to the room.", p("tag", "string")),
			m("UnsetTag", "void", "Removes a tag from the room.", p("tag", "string")),
			m("HasMutator", "boolean", "Returns true if the room has a mutator.", p("mutName", "string")),
			m("AddMutator", "void", "Adds a mutator to the room.", p("mutName", "string")),
			m("RemoveMutator", "void", "Removes a mutator from the room.", p("mutName", "string")),
			m("GetExits", "object", "Returns the room's exits."),
			m("HasQuest", "number[]", "Returns user IDs in the room with quest progress.", p("questId", "string"), p("partyUserId?", "number")),
			m("MissingQuest", "number[]", "Returns user IDs missing a quest.", p("questId", "string"), p("partyUserId?", "number")),
			m("SendText", "void", "Sends text to everyone in the room.", p("msg", "string"), p("...excludeUserIds", "number")),
			m("SendTextToExits", "void", "Sends text to adjacent rooms.", p("msg", "string"), p("isQuiet", "boolean"), p("...excludeUserIds", "number")),
			m("GetGold", "number", "Returns gold on the room floor."),
			m("AddGold", "void", "Adds gold to the room floor.", p("amount", "number")),
			m("RemoveGold", "number", "Removes gold from the room floor.", p("amount", "number")),
			m("GetNouns", "object", "Returns the room's nouns map."),
			m("AddNoun", "void", "Adds a noun and its description.", p("noun", "string"), p("description", "string")),
			m("RemoveNoun", "void", "Removes a noun.", p("noun", "string")),
			m("GetSigns", "string[]", "Returns the room's signs."),
			m("AddSign", "boolean", "Adds a sign to the room.", p("text", "string"), p("userId", "number"), p("days", "number")),
			m("MobCount", "number", "Returns the number of mobs in the room."),
			m("PlayerCount", "number", "Returns the number of players in the room."),
			m("GetVisibility", "number", "Returns the room's visibility level."),
			m("IsCalm", "boolean", "Returns true if combat is not allowed."),
			m("IsPvp", "boolean", "Returns true if the room is PvP."),
			m("IsBank", "boolean", "Returns true if the room is a bank."),
			m("IsEphemeral", "boolean", "Returns true if the room is ephemeral."),
			m("AreMobsAttacking", "boolean", "Returns true if mobs are attacking the user.", p("userId", "number")),
			m("ArePlayersAttacking", "boolean", "Returns true if players are attacking the user.", p("userId", "number")),
			m("HasRecentVisitors", "boolean", "Returns true if the room had recent visitors."),
			m("PlaySound", "void", "Plays a sound for everyone in the room.", p("soundId", "string"), p("category", "string"), p("...excludeUserIds", "number")),
		},
	}
}

func itemObjectType() ObjectTypeDef {
	return ObjectTypeDef{
		Name:        "ItemObject",
		Description: "An item instance.",
		Methods: []ObjectMethod{
			m("ItemId", "number", "Returns the item spec ID."),
			m("ShorthandId", "string", "Returns the shorthand item ID."),
			m("Name", "string", "Returns the item name.", p("simpleVersion?", "boolean")),
			m("NameSimple", "string", "Returns the simple item name."),
			m("NameComplex", "string", "Returns the complex item name."),
			m("Rename", "void", "Renames the item.", p("newName", "string"), p("displayNameOrStyle?", "string")),
			m("GetDescription", "string", "Returns the item description."),
			m("Redescribe", "void", "Sets the item description.", p("newDescription", "string")),
			m("GetType", "string", "Returns the item type."),
			m("GetSubtype", "string", "Returns the item subtype."),
			m("GetValue", "number", "Returns the item's value in gold."),
			m("GetElement", "string", "Returns the item's element."),
			m("GetQuestToken", "string", "Returns the item's quest token."),
			m("GetBuffIds", "number[]", "Returns buffs granted on use."),
			m("GetWornBuffIds", "number[]", "Returns buffs granted while worn."),
			m("GetStatMods", "object", "Returns the item's stat modifiers."),
			m("GetDamageReduction", "number", "Returns the item's damage reduction."),
			m("GetDamage", "object", "Returns the item's damage profile."),
			m("GetBreakChance", "number", "Returns the item's break chance."),
			m("GetKeyLockId", "string", "Returns the lock ID this key opens."),
			m("GetUsesLeft", "number", "Returns remaining uses."),
			m("SetUsesLeft", "number", "Sets remaining uses.", p("amount", "number")),
			m("AddUsesLeft", "number", "Adds to remaining uses.", p("amount", "number")),
			m("GetLastUsedRound", "number", "Returns the round the item was last used."),
			m("MarkLastUsed", "number", "Marks the item as used this round.", p("clear?", "boolean")),
			m("IsCursed", "boolean", "Returns true if the item is cursed."),
			m("HasUses", "boolean", "Returns true if the item has limited uses."),
			m("IsWearable", "boolean", "Returns true if the item can be worn."),
			m("IsWeapon", "boolean", "Returns true if the item is a weapon."),
			m("SetTempData", "void", "Stores temporary data on the item.", p("key", "string"), p("value", "any")),
			m("GetTempData", "any", "Retrieves temporary data from the item.", p("key", "string")),
		},
	}
}

func petObjectType() ObjectTypeDef {
	return ObjectTypeDef{
		Name:        "PetObject",
		Description: "A pet companion.",
		Methods: []ObjectMethod{
			m("Name", "string", "Returns the pet's display name."),
			m("NameSimple", "string", "Returns the pet's simple name."),
			m("SetName", "void", "Sets the pet's name.", p("name", "string")),
			m("Type", "string", "Returns the pet type."),
			m("Level", "number", "Returns the pet's level."),
			m("Food", "string", "Returns the pet's food type."),
			m("FoodLevel", "number", "Returns the pet's food level."),
			m("Feed", "void", "Feeds the pet."),
			m("Starve", "void", "Reduces the pet's food level."),
			m("GetStatMod", "number", "Returns a named stat modifier the pet grants.", p("statName", "string")),
			m("GetCapacity", "number", "Returns the pet's item capacity."),
			m("ItemCount", "number", "Returns the number of items the pet carries."),
			m("IsMissing", "boolean", "Returns true if the pet is missing."),
			m("GoMissing", "void", "Sends the pet missing for a number of rounds.", p("rounds", "number")),
			m("HasScript", "boolean", "Returns true if the pet type has a script."),
			m("GetItems", "ItemObject[]", "Returns the items the pet carries."),
			m("FindItem", "ItemObject", "Finds a carried item by name.", p("itemName", "string")),
			m("StoreItem", "boolean", "Stores an item on the pet.", p("item", "ItemObject")),
			m("RemoveItem", "boolean", "Removes an item from the pet.", p("item", "ItemObject")),
			m("GetBuffIds", "number[]", "Returns buffs the pet grants its owner."),
		},
	}
}

func partyObjectType() ObjectTypeDef {
	return ObjectTypeDef{
		Name:        "PartyObject",
		Description: "A group of actors (party members and their charmed creatures). Most methods apply the same action to every member.",
		Methods: []ObjectMethod{
			m("GetMembers", "ActorObject[]", "Returns all actors in the party."),
			m("SendText", "void", "Sends a text message to every party member.", p("msg", "string")),
			m("SetResetRoomId", "void", "Sets the reset room for every party member.", p("roomId", "number")),
			m("GiveQuest", "void", "Grants a quest token to every party member.", p("questId", "string")),
			m("AddGold", "void", "Adds gold to every party member (and optionally the bank).", p("amount", "number"), p("bankAmount?", "number")),
			m("AddHealth", "void", "Adds (or subtracts) health for every party member.", p("amt", "number")),
			m("AddMana", "void", "Adds (or subtracts) mana for every party member.", p("amt", "number")),
			m("Command", "void", "Queues a command for every party member to perform.", p("cmd", "string"), p("waitSeconds?", "number")),
			m("TrainSkill", "void", "Sets a skill to a given level for every party member.", p("skillName", "string"), p("level", "number")),
			m("MoveRoom", "void", "Moves every party member to another room.", p("roomId", "number")),
			m("AddEventLog", "void", "Adds an entry to every party member's event log.", p("category", "string"), p("message", "string")),
			m("GiveBuff", "void", "Applies a buff to every party member.", p("buffId", "number"), p("source", "string")),
			m("CancelBuffWithFlag", "void", "Cancels buffs carrying the given flag on every party member.", p("buffFlag", "string")),
			m("RemoveBuff", "void", "Removes a buff from every party member.", p("buffId", "number")),
			m("ChangeAlignment", "void", "Adjusts alignment by the given amount for every party member.", p("alignmentChange", "number")),
			m("LearnSpell", "void", "Teaches a spell to every party member.", p("spellId", "string")),
			m("SetHealth", "void", "Sets current health for every party member.", p("amt", "number")),
			m("SetAdjective", "void", "Adds or removes an adjective on every party member.", p("adj", "string"), p("addIt", "boolean")),
			m("GiveTrainingPoints", "void", "Grants training points to every party member.", p("count", "number")),
			m("GiveStatPoints", "void", "Grants stat points to every party member.", p("count", "number")),
			m("GiveExtraLife", "void", "Grants an extra life to every party member."),
			m("GrantXP", "void", "Grants experience to every party member.", p("amount", "number"), p("reason", "string")),
			m("TimerSet", "void", "Starts a named timer for every party member.", p("name", "string"), p("period", "string")),
			m("MarkVisitedRoom", "void", "Marks rooms as visited for every party member.", p("...roomIds", "number")),
			m("MarkVisitedZone", "void", "Marks all rooms in a zone as visited for every party member.", p("zoneName", "string")),
		},
	}
}

func containerScriptObjectType() ObjectTypeDef {
	return ObjectTypeDef{
		Name:        "ContainerObject",
		Description: "A named container inside a room.",
		Methods: []ObjectMethod{
			m("Name", "string", "Returns the container name."),
			m("HasLock", "boolean", "Returns true if a lock is configured."),
			m("IsLocked", "boolean", "Returns true if currently locked."),
			m("Lock", "void", "Locks the container."),
			m("Unlock", "void", "Unlocks the container."),
			m("GetLockDifficulty", "number", "Returns the lock difficulty."),
			m("GetTrapBuffIds", "number[]", "Returns trap buff IDs."),
			m("GetItems", "ItemObject[]", "Returns items in the container."),
			m("FindItem", "ItemObject", "Finds an item by name.", p("itemName", "string")),
			m("AddItem", "boolean", "Adds an item to the container.", p("item", "ItemObject")),
			m("RemoveItem", "boolean", "Removes an item from the container.", p("item", "ItemObject")),
			m("GetGold", "number", "Returns gold in the container."),
			m("AddGold", "void", "Adds gold to the container.", p("amount", "number")),
			m("RemoveGold", "number", "Removes gold from the container.", p("amount", "number")),
			m("Count", "number", "Counts items with a given spec ID.", p("itemId", "number")),
			m("IsTemporary", "boolean", "Returns true if the container is temporary."),
			m("GetDespawnRound", "number", "Returns the despawn round, or 0."),
			m("Exists", "boolean", "Returns true if the container still exists."),
		},
	}
}

func panelLayoutObjectType() ObjectTypeDef {
	return ObjectTypeDef{
		Name:        "PanelLayoutObject",
		Description: "A terminal panel layout built from titled boxes arranged in slots and rows.",
		Methods: []ObjectMethod{
			m("AddSlot", "SlotObject", "Adds a new vertical slot (column) to the layout. Returns the slot for adding rows."),
			m("Panel", "PanelObject", "Returns the named panel for configuration and data population. Throws if the id does not exist.", p("id", "string")),
			m("Render", "string", "Renders the entire layout into a terminal string ready to send to a player."),
		},
	}
}

func slotObjectType() ObjectTypeDef {
	return ObjectTypeDef{
		Name:        "SlotObject",
		Description: "A vertical slot (column) within a panel layout, holding rows of panels.",
		Methods: []ObjectMethod{
			m("AddRow", "SlotObject", "Adds a horizontal row of panels (by id) to this slot. Returns the slot for chaining.", p("ids", "string[]")),
		},
	}
}

func panelObjectType() ObjectTypeDef {
	return ObjectTypeDef{
		Name:        "PanelObject",
		Description: "A single titled box within a panel layout. Configuration methods return the panel for chaining.",
		Methods: []ObjectMethod{
			m("SetTitle", "PanelObject", "Sets the panel's title string (used verbatim in the top border). May contain ANSI tags.", p("title", "string")),
			m("SetLabelWidth", "PanelObject", "Sets a fixed visual width that all labels are padded to.", p("w", "number")),
			m("SetWidth", "PanelObject", "Sets the panel's total border-inclusive width.", p("w", "number")),
			m("SetCharset", "PanelObject", `Overrides the border character set ("single", "double", "rounded", or an 8-rune literal).`, p("name", "string")),
			m("SetColumns", "PanelObject", "Sets the number of label+value pairs per line (1 or 2).", p("n", "number")),
			m("SetColumnGap", "PanelObject", "Sets the spaces between columns when using 2-column mode.", p("n", "number")),
			m("Add", "PanelObject", "Appends a label+value row. fullLabel is used when it fits; shortLabel is the fallback.", p("fullLabel", "string"), p("shortLabel", "string"), p("value", "string")),
			m("AddWithWrapWidth", "PanelObject", "Appends a label+value row, wrapping the value when it exceeds wrapWidth.", p("fullLabel", "string"), p("shortLabel", "string"), p("value", "string"), p("wrapWidth", "number")),
			m("AddBlank", "PanelObject", "Appends an empty spacer row for visual separation."),
		},
	}
}
