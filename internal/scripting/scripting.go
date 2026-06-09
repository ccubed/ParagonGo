package scripting

import (
	"errors"
	"time"

	"github.com/GoMudEngine/GoMud/internal/colorpatterns"
)

var (
	ErrEventNotFound = errors.New(`event not found`)
)

type TextWrapperStyle struct {
	cache             string
	Fg                string // ansi class name for foreground
	Bg                string // ansi class name for background
	ColorPattern      string // optional color pattern
	colorPatternStyle colorpatterns.ColorizeStyle
}

func (t *TextWrapperStyle) Set(fg string, bg string, colorpattern string, colorStyle ...colorpatterns.ColorizeStyle) {
	t.Fg = fg
	t.Bg = bg
	t.ColorPattern = colorpattern
	if len(colorStyle) > 0 {
		t.colorPatternStyle = colorStyle[0]
	} else {
		t.colorPatternStyle = colorpatterns.Default
	}
}

func (t *TextWrapperStyle) Reset() {
	t.cache = ``
	t.Fg = ``
	t.Bg = ``
	t.ColorPattern = ``
}

func (t *TextWrapperStyle) Empty() bool {
	return t.Fg == `` && t.Bg == `` && t.ColorPattern == ``
}

func (t *TextWrapperStyle) AnsiClass() string {
	if t.cache != `` {
		return t.cache
	}

	if t.Fg != `` {
		t.cache += ` fg="` + t.Fg + `"`
	}

	if t.Bg != `` {
		t.cache += ` bg="` + t.Bg + `"`
	}

	return t.cache
}

func (t *TextWrapperStyle) Wrap(str string) string {
	if !t.Empty() {

		if t.ColorPattern != `` {
			str = colorpatterns.ApplyColorPattern(str, t.ColorPattern, t.colorPatternStyle)
		}

		if ac := t.AnsiClass(); ac != `` {
			str = `<ansi ` + t.AnsiClass() + `>` + str + `</ansi>`
		}
	}
	return str
}

var (
	errNoScript = errors.New("no script")
	errTimeout  = errors.New("script timeout")

	// If non empty, will wrap output to users or rooms in this style
	userTextWrap = TextWrapperStyle{}
	roomTextWrap = TextWrapperStyle{}

	// scriptHotReload enables mtime-based cache invalidation on every VM
	// lookup. Off by default; enable in development or admin environments.
	scriptHotReload bool
)

func Setup(scriptLoadTimeoutMs int, scriptRoomTimeoutMs int) {

	scriptLoadTimeout = time.Duration(scriptLoadTimeoutMs) * time.Millisecond

	t := time.Duration(scriptRoomTimeoutMs) * time.Millisecond
	scriptRoomTimeout = t
	scriptBuffTimeout = t
	scriptItemTimeout = t
	scriptMobTimeout = t
	scriptPetTimeout = t
	scriptSpellTimeout = t
}

// SetHotReload enables or disables mtime-based script hot-reload.
// When enabled, each VM lookup checks whether the script file on disk is newer
// than the cached VM and reloads automatically if so.
func SetHotReload(enabled bool) {
	scriptHotReload = enabled
}

func setAllScriptingFunctions(vm registrar) {
	setMessagingFunctions(vm)
	setRoomFunctions(vm)
	setActorFunctions(vm)
	setSpellFunctions(vm)
	setItemFunctions(vm)
	setPetFunctions(vm)
	setUtilFunctions(vm)
	setModuleFunctions(vm)
	setPanelFunctions(vm)
}

type ValidationResult struct {
	Valid  bool   `json:"valid"`
	Error  string `json:"error,omitempty"`
	Line   int    `json:"line,omitempty"`
	Column int    `json:"column,omitempty"`
}

// ValidateScript compiles the given script source without running it and
// reports syntax errors. label is used in error messages. The optional lang
// selects the engine; it defaults to JavaScript for backward compatibility.
func ValidateScript(label string, script string, lang ...ScriptLang) ValidationResult {
	l := LangJS
	if len(lang) > 0 && lang[0] != LangNone {
		l = lang[0]
	}
	switch l {
	case LangLua:
		return validateLuaScript(label, script)
	default:
		return validateGojaScript(label, script)
	}
}

func PruneVMs(forceClear ...bool) {

	if len(forceClear) > 0 && forceClear[0] {
		ClearRoomVMs()
		ClearMobVMs()
		ClearBuffVMs()
		ClearItemVMs()
		ClearPetVMs()
		ClearSpellVMs()
	} else {
		PruneRoomVMs()
		PruneMobVMs()
		PruneBuffVMs()
		PruneItemVMs()
		PruneSpellVMs()
	}

}
