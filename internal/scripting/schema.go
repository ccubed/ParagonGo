package scripting

type ParamVariant struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

type ScriptFuncParam struct {
	Name         string                   `json:"name"`
	Type         string                   `json:"type"`
	Description  string                   `json:"description"`
	TypeVariants map[string]*ParamVariant `json:"typeVariants,omitempty"`
}

type DynamicName struct {
	Placeholder string `json:"placeholder"`
	Label       string `json:"label"`
	Description string `json:"description"`
	InputType   string `json:"inputType"`
}

type ScriptFuncDef struct {
	Name            string            `json:"name"`
	Description     string            `json:"description"`
	Params          []ScriptFuncParam `json:"params"`
	ReturnSemantics string            `json:"returnSemantics"`
	ExtendedTimeout bool              `json:"extendedTimeout"`
	Dynamic         *DynamicName      `json:"dynamic"`
	Stub            string            `json:"stub"`
}

type ScriptTypeDef struct {
	Label       string          `json:"label"`
	Description string          `json:"description"`
	Functions   []ScriptFuncDef `json:"functions"`
}

type ScriptFunctionsSchema struct {
	Version     int                       `json:"version"`
	ScriptTypes map[string]*ScriptTypeDef `json:"scriptTypes"`
}

var commandDynamic = &DynamicName{
	Placeholder: "{command}",
	Label:       "Command Name",
	Description: "The specific command word this handler responds to (e.g., 'pull', 'push', 'activate').",
	InputType:   "text",
}

func GetScriptFunctionsSchema() *ScriptFunctionsSchema {
	return &ScriptFunctionsSchema{
		Version: 1,
		ScriptTypes: map[string]*ScriptTypeDef{
			"room":  roomScriptType(),
			"mob":   mobScriptType(),
			"item":  itemScriptType(),
			"pet":   petScriptType(),
			"spell": spellScriptType(),
			"buff":  buffScriptType(),
		},
	}
}

func roomScriptType() *ScriptTypeDef {
	return &ScriptTypeDef{
		Label:       "Room Script",
		Description: "Scripts attached to rooms. Triggered by player actions and room lifecycle events.",
		Functions: []ScriptFuncDef{
			{
				Name:        "onLoad",
				Description: "Called once when the room is first loaded into memory. Has an extended timeout for initialization work.",
				Params: []ScriptFuncParam{
					{Name: "room", Type: "RoomObject", Description: "The room being loaded."},
				},
				ReturnSemantics: "Return value is ignored.",
				ExtendedTimeout: true,
				Stub:            "function onLoad(room) {\n\n}\n",
			},
			{
				Name:        "onEnter",
				Description: "Called when a player enters the room.",
				Params: []ScriptFuncParam{
					{Name: "user", Type: "ActorObject", Description: "The player entering the room."},
					{Name: "room", Type: "RoomObject", Description: "The room being entered."},
				},
				ReturnSemantics: "Return false to suppress the default room description on arrival.",
				Stub:            "function onEnter(user, room) {\n\n}\n",
			},
			{
				Name:        "onTryExit",
				Description: "Called before a player leaves the room. Return false to prevent the movement.",
				Params: []ScriptFuncParam{
					{Name: "exitName", Type: "string", Description: "The exit direction or name the player is attempting to use."},
					{Name: "user", Type: "ActorObject", Description: "The player attempting to leave."},
					{Name: "room", Type: "RoomObject", Description: "The room being exited."},
				},
				ReturnSemantics: "Return false to block the movement. Any other return value (or none) allows it.",
				Stub:            "function onTryExit(exitName, user, room) {\n\n    return true;\n}\n",
			},
			{
				Name:        "onExit",
				Description: "Called when a player leaves the room.",
				Params: []ScriptFuncParam{
					{Name: "user", Type: "ActorObject", Description: "The player leaving the room."},
					{Name: "room", Type: "RoomObject", Description: "The room being left."},
				},
				ReturnSemantics: "Return value is ignored.",
				Stub:            "function onExit(user, room) {\n\n}\n",
			},
			{
				Name:        "onIdle",
				Description: "Called each round when the room has players in it.",
				Params: []ScriptFuncParam{
					{Name: "room", Type: "RoomObject", Description: "The current room."},
				},
				ReturnSemantics: "Return true to suppress generic idle actions.",
				Stub:            "function onIdle(room) {\n\n    return false;\n}\n",
			},
			{
				Name:        "onCommand",
				Description: "Called when any command is typed in this room. Fires after mob onCommand handlers.",
				Params: []ScriptFuncParam{
					{Name: "cmd", Type: "string", Description: "The command word typed by the player (e.g., 'look', 'north')."},
					{Name: "rest", Type: "string", Description: "Everything entered after the command word."},
					{Name: "user", Type: "ActorObject", Description: "The player who typed the command."},
					{Name: "room", Type: "RoomObject", Description: "The current room."},
				},
				ReturnSemantics: "Return true to halt further command processing.",
				Stub:            "function onCommand(cmd, rest, user, room) {\n\n    return false;\n}\n",
			},
			{
				Name:        "onCommand_{command}",
				Description: "Called when a specific command is typed in this room. If defined, the generic onCommand() will not fire for this command.",
				Params: []ScriptFuncParam{
					{Name: "rest", Type: "string", Description: "Everything entered after the command word."},
					{Name: "user", Type: "ActorObject", Description: "The player who typed the command."},
					{Name: "room", Type: "RoomObject", Description: "The current room."},
				},
				ReturnSemantics: "Return true to halt further command processing.",
				Dynamic:         commandDynamic,
				Stub:            "function onCommand_{command}(rest, user, room) {\n\n    return true;\n}\n",
			},
		},
	}
}

