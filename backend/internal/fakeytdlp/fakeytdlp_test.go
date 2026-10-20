package fakeytdlp

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// The suite runs on Windows and on Linux. The Windows fake is a batch file,
// the Linux one a POSIX shell script — and a shell script that only ever runs
// on Linux can hide syntax errors for months. These tests run the POSIX
// script wherever a POSIX shell is available.
func TestShellScriptIsValidPOSIX(t *testing.T) {
	sh := lookShell(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "fake-ytdlp.sh")
	if err := os.WriteFile(path, []byte(shellScript(ModeOK)), 0o755); err != nil {
		t.Fatalf("write the script: %v", err)
	}

	out, err := exec.Command(sh, "-n", path).CombinedOutput()
	if err != nil {
		t.Fatalf("the shell fake is not valid POSIX: %v\n%s", err, out)
	}
}

func TestShellScriptProbesAPlaylist(t *testing.T) {
	sh := lookShell(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "fake-ytdlp.sh")
	if err := os.WriteFile(path, []byte(shellScript(ModeOK)), 0o755); err != nil {
		t.Fatalf("write the script: %v", err)
	}

	out, err := exec.Command(sh, path, "-J", "--flat-playlist", "https://www.youtube.com/playlist?list=PLfake").CombinedOutput()
	if err != nil {
		t.Fatalf("the shell fake failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "Fake Playlist") {
		t.Errorf("output = %q, want the fake playlist", string(out))
	}
}

func TestShellScriptCountsDownloads(t *testing.T) {
	sh := lookShell(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "fake-ytdlp.sh")
	if err := os.WriteFile(path, []byte(shellScript(ModeOK)), 0o755); err != nil {
		t.Fatalf("write the script: %v", err)
	}

	count := filepath.Join(t.TempDir(), "downloads")
	if err := os.MkdirAll(count, 0o755); err != nil {
		t.Fatalf("create the counter directory: %v", err)
	}
	cmd := exec.Command(sh, path, "-x", "-o", filepath.Join(dir, "out.%(ext)s"), "https://www.youtube.com/watch?v=dQw4w9WgXcQ")
	cmd.Env = append(os.Environ(), "FAKE_YTDLP_COUNT="+count)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("the shell fake failed: %v\n%s", err, out)
	}

	marks, err := os.ReadDir(count)
	if err != nil {
		t.Fatalf("the shell fake did not count the download: %v", err)
	}
	if len(marks) != 1 {
		t.Errorf("the counter holds %d downloads, want 1", len(marks))
	}

	mp3 := filepath.Join(dir, "Fake Artist - Fake Video [dQw4w9WgXcQ].mp3")
	content, err := os.ReadFile(mp3)
	if err != nil {
		t.Fatalf("the shell fake wrote no mp3: %v", err)
	}
	if string(content) != "fake mp3" {
		t.Errorf("the mp3 holds %q", string(content))
	}
}

// lookShell finds a POSIX shell. Git for Windows ships one, so the test also
// runs on a Windows machine. The "sh" on a Windows PATH is often the WSL
// shim, which fails without an installed distribution, so Git Bash is
// preferred there.
func lookShell(t *testing.T) string {
	t.Helper()

	if runtime.GOOS == "windows" {
		for _, candidate := range []string{
			`C:\Program Files\Git\bin\bash.exe`,
			`C:\Program Files\Git\usr\bin\bash.exe`,
		} {
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
		}
		t.Skip("no POSIX shell on this machine")
	}

	for _, name := range []string{"sh", "bash"} {
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
	}
	t.Skip("no POSIX shell on this machine")
	return ""
}
