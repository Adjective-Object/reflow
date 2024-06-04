package wrap

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

	"github.com/muesli/reflow/internal/statemachine"
)

var tt = []struct {
	Input         string
	Expected      string
	Limit         int
	KeepNewlines  bool
	BreakAnsi     bool
	PreserveSpace bool
	TabWidth      int
}{
	// No-op, should pass through, including trailing whitespace:
	{
		Input:         "foobar\n ",
		Expected:      "foobar\n ",
		Limit:         0,
		KeepNewlines:  true,
		PreserveSpace: false,
		TabWidth:      0,
	},
	// Nothing to wrap here, should pass through:
	{
		Input:         "foo",
		Expected:      "foo",
		Limit:         4,
		KeepNewlines:  true,
		PreserveSpace: false,
		TabWidth:      0,
	},
	// In contrast to wordwrap we break a long word to obey the given limit
	{
		Input:         "foobarfoo",
		Expected:      "foob\narfo\no",
		Limit:         4,
		KeepNewlines:  true,
		PreserveSpace: false,
		TabWidth:      0,
	},
	// Newlines in the input are respected if desired
	{
		Input:         "f\no\nobar",
		Expected:      "f\no\noba\nr",
		Limit:         3,
		KeepNewlines:  true,
		PreserveSpace: false,
		TabWidth:      0,
	},
	// Newlines in the input can be ignored if desired
	{
		Input:         "f\no\nobar",
		Expected:      "foo\nbar",
		Limit:         3,
		KeepNewlines:  false,
		PreserveSpace: false,
		TabWidth:      0,
	},
	// Leading whitespaces after forceful line break can be preserved if desired
	{
		Input:         "foo bar\n  baz",
		Expected:      "foo\n ba\nr\n  b\naz",
		Limit:         3,
		KeepNewlines:  true,
		PreserveSpace: true,
		TabWidth:      0,
	},
	// Leading whitespaces after forceful line break can be removed if desired
	{
		Input:         "foo bar\n  baz",
		Expected:      "foo\nbar\n  b\naz",
		Limit:         3,
		KeepNewlines:  true,
		PreserveSpace: false,
		TabWidth:      0,
	},
	// Tabs are broken up according to the configured TabWidth
	{
		Input:         "foo\tbar",
		Expected:      "foo \n  ba\nr",
		Limit:         4,
		KeepNewlines:  true,
		PreserveSpace: true,
		TabWidth:      3,
	},
	// Remaining width of wrapped tab is ignored when space is not preserved
	{
		Input:         "foo\tbar",
		Expected:      "foo \nbar",
		Limit:         4,
		KeepNewlines:  true,
		PreserveSpace: false,
		TabWidth:      3,
	},
	// ANSI sequence codes don't affect length calculation:
	{
		Input:         "\x1B[38;2;249;38;114mfoo\x1B[0m\x1B[38;2;248;248;242m \x1B[0m\x1B[38;2;230;219;116mbar\x1B[0m",
		Expected:      "\x1B[38;2;249;38;114mfoo\x1B[0m\x1B[38;2;248;248;242m \x1B[0m\x1B[38;2;230;219;116mbar\x1B[0m",
		Limit:         7,
		KeepNewlines:  true,
		PreserveSpace: false,
		TabWidth:      0,
	},
	// ANSI control codes don't get wrapped when BreakAnsi = false
	{
		Input:         "\x1B[38;2;249;38;114m(\x1B[0m\x1B[38;2;248;248;242mjust another test\x1B[38;2;249;38;114m)\x1B[0m",
		Expected:      "\x1B[38;2;249;38;114m(\x1B[0m\x1B[38;2;248;248;242mju\nst \nano\nthe\nr t\nest\x1B[38;2;249;38;114m\n)\x1B[0m",
		Limit:         3,
		BreakAnsi:     false,
		KeepNewlines:  true,
		PreserveSpace: false,
		TabWidth:      0,
	},
	// Link bodies get wrapped when BreakAnsi = true
	{
		Input: "\x1b]8;id=17175;https://example.website/docs\x1b\\The documentation website with a long link body!!\x1b]8;;\x1b\\\n",
		Expected: "\x1b]8;id=17175;https://example.website/docs\x1b\\" +
			"The documentation websit\x1b]8;id=17175;\x1b\\\n" +
			"\x1b]8;id=17175;https://example.website/docs\x1b\\" +
			"e with a long link body!\x1b]8;id=17175;\x1b\\\n" +
			"\x1b]8;id=17175;https://example.website/docs\x1b\\" +
			"!\x1b]8;;\x1b\\\n",
		Limit:         24,
		BreakAnsi:     true,
		KeepNewlines:  true,
		PreserveSpace: false,
		TabWidth:      2,
	},
}

func stripAnsi(input string) string {
	state := statemachine.StateMachine{}
	var out []byte
	for _, b := range []byte(input) {
		step := state.Next(b)
		if step.IsPrinting() {
			out = append(out, b)
		}
	}
	return string(out)
}