func mobScriptType() *ScriptTypeDef {
	return &ScriptTypeDef{
		Label:       "Mob Script",
		Description: "Scripts attached to mobs/NPCs. Triggered by mob lifecycle, player interactions, and combat events.",
		Functions: []ScriptFuncDef{
			{
				Name:        "onLoad",
				Description: "Called when a mob instance is created. Has an extended timeout for initialization work.",
				Params: []ScriptFuncParam{
					{Name: "mob", Type: "ActorObject", Description: "The mob being loaded."},
				},
				ReturnSemantics: "Return value is ignored.",
				ExtendedTimeout: true,
				Stub:            "function onLoad(mob) {\n\n}\n",
			},
			{
				Name:        "onIdle",
				Description: "Called each round when the mob is not in combat.",
				Params: []ScriptFuncParam{
					{Name: "mob", Type: "ActorObject", Description: "The mob."},
					{Name: "room", Type: "RoomObject", Description: "The room the mob is in."},
				},
				ReturnSemantics: "Return false to allow other default idle behaviors to occur.",
				Stub:            "function onIdle(mob, room) {\n\n    return false;\n}\n",
			},
			{
				Name:        "onCommand",
				Description: "Called when any command is typed in the mob's room. Fires before the room's onCommand handler.",
				Params: []ScriptFuncParam{
					{Name: "cmd", Type: "string", Description: "The command word typed."},
					{Name: "rest", Type: "string", Description: "Everything entered after the command word."},
					{Name: "mob", Type: "ActorObject", Description: "The mob."},
					{Name: "room", Type: "RoomObject", Description: "The room the mob is in."},
					{Name: "eventDetails", Type: "object", Description: "Contains sourceId (int) and sourceType ('user' or 'mob')."},
				},
				ReturnSemantics: "Return true to halt further command processing.",
				Stub:            "function onCommand(cmd, rest, mob, room, eventDetails) {\n\n    return false;\n}\n",
			},
			{
				Name:        "onCommand_{command}",
				Description: "Called when a specific command is typed in the mob's room. If defined, the generic onCommand() will not fire for this command.",
				Params: []ScriptFuncParam{
					{Name: "rest", Type: "string", Description: "Everything entered after the command word."},
					{Name: "mob", Type: "ActorObject", Description: "The mob."},
					{Name: "room", Type: "RoomObject", Description: "The room the mob is in."},
					{Name: "eventDetails", Type: "object", Description: "Contains sourceId (int) and sourceType ('user' or 'mob')."},
				},
				ReturnSemantics: "Return true to halt further command processing.",
				Dynamic:         commandDynamic,
				Stub:            "function onCommand_{command}(rest, mob, room, eventDetails) {\n\n    return true;\n}\n",
			},
			{
				Name:        "onGive",
				Description: "Called when an item or gold is given to the mob.",
				Params: []ScriptFuncParam{
					{Name: "mob", Type: "ActorObject", Description: "The mob receiving the gift."},
					{Name: "room", Type: "RoomObject", Description: "The room the mob is in."},
					{Name: "eventDetails", Type: "object", Description: "Contains sourceId, sourceType, item (ItemObject), and gold (int)."},
				},
				ReturnSemantics: "Return true to prevent the mob from automatically wearing/equipping the item.",
				Stub:            "function onGive(mob, room, eventDetails) {\n\n    return false;\n}\n",
			},
			{
				Name:        "onShow",
				Description: "Called when an item is shown to the mob.",
				Params: []ScriptFuncParam{
					{Name: "mob", Type: "ActorObject", Description: "The mob being shown an item."},
					{Name: "room", Type: "RoomObject", Description: "The room the mob is in."},
					{Name: "eventDetails", Type: "object", Description: "Contains sourceId, sourceType, and item (ItemObject)."},
				},
				ReturnSemantics: "Return value is ignored.",
				Stub:            "function onShow(mob, room, eventDetails) {\n\n}\n",
			},
			{
				Name:        "onAsk",
				Description: "Called when the mob is asked a question.",
				Params: []ScriptFuncParam{
					{Name: "mob", Type: "ActorObject", Description: "The mob being asked."},
					{Name: "room", Type: "RoomObject", Description: "The room the mob is in."},
					{Name: "eventDetails", Type: "object", Description: "Contains sourceId, sourceType, and askText (string)."},
				},
				ReturnSemantics: "Return false to trigger a generic rejection response.",
				Stub:            "function onAsk(mob, room, eventDetails) {\n\n    return false;\n}\n",
			},
			{
				Name:        "onHurt",
				Description: "Called when the mob takes damage.",
				Params: []ScriptFuncParam{
					{Name: "mob", Type: "ActorObject", Description: "The mob taking damage."},
					{Name: "room", Type: "RoomObject", Description: "The room the mob is in."},
					{Name: "eventDetails", Type: "object", Description: "Contains sourceId, sourceType, damage (int), and crit (bool)."},
				},
				ReturnSemantics: "Return value is ignored.",
				Stub:            "function onHurt(mob, room, eventDetails) {\n\n}\n",
			},
			{
				Name:        "onDie",
				Description: "Called when the mob dies. Called once per attacker that damaged it.",
				Params: []ScriptFuncParam{
					{Name: "mob", Type: "ActorObject", Description: "The dying mob."},
					{Name: "room", Type: "RoomObject", Description: "The room the mob is in."},
					{Name: "eventDetails", Type: "object", Description: "Contains sourceId, sourceType, and attackerCount (int)."},
				},
				ReturnSemantics: "Return value is ignored.",
				Stub:            "function onDie(mob, room, eventDetails) {\n\n}\n",
			},
			{
				Name:        "onPath",
				Description: "Called during mob pathfinding at key stages.",
				Params: []ScriptFuncParam{
					{Name: "mob", Type: "ActorObject", Description: "The mob that is pathing."},
					{Name: "room", Type: "RoomObject", Description: "The room the mob is currently in."},
					{Name: "eventDetails", Type: "object", Description: "Contains sourceId, sourceType, and status ('start', 'waypoint', or 'end')."},
				},
				ReturnSemantics: "Return true to end the current pathing.",
				Stub:            "function onPath(mob, room, eventDetails) {\n\n    return false;\n}\n",
			},
			{
				Name:        "onPlayerDowned",
				Description: "Called when a player is downed (knocked out, not killed) by this mob.",
				Params: []ScriptFuncParam{
					{Name: "mob", Type: "ActorObject", Description: "The mob that downed the player."},
					{Name: "user", Type: "ActorObject", Description: "The player who was downed."},
					{Name: "room", Type: "RoomObject", Description: "The room where it happened."},
				},
				ReturnSemantics: "Return true to ensure this script only runs once per MobId per downed event.",
				Stub:            "function onPlayerDowned(mob, user, room) {\n\n    return true;\n}\n",
			},
		},
	}
}

