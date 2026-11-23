package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"maps"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"go.yaml.in/yaml/v3"
)

var (
	oscPattern         = regexp.MustCompile(`\x1b\][\s\S]*?(?:\x07|\x1b\\)`)
	dcsPattern         = regexp.MustCompile(`\x1b[P^_][\s\S]*?(?:\x1b\\)`)
	csiPattern         = regexp.MustCompile(`\x1b\[[\x30-\x3f]*[\x20-\x2f]*[\x40-\x7e]`)
	simpleEscPattern   = regexp.MustCompile(`\x1b[\x20-\x7e]`)
	c1Pattern          = regexp.MustCompile(`[\x{80}-\x{9f}]`)
	controlPattern     = regexp.MustCompile(`[\x00-\x06\x07\x08\x0b\x0c\x0d-\x1a\x1c-\x1f\x7f]`)
	lineBreakPattern   = regexp.MustCompile(`[\r\n]+`)
	nameInvalidPattern = regexp.MustCompile(`[^a-z0-9._]+`)
	nameEdgePattern    = regexp.MustCompile(`^[.\-]+|[.\-]+$`)
	frontmatterPattern = regexp.MustCompile(`^\x{FEFF}?---\r?\n([\s\S]*?)\r?\n---\r?\n?([\s\S]*)$`)
	nameHeadPattern    = regexp.MustCompile(`^(\x{FEFF}?---(\r?\n))([\s\S]*?)(\r?\n---(?:\r?\n|$))`)
	nameLinePattern    = regexp.MustCompile(`^name\s*:`)
	lineSplitPattern   = regexp.MustCompile(`\r?\n`)
	leadingEOLPattern  = regexp.MustCompile(`^\r?\n`)
	yamlCoreInt        = regexp.MustCompile(`^(?:[-+]?[0-9]+|0o[0-7]+|0x[0-9a-fA-F]+)$`)
	yamlCoreFloat      = regexp.MustCompile(`^(?:[-+]?(?:\.[0-9]+|[0-9]+(?:\.[0-9]*)?)(?:[eE][-+]?[0-9]+)?|[-+]?\.(?:inf|Inf|INF)|\.nan|\.NaN|\.NAN)$`)
)

func stripTerminalEscapes(s string) string {
	for _, re := range []*regexp.Regexp{oscPattern, dcsPattern, csiPattern, simpleEscPattern, c1Pattern, controlPattern} {
		s = re.ReplaceAllString(s, "")
	}
	return s
}

func jsSpace(r rune) bool {
	return r == '\uFEFF' || (r != '\u0085' && unicode.IsSpace(r))
}

func sanitizeMetadata(s string) string {
	return strings.TrimFunc(lineBreakPattern.ReplaceAllString(stripTerminalEscapes(s), " "), jsSpace)
}

func sanitizeName(name string) string {
	s := nameEdgePattern.ReplaceAllString(nameInvalidPattern.ReplaceAllString(strings.ToLower(name), "-"), "")
	if len(s) > 255 {
		s = s[:255]
	}
	if s == "" {
		return "unnamed-skill"
	}
	return s
}

func normalizeSkillName(name string) string {
	var b strings.Builder
	run := false
	for _, r := range strings.ToLower(name) {
		if jsSpace(r) || r == '_' {
			if !run {
				b.WriteByte('-')
			}
			run = true
			continue
		}
		run = false
		b.WriteRune(r)
	}
	return b.String()
}

func parseFrontmatter(raw string) (map[string]any, string, error) {
	m := frontmatterPattern.FindStringSubmatch(raw)
	if m == nil {
		return map[string]any{}, raw, nil
	}
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(m[1]), &doc); err != nil {
		return nil, "", err
	}
	v, err := yamlValue(&doc)
	if err != nil {
		return nil, "", err
	}
	data, ok := v.(map[string]any)
	if !ok {
		data = map[string]any{}
	}
	return data, m[2], nil
}

