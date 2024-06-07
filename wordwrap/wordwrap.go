package wordwrap

import (
	"bytes"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
	"github.com/muesli/reflow/internal/statemachine"
)

var (
	defaultBreakpoints = []rune{'-'}
	defaultNewline     = []rune{'\n'}
)

// WordWrap contains settings and state for customisable text reflowing with
// support for ANSI escape sequences. This means you can style your terminal
// output without affecting the word wrapping algorithm.
type WordWrap struct {
	Limit        int
	Breakpoints  []rune
	Newline      []rune
	KeepNewlines bool
	BreakAnsi    bool

	buf   bytes.Buffer
	space bytes.Buffer

	word               bytes.Buffer
	printableWordWidth int

	lineLen int
	state   statemachine.AnsiState
}

// NewWriter returns a new instance of a word-wrapping writer, initialized with
// default settings.
func NewWriter(limit int) *WordWrap {
	w := DefaultWriter(limit)
	return &w
}

// NewWriter returns a new instance of a word-wrapping writer, initialized with
// default settings.
func DefaultWriter(limit int) WordWrap {
	return WordWrap{
		Limit:        limit,
		Breakpoints:  defaultBreakpoints,
		Newline:      defaultNewline,
		BreakAnsi:    false,
		KeepNewlines: true,
	}
}

// Bytes is shorthand for declaring a new default WordWrap instance,
// used to immediately word-wrap a byte slice.
func Bytes(b []byte, limit int) []byte {
	f := NewWriter(limit)
	_, _ = f.Write(b)
	_ = f.Close()

	return f.Bytes()
}

// String is shorthand for declaring a new default WordWrap instance,
// used to immediately word-wrap a string.
func String(s string, limit int) string {
	return string(Bytes([]byte(s), limit))
}

func (w *WordWrap) addSpace() {
	w.lineLen += w.space.Len()
	_, _ = w.buf.Write(w.space.Bytes())
	w.space.Reset()
}

func (w *WordWrap) addWord() {
	if w.word.Len() > 0 {
		w.addSpace()
		w.lineLen += w.printableWordWidth
		_, _ = w.buf.Write(w.word.Bytes())
		w.word.Reset()
		w.printableWordWidth = 0
	}
}

func (w *WordWrap) addNewLine() {
	if w.BreakAnsi {
		w.state.WriteResetSequence(&w.buf)
	}
	_, _ = w.buf.WriteRune('\n')
	if w.BreakAnsi {
		w.state.WriteRestoreSequence(&w.buf)
	}
	w.lineLen = 0
	w.space.Reset()
}

func inGroup(a []rune, c rune) bool {
	for _, v := range a {
		if v == c {
			return true
		}
	}
	return false
}

func (w *WordWrap) Write(b []byte) (int, error) {
	if w.Limit <= 0 {
		// pass-through if no limit is set
		return w.buf.Write(b)
	}

	if !w.KeepNewlines {
		b = bytes.TrimSpace(b)
	}

	i := 0
	for i < len(b) {
		// iterate runewise over the input byte slice w/o allocating
		c, cw := utf8.DecodeRune(b[i:])
		var step statemachine.StateTransition
		nextI := i + cw
		for j := i; j < nextI; j++ {
			step = w.state.Next(b[j]).StateTransition
		}
		i = nextI

		w.stepState(step, c)
	}

	return len(b), nil
}

func (w *WordWrap) WriteString(s string) (int, error) {
	if w.Limit <= 0 {
		// pass-through if no limit is set
		return w.buf.WriteString(s)
	}

	if !w.KeepNewlines {
		s = strings.TrimSpace(s)
	}

	// iterate the bytes of the string, without allocating
	for i := 0; i < len(s); {
		c, charWidth := utf8.DecodeRuneInString(s[i:])
		nextI := i + charWidth

		// update state machine by stepping over the current character, one byte at a time
		var step statemachine.StateTransition
		for j := i; j < nextI; j++ {
			step = w.state.Next(s[j]).StateTransition
		}

		w.stepState(step, c)
		i = nextI
	}

	return len(s), nil
}

func (w *WordWrap) stepState(step statemachine.StateTransition, c rune) {
	if !step.IsPrinting() {
		// echo through non-printable characters
		_, _ = w.word.WriteRune(c)
		return
	}

	if inGroup(w.Newline, c) && !w.KeepNewlines {
		// if KeepNewlines is false, treat all newlines as spaces
		w.stepPrintableState(' ')
	} else {
		w.stepPrintableState(c)
	}
}

func (w *WordWrap) stepPrintableState(c rune) {
	if inGroup(w.Newline, c) {
		// end of current line
		// see if we can add the content of the space buffer to the current line
		if w.word.Len() == 0 {
			if w.lineLen+w.space.Len() > w.Limit {
				w.lineLen = 0
			} else {
				// preserve whitespace
				_, _ = w.buf.Write(w.space.Bytes())
			}
			w.space.Reset()
		}

		w.addWord()
		w.addNewLine()
	} else if unicode.IsSpace(c) {
		// end of current word
		w.addWord()
		_, _ = w.space.WriteRune(c)
	} else if inGroup(w.Breakpoints, c) {
		// valid breakpoint
		w.addSpace()
		w.addWord()
		_, _ = w.buf.WriteRune(c)
	} else {
		// any other character
		_, _ = w.word.WriteRune(c)
		w.printableWordWidth += runewidth.RuneWidth(c)

		// add a line break if the current word would exceed the line's
		// character limit
		if w.lineLen+w.space.Len()+w.printableWordWidth > w.Limit &&
			w.printableWordWidth < w.Limit {
			w.addNewLine()
		}
	}
}

// Close will finish the word-wrap operation. Always call it before trying to
// retrieve the final result.
func (w *WordWrap) Close() error {
	w.addWord()
	return nil
}

// Bytes returns the word-wrapped result as a byte slice.
func (w *WordWrap) Bytes() []byte {
	return w.buf.Bytes()
}

// String returns the word-wrapped result as a string.
func (w *WordWrap) String() string {
	return w.buf.String()
}
