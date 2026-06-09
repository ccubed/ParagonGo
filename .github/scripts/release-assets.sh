#!/usr/bin/env bash
set -euo pipefail

datafiles_archive="${DATAFILES_ARCHIVE:-gomud-ALL-datafiles.zip}"
checksums_file="${CHECKSUMS_FILE:-SHA256SUMS.txt}"

release_targets() {
	cat <<'EOF'
Linux amd64|linux|amd64||gomud-linux_x64
Linux arm64|linux|arm64||gomud-linux_arm64
Linux arm/v7|linux|arm|7|gomud-linux_armv7
Windows amd64|windows|amd64||gomud-windows_x64.exe
Windows arm64|windows|arm64||gomud-windows_arm64.exe
macOS amd64|darwin|amd64||gomud-darwin_x64
macOS arm64|darwin|arm64||gomud-darwin_arm64
EOF
}

binary_names() {
	local label goos goarch goarm asset

	while IFS='|' read -r label goos goarch goarm asset; do
		printf '%s\n' "$asset"
	done < <(release_targets)
}

binary_paths() {
	local asset

	while IFS= read -r asset; do
		printf 'bin/%s\n' "$asset"
	done < <(binary_names)
}

checksum_names() {
	binary_names
	printf '%s\n' "$datafiles_archive"
}

upload_paths() {
	binary_paths
	printf 'bin/%s\n' "$datafiles_archive"
	printf 'bin/%s\n' "$checksums_file"
}

attestation_paths() {
	binary_paths
	printf 'bin/%s\n' "$checksums_file"
}

downloads_markdown() {
	local label goos goarch goarm asset

	while IFS='|' read -r label goos goarch goarm asset; do
		printf -- '- %s: `%s`\n' "$label" "$asset"
	done < <(release_targets)
	printf -- '- Datafiles: `%s`\n' "$datafiles_archive"
	printf -- '- Checksums: `%s`\n' "$checksums_file"
}

case "${1:-}" in
targets)
	release_targets
	;;
binary-names)
	binary_names
	;;
binary-paths)
	binary_paths
	;;
checksum-names)
	checksum_names
	;;
upload-paths)
	upload_paths
	;;
attestation-paths)
	attestation_paths
	;;
downloads-markdown)
	downloads_markdown
	;;
*)
	printf 'Usage: %s COMMAND\n' "$0" >&2
	printf 'Commands: targets, binary-names, binary-paths, ' >&2
	printf 'checksum-names, upload-paths, attestation-paths, ' >&2
	printf 'downloads-markdown\n' >&2
	exit 2
	;;
esac
