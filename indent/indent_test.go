package indent

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/muesli/reflow/internal/ansi_tutils"
)

type params struct {
	Indent     uint
	IndentFunc IndentFunc
}

var tt = []ansi_tutils.TestCase{
	// No-op, should pass through:
	{
		Input:    "foobar",
		Expected: "foobar",
		Params:   params{0, nil},
	},
	// Basic indentation:
	{
		Input:    "foobar",
		Expected: "    foobar",
		Params:   params{4, nil},
	},
	// Multi-line indentation:
	{
		Input:    "foo\nbar",
		Expected: "    foo\n    bar",
		Params:   params{4, nil},
	},
	// Multi-line with custom indenter:
	{
		Input:    "foo\nbar",
		Expected: "----foo\n----bar",
		Params: params{4, func(w io.Writer) {
			// custom indenter
			w.Write([]byte("-"))
		}},
	},
	// ANSI color sequence codes:
	{
		Input:    "\x1B[38;2;249;38;114mfoo",
		Expected: "\x1B[38;2;249;38;114m\x1B[0m    \x1B[38;2;249;38;114mfoo",
		Params:   params{4, nil},
	},
	// ANSI color sequence codes interacting with newlines:
	{
		Input:    "\x1B[38;2;249;38;114mfoo\nbar",
		Expected: "\x1B[38;2;249;38;114m\x1B[0m    \x1B[38;2;249;38;114mfoo\n\x1B[0m    \x1B[38;2;249;38;114mbar",
		Params:   params{4, nil},
	},
	// XTerm Links
	{
		Input:    "\x1B]8;;https://gith\nub.com\x07foo",
		Expected: "\x1B]8;;https://gith\nub.com\x07\x1B]8;;\x1b\\    \x1B]8;;https://gith\nub.com\x1b\\foo",
		Params:   params{4, nil},
	},
	// XTerm Links with IDs
	{
		Input:    "\x1B]8;id=1;https://gith\nub.com\x07foo",
		Expected: "\x1B]8;id=1;https://gith\nub.com\x07\x1B]8;id=1;\x1b\\    \x1B]8;id=1;https://gith\nub.com\x1b\\foo",
		Params:   params{4, nil},
	},
}

func makeTestWriter(
	t testing.TB,
	w io.Writer,
	param interface{}) ansi_tutils.WriterWithBuffer {
	a := param.(params)
	if w == nil {
		return NewWriter(a.Indent, a.IndentFunc)
	} else {
		return NewWriterPipe(w, a.Indent, a.IndentFunc)
	}
}

func TestIndent(t *testing.T) {
	t.Parallel()

	ansi_tutils.RunTests(t, tt, makeTestWriter)
}

func FuzzEq(t *testing.F) {
	ansi_tutils.RunFuzzEq(t, tt, makeTestWriter)
}

func TestIndentWriter(t *testing.T) {
	t.Parallel()

	f := NewWriter(4, nil)

	_, err := f.Write([]byte("foo\n"))
	if err != nil {
		t.Error(err)
	}
	_, err = f.Write([]byte("bar"))
	if err != nil {
		t.Error(err)
	}

	exp := "    foo\n    bar"
	if f.String() != exp {
		t.Errorf("expected:\n\n`%s`\n\nActual Output:\n\n`%s`", exp, f.String())
	}
}

func TestIndentString(t *testing.T) {
	t.Parallel()

	actual := String("foobar", 3)
	expected := "   foobar"
	if actual != expected {
		t.Errorf("expected:\n\n`%s`\n\nActual Output:\n\n`%s`", expected, actual)
	}
}

func BenchmarkIndentString(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		b.ReportAllocs()
		b.ResetTimer()
		for pb.Next() {
			String("foo", 2)
		}
	})
}

func BenchmarkIndentBytes(b *testing.B) {
	foo := []byte("foo")
	b.RunParallel(func(pb *testing.PB) {
		b.ReportAllocs()
		b.ResetTimer()
		for pb.Next() {
			Bytes(foo, 2)
		}
	})
}

