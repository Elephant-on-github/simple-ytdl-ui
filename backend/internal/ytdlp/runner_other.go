//go:build !windows

package ytdlp

import (
	"context"
	"os/exec"
)

// prepareCmdPlatform builds the command used to run the yt-dlp binary. Only
// Windows needs special handling for batch files.
func prepareCmdPlatform(ctx context.Context, bin string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, bin, args...)
}
