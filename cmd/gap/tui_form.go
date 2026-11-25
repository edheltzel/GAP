package main

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
)

const (
	scopeGlobal  = "global"
	scopeProject = "project"
)

func huhTheme(p palette) huh.ThemeFunc {
	return func(isDark bool) *huh.Styles {
		t := huh.ThemeBase(isDark)
		t.Focused.Base = t.Focused.Base.BorderForeground(p.Accent)
		t.Focused.Card = t.Focused.Base
		t.Focused.Title = t.Focused.Title.Foreground(p.Accent).Bold(true)
		t.Focused.Description = t.Focused.Description.Foreground(p.Dim)
		t.Focused.ErrorIndicator = t.Focused.ErrorIndicator.Foreground(p.Bad)
		t.Focused.ErrorMessage = t.Focused.ErrorMessage.Foreground(p.Bad)
		t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(p.OK)
		t.Focused.Option = t.Focused.Option.Foreground(p.Text)
		t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(p.OK)
		t.Focused.UnselectedOption = t.Focused.UnselectedOption.Foreground(p.Text)
		t.Focused.FocusedButton = t.Focused.FocusedButton.Foreground(p.OnAcc).Background(p.Accent)
		t.Focused.Next = t.Focused.FocusedButton
		t.Focused.BlurredButton = t.Focused.BlurredButton.Foreground(p.Text).Background(p.Border)
		t.Focused.TextInput.Cursor = t.Focused.TextInput.Cursor.Foreground(p.OK)
		t.Focused.TextInput.Placeholder = t.Focused.TextInput.Placeholder.Foreground(p.Dim)
		t.Focused.TextInput.Prompt = t.Focused.TextInput.Prompt.Foreground(p.Accent)
		t.Blurred = t.Focused
		t.Blurred.Base = t.Focused.Base.BorderStyle(lipgloss.HiddenBorder())
		t.Blurred.Card = t.Blurred.Base
		t.Group.Title = t.Focused.Title
		t.Group.Description = t.Focused.Description
		return t
	}
}

type formState struct {
	title  string
	form   *huh.Form
	submit func() tea.Cmd
}

// project is the folder the TUI was opened in, or "" when it is home or unknown.
func (m *tuiModel) here() string {
	if m.projectRoot == "" || filepath.Clean(m.projectRoot) == filepath.Clean(m.home) {
		return ""
	}
	return m.projectRoot
}

func (m *tuiModel) scopeOptions(project string) []huh.Option[string] {
	opts := []huh.Option[string]{huh.NewOption("Everywhere    "+m.short(userConfigFile(m.home)), scopeGlobal)}
	if project != "" {
		opts = append(opts, huh.NewOption("This project  "+filepath.Join(filepath.Base(project), ".gap", "gap.toml"), scopeProject))
	}
	return opts
}

func (m *tuiModel) short(p string) string {
	if root := m.here(); root != "" {
		if rel, err := filepath.Rel(root, p); err == nil && !strings.HasPrefix(rel, "..") {
			return filepath.Join(filepath.Base(root), rel)
		}
	}
	if rel, err := filepath.Rel(m.home, p); err == nil && !strings.HasPrefix(rel, "..") {
		return filepath.Join("~", rel)
	}
	return p
}

func (m *tuiModel) openForm(title string, submit func() tea.Cmd, fields ...huh.Field) tea.Cmd {
	f := huh.NewForm(huh.NewGroup(fields...)).
		WithShowHelp(true).
		WithTheme(huhTheme(m.st.pal)).
		WithWidth(max(min(m.width-6, 90), 20))
	m.formState = &formState{title: title, form: f, submit: submit}
	m.mode = modeForm
	return f.Init()
}

func (m *tuiModel) formUpdate(msg tea.Msg) tea.Cmd {
	fs := m.formState
	if k, ok := msg.(tea.KeyPressMsg); ok && key.Matches(k, m.keys.Back) {
		m.formState, m.mode, m.note = nil, modeBrowse, "Nothing changed."
		return nil
	}
	model, cmd := fs.form.Update(msg)
	if f, ok := model.(*huh.Form); ok {
		fs.form = f
	}
	switch fs.form.State {
	case huh.StateCompleted:
		m.formState, m.mode = nil, modeBrowse
		return tea.Batch(cmd, fs.submit())
	case huh.StateAborted:
		m.formState, m.mode, m.note = nil, modeBrowse, "Nothing changed."
		return nil
	}
	return cmd
}

func (m *tuiModel) formView() string {
	fs := m.formState
	head := m.st.title.Render(" "+fs.title+" ") + "  " + m.st.faint.Render("esc cancels")
	return m.st.pane.Width(m.width).Height(m.bodyHeight()).Render(head + "\n\n" + fs.form.View())
}

// addForm asks for a repo and where it goes, then hands the terminal to gap add so its trust prompt works.
func (m *tuiModel) addForm(project string) tea.Cmd {
	var spec, id string
	scope := scopeGlobal
	if project != "" && m.tab == tabProjects {
		scope = scopeProject
	}
	fields := []huh.Field{
		huh.NewInput().Title("Repo").Description("owner/repo, owner/repo@skill, owner/repo#ref, or a git URL").
			Placeholder("pbakaus/agent-reviews").Value(&spec).
			Validate(func(s string) error {
				if strings.TrimSpace(s) == "" {
					return errors.New("enter a repo")
				}
				_, err := parseSkillSource(strings.TrimSpace(s), cmp.Or(project, m.home))
				return err
			}),
		huh.NewInput().Title("ID").Description("Leave empty to use the repo name.").Value(&id).
			Validate(func(s string) error {
				if s != "" && !validID(s) {
					return errors.New("lowercase letters, digits, dot, dash, or underscore")
				}
				return nil
			}),
	}
	if project != "" {
		fields = append(fields, huh.NewSelect[string]().Title("Install for").Options(m.scopeOptions(project)...).Value(&scope))
	}
	return m.openForm("Add a utility", func() tea.Cmd {
		args := []string{"add", strings.TrimSpace(spec), "--global"}
		if id != "" {
			args = append(args, "--id", id)
		}
		dir := ""
		if scope == scopeProject {
			args[2], dir = "--project", project
		}
		return m.handOverIn(dir, args...)
	}, fields...)
}

