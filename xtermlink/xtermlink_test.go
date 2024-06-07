package xtermlink

import (
	"strconv"
	"testing"
)

type testCase struct {
	input    string
	expected string
}

func runTestCase(
	t *testing.T,
	tc testCase,
) {
	t.Helper()

	actual := String(tc.input)
	if actual != tc.expected {
		t.Errorf("WrapXtermHyperlinks(%s)\ngot:      %s\nexpected: %s",
			tc.input, strconv.Quote(actual), strconv.Quote(tc.expected))
	}
}

func TestWrapXtermHyperlinks(t *testing.T) {
	t.Parallel()

	linkIdStr := func(s string, offset int) string {
		return linkID([]byte(s), offset)
	}

	t.Run("simple", func(t *testing.T) {
		runTestCase(t, testCase{
			input: "https://example.com",
			expected: "\x1b]8;id=" + linkIdStr("https://example.com", 0) +
				";https://example.com\x07https://example.com\x1b]8;;\x07",
		})
	})

	t.Run("trailing period", func(t *testing.T) {
		runTestCase(t, testCase{
			input: "https://example.com.",
			expected: "\x1b]8;id=" + linkIdStr("https://example.com", 0) +
				";https://example.com\x07https://example.com\x1b]8;;\x07.",
		})
	})

	t.Run("trailing colon", func(t *testing.T) {
		runTestCase(t, testCase{
			input: "https://example.com:",
			expected: "\x1b]8;id=" + linkIdStr("https://example.com", 0) +
				";https://example.com\x07https://example.com\x1b]8;;\x07:",
		})
	})

	t.Run("nested link", func(t *testing.T) {
		runTestCase(t, testCase{
			input: "\x1b]8;id=34388;https://example.com\x07 This inner link should be wrapped, but the xterm link header shouldnt be https://example2.com ...\x1b]8;;\x07",
			expected: "\x1b]8;id=34388;https://example.com\x07 This inner link should be wrapped, but the xterm link header shouldnt be \x1b]8;id=" + linkIdStr("https://example2.com", 0) +
				";https://example2.com\x07https://example2.com\x1b]8;;\x07\x1b]8;id=34388;https://example.com\x07 ...\x1b]8;;\x07",
		})
	})

	t.Run("multiple links", func(t *testing.T) {
		runTestCase(t, testCase{
			input: "multiple https://example.com links https://example.com",
			expected: "multiple \x1b]8;id=" + linkIdStr("https://example.com", 0) +
				";https://example.com\x07https://example.com\x1b]8;;\x07 links \x1b]8;id=" + linkIdStr("https://example.com", 1) +
				";https://example.com\x07https://example.com\x1b]8;;\x07",
		})
	})

	t.Run("link with path", func(t *testing.T) {
		runTestCase(t, testCase{
			input: "https://example.com/with/path",
			expected: "\x1b]8;id=" + linkIdStr("https://example.com/with/path", 0) +
				";https://example.com/with/path\x07https://example.com/with/path\x1b]8;;\x07",
		})
	})

	t.Run("link with path and query", func(t *testing.T) {
		runTestCase(t, testCase{
			input: "https://example.com/with/path?and=query",
			expected: "\x1b]8;id=" + linkIdStr("https://example.com/with/path?and=query", 0) +
				";https://example.com/with/path?and=query\x07https://example.com/with/path?and=query\x1b]8;;\x07",
		})
	})

	t.Run("link with path, query, and fragment", func(t *testing.T) {
		runTestCase(t, testCase{
			input: "https://example.com/with/path?and=query#and-fragment",
			expected: "\x1b]8;id=" + linkIdStr("https://example.com/with/path?and=query#and-fragment", 0) +
				";https://example.com/with/path?and=query#and-fragment\x07https://example.com/with/path?and=query#and-fragment\x1b]8;;\x07",
		})
	})

	t.Run("link with path, query, fragment, and escaped spaces", func(t *testing.T) {
		runTestCase(t, testCase{
			input: "https://example.com/with/path?and=query#and-fragment\\ with\\ spaces",
			expected: "\x1b]8;id=" + linkIdStr("https://example.com/with/path?and=query#and-fragment\\ with\\ spaces", 0) +
				";https://example.com/with/path?and=query#and-fragment\\ with\\ spaces\x07https://example.com/with/path?and=query#and-fragment\\ with\\ spaces\x1b]8;;\x07",
		})
	})

	t.Run("link with path, query, fragment, and encoded spaces", func(t *testing.T) {
		runTestCase(t, testCase{
			input: "https://example.com/with/path?and=query#and-fragment%20with%20spaces",
			expected: "\x1b]8;id=" + linkIdStr("https://example.com/with/path?and=query#and-fragment%20with%20spaces", 0) +
				";https://example.com/with/path?and=query#and-fragment%20with%20spaces\x07https://example.com/with/path?and=query#and-fragment%20with%20spaces\x1b]8;;\x07",
		})
	})

	t.Run("paths are not links", func(t *testing.T) {
		runTestCase(t, testCase{
			input:    "with/path",
			expected: "with/path",
		})
	})

	t.Run("unix file links", func(t *testing.T) {
		runTestCase(t, testCase{
			input: "./with/path",
			expected: "\x1b]8;id=" + linkIdStr("./with/path", 0) +
				";file://./with/path\x07./with/path\x1b]8;;\x07",
		})
		runTestCase(t, testCase{
			input: "../with/path",
			expected: "\x1b]8;id=" + linkIdStr("../with/path", 0) +
				";file://../with/path\x07../with/path\x1b]8;;\x07",
		})
	})

	t.Run("abs unix file links", func(t *testing.T) {
		runTestCase(t, testCase{
			input: "/with/path",
			expected: "\x1b]8;id=" + linkIdStr("/with/path", 0) +
				";file:///with/path\x07/with/path\x1b]8;;\x07",
		})
	})

	t.Run("windows file links with unix path separators", func(t *testing.T) {
		runTestCase(t, testCase{
			input: "C://with/path",
			expected: "\x1b]8;id=" + linkIdStr("C://with/path", 0) +
				";file://C://with/path\x07C://with/path\x1b]8;;\x07",
		})
	})

	t.Run("windows file links with windows path separators", func(t *testing.T) {
		runTestCase(t, testCase{
			input: "C:\\\\with\\path",
			expected: "\x1b]8;id=" + linkIdStr("C:\\\\with\\path", 0) +
				";file://C:\\\\with\\path\x07C:\\\\with\\path\x1b]8;;\x07",
		})
	})

	t.Run("windows file links with repeated paths should have unique hover IDs", func(t *testing.T) {
		runTestCase(t, testCase{
			input: "multiple C:\\\\with\\path identical links C:\\\\with\\path",
			expected: "multiple \x1b]8;id=" + linkIdStr("C:\\\\with\\path", 0) +
				";file://C:\\\\with\\path\x07C:\\\\with\\path\x1b]8;;\x07 identical links \x1b]8;id=" + linkIdStr("C:\\\\with\\path", 1) +
				";file://C:\\\\with\\path\x07C:\\\\with\\path\x1b]8;;\x07",
		})
	})

	t.Run("noop", func(t *testing.T) {
		runTestCase(t, testCase{
			input:    "not yet started",
			expected: "not yet started",
		})
	})

}
