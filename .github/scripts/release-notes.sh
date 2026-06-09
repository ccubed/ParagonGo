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
checksums_file="${CHECKSUMS_FILE:-SHA256SUMS.txt}"
downloads="$(
	DATAFILES_ARCHIVE="$datafiles_archive" \
		CHECKSUMS_FILE="$checksums_file" \
		.github/scripts/release-assets.sh downloads-markdown
)"

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
if [ -n "$previous_tag" ] && [ "$previous_tag" != "$release_tag" ]; then
	changes_since="Changes since: \`${previous_tag}\`"
else
	changes_since="Changes since: initial release history"
fi

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
- ${changes_since}

${summary}

## Downloads

${downloads}

## Install From Source

Clone the repository, check out this release tag or commit, install the Go
toolchain from \`go.mod\`, and run \`make build\`.

## Manual Binary Install

Download the binary for your operating system and CPU architecture, download
\`${datafiles_archive}\`, unpack the datafiles next to the binary, and make the
binary executable on Unix-like systems.

## Verify Provenance

Download \`${checksums_file}\` and run \`sha256sum -c ${checksums_file}\` in the
directory containing the release assets. Verify build provenance with
\`gh attestation verify <asset> --repo ${repository}\`.

## Changes

EOF

cat "$generated_notes_file" >>"$notes_file"
