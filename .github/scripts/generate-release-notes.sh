#!/usr/bin/env bash
set -euo pipefail

output_file="${1:-release-notes.md}"

require_env() {
  local name="$1"
  if [ -z "${!name:-}" ]; then
    echo "${name} is required" >&2
    exit 1
  fi
}

require_env BINARY_VERSION
require_env CHECKSUMS_FILE
require_env DATAFILES_ARCHIVE
require_env GITHUB_REPOSITORY
require_env GITHUB_RUN_ATTEMPT
require_env GITHUB_RUN_ID
require_env GITHUB_SHA
require_env RELEASE_NOTES_INTRO
require_env RELEASE_NOTES_SUMMARY

latest_release_tag="$(
  gh api "repos/${GITHUB_REPOSITORY}/releases/latest" \
    --jq '.tag_name' \
    2>/dev/null || true
)"

generate_notes_args=(
  -f "tag_name=release-notes-${GITHUB_RUN_ID}-${GITHUB_RUN_ATTEMPT}"
  -f "target_commitish=${GITHUB_SHA}"
)

if [ -n "$latest_release_tag" ]; then
  generate_notes_args+=(-f "previous_tag_name=${latest_release_tag}")
  changes_since_line="- Changes since: \`${latest_release_tag}\`"
else
  changes_since_line="- Changes since: initial release history"
fi

generated_release_notes="$(
  gh api \
    -X POST \
    "repos/${GITHUB_REPOSITORY}/releases/generate-notes" \
    "${generate_notes_args[@]}" \
    --jq '.body'
)"
published_at="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

cat > "$output_file" <<EOF
${RELEASE_NOTES_INTRO}

- Version: \`${BINARY_VERSION}\`
- Commit: \`${GITHUB_SHA}\`
- Published: \`${published_at}\`
${changes_since_line}

### Downloads

- Windows, most Intel/AMD PCs: \`gomud-windows_x64.exe\`
- Windows on ARM: \`gomud-windows_arm64.exe\`
- macOS Apple Silicon: \`gomud-darwin_arm64\`
- macOS Intel: \`gomud-darwin_x64\`
- Linux x86_64: \`gomud-linux_x64\`
- Linux ARM64: \`gomud-linux_arm64\`
- Raspberry Pi / ARMv7 Linux: \`gomud-linux_armv7\`
- Server data files: \`${DATAFILES_ARCHIVE}\`
- Checksums: \`${CHECKSUMS_FILE}\`

${RELEASE_NOTES_SUMMARY}
For permanent downloadable builds, use the numbered releases.

${generated_release_notes}
EOF
