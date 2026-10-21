package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// parseTOML reads the gap.toml subset: comments, [a] and [a.b.c] headers,
// bare and quoted keys, dotted keys, basic strings, and string arrays.
// Anything else errors with its line.
func parseTOML(text string) (map[string]any, error) {
	root := map[string]any{}
	current := root
	lines := strings.Split(text, "\n")
	for i := 0; i < len(lines); i++ {
		lineNo := i + 1
		raw := lines[i]
		line := stripComment(raw)
		if strings.TrimSpace(line) == "" {
			continue
		}
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "[") && !arrayContinues(trim) {
			if !strings.HasSuffix(trim, "]") || strings.Count(trim, "[") != 1 {
				return nil, tomlErr(lineNo, "unsupported header")
			}
			name := strings.TrimSpace(trim[1 : len(trim)-1])
			if name == "" || strings.ContainsAny(name, "[]") {
				return nil, tomlErr(lineNo, "unsupported header")
			}
			keys, err := splitKey(name, lineNo)
			if err != nil {
				return nil, err
			}
			table, err := descendTable(root, keys, lineNo, true)
			if err != nil {
				return nil, err
			}
			current = table
			continue
		}
		key, value, ok := strings.Cut(trim, "=")
		if !ok {
			return nil, tomlErr(lineNo, "expected key = value")
		}
		keys, err := splitKey(strings.TrimSpace(key), lineNo)
		if err != nil {
			return nil, err
		}
		if len(keys) == 0 || keys[0] == "" {
			return nil, tomlErr(lineNo, "missing key")
		}
		valText := strings.TrimSpace(value)
		for arrayContinues(valText) && i+1 < len(lines) {
			i++
			valText += "\n" + stripComment(lines[i])
		}
		if arrayContinues(valText) {
			return nil, tomlErr(lineNo, "unterminated array")
		}
		parsed, err := parseValue(strings.TrimSpace(valText), lineNo)
		if err != nil {
			return nil, err
		}
		parent, err := descendTable(current, keys[:len(keys)-1], lineNo, false)
		if err != nil {
			return nil, err
		}
		last := keys[len(keys)-1]
		if _, exists := parent[last]; exists {
			return nil, tomlErr(lineNo, "duplicate key "+last)
		}
		parent[last] = parsed
	}
	return root, nil
}

func tomlErr(line int, msg string) error {
	return fmt.Errorf("toml line %d: %s", line, msg)
}

// stripComment drops an unquoted # and the rest of the line.
func stripComment(line string) string {
	quote := false
	esc := false
	for i := 0; i < len(line); i++ {
		c := line[i]
		if quote {
			if esc {
				esc = false
				continue
			}
			if c == '\\' {
				esc = true
				continue
			}
			if c == '"' {
				quote = false
			}
			continue
		}
		if c == '"' {
			quote = true
			continue
		}
		if c == '#' {
			return line[:i]
		}
	}
	return line
}

// arrayContinues reports an unclosed [ that is not inside a string or comment.
func arrayContinues(s string) bool {
	quote, esc := false, false
	depth := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if quote {
			if esc {
				esc = false
				continue
			}
			if c == '\\' {
				esc = true
				continue
			}
			if c == '"' {
				quote = false
			}
			continue
		}
		switch c {
		case '"':
			quote = true
		case '#':
			return depth > 0
		case '[':
			depth++
		case ']':
			if depth > 0 {
				depth--
			}
		}
	}
	return depth > 0
}

func splitKey(s string, line int) ([]string, error) {
	raw := splitDotsRaw(s)
	keys := make([]string, len(raw))
	for i, part := range raw {
		if err := validSegment(part, line); err != nil {
			return nil, err
		}
		keys[i] = decodeKey(part)
	}
	return keys, nil
}

// validSegment accepts a bare key or one quoted basic string. The last dotted segment uses the same rule.
func validSegment(part string, line int) error {
	part = strings.TrimSpace(part)
	if len(part) >= 2 && part[0] == '"' && part[len(part)-1] == '"' {
		if _, err := unquote(part); err != nil {
			return tomlErr(line, "bad key")
		}
		return nil
	}
	return validKey(part, line)
}

func validKey(key string, line int) error {
	if key == "" {
		return tomlErr(line, "bad key "+strconv.Quote(key))
	}
	for _, r := range key {
		if r > unicode.MaxASCII || (!unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '-') {
			return tomlErr(line, "bad key "+strconv.Quote(key))
		}
	}
	return nil
}

func splitDotsRaw(s string) []string {
	var keys []string
	var b strings.Builder
	quote := false
	esc := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if quote {
			b.WriteByte(c)
			if esc {
				esc = false
				continue
			}
			if c == '\\' {
				esc = true
				continue
			}
			if c == '"' {
				quote = false
			}
			continue
		}
		if c == '"' {
			quote = true
			b.WriteByte(c)
			continue
		}
		if c == '.' {
			keys = append(keys, b.String())
			b.Reset()
			continue
		}
		b.WriteByte(c)
	}
	return append(keys, b.String())
}

