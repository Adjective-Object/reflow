package xtermlink

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"regexp"
	"strconv"
)

// WrapLinkBytes wraps the content in an xterm hyperlink escape sequence
func WrapLinkBytes(linkID string, content []byte, link []byte) []byte {
	const LINK_TAG_START = "\x1b]8;"
	const LINK_TAG_END = "\x1b\\"
	out := bytes.Buffer{}
	out.Grow(
		2*len(LINK_TAG_START) + // opening tags
			2*len(LINK_TAG_END) + // closing tags
			3 + // id=
			2 + // semicolons
			len(linkID) + len(content) + len(link), // user content
	)

	out.WriteString(LINK_TAG_START)
	out.WriteString("id=")
	out.WriteString(linkID)
	out.WriteString(";")
	out.Write(link)
	out.WriteString(LINK_TAG_END)
	out.Write(content)
	out.WriteString(LINK_TAG_START + ";" + LINK_TAG_END)
	return out.Bytes()
}

// WrapLink wraps the content in an xterm hyperlink escape sequence
func WrapLink(linkID string, content string, link string) string {
	return string(WrapLinkBytes(linkID, []byte(content), []byte(link)))
}

type match struct {
	start           int
	end             int
	addFileProtocol bool
}

// TODO: this is a simple regexp for now, but we should use a proper URL parser
//
// regexps are generally not very performant, and will produce a lot of garbage
// during URL matching

var linkRegexp = regexp.MustCompile(`(?:(https?|file)://(?:(?:\\\s)|\S)+[^\s.]|(?:[A-Za-z]:[\\/]|\./|\.\./|\/)(?:(?:[^ \\/]*[\\/])+[^ \\/]*))`)

func findXTermlinkMatches(text []byte) []match {
	regexpRes := linkRegexp.FindAllSubmatchIndex(text, -1)
	result := make([]match, len(regexpRes))
	for i, idx := range regexpRes {
		result[i] = match{start: idx[0], end: idx[1]}
		// check for protocol capturing group
		if idx[2] == -1 {
			result[i].addFileProtocol = true
		}
	}
	return result
}

func linkID(b []byte, offset int) string {
	hash := md5.Sum(b)
	hash[0] += hash[2] + hash[4] + hash[6] + hash[8] + hash[10] + hash[12] + hash[14]
	hash[1] += hash[3] + hash[5] + hash[7] + hash[9] + hash[11] + hash[13] + hash[15]

	return strconv.Itoa(int(binary.LittleEndian.Uint16(hash[:2])) + offset)
}
