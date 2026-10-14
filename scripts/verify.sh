#!/usr/bin/env bash
set -euo pipefail

if (($# != 2)) || [[ ! "$1" =~ ^[^[:space:]@]+/[^[:space:]@]+@sha256:[a-f0-9]{64}$ || ! "$2" =~ ^[a-f0-9]{40}$ ]]; then
  echo 'Usage: scripts/verify.sh IMAGE@sha256:DIGEST APPROVED_WORKFLOW_COMMIT' >&2
  exit 2
fi
image=$1
revision=$2
script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
base=registry.access.redhat.com/hi/core-runtime@sha256:e4b8f4ccc7a1ef92c469f53405de43292132f5da648ec5d2f95024f09762f407
parent=registry.access.redhat.com/hi/core-runtime@sha256:07247ee02238b21022a5eb877e8e44c4560999627bf5e9dd9ab54922d2d74ae4
workflow=https://github.com/p5/base-image-provenance-verification/.github/workflows/demo.yml@refs/heads/main
provider_key=${PROVIDER_KEY:-"$script_dir/../keys/redhat-release3.pub"}

# Verify public access without the caller's registry credentials.
work=$(mktemp -d)
trap 'rm -rf -- "$work"' EXIT
export DOCKER_CONFIG="$work"
export REGISTRY_AUTH_FILE="$work/config.json"
printf '{"auths":{}}\n' > "$REGISTRY_AUTH_FILE"
unset COSIGN_REPOSITORY
cp "$script_dir/../policy/base.cue" "$work/policy.cue"
printf '_trust: {base: "%s", parent: "%s", workflow: "%s", revision: "%s"}\n' \
  "$base" "$parent" "$workflow" "$revision" > "$work/trust.cue"

# Cosign checks the output digest, workflow identity, commit, and ancestry policy.
cosign verify-attestation --type slsaprovenance1 \
  --certificate-identity "$workflow" \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --certificate-github-workflow-sha "$revision" \
  --policy "$work/policy.cue" --policy "$work/trust.cue" \
  "$image" > "$work/verified.jsonl"

# Existing provider evidence uses published keys without transparency-log records.
cosign verify --key "$provider_key" --insecure-ignore-tlog "$base" > /dev/null
cosign verify-attestation --type slsaprovenance02 --new-bundle-format=false \
  --key "$script_dir/../keys/hummingbird.pub" --insecure-ignore-tlog "$base" > /dev/null

printf 'Verified customer provenance and signed provider base for %s\n' "$image" >&2
cat "$work/verified.jsonl"
