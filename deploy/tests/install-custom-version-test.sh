#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
TEMP_DIR=$(mktemp -d)
trap 'rm -rf "$TEMP_DIR"' EXIT

# Load functions only. Never run the installer or touch system services.
source <(head -n -1 "$ROOT_DIR/deploy/install.sh")
test "$GITHUB_REPO" = "Drunkard-baifeng/sub2api"
INSTALL_DIR="$TEMP_DIR"
test "$(get_current_version)" = "not_installed"

cat > "$INSTALL_DIR/sub2api" <<'EOF'
#!/usr/bin/env bash
test "$1" = "--version"
printf '2026-10-01 INFO Sub2API %s (commit: fixture, built: fixture)\n' "$TEST_VERSION" >&"$TEST_OUTPUT_FD"
EOF
chmod +x "$INSTALL_DIR/sub2api"

for TEST_OUTPUT_FD in 1 2; do
    export TEST_OUTPUT_FD
    for TEST_VERSION in 0.2.13 v0.2.13 0.2.13-custom.1 v0.2.13-custom.10; do
        export TEST_VERSION
        actual=$(get_current_version)
        if [ "$actual" != "$TEST_VERSION" ]; then
            echo "version mismatch: expected $TEST_VERSION, got $actual" >&2
            exit 1
        fi
    done
done

printf '#!/usr/bin/env bash\necho "unrecognized output" >&2\nexit 1\n' > "$INSTALL_DIR/sub2api"
test "$(get_current_version)" = "unknown"

echo "install custom version checks passed"
