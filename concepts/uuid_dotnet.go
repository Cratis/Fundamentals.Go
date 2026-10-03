// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package concepts

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
	"unicode/utf8"
)

// ParseDotNetGUID explicitly accepts .NET 10.0.12 Guid.Parse(string) input forms
// and legacy parsing behavior for valid UTF-8 strings. It returns RFC/network
// order bytes with canonical lowercase dashed String output. Invalid UTF-8 and
// rejected input return UUID{} and an error; a valid Guid.Empty returns zero
// with no error. Errors do not reproduce .NET exception types or messages.
//
// This is compatibility conversion, not canonical-input validation or correlation
// ID admission. X retains legacy zero-valued prefixes and uint16 truncation; D's
// triggered compatibility fallback retains conditional trailing-NUL acceptance.
// Scanning is linear with no compatibility length cap. Callers should limit
// untrusted payloads. This opt-in parser does not change ParseUUID or UUID's text,
// JSON or SQL codecs.
func ParseDotNetGUID(text string) (UUID, error) {
	if utf8.ValidString(text) {
		text = strings.TrimFunc(text, dotNetGUIDWhitespace)
		if id, ok := parseDotNetGUID(text); ok {
			return id, nil
		}
	}
	return UUID{}, fmt.Errorf("invalid .NET GUID")
}

// Dispatch and compatibility rules follow Guid.cs at runtime v10.0.12,
// 4271d88e0aebf3d04f188f1334c2220d80555ef6. See testdata/guid_parse.README.md.
func parseDotNetGUID(text string) (UUID, bool) {
	if len(text) < 32 {
		return UUID{}, false
	}
	switch text[0] {
	case '(':
		if len(text) == 38 && text[37] == ')' {
			return parseDotNetGUIDD(text[1:37])
		}
	case '{':
		if text[9] == '-' {
			if len(text) == 38 && text[37] == '}' {
				return parseDotNetGUIDD(text[1:37])
			}
		} else {
			return parseDotNetGUIDX(text)
		}
	default:
		if text[8] == '-' {
			return parseDotNetGUIDD(text)
		}
		if len(text) == 32 {
			var id UUID
			_, err := hex.Decode(id[:], []byte(text))
			if err == nil {
				return id, true
			}
		}
	}
	return UUID{}, false
}

func parseDotNetGUIDD(text string) (UUID, bool) {
	if len(text) != 36 || text[8] != '-' || text[13] != '-' || text[18] != '-' || text[23] != '-' {
		return UUID{}, false
	}
	if id, err := ParseUUID(text); err == nil {
		return id, true
	}
	// Only a failed strict D parse containing a legacy trigger enters fallback.
	if !strings.ContainsAny(text, "xX+") {
		return UUID{}, false
	}
	var id UUID
	for _, field := range [...]struct{ start, end, offset, width int }{
		{0, 8, 0, 4}, {9, 13, 4, 2}, {14, 18, 6, 2},
		{19, 23, 8, 2}, {24, 28, 10, 2},
	} {
		value, ok := dotNetGUIDLegacyHex(text[field.start:field.end])
		if !ok {
			return UUID{}, false
		}
		if field.width == 4 {
			binary.BigEndian.PutUint32(id[field.offset:], value)
		} else {
			binary.BigEndian.PutUint16(id[field.offset:], uint16(value&0xffff))
		}
	}
	// The last eight characters use Number's AllowHexSpecifier parser, not
	// Guid's legacy parser. Only here may ASCII hex be followed by NULs.
	tail := text[28:36]
	if i := strings.IndexByte(tail, 0); i >= 0 {
		for j := i; j < len(tail); j++ {
			if tail[j] != 0 {
				return UUID{}, false
			}
		}
		tail = tail[:i]
	}
	if len(tail) == 0 {
		return UUID{}, false
	}
	value, ok := dotNetGUIDHex(tail)
	if !ok {
		return UUID{}, false
	}
	binary.BigEndian.PutUint32(id[12:], value)
	return id, true
}

func parseDotNetGUIDX(text string) (UUID, bool) {
	// X alone removes all .NET whitespace, even within prefixes and digits.
	text = strings.Map(func(r rune) rune {
		if dotNetGUIDWhitespace(r) {
			return -1
		}
		return r
	}, text)
	if len(text) == 0 || text[0] != '{' {
		return UUID{}, false
	}
	var id UUID
	pos := 1
	for field := 0; field < 11; field++ {
		if field == 3 {
			if pos >= len(text) || text[pos] != '{' {
				return UUID{}, false
			}
			pos++
		}
		if pos+2 > len(text) || text[pos] != '0' || (text[pos+1] != 'x' && text[pos+1] != 'X') {
			return UUID{}, false
		}
		pos += 2
		separator := byte(',')
		if field == 10 {
			separator = '}'
		}
		length := strings.IndexByte(text[pos:], separator)
		// Outer 0x must be followed by something before legacy prefixes are removed.
		if length <= 0 {
			return UUID{}, false
		}
		value, ok := dotNetGUIDLegacyHex(text[pos : pos+length])
		if !ok {
			return UUID{}, false
		}
		switch field {
		case 0:
			binary.BigEndian.PutUint32(id[:], value)
		case 1, 2:
			// Guid deliberately parses these fields as uint32, then truncates.
			binary.BigEndian.PutUint16(id[2+field*2:], uint16(value&0xffff))
		default:
			if value > 255 {
				return UUID{}, false
			}
			id[field+5] = byte(value & 0xff)
		}
		pos += length + 1
	}
	if pos != len(text)-1 || text[pos] != '}' {
		return UUID{}, false
	}
	return id, true
}

func dotNetGUIDLegacyHex(text string) (uint32, bool) {
	if len(text) > 0 && text[0] == '+' {
		text = text[1:]
	}
	if len(text) > 1 && text[0] == '0' && (text[1] == 'x' || text[1] == 'X') {
		text = text[2:]
	}
	// Empty after legacy prefix removal is accepted as zero by Guid's parser.
	return dotNetGUIDHex(text)
}

func dotNetGUIDHex(text string) (uint32, bool) {
	var value uint32
	for i := 0; i < len(text); i++ {
		var digit uint32
		switch c := text[i]; {
		case c >= '0' && c <= '9':
			digit = uint32(c - '0')
		case c >= 'a' && c <= 'f':
			digit = uint32(c-'a') + 10
		case c >= 'A' && c <= 'F':
			digit = uint32(c-'A') + 10
		default:
			return 0, false
		}
		if value > (uint32(0xffffffff)-digit)/16 {
			return 0, false
		}
		value = value*16 + digit
	}
	return value, true
}

func dotNetGUIDWhitespace(r rune) bool {
	return r >= '\u0009' && r <= '\u000d' || r == '\u0020' ||
		r == '\u0085' || r == '\u00a0' || r == '\u1680' ||
		r >= '\u2000' && r <= '\u200a' || r == '\u2028' || r == '\u2029' ||
		r == '\u202f' || r == '\u205f' || r == '\u3000'
}