func itemScriptType() *ScriptTypeDef {
	return &ScriptTypeDef{
		Label:       "Item Script",
		Description: "Scripts attached to items. Triggered when items are acquired, lost, used, or purchased.",
		Functions: []ScriptFuncDef{
			{
				Name:        "onFound",
				Description: "Called when the item is picked up by a player. Changes to the item are saved.",
				Params: []ScriptFuncParam{
					{Name: "user", Type: "ActorObject", Description: "The player who picked up the item."},
					{Name: "item", Type: "ItemObject", Description: "The item that was picked up."},
					{Name: "room", Type: "RoomObject", Description: "The room where the item was found."},
				},
				ReturnSemantics: "Return value is ignored.",
				Stub:            "function onFound(user, item, room) {\n\n}\n",
			},
			{
				Name:        "onLost",
				Description: "Called when the item is dropped by a player. Changes to the item are NOT saved.",
				Params: []ScriptFuncParam{
					{Name: "user", Type: "ActorObject", Description: "The player who dropped the item."},
					{Name: "item", Type: "ItemObject", Description: "The item that was dropped."},
					{Name: "room", Type: "RoomObject", Description: "The room where the item was dropped."},
				},
				ReturnSemantics: "Return value is ignored.",
				Stub:            "function onLost(user, item, room) {\n\n}\n",
			},
			{
				Name:        "onCommand",
				Description: "Called when any command is typed by a player carrying this item.",
				Params: []ScriptFuncParam{
					{Name: "cmd", Type: "string", Description: "The command word typed by the player."},
					{Name: "user", Type: "ActorObject", Description: "The player carrying the item."},
					{Name: "item", Type: "ItemObject", Description: "The item."},
					{Name: "room", Type: "RoomObject", Description: "The current room."},
				},
				ReturnSemantics: "Return true to halt further command processing.",
				Stub:            "function onCommand(cmd, user, item, room) {\n\n    return false;\n}\n",
			},
			{
				Name:        "onCommand_{command}",
				Description: "Called when a specific command is typed by a player carrying this item. If defined, the generic onCommand() will not fire for this command.",
				Params: []ScriptFuncParam{
					{Name: "user", Type: "ActorObject", Description: "The player carrying the item."},
					{Name: "item", Type: "ItemObject", Description: "The item."},
					{Name: "room", Type: "RoomObject", Description: "The current room."},
				},
				ReturnSemantics: "Return true to halt further command processing.",
				Dynamic:         commandDynamic,
				Stub:            "function onCommand_{command}(user, item, room) {\n\n    return true;\n}\n",
			},
			{
				Name:        "onPurchase",
				Description: "Called when the item is purchased from a shop.",
				Params: []ScriptFuncParam{
					{Name: "user", Type: "ActorObject", Description: "The player who purchased the item."},
					{Name: "item", Type: "ItemObject", Description: "The item being purchased."},
					{Name: "room", Type: "RoomObject", Description: "The room where the purchase occurred."},
				},
				ReturnSemantics: "Return false to prevent giving the item to the player (useful for tickets, passes, etc.).",
				Stub:            "function onPurchase(user, item, room) {\n\n    return true;\n}\n",
			},
		},
	}
}

