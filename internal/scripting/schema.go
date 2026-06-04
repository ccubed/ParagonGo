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

type EngineGlobalFuncDef struct {
	Name            string            `json:"name"`
	Description     string            `json:"description"`
	Params          []ScriptFuncParam `json:"params"`
	ReturnType      string            `json:"returnType,omitempty"`
	ReturnSemantics string            `json:"returnSemantics,omitempty"`
}

type ScriptFunctionsSchema struct {
	Version         int                       `json:"version"`
	ScriptTypes     map[string]*ScriptTypeDef `json:"scriptTypes"`
	EngineFunctions []EngineGlobalFuncDef     `json:"engineFunctions"`
}

var commandDynamic = &DynamicName{
	Placeholder: "{command}",
	Label:       "Command Name",
	Description: "The specific command word this handler responds to (e.g., 'pull', 'push', 'activate').",
	InputType:   "text",
}

func GetScriptFunctionsSchema() *ScriptFunctionsSchema {
	return &ScriptFunctionsSchema{
		Version: 2,
		ScriptTypes: map[string]*ScriptTypeDef{
			"room":      roomScriptType(),
			"mob":       mobScriptType(),
			"item":      itemScriptType(),
			"pet":       petScriptType(),
			"spell":     spellScriptType(),
			"buff":      buffScriptType(),
			"container": containerObjectType(),
		},
		EngineFunctions: engineGlobalFunctions(),
	}
}

