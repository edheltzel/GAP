package main

import (
	"cmp"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

type skillSource struct {
	Kind, URL, Ref, Subpath, Skill, Local string
}

var (
	versionPinPattern      = regexp.MustCompile(`^v?\d+(?:\.\d+)+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)
	commitSHAPattern       = regexp.MustCompile(`(?i)^[0-9a-f]{40}$`)
	sshGitPattern          = regexp.MustCompile(`(?i)^ssh://.+\.git(?:$|[/?])`)
	httpGitPattern         = regexp.MustCompile(`(?i)^https?://.+\.git(?:$|[/?])`)
	githubRepoPathPattern  = regexp.MustCompile(`^/[^/]+/[^/]+(?:\.git)?(?:/tree/[^/]+(?:/.*)?)?/?$`)
	gitlabRepoPathPattern  = regexp.MustCompile(`^/.+?/[^/]+(?:\.git)?(?:/-/tree/[^/]+(?:/.*)?)?/?$`)
	shorthandLikePattern   = regexp.MustCompile(`^([^/]+)/([^/]+)(?:/(.+)|@(.+))?$`)
	hostedArtifactPattern  = regexp.MustCompile(`^/[^/]+/[^/]+/(?:archive/|raw/|releases/(?:download/|latest/download/))`)
	gitlabArtifactPattern  = regexp.MustCompile(`/-/(?:archive|raw)/`)
	githubSSHTreePattern   = regexp.MustCompile(`^git@([^:]+):([^/]+)/([^/]+?)(?:\.git)?/tree/([^/]+)(?:/(.+))?$`)
	pathOnHostPattern      = regexp.MustCompile(`(?i)^(?:https?://)?([^/?#]*)([^?#]*)`)
	githubTreePattern      = regexp.MustCompile(`^/([^/]+)/([^/]+)/tree/([^/]+)(?:/(.+?))?/?$`)
	githubRepoPattern      = regexp.MustCompile(`^/([^/]+)/([^/]+)`)
	gitlabTreePathPattern  = regexp.MustCompile(`^(https?)://([^/]+)/(.+?)/-/tree/([^/]+)/(.+)`)
	gitlabTreePattern      = regexp.MustCompile(`^(https?)://([^/]+)/(.+?)/-/tree/([^/]+)$`)
	gitlabRepoPattern      = regexp.MustCompile(`^/(.+?)(?:\.git)?/?$`)
	mirroredHostPattern    = regexp.MustCompile(`(?i)^(/(?:.*?/)?(?:https?://?)?(?:www\.)?(github|gitlab)\.com)(/.*)$`)
	mirroredGitHubPattern  = regexp.MustCompile(`^(/[^/]+/[^/]+?)(?:\.git)?(?:/tree/([^/]+)(?:/(.+?))?)?/?$`)
	mirroredGitLabPattern  = regexp.MustCompile(`^(/[^/]+(?:/[^/]+)*?/[^/]+?)(?:\.git)?(?:/-/tree/([^/]+)(?:/(.+?))?)?/?$`)
	azureVersionPattern    = regexp.MustCompile(`(?i)^(?:GB|GT)(.+)$`)
	atSkillPattern         = regexp.MustCompile(`^([^/]+)/([^/@]+)@(.+)$`)
	shorthandPattern       = regexp.MustCompile(`^([^/]+)/([^/]+)(?:/(.+?))?/?$`)
	notionHostPattern      = regexp.MustCompile(`(^|\.)notion\.(so|com)$`)
	notionPageIDPattern    = regexp.MustCompile(`(?:^|-)([0-9a-f]{32}|[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})$`)
	windowsDrivePattern    = regexp.MustCompile(`^[a-zA-Z]:[/\\]`)
	userinfoPattern        = regexp.MustCompile(`^[^@]*@`)
	portSuffixPattern      = regexp.MustCompile(`:\d*$`)
	scpRemotePattern       = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._~-]*@[A-Za-z0-9][A-Za-z0-9.-]*:.+$`)
	githubHTTPSRepoPattern = regexp.MustCompile(`^https://([^/]+)/([^/]+)/([^/]+?)(?:\.git)?/?$`)
)

var sourceAliases = map[string]string{
	"coinbase/agentWallet":      "coinbase/agentic-wallet-skills",
	"vercel-labs/vercel-skills": "vercel-labs/agent-skills",
}

func webURL(input string) (*url.URL, bool) {
	u, err := url.Parse(input)
	if err != nil || u.Scheme == "" {
		return nil, false
	}
	u.Host = strings.ToLower(u.Host)
	if u.Scheme == "https" {
		u.Host = strings.TrimSuffix(u.Host, ":443")
	} else if u.Scheme == "http" {
		u.Host = strings.TrimSuffix(u.Host, ":80")
	}
	return u, true
}

func webPath(u *url.URL) string {
	if p := u.EscapedPath(); p != "" {
		return p
	}
	return "/"
}

func githubHost() string {
	host := strings.TrimSpace(os.Getenv("GH_HOST"))
	if host == "" {
		return "github.com"
	}
	u, err := url.Parse("https://" + host)
	if err != nil || u.Hostname() == "" {
		return "github.com"
	}
	password, _ := u.User.Password()
	if u.User.Username() != "" || password != "" || u.Port() != "" && u.Port() != "443" || u.Path != "" && u.Path != "/" || u.RawQuery != "" || u.Fragment != "" {
		return "github.com"
	}
	return strings.ToLower(u.Hostname())
}

func isGitHubHost(host string) bool {
	host = strings.ToLower(host)
	return host == "github.com" || host == githubHost()
}

func isVersionPin(value string) bool {
	return versionPinPattern.MatchString(value) || commitSHAPattern.MatchString(value)
}

func decodeURIComponent(value string) string {
	if decoded, err := url.PathUnescape(value); err == nil {
		return decoded
	}
	return value
}

func encodeURIComponent(value string) string {
	var b strings.Builder
	for _, c := range []byte(value) {
		if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || strings.IndexByte("-_.!~*'()", c) >= 0 {
			b.WriteByte(c)
		} else {
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}

func mirroredRepoPath(pathname string) (base, ref, subpath string, ok bool) {
	m := mirroredHostPattern.FindStringSubmatch(pathname)
	if m == nil {
		return "", "", "", false
	}
	repo := mirroredGitLabPattern
	if strings.EqualFold(m[2], "github") {
		repo = mirroredGitHubPattern
	}
	r := repo.FindStringSubmatch(m[3])
	if r == nil {
		return "", "", "", false
	}
	return m[1] + r[1], r[2], r[3], true
}

func looksLikeGitSource(input string) bool {
	if strings.HasPrefix(input, "github:") || strings.HasPrefix(input, "gitlab:") || strings.HasPrefix(input, "git@") || sshGitPattern.MatchString(input) {
		return true
	}
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		if u, ok := webURL(input); ok {
			pathname := webPath(u)
			if isGitHubHost(u.Host) {
				return githubRepoPathPattern.MatchString(pathname)
			}
			if u.Hostname() == "gitlab.com" {
				return gitlabRepoPathPattern.MatchString(pathname)
			}
			if _, _, _, ok := mirroredRepoPath(pathname); ok {
				return true
			}
			if azureGitIndex(pathname) >= 0 {
				return true
			}
		}
	}
	if httpGitPattern.MatchString(input) {
		return true
	}
	return !strings.Contains(input, ":") && !strings.HasPrefix(input, ".") && !strings.HasPrefix(input, "/") && shorthandLikePattern.MatchString(input)
}

func azureGitIndex(pathname string) int {
	segments := slices.DeleteFunc(strings.Split(pathname, "/"), func(s string) bool { return s == "" })
	i := slices.Index(segments, "_git")
	if i < 0 || i == len(segments)-1 {
		return -1
	}
	return i
}

func parseFragmentRef(input string) (string, string, string) {
	i := strings.Index(input, "#")
	if i < 0 {
		return input, "", ""
	}
	without, fragment := input[:i], input[i+1:]
	if fragment == "" || !looksLikeGitSource(without) {
		return input, "", ""
	}
	ref, skill, found := strings.Cut(fragment, "@")
	if !found {
		return without, decodeURIComponent(fragment), ""
	}
	if ref != "" {
		ref = decodeURIComponent(ref)
	}
	if skill != "" {
		skill = decodeURIComponent(skill)
	}
	return without, ref, skill
}

func appendFragmentRef(input, ref, skill string) string {
	if ref == "" && skill == "" {
		return input
	}
	out := input + "#"
	if ref != "" {
		out += encodeURIComponent(ref)
	}
	if skill != "" {
		out += "@" + encodeURIComponent(skill)
	}
	return out
}

func hostedArtifact(input string) bool {
	u, ok := webURL(input)
	if !ok {
		return false
	}
	switch u.Hostname() {
	case "raw.githubusercontent.com", "codeload.github.com", "objects.githubusercontent.com":
		return true
	case "github.com":
		return hostedArtifactPattern.MatchString(webPath(u))
	case "gitlab.com":
		return gitlabArtifactPattern.MatchString(webPath(u))
	}
	return false
}

func pathOnHost(input, host string) (string, bool) {
	m := pathOnHostPattern.FindStringSubmatch(input)
	hostname := strings.ToLower(portSuffixPattern.ReplaceAllString(userinfoPattern.ReplaceAllString(m[1], ""), ""))
	if hostname == host || hostname == "www."+host {
		return m[2], true
	}
	return "", false
}

func notionSource(input string) bool {
	if strings.EqualFold(input, "notion") {
		return true
	}
	u, err := url.Parse(input)
	if err != nil || u.Scheme != "https" && u.Scheme != "http" || !notionHostPattern.MatchString(strings.ToLower(u.Hostname())) {
		return false
	}
	segments := slices.DeleteFunc(strings.Split(u.EscapedPath(), "/"), func(s string) bool { return s == "" })
	if len(segments) == 0 {
		return false
	}
	return notionPageIDPattern.MatchString(strings.ToLower(decodeURIComponent(segments[len(segments)-1])))
}

func refuseSource(input, kind string) error {
	return fmt.Errorf("gap add installs from git repositories and local folders; %s is a %s source", input, kind)
}

func isLocalSource(input string) bool {
	return filepath.IsAbs(input) || strings.HasPrefix(input, "./") || strings.HasPrefix(input, "../") || input == "." || input == ".." || windowsDrivePattern.MatchString(input)
}

func parseSkillSource(input, cwd string) (skillSource, error) {
	if notionSource(input) {
		return skillSource{}, refuseSource(input, "Notion")
	}
	return parseSourceInput(input, input, cwd)
}

func withSubpath(s skillSource, subpath string) (skillSource, error) {
	if subpath == "" {
		return s, nil
	}
	if err := unsafeSubpath(subpath); err != nil {
		return skillSource{}, err
	}
	s.Subpath = subpath
	return s, nil
}

func parseSourceInput(input, original, cwd string) (skillSource, error) {
	if isLocalSource(input) {
		path := input
		if !filepath.IsAbs(path) {
			path = filepath.Join(cwd, path)
		}
		path = filepath.Clean(path)
		return skillSource{Kind: "local", URL: path, Local: path}, nil
	}
	input, fragmentRef, fragmentSkill := parseFragmentRef(input)
	if alias, ok := sourceAliases[input]; ok {
		input = alias
	}
	if rest, ok := strings.CutPrefix(input, "github:"); ok && rest != "" {
		return parseSourceInput(appendFragmentRef(rest, fragmentRef, fragmentSkill), original, cwd)
	}
	if rest, ok := strings.CutPrefix(input, "gitlab:"); ok && rest != "" {
		return parseSourceInput(appendFragmentRef("https://gitlab.com/"+rest, fragmentRef, fragmentSkill), original, cwd)
	}
	if hostedArtifact(input) {
		return skillSource{}, refuseSource(original, "download")
	}
	if host := githubHost(); host != "github.com" && (strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://")) {
		if u, ok := webURL(input); ok && isGitHubHost(u.Host) && u.Host != "github.com" {
			parts := slices.DeleteFunc(strings.Split(webPath(u), "/"), func(s string) bool { return s == "" })
			if len(parts) >= 2 {
				s := skillSource{Kind: "git", URL: u.Scheme + "://" + u.Host + "/" + parts[0] + "/" + strings.TrimSuffix(parts[1], ".git") + ".git", Ref: fragmentRef}
				if len(parts) >= 4 && parts[2] == "tree" {
					s.Ref = parts[3]
					return withSubpath(s, strings.Join(parts[4:], "/"))
				}
				return s, nil
			}
		}
	}
	if m := githubSSHTreePattern.FindStringSubmatch(input); m != nil && isGitHubHost(m[1]) {
		return withSubpath(skillSource{Kind: "git", URL: "git@" + m[1] + ":" + m[2] + "/" + m[3] + ".git", Ref: m[4]}, m[5])
	}
	if githubPath, ok := pathOnHost(input, "github.com"); ok {
		if m := githubTreePattern.FindStringSubmatch(githubPath); m != nil {
			return withSubpath(skillSource{Kind: "github", URL: "https://github.com/" + m[1] + "/" + m[2] + ".git", Ref: cmp.Or(m[3], fragmentRef)}, m[4])
		}
		if m := githubRepoPattern.FindStringSubmatch(githubPath); m != nil {
			return skillSource{Kind: "github", URL: "https://github.com/" + m[1] + "/" + strings.TrimSuffix(m[2], ".git") + ".git", Ref: fragmentRef}, nil
		}
	}
	if m := gitlabTreePathPattern.FindStringSubmatch(input); m != nil && m[2] != "github.com" && m[3] != "" {
		return withSubpath(skillSource{Kind: "gitlab", URL: m[1] + "://" + m[2] + "/" + strings.TrimSuffix(m[3], ".git") + ".git", Ref: cmp.Or(m[4], fragmentRef)}, m[5])
	}
	if m := gitlabTreePattern.FindStringSubmatch(input); m != nil && m[2] != "github.com" && m[3] != "" {
		return skillSource{Kind: "gitlab", URL: m[1] + "://" + m[2] + "/" + strings.TrimSuffix(m[3], ".git") + ".git", Ref: cmp.Or(m[4], fragmentRef)}, nil
	}
	if gitlabPath, ok := pathOnHost(input, "gitlab.com"); ok {
		if m := gitlabRepoPattern.FindStringSubmatch(gitlabPath); m != nil && strings.Contains(m[1], "/") {
			return skillSource{Kind: "gitlab", URL: "https://gitlab.com/" + m[1] + ".git", Ref: fragmentRef}, nil
		}
	}
	if s, ok, err := mirroredSource(input, fragmentRef, fragmentSkill); ok || err != nil {
		return s, err
	}
	if s, ok, err := azureSource(input, fragmentRef, fragmentSkill); ok || err != nil {
		return s, err
	}
	host := githubHost()
	kind := "github"
	if host != "github.com" {
		kind = "git"
	}
	plain := !strings.Contains(input, ":") && !strings.HasPrefix(input, ".") && !strings.HasPrefix(input, "/")
	if m := atSkillPattern.FindStringSubmatch(input); m != nil && plain {
		s := skillSource{Kind: kind, URL: "https://" + host + "/" + m[1] + "/" + m[2] + ".git"}
		if fragmentRef == "" && fragmentSkill == "" && isVersionPin(m[3]) {
			s.Ref = m[3]
			return s, nil
		}
		s.Ref, s.Skill = fragmentRef, cmp.Or(fragmentSkill, m[3])
		return s, nil
	}
	if m := shorthandPattern.FindStringSubmatch(input); m != nil && plain {
		return withSubpath(skillSource{Kind: kind, URL: "https://" + host + "/" + m[1] + "/" + m[2] + ".git", Ref: fragmentRef, Skill: fragmentSkill}, m[3])
	}
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		if u, ok := webURL(input); ok && !slices.Contains([]string{"github.com", "gitlab.com", "raw.githubusercontent.com"}, u.Hostname()) && !strings.HasSuffix(input, ".git") {
			return skillSource{}, refuseSource(original, "well-known")
		}
	}
	if len(input) >= 5 && strings.EqualFold(input[:5], "ext::") {
		return skillSource{}, fmt.Errorf("Unsupported Git transport: ext")
	}
	return skillSource{Kind: "git", URL: input, Ref: fragmentRef}, nil
}

func mirroredSource(input, fragmentRef, fragmentSkill string) (skillSource, bool, error) {
	if !strings.HasPrefix(strings.ToLower(input), "http://") && !strings.HasPrefix(strings.ToLower(input), "https://") {
		return skillSource{}, false, nil
	}
	u, ok := webURL(input)
	if !ok {
		return skillSource{}, false, nil
	}
	base, ref, subpath, ok := mirroredRepoPath(webPath(u))
	if !ok {
		return skillSource{}, false, nil
	}
	u.RawPath = base + ".git"
	u.Path = decodeURIComponent(u.RawPath)
	u.RawQuery, u.ForceQuery, u.Fragment, u.RawFragment = "", false, "", ""
	s := skillSource{Kind: "git", URL: u.String(), Ref: fragmentRef, Skill: fragmentSkill}
	if ref != "" {
		s.Ref = decodeURIComponent(ref)
	}
	s, err := withSubpath(s, decodeURIComponent(subpath))
	return s, true, err
}

func azureSource(input, fragmentRef, fragmentSkill string) (skillSource, bool, error) {
	if !strings.HasPrefix(input, "http://") && !strings.HasPrefix(input, "https://") {
		return skillSource{}, false, nil
	}
	u, ok := webURL(input)
	if !ok {
		return skillSource{}, false, nil
	}
	segments := slices.DeleteFunc(strings.Split(webPath(u), "/"), func(s string) bool { return s == "" })
	i := azureGitIndex(webPath(u))
	if i < 0 {
		return skillSource{}, false, nil
	}
	repo := strings.TrimSuffix(segments[i+1], ".git")
	if repo == "" {
		return skillSource{}, false, nil
	}
	var encoded []string
	for _, segment := range append(slices.Clone(segments[:i]), "_git", repo) {
		encoded = append(encoded, encodeURIComponent(decodeURIComponent(segment)))
	}
	s := skillSource{Kind: "git", URL: u.Scheme + "://" + u.Host + "/" + strings.Join(encoded, "/"), Ref: fragmentRef, Skill: fragmentSkill}
	query := u.Query()
	if m := azureVersionPattern.FindStringSubmatch(query.Get("version")); m != nil {
		s.Ref = m[1]
	}
	s, err := withSubpath(s, strings.TrimLeft(query.Get("path"), "/"))
	return s, true, err
}

func githubSSHFallback(remote string) (string, string) {
	m := githubHTTPSRepoPattern.FindStringSubmatch(remote)
	if m == nil || !isGitHubHost(m[1]) {
		return "", ""
	}
	return "git@" + m[1] + ":" + m[2] + "/" + m[3] + ".git", m[1]
}

func gitAuthFailure(message string) bool {
	lower := strings.ToLower(message)
	for _, s := range []string{"Authentication failed", "could not read Username", "Permission denied", "Repository not found", "requested URL returned error: 403"} {
		if strings.Contains(message, s) {
			return true
		}
	}
	for _, s := range []string{"saml sso", "enforced sso", "enabled or enforced saml", "re-authorize the oauth application"} {
		if strings.Contains(lower, s) {
			return true
		}
	}
	return false
}

func sourceRepo(s skillSource) string {
	if s.Kind == "github" {
		return "github.com/" + sourceLabel(s)
	}
	return sourceLabel(s)
}

func sourceLabel(s skillSource) string {
	if s.Kind == "github" {
		return strings.TrimSuffix(strings.TrimPrefix(s.URL, "https://github.com/"), ".git")
	}
	return s.URL
}

func sourceID(repo string) string {
	repo = strings.TrimSuffix(strings.TrimRight(repo, "/"), ".git")
	if i := strings.LastIndexAny(repo, "/:"); i >= 0 {
		repo = repo[i+1:]
	}
	return strings.ToLower(repo)
}
