package web

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/scripting"
)

// handwritten interface declarations for the engine's script object types.
// These correspond to the Go structs ScriptActor, ScriptRoom, ScriptItem,
// ScriptPet, and the schema-driven ContainerObject.
const engineObjectInterfaces = `
declare interface ActorObject {
    UserId(): number;
    InstanceId(): number;
    MobTypeId(): number;
    ShorthandId(): string;
    GetLevel(): number;
    GetStat(statName: string): number;
    GetStatMod(statModName: string): number;
    GetRace(): string;
    GetTrueRace(): string;
    GetSize(): string;
    IsFormChanged(): boolean;
    ApplyFormChange(raceId: number): boolean;
    RevertFormChange(): boolean;
    SendText(msg: string): void;
    GetCharacterName(wrapInTags: boolean): string;
    SetCharacterName(newName: string): void;
    GetDescription(): string;
    SetTempData(key: string, value: any): void;
    GetTempData(key: string): any;
    SetMiscCharacterData(key: string, value: any): void;
    GetMiscCharacterData(key: string): any;
    GetMiscCharacterDataKeys(...prefixMatches: string[]): string[];
    GetRoomId(): number;
    MoveRoom(roomId: number): void;
    GetHealth(): number;
    GetHealthMax(): number;
    GetHealthPct(): number;
    GetHealthAppearance(): string;
    SetHealth(amt: number): void;
    AddHealth(amt: number): number;
    GetMana(): number;
    GetManaMax(): number;
    GetManaPct(): number;
    AddMana(amt: number): number;
    GetActionPoints(): number;
    GetActionPointsMax(): number;
    GetGold(): number;
    GetBank(): number;
    AddGold(amount: number, bankAmount?: number): void;
    GetAlignment(): number;
    GetAlignmentName(): string;
    ChangeAlignment(alignmentChange: number): void;
    HasBuff(buffId: number): boolean;
    GiveBuff(buffId: number, source: string): void;
    RemoveBuff(buffId: number): boolean;
    HasBuffFlag(buffFlag: string): boolean;
    CancelBuffWithFlag(buffFlag: string): boolean;
    GiveItem(item: ItemObject | number): void;
    TakeItem(item: ItemObject): void;
    GetWornItems(): ItemObject[];
    GetWornItem(slot: string): ItemObject | null;
    GetBackpackItems(): ItemObject[];
    FindInBackpack(itemName: string): ItemObject | null;
    FindOnBody(itemName: string): ItemObject | null;
    HasItemId(itemId: number, excludeWorn?: boolean): boolean;
    UpdateItem(item: ItemObject): void;
    IsTameable(): boolean;
    IsCharmed(userId?: number): boolean;
    IsInCombat(): boolean;
    IsHome(): boolean;
    IsDowned(): boolean;
    IsAggro(actor: ActorObject): boolean;
    Command(cmd: string, waitSeconds?: number): void;
    CommandFlagged(cmd: string, flags: number, waitSeconds?: number): void;
    GetSkillLevel(skillName: string): number;
    GetAllSkills(): {[skill: string]: number};
    TrainSkill(skillName: string, level: number): boolean;
    HasSpell(spellId: string): boolean;
    LearnSpell(spellId: string): boolean;
    UnLearnSpell(spellId: string): boolean;
    DisableSpell(spellId: string): boolean;
    EnableSpell(spellId: string): boolean;
    GetSpells(): {[spellId: string]: number};
    HasQuest(questId: string): boolean;
    GiveQuest(questId: string): void;
    IsQuestDone(questToken: string): boolean;
    ClearQuestToken(questToken: string): void;
    GetParty(excludeSelf?: boolean): PartyObject;
    GetPartyPresent(excludeSelf?: boolean): PartyObject;
    GetPartyMissing(): PartyObject;
    GetMobKills(mobId: number): number;
    GetRaceKills(race: string): number;
    GetCharmCount(): number;
    GetMaxCharmCount(): number;
    GetCharmedUserId(): number;
    CharmSet(userId: number, charmRounds: number, onRevertCommand?: string): void;
    CharmRemove(): void;
    CharmExpire(): void;
    GetTameMastery(): {[mobId: number]: number};
    SetTameMastery(mobId: number, skillLevel: number): void;
    GetChanceToTame(target: ActorObject): number;
    GetTrainingPoints(): number;
    GiveTrainingPoints(count: number): void;
    GetStatPoints(): number;
    GiveStatPoints(count: number): void;
    GetExperience(): number;
    GrantXP(amount: number, reason: string): void;
    GetExtraLives(): number;
    GiveExtraLife(): void;
    GetDefense(): number;
    GetGearValue(): number;
    GetCarryCapacity(): number;
    GetAdjectives(): string[];
    HasAdjective(adj: string): boolean;
    SetAdjective(adj: string, addIt: boolean): void;
    GetCooldown(tag: string): number;
    TryCooldown(tag: string, period: string): boolean;
    GetSetting(name: string): string;
    SetSetting(name: string, value: string): void;
    TimerSet(name: string, period: string): void;
    TimerExpired(name: string): boolean;
    TimerExists(name: string): boolean;
    SetResetRoomId(roomId: number): void;
    AddEventLog(category: string, message: string): void;
    MarkVisitedRoom(...roomIds: number[]): void;
    MarkVisitedZone(zoneName: string): void;
    GetZoneVisitProgress(zoneName: string): {visited: number, total: number, percent: number};
    Pathing(): boolean;
    PathingAtWaypoint(): boolean;
    Sleep(seconds: number): void;
    GetLastInputRound(): number;
    PlaySound(soundId: string, category: string): void;
    PlayMusic(musicFileOrId: string): void;
    Uncurse(): ItemObject[];
    GetPet(): PetObject | null;
}

declare interface RoomObject {
    RoomId(): number;
    RoomIdSource(): number;
    GetTitle(): string;
    SetTitle(title: string): void;
    GetDescription(): string;
    SetDescription(desc: string): void;
    GetZone(): string;
    SetTempData(key: string, value: any): void;
    GetTempData(key: string): any;
    SetPermData(key: string, value: any): void;
    GetPermData(key: string): any;
    GetItems(): ItemObject[];
    GetStashItems(): ItemObject[];
    DestroyItem(item: ItemObject): void;
    SpawnItem(itemId: number, inStash: boolean): void;
    RepeatSpawnItem(itemId: number, roundFrequency: number, containerName?: string): boolean;
    GetMobs(mobId?: number): ActorObject[];
    GetMob(mobId: number, createIfMissing?: boolean): ActorObject | null;
    GetPlayers(): ActorObject[];
    GetAllActors(): ActorObject[];
    GetContainers(): ContainerObject[];
    GetContainer(name: string): ContainerObject | null;
    SpawnMob(mobId: number): ActorObject | null;
    SpawnTempContainer(name: string, duration: string, lockDifficulty: number, ...trapBuffIds: number[]): string;
    AddTemporaryExit(exitNameSimple: string, exitNameFancy: string, exitRoomId: number, expiresTimeString: string): boolean;
    RemoveTemporaryExit(exitNameSimple: string, exitNameFancy: string, exitRoomId: number): boolean;
    IsLocked(exitName: string): boolean;
    SetLocked(exitName: string, lockIt: boolean): void;
    HasTag(tag: string): boolean;
    SetTag(tag: string): void;
    UnsetTag(tag: string): void;
    HasMutator(mutName: string): boolean;
    AddMutator(mutName: string): void;
    RemoveMutator(mutName: string): void;
    GetExits(): {Name: string, RoomId: number, Secret: boolean, Lock: any, temporary: boolean}[];
    HasQuest(questId: string, partyUserId?: number): number[];
    MissingQuest(questId: string, partyUserId?: number): number[];
    SendText(msg: string, ...excludeUserIds: number[]): void;
    SendTextToExits(msg: string, isQuiet: boolean, ...excludeUserIds: number[]): void;
    GetGold(): number;
    AddGold(amount: number): void;
    RemoveGold(amount: number): number;
    GetNouns(): {[noun: string]: string};
    AddNoun(noun: string, description: string): void;
    RemoveNoun(noun: string): void;
    GetSigns(): string[];
    AddSign(text: string, userId: number, days: number): boolean;
    MobCount(): number;
    PlayerCount(): number;
    GetVisibility(): number;
    IsCalm(): boolean;
    IsPvp(): boolean;
    IsBank(): boolean;
    IsEphemeral(): boolean;
    AreMobsAttacking(userId: number): boolean;
    ArePlayersAttacking(userId: number): boolean;
    HasRecentVisitors(): boolean;
    PlaySound(soundId: string, category: string, ...excludeUserIds: number[]): void;
}

declare interface ItemObject {
    ItemId(): number;
    ShorthandId(): string;
    Name(simpleVersion?: boolean): string;
    NameSimple(): string;
    NameComplex(): string;
    Rename(newName: string, displayNameOrStyle?: string): void;
    GetDescription(): string;
    Redescribe(newDescription: string): void;
    GetType(): string;
    GetSubtype(): string;
    GetValue(): number;
    GetElement(): string;
    GetQuestToken(): string;
    GetBuffIds(): number[];
    GetWornBuffIds(): number[];
    GetStatMods(): {[stat: string]: number};
    GetDamageReduction(): number;
    GetDamage(): {attacks: number, diceCount: number, diceSides: number, bonusDamage: number, diceRoll: string};
    GetBreakChance(): number;
    GetKeyLockId(): string;
    GetUsesLeft(): number;
    SetUsesLeft(amount: number): number;
    AddUsesLeft(amount: number): number;
    GetLastUsedRound(): number;
    MarkLastUsed(clear?: boolean): number;
    IsCursed(): boolean;
    HasUses(): boolean;
    IsWearable(): boolean;
    IsWeapon(): boolean;
    SetTempData(key: string, value: any): void;
    GetTempData(key: string): any;
}

declare interface PetObject {
    Name(): string;
    NameSimple(): string;
    SetName(name: string): void;
    Type(): string;
    Level(): number;
    Food(): string;
    FoodLevel(): number;
    Feed(): void;
    Starve(): void;
    GetStatMod(statName: string): number;
    GetCapacity(): number;
    ItemCount(): number;
    IsMissing(): boolean;
    GoMissing(rounds: number): void;
    HasScript(): boolean;
    GetItems(): ItemObject[];
    FindItem(itemName: string): ItemObject | null;
    StoreItem(item: ItemObject): boolean;
    RemoveItem(item: ItemObject): boolean;
    GetBuffIds(): number[];
}

declare interface PartyObject {
    GetMembers(): ActorObject[];
    SendText(msg: string): void;
    SetResetRoomId(roomId: number): void;
    GiveQuest(questId: string): void;
    AddGold(amount: number, bankAmount?: number): void;
    AddHealth(amt: number): void;
    AddMana(amt: number): void;
    Command(cmd: string, waitSeconds?: number): void;
    TrainSkill(skillName: string, level: number): void;
    MoveRoom(roomId: number): void;
    AddEventLog(category: string, message: string): void;
    GiveBuff(buffId: number, source: string): void;
    CancelBuffWithFlag(buffFlag: string): void;
    RemoveBuff(buffId: number): void;
    ChangeAlignment(alignmentChange: number): void;
    LearnSpell(spellId: string): void;
    SetHealth(amt: number): void;
    SetAdjective(adj: string, addIt: boolean): void;
    GiveTrainingPoints(count: number): void;
    GiveStatPoints(count: number): void;
    GiveExtraLife(): void;
    GrantXP(amount: number, reason: string): void;
    TimerSet(name: string, period: string): void;
    MarkVisitedRoom(...roomIds: number[]): void;
    MarkVisitedZone(zoneName: string): void;
}

declare interface ContainerObject {
    Name(): string;
    HasLock(): boolean;
    IsLocked(): boolean;
    Lock(): void;
    Unlock(): void;
    GetLockDifficulty(): number;
    GetTrapBuffIds(): number[];
    GetItems(): ItemObject[];
    FindItem(itemName: string): ItemObject | null;
    AddItem(item: ItemObject): boolean;
    RemoveItem(item: ItemObject): boolean;
    GetGold(): number;
    AddGold(amount: number): void;
    RemoveGold(amount: number): number;
    Count(itemId: number): number;
    IsTemporary(): boolean;
    GetDespawnRound(): number;
    Exists(): boolean;
}

declare interface CommandEventDetails {
    sourceId: number;
    sourceType: string;
}

declare interface GiveEventDetails {
    sourceId: number;
    sourceType: string;
    item: ItemObject | null;
    gold: number;
}

declare interface ShowEventDetails {
    sourceId: number;
    sourceType: string;
    item: ItemObject | null;
}

declare interface AskEventDetails {
    sourceId: number;
    sourceType: string;
    askText: string;
}

declare interface HurtEventDetails {
    sourceId: number;
    sourceType: string;
    damage: number;
    crit: boolean;
}

declare interface DieEventDetails {
    sourceId: number;
    sourceType: string;
    attackerCount: number;
}

declare interface PathEventDetails {
    sourceId: number;
    sourceType: string;
    status: string;
}

declare interface LevelEventDetails {
    newLevel: number;
    previousLevel: number;
    levelsGained: number;
    trainingPoints: number;
    statPoints: number;
    livesGained: number;
}

declare interface PanelLayoutObject {
    /** Adds a new vertical slot (column) to the layout. Returns the slot for adding rows. */
    AddSlot(): SlotObject;
    /** Returns the named panel for configuration and data population. Throws if the id does not exist. */
    Panel(id: string): PanelObject;
    /** Renders the entire layout into a terminal string ready to send to a player. */
    Render(): string;
}

declare interface SlotObject {
    /** Adds a horizontal row of panels (by id) to this slot. Returns the slot for chaining. */
    AddRow(ids: string[]): SlotObject;
}

declare interface PanelObject {
    /** Sets the panel's title string (used verbatim in the top border). May contain ANSI tags. Returns the panel for chaining. */
    SetTitle(title: string): PanelObject;
    /** Sets a fixed visual width that all labels are padded to. Returns the panel for chaining. */
    SetLabelWidth(w: number): PanelObject;
    /** Sets the panel's total border-inclusive width. Returns the panel for chaining. */
    SetWidth(w: number): PanelObject;
    /** Overrides the border character set ("single", "double", "rounded", or an 8-rune literal). Returns the panel for chaining. */
    SetCharset(name: string): PanelObject;
    /** Sets the number of label+value pairs per line (1 or 2). Returns the panel for chaining. */
    SetColumns(n: number): PanelObject;
    /** Sets the spaces between columns when using 2-column mode. Returns the panel for chaining. */
    SetColumnGap(n: number): PanelObject;
    /** Appends a label+value row. fullLabel is used when it fits; shortLabel is the fallback. Returns the panel for chaining. */
    Add(fullLabel: string, shortLabel: string, value: string): PanelObject;
    /** Appends a label+value row, wrapping the value when it exceeds wrapWidth. Returns the panel for chaining. */
    AddWithWrapWidth(fullLabel: string, shortLabel: string, value: string, wrapWidth: number): PanelObject;
    /** Appends an empty spacer row for visual separation. Returns the panel for chaining. */
    AddBlank(): PanelObject;
}

declare interface MatchResult {
    found: boolean;
    exact: string;
    close: string;
}

declare interface GameDate {
    Calendar: string;
    RoundNumber: number;
    RoundsPerDay: number;
    NightHoursPerDay: number;
    Year: number;
    Month: number;
    Week: number;
    Day: number;
    Hour: number;
    Hour24: number;
    Minute: number;
    MinuteFloat: number;
    AmPm: string;
    Night: boolean;
    DayStart: number;
    NightStart: number;
    DuskHours: number;
    SunCount: number;
    MoonCount: number;
}
`

