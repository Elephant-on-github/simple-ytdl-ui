#!/bin/sh
# The container starts as root so it can fix the ownership of the mounted
# volumes, which are usually owned by someone else on the host. After that
# the server runs as the unprivileged "ytdl" user.
set -e

# Fix ownership of the data and library directories.
for dir in /data /library; do
  if [ -d "$dir" ]; then
    chown -R ytdl:ytdl "$dir"
  fi
done

exec su-exec ytdl /usr/local/bin/server
