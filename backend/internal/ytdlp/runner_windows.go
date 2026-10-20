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
