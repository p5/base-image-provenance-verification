#!/usr/bin/env bash
set -euo pipefail
script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
fixture="$script_dir/../testdata/statement.json"
work=$(mktemp -d)
trap 'rm -rf -- "$work"' EXIT

base=$(jq -r '.predicate.buildDefinition.externalParameters.base' "$fixture")
parent=$(jq -r '.predicate.buildDefinition.resolvedDependencies[] | select(.name == "runtime-base") | .uri | ltrimstr("oci://")' "$fixture")
workflow=$(jq -r '.predicate.runDetails.builder.id' "$fixture")
revision=$(jq -r '.predicate.buildDefinition.externalParameters.revision' "$fixture")
cp "$script_dir/../policy/base.cue" "$work/policy.cue"
printf '_trust: {base: "%s", parent: "%s", workflow: "%s", revision: "%s"}\n' \
  "$base" "$parent" "$workflow" "$revision" > "$work/trust.cue"

check_policy() {
  cue vet -c "$work/policy.cue" "$work/trust.cue" "$1" > /dev/null 2>&1
}

cp "$fixture" "$work/valid.json"
check_policy "$work/valid.json"
jq '.predicate.buildDefinition.resolvedDependencies |= reverse' "$fixture" > "$work/reordered.json"
check_policy "$work/reordered.json"
count=2
for mutation in \
  '.subject[0].digest.sha256 = "invalid"' \
  '.subject = []' \
  '.predicateType = "other"' \
  '.predicate.runDetails.builder.id = "other"' \
  '.predicate.buildDefinition.buildType = "other"' \
  '.predicate.buildDefinition.externalParameters.revision = ("0" * 40)' \
  '.predicate.buildDefinition.externalParameters.source = "https://github.com/attacker/repo"' \
  '.predicate.buildDefinition.externalParameters.base = "registry.example/base:latest"' \
  '.predicate.buildDefinition.externalParameters.platform = "linux/arm64"' \
  '.predicate.buildDefinition.externalParameters.extra = "unexpected input"' \
  'del(.predicate.buildDefinition.externalParameters.base)' \
  'del(.predicate.buildDefinition.resolvedDependencies[0].digest.sha256)' \
  '.predicate.buildDefinition.resolvedDependencies[0].digest.sha256 = ("0" * 64)' \
  '.predicate.buildDefinition.resolvedDependencies[1].uri = "oci://registry.example/other"' \
  '.predicate.buildDefinition.resolvedDependencies[1].digest.sha256 = ("0" * 64)' \
  '.predicate.buildDefinition.resolvedDependencies += [.predicate.buildDefinition.resolvedDependencies[1]]' \
  '.predicate.buildDefinition.resolvedDependencies |= .[1:]' \
  '.predicate.buildDefinition.resolvedDependencies[3].digest.gitCommit = ("0" * 40)'; do
  jq "$mutation" "$fixture" > "$work/invalid.json"
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