func spellScriptType() *ScriptTypeDef {
	spellTargetParam := ScriptFuncParam{
		Name:        "target",
		Type:        "varies",
		Description: "Depends on spell type: a string for neutral spells, an ActorObject for single-target, or ActorObject[] for multi-target.",
		TypeVariants: map[string]*ParamVariant{
			"neutral":    {Type: "string", Description: "The text entered after the spell command."},
			"harmsingle": {Type: "ActorObject", Description: "The single target actor."},
			"helpsingle": {Type: "ActorObject", Description: "The single target actor."},
			"harmmulti":  {Type: "ActorObject[]", Description: "Array of target actors."},
			"helpmulti":  {Type: "ActorObject[]", Description: "Array of target actors."},
			"harmarea":   {Type: "ActorObject[]", Description: "All actors in the room."},
			"helparea":   {Type: "ActorObject[]", Description: "All actors in the room."},
		},
	}

	sourceParam := ScriptFuncParam{
		Name:        "sourceActor",
		Type:        "ActorObject",
		Description: "The actor casting the spell.",
	}

	return &ScriptTypeDef{
		Label:       "Spell Script",
		Description: "Scripts attached to spells. The second parameter type depends on the spell type (neutral, harmsingle, helpsingle, harmmulti, helpmulti, harmarea, helparea).",
		Functions: []ScriptFuncDef{
			{
				Name:            "onCast",
				Description:     "Called when casting is initiated.",
				Params:          []ScriptFuncParam{sourceParam, spellTargetParam},
				ReturnSemantics: "Return false to abort the casting.",
				Stub:            "function onCast(sourceActor, target) {\n\n    return true;\n}\n",
			},
			{
				Name:            "onWait",
				Description:     "Called each round while waiting for the cast to complete.",
				Params:          []ScriptFuncParam{sourceParam, spellTargetParam},
				ReturnSemantics: "Return value is ignored.",
				Stub:            "function onWait(sourceActor, target) {\n\n}\n",
			},
			{
				Name:            "onMagic",
				Description:     "Called when the spell successfully executes.",
				Params:          []ScriptFuncParam{sourceParam, spellTargetParam},
				ReturnSemantics: "Return true to prevent automatic mob retaliation for harmful spells.",
				Stub:            "function onMagic(sourceActor, target) {\n\n}\n",
			},
			{
				Name:            "onFail",
				Description:     "Called when the spell fails. Reserved for future use.",
				Params:          []ScriptFuncParam{sourceParam, spellTargetParam},
				ReturnSemantics: "Return value is ignored.",
				Stub:            "function onFail(sourceActor, target) {\n\n}\n",
			},
		},
	}
}

