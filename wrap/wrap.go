package wrap

import (
	"bytes"
	"unicode"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
	"github.com/muesli/reflow/internal/statemachine"
)

var (
	defaultNewline  = []rune{'\n'}
	defaultTabWidth = 4
)

type Wrap struct {
	Limit         int
	Newline       []rune
	KeepNewlines  bool
	PreserveSpace bool
	TabWidth      int

	buf             *bytes.Buffer
	lineLen         int
	state           statemachine.StateMachine
	forcefulNewline bool
}

// NewWriter returns a new instance of a wrapping writer, initialized with
// default settings.
func NewWriter(limit int) *Wrap {
	return &Wrap{
		Limit:        limit,
		Newline:      defaultNewline,
		KeepNewlines: true,
		// Keep whitespaces following a forceful line break. If disabled,
		// leading whitespaces in a line are only kept if the line break
		// was not forceful, meaning a line break that was already present
		// in the input
		PreserveSpace: false,
		TabWidth:      defaultTabWidth,

		buf: &bytes.Buffer{},
	}
}

// Bytes is shorthand for declaring a new default Wrap instance,
// used to immediately wrap a byte slice.
func Bytes(b []byte, limit int) []byte {
	f := NewWriter(limit)
	_, _ = f.Write(b)

	return f.Bytes()
}

// String is shorthand for declaring a new default Wrap instance,
// used to immediately wrap a string.
func String(s string, limit int) string {
	return string(Bytes([]byte(s), limit))
}

func (w *Wrap) addNewLine(c rune) {
	// fmt.Println("  newline")
	_, _ = w.buf.WriteRune(c)
	w.lineLen = 0
}

func (w *Wrap) Write(b []byte) (int, error) {
	// guard against negative TabWidth
	if w.TabWidth < 0 {
		w.TabWidth = 0
	}

	if w.Limit <= 0 {
		// pass-through if no limit is set
		return w.buf.Write(b)
	}

	i := 0
	for i < len(b) {
		// iterate runewise over the input byte slice w/o allocating
		c, cw := utf8.DecodeRune(b[i:])
		var step statemachine.StateTransition
		nextI := i + cw
		for j := i; j < nextI; j++ {
			step = w.state.Next(b[j])
		}
		i = nextI

		w.stepState(step, c)
	}

	return len(b), nil
}

func (w *Wrap) WriteString(s string) (int, error) {
	// guard against negative TabWidth
	if w.TabWidth < 0 {
		w.TabWidth = 0
	}

	if w.Limit <= 0 {
		// pass-through if no limit is set
		return w.buf.WriteString(s)
	}

	// iterate the bytes of the string, without allocating
	for i := 0; i < len(s); {
		c, charWidth := utf8.DecodeRuneInString(s[i:])
		nextI := i + charWidth

		// update state machine by stepping over the current character, one byte at a time
		var step statemachine.StateTransition
		for j := i; j < nextI; j++ {
			step = w.state.Next(s[j])
		}

		w.stepState(step, c)
		i = nextI
	}

	return len(s), nil
}

func (w *Wrap) stepState(step statemachine.StateTransition, c rune) {
	if !step.IsPrinting() {
		// echo through non-printable characters
		_, _ = w.buf.WriteRune(c)
		return
	}

	if inGroup(w.Newline, c) && !w.KeepNewlines {
		return
	}

	if c == '\t' {
		for i := 0; i < w.TabWidth; i++ {
			w.stepPrintableState(' ')
		}
	} else {
		w.stepPrintableState(c)
	}
}

func (w *Wrap) stepPrintableState(c rune) {
	// fmt.Println("stepPrintableState\t", strconv.Quote(string(c)))

	if inGroup(w.Newline, c) {
		w.addNewLine(c)
		w.forcefulNewline = false
		return
	} else {
		width := runewidth.RuneWidth(c)

		if !w.PreserveSpace &&
			((w.lineLen == 0 && w.forcefulNewline) || w.lineLen+width > w.Limit) &&
			unicode.IsSpace(c) {
			// skip leading whitespaces on a new line if PreserveSpace == false
			// also skip forcing a newline if the next line would be only a stripped whitespace.
			return
		}

		if w.lineLen > 0 && w.lineLen+width > w.Limit {
			w.addNewLine('\n')
			w.forcefulNewline = true
		} else {
			// clear forceful newline flag if we didn't just force a newline
			w.forcefulNewline = false
		}

		w.lineLen += width
	}

	_, _ = w.buf.WriteRune(c)
}

