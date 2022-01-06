package loghandler

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/apex/log"
	"github.com/fatih/color"
	colorable "github.com/mattn/go-colorable"
)

// Based on github.com/apex/log/handlers/cli

// Colors mapping.
var Colors = [...]*color.Color{
	log.DebugLevel: color.New(),
	log.InfoLevel:  color.New(color.FgWhite),
	log.WarnLevel:  color.New(color.FgYellow),
	log.ErrorLevel: color.New(color.FgRed),
	log.FatalLevel: color.New(color.FgRed),
}

// Strings mapping.
var Strings = [...]string{
	log.DebugLevel: "•",
	log.InfoLevel:  "•",
	log.WarnLevel:  "△",
	log.ErrorLevel: "✗",
	log.FatalLevel: "✘",
}

// Handler implementation.
type Handler struct {
	mu      sync.Mutex
	Writer  io.Writer
	Padding int
	Prefix  string
}

// New handler.
func New(w io.Writer) *Handler {
	if f, ok := w.(*os.File); ok {
		return &Handler{
			Writer:  colorable.NewColorable(f),
			Padding: 3,
		}
	}

	return &Handler{
		Writer:  w,
		Padding: 3,
	}
}

// HandleLog implements log.Handler.
func (h *Handler) HandleLog(e *log.Entry) error {
	color := Colors[e.Level]
	level := Strings[e.Level]
	names := e.Fields.Names()

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.Prefix != "" {
		fmt.Fprint(h.Writer, h.Prefix, " ")
	}
	color.Fprintf(h.Writer, "%s %-40s", level, e.Message)

	for _, name := range names {
		if name == "source" {
			continue
		}
		fmt.Fprintf(h.Writer, " %s=%v", color.Sprint(name), e.Fields.Get(name))
	}

	fmt.Fprintln(h.Writer)

	return nil
}
