#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd -- "${script_dir}/../.." && pwd)"

ACT_FLAGS="${ACT_FLAGS:---pull=false -P ubuntu-24.04=catthehacker/ubuntu:act-latest}"
ACT_DRYRUN_SECRETS="${ACT_DRYRUN_SECRETS:--s DISCORD_WEBHOOK_URL=https://example.invalid/webhook}"
XDG_CONFIG_HOME="${ACT_CONFIG_HOME:-${repo_root}/.github}"
export XDG_CONFIG_HOME

mkdir -p "${XDG_CONFIG_HOME}/act"
touch "${XDG_CONFIG_HOME}/act/actrc"

run_act() {
	local event="$1"
	local event_file="$2"
	local workflow="$3"
	shift 3

	act ${ACT_FLAGS:-} --dryrun "$event" "$@" \
		-e "$event_file" \
		-W "$workflow"
}

# CI combines the old lint and PR test workflows. Dry-run both event shapes
# because pull requests cancel superseded runs while pushes to master do not.
run_act push .github/act/push_master.json .github/workflows/ci.yml
run_act pull_request .github/act/pull_request.json .github/workflows/ci.yml
run_act pull_request .github/act/pull_request.json \
	.github/workflows/discord-notify.yml ${ACT_DRYRUN_SECRETS:-}
run_act push .github/act/push_master.json .github/workflows/prerelease.yml
run_act workflow_dispatch .github/act/stable_release.json \
	.github/workflows/stable-release.yml
run_act push .github/act/push_master.json \
	.github/workflows/docker-package.yml ${ACT_DRYRUN_SECRETS:-}
run_act pull_request .github/act/pull_request.json \
	.github/workflows/docker-package.yml ${ACT_DRYRUN_SECRETS:-}
