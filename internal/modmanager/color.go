package modmanager

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

// ANSI escape sequences. All output is gated through colorEnabled so that
// piped / non-terminal output stays plain text.
const (
	ansiReset     = "\033[0m"
	ansiBold      = "\033[1m"
	ansiDim       = "\033[2m"
	ansiCyan      = "\033[36m"
	ansiBoldCyan  = "\033[1;36m"
	ansiGreen     = "\033[32m"
	ansiBoldGreen = "\033[1;32m"
	ansiYellow    = "\033[33m"
	ansiRed       = "\033[31m"
	ansiBoldRed   = "\033[1;31m"
	ansiBlue      = "\033[34m"
	ansiBoldBlue  = "\033[1;34m"
	ansiMagenta   = "\033[35m"
	ansiWhite     = "\033[97m"
	ansiGray      = "\033[90m"
)

// colorEnabled is true when stdout is an interactive terminal that likely
// supports ANSI escape sequences.
var colorEnabled = term.IsTerminal(int(os.Stdout.Fd()))

// col wraps s in the given ANSI escape code when color is enabled.
func col(code, s string) string {
	if !colorEnabled {
		return s
	}
	return code + s + ansiReset
}

func bold(s string) string    { return col(ansiBold, s) }
func cyan(s string) string    { return col(ansiBoldCyan, s) }
func green(s string) string   { return col(ansiBoldGreen, s) }
func yellow(s string) string  { return col(ansiYellow, s) }
func red(s string) string     { return col(ansiBoldRed, s) }
func blue(s string) string    { return col(ansiBoldBlue, s) }
func magenta(s string) string { return col(ansiMagenta, s) }
func gray(s string) string    { return col(ansiGray, s) }
func dimStr(s string) string  { return col(ansiDim, s) }
func white(s string) string   { return col(ansiWhite, s) }

// statusColor returns the module status string wrapped in an appropriate color.
func statusColor(status string) string {
	switch status {
	case "installed":
		return green(status)
	case "update avail":
		return yellow(status)
	default:
		return dimStr(status)
	}
}

// header renders a prominent section header.
func header(title string) string {
	return cyan(title)
}

// codeSnippet renders a shell command in a distinct style.
func codeSnippet(s string) string {
	return col(ansiGray, "  "+s)
}

// printError prints a formatted error line to stderr.
func printError(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(os.Stderr, red("error: ")+msg)
}

// printWarning prints a formatted warning line to stderr.
func printWarning(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(os.Stderr, yellow("warning: ")+msg)
}

// printStep prints a progress step line.
func printStep(format string, args ...any) {
	fmt.Printf(col(ansiCyan, "  -> ")+format+"\n", args...)
}

// printSuccess prints a success message.
func printSuccess(format string, args ...any) {
	fmt.Printf(green("  ✓ ")+format+"\n", args...)
}

// divider returns a horizontal rule of the given width, dimmed.
func divider(width int) string {
	return dimStr(strings.Repeat("─", width))
}

// padRight pads s with spaces on the right so the visible width equals width.
// It accounts for ANSI escape sequences which do not occupy visible columns.
func padRight(s string, width int) string {
	visible := visibleLen(s)
	if visible >= width {
		return s
	}
	return s + strings.Repeat(" ", width-visible)
}

// visibleLen returns the number of visible (non-ANSI) characters in s.
func visibleLen(s string) int {
	inEscape := false
	count := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\033' {
			inEscape = true
			continue
		}
		if inEscape {
			if s[i] == 'm' {
				inEscape = false
			}
			continue
		}
		count++
	}
	return count
}
