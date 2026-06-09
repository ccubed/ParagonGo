#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd -- "${script_dir}/../.." && pwd)"
bin_dir="${RELEASE_BIN_DIR:-bin}"
datafiles_archive="${DATAFILES_ARCHIVE:-gomud-ALL-datafiles.zip}"
checksums_file="${CHECKSUMS_FILE:-SHA256SUMS.txt}"

cd "$repo_root"

mapfile -t checksum_assets < <(
	DATAFILES_ARCHIVE="$datafiles_archive" \
		CHECKSUMS_FILE="$checksums_file" \
		"${script_dir}/release-assets.sh" checksum-names
)

zip -qr "${bin_dir}/${datafiles_archive}" _datafiles

cd "$bin_dir"
sha256sum "${checksum_assets[@]}" >"$checksums_file"