// goTypeToTS converts a schema type string to its TypeScript equivalent.
func goTypeToTS(t string) string {
	switch t {
	case "number", "string", "boolean", "void", "any", "object":
		return t
	case "number[]":
		return "number[]"
	case "string[]":
		return "string[]"
	case "boolean[]":
		return "boolean[]"
	case "ActorObject[]":
		return "ActorObject[]"
	case "ItemObject[]":
		return "ItemObject[]"
	case "number|string":
		return "number | string"
	case "varies":
		return "ActorObject | ActorObject[] | string"
	case "RoomObject":
		return "RoomObject"
	case "ActorObject":
		return "ActorObject"
	case "ItemObject":
		return "ItemObject"
	case "PetObject":
		return "PetObject"
	case "PartyObject":
		return "PartyObject"
	case "ContainerObject":
		return "ContainerObject"
	case "PanelLayoutObject":
		return "PanelLayoutObject"
	default:
		// Pass through unknown types (e.g. union types already written as TS)
		return t
	}
}

// buildParamList renders a TypeScript parameter list from a slice of ScriptFuncParam.
func buildParamList(params []scripting.ScriptFuncParam) string {
	parts := make([]string, 0, len(params))
	for _, p := range params {
		name := p.Name
		tsType := goTypeToTS(p.Type)
		// Variadic params use "..." prefix in the name field
		if strings.HasPrefix(name, "...") {
			parts = append(parts, fmt.Sprintf("...%s: %s[]", name[3:], goTypeToTS(p.Type)))
			continue
		}
		// Optional params use "?" suffix in the name field
		optional := strings.HasSuffix(name, "?")
		if optional {
			name = name[:len(name)-1]
		}
		if optional {
			parts = append(parts, fmt.Sprintf("%s?: %s", name, tsType))
		} else {
			parts = append(parts, fmt.Sprintf("%s: %s", name, tsType))
		}
	}
	return strings.Join(parts, ", ")
}

