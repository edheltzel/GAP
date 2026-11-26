package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func gitT(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false"}, args...)...)
	cmd.Dir = dir
	cmd.Env = gitEnv()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

type selfUpdateEnv struct {
	origin, checkout, exe string
	built                 []string
}

func newSelfUpdateEnv(t *testing.T) *selfUpdateEnv {
	t.Helper()
	e := &selfUpdateEnv{origin: filepath.Join(t.TempDir(), "origin"), checkout: filepath.Join(t.TempDir(), "checkout")}
	write(t, filepath.Join(e.origin, "global", "AGENTS.md"), "v1\n")
	gitT(t, e.origin, "init", "-q", "-b", "master")
	gitT(t, e.origin, "add", "-A")
	gitT(t, e.origin, "commit", "-q", "-m", "v1")
	gitT(t, t.TempDir(), "clone", "-q", e.origin, e.checkout)
	e.exe = filepath.Join(t.TempDir(), "gap")
	write(t, e.exe, "old binary")
	oldRoot, oldSelf, oldBuild, oldVersion := sourceRootFor, selfPath, buildGap, version
	sourceRootFor = e.checkout
	selfPath = func() (string, error) { return e.exe, nil }
	buildGap = func(root, source, out string) error {
		if source != e.checkout {
			t.Errorf("binary would look for its checkout at %s, want %s", source, e.checkout)
		}
		e.built = append(e.built, root)
		return os.WriteFile(out, []byte("built from "+gitT(t, root, "rev-parse", "HEAD")), 0o644)
	}
	t.Cleanup(func() { sourceRootFor, selfPath, buildGap, version = oldRoot, oldSelf, oldBuild, oldVersion })
	return e
}

func (e *selfUpdateEnv) advanceOrigin(t *testing.T) string {
	t.Helper()
	write(t, filepath.Join(e.origin, "global", "AGENTS.md"), "v2\n")
	gitT(t, e.origin, "commit", "-q", "-am", "v2")
	return gitT(t, e.origin, "rev-parse", "HEAD")
}

func selfUpdateRun(t *testing.T) (int, string) {
	t.Helper()
	code := 0
	out, stderr := captureStd(t, func() { code = run([]string{"self-update"}) })
	return code, out + stderr
}

func TestSelfUpdateSourceBuildFastForwardsAndRebuilds(t *testing.T) {
	e := newSelfUpdateEnv(t)
	head := e.advanceOrigin(t)
	code, out := selfUpdateRun(t)
	if code != 0 {
		t.Fatalf("self-update exit %d:\n%s", code, out)
	}
	if got := gitT(t, e.checkout, "rev-parse", "HEAD"); got != head {
		t.Fatalf("checkout at %s, want origin %s", got, head)
	}
	if got := readString(t, e.exe); got != "built from "+head {
		t.Fatalf("binary %q, want a build of %s", got, head)
	}
	info, err := os.Stat(e.exe)
	if err != nil || info.Mode().Perm() != 0o755 {
		t.Fatalf("binary mode %v, err %v", info.Mode(), err)
	}
	if !strings.Contains(out, short(head)) || !strings.Contains(out, "gap sync") {
		t.Fatalf("output does not name the new commit and gap sync:\n%s", out)
	}
	matches, _ := filepath.Glob(filepath.Join(filepath.Dir(e.exe), ".gap.self-update-*"))
	if len(matches) != 0 {
		t.Fatalf("temp binaries left behind: %v", matches)
	}
}

func TestSelfUpdateBuildsOnlyTheCommit(t *testing.T) {
	e := newSelfUpdateEnv(t)
	head := e.advanceOrigin(t)
	write(t, filepath.Join(e.checkout, "cmd", "gap", "wip.go"), "package main\n")
	var roots []string
	buildGap = func(root, source, out string) error {
		roots = append(roots, root)
		if fileExists(filepath.Join(root, "cmd", "gap", "wip.go")) {
			t.Errorf("build root %s holds the untracked wip.go", root)
		}
		return os.WriteFile(out, []byte("built from "+gitT(t, root, "rev-parse", "HEAD")), 0o644)
	}
	if code, out := selfUpdateRun(t); code != 0 {
		t.Fatalf("self-update exit %d:\n%s", code, out)
	}
	if len(roots) != 1 || roots[0] == e.checkout {
		t.Fatalf("built in %v, want one clean copy outside %s", roots, e.checkout)
	}
	if got := readString(t, e.exe); got != "built from "+head {
		t.Fatalf("binary %q, want a build of %s", got, head)
	}
	if fileExists(roots[0]) || strings.Contains(gitT(t, e.checkout, "worktree", "list"), roots[0]) {
		t.Fatalf("build copy %s left behind", roots[0])
	}
}

func TestBuildGapBakesCheckoutWithSpaces(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the real binary")
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	src := filepath.Dir(filepath.Dir(wd))
	for _, root := range []string{filepath.Join(t.TempDir(), "gap checkout"), filepath.Join(t.TempDir(), "ed's gap")} {
		bin := filepath.Join(t.TempDir(), "gap")
		if err := buildGap(src, root, bin); err != nil {
			t.Fatalf("build for %q: %v", root, err)
		}
		out, err := exec.Command(bin, "source").Output()
		if err != nil || strings.TrimSpace(string(out)) != root {
			t.Fatalf("gap source = %q (%v), want %q", out, err, root)
		}
	}
}

func TestSelfUpdateBuildFailureKeepsCheckout(t *testing.T) {
	e := newSelfUpdateEnv(t)
	before := gitT(t, e.checkout, "rev-parse", "HEAD")
	e.advanceOrigin(t)
	buildGap = func(src, root, out string) error { return fmt.Errorf("compile error") }
	code, out := selfUpdateRun(t)
	if code == 0 || !strings.Contains(out, "checkout and binary unchanged") {
		t.Fatalf("self-update exit %d on a failed build:\n%s", code, out)
	}
	if got := gitT(t, e.checkout, "rev-parse", "HEAD"); got != before {
		t.Fatalf("checkout advanced to %s although the build failed", got)
	}
	if got := gitT(t, e.checkout, "rev-parse", "origin/master"); got != before {
		t.Fatalf("origin/master advanced to %s although the build failed; gap sync would refuse the checkout", got)
	}
	if got := readString(t, e.exe); got != "old binary" {
		t.Fatalf("binary replaced with %q", got)
	}
}

func TestSelfUpdateCheckoutMovedDuringBuildKeepsBinary(t *testing.T) {
	e := newSelfUpdateEnv(t)
	target := e.advanceOrigin(t)
	write(t, filepath.Join(e.origin, "global", "AGENTS.md"), "v3\n")
	gitT(t, e.origin, "commit", "-q", "-am", "v3")
	newer := gitT(t, e.origin, "rev-parse", "HEAD")
	build := buildGap
	gitT(t, e.origin, "update-ref", "refs/heads/master", target)
	pulled := false
	buildGap = func(src, root, out string) error {
		if !pulled {
			gitT(t, e.origin, "update-ref", "refs/heads/master", newer)
			gitT(t, e.checkout, "pull", "-q", "--ff-only")
			pulled = true
		}
		return build(src, root, out)
	}
	code, out := selfUpdateRun(t)
	if code == 0 || !strings.Contains(out, "moved during the update") {
		t.Fatalf("self-update exit %d after the checkout moved mid-build:\n%s", code, out)
	}
	if got := gitT(t, e.checkout, "rev-parse", "HEAD"); got != newer {
		t.Fatalf("checkout at %s, want the pulled %s left alone", got, newer)
	}
	if got := readString(t, e.exe); got != "old binary" {
		t.Fatalf("binary replaced with %q from a stale commit", got)
	}
}

func TestSelfUpdateKeepsNewerTrackingRef(t *testing.T) {
	e := newSelfUpdateEnv(t)
	target := e.advanceOrigin(t)
	write(t, filepath.Join(e.origin, "global", "AGENTS.md"), "v3\n")
	gitT(t, e.origin, "commit", "-q", "-am", "v3")
	newer := gitT(t, e.origin, "rev-parse", "HEAD")
	gitT(t, e.origin, "update-ref", "refs/heads/master", target)
	build := buildGap
	fetched := false
	buildGap = func(src, root, out string) error {
		if !fetched {
			gitT(t, e.origin, "update-ref", "refs/heads/master", newer)
			gitT(t, e.checkout, "fetch", "-q", "origin")
			fetched = true
		}
		return build(src, root, out)
	}
	if code, out := selfUpdateRun(t); code != 0 {
		t.Fatalf("self-update exit %d:\n%s", code, out)
	}
	if got := gitT(t, e.checkout, "rev-parse", "HEAD"); got != target {
		t.Fatalf("checkout at %s, want %s", got, target)
	}
	if got := gitT(t, e.checkout, "rev-parse", "origin/master"); got != newer {
		t.Fatalf("origin/master rewound to %s, want the newer fetch %s kept", got, newer)
	}
}

func TestSelfUpdateRefusesBeforeChangingAnything(t *testing.T) {
	cases := map[string]func(t *testing.T, e *selfUpdateEnv){
		"dirty": func(t *testing.T, e *selfUpdateEnv) {
			write(t, filepath.Join(e.checkout, "global", "AGENTS.md"), "local edit\n")
		},
		"branch": func(t *testing.T, e *selfUpdateEnv) {
			gitT(t, e.checkout, "switch", "-q", "-c", "feature")
		},
		"diverged": func(t *testing.T, e *selfUpdateEnv) {
			write(t, filepath.Join(e.checkout, "local.md"), "local\n")
			gitT(t, e.checkout, "add", "local.md")
			gitT(t, e.checkout, "commit", "-q", "-m", "local")
		},
		"quotes": func(t *testing.T, e *selfUpdateEnv) {
			moved := filepath.Join(t.TempDir(), `ed's "gap"`)
			if err := os.Rename(e.checkout, moved); err != nil {
				t.Fatal(err)
			}
			e.checkout = moved
			sourceRootFor = moved
		},
		"ignored": func(t *testing.T, e *selfUpdateEnv) {
			write(t, filepath.Join(e.origin, "notes.md"), "upstream\n")
			gitT(t, e.origin, "add", "notes.md")
			gitT(t, e.origin, "commit", "-q", "-m", "track notes")
			write(t, filepath.Join(e.checkout, ".git", "info", "exclude"), "notes.md\n")
			write(t, filepath.Join(e.checkout, "notes.md"), "mine\n")
			t.Cleanup(func() {
				if got := readString(t, filepath.Join(e.checkout, "notes.md")); got != "mine\n" {
					t.Errorf("ignored notes.md overwritten with %q", got)
				}
			})
		},
	}
	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			e := newSelfUpdateEnv(t)
			e.advanceOrigin(t)
			setup(t, e)
			before := gitT(t, e.checkout, "rev-parse", "HEAD")
			code, out := selfUpdateRun(t)
			if code == 0 {
				t.Fatalf("self-update succeeded on a %s checkout:\n%s", name, out)
			}
			if got := gitT(t, e.checkout, "rev-parse", "HEAD"); got != before {
				t.Fatalf("checkout moved to %s", got)
			}
			if got := readString(t, e.exe); got != "old binary" {
				t.Fatalf("binary changed to %q after a refusal", got)
			}
		})
	}
}