// Bytes returns the wrapped result as a byte slice.
func (w *Wrap) Bytes() []byte {
	return w.buf.Bytes()
}

// String returns the wrapped result as a string.
func (w *Wrap) String() string {
	return w.buf.String()
}

func inGroup(a []rune, c rune) bool {
	for _, v := range a {
		if v == c {
			return true
		}
	}
	return false
}

// Limited version of wordwrap.Wrap that only counts
// the height of the wrapped message in newlines
type WrapHeight struct {
	Limit         int
	KeepNewlines  bool
	PreserveSpace bool
	TabWidth      int
	Newline       []rune

	// Tracks if the last newline was "forced" (e.g. was inserted due to
	// the line-length limit)
	//
	// This is because `PreserveSpace` is only applied after forced
	// newlines
	forcefulNewline bool
	ansiState       statemachine.StateMachine
	height          int
	lineLen         int // current length of the current line
}

func (h WrapHeight) Height() int {
	return h.height + 1
}

func (w *WrapHeight) Write(b []byte) (int, error) {
	// parse the input byte slice as runes, without allocating
	i := 0
	for i < len(b) {
		c, byteWidth := utf8.DecodeRune(b[i:])
		nextI := i + byteWidth

		// consume all the bytes of this character in the statemachine
		var step statemachine.StateTransition
		for j := i; j < nextI; j++ {
			step = w.ansiState.Next(b[j])
		}

		w.stepState(step, c)
		i = nextI
	}

	return len(b), nil
}

func (w *WrapHeight) WriteString(s string) (int, error) {
	// iterate the bytes of the string, without allocating
	for i := 0; i < len(s); {
		c, charWidth := utf8.DecodeRuneInString(s[i:])
		nextI := i + charWidth

		// update state machine by stepping over the current character, one byte at a time
		var step statemachine.StateTransition
		for j := i; j < nextI; j++ {
			step = w.ansiState.Next(s[j])
		}

		w.stepState(step, c)
		i = nextI
	}

	return len(s), nil
}

func (w *WrapHeight) stepState(step statemachine.StateTransition, c rune) {
	// fmt.Println("stepState\t", strconv.Quote(string(c)), w.height)

	if w.Limit <= 0 {
		// if limit <=0, ignore all special behaviour and just
		// count the printable newlines
		if inGroup(w.Newline, c) && step.IsPrinting() {
			w.height++
		}
		return
	}

	// special case: intercept tabs and treat them as some number of spaces
	if c == '\t' {
		for i := 0; i < w.TabWidth; i++ {
			w.stepState(step, ' ')
		}
		return
	}

	// if we are not printing, totally ignore the character for the stepper calculation
	if !step.IsPrinting() {
		return
	}

	// special case: if !KeepNewlines, drop newlines
	if inGroup(w.Newline, c) {
		if !w.KeepNewlines {
			// if not in KeepNewlines,
			// skip newline characters
			return
		} else {
			w.forcefulNewline = false
			// Otherwise, consume this newline
			w.height++
			w.lineLen = 0
			return
		}
	}

	width := runewidth.RuneWidth(c)

	if !w.PreserveSpace &&
		((w.lineLen == 0 && w.forcefulNewline) || w.lineLen+width > w.Limit) &&
		unicode.IsSpace(c) {
		// skip leading whitespaces on a new line if PreserveSpace == false
		// also skip forcing a newline if the next line would be only a stripped whitespace.
		return
	}

	if w.lineLen > 0 && w.lineLen+width > w.Limit {
		// end of current line
		w.height++
		w.forcefulNewline = true

		// This effectively pre-buffers a line break for the next line, which matches
		// the real implementation. We do this to avoid
		w.lineLen = 0
	} else {
		// clear forceful newline flag if we didn't just force a newline
		w.forcefulNewline = false
	}

	w.lineLen += width
}

func NewHeightWriter(limit int) *WrapHeight {
	return &WrapHeight{
		Limit:         limit,
		Newline:       defaultNewline,
		KeepNewlines:  true,
		PreserveSpace: false,
		TabWidth:      defaultTabWidth,
	}
}

func Height(s string, limit int) int {
	writer := WrapHeight{
		Limit:    limit,
		Newline:  defaultNewline,
		TabWidth: defaultTabWidth,
	}
	writer.WriteString(s)

	return writer.Height()
}

func HeightBytes(b []byte, limit int) int {
	writer := WrapHeight{
		Limit:    limit,
		Newline:  defaultNewline,
		TabWidth: defaultTabWidth,
	}
	writer.Write(b)

	return writer.Height()
}
