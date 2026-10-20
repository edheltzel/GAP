package main

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// tomlTable is a TOML table. Values are string or tomlTable.
type tomlTable map[string]any

// parseTOML reads the subset gap uses: tables, string keys, quoted string
// values, and comments. Arrays, numbers, booleans, dotted keys, and
// multiline strings are rejected.
func parseTOML(data []byte) (tomlTable, error) {
	text := string(data)
	if strings.HasPrefix(text, "\ufeff") {
		text = strings.TrimPrefix(text, "\ufeff")
	}
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	root := tomlTable{}
	cur := root
	defined := map[string]bool{"": true}
	for i, line := range strings.Split(text, "\n") {
		if err := parseTOMLLine(root, &cur, defined, line); err != nil {
			return nil, fmt.Errorf("line %d: %w", i+1, err)
		}
	}
	return root, nil
}

func parseTOMLLine(root tomlTable, cur *tomlTable, defined map[string]bool, line string) error {
	rest := strings.TrimLeftFunc(line, unicode.IsSpace)
	if rest == "" || rest[0] == '#' {
		return nil
	}
	if rest[0] == '[' {
		table, path, err := parseTOMLHeader(root, rest)
		if err != nil {
			return err
		}
		if defined[path] {
			return fmt.Errorf("duplicate table [%s]", path)
		}
		defined[path] = true
		*cur = table
		return nil
	}
	key, value, err := parseTOMLKeyValue(rest)
	if err != nil {
		return err
	}
	if _, exists := (*cur)[key]; exists {
		return fmt.Errorf("duplicate key %s", key)
	}
	(*cur)[key] = value
	return nil
}

func parseTOMLHeader(root tomlTable, s string) (tomlTable, string, error) {
	s = s[1:]
	var parts []string
	s = strings.TrimLeftFunc(s, unicode.IsSpace)
	for {
		if s == "" {
			return nil, "", fmt.Errorf("unclosed table name")
		}
		var key string
		var err error
		key, s, err = parseTOMLKey(s)
		if err != nil {
			return nil, "", err
		}
		parts = append(parts, key)
		s = strings.TrimLeftFunc(s, unicode.IsSpace)
		if s == "" {
			return nil, "", fmt.Errorf("unclosed table name")
		}
		if s[0] == ']' {
			s = s[1:]
			break
		}
		if s[0] != '.' {
			return nil, "", fmt.Errorf("expected '.' or ']' in table name")
		}
		s = strings.TrimLeftFunc(s[1:], unicode.IsSpace)
	}
	s = strings.TrimLeftFunc(s, unicode.IsSpace)
	if s != "" && s[0] != '#' {
		return nil, "", fmt.Errorf("trailing text after table name")
	}
	table, err := ensureTOMLTable(root, parts)
	return table, strings.Join(parts, "."), err
}

func parseTOMLKeyValue(s string) (string, string, error) {
	key, rest, err := parseTOMLKey(s)
	if err != nil {
		return "", "", err
	}
	rest = strings.TrimLeftFunc(rest, unicode.IsSpace)
	if rest == "" || rest[0] != '=' {
		return "", "", fmt.Errorf("expected '=' after key")
	}
	rest = strings.TrimLeftFunc(rest[1:], unicode.IsSpace)
	if rest == "" || rest[0] != '"' {
		return "", "", fmt.Errorf("value must be a quoted string")
	}
	value, rest, err := parseTOMLString(rest)
	if err != nil {
		return "", "", err
	}
	rest = strings.TrimLeftFunc(rest, unicode.IsSpace)
	if rest != "" && rest[0] != '#' {
		return "", "", fmt.Errorf("trailing text after value")
	}
	return key, value, nil
}

func parseTOMLKey(s string) (string, string, error) {
	if s == "" {
		return "", "", fmt.Errorf("missing key")
	}
	if s[0] == '"' {
		return parseTOMLString(s)
	}
	i := 0
	for i < len(s) {
		r, size := utf8.DecodeRuneInString(s[i:])
		if !isBareKeyRune(r) {
			break
		}
		i += size
	}
	if i == 0 {
		return "", "", fmt.Errorf("invalid key")
	}
	return s[:i], s[i:], nil
}

func isBareKeyRune(r rune) bool {
	return r == '_' || r == '-' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

func parseTOMLString(s string) (string, string, error) {
	if s == "" || s[0] != '"' {
		return "", "", fmt.Errorf("expected quoted string")
	}
	var b strings.Builder
	i := 1
	for i < len(s) {
		c := s[i]
		if c == '"' {
			return b.String(), s[i+1:], nil
		}
		if c == '\\' {
			if i+1 >= len(s) {
				return "", "", fmt.Errorf("unterminated escape")
			}
			switch s[i+1] {
			case '"', '\\':
				b.WriteByte(s[i+1])
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			default:
				return "", "", fmt.Errorf("unknown escape \\%c", s[i+1])
			}
			i += 2
			continue
		}
		b.WriteByte(c)
		i++
	}
	return "", "", fmt.Errorf("unterminated string")
}

func ensureTOMLTable(root tomlTable, parts []string) (tomlTable, error) {
	cur := root
	for i, part := range parts {
		next, exists := cur[part]
		if !exists {
			child := tomlTable{}
			cur[part] = child
			cur = child
			continue
		}
		child, ok := next.(tomlTable)
		if !ok {
			return nil, fmt.Errorf("%s is not a table", strings.Join(parts[:i+1], "."))
		}
		cur = child
	}
	return cur, nil
}

func tomlString(t tomlTable, key string) string {
	if t == nil {
		return ""
	}
	s, _ := t[key].(string)
	return s
}

func tomlChild(t tomlTable, key string) tomlTable {
	if t == nil {
		return nil
	}
	child, _ := t[key].(tomlTable)
	return child
}