func releaseServer(t *testing.T, tag string, binary []byte, corrupt bool) {
	t.Helper()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(zw)
	for _, f := range []struct {
		name string
		body []byte
	}{{"README.md", []byte("readme")}, {"gap", binary}} {
		if err := tw.WriteHeader(&tar.Header{Name: f.name, Mode: 0o755, Size: int64(len(f.body)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		tw.Write(f.body)
	}
	tw.Close()
	zw.Close()
	archive := fmt.Sprintf("gap_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)
	sum := sha256.Sum256(buf.Bytes())
	want := hex.EncodeToString(sum[:])
	if corrupt {
		want = strings.Repeat("0", 64)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/releases/latest":
			http.Redirect(w, r, "/releases/tag/"+tag, http.StatusFound)
		case "/releases/download/" + tag + "/checksums.txt":
			fmt.Fprintf(w, "%s  gap_other.tar.gz\n%s  %s\n", strings.Repeat("1", 64), want, archive)
		case "/releases/download/" + tag + "/" + archive:
			w.Write(buf.Bytes())
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	old := releaseBase
	releaseBase = srv.URL + "/releases"
	t.Cleanup(func() { releaseBase = old })
}

func TestSelfUpdateReleaseBuildDownloadsVerifiedBinary(t *testing.T) {
	e := newSelfUpdateEnv(t)
	version = "1.0.0"
	head := e.advanceOrigin(t)
	releaseServer(t, "v1.1.0", []byte("release 1.1.0"), false)
	code, out := selfUpdateRun(t)
	if code != 0 {
		t.Fatalf("self-update exit %d:\n%s", code, out)
	}
	if got := readString(t, e.exe); got != "release 1.1.0" || len(e.built) != 0 {
		t.Fatalf("binary %q (builds %v), want the release binary", got, e.built)
	}
	if got := gitT(t, e.checkout, "rev-parse", "HEAD"); got != head {
		t.Fatalf("checkout at %s, want origin %s", got, head)
	}
	if !strings.Contains(out, "1.0.0 -> v1.1.0") {
		t.Fatalf("output does not name the version change:\n%s", out)
	}
}

func TestSelfUpdateReleaseChecksumMismatchKeepsBinary(t *testing.T) {
	e := newSelfUpdateEnv(t)
	version = "1.0.0"
	before := gitT(t, e.checkout, "rev-parse", "HEAD")
	e.advanceOrigin(t)
	releaseServer(t, "v1.1.0", []byte("tampered"), true)
	code, out := selfUpdateRun(t)
	if code == 0 || !strings.Contains(out, "checksum mismatch") {
		t.Fatalf("self-update exit %d on a bad checksum:\n%s", code, out)
	}
	if got := gitT(t, e.checkout, "rev-parse", "HEAD"); got != before {
		t.Fatalf("checkout advanced to %s although the download failed", got)
	}
	if got := readString(t, e.exe); got != "old binary" {
		t.Fatalf("binary replaced with %q despite the mismatch", got)
	}
}

func TestSelfUpdateReleaseNeverDowngrades(t *testing.T) {
	e := newSelfUpdateEnv(t)
	version = "0.10.0"
	releaseServer(t, "v0.9.4", []byte("older"), false)
	code, out := selfUpdateRun(t)
	if code != 0 || !strings.Contains(out, "newer than the latest release v0.9.4") {
		t.Fatalf("self-update exit %d:\n%s", code, out)
	}
	if got := readString(t, e.exe); got != "old binary" {
		t.Fatalf("binary downgraded to %q", got)
	}
}

func TestSelfUpdateReleasePromotesCandidate(t *testing.T) {
	e := newSelfUpdateEnv(t)
	version = "4.1.0-rc.1"
	releaseServer(t, "v4.1.0", []byte("final"), false)
	if code, out := selfUpdateRun(t); code != 0 || !strings.Contains(out, "4.1.0-rc.1 -> v4.1.0") {
		t.Fatalf("self-update exit %d:\n%s", code, out)
	}
	if got := readString(t, e.exe); got != "final" {
		t.Fatalf("binary %q, want the final release", got)
	}
}

func TestSelfUpdateReleaseAlreadyLatest(t *testing.T) {
	e := newSelfUpdateEnv(t)
	version = "1.1.0"
	releaseServer(t, "v1.1.0", []byte("same"), false)
	code, out := selfUpdateRun(t)
	if code != 0 || !strings.Contains(out, "already at v1.1.0") {
		t.Fatalf("self-update exit %d:\n%s", code, out)
	}
	if got := readString(t, e.exe); got != "old binary" {
		t.Fatalf("binary replaced with %q though it was current", got)
	}
}
