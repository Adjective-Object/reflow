package truncate

import (
	"bytes"
	"io"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
	"github.com/muesli/reflow/ansi"
	"github.com/muesli/reflow/internal/statemachine"
)

type Writer struct {
	width uint
	tail  []byte

	writer io.Writer
	buf    bytes.Buffer
}

func makeWriter(width uint, tail []byte) Writer {
	w := Writer{
		width: width,
		tail:  tail,
	}
	w.buf.Grow(int(width) + 1)
	return w
}

func NewWriterBytes(width uint, tail []byte) *Writer {
	m := makeWriter(width, tail)
	return &m
}

func NewWriter(width uint, tail string) *Writer {
	m := makeWriter(width, []byte(tail))
	return &m
}

func NewWriterPipe(forward io.Writer, width uint, tail string) *Writer {
	return &Writer{
		width:  width,
		tail:   []byte(tail),
		writer: forward,
	}
}

func NewWriterPipeBytes(forward io.Writer, width uint, tail []byte) *Writer {
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
	f := makeWriter(width, tail)
	_, _ = f.Write(b)

	return f.Bytes()
}

// String is shorthand for declaring a new default truncate-writer instance,
// used to immediately truncate a string.
func String(s string, width uint) string {
	return string(BytesWithTail([]byte(s), width, nil))
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
	tw := ansi.PrintableRuneWidthBytes(w.tail)
	if w.width < uint(tw) {
		return w.buf.Write(w.tail)
	}

	w.width -= uint(tw)
	var curWidth uint

	collector := statemachine.CommandCollector{}

	isTruncating := false

	// In order to maintain legacy compatibility, the truncator
	// will automatically add a reset color sequence to the end
	// of any truncated sequence that contains a color sequence,
	// that is not already reset
	needsColorReset := false
	i := 0
	// iterate runes without copying the byte array onto the heap
	for i < len(b) {
		curChar, charWidth := utf8.DecodeRune(b[i:])
		// consume all the bytes of this character in the statemachine
		var step statemachine.CollectorStep
		nextI := i + charWidth
		for j := i; j < nextI; j++ {
			step = collector.Next(b[j])
		}

		// if we're in a non-printing sequence, don't count the width of this character
		isPrinting := step.IsPrinting()
		if isPrinting {
			curWidth += uint(runewidth.RuneWidth(curChar))
		}

		// check if we just stepped a command
		if step.Command.Type == statemachine.TypeCSICommand {
			if bytes.Equal(step.Command.CommandId, []byte{'0', 'm'}) {
				// Reset color sequence
				needsColorReset = false
			} else if bytes.HasSuffix(step.Command.CommandId, []byte{'m'}) {
				// Some non-reset color sequence -- we may need to reset
				// at the end of the sequence
				needsColorReset = true
			}
		}

		// once we hit the max width, start truncating
		if !isTruncating && curWidth > w.width {
			// when we start truncating, write the tail
			n, err := w.writeBuffer(w.tail)
			if err != nil {
				return i + n, err
			}
			isTruncating = true
		}

		// when we start truncating, only write non-printable
		// characters to the buffer.
		if !isPrinting || !isTruncating {
			// write the full character to the buffer
			n, err := w.writeBuffer(b[i:nextI])
			if err != nil {
				return i + n, err
			}
		}

		// advance the index by the number of bytes in the character
		i = nextI
	}

	if isTruncating && needsColorReset {
		// Append a color reset sequence
		n, err := w.writeBuffer([]byte("\x1b[0m"))
		return len(b) + n, err
	}
	return len(b), nil
}

func (w *Writer) writeBuffer(b []byte) (int, error) {
	if w.writer != nil {
		return w.writer.Write(b)
	}
	return w.buf.Write(b)
}

// Bytes returns the truncated result as a byte slice.
func (w *Writer) Bytes() []byte {
	return w.buf.Bytes()
}

// String returns the truncated result as a string.
func (w *Writer) String() string {
	return w.buf.String()
}
