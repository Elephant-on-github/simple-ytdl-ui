//go:build windows

package ytdlp

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// prepareCmdPlatform builds the command used to run the yt-dlp binary.
//
// On Windows a .bat/.cmd file is not executed directly: cmd.exe parses the
// command line with its own rules, so arguments holding characters such as
// "|", "&" or "%" (the frozen progress template has all of them) break the
// invocation. For those binaries the command line is quoted here.
//
// The real yt-dlp is an .exe on Windows and a plain binary everywhere else, so
// it never goes through cmd.exe; this path exists for wrappers and for the
// fake the test suite uses. cmd.exe expands %NAME% pairs on its command line
// and there is no reliable escape for it, so a percent sign in an argument
// that matters (the output template) can still be mangled here.
func prepareCmdPlatform(ctx context.Context, bin string, args ...string) *exec.Cmd {
	if !isBatchFile(bin) {
		return exec.CommandContext(ctx, bin, args...)
	}

	quoted := make([]string, 0, len(args)+1)
	quoted = append(quoted, `"`+bin+`"`)
	for _, a := range args {
		quoted = append(quoted, `"`+strings.ReplaceAll(a, `"`, `""`)+`"`)
	}

	comspec := os.Getenv("COMSPEC")
	if comspec == "" {
		comspec = "cmd.exe"
	}

	cmd := exec.CommandContext(ctx, comspec)
	cmd.Args = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{
		// /s strips the outer quotes, /d disables AutoRun commands.
		CmdLine: `/d /s /c "` + strings.Join(quoted, " ") + `"`,
	}
	return cmd
}

func isBatchFile(bin string) bool {
	switch strings.ToLower(filepath.Ext(bin)) {
	case ".bat", ".cmd":
		return true
	}
	return false
}
