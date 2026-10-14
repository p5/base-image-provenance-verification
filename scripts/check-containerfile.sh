#!/usr/bin/env bash
set -euo pipefail

recipe=${1:?Containerfile path required}
base=${2:?Digest-pinned base required}

awk -v base="$base" '
  NF == 0 { next }
  /\\|\r|<</ { exit 1 }
  toupper($1) == "FROM" {
    count++
    if (count != 1 || NF != 2 || $2 != base) exit 1
    next
  }
  toupper($1) == "COPY" {
    if (count != 1 || NF < 3 || tolower($0) ~ /--from/) exit 1
    next
  }
  toupper($1) == "LABEL" {
    if (count != 1 || NF < 2) exit 1
    next
  }
  { exit 1 }
  END { if (count != 1) exit 1 }
' "$recipe" || {
  echo 'Require one literal FROM, local COPY instructions, and LABEL instructions.' >&2
  exit 1
}
