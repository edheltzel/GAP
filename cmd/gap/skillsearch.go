package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strings"
	"time"
)

var githubName = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

var skillSearchURL = "https://skills.sh/api/search"

func init() {
	if v := os.Getenv("GAP_SKILL_SEARCH_URL"); v != "" {
		skillSearchURL = v
	}
}

var trustedSkillSources = []string{"vercel-labs", "anthropics", "microsoft", "google-labs-code"}

type skillHit struct {
	ID       string
	Source   string
	Skill    string
	Name     string
	Installs int
	Trusted  bool
}

// searchSkills asks skills.sh for skills matching query. Only hits from an owner/repo source are kept,
// because gap add installs from a GitHub repo.
func searchSkills(ctx context.Context, query string, limit int) ([]skillHit, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	u := skillSearchURL + "?" + url.Values{"q": {query}, "limit": {fmt.Sprint(limit)}}.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("skills.sh search: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("skills.sh search: %s", resp.Status)
	}
	var body struct {
		Skills []struct {
			ID       string `json:"id"`
			Source   string `json:"source"`
			SkillID  string `json:"skillId"`
			Name     string `json:"name"`
			Installs int    `json:"installs"`
		} `json:"skills"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("skills.sh search: %w", err)
	}
	var hits []skillHit
	for _, s := range body.Skills {
		owner, repo, ok := strings.Cut(s.Source, "/")
		if !ok || !githubName.MatchString(owner) || strings.Contains(owner, ".") || !githubName.MatchString(repo) || repo == "." || repo == ".." || !githubName.MatchString(s.SkillID) {
			continue
		}
		hits = append(hits, skillHit{ID: s.ID, Source: s.Source, Skill: s.SkillID, Name: s.Name, Installs: s.Installs, Trusted: slices.Contains(trustedSkillSources, owner)})
	}
	slices.SortStableFunc(hits, func(a, b skillHit) int { return b.Installs - a.Installs })
	return hits, nil
}