func petScriptType() *ScriptTypeDef {
	return &ScriptTypeDef{
		Label:       "Pet Script",
		Description: "Scripts attached to pet types. Triggered by owner commands and round-based pet actions.",
		Functions: []ScriptFuncDef{
			{
				Name:        "PetAct",
				Description: "Called each round when the pet's owner is in a room. Use this to produce scripted pet behavior such as emotes, messages, or reactions.",
				Params: []ScriptFuncParam{
					{Name: "pet", Type: "PetObject", Description: "The pet."},
					{Name: "actor", Type: "ActorObject", Description: "The owner of the pet."},
					{Name: "room", Type: "RoomObject", Description: "The room the pet and owner are in."},
				},
				ReturnSemantics: "Return value is ignored.",
				Stub:            "function PetAct(pet, actor, room) {\n\n}\n",
			},
			{
				Name:        "PetLeave",
				Description: "Called when the pet goes missing (GoMissing is invoked with a positive value). Shares the same signature as PetAct.",
				Params: []ScriptFuncParam{
					{Name: "pet", Type: "PetObject", Description: "The pet."},
					{Name: "actor", Type: "ActorObject", Description: "The owner of the pet."},
					{Name: "room", Type: "RoomObject", Description: "The room the pet and owner are in."},
				},
				ReturnSemantics: "Return value is ignored.",
				Stub:            "function PetLeave(pet, actor, room) {\n\n}\n",
			},
			{
				Name:        "PetReturn",
				Description: "Called when the pet returns after its MissingCountdown reaches zero. Shares the same signature as PetAct.",
				Params: []ScriptFuncParam{
					{Name: "pet", Type: "PetObject", Description: "The pet."},
					{Name: "actor", Type: "ActorObject", Description: "The owner of the pet."},
					{Name: "room", Type: "RoomObject", Description: "The room the pet and owner are in."},
				},
				ReturnSemantics: "Return value is ignored.",
				Stub:            "function PetReturn(pet, actor, room) {\n\n}\n",
			},
			{
				Name:        "onCommand",
				Description: "Called when any command is typed by the pet's owner.",
				Params: []ScriptFuncParam{
					{Name: "cmd", Type: "string", Description: "The command word typed by the owner."},
					{Name: "rest", Type: "string", Description: "Everything entered after the command word."},
					{Name: "pet", Type: "PetObject", Description: "The pet."},
					{Name: "actor", Type: "ActorObject", Description: "The owner of the pet."},
					{Name: "room", Type: "RoomObject", Description: "The current room."},
				},
				ReturnSemantics: "Return true to halt further command processing.",
				Stub:            "function onCommand(cmd, rest, pet, actor, room) {\n\n    return false;\n}\n",
			},
			{
				Name:        "onCommand_{command}",
				Description: "Called when a specific command is typed by the pet's owner. If defined, the generic onCommand() will not fire for this command.",
				Params: []ScriptFuncParam{
					{Name: "rest", Type: "string", Description: "Everything entered after the command word."},
					{Name: "pet", Type: "PetObject", Description: "The pet."},
					{Name: "actor", Type: "ActorObject", Description: "The owner of the pet."},
					{Name: "room", Type: "RoomObject", Description: "The current room."},
				},
				ReturnSemantics: "Return true to halt further command processing.",
				Dynamic:         commandDynamic,
				Stub:            "function onCommand_{command}(rest, pet, actor, room) {\n\n    return true;\n}\n",
			},
		},
	}
}

