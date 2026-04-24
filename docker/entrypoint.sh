#!/bin/bash
set -e

# Match the container user to the host user's UID/GID.
# This ensures bind-mounted files have correct permissions.
CODINGBOX_UID="${CODINGBOX_UID:-1000}"
CODINGBOX_GID="${CODINGBOX_GID:-1000}"
CODINGBOX_HOME="${CODINGBOX_HOME:-/home/codingbox}"
USERNAME="codingbox"

# Create group if it doesn't exist.
if ! getent group "$CODINGBOX_GID" > /dev/null 2>&1; then
    groupadd -g "$CODINGBOX_GID" "$USERNAME"
fi
GROUP_NAME=$(getent group "$CODINGBOX_GID" | cut -d: -f1)

# Create user if it doesn't exist.
if ! id -u "$CODINGBOX_UID" > /dev/null 2>&1; then
    useradd -u "$CODINGBOX_UID" -g "$CODINGBOX_GID" -d "$CODINGBOX_HOME" -s /bin/bash -m "$USERNAME" 2>/dev/null || true
fi
USER_NAME=$(id -un "$CODINGBOX_UID" 2>/dev/null || echo "$USERNAME")

# Ensure home directory exists and is owned correctly.
mkdir -p "$CODINGBOX_HOME"
chown "$CODINGBOX_UID:$CODINGBOX_GID" "$CODINGBOX_HOME"

# Set HOME for the user.
export HOME="$CODINGBOX_HOME"

# Add user to sudoers for convenience (no password).
echo "$USER_NAME ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/codingbox 2>/dev/null || true

# Copy uv and tools from root install to user if they don't exist yet.
if [ -d /root/.local/bin ] && [ ! -f "$HOME/.local/bin/uv" ]; then
    mkdir -p "$HOME/.local/bin"
    cp -rn /root/.local/bin/* "$HOME/.local/bin/" 2>/dev/null || true
    chown -R "$CODINGBOX_UID:$CODINGBOX_GID" "$HOME/.local" 2>/dev/null || true
fi

# Configure npm global prefix to a user-writable directory so npm install -g
# and npm update work without sudo.
NPM_GLOBAL="$HOME/.npm-global"
mkdir -p "$NPM_GLOBAL"
chown -R "$CODINGBOX_UID:$CODINGBOX_GID" "$NPM_GLOBAL"
cat > "$HOME/.npmrc" <<EOF
prefix=$NPM_GLOBAL
EOF
chown "$CODINGBOX_UID:$CODINGBOX_GID" "$HOME/.npmrc"

# Ensure global tool paths are available (npm-global, uv, go, user local bins).
export PATH="$NPM_GLOBAL/bin:/usr/local/go/bin:$HOME/go/bin:$HOME/.local/bin:/root/.local/bin:$PATH"

# Execute the command as the matched user.
exec gosu "$CODINGBOX_UID:$CODINGBOX_GID" "$@"