// editForm edits one item's settings in the global or this project's gap.toml.
func (m *tuiModel) editForm(r itemRow) tea.Cmd {
	project := m.here()
	scope := scopeGlobal
	if r.Scope == "project" && project != "" {
		scope = scopeProject
	}
	enabled, hold, ref, hue := !r.Disabled, r.Held, r.Ref, r.Color
	fields := []huh.Field{}
	if project != "" {
		fields = append(fields, huh.NewSelect[string]().Title("Save to").
			Description("Enabled, hold, and color can differ per project. The version is the same everywhere.").
			Options(m.scopeOptions(project)...).Value(&scope))
	}
	fields = append(fields,
		huh.NewConfirm().Title("Enabled").Affirmative("On").Negative("Off").Value(&enabled),
		huh.NewConfirm().Title("Hold updates").Description("gap update and gap check skip it unless you name it.").Affirmative("Hold").Negative("Follow").Value(&hold),
		huh.NewInput().Title("Version").Description("A branch, tag, or commit. Changing it moves the pin now.").Value(&ref).
			Validate(func(s string) error {
				if strings.ContainsAny(strings.TrimSpace(s), " @") {
					return errors.New("enter a branch, tag, or commit")
				}
				return nil
			}),
		huh.NewInput().Title("Color").Description("#RRGGBB, or empty for the palette color.").Placeholder("#B392D7").Value(&hue).
			Validate(func(s string) error {
				if s != "" && !validColor(s) {
					return errors.New("use #RRGGBB")
				}
				return nil
			}),
	)
	return m.openForm("Edit "+r.ID, func() tea.Cmd {
		path := userConfigFile(m.home)
		if scope == scopeProject {
			path = projectConfigFile(project)
		}
		var set []configField
		if enabled == r.Disabled {
			set = append(set, configField{"enabled", enabled})
		}
		if hold != r.Held {
			set = append(set, configField{"pin", hold})
		}
		if hue = strings.ToUpper(hue); hue != strings.ToUpper(r.Color) {
			var v any
			if hue != "" {
				v = hue
			}
			set = append(set, configField{"color", v})
		}
		if ref = strings.TrimSpace(ref); ref == "" {
			ref = r.Ref
		}
		if err := saveItem(m.home, path, scope == scopeProject, project, r.ID, set); err != nil {
			m.note = err.Error()
			return nil
		}
		if ref != r.Ref {
			if err := saveItem(m.home, userConfigFile(m.home), false, "", r.ID, []configField{{"ref", ref}}); err != nil {
				m.note = err.Error()
				return nil
			}
			if r.user() {
				return m.handOverIn("", "update", r.ID)
			}
			return m.handOverIn("", "pin", r.ID+"@"+ref)
		}
		m.note = "Nothing changed."
		if len(set) > 0 {
			m.note = "Saved to " + m.short(path) + ". s syncs it."
		}
		return m.reload()
	}, fields...)
}

func saveItem(home, path string, project bool, root, id string, set []configField) error {
	if len(set) == 0 {
		return nil
	}
	if project {
		if err := ensureProjectConfig(root); err != nil {
			return err
		}
	}
	return setComponentFields(path, id, set)
}

func ensureProjectConfig(root string) error {
	path := projectConfigFile(root)
	if _, err := os.Stat(path); err == nil || !os.IsNotExist(err) {
		return err
	}
	if err := writeAtomic(path, "profile = \"project\"\nslug = "+fmt.Sprintf("%q", filepath.Base(root))+"\n"); err != nil {
		return err
	}
	noteProject(root)
	return nil
}

// editorForm picks which gap.toml to open, then opens it in $VISUAL or $EDITOR.
func (m *tuiModel) editorForm(project string) tea.Cmd {
	if project == "" {
		return m.openEditor(userConfigFile(m.home), "")
	}
	scope := scopeGlobal
	if m.tab == tabProjects {
		scope = scopeProject
	}
	return m.openForm("Open in editor", func() tea.Cmd {
		if scope == scopeProject {
			return m.openEditor(projectConfigFile(project), project)
		}
		return m.openEditor(userConfigFile(m.home), "")
	}, huh.NewSelect[string]().Title("File").Options(m.scopeOptions(project)...).Value(&scope))
}

func (m *tuiModel) openEditor(path, project string) tea.Cmd {
	if project != "" {
		if err := ensureProjectConfig(project); err != nil {
			m.note = err.Error()
			return nil
		}
	} else if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		m.note = err.Error()
		return nil
	}
	editor := strings.Fields(cmp.Or(os.Getenv("VISUAL"), os.Getenv("EDITOR"), "vi"))
	cmd := exec.Command(editor[0], append(editor[1:], path)...)
	return tea.ExecProcess(cmd, func(err error) tea.Msg { return handedBackMsg{err: err, edited: path} })
}