func yamlValue(root *yaml.Node) (any, error) {
	active := map[*yaml.Node]bool{}
	aliases := 0
	var walk func(*yaml.Node) (any, error)
	walk = func(n *yaml.Node) (any, error) {
		switch n.Kind {
		case 0:
			return nil, nil
		case yaml.DocumentNode:
			if len(n.Content) == 0 {
				return nil, nil
			}
			return walk(n.Content[0])
		case yaml.AliasNode:
			if active[n.Alias] {
				return nil, nil
			}
			if aliases++; aliases > 100 {
				return nil, fmt.Errorf("line %d: excessive alias count indicates a resource exhaustion attack", n.Line)
			}
			return walk(n.Alias)
		case yaml.MappingNode:
			active[n] = true
			defer delete(active, n)
			out := map[string]any{}
			for i := 0; i+1 < len(n.Content); i += 2 {
				key := n.Content[i].Value
				if _, dup := out[key]; dup {
					return nil, fmt.Errorf("line %d: map keys must be unique", n.Content[i].Line)
				}
				v, err := walk(n.Content[i+1])
				if err != nil {
					return nil, err
				}
				out[key] = v
			}
			return out, nil
		case yaml.SequenceNode:
			active[n] = true
			defer delete(active, n)
			out := make([]any, 0, len(n.Content))
			for _, item := range n.Content {
				v, err := walk(item)
				if err != nil {
					return nil, err
				}
				out = append(out, v)
			}
			return out, nil
		}
		tag := n.ShortTag()
		if tag == "!!timestamp" || n.Style == 0 && (tag == "!!int" && !yamlCoreInt.MatchString(n.Value) || tag == "!!float" && !yamlCoreFloat.MatchString(n.Value)) {
			return n.Value, nil
		}
		var v any
		if err := n.Decode(&v); err != nil {
			return nil, err
		}
		return v, nil
	}
	return walk(root)
}

func jsTruthy(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	case string:
		return x != ""
	case int:
		return x != 0
	case int64:
		return x != 0
	case uint64:
		return x != 0
	case float64:
		return x != 0 && !math.IsNaN(x)
	}
	return true
}

func jsTypeof(v any) string {
	switch v.(type) {
	case string:
		return "string"
	case bool:
		return "boolean"
	case int, int64, uint64, float64:
		return "number"
	}
	return "object"
}

type skillMeta struct {
	Name, Description, Dir string
}

func installInternalSkills() bool {
	v := os.Getenv("INSTALL_INTERNAL_SKILLS")
	return v == "1" || v == "true"
}

