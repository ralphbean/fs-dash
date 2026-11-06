#!/bin/sh
set -eu

if [ "$#" -ne 1 ] || [ ! -f "$1" ]; then
	printf '%s\n' 'Usage: check-conventional-commit.sh <commit-message-file>' >&2
	exit 2
fi

subject=$(awk 'NF && $0 !~ /^[[:space:]]*#/ { sub(/\r$/, ""); print; exit }' "$1")
pattern='^[a-z][a-z0-9-]*([(][[:alnum:]][[:alnum:]./_-]*[)])?(!)?: [^[:space:]].*'

if printf '%s\n' "$subject" | grep -Eq "$pattern"; then
	exit 0
fi

printf 'Invalid commit subject: %s\n' "$subject" >&2
printf '%s\n' 'Use Conventional Commits, e.g. feat(ui): add table sorting.' >&2
exit 1
