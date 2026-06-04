package web

// buildAdminNav returns the full admin navigation, combining hardcoded core
// entries with module-contributed entries.
func buildAdminNav() []WebNavItem {
	nav := []WebNavItem{
		{
			Name:        "Lifeforms",
			Description: "Manage players, NPCs, conversations, pets, and races.",
			Children: []WebNavItem{
				{
					Name:        "Users",
					Target:      "/admin/users",
					Description: "Manage player accounts, roles, and permissions.",
					Children: []WebNavItem{
						{Label: "View / Edit", Target: "/admin/users", Description: "Browse, search, and edit player account records."},
						{Label: "API Docs", Target: "/admin/users-api", Description: "REST API reference for the users endpoint."},
					},
				},
				{
					Name:        "Mobs",
					Target:      "/admin/mobs",
					Description: "Define and manage non-player character templates.",
					Children: []WebNavItem{
						{Label: "View / Edit", Target: "/admin/mobs", Description: "Browse, create, and edit mob definitions."},
						{Label: "Rankings", Target: "/admin/mob-rankings", Description: "View mob power and loot rankings across all types."},
						{Label: "API Docs", Target: "/admin/mobs-api", Description: "REST API reference for the mobs endpoint."},
					},
				},
				{
					Name:        "Conversations",
					Target:      "/admin/conversations",
					Description: "Script dialogue sequences between NPCs.",
					Children: []WebNavItem{
						{Label: "View / Edit", Target: "/admin/conversations", Description: "Browse and edit NPC conversation scripts."},
						{Label: "API Docs", Target: "/admin/conversations-api", Description: "REST API reference for the conversations endpoint."},
					},
				},
				{
					Name:        "Pets",
					Target:      "/admin/pets",
					Description: "Configure tameable companion types and rankings.",
					Children: []WebNavItem{
						{Label: "View / Edit", Target: "/admin/pets", Description: "Browse and edit pet type definitions."},
						{Label: "Rankings", Target: "/admin/pets-ranks", Description: "View pet power rankings across all types."},
						{Label: "API Docs", Target: "/admin/pets-api", Description: "REST API reference for the pets endpoint."},
					},
				},
				{
					Name:        "Races",
					Target:      "/admin/races",
					Description: "Define playable and NPC races with stats and traits.",
					Children: []WebNavItem{
						{Label: "View / Edit", Target: "/admin/races", Description: "Browse and edit race definitions."},
						{Label: "API Docs", Target: "/admin/races-api", Description: "REST API reference for the races endpoint."},
					},
				},
			},
		},

		{
			Name:        "Content",
			Description: "Manage items, rooms, buffs, spells, and quests.",
			Children: []WebNavItem{
				{
					Name:        "Items",
					Target:      "/admin/items",
					Description: "Create and manage all equippable and usable item types.",
					Children: []WebNavItem{
						{Label: "View / Edit", Target: "/admin/items", Description: "Browse, create, and edit item definitions."},
						{Label: "Item Ranks (Weapons)", Target: "/admin/items-rank-weapons", Description: "View and adjust weapon tier rankings."},
						{Label: "Item Ranks (Armor)", Target: "/admin/items-rank-armor", Description: "View and adjust armor tier rankings."},
						{Label: "Attack Messages", Target: "/admin/items-attack-messages", Description: "Browse and edit attack message sets."},
						{Label: "API Docs", Target: "/admin/items-api", Description: "REST API reference for the items endpoint."},
					},
				},
				{
					Name:        "Rooms",
					Target:      "/admin/rooms",
					Description: "Build and configure the game world: rooms, zones, biomes, and environmental modifiers.",
					Children: []WebNavItem{
						{Label: "View / Edit", Target: "/admin/rooms", Description: "Browse, create, and edit room data."},
						{Label: "Mapper", Target: "/admin/mapper", Description: "Visualize and navigate the world map interactively."},
						{Label: "Mutators", Target: "/admin/mutators", Description: "Browse and edit mutator definitions."},
						{Label: "Biomes", Target: "/admin/biomes", Description: "Browse and edit biome definitions."},
						{Label: "API Docs", Target: "/admin/rooms-api", Description: "REST API reference for the rooms endpoint."},
					},
				},
				{
					Name:        "Buffs",
					Target:      "/admin/buffs",
					Description: "Define status effects applied to characters by spells, items, and rooms.",
					Children: []WebNavItem{
						{Label: "View / Edit", Target: "/admin/buffs", Description: "Browse, create, and edit buff definitions."},
						{Label: "API Docs", Target: "/admin/buffs-api", Description: "REST API reference for the buffs endpoint."},
					},
				},
				{
					Name:        "Spells",
					Target:      "/admin/spells",
					Description: "Configure castable spells and their scripted behaviors.",
					Children: []WebNavItem{
						{Label: "View / Edit", Target: "/admin/spells", Description: "Browse, create, and edit spell definitions."},
						{Label: "API Docs", Target: "/admin/spells-api", Description: "REST API reference for the spells endpoint."},
					},
				},
				{
					Name:        "Quests",
					Target:      "/admin/quests",
					Description: "Design multi-step quest chains with token-based progression.",
					Children: []WebNavItem{
						{Label: "View / Edit", Target: "/admin/quests", Description: "Browse, create, and edit quest definitions."},
						{Label: "API Docs", Target: "/admin/quests-api", Description: "REST API reference for the quests endpoint."},
					},
				},
			},
		},
		{
			Name:        "Metrics",
			Description: "Inspect runtime stats, search telemetry/event logs.",
			Children: []WebNavItem{
				{
					Name:        "Telemetry",
					Target:      "/admin/telemetry",
					Description: "Query and inspect server-side telemetry event data.",
					Children: []WebNavItem{
						{Label: "View / Query", Target: "/admin/telemetry", Description: "Browse and filter recorded telemetry events."},
						{Label: "API Docs", Target: "/admin/telemetry-api", Description: "REST API reference for the telemetry endpoint."},
					},
				},
				{
					Name:        "Stats",
					Target:      "/admin/stats",
					Description: "View real-time memory and runtime statistics.",
					Children: []WebNavItem{
						{Label: "View", Target: "/admin/stats", Description: "Display current server memory and runtime metrics."},
						{Label: "API Docs", Target: "/admin/stats-api", Description: "REST API reference for the stats endpoint."},
					},
				},
			},
		},
		{
			Name:        "Settings",
			Description: "Configure the server, manage time, audio, panels, and scripting.",
			Children: []WebNavItem{
				{
					Name:        "Config",
					Target:      "/admin/config",
					Description: "View and modify live server configuration settings.",
					Children: []WebNavItem{
						{Label: "View / Edit", Target: "/admin/config", Description: "Browse and edit all server configuration values."},
						{Label: "Config Wizard", Target: "/admin/config-wizard", Description: "Step-by-step guided configuration wizard."},
						{Label: "API Docs", Target: "/admin/config-api", Description: "REST API reference for the config endpoint."},
					},
				},
				{
					Name:        "Progression",
					Target:      "/admin/progression",
					Description: "Configure level-up stat gains and experience curves.",
					Children: []WebNavItem{
						{Label: "View / Edit", Target: "/admin/progression", Description: "View and adjust character progression tables."},
					},
				},
				{
					Name:        "GameTime",
					Target:      "/admin/gametime",
					Description: "Inspect and control the in-game calendar and clock.",
					Children: []WebNavItem{
						{Label: "View / Edit", Target: "/admin/gametime", Description: "View the current game date and adjust time settings."},
						{Label: "API Docs", Target: "/admin/gametime-api", Description: "REST API reference for the gametime endpoint."},
					},
				},
				{
					Name:        "Audio",
					Target:      "/admin/audio",
					Description: "Manage ambient and event-triggered audio cues.",
					Children: []WebNavItem{
						{Label: "View / Edit", Target: "/admin/audio", Description: "Browse and edit audio cue definitions."},
						{Label: "API Docs", Target: "/admin/audio-api", Description: "REST API reference for the audio endpoint."},
					},
				},
				{
					Name:        "Panels",
					Target:      "/admin/panels",
					Description: "Configure custom UI panels displayed in the web client.",
					Children: []WebNavItem{
						{Label: "View / Edit", Target: "/admin/panels", Description: "Browse and edit panel layout definitions."},
						{Label: "API Docs", Target: "/admin/panels-api", Description: "REST API reference for the panels endpoint."},
					},
				},
				{
					Name:        "Colors",
					Target:      "/admin/colorpatterns",
					Description: "Manage color patterns, aliases, and the interactive message crafter.",
					Children: []WebNavItem{
						{Label: "Patterns", Target: "/admin/colorpatterns", Description: "Browse and edit color pattern definitions."},
						{Label: "Patterns API Docs", Target: "/admin/colorpatterns-api", Description: "REST API reference for the color patterns endpoint."},
						{Label: "Aliases", Target: "/admin/color-aliases", Description: "Browse and edit color alias mappings."},
						{Label: "Aliases API Docs", Target: "/admin/color-aliases-api", Description: "REST API reference for color aliases."},
						{Label: "Message Crafter", Target: "/admin/color-tester", Description: "Interactively compose and preview ANSI-colored messages."},
					},
				},
				{
					Name:        "Keywords",
					Target:      "/admin/keywords",
					Description: "Manage command aliases, help topic keywords, and direction shortcuts.",
					Children: []WebNavItem{
						{Label: "View / Edit", Target: "/admin/keywords", Description: "Browse and edit keyword and alias definitions."},
						{Label: "API Docs", Target: "/admin/keywords-api", Description: "REST API reference for the keywords endpoint."},
					},
				},
			},
		},
		{
			Name:        "Docs",
			Description: "Various documentation on how to run this mud as an administrator or content creator.",
			Children: []WebNavItem{
				{
					Name:        "Scripting",
					Target:      "/admin/scripting",
					Description: "Reference documentation for the server-side scripting system.",
					Children: []WebNavItem{
						{Label: "Overview", Target: "/admin/scripting", Description: "Introduction, time periods, and targeting symbols."},
						{Label: "Room Scripts", Target: "/admin/scripting-rooms", Description: "Event hooks for room scripts."},
						{Label: "Mob Scripts", Target: "/admin/scripting-mobs", Description: "Event hooks for mob/NPC scripts."},
						{Label: "Item Scripts", Target: "/admin/scripting-items", Description: "Event hooks for item scripts."},
						{Label: "Buff Scripts", Target: "/admin/scripting-buffs", Description: "Event hooks for buff/status effect scripts."},
						{Label: "Spell Scripts", Target: "/admin/scripting-spells", Description: "Event hooks for spell scripts."},
						{Label: "Pet Scripts", Target: "/admin/scripting-pets", Description: "Event hooks for pet scripts."},
						{Label: "Object Reference", Target: "/admin/scripting-objects", Description: "API reference for ActorObject, RoomObject, ItemObject, PetObject, PartyObject, and ContainerObject."},
						{Label: "Function Reference", Target: "/admin/scripting-functions", Description: "Global utility, messaging, and panel layout functions."},
						{Label: "REST API", Target: "/admin/scripting-api", Description: "REST endpoints for script validation and schema."},
					},
				},
				{
					Name:        "Coding",
					Target:      "/admin/docs-coding",
					Description: "Reference documentation for engine extension points available to Go module authors.",
					Children: []WebNavItem{
						{Label: "Hooks", Target: "/admin/docs-coding", Description: "Complete reference for all util.Hook extension points in the engine."},
					},
				},
				{
					Name:        "Modules",
					Target:      "/admin/docs-modules",
					Description: "Reference documentation for building Go modules that extend the server.",
					Children: []WebNavItem{
						{Label: "Overview", Target: "/admin/docs-modules", Description: "Lifecycle, full plugin API reference, event catalog, and worked examples."},
					},
				},
				{
					Name:        "Backups",
					Target:      "/admin/docs-backups",
					Description: "Guide to configuring automatic world data backups with Amazon S3.",
					Children: []WebNavItem{
						{Label: "Amazon S3 Setup", Target: "/admin/docs-backups", Description: "Step-by-step guide to setting up S3 backups."},
					},
				},
				{
					Name:        "Hosting",
					Target:      "/admin/docs-aws",
					Description: "Guides for hosting GoMud on cloud providers.",
					Children: []WebNavItem{
						{Label: "Amazon / AWS", Target: "/admin/docs-aws", Description: "Step-by-step guide to launching and configuring an AWS EC2 instance for GoMud."},
					},
				},
			},
		},
	}
	nav = append(nav, defaultRegistrar.navItems...)
	return nav
}