func parseSkillMD(path string, raw []byte, includeInternal bool, warn func(string)) (skillMeta, bool) {
	skip := func(reason string) (skillMeta, bool) {
		if warn != nil {
			warn(fmt.Sprintf("⚠ Skipped %s — %s", sanitizeMetadata(path), stripTerminalEscapes(reason)))
		}
		return skillMeta{}, false
	}
	data, _, err := parseFrontmatter(string(raw))
	if err != nil {
		return skip("YAML parse error: " + err.Error())
	}
	var missing []string
	for _, key := range []string{"name", "description"} {
		if !jsTruthy(data[key]) {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return skip("missing required frontmatter field(s): " + strings.Join(missing, ", "))
	}
	name, nameOK := data["name"].(string)
	description, descriptionOK := data["description"].(string)
	if !nameOK || !descriptionOK {
		return skip(fmt.Sprintf(`frontmatter "name" and "description" must be strings (got %s and %s)`, jsTypeof(data["name"]), jsTypeof(data["description"])))
	}
	name, description = sanitizeMetadata(name), sanitizeMetadata(description)
	var empty []string
	if name == "" {
		empty = append(empty, "name")
	}
	if description == "" {
		empty = append(empty, "description")
	}
	if len(empty) > 0 {
		return skip("empty required frontmatter field(s): " + strings.Join(empty, ", "))
	}
	metadata, _ := data["metadata"].(map[string]any)
	if internal := metadata["internal"]; (internal == true || internal == "true" || internal == "1") && !includeInternal && !installInternalSkills() {
		return skillMeta{}, false
	}
	return skillMeta{Name: name, Description: description, Dir: filepath.Dir(path)}, true
}

func yamlText(v any) (string, error) {
	var b strings.Builder
	enc := yaml.NewEncoder(&b)
	enc.SetIndent(2)
	if err := enc.Encode(v); err != nil {
		return "", err
	}
	if err := enc.Close(); err != nil {
		return "", err
	}
	return strings.TrimRight(b.String(), " \t\r\n"), nil
}

func setFrontmatterName(raw, name string) string {
	m := nameHeadPattern.FindStringSubmatch(raw)
	if m == nil {
		return raw
	}
	whole, open, eol, body, end := m[0], m[1], m[2], m[3], m[4]
	line, err := yamlText(map[string]string{"name": name})
	if err != nil {
		return raw
	}
	lines := lineSplitPattern.Split(body, -1)
	index := -1
	for i, l := range lines {
		if nameLinePattern.MatchString(l) {
			index = i
			break
		}
	}
	if index < 0 {
		lines = append([]string{line}, lines...)
	} else {
		stop := index + 1
		for stop < len(lines) && (strings.HasPrefix(lines[stop], " ") || strings.HasPrefix(lines[stop], "\t")) {
			stop++
		}
		lines = append(append(append([]string{}, lines[:index]...), line), lines[stop:]...)
	}
	return open + strings.Join(lines, eol) + end + raw[len(whole):]
}

func eveFrontmatter(raw string) (string, error) {
	data, content, err := parseFrontmatter(raw)
	if err != nil {
		return "", err
	}
	content = leadingEOLPattern.ReplaceAllString(content, "")
	mapping := &yaml.Node{Kind: yaml.MappingNode}
	add := func(key string, value any) error {
		var k, v yaml.Node
		if err := k.Encode(key); err != nil {
			return err
		}
		if err := v.Encode(value); err != nil {
			return err
		}
		mapping.Content = append(mapping.Content, &k, &v)
		return nil
	}
	for _, key := range []string{"name", "description", "license", "compatibility", "version", "metadata"} {
		value, present := data[key]
		keep := false
		switch key {
		case "compatibility":
			keep = present
		case "version":
			keep = jsTypeof(value) == "string" || jsTypeof(value) == "number"
		case "metadata":
			_, keep = value.(map[string]any)
		default:
			_, keep = value.(string)
		}
		if !keep {
			continue
		}
		if err := add(key, value); err != nil {
			return "", err
		}
	}
	if len(mapping.Content) == 0 {
		return content, nil
	}
	text, err := yamlText(mapping)
	if err != nil {
		return "", err
	}
	return "---\n" + text + "\n---\n" + content, nil
}

func skillCopyExcluded(name string, dir bool) bool {
	return name == "metadata.json" || dir && (name == ".git" || name == "__pycache__" || name == "__pypackages__")
}

func withinRoot(root, target string) bool {
	return target == root || strings.HasPrefix(target, root+string(filepath.Separator))
}

func skillsComputedHash(dir string) (string, error) {
	root, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return "", err
	}
	type file struct {
		path    string
		content []byte
	}
	var files []file
	var walk func(dir, prefix string, ancestors map[string]bool) error
	walk = func(dir, prefix string, ancestors map[string]bool) error {
		real, err := filepath.EvalSymlinks(dir)
		if err != nil {
			return err
		}
		if ancestors[real] {
			return nil
		}
		branch := maps.Clone(ancestors)
		branch[real] = true
		entries, err := os.ReadDir(dir)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			name := entry.Name()
			isDir := entry.IsDir()
			if skillCopyExcluded(name, isDir) {
				continue
			}
			full := filepath.Join(dir, name)
			if entry.Type()&fs.ModeSymlink != 0 {
				target, err := filepath.EvalSymlinks(full)
				if err != nil || !withinRoot(root, target) {
					continue
				}
				info, err := os.Stat(target)
				if err != nil {
					return err
				}
				if isDir = info.IsDir(); isDir && skillCopyExcluded(name, true) {
					continue
				}
			} else if !isDir && !entry.Type().IsRegular() {
				continue
			}
			if isDir {
				if err := walk(full, prefix+name+"/", branch); err != nil {
					return err
				}
				continue
			}
			b, err := os.ReadFile(full)
			if err != nil {
				return err
			}
			if !bytes.Contains(b, []byte{0}) && bytes.Contains(b, []byte("\r\n")) {
				b = bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))
			}
			files = append(files, file{prefix + name, b})
		}
		return nil
	}
	if err := walk(dir, "", map[string]bool{}); err != nil {
		return "", err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].path < files[j].path })
	sum := sha256.New()
	for _, f := range files {
		sum.Write([]byte(f.path))
		sum.Write([]byte{0})
		sum.Write([]byte(strconv.Itoa(len(f.content))))
		sum.Write([]byte{0})
		sum.Write(f.content)
	}
	return hex.EncodeToString(sum.Sum(nil)), nil
}
