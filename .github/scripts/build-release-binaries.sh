#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd -- "${script_dir}/../.." && pwd)"
dist_dir="${RELEASE_DIST_DIR:-dist}"
binary_version="${BINARY_VERSION:-}"
target_asset="${RELEASE_TARGET_ASSET:-}"
target_found=false

cd "$repo_root"
mkdir -p "$dist_dir"

while IFS='|' read -r _label goos goarch goarm asset; do
	if [ -n "$target_asset" ] && [ "$asset" != "$target_asset" ]; then
		continue
	fi
	target_found=true

	target="${goos}/${goarch}"
	if [ -n "$goarm" ]; then
		target="${target} GOARM=${goarm}"
	fi

	build_args=()
	if [ -n "$binary_version" ]; then
		build_args+=(-ldflags "-X main.version=${binary_version}")
	fi

	echo "::group::Build ${asset} (${target})"
	if [ -n "$goarm" ]; then
		env GOOS="$goos" GOARCH="$goarch" GOARM="$goarm" \
			go build "${build_args[@]}" -o "${dist_dir}/${asset}" .
	else
		env GOOS="$goos" GOARCH="$goarch" \
			go build "${build_args[@]}" -o "${dist_dir}/${asset}" .
	fi
	echo "::endgroup::"
done < <("${script_dir}/release-assets.sh" targets)

if [ -n "$target_asset" ] && [ "$target_found" != true ]; then
	printf 'Unknown release target asset: %s\n' "$target_asset" >&2
	exit 2
fi
