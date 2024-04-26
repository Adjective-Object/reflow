package indent

import (
	"bytes"
	"io"
	"unicode/utf8"

	"github.com/muesli/reflow/ansi"
)

type IndentFunc func(w io.Writer)

type Writer struct {
	Indent     uint
	IndentFunc IndentFunc

	ansiWriter ansi.Writer
	buf        bytes.Buffer
	skipIndent bool
	ansi       bool
}

func NewWriter(indent uint, indentFunc IndentFunc) *Writer {
	w := &Writer{
		Indent:     indent,
		IndentFunc: indentFunc,
	}
	w.ansiWriter = ansi.Writer{
		Forward: &w.buf,
	}
	return w
}

func NewWriterPipe(forward io.Writer, indent uint, indentFunc IndentFunc) *Writer {
	return &Writer{
		Indent:     indent,
		IndentFunc: indentFunc,
		ansiWriter: ansi.Writer{
			Forward: forward,
		},
	}
}

// Bytes is shorthand for declaring a new default indent-writer instance,
// used to immediately indent a byte slice.
func Bytes(b []byte, indent uint) []byte {
	// Since the Writer is not returned, we can use a fully on-stack writer
	// and include a pointer into it in the ansiWriter.
	f := Writer{
		Indent: indent,
	}
	f.ansiWriter = ansi.Writer{
		Forward: &f.buf,
	}
	f.buf.Grow(len(b))

	_, _ = f.Write(b)

	return f.Bytes()
}

// String is shorthand for declaring a new default indent-writer instance,
// used to immediately indent a string.
func String(s string, indent uint) string {
	// Since the Writer is not returned, we can use a fully on-stack writer
	// and include a pointer into it in the ansiWriter.
	//
	// TODO: revisit heap escape here: it currently escapes becaues it's
	// referenced by ansiWriter, which shouldn't be escaping here..
	f := Writer{
		Indent: indent,
	}
	f.ansiWriter = ansi.Writer{
		Forward: &f.buf,
	}
	// preallocate buffer to speed up
	f.buf.Grow(len(s))

	_, _ = f.WriteString(s)

	return f.String()
}

// Write is used to write content to the indent buffer.
func (w *Writer) Write(b []byte) (int, error) {
	// iterate runewise without reallocating
	i := 0
	// iterate runes without copying the byte array onto the heap
	for i < len(b) {
		c, charWidth := utf8.DecodeRune(b[i:])
		i += charWidth

		if err := w.WriteRune(c); err != nil {
			return i, err
		}
	}

	return len(b), nil
}

// Write is used to write content to the indent buffer.
func (w *Writer) WriteString(s string) (int, error) {
	// iterate runewise without reallocating
	// iterate runes without copying the byte array onto the heap
	for i, c := range s {
		if err := w.WriteRune(c); err != nil {
			return i, err
		}
	}

	return len(s), nil
}

func (w *Writer) WriteRune(c rune) error {
	if c == '\x1B' {
		// ANSI escape sequence
		w.ansi = true
	} else if w.ansi {
		if (c >= 0x41 && c <= 0x5a) || (c >= 0x61 && c <= 0x7a) {
			// ANSI sequence terminated
			w.ansi = false
		}
	} else {
		if !w.skipIndent {
			w.ansiWriter.ResetAnsi()
			if w.IndentFunc != nil {
				for i := 0; i < int(w.Indent); i++ {
					w.IndentFunc(&w.ansiWriter)
				}
			} else {
				for i := 0; i < int(w.Indent); i++ {
					if err := w.ansiWriter.WriteRune(' '); err != nil {
						return err
					}
				}
			}

			w.skipIndent = true
			w.ansiWriter.RestoreAnsi()
		}

		if c == '\n' {
			// end of current line
			w.skipIndent = false
		}
	}

	return w.ansiWriter.WriteRune(c)
}

// Bytes returns the indented result as a byte slice.
func (w *Writer) Bytes() []byte {
	return w.buf.Bytes()
}

// String returns the indented result as a string.
func (w *Writer) String() string {
	return w.buf.String()
}
