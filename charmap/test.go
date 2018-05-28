package charmap

import (
	"bytes"
	"testing"
	"unicode/utf8"

	"golang.org/x/text/encoding"
)

type ucPair struct {
	ucp rune
	cp  uint8
}

func testIt(t *testing.T, e encoding.Encoding, testData []ucPair) {
	enc := e.NewEncoder()
	for _, pair := range testData {
		utf8bytes := make([]byte, 3)
		n := utf8.EncodeRune(utf8bytes, pair.ucp)
		utf8bytes = utf8bytes[0:n]

		result, err := enc.Bytes(utf8bytes)
		if err != nil {
			t.Fatal(err)
		}
		expected := []byte{pair.cp}
		if bytes.Compare(expected, result) != 0 {
			t.Logf("U+%04x: %v != %v", pair.ucp, expected, result)
			t.Fail()
		}
	}

	dec := e.NewDecoder()
	for _, pair := range testData {
		expected := make([]byte, 3)
		n := utf8.EncodeRune(expected, pair.ucp)
		expected = expected[0:n]

		result, err := dec.Bytes([]byte{pair.cp})
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Compare(expected, result) != 0 {
			t.Logf("\\x%02x: %v != %v", pair.cp, expected, result)
			t.Fail()
		}
	}
}
