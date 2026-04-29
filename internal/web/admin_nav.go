package web

// buildAdminNav returns the full admin navigation, combining hardcoded core
// entries with module-contributed entries.
func buildAdminNav() []WebNavItem {
	nav := []WebNavItem{
		{
			Name:   "Dashboard",
			Target: "/admin/",
		},

		{
			Name: "Lifeforms",
			SubMenus: []WebNavItem{
				{
					Name:   "Users",
					Target: "/admin/users",
					SubItems: []WebNavSub{
						{Label: "View / Edit", Target: "/admin/users"},
						{Label: "API Docs", Target: "/admin/users-api"},
					},
				},
				{
					Name:   "Mobs",
					Target: "/admin/mobs",
					SubItems: []WebNavSub{
						{Label: "View / Edit", Target: "/admin/mobs"},
						{Label: "API Docs", Target: "/admin/mobs-api"},
					},
				},
				{
					Name:   "Races",
					Target: "/admin/races",
					SubItems: []WebNavSub{
						{Label: "View / Edit", Target: "/admin/races"},
						{Label: "API Docs", Target: "/admin/races-api"},
					},
				},
			},
		},

		{
			Name: "Items",
			SubMenus: []WebNavItem{
				{
					Name:   "Items",
					Target: "/admin/items",
					SubItems: []WebNavSub{
						{Label: "View / Edit", Target: "/admin/items"},
						{Label: "View Rankings", Target: "/admin/items-rank"},
						{Label: "API Docs", Target: "/admin/items-api"},
					},
				},
				{
					Name:   "Attack Messages",
					Target: "/admin/items-attack-messages",
					SubItems: []WebNavSub{
						{Label: "View / Edit", Target: "/admin/items-attack-messages"},
						{Label: "API Docs", Target: "/admin/items-attack-messages-api"},
					},
				},
			},
		},
		{
			Name: "Rooms",
			SubMenus: []WebNavItem{
				{
					Name:   "Rooms",
					Target: "/admin/rooms",
					SubItems: []WebNavSub{
						{Label: "View / Edit", Target: "/admin/rooms"},
						{Label: "API Docs", Target: "/admin/rooms-api"},
					},
				},
				{
					Name:   "Mutators",
					Target: "/admin/mutators",
					SubItems: []WebNavSub{
						{Label: "View / Edit", Target: "/admin/mutators"},
						{Label: "API Docs", Target: "/admin/mutators-api"},
					},
				},
			},
		},
		{
			Name:   "Buffs",
			Target: "/admin/buffs",
			SubItems: []WebNavSub{
				{Label: "View / Edit", Target: "/admin/buffs"},
				{Label: "API Docs", Target: "/admin/buffs-api"},
			},
		},
		{
			Name:   "Spells",
			Target: "/admin/spells",
			SubItems: []WebNavSub{
				{Label: "View / Edit", Target: "/admin/spells"},
				{Label: "API Docs", Target: "/admin/spells-api"},
			},
		},
		{
			Name:   "Quests",
			Target: "/admin/quests",
			SubItems: []WebNavSub{
				{Label: "View / Edit", Target: "/admin/quests"},
				{Label: "API Docs", Target: "/admin/quests-api"},
			},
		},
		{
			Name: "Colors",
			SubMenus: []WebNavItem{
				{
					Name:   "Patterns",
					Target: "/admin/colorpatterns",
					SubItems: []WebNavSub{
						{Label: "View / Edit", Target: "/admin/colorpatterns"},
						{Label: "API Docs", Target: "/admin/colorpatterns-api"},
					},
				},
				{
					Name:   "Aliases",
					Target: "/admin/color-aliases",
					SubItems: []WebNavSub{
						{Label: "View / Edit", Target: "/admin/color-aliases"},
						{Label: "API Docs", Target: "/admin/color-aliases-api"},
					},
				},
				{
					Name:   "Message Crafter",
					Target: "/admin/color-tester",
				},
			},
		},
		{
			Name:   "Keywords",
			Target: "/admin/keywords",
			SubItems: []WebNavSub{
				{Label: "View / Edit", Target: "/admin/keywords"},
				{Label: "API Docs", Target: "/admin/keywords-api"},
			},
		},
		{
			Name: "Server",
			SubMenus: []WebNavItem{
				{
					Name:   "Config",
					Target: "/admin/config",
					SubItems: []WebNavSub{
						{Label: "View / Edit", Target: "/admin/config"},
						{Label: "API Docs", Target: "/admin/config-api"},
					},
				},
				{
					Name:   "Stats",
					Target: "/admin/stats",
					SubItems: []WebNavSub{
						{Label: "View", Target: "/admin/stats"},
						{Label: "API Docs", Target: "/admin/stats-api"},
					},
				},
				{
					Name:   "Audio",
					Target: "/admin/audio",
					SubItems: []WebNavSub{
						{Label: "View / Edit", Target: "/admin/audio"},
						{Label: "API Docs", Target: "/admin/audio-api"},
					},
				},
			},
		},
	}
	nav = append(nav, defaultRegistrar.navItems...)
	return nav
}
