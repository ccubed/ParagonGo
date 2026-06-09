#!/usr/bin/env bash
set -euo pipefail

notes_file="${RELEASE_NOTES_FILE:-release-notes.md}"
generated_notes_file="${GENERATED_NOTES_FILE:-generated-release-notes.md}"
release_kind="${RELEASE_KIND:-}"
release_tag="${RELEASE_TAG:-}"
binary_version="${BINARY_VERSION:-}"
repository="${GITHUB_REPOSITORY:-}"
commit_sha="${GITHUB_SHA:-}"
ref_name="${GITHUB_REF_NAME:-}"
datafiles_archive="${DATAFILES_ARCHIVE:-gomud-ALL-datafiles.zip}"

require_env() {
	local name="$1"
	local value="$2"

	if [ -z "$value" ]; then
		printf '%s is required\n' "$name" >&2
		exit 1
	fi
}

require_env RELEASE_KIND "$release_kind"
require_env RELEASE_TAG "$release_tag"
require_env BINARY_VERSION "$binary_version"
require_env GITHUB_REPOSITORY "$repository"
require_env GITHUB_SHA "$commit_sha"

case "$release_kind" in
prerelease | stable)
	;;
*)
	printf 'RELEASE_KIND must be prerelease or stable\n' >&2
	exit 1
	;;
esac

previous_tag="${PREVIOUS_TAG_NAME:-}"
if [ -z "$previous_tag" ] && [ "${RELEASE_NOTES_SKIP_GH:-}" != "true" ]; then
	previous_tag="$(
		gh api "repos/${repository}/releases/latest" \
			--jq '.tag_name' \
			2>/dev/null || true
	)"
fi

if [ "${RELEASE_NOTES_SKIP_GH:-}" = "true" ]; then
	printf 'Generated release notes skipped for local dry run.\n' \
		>"$generated_notes_file"
else
	notes_tag="$release_tag"
	if [ "$release_kind" = "prerelease" ]; then
		# GitHub ignores target_commitish when tag_name already exists.
		notes_tag="${release_tag}-notes-${commit_sha}"
	fi

	generate_notes_args=(
		-f "tag_name=${notes_tag}"
		-f "target_commitish=${commit_sha}"
	)
	if [ -n "$previous_tag" ] && [ "$previous_tag" != "$release_tag" ]; then
		generate_notes_args+=(-f "previous_tag_name=${previous_tag}")
	fi

	gh api \
		-X POST \
		"repos/${repository}/releases/generate-notes" \
		"${generate_notes_args[@]}" \
		--jq '.body' \
		>"$generated_notes_file"
fi

published_at="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

if [ "$release_kind" = "prerelease" ]; then
	overview="Rolling prerelease build from \`${ref_name:-master}\`."
	summary="This mutable prerelease is replaced on each successful merge to \`master\`."
else
	overview="Stable release \`${release_tag}\`."
	summary="This stable release is immutable. Tags and assets are not replaced by workflow policy."
fi

cat >"$notes_file" <<EOF
## Overview

${overview}

- Version: \`${binary_version}\`
- Commit: \`${commit_sha}\`
- Published: \`${published_at}\`


${summary}

## Install GoMud

<details>

<summary>Instructions (expand)</summary>

### Quick Install (Recommended)

- The fastest way to get GoMud running.
- These scripts install Go and Git (if needed), clone the GoMud repo, and build the server binary automatically.

#### Linux / macOS

Open a Terminal and run:

\`\`\`shell
curl -fsSL https://raw.githubusercontent.com/GoMudEngine/GoMud/master/scripts/install.sh | sh
\`\`\`

#### Windows

Open a \`Powershell\` window and run:

\`\`\`powershell
irm https://raw.githubusercontent.com/GoMudEngine/GoMud/master/scripts/install.ps1 | iex
\`\`\`

- Both scripts install GoMud to \`~/GoMud\` by default.
- Set the \`GOMUD_DIR\` environment variable before running to choose a different location.

### Alternative: Manual Install

- Scroll down to the **"Assets"** section below and expand it
- Download the datafiles (needed by all operating systems):
  - **\`${datafiles_archive}\`**
  - Extract this zip file into the same folder as the GoMud binary
- Download the GoMud binary/executable specific for your operating system, based on the table below:

**Most common:**

| Filename | Operating System | CPU Architecture | Typical Devices |
|------------|------------------|------------------|-----------------|
| gomud-darwin_arm64 | macOS | ARM64 (Apple Silicon) | Apple M1, M2, M3, M4 Macs |
| gomud-linux_x64 | Linux | x86_64 (Intel/AMD 64-bit) | Most modern desktop/server Linux systems |
| gomud-windows_x64.exe | Windows | x86_64 (Intel/AMD 64-bit) | Most modern Windows PCs |

**Other options:**

| Filename | Operating System | CPU Architecture | Typical Devices |
|------------|------------------|------------------|-----------------|
| gomud-darwin_x64 | macOS | x86_64 (Intel 64-bit) | Older Intel-based Macs |
| gomud-windows_arm64.exe | Windows | ARM64 | Surface Pro X, Snapdragon X Elite laptops, Windows on ARM
| gomud-linux_arm64 | Linux | ARM64 (AArch64) | Linux systems using ARM-based CPUs |
| gomud-linux_armv7 | Linux | ARMv7 (32-bit) | Raspberry Pi 2/3 running 32-bit OS |
</details>

EOF

cat "$generated_notes_file" >>"$notes_file"
