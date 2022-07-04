package logging

import (
	"fmt"

	"github.com/rs/zerolog"
)

// consoleDefaultFormatLevel is copied from zerolog/console.go to modify the
// names and colors used for levels.
func consoleFormatLevel(i interface{}) string {
	noColor := !isTerminal()
	l, ok := i.(string)
	if !ok {
		return fmt.Sprintf("%v", i)
	}

	switch l {
	case zerolog.LevelTraceValue:
		l = colorize("TRACE", colorMagenta, noColor)
	case zerolog.LevelDebugValue:
		l = colorize("DEBUG", colorYellow, noColor)
	case zerolog.LevelInfoValue:
		l = colorize("INFO ", colorGreen, noColor)
	case zerolog.LevelWarnValue:
		l = colorize("WARN ", colorRed, noColor)
	case zerolog.LevelErrorValue:
		l = colorize(colorize("ERROR", colorRed, noColor), colorBold, noColor)
	case zerolog.LevelFatalValue:
		l = colorize(colorize("FATAL", colorRed, noColor), colorBold, noColor)
	case zerolog.LevelPanicValue:
		l = colorize(colorize("PANIC", colorRed, noColor), colorBold, noColor)
	default:
		l = colorize("?????", colorBold, noColor)
	}
	return l
}

// nolint:unused,deadcode,varcheck
const (
	colorBlack = iota + 30
	colorRed
	colorGreen
	colorYellow
	colorBlue
	colorMagenta
	colorCyan
	colorWhite

	colorBold     = 1
	colorDarkGray = 90
)

// colorize returns the string s wrapped in ANSI code c, unless disabled is true.
// Copied from zerolog/console.go
func colorize(s interface{}, c int, disabled bool) string {
	if disabled {
		return fmt.Sprintf("%s", s)
	}
	return fmt.Sprintf("\x1b[%dm%v\x1b[0m", c, s)
}
