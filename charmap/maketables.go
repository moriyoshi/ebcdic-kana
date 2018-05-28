// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/moriyoshi/ebcdic-kana/internal/gen"
)

const ascii = "\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f" +
	"\x10\x11\x12\x13\x14\x15\x16\x17\x18\x19\x1a\x1b\x1c\x1d\x1e\x1f" +
	` !"#$%&'()*+,-./0123456789:;<=>?` +
	`@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\]^_` +
	"`abcdefghijklmnopqrstuvwxyz{|}~\u007f"

type encoding struct {
	name        string
	comment     string
	varName     string
	replacement byte
	mapping     []rune
}

var metadataRe = regexp.MustCompile("<([^>]+)>\\s*(\\S+)")

var cescapeRe = regexp.MustCompile("\\\\x([0-9a-fA-f]+)")

func unescape(v []byte) []byte {
	return cescapeRe.ReplaceAllFunc(v, func(v []byte) []byte {
		c, err := strconv.ParseInt(string(v[2:]), 16, 8)
		if err != nil {
			return nil
		}
		return []byte{byte(c)}
	})
}

func toVarName(v string) string {
	var buf []byte
	for _, v := range strings.Split(v, "-") {
		buf = append(buf, strings.Title(v)...)
	}
	return string(buf)
}

func lowerTitle(v string) string {
	var lvn int
	var c rune
	for lvn, c = range v {
		if c < 'A' || c > 'Z' {
			break
		}
	}
	return strings.ToLower(v[:lvn]) + v[lvn:]
}

func getUCM(filename string) (*encoding, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", filename, err)
	}
	defer f.Close()

	mapping := make([]rune, 256)
	for i := range mapping {
		mapping[i] = '\ufffd'
	}
	rm := make(map[rune]byte)
	charsFound := 0
	state := 0
	scanner := bufio.NewScanner(f)
	var codeSetName string
	var subchar byte
	for scanner.Scan() {
		s := strings.TrimSpace(scanner.Text())
		if s == "" || s[0] == '#' {
			continue
		}
		switch state {
		case 0:
			if s == "CHARMAP" {
				state = 1
			} else {
				m := metadataRe.FindStringSubmatch(s)
				if m != nil {
					switch m[1] {
					case "code_set_name":
						codeSetName, err = strconv.Unquote(m[2])
						if err != nil {
							return nil, fmt.Errorf("failed to parse <code_set_name>: %w", err)
						}
					case "subchar":
						subchar = unescape([]byte(m[2]))[0]
					}
				}
			}
		case 1:
			{
				var c byte
				var r rune
				var rt int
				if _, err := fmt.Sscanf(s, `<U%x> \x%x |%d`, &r, &c, &rt); err != nil {
					continue
				}
				if rt == 0 {
					if _, ok := rm[r]; ok {
						return nil, fmt.Errorf("%s: U+%04d is already mapped to \\x%02x\n", filename, r, c)
					}
					rm[r] = c
				}
				if mapping[c] != '\ufffd' {
					return nil, fmt.Errorf("%s: \\x%02x is already mapped to U+%04x\n", filename, c, r)
				}
				mapping[c] = r
				charsFound++
			}
		}
	}

	if charsFound < 128 {
		return nil, fmt.Errorf("%s: only %d characters found (wrong page format?)", filename, charsFound)
	}

	return &encoding{
		name:        codeSetName,
		comment:     "",
		varName:     toVarName(codeSetName),
		replacement: subchar,
		mapping:     mapping,
	}, nil
}

func main() {
	all := []string{}

	w := gen.NewCodeWriter()

	printf := func(s string, a ...interface{}) { fmt.Fprintf(w, s, a...) }

	printf("import (\n")
	printf("\t\"golang.org/x/text/encoding\"\n")
	printf(")\n\n")
	var encodings []*encoding
	for _, filename := range os.Args[1:] {
		encoding, err := getUCM(filename)
		if err != nil {
			log.Fatal(err)
		}
		encodings = append(encodings, encoding)
	}

	for _, e := range encodings {
		varName := e.varName
		all = append(all, varName)

		asciiSuperset, low := strings.HasPrefix(string(e.mapping), ascii), 0x00
		if asciiSuperset {
			low = 0x80
		}
		lowerVarName := lowerTitle(varName)
		printf("// %s is the %s encoding.\n", varName, e.name)
		if e.comment != "" {
			printf("//\n// %s\n", e.comment)
		}
		printf("var %s *Charmap = &%s\n\nvar %s = Charmap{\nname: %q,\n",
			varName, lowerVarName, lowerVarName, e.name)
		printf("asciiSuperset: %t,\n", asciiSuperset)
		printf("low: 0x%02x,\n", low)
		printf("replacement: 0x%02x,\n", e.replacement)

		printf("decode: [256]utf8Enc{\n")
		i, backMapping := 0, map[rune]byte{}
		for _, c := range e.mapping {
			if _, ok := backMapping[c]; !ok && c != utf8.RuneError {
				backMapping[c] = byte(i)
			}
			var buf [8]byte
			n := utf8.EncodeRune(buf[:], c)
			if n > 3 {
				log.Fatalf("rune %q (%U) is too long", c, c)
			}
			printf("{%d,[3]byte{0x%02x,0x%02x,0x%02x}},", n, buf[0], buf[1], buf[2])
			if i%2 == 1 {
				printf("\n")
			}
			i++
		}
		printf("},\n")

		printf("encode: [256]uint32{\n")
		encode := make([]uint32, 0, 256)
		for c, i := range backMapping {
			encode = append(encode, uint32(i)<<24|uint32(c))
		}
		sort.Sort(byRune(encode))
		for len(encode) < cap(encode) {
			encode = append(encode, encode[len(encode)-1])
		}
		for i, enc := range encode {
			printf("0x%08x,", enc)
			if i%8 == 7 {
				printf("\n")
			}
		}
		printf("},\n}\n")

		// Add an estimate of the size of a single Charmap{} struct value, which
		// includes two 256 elem arrays of 4 bytes and some extra fields, which
		// align to 3 uint64s on 64-bit architectures.
		w.Size += 2*4*256 + 3*8
	}
	// TODO: add proper line breaking.
	printf("var listAll = []encoding.Encoding{\n%s,\n}\n\n", strings.Join(all, ",\n"))

	w.WriteGoFile("tables.go", "charmap")
}

type byRune []uint32

func (b byRune) Len() int           { return len(b) }
func (b byRune) Less(i, j int) bool { return b[i]&0xffffff < b[j]&0xffffff }
func (b byRune) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
