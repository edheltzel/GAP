package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	releaseBase = "https://github.com/edheltzel/GAP/releases"
	selfPath    = os.Executable
	buildGap    = func(root, out string) error {
		cmd := exec.Command("go", "build", "-o", out, "./cmd/gap")
		cmd.Dir = root
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
)

const maxReleaseBytes = 256 << 20

func selfUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "self-update",
		Short: "Update the gap binary and its source checkout to the latest master or release",
		Long: `self-update brings gap itself up to date. It never moves component pins; that is gap update.

A source build fast-forwards its build checkout to origin's primary branch, then rebuilds
the running binary with go build. A release build fast-forwards ~/.gap/checkout, then
downloads the latest release for this OS and architecture, checks it against
checksums.txt, and replaces the running binary.

A checkout that is not on its primary branch, or has uncommitted changes to tracked
files, is refused before anything changes. Run gap sync afterwards to publish.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return selfUpdate(newReport(os.Stdout))
		},
	}
}

func selfUpdate(out *report) error {
	exe, err := selfPath()
	if err != nil {
		return err
	}
	if exe, err = filepath.EvalSymlinks(exe); err != nil {
		return err
	}
	root, err := sourceRoot()
	if err != nil {
		return err
	}
	before, after, err := fastForward(root)
	if err != nil {
		return err
	}
	if before == after {
		out.done(true, fmt.Sprintf("checkout %s already at %s", tildePath(root), short(after)))
	} else {
		out.done(true, fmt.Sprintf("checkout %s %s -> %s", tildePath(root), short(before), short(after)))
	}
	if version == "" {
		if err := replaceExe(exe, func(tmp string) error { return buildGap(root, tmp) }); err != nil {
			return fmt.Errorf("go build in %s: %w", root, err)
		}
		out.done(true, fmt.Sprintf("rebuilt %s from %s", tildePath(exe), short(after)))
	} else {
		tag, err := latestRelease()
		if err != nil {
			return err
		}
		switch cmp := compareVersions(version, tag); {
		case cmp == 0:
			out.done(true, fmt.Sprintf("%s already at %s", tildePath(exe), tag))
		case cmp > 0:
			out.done(true, fmt.Sprintf("%s %s is newer than the latest release %s; kept", tildePath(exe), version, tag))
		default:
			if err := replaceExe(exe, func(tmp string) error { return downloadRelease(tag, tmp) }); err != nil {
				return err
			}
			out.done(true, fmt.Sprintf("%s %s -> %s", tildePath(exe), version, tag))
		}
	}
	out.next("gap sync    publish the updated policy and install the pinned components")
	return nil
}

func fastForward(root string) (string, string, error) {
	originHead, err := git(root, "symbolic-ref", "--short", "refs/remotes/origin/HEAD")
	if err != nil {
		return "", "", fmt.Errorf("%s: origin/HEAD is not set; run git -C %s remote set-head origin --auto", root, root)
	}
	primary := strings.TrimPrefix(originHead, "origin/")
	branch, _ := git(root, "symbolic-ref", "--quiet", "--short", "HEAD")
	if branch != primary {
		return "", "", fmt.Errorf("%s is on %q, not %s; switch it to %s, then run gap self-update again", root, branch, primary, primary)
	}
	dirty, err := git(root, "status", "--porcelain", "--untracked-files=no")
	if err != nil {
		return "", "", err
	}
	if dirty != "" {
		return "", "", fmt.Errorf("%s has uncommitted changes; commit or move them, then run gap self-update again", root)
	}
	before, err := git(root, "rev-parse", "HEAD")
	if err != nil {
		return "", "", err
	}
	if _, err := git(root, "fetch", "--quiet", "origin", primary); err != nil {
		return "", "", err
	}
	if _, err := git(root, "merge", "--ff-only", "--quiet", "FETCH_HEAD"); err != nil {
		return "", "", fmt.Errorf("%s cannot fast-forward to origin/%s: %w", root, primary, err)
	}
	after, err := git(root, "rev-parse", "HEAD")
	return before, after, err
}

func replaceExe(exe string, write func(tmp string) error) error {
	f, err := os.CreateTemp(filepath.Dir(exe), ".gap.self-update-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	f.Close()
	defer os.Remove(tmp)
	if err := write(tmp); err != nil {
		return err
	}
	if err := os.Chmod(tmp, 0o755); err != nil {
		return err
	}
	return os.Rename(tmp, exe)
}

func latestRelease() (string, error) {
	client := &http.Client{
		Timeout:       30 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	resp, err := client.Get(releaseBase + "/latest")
	if err != nil {
		return "", fmt.Errorf("latest gap release: %w", err)
	}
	resp.Body.Close()
	loc := resp.Header.Get("Location")
	if !strings.Contains(loc, "/releases/tag/") {
		return "", fmt.Errorf("latest gap release: %s did not redirect to a tag", releaseBase+"/latest")
	}
	return path.Base(loc), nil
}

func compareVersions(a, b string) int {
	pa, pb := versionParts(a), versionParts(b)
	for i := range pa {
		if pa[i] < pb[i] {
			return -1
		}
		if pa[i] > pb[i] {
			return 1
		}
	}
	return 0
}

func versionParts(v string) [3]int {
	var p [3]int
	core, _, _ := strings.Cut(strings.TrimPrefix(v, "v"), "-")
	for i, s := range strings.SplitN(core, ".", 3) {
		p[i], _ = strconv.Atoi(s)
	}
	return p
}

func releaseArchive() (string, error) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" || runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64" {
		return "", fmt.Errorf("no gap release for %s/%s; install Go and run the installer: %s", runtime.GOOS, runtime.GOARCH, installHint)
	}
	return fmt.Sprintf("gap_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH), nil
}

func fetch(url string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", url, resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxReleaseBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxReleaseBytes {
		return nil, fmt.Errorf("%s: larger than %d bytes", url, maxReleaseBytes)
	}
	return body, nil
}

func downloadRelease(tag, dst string) error {
	archive, err := releaseArchive()
	if err != nil {
		return err
	}
	base := releaseBase + "/download/" + tag + "/"
	sums, err := fetch(base + "checksums.txt")
	if err != nil {
		return err
	}
	want := ""
	for line := range strings.Lines(string(sums)) {
		fields := strings.Fields(line)
		if len(fields) == 2 && strings.TrimPrefix(fields[1], "*") == archive {
			want = fields[0]
		}
	}
	if want == "" {
		return fmt.Errorf("checksums.txt for %s has no entry for %s; nothing installed", tag, archive)
	}
	body, err := fetch(base + archive)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(body)
	if got := hex.EncodeToString(sum[:]); got != want {
		return fmt.Errorf("checksum mismatch for %s (want %s, got %s); nothing installed", archive, want, got)
	}
	zr, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		return err
	}
	tr := tar.NewReader(zr)
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return fmt.Errorf("%s has no gap binary; nothing installed", archive)
		}
		if err != nil {
			return err
		}
		if h.Typeflag != tar.TypeReg || path.Clean(h.Name) != "gap" {
			continue
		}
		f, err := os.OpenFile(dst, os.O_WRONLY|os.O_TRUNC, 0o755)
		if err != nil {
			return err
		}
		if _, err := io.Copy(f, io.LimitReader(tr, maxReleaseBytes)); err != nil {
			f.Close()
			return err
		}
		return f.Close()
	}
}
