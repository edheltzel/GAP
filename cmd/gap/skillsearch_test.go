package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSearchSkills(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		switch r.URL.Query().Get("q") {
		case "broken":
			w.Write([]byte("{"))
		case "down":
			http.Error(w, "nope", http.StatusBadGateway)
		default:
			w.Write([]byte(`{"skills":[
				{"id":"a/b/low","source":"a/b","skillId":"low","name":"low","installs":5},
				{"id":"open.feishu.cn/lark","source":"open.feishu.cn","skillId":"lark","name":"lark","installs":900},
				{"id":"vercel-labs/agent-skills/react","source":"vercel-labs/agent-skills","skillId":"react","name":"react","installs":100},
				{"id":"x/next.js/ok","source":"x/next.js","skillId":"ok","name":"ok","installs":50}
			]}`))
		}
	}))
	defer srv.Close()
	old := skillSearchURL
	skillSearchURL = srv.URL
	t.Cleanup(func() { skillSearchURL = old })
	hits, err := searchSkills(context.Background(), "react hooks", 20)
	if err != nil {
		t.Fatal(err)
	}
	if gotQuery != "limit=20&q=react+hooks" {
		t.Fatalf("query %q", gotQuery)
	}
	want := []skillHit{
		{ID: "vercel-labs/agent-skills/react", Source: "vercel-labs/agent-skills", Skill: "react", Name: "react", Installs: 100, Trusted: true},
		{ID: "x/next.js/ok", Source: "x/next.js", Skill: "ok", Name: "ok", Installs: 50},
		{ID: "a/b/low", Source: "a/b", Skill: "low", Name: "low", Installs: 5},
	}
	if len(hits) != len(want) {
		t.Fatalf("hits %+v", hits)
	}
	for i := range want {
		if hits[i] != want[i] {
			t.Fatalf("hit %d = %+v, want %+v", i, hits[i], want[i])
		}
	}
	for _, q := range []string{"broken", "down"} {
		if _, err := searchSkills(context.Background(), q, 5); err == nil {
			t.Fatalf("%s: no error", q)
		}
	}
}
