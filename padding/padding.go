package padding

import (
	"bytes"
	"io"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
	"github.com/muesli/reflow/ansi"
	"github.com/muesli/reflow/internal/statemachine"
)

type PaddingFunc func(w io.Writer)

type Writer struct {
	Padding uint
	PadFunc PaddingFunc

	ansiState statemachine.AnsiState
	buf       bytes.Buffer
	cache     bytes.Buffer
	lineLen   int
	ansi      bool
}

func NewWriter(width uint, paddingFunc PaddingFunc) *Writer {
	w := &Writer{
		Padding: width,
		PadFunc: paddingFunc,
	}
	return w
}

func NewWriterPipe(forward io.Writer, width uint, paddingFunc PaddingFunc) *Writer {
	return &Writer{
		Padding: width,
		PadFunc: paddingFunc,
	}
}

// Bytes is shorthand for declaring a new default padding-writer instance,
// used to immediately pad a byte slice.
func Bytes(b []byte, width uint) []byte {
	f := Writer{
		Padding: width,
	}
	f.buf.Grow(int(width))
	_, _ = f.Write(b)
	_ = f.Flush()

	return f.Bytes()
}

// String is shorthand for declaring a new default padding-writer instance,
// used to immediately pad a string.
func String(s string, width uint) string {
	f := Writer{
		Padding: width,
	}
	f.buf.Grow(int(width))
	_, _ = f.WriteString(s)
	_ = f.Flush()

	return f.String()
}

// Write is used to write content to the padding buffer.
func (w *Writer) Write(b []byte) (int, error) {
	i := 0

	// iterate runes without copying the byte array onto the heap
	for i < len(b) {
		c, charWidth := utf8.DecodeRune(b[i:])
		// consume all the bytes of this character in the statemachine
		nextI := i + charWidth
		var step statemachine.CollectorStep
		for j := i; j < nextI; j++ {
			step = w.ansiState.Next(b[j])
		}

		if step.IsPrinting() {
			w.lineLen += runewidth.RuneWidth(c)

			if b[i] == '\n' {
				// end of current line
				err := w.pad()
				if err != nil {
					return 0, err
				}
				if w.ansiState.IsDirty() {
					w.ansiState.WriteResetSequence(&w.buf)
				}
				w.lineLen = 0
			}
		}

		if n, err := w.buf.Write(b[i:nextI]); err != nil {
			return i + n, err
		}

		i = nextI
	}

	return len(b), nil
}

// Write is used to write content to the padding buffer.
func (w *Writer) WriteString(s string) (int, error) {
	i := 0
	for nextI, c := range s {
		var step statemachine.CollectorStep
		for j := i; j < nextI; j++ {
			step = w.ansiState.Next(s[j])
		}

		if step.IsPrinting() {
			w.lineLen += runewidth.RuneWidth(c)
			if c == '\n' {
				// end of current line
				err := w.pad()
				if err != nil {
					return 0, err
				}
				w.ansiState.WriteResetSequence(&w.buf)
				w.lineLen = 0
			}
		}

		if _, err := w.buf.WriteRune(c); err != nil {
			return i, err
		}

		i = nextI
	}

	return len(s), nil
}

func (w *Writer) pad() error {
	if w.Padding > 0 && uint(w.lineLen) < w.Padding {
		if w.PadFunc != nil {
			// if we have a padding function, then we need an actual ansi writer
			// in order to intercept the arbitrary write operations that a consumer
			// might perform.
			writer := ansi.WriterForState(w.ansiState, &w.buf)
			for i := 0; i < int(w.Padding)-w.lineLen; i++ {
				w.PadFunc(writer)
			}
			w.ansiState = writer.ExportState()
		} else {
			// Otherwise, we can just write spaces directly to the buffer
			for i := 0; i < int(w.Padding)-w.lineLen; i++ {
				_, err := w.buf.WriteRune(' ')
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// Close will finish the padding operation.
func (w *Writer) Close() (err error) {
	return w.Flush()
}

// Bytes returns the padded result as a byte slice.
func (w *Writer) Bytes() []byte {
	return w.cache.Bytes()
}

// String returns the padded result as a string.
func (w *Writer) String() string {
	return w.cache.String()
}

// Flush will finish the padding operation. Always call it before trying to
// retrieve the final result.
func (w *Writer) Flush() (err error) {
	if w.lineLen != 0 {
		if err = w.pad(); err != nil {
			return
		}
	}

	w.cache.Reset()
	_, err = w.buf.WriteTo(&w.cache)
	w.lineLen = 0
	w.ansi = false

	return
}
