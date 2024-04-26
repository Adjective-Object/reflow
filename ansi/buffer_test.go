package ansi

import (
	"bytes"
	"testing"
)

func TestBuffer_PrintableRuneWidth(t *testing.T) {
	t.Parallel()

	var bb bytes.Buffer
	bb.WriteString("\x1B[38;2;249;38;114mfoo")
	b := Buffer{bb}

	if n := b.PrintableRuneWidth(); n != 3 {
		t.Fatalf("width should be 3, got %d", n)
	}
}

func TestBuffer_PrintableRuneWidth_XTerm(t *testing.T) {
	t.Parallel()

	var bb bytes.Buffer
	bb.WriteString("\x1B]8;;https://github.com\x07foo\x1B]8;;\\ bar")
	b := Buffer{bb}

	if n := b.PrintableRuneWidth(); n != 7 {
		t.Fatalf("width should be 7, got %d", n)
	}
}

// go test -bench=Benchmark_PrintableRuneWidth -benchmem -count=4
func Benchmark_PrintableRuneWidth(b *testing.B) {
	s := "\x1B[38;2;249;38;114mfoo"

	b.RunParallel(func(pb *testing.PB) {
		b.ReportAllocs()
		b.ResetTimer()
		for pb.Next() {
			PrintableRuneWidth(s)
		}
	})
}

// go test -bench=Benchmark_PrintableRuneWidth -benchmem -count=4
func Benchmark_PrintableRuneWidth_Both(b *testing.B) {
	s := "\x1B[38;2;249;38;114mfoo"
	sXTerm := "\x1B]8;;https://github.com\x07foo\x1B]8;;\\ bar"

	b.RunParallel(func(pb *testing.PB) {
		b.ReportAllocs()
		b.ResetTimer()
		for pb.Next() {
			PrintableRuneWidth(s)
			PrintableRuneWidth(sXTerm)
		}
	})
}
