package main

import (
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

type configField struct {
	key   string
	value any
}

// setComponentFields writes fields into [components.<id>] of a gap.toml, leaving every other line as it was.
// A nil value removes that key. The result must parse and must read back the values written.
func setComponentFields(path, id string, fields []configField) error {
	text, err := readOptional(path)
	if err != nil {
		return err
	}
	next := text
	for _, f := range fields {
		literal := ""
		switch v := f.value.(type) {
		case nil:
		case bool:
			literal = strconv.FormatBool(v)
		case string:
			literal = strconv.Quote(v)
		case []string:
			literal = tomlStrings(v)
		default:
			return fmt.Errorf("%s: unsupported value for %s", path, f.key)
		}
		if next, err = withKey(next, id, f.key, literal); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
	}
	doc, err := parseTOML(next)
	if err != nil {
		return fmt.Errorf("editing components.%s in %s: %w", id, path, err)
	}
	components, _ := doc["components"].(map[string]any)
	table, _ := components[id].(map[string]any)
	for _, f := range fields {
		if got, set := table[f.key]; (f.value == nil && set) || (f.value != nil && !reflect.DeepEqual(got, f.value)) {
			return fmt.Errorf("%s: could not set %s for %s", path, f.key, id)
		}
	}
	if next == text {
		return nil
	}
	return writeAtomic(path, next)
}

// withKey sets key = literal in [components.<id>], or removes the key when literal is empty.
func withKey(text, id, key, literal string) (string, error) {
	line := key + " = " + literal + "\n"
	lines := strings.SplitAfter(text, "\n")
	header := -1
	var section []string
	for i, raw := range lines {
		trim := strings.TrimSpace(stripComment(raw))
		if strings.HasPrefix(trim, "[") {
			var err error
			if section, err = splitKey(strings.TrimSuffix(strings.TrimPrefix(trim, "["), "]"), i+1); err != nil {
				return "", err
			}
			if len(section) == 2 && section[0] == "components" && section[1] == id {
				header = i
			}
			continue
		}
		if header < 0 || len(section) != 2 || section[0] != "components" || section[1] != id {
			continue
		}
		if k, _, ok := strings.Cut(trim, "="); ok && strings.TrimSpace(k) == key {
			if literal == "" {
				return strings.Join(slices.Delete(lines, i, i+1), ""), nil
			}
			if !strings.HasSuffix(raw, "\n") {
				line = strings.TrimSuffix(line, "\n")
			}
			lines[i] = line
			return strings.Join(lines, ""), nil
		}
	}
	if literal == "" {
		return text, nil
	}
	if header >= 0 {
		if !strings.HasSuffix(lines[header], "\n") {
			lines[header] += "\n"
		}
		lines = slices.Insert(lines, header+1, line)
		return strings.Join(lines, ""), nil
	}
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	if text != "" {
		text += "\n"
	}
	return text + "[components." + tomlKey(id) + "]\n" + line, nil
}
