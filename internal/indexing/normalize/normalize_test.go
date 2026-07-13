package normalize_test

import (
	"testing"

	"github.com/fastygo/context/internal/indexing/normalize"
	"golang.org/x/text/unicode/norm"
)

func TestForHashingBOMAndCRLF(t *testing.T) {
	t.Parallel()
	original := append([]byte{0xEF, 0xBB, 0xBF}, []byte("a\r\nb\rc")...)
	got, err := normalize.ForHashing(original)
	if err != nil {
		t.Fatal(err)
	}
	if got != "a\nb\nc" {
		t.Fatalf("got %q", got)
	}
}

func TestForHashingNFC(t *testing.T) {
	t.Parallel()
	nfd := norm.NFD.String("é")
	got, err := normalize.ForHashing([]byte(nfd))
	if err != nil {
		t.Fatal(err)
	}
	if got != norm.NFC.String("é") {
		t.Fatalf("got %q", got)
	}
}

func TestRejectsInvalidUTF8(t *testing.T) {
	t.Parallel()
	if _, err := normalize.ForHashing([]byte{0xff, 0xfe}); err == nil {
		t.Fatal("expected invalid utf-8 error")
	}
}
