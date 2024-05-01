package xtermlink

import (
	"bytes"

	"github.com/muesli/reflow/internal/statemachine"
)

// Writer is a simple writer that will detect link-link structures in
// the input text, and wrap them in xterm hyperlink annotations
type Writer struct {
	state   statemachine.AnsiState
	buf     bytes.Buffer
	linkBuf bytes.Buffer
}

func String(s string) string {
	w := Writer{}
	w.buf.Grow(len(s))
	_, _ = w.WriteString(s)
	return w.String()
}

func Bytes(b []byte) []byte {
	w := Writer{}
	w.buf.Grow(len(b))
	_, _ = w.Write(b)
	return w.Bytes()
}

func (w *Writer) Write(b []byte) (int, error) {
	for i := 0; i < len(b); i++ {
		w.WriteByte(b[i])
	}
	return len(b), nil
}

func (w *Writer) WriteString(s string) (int, error) {
	// iterate without allocating
	for i := 0; i < len(s); i++ {
		w.WriteByte(s[i])
	}

	return len(s), nil
}

func (w *Writer) WriteByte(b byte) error {
	step := w.state.Next(b)
	if step.IsPrinting() {
		w.linkBuf.WriteByte(b)
	} else {
		if w.linkBuf.Len() > 0 {
			w.Flush()
		}
		w.buf.WriteByte(b)
	}

	return nil
}

func (w *Writer) Flush() {
	linkBufBytes := w.linkBuf.Bytes()
	matches := findXTermlinkMatches(linkBufBytes)
	if len(matches) == 0 {
		// no links: just echo through
		w.buf.Write(w.linkBuf.Bytes())
		w.linkBuf.Reset()
		return
	}

	// iterate over the matches and wrap them in xterm hyperlink annotations
	head := 0
	for i, match := range matches {
		// copy the link buffer to the output buffer, wrapping the links
		// parse https-like links
		w.buf.Write(linkBufBytes[head:match.start])
		link := linkBufBytes[match.start:match.end]
		content := link
		if match.addFileProtocol {
			content = append([]byte("file://"), content...)
		}

		w.buf.Write(WrapLinkBytes(
			linkID(link, i),
			link, content))

		// restore ansi state
		w.state.WriteXtermRestoreSequence(&w.buf)
		head = match.end
	}
	w.buf.Write(linkBufBytes[head:])
	w.linkBuf.Reset()
}

func (w *Writer) String() string {
	w.Flush()
	return w.buf.String()
}

func (w *Writer) Bytes() []byte {
	w.Flush()
	return w.buf.Bytes()
}