func TestWrap(t *testing.T) {
	for i, tc := range tt {
		f := NewWriter(tc.Limit)
		f.KeepNewlines = tc.KeepNewlines
		f.PreserveSpace = tc.PreserveSpace
		f.TabWidth = tc.TabWidth
		f.BreakAnsi = tc.BreakAnsi

		_, err := f.Write([]byte(tc.Input))
		if err != nil {
			t.Error(err)
		}

		if f.String() != tc.Expected {
			t.Errorf("Test %d, expected:\n\n`%s`\n\nActual Output:\n\n`%s`", i, strconv.Quote(tc.Expected), strconv.Quote(f.String()))
		} else {
			// check height writer with WriteString
			h1 := NewHeightWriter(tc.Limit)
			h1.KeepNewlines = tc.KeepNewlines
			h1.PreserveSpace = tc.PreserveSpace
			h1.TabWidth = tc.TabWidth
			h1.WriteString(tc.Input)
			if realHeight := strings.Count(stripAnsi(tc.Expected), "\n") + 1; realHeight != h1.Height() {
				t.Errorf("Test %d, WriteString(%s) expected height %d, got %d", i, strconv.Quote(tc.Input), realHeight, h1.Height())
			}

			h2 := NewHeightWriter(tc.Limit)
			h2.KeepNewlines = tc.KeepNewlines
			h2.PreserveSpace = tc.PreserveSpace
			h2.TabWidth = tc.TabWidth
			h2.Write([]byte(tc.Input))
			if realHeight := strings.Count(stripAnsi(tc.Expected), "\n") + 1; realHeight != h2.Height() {
				t.Errorf("Test %d, Write(%s) expected height %d, got %d", i, strconv.Quote(tc.Input), realHeight, h2.Height())
			}
		}
	}
}

func FuzzWrapHeightMatchesOriginal(f *testing.F) {
	for _, tc := range tt {
		f.Add(
			tc.Input, tc.Limit, tc.KeepNewlines, tc.PreserveSpace, tc.TabWidth,
		)
	}
	f.Fuzz(func(t *testing.T,
		input string,
		limit int,
		keepNewlines bool,
		preserveSpace bool,
		tabWidth int) {

		w := NewWriter(limit)
		w.KeepNewlines = keepNewlines
		w.PreserveSpace = preserveSpace
		w.TabWidth = tabWidth
		_, err := w.Write([]byte(input))
		if err != nil {
			return
		}

		hw := NewHeightWriter(limit)
		hw.KeepNewlines = keepNewlines
		hw.PreserveSpace = preserveSpace
		hw.TabWidth = tabWidth
		hw.WriteString(input)

		result := w.String()
		realHeight := strings.Count(stripAnsi(result), "\n") + 1
		if realHeight != hw.Height() {
			t.Errorf("WriteString(string) expected height %d, got %d (output: %s, stripped: %s)", realHeight, hw.Height(), strconv.Quote(result), strconv.Quote(stripAnsi(result)))
		}

		hw2 := NewHeightWriter(limit)
		hw2.KeepNewlines = keepNewlines
		hw2.PreserveSpace = preserveSpace
		hw2.TabWidth = tabWidth
		hw2.Write([]byte(input))

		if realHeight != hw2.Height() {
			t.Errorf("Write([]byte) expected height %d, got %d (output: %s, stripped: %s)", realHeight, hw2.Height(), strconv.Quote(result), strconv.Quote(stripAnsi(result)))
		}
	})
}

func BenchmarkWrapHeight(b *testing.B) {
	for _, tc := range tt {
		for i := 0; i < b.N; i++ {
			t := NewHeightWriter(tc.Limit)
			t.KeepNewlines = tc.KeepNewlines
			t.PreserveSpace = tc.PreserveSpace
			t.TabWidth = tc.TabWidth
			t.WriteString(tc.Input)
		}
	}
}

func BenchmarkWrap(b *testing.B) {
	for _, tc := range tt {
		for i := 0; i < b.N; i++ {
			t := NewWriter(tc.Limit)
			t.KeepNewlines = tc.KeepNewlines
			t.PreserveSpace = tc.PreserveSpace
			t.TabWidth = tc.TabWidth
			t.BreakAnsi = tc.BreakAnsi
			t.WriteString(tc.Input)
		}
	}
}

func TestWrapString(t *testing.T) {
	t.Parallel()

	actual := String("foo bar", 3)
	expected := "foo\nbar"
	if actual != expected {
		t.Errorf("expected:\n\n`%s`\n\nActual Output:\n\n`%s`", expected, actual)
	}
}

func TestWrapBytes(t *testing.T) {
	t.Parallel()

	actual := Bytes([]byte("foo bar"), 3)
	expected := []byte("foo\nbar")
	if !bytes.Equal(actual, expected) {
		t.Errorf("expected:\n\n`%s`\n\nActual Output:\n\n`%s`", expected, actual)
	}
}