// GET /admin/api/v1/scripting/types.d.ts?type=<scriptType>
func apiV1GetScriptingTypesDts(w http.ResponseWriter, r *http.Request) {
	scriptType := r.URL.Query().Get("type")
	schema := scripting.GetScriptFunctionsSchema()

	var sb strings.Builder

	// Engine object interfaces (hand-authored, stable)
	sb.WriteString(engineObjectInterfaces)
	sb.WriteByte('\n')

	// Global engine function declarations (generated from schema)
	sb.WriteString("// Global engine functions\n")
	for _, fn := range schema.EngineFunctions {
		params := buildParamList(fn.Params)
		retType := "void"
		if fn.ReturnType != "" {
			retType = goTypeToTS(fn.ReturnType)
		}
		if fn.Description != "" {
			sb.WriteString(fmt.Sprintf("/** %s */\n", fn.Description))
		}
		sb.WriteString(fmt.Sprintf("declare function %s(%s): %s;\n", fn.Name, params, retType))
	}

	// Script-type-specific event callback declarations
	if scriptType != "" {
		typeDef, ok := schema.ScriptTypes[scriptType]
		if ok {
			sb.WriteString(fmt.Sprintf("\n// Event callbacks for script type: %s\n", scriptType))
			for _, fn := range typeDef.Functions {
				if fn.Dynamic != nil {
					placeholder := fn.Dynamic.Placeholder
					exampleName := strings.ReplaceAll(fn.Name, placeholder, "commandName")
					params := buildParamList(fn.Params)
					retType := "boolean | void"
					if fn.ReturnType != "" {
						retType = goTypeToTS(fn.ReturnType)
					}
					if fn.Description != "" {
						sb.WriteString(fmt.Sprintf("/** %s (dynamic: replace %s with the actual command word) */\n", fn.Description, placeholder))
					}
					sb.WriteString(fmt.Sprintf("declare function %s(%s): %s;\n", exampleName, params, retType))
					continue
				}
				params := buildParamList(fn.Params)
				retType := "void"
				if fn.ReturnType != "" {
					retType = goTypeToTS(fn.ReturnType)
				}
				if fn.Description != "" {
					sb.WriteString(fmt.Sprintf("/** %s */\n", fn.Description))
				}
				sb.WriteString(fmt.Sprintf("declare function %s(%s): %s;\n", fn.Name, params, retType))
			}

			// Emit @callback JSDoc typedefs for every non-dynamic callback.
			// When the user writes a function with a matching name, Monaco's
			// JS checker resolves the @callback type and infers parameter types
			// inside the function body, enabling member completions on typed params.
			sb.WriteString(fmt.Sprintf("\n// JSDoc callback typedefs for script type: %s\n", scriptType))
			for _, fn := range typeDef.Functions {
				if fn.Dynamic != nil {
					continue
				}
				cbName := strings.ToUpper(fn.Name[:1]) + fn.Name[1:] + "Callback"
				sb.WriteString(fmt.Sprintf("/**\n * @callback %s\n", cbName))
				for _, p := range fn.Params {
					paramName := p.Name
					if strings.HasPrefix(paramName, "...") {
						paramName = paramName[3:]
					}
					if strings.HasSuffix(paramName, "?") {
						paramName = paramName[:len(paramName)-1]
					}
					tsType := goTypeToTS(p.Type)
					sb.WriteString(fmt.Sprintf(" * @param {%s} %s - %s\n", tsType, paramName, p.Description))
				}
				retType := "void"
				if fn.ReturnType != "" {
					retType = goTypeToTS(fn.ReturnType)
				}
				sb.WriteString(fmt.Sprintf(" * @returns {%s}\n */\n", retType))
			}
		}
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	fmt.Fprint(w, sb.String())
}
