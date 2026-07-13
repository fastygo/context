// Package normalize applies ADR-0018 text normalization for hashing and chunking.
package normalize

import (
	"bytes"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// ForHashing prepares text for chunk spans and Merkle leaves:
// strip UTF-8 BOM, normalize newlines to LF, apply NFC.
// Original artifact bytes must remain untouched in the artifact store.
func ForHashing(original []byte) (string, error) {
	b := original
	if bytes.HasPrefix(b, utf8BOM) {
		b = b[len(utf8BOM):]
	}
	if !utf8.Valid(b) {
		return "", errInvalidUTF8
	}
	b = bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))
	b = bytes.ReplaceAll(b, []byte("\r"), []byte("\n"))
	return string(norm.NFC.Bytes(b)), nil
}

// RelativePath canonicalizes a project-relative path for path_key input.
func RelativePath(p string) string {
	p = string(norm.NFC.String(p))
	p = bytesToSlash(p)
	for len(p) > 2 && p[:2] == "./" {
		p = p[2:]
	}
	for len(p) > 0 && p[0] == '/' {
		p = p[1:]
	}
	return p
}

func bytesToSlash(p string) string {
	b := []byte(p)
	b = bytes.ReplaceAll(b, []byte(`\`), []byte(`/`))
	return string(b)
}

type invalidUTF8Error struct{}

func (invalidUTF8Error) Error() string { return "normalize: invalid UTF-8" }

var errInvalidUTF8 invalidUTF8Error
