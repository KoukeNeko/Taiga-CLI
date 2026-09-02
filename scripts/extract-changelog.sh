#!/usr/bin/env sh
# Print the CHANGELOG section for one version, for use as GitHub Release notes.
#
# A release without notes is a release nobody can evaluate, so a missing
# section is an error rather than an empty body.
set -eu

usage() {
    printf '%s\n' "usage: $0 <version-tag> [changelog-file]" >&2
    exit 2
}

[ "$#" -ge 1 ] && [ "$#" -le 2 ] || usage
tag=$1
changelog=${2:-CHANGELOG.md}

case "$tag" in
    v*) version=${tag#v} ;;
    *)
        printf '%s\n' "version tag $tag must start with v" >&2
        exit 2
        ;;
esac

[ -f "$changelog" ] || {
    printf '%s\n' "no such changelog: $changelog" >&2
    exit 2
}

section=$(awk -v heading="## [$version]" '
    index($0, heading) == 1 { collecting = 1; next }
    collecting && /^## \[/ { exit }
    collecting { print }
' "$changelog")

# Trim leading and trailing blank lines so the release body starts at content.
section=$(printf '%s\n' "$section" | sed -e '/./,$!d' | sed -e ':a' -e '/^\n*$/{$d;N;ba' -e '}')

[ -n "$section" ] || {
    printf '%s\n' "no '## [$version]' section in $changelog" >&2
    exit 1
}

printf '%s\n' "$section"