func decodeKey(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		v, err := unquote(s)
		if err == nil {
			if text, ok := v.(string); ok {
				return text
			}
		}
	}
	return s
}

// descendTable walks or creates nested tables. header is true for a [table]
// line, which must not repeat a header already opened.
func descendTable(root map[string]any, keys []string, line int, header bool) (map[string]any, error) {
	cur := root
	for i, key := range keys {
		// splitKey already accepted each segment as bare or quoted. A decoded quoted key may contain dots or spaces.
		next, ok := cur[key]
		if !ok {
			made := map[string]any{}
			cur[key] = made
			cur = made
			continue
		}
		table, ok := next.(map[string]any)
		if !ok {
			return nil, tomlErr(line, "redefining "+key)
		}
		if header && i == len(keys)-1 {
			if _, marked := table["\x00header"]; marked {
				return nil, tomlErr(line, "redefining table "+strings.Join(keys, "."))
			}
		}
		cur = table
	}
	if header {
		cur["\x00header"] = true
	}
	return cur, nil
}

func parseValue(s string, line int) (any, error) {
	if s == "" {
		return nil, tomlErr(line, "missing value")
	}
	switch s[0] {
	case '"':
		v, err := unquote(s)
		if err != nil {
			return nil, tomlErr(line, err.Error())
		}
		return v, nil
	case '[':
		return parseArray(s, line)
	}
	if s[0] >= '0' && s[0] <= '9' || s[0] == '-' || s[0] == '+' {
		return nil, tomlErr(line, "numbers are not supported")
	}
	return nil, tomlErr(line, "unsupported value")
}

func unquote(s string) (any, error) {
	if len(s) < 2 || s[0] != '"' || !strings.HasSuffix(s, "\"") || strings.HasPrefix(s, "\"\"\"") {
		return nil, fmt.Errorf("toml: bad string")
	}
	var b strings.Builder
	esc := false
	for i := 1; i < len(s)-1; i++ {
		c := s[i]
		if esc {
			switch c {
			case '\\', '"':
				b.WriteByte(c)
			case 'n':
				b.WriteByte('\n')
			default:
				return nil, fmt.Errorf("toml: bad escape")
			}
			esc = false
			continue
		}
		if c == '\\' {
			esc = true
			continue
		}
		if c == '"' || c == '\n' {
			return nil, fmt.Errorf("toml: bad string")
		}
		b.WriteByte(c)
	}
	if esc {
		return nil, fmt.Errorf("toml: bad string")
	}
	return b.String(), nil
}

func parseArray(s string, line int) ([]string, error) {
	if !strings.HasPrefix(s, "[") || !strings.HasSuffix(strings.TrimSpace(s), "]") {
		return nil, tomlErr(line, "bad array")
	}
	body := strings.TrimSpace(s[1 : len(s)-1])
	if strings.TrimSpace(body) == "" {
		return []string{}, nil
	}
	var out []string
	for _, part := range splitComma(body) {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if part[0] != '"' {
			return nil, tomlErr(line, "arrays must be strings")
		}
		v, err := unquote(part)
		if err != nil {
			return nil, tomlErr(line, "bad string in array")
		}
		out = append(out, v.(string))
	}
	if out == nil {
		out = []string{}
	}
	return out, nil
}

// splitComma splits on commas that are outside quotes and brackets.
func splitComma(s string) []string {
	var parts []string
	var b strings.Builder
	quote, esc := false, false
	depth := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if quote {
			b.WriteByte(c)
			if esc {
				esc = false
				continue
			}
			if c == '\\' {
				esc = true
				continue
			}
			if c == '"' {
				quote = false
			}
			continue
		}
		switch c {
		case '"':
			quote = true
		case '[':
			depth++
		case ']':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				parts = append(parts, b.String())
				b.Reset()
				continue
			}
		}
		b.WriteByte(c)
	}
	return append(parts, b.String())
}

// writeLockTOML emits schema = "2" and sorted [components.<id>] tables.
func writeLockTOML(components map[string]lockEntryM1) string {
	ids := make([]string, 0, len(components))
	for id := range components {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	var b strings.Builder
	b.WriteString("schema = \"2\"\n")
	for _, id := range ids {
		e := components[id]
		b.WriteString("\n[components.")
		b.WriteString(tomlKey(id))
		b.WriteString("]\n")
		b.WriteString("ref = " + strconv.Quote(e.Ref) + "\n")
		b.WriteString("commit = " + strconv.Quote(e.Commit) + "\n")
		b.WriteString("tree = " + strconv.Quote(e.Tree) + "\n")
	}
	return b.String()
}

func tomlKey(id string) string {
	if id == "" {
		return strconv.Quote(id)
	}
	for _, r := range id {
		if r > unicode.MaxASCII || (!unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '-') {
			return strconv.Quote(id)
		}
	}
	r, _ := utf8.DecodeRuneInString(id)
	if !unicode.IsLetter(r) && r != '_' {
		return strconv.Quote(id)
	}
	return id
}
