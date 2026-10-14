#!/usr/bin/env bash
set -euo pipefail
script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
fixture="$script_dir/../testdata/statement.json"
work=$(mktemp -d)
trap 'rm -rf -- "$work"' EXIT

image="ghcr.io/p5/base-image-provenance-verification@sha256:$(jq -r '.subject[0].digest.sha256' "$fixture")"
base=$(jq -r '.predicate.buildDefinition.externalParameters.base' "$fixture")
parent=$(jq -r '.predicate.buildDefinition.resolvedDependencies[] | select(.name == "runtime-base") | .uri | ltrimstr("oci://")' "$fixture")
workflow=$(jq -r '.predicate.runDetails.builder.id' "$fixture")
revision=$(jq -r '.predicate.buildDefinition.externalParameters.revision' "$fixture")
source=$(jq -r '.predicate.buildDefinition.externalParameters.source' "$fixture")

check_policy() {
  jq -e --arg image "$image" --arg base "$base" --arg parent "$parent" \
    --arg workflow "$workflow" --arg revision "$revision" --arg source "$source" \
    -f "$script_dir/provenance-policy.jq" "$1" > /dev/null 2>&1
}

jq '[.]' "$fixture" > "$work/valid.json"
check_policy "$work/valid.json"
count=1
for mutation in \
  '.subject[0].digest.sha256 = ("0" * 64)' \
  '.predicateType = "other"' \
  '.predicate.runDetails.builder.id = "other"' \
  '.predicate.buildDefinition.buildType = "other"' \
  '.predicate.buildDefinition.externalParameters.revision = ("0" * 40)' \
  '.predicate.buildDefinition.externalParameters.source = "https://github.com/attacker/repo"' \
  '.predicate.buildDefinition.externalParameters.base = "registry.example/base:latest"' \
  '.predicate.buildDefinition.externalParameters.platform = "linux/arm64"' \
  '.predicate.buildDefinition.externalParameters.extra = "unexpected input"' \
  '.predicate.buildDefinition.resolvedDependencies[0].digest.sha256 = ("0" * 64)' \
  '.predicate.buildDefinition.resolvedDependencies[1].uri = "oci://registry.example/other"' \
  '.predicate.buildDefinition.resolvedDependencies[1].digest.sha256 = ("0" * 64)' \
  '.predicate.buildDefinition.resolvedDependencies += [.predicate.buildDefinition.resolvedDependencies[1]]' \
  '.predicate.buildDefinition.resolvedDependencies |= .[1:]' \
  '.predicate.buildDefinition.resolvedDependencies[3].digest.gitCommit = ("0" * 40)'; do
  jq "[($mutation)]" "$fixture" > "$work/invalid.json"
  if check_policy "$work/invalid.json"; then
    echo "FAIL: accepted $mutation" >&2
    exit 1
  fi
  count=$((count + 1))
done

jq -r '.predicate.buildDefinition.resolvedDependencies[] | select(.name == "containerfile") | .content' \
  "$fixture" | base64 --decode > "$work/Containerfile"
"$script_dir/check-containerfile.sh" "$work/Containerfile" "$base"
for recipe in \
  'FROM scratch' \
  "FROM $base AS runtime" \
  "FROM $base\nFROM scratch" \
  "FROM $base\nCOPY --from=builder /file /file" \
  "ARG BASE\nFROM $base" \
  "# syntax=other/frontend\nFROM $base" \
  "FROM $base\nONBUILD RUN echo bad"; do
  printf '%b\n' "$recipe" > "$work/Containerfile"
  if "$script_dir/check-containerfile.sh" "$work/Containerfile" "$base" 2>/dev/null; then
    echo 'FAIL: accepted an ambiguous Containerfile' >&2
    exit 1
  fi
  count=$((count + 1))
done
printf '%s policy checks passed.\n' "$count"
