#!/bin/bash
set -euo pipefail
BIN="/home/greddyisbetter/t3-ops/webux-src/webux/build/webux"
test -x "$BIN"
systemctl stop webux
cp "$BIN" /usr/local/bin/webux
chmod 755 /usr/local/bin/webux
rm -f /tmp/webux-hermes-serve.log
systemctl start webux
systemctl is-active webux
echo "Installed $(stat -c %s /usr/local/bin/webux) bytes"
