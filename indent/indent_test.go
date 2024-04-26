package indent

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/muesli/reflow/internal/ansitransform"
	"github.com/muesli/reflow/internal/ansitransform/ansi_tutils"
)

type args struct {
	Indent     uint
	IndentFunc IndentFunc
}

var tt = []ansi_tutils.TestCase{
	// No-op, should pass through:
	{
		"foobar",
		"foobar",
		args{0, nil},
	},
	// Basic indentation:
	{
		"foobar",
		"    foobar",
		args{4, nil},
	},
	// Multi-line indentation:
	{
		"foo\nbar",
		"    foo\n    bar",
		args{4, nil},
	},
	// Multi-line with custom indenter:
	{
		"foo\nbar",
		"----foo\n----bar",
		args{4, func(w io.Writer) {
			// custom indenter
			w.Write([]byte("-"))
		}},
	},
	// ANSI color sequence codes:
	{
		"\x1B[38;2;249;38;114mfoo",
		"\x1B[38;2;249;38;114m\x1B[0m    \x1B[38;2;249;38;114mfoo",
		args{4, nil},
	},
	// XTerm Links
	{
		"\x1B]8;;https://gith\nub.com\x07foo",
		"\x1B]8;;https://gith\nub.com\x07\x1B]8;;\x1b\\    \x1B]8;;https://gith\nub.com\x1b\\foo",
		args{4, nil},
	},
}

func runTest(t testing.TB, w io.Writer, input string, param interface{}) (string, error) {
	a := param.(args)
	f := NewWriterPipe(w, a.Indent, a.IndentFunc)
	_, err := f.Write([]byte(input))
	return f.String(), err
}

func TestIndent(t *testing.T) {
	t.Parallel()

	ansi_tutils.RunTests(t, tt, runTest)
}

func FuzzEq(t *testing.F) {
	ansi_tutils.RunFuzzEq(t, tt, runTest)
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

	f := &Writer{
		Indent: 2,
		ansi: ansitransform.Ansi{
			Forward: fakeWriter{},
		},
	}

	if _, err := f.Write([]byte("foo")); err != fakeErr {
		t.Error(err)
	}

	f.skipIndent = true

	if _, err := f.Write([]byte("foo")); err != fakeErr {
		t.Error(err)
	}
}

var fakeErr = errors.New("fake error")

type fakeWriter struct{}

func (fakeWriter) Write(_ []byte) (int, error) {
	return 0, fakeErr
}