func BenchmarkCompatTests_SimpleWriter(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		b.ReportAllocs()
		b.ResetTimer()
		for pb.Next() {
			for _, t := range tt {
				// filter out tests that don't use an indent function
				// since those are the only tests writeable by all 3
				// writers
				if t.Params.(params).IndentFunc == nil {
					continue
				}

				writer := NewSimpleWriter(t.Params.(params).Indent)
				writer.Write([]byte(t.Input))
			}
		}
	})
}

func BenchmarkCompatTests_AdvancedWriter_Fwd(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		b.ReportAllocs()
		b.ResetTimer()
		for pb.Next() {
			for _, t := range tt {
				// filter out tests that don't use an indent function
				// since those are the only tests writeable by all 3
				// writers
				if t.Params.(params).IndentFunc == nil {
					continue
				}

				writer := NewAdvancedWriter(&bytes.Buffer{}, t.Params.(params).Indent, t.Params.(params).IndentFunc)
				writer.Write([]byte(t.Input))
			}
		}
	})
}

func BenchmarkCompatTests_AdvancedWriter_Buf(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		b.ReportAllocs()
		b.ResetTimer()
		for pb.Next() {
			for _, t := range tt {
				// filter out tests that don't use an indent function
				// since those are the only tests writeable by all 3
				// writers
				if t.Params.(params).IndentFunc == nil {
					continue
				}

				writer := NewAdvancedWriter(nil, t.Params.(params).Indent, t.Params.(params).IndentFunc)
				writer.Write([]byte(t.Input))
			}
		}
	})
}

func BenchmarkCompatTests_InterfaceWriter_Fwd(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		b.ReportAllocs()
		b.ResetTimer()
		for pb.Next() {
			for _, t := range tt {
				// filter out tests that don't use an indent function
				// since those are the only tests writeable by all 3
				// writers
				if t.Params.(params).IndentFunc == nil {
					continue
				}

				p := t.Params.(params)
				writer := NewWriterPipe(&bytes.Buffer{}, p.Indent, p.IndentFunc)
				writer.Write([]byte(t.Input))
			}
		}
	})
}

func BenchmarkCompatTests_InterfaceWriter_Buf(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		b.ReportAllocs()
		b.ResetTimer()
		for pb.Next() {
			for _, t := range tt {
				// filter out tests that don't use an indent function
				// since those are the only tests writeable by all 3
				// writers
				if t.Params.(params).IndentFunc == nil {
					continue
				}

				p := t.Params.(params)
				writer := NewWriterPipe(nil, p.Indent, p.IndentFunc)
				writer.Write([]byte(t.Input))
			}
		}
	})
}

func TestIndentWriterWithIndentFunc(t *testing.T) {
	t.Parallel()

	f := NewWriter(2, func(w io.Writer) {
		_, _ = w.Write([]byte("."))
	})

	_, err := f.Write([]byte("foo\n"))
	if err != nil {
		t.Error(err)
	}
	_, err = f.Write([]byte("bar"))
	if err != nil {
		t.Error(err)
	}

	exp := "..foo\n..bar"
	if f.String() != exp {
		t.Errorf("expected:\n\n`%s`\n\nActual Output:\n\n`%s`", exp, f.String())
	}
}

func TestNewWriterPipe(t *testing.T) {
	t.Parallel()

	b := &bytes.Buffer{}
	f := NewWriterPipe(b, 2, nil)

	if _, err := f.Write([]byte("foo")); err != nil {
		t.Error(err)
	}

	actual := b.String()
	expected := "  foo"

	if actual != expected {
		t.Errorf("expected:\n\n`%s`\n\nActual Output:\n\n`%s`", expected, actual)
	}
}

func TestWriter_Error(t *testing.T) {
	t.Parallel()

	f := NewWriterPipe(fakeWriter{}, 2, nil)

	if _, err := f.Write([]byte("foo")); err != errFakeErr {
		t.Error(err)
	}
}

var errFakeErr = errors.New("fake error")

type fakeWriter struct{}

func (fakeWriter) Write(_ []byte) (int, error) {
	return 0, errFakeErr
}
