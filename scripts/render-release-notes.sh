#!/usr/bin/env sh
# Compose bilingual GitHub Release notes for one version.
#
# English leads because the repository front page does, and the Traditional
# Chinese section follows inside a collapsed block so the page stays readable
# while carrying both.
set -eu

ENGLISH_CHANGELOG="CHANGELOG.md"
CHINESE_CHANGELOG="CHANGELOG.zh-TW.md"

usage() {
    printf '%s\n' "usage: $0 <version-tag>" >&2
    exit 2
}

[ "$#" -eq 1 ] || usage
tag=$1
script_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
extract="$script_dir/extract-changelog.sh"

english=$("$extract" "$tag" "$ENGLISH_CHANGELOG")
chinese=$("$extract" "$tag" "$CHINESE_CHANGELOG")

printf '%s\n' "$english"
printf '\n---\n\n'
printf '<details>\n<summary><b>繁體中文</b></summary>\n\n'
printf '%s\n' "$chinese"
printf '\n</details>\n'