func buffScriptType() *ScriptTypeDef {
	return &ScriptTypeDef{
		Label:       "Buff Script",
		Description: "Scripts attached to buffs/status effects. Triggered during the buff lifecycle.",
		Functions: []ScriptFuncDef{
			{
				Name:        "onStart",
				Description: "Called when the buff is first applied to an actor.",
				Params: []ScriptFuncParam{
					{Name: "actor", Type: "ActorObject", Description: "The actor receiving the buff."},
					{Name: "triggersLeft", Type: "number", Description: "How many trigger rounds remain."},
				},
				ReturnSemantics: "Return value is ignored.",
				Stub:            "function onStart(actor, triggersLeft) {\n\n}\n",
			},
			{
				Name:        "onTrigger",
				Description: "Called each time the buff triggers (based on its round interval).",
				Params: []ScriptFuncParam{
					{Name: "actor", Type: "ActorObject", Description: "The actor with the buff."},
					{Name: "triggersLeft", Type: "number", Description: "How many trigger rounds remain."},
				},
				ReturnSemantics: "Return value is ignored.",
				Stub:            "function onTrigger(actor, triggersLeft) {\n\n}\n",
			},
			{
				Name:        "onEnd",
				Description: "Called when the buff expires, just before it is removed.",
				Params: []ScriptFuncParam{
					{Name: "actor", Type: "ActorObject", Description: "The actor losing the buff."},
					{Name: "triggersLeft", Type: "number", Description: "Triggers remaining (usually 0)."},
				},
				ReturnSemantics: "Return value is ignored.",
				Stub:            "function onEnd(actor, triggersLeft) {\n\n}\n",
			},
			{
				Name:        "onCommand",
				Description: "Called when any command is typed by an actor with this buff.",
				Params: []ScriptFuncParam{
					{Name: "cmd", Type: "string", Description: "The command word typed."},
					{Name: "rest", Type: "string", Description: "Everything entered after the command word."},
					{Name: "actor", Type: "ActorObject", Description: "The actor with the buff."},
					{Name: "room", Type: "RoomObject", Description: "The current room."},
				},
				ReturnSemantics: "Return true to halt further command processing.",
				Stub:            "function onCommand(cmd, rest, actor, room) {\n\n    return false;\n}\n",
			},
			{
				Name:        "onCommand_{command}",
				Description: "Called when a specific command is typed by an actor with this buff. If defined, the generic onCommand() will not fire for this command.",
				Params: []ScriptFuncParam{
					{Name: "rest", Type: "string", Description: "Everything entered after the command word."},
					{Name: "actor", Type: "ActorObject", Description: "The actor with the buff."},
					{Name: "room", Type: "RoomObject", Description: "The current room."},
				},
				ReturnSemantics: "Return true to halt further command processing.",
				Dynamic:         commandDynamic,
				Stub:            "function onCommand_{command}(rest, actor, room) {\n\n    return true;\n}\n",
			},
		},
	}
}