func engineGlobalFunctions() []EngineGlobalFuncDef {
	return []EngineGlobalFuncDef{
		// Messaging
		{
			Name:        "SendUserMessage",
			Description: "Sends a message to a specific online user by their user ID.",
			Params: []ScriptFuncParam{
				{Name: "userId", Type: "number", Description: "The user ID of the recipient."},
				{Name: "message", Type: "string", Description: "The message text (supports ANSI tags)."},
			},
		},
		{
			Name:        "SendRoomMessage",
			Description: "Sends a message to all players currently in a room.",
			Params: []ScriptFuncParam{
				{Name: "roomId", Type: "number", Description: "The room ID to send the message to."},
				{Name: "message", Type: "string", Description: "The message text (supports ANSI tags)."},
				{Name: "...excludeIds", Type: "number", Description: "Optional user IDs to exclude from receiving the message."},
			},
		},
		{
			Name:        "SendRoomExitsMessage",
			Description: "Sends a message to players in all rooms adjacent to the given room.",
			Params: []ScriptFuncParam{
				{Name: "roomId", Type: "number", Description: "The source room ID whose exits are targeted."},
				{Name: "message", Type: "string", Description: "The message text (supports ANSI tags)."},
				{Name: "isQuiet", Type: "boolean", Description: "If true, suppresses the message in certain quiet contexts."},
				{Name: "...excludeUserIds", Type: "number", Description: "Optional user IDs to exclude from receiving the message."},
			},
		},
		{
			Name:        "SendBroadcast",
			Description: "Sends a server-wide broadcast message to all online players.",
			Params: []ScriptFuncParam{
				{Name: "message", Type: "string", Description: "The message text (supports ANSI tags)."},
			},
		},
		// Room
		{
			Name:        "GetRoom",
			Description: "Loads and returns a room by its ID. Returns null if the room does not exist.",
			Params: []ScriptFuncParam{
				{Name: "roomId", Type: "number", Description: "The room ID to load."},
			},
			ReturnType:      "RoomObject",
			ReturnSemantics: "The room object, or null if not found.",
		},
		{
			Name:        "GetMap",
			Description: "Renders an ASCII map centered on a room and returns it as a formatted string.",
			Params: []ScriptFuncParam{
				{Name: "mapRoomId", Type: "number", Description: "The room ID the map is centered on."},
				{Name: "zoomLevel", Type: "number", Description: "Zoom level for the map."},
				{Name: "mapHeight", Type: "number", Description: "Height of the map in rows."},
				{Name: "mapWidth", Type: "number", Description: "Width of the map in columns."},
				{Name: "mapName", Type: "string", Description: "Title displayed above the map."},
				{Name: "showSecrets", Type: "boolean", Description: "If true, secret exits and rooms are included."},
				{Name: "...mapMarkers", Type: "string", Description: `Optional custom markers in the format "roomId,symbol,legend" (e.g. "1,×,Here").`},
			},
			ReturnType:      "string",
			ReturnSemantics: "The rendered map string with ANSI color tags.",
		},
		{
			Name:        "CreateEmptyRoomInstances",
			Description: "Creates a number of blank ephemeral (temporary) rooms and returns their IDs.",
			Params: []ScriptFuncParam{
				{Name: "quantity", Type: "number", Description: "How many empty ephemeral rooms to create."},
			},
			ReturnType:      "number[]",
			ReturnSemantics: "Array of newly allocated ephemeral room IDs.",
		},
		{
			Name:        "CreateInstancesFromRoomIds",
			Description: "Creates ephemeral copies of the specified rooms and returns a mapping of original ID to new ephemeral ID.",
			Params: []ScriptFuncParam{
				{Name: "...roomIds", Type: "number", Description: "One or more room IDs to copy into ephemeral instances."},
			},
			ReturnType:      "object",
			ReturnSemantics: "An object mapping each original room ID to its new ephemeral room ID.",
		},
		{
			Name:        "CreateInstancesFromZone",
			Description: "Creates ephemeral copies of every room in a zone and returns a mapping of original IDs to new ephemeral IDs.",
			Params: []ScriptFuncParam{
				{Name: "zoneName", Type: "string", Description: "The name of the zone to duplicate."},
			},
			ReturnType:      "object",
			ReturnSemantics: "An object mapping each original room ID to its new ephemeral room ID.",
		},
		// Actor
		{
			Name:        "GetUser",
			Description: "Retrieves a player actor by their user ID. Returns null if the user is not online.",
			Params: []ScriptFuncParam{
				{Name: "userId", Type: "number", Description: "The user ID of the player."},
			},
			ReturnType:      "ActorObject",
			ReturnSemantics: "The player's actor object, or null if not found.",
		},
		{
			Name:        "GetMob",
			Description: "Retrieves a mob actor by its instance ID. Returns null if the instance does not exist.",
			Params: []ScriptFuncParam{
				{Name: "mobInstanceId", Type: "number", Description: "The instance ID of the mob."},
			},
			ReturnType:      "ActorObject",
			ReturnSemantics: "The mob's actor object, or null if not found.",
		},
		{
			Name:        "ActorNames",
			Description: "Formats a list of actor objects into a human-readable name string (e.g. \"Alice, Bob and Charlie\").",
			Params: []ScriptFuncParam{
				{Name: "actorList", Type: "ActorObject[]", Description: "Array of actor objects whose names should be joined."},
			},
			ReturnType:      "string",
			ReturnSemantics: "A comma-and-\"and\"-separated list of actor names with ANSI color tags.",
		},
		// Item
		{
			Name:        "CreateItem",
			Description: "Creates a new item instance by item spec ID. Returns null if the item ID does not exist.",
			Params: []ScriptFuncParam{
				{Name: "itemId", Type: "number", Description: "The item spec ID to instantiate."},
			},
			ReturnType:      "ItemObject",
			ReturnSemantics: "A new item instance, or null if the item ID is not found.",
		},
		// Panel
		{
			Name:        "PanelLayoutLoad",
			Description: "Loads a panel layout from the datafiles panel-layouts directory by name. Throws on failure.",
			Params: []ScriptFuncParam{
				{Name: "name", Type: "string", Description: `Relative path under panel-layouts/ without extension (e.g. "character/status").`},
			},
			ReturnType:      "PanelLayoutObject",
			ReturnSemantics: "The loaded panel layout object.",
		},
		{
			Name:        "PanelLayoutNew",
			Description: "Creates a panel layout entirely in script without a YAML file.",
			Params: []ScriptFuncParam{
				{Name: "opts?", Type: "object", Description: "Optional settings: border (\"full\"/\"top\"/\"none\"), charset (\"single\"/\"double\"/\"rounded\"), gap (number), margin (number)."},
			},
			ReturnType:      "PanelLayoutObject",
			ReturnSemantics: "A new panel layout object ready for slot and panel configuration.",
		},
		// Util
		{
			Name:        "RandInt",
			Description: "Returns a random integer between min and max, inclusive.",
			Params: []ScriptFuncParam{
				{Name: "min", Type: "number", Description: "Minimum value (inclusive)."},
				{Name: "max", Type: "number", Description: "Maximum value (inclusive)."},
			},
			ReturnType:      "number",
			ReturnSemantics: "A random integer in the range [min, max].",
		},
		{
			Name:        "UtilGetRoundNumber",
			Description: "Returns the current game round number.",
			Params:      []ScriptFuncParam{},
			ReturnType:  "number",
		},
		{
			Name:        "UtilFindMatchIn",
			Description: "Fuzzy-matches a search string against a list of strings. Returns an object with found (bool), exact (string), and close (string) fields.",
			Params: []ScriptFuncParam{
				{Name: "search", Type: "string", Description: "The string to search for."},
				{Name: "items", Type: "string[]", Description: "The list of strings to search within."},
			},
			ReturnType:      "object",
			ReturnSemantics: "Object with fields: found (bool), exact (string, the exact match or empty), close (string, a close prefix match or empty).",
		},
		{
			Name:        "UtilGetSecondsToRounds",
			Description: "Converts a number of real-time seconds to the equivalent number of game rounds.",
			Params: []ScriptFuncParam{
				{Name: "seconds", Type: "number", Description: "Number of real-time seconds."},
			},
			ReturnType: "number",
		},
		{
			Name:        "UtilGetMinutesToRounds",
			Description: "Converts a number of real-time minutes to the equivalent number of game rounds.",
			Params: []ScriptFuncParam{
				{Name: "minutes", Type: "number", Description: "Number of real-time minutes."},
			},
			ReturnType: "number",
		},
		{
			Name:        "UtilGetSecondsToTurns",
			Description: "Converts a number of real-time seconds to the equivalent number of game turns.",
			Params: []ScriptFuncParam{
				{Name: "seconds", Type: "number", Description: "Number of real-time seconds."},
			},
			ReturnType: "number",
		},
		{
			Name:        "UtilGetMinutesToTurns",
			Description: "Converts a number of real-time minutes to the equivalent number of game turns.",
			Params: []ScriptFuncParam{
				{Name: "minutes", Type: "number", Description: "Number of real-time minutes."},
			},
			ReturnType: "number",
		},
		{
			Name:        "UtilStripPrepositions",
			Description: "Strips common leading prepositions (e.g. \"at\", \"the\", \"from\") from a string.",
			Params: []ScriptFuncParam{
				{Name: "input", Type: "string", Description: "The string to strip prepositions from."},
			},
			ReturnType: "string",
		},
		{
			Name:        "UtilDiceRoll",
			Description: "Rolls a set of dice and returns the total. Equivalent to rolling diceQty dice each with diceSides sides.",
			Params: []ScriptFuncParam{
				{Name: "diceQty", Type: "number", Description: "Number of dice to roll."},
				{Name: "diceSides", Type: "number", Description: "Number of sides on each die."},
			},
			ReturnType: "number",
		},
		{
			Name:        "UtilGetTime",
			Description: "Returns the current in-game date and time as a GameDate object with fields for Year, Month, Day, Hour, Minute, Night, etc.",
			Params:      []ScriptFuncParam{},
			ReturnType:  "object",
		},
		{
			Name:        "UtilGetTimeString",
			Description: "Returns the current in-game time as a human-readable formatted string with ANSI color tags.",
			Params:      []ScriptFuncParam{},
			ReturnType:  "string",
		},
		{
			Name:        "UtilSetTime",
			Description: "Sets the in-game clock to a specific hour and minute.",
			Params: []ScriptFuncParam{
				{Name: "hour", Type: "number", Description: "The hour to set (0-23)."},
				{Name: "minutes", Type: "number", Description: "The minute to set (0-59)."},
			},
		},
		{
			Name:        "UtilSetTimeDay",
			Description: "Advances the in-game clock to the next daytime period.",
			Params:      []ScriptFuncParam{},
		},
		{
			Name:        "UtilSetTimeNight",
			Description: "Advances the in-game clock to the next nighttime period.",
			Params:      []ScriptFuncParam{},
		},
		{
			Name:        "UtilIsDay",
			Description: "Returns true if it is currently daytime in the game world.",
			Params:      []ScriptFuncParam{},
			ReturnType:  "boolean",
		},
		{
			Name:        "UtilLocateUser",
			Description: "Returns the room ID where a user is currently located. Accepts either a user ID (number) or character name (string). Returns 0 if the user is not online.",
			Params: []ScriptFuncParam{
				{Name: "idOrName", Type: "number|string", Description: "The user ID or character name to locate."},
			},
			ReturnType:      "number",
			ReturnSemantics: "The room ID the user is in, or 0 if not found.",
		},
		{
			Name:        "UtilApplyColorPattern",
			Description: "Applies a named color pattern to a string and returns the colorized result.",
			Params: []ScriptFuncParam{
				{Name: "input", Type: "string", Description: "The string to colorize."},
				{Name: "patternName", Type: "string", Description: "The name of the color pattern to apply."},
				{Name: "wordsOnly?", Type: "boolean", Description: "If true, applies the pattern word-by-word instead of character-by-character."},
			},
			ReturnType: "string",
		},
		{
			Name:        "UtilGetConfig",
			Description: "Returns the current server configuration object. Useful for reading server settings such as game name, limits, and feature flags.",
			Params:      []ScriptFuncParam{},
			ReturnType:  "object",
		},
		{
			Name:        "ColorWrap",
			Description: "Wraps text in ANSI color tags using named color classes.",
			Params: []ScriptFuncParam{
				{Name: "txt", Type: "string", Description: "The text to wrap."},
				{Name: "fg?", Type: "string", Description: "Optional foreground color class name (e.g. \"red\", \"username\")."},
				{Name: "bg?", Type: "string", Description: "Optional background color class name."},
			},
			ReturnType: "string",
		},
		{
			Name:        "RaiseEvent",
			Description: "Raises a named scripted event with an arbitrary data payload. Other scripts or modules can listen for this event by name.",
			Params: []ScriptFuncParam{
				{Name: "name", Type: "string", Description: "The event name."},
				{Name: "data", Type: "object", Description: "Arbitrary key-value data attached to the event."},
			},
		},
		{
			Name:        "ExpandCommand",
			Description: "Expands command aliases in a string. For example, \"n\" may expand to \"north\".",
			Params: []ScriptFuncParam{
				{Name: "cmd", Type: "string", Description: "The command string to expand."},
				{Name: "limit?", Type: "number", Description: "Optional maximum number of tokens to expand. -1 means unlimited."},
			},
			ReturnType: "string",
		},
		{
			Name:        "EventFlags",
			Description: "A constant object mapping event flag names to their numeric values. Use with CommandFlagged() to control command execution behavior (e.g. EventFlags.CmdSkipScripts, EventFlags.CmdSecretly).",
			Params:      []ScriptFuncParam{},
			ReturnType:  "object",
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
				Name:        "onTryEnter",
				Description: "Called on this room before a player enters it. Return false to prevent the movement.",
				Params: []ScriptFuncParam{
					{Name: "user", Type: "ActorObject", Description: "The player attempting to enter."},
					{Name: "room", Type: "RoomObject", Description: "The room being entered."},
				},
				ReturnSemantics: "Return false to block the movement. Any other return value (or none) allows it.",
				Stub:            "function onTryEnter(user, room) {\n\n    return true;\n}\n",
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
			{
				Name:        "onTryPurchase",
				Description: "Called after the player has enough gold but before the purchase is committed. Return false to block the purchase entirely (no gold is deducted).",
				Params: []ScriptFuncParam{
					{Name: "user", Type: "ActorObject", Description: "The player attempting to purchase the item."},
					{Name: "item", Type: "ItemObject", Description: "The item being purchased."},
					{Name: "room", Type: "RoomObject", Description: "The room where the purchase is occurring."},
				},
				ReturnSemantics: "Return false to block the purchase. Any other return value (or none) allows it.",
				Stub:            "function onTryPurchase(user, item, room) {\n\n    return true;\n}\n",
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

func containerObjectType() *ScriptTypeDef {
	return &ScriptTypeDef{
		Label:       "ContainerObject",
		Description: "Represents a named container inside a room. Returned by RoomObject.GetContainers(). Provides access to container contents, lock state, and gold.",
		Functions: []ScriptFuncDef{
			{
				Name:            "Name",
				Description:     "Returns the name of the container (its key in the room's container map).",
				Params:          []ScriptFuncParam{},
				ReturnSemantics: "The container name string.",
			},
			{
				Name:            "HasLock",
				Description:     "Returns true if the container has a lock (difficulty > 0), regardless of whether it is currently locked or unlocked.",
				Params:          []ScriptFuncParam{},
				ReturnSemantics: "true if a lock is configured on this container.",
			},
			{
				Name:            "IsLocked",
				Description:     "Returns true if the container is currently locked.",
				Params:          []ScriptFuncParam{},
				ReturnSemantics: "true if locked, false if unlocked or no lock.",
			},
			{
				Name:            "Lock",
				Description:     "Locks the container. Has no effect if the container has no lock configured.",
				Params:          []ScriptFuncParam{},
				ReturnSemantics: "Return value is ignored.",
			},
			{
				Name:            "Unlock",
				Description:     "Unlocks the container. Has no effect if the container has no lock configured.",
				Params:          []ScriptFuncParam{},
				ReturnSemantics: "Return value is ignored.",
			},
			{
				Name:            "GetItems",
				Description:     "Returns all items currently inside the container.",
				Params:          []ScriptFuncParam{},
				ReturnSemantics: "Array of ItemObject.",
			},
			{
				Name:        "FindItem",
				Description: "Searches the container for an item matching the given name. Supports fuzzy matching.",
				Params: []ScriptFuncParam{
					{Name: "itemName", Type: "string", Description: "The name to search for."},
				},
				ReturnSemantics: "The matching ItemObject, or null if not found.",
			},
			{
				Name:        "AddItem",
				Description: "Adds an item to the container.",
				Params: []ScriptFuncParam{
					{Name: "item", Type: "ItemObject", Description: "The item to add."},
				},
				ReturnSemantics: "true if the item was added successfully, false if the item was invalid.",
			},
			{
				Name:        "RemoveItem",
				Description: "Removes an item from the container.",
				Params: []ScriptFuncParam{
					{Name: "item", Type: "ItemObject", Description: "The item to remove."},
				},
				ReturnSemantics: "true if the item was found and removed, false otherwise.",
			},
			{
				Name:            "GetGold",
				Description:     "Returns the amount of gold currently in the container.",
				Params:          []ScriptFuncParam{},
				ReturnSemantics: "Gold amount as a number.",
			},
			{
				Name:        "AddGold",
				Description: "Adds gold to the container.",
				Params: []ScriptFuncParam{
					{Name: "amount", Type: "number", Description: "Amount of gold to add. Must be positive."},
				},
				ReturnSemantics: "Return value is ignored.",
			},
			{
				Name:        "RemoveGold",
				Description: "Removes gold from the container. Removes at most the amount currently present.",
				Params: []ScriptFuncParam{
					{Name: "amount", Type: "number", Description: "Amount of gold to remove."},
				},
				ReturnSemantics: "The actual amount removed (may be less than requested if the container had insufficient gold).",
			},
			{
				Name:        "Count",
				Description: "Returns the number of items with the given item spec ID currently in the container.",
				Params: []ScriptFuncParam{
					{Name: "itemId", Type: "number", Description: "The item spec ID to count."},
				},
				ReturnSemantics: "Count of matching items.",
			},
			{
				Name:            "IsTemporary",
				Description:     "Returns true if this container is temporary (it has a despawn round set and will disappear over time).",
				Params:          []ScriptFuncParam{},
				ReturnSemantics: "true if the container has a despawn round configured.",
			},
		},
	}
}
