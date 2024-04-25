package truncate

import (
	"bytes"
	"fmt"
	"io"
	"strconv"

	"strings"

	"github.com/mattn/go-runewidth"
	"github.com/muesli/reflow/ansi"
	"github.com/muesli/reflow/internal/statemachine"
)

type Writer struct {
	width uint
	tail  string

	writer io.Writer
	buf    bytes.Buffer
}

func NewWriter(width uint, tail string) *Writer {
	w := &Writer{
		width: width,
		tail:  tail,
	}
	w.writer = &w.buf
	return w
}

func NewWriterPipe(forward io.Writer, width uint, tail string) *Writer {
	return &Writer{
		width:  width,
		tail:   tail,
		writer: forward,
	}
}

// Bytes is shorthand for declaring a new default truncate-writer instance,
// used to immediately truncate a byte slice.
func Bytes(b []byte, width uint) []byte {
	return BytesWithTail(b, width, []byte(""))
}

// Bytes is shorthand for declaring a new default truncate-writer instance,
// used to immediately truncate a byte slice. A tail is then added to the
// end of the byte slice.
func BytesWithTail(b []byte, width uint, tail []byte) []byte {
	f := NewWriter(width, string(tail))
	_, _ = f.Write(b)

	return f.Bytes()
}

// String is shorthand for declaring a new default truncate-writer instance,
// used to immediately truncate a string.
func String(s string, width uint) string {
	return StringWithTail(s, width, "")
}

// StringWithTail is shorthand for declaring a new default truncate-writer instance,
// used to immediately truncate a string. A tail is then added to the end of the
// string.
func StringWithTail(s string, width uint, tail string) string {
	return string(BytesWithTail([]byte(s), width, []byte(tail)))
}

// Write truncates content at the given printable cell width, leaving any
// ansi sequences intact.
func (w *Writer) Write(b []byte) (int, error) {
	tw := ansi.PrintableRuneWidth(w.tail)
	if w.width < uint(tw) {
		return w.buf.WriteString(w.tail)
	}

	w.width -= uint(tw)
	var curWidth uint

	collector := statemachine.CommandCollector{}

	var debugSequence []string
	defer func() {
		fmt.Println("printable sequence:", strings.Join(debugSequence, ", "))
	}()

	bi := 0
	s := string(b)

	isTruncating := false

	// In order to maintain legacy compatibility, the truncator
	// will automatically add a reset color sequence to the end
	// of any truncated sequence that contains a color sequence,
	// that is not already reset
	needsColorReset := false

	for i, c := range s {
		// consume all the bytes of this character in the statemachine
		var step statemachine.CollectorStep
		for ; bi <= i; bi++ {
			step = collector.Next(s[bi])
		}

		// if we're in a non-printing sequence, don't count the width of this character
		isPrinting := step.IsPrintingStep()
		if isPrinting {
			curWidth += uint(runewidth.RuneWidth(c))
			// TODO delete
			debugSequence = append(debugSequence, strconv.Quote(string(c)))
		}

		// check if we just stepped a command
		if step.Command.Type == statemachine.TypeCSICommand {
			if step.Command.CommandId == "0m" {
				// Reset color sequence
				needsColorReset = false
			} else if strings.HasSuffix(
				step.Command.CommandId, "m",
			) {
				// Some non-reset color sequence -- we may need to reset
				// at the end of the sequence
				needsColorReset = true
			}
		}

		// once we hit the max width, start truncating
		if !isTruncating && curWidth > w.width {
			// when we start truncating, write the tail
			n, err := w.writer.Write([]byte(w.tail))
			if err != nil {
				return i + n, err
			}
			isTruncating = true
		}

		// TODO delete
		fmt.Printf("w: %d \tc: %s \t\tstep: %s (printing: %v truncating: %v)\n", curWidth, strconv.Quote(string(c)), step,
			isPrinting, isTruncating)

		// when we start truncating, only write non-printable
		// characters to the buffer.
		if !isPrinting || !isTruncating {
			fmt.Println("writing")
			_, err := w.writer.Write([]byte(string(c)))
			if err != nil {
				return 0, err
			}
		}
	}

	if isTruncating && needsColorReset {
		// Append a color reset sequence
		n, err := w.writer.Write([]byte("\x1b[0m"))
		return len(b) + n, err
	}
	return len(b), nil
}

// Bytes returns the truncated result as a byte slice.
func (w *Writer) Bytes() []byte {
	return w.buf.Bytes()
}

// String returns the truncated result as a string.
func (w *Writer) String() string {
	return w.buf.String()
}
