#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
base=registry.access.redhat.com/hi/core-runtime@sha256:e4b8f4ccc7a1ef92c469f53405de43292132f5da648ec5d2f95024f09762f407
workflow=https://github.com/p5/base-image-provenance-verification/.github/workflows/demo.yml@refs/heads/main
provider_key="$script_dir/../keys/redhat-release3.pub"
provenance_key="$script_dir/../keys/hummingbird.pub"
provider_tlog=()
image=
revision=

while (($#)); do
  case "$1" in
    --image) image=${2:?Image reference required}; shift 2 ;;
    --workflow-sha) revision=${2:?Approved workflow commit required}; shift 2 ;;
    --workflow) workflow=${2:?Workflow identity required}; shift 2 ;;
    --base) base=${2:?Base image required}; shift 2 ;;
    --provider-key) provider_key=${2:?Provider key required}; shift 2 ;;
    --provider-provenance-key) provenance_key=${2:?Provider provenance key required}; shift 2 ;;
    --provider-ignore-tlog) provider_tlog=(--insecure-ignore-tlog); shift ;;
    *) echo "Unknown argument: $1" >&2; exit 2 ;;
  esac
done

for ref in "$image" "$base"; do
  if [[ ! "$ref" =~ ^[^[:space:]@]+/[^[:space:]@]+@sha256:[a-f0-9]{64}$ ]]; then
    echo 'Use --image and --base with immutable SHA-256 references.' >&2
    exit 2
  fi
done
if [[ ! "$revision" =~ ^[a-f0-9]{40}$ || "$workflow" != https://github.com/*/.github/workflows/* ]]; then
  echo 'Supply --workflow-sha with the approved 40-character commit.' >&2
  exit 2
fi
for tool in cosign oras jq sha256sum base64 awk; do command -v "$tool" >/dev/null; done

work=$(mktemp -d)
trap 'rm -rf -- "$work"' EXIT
mkdir "$work/auth"
printf '{"auths":{}}\n' > "$work/auth/config.json"
export DOCKER_CONFIG="$work/auth"
export REGISTRY_AUTH_FILE="$work/auth/config.json"
unset COSIGN_REPOSITORY

check_digest() {
  local actual
  actual="sha256:$(sha256sum "$2" | awk '{print $1}')"
  [[ "$actual" == "$1" ]] || { echo "Digest mismatch: $1" >&2; exit 1; }
}

fetch_manifest() {
  oras manifest fetch --registry-config "$REGISTRY_AUTH_FILE" "$1" > "$2"
  check_digest "${1##*@}" "$2"
}

check_platform() {
  local ref=$1 prefix=$2 config
  fetch_manifest "$ref" "$prefix.manifest.json"
  config=$(jq -er '.config.digest | select(test("^sha256:[a-f0-9]{64}$"))' "$prefix.manifest.json")
  oras blob fetch --registry-config "$REGISTRY_AUTH_FILE" \
    --output "$prefix.config.json" "${ref%@*}@$config"
  check_digest "$config" "$prefix.config.json"
  jq -e '.os == "linux" and .architecture == "amd64"' "$prefix.config.json" >/dev/null
}

# Customer provenance always requires certificate and transparency-log checks.
cosign verify-attestation --type slsaprovenance1 \
  --certificate-identity "$workflow" \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --certificate-github-workflow-sha "$revision" \
  "$image" > "$work/envelopes.jsonl"

fetch_manifest "$base" "$work/base.json"
parent_digest=$(jq -er '
  if has("manifests") then
    [.manifests[] | select(.platform.os == "linux" and .platform.architecture == "amd64")]
    | if length == 1 then .[0].digest else error("ambiguous or missing AMD64 platform") end
  else empty end
' "$work/base.json")
[[ "$parent_digest" =~ ^sha256:[a-f0-9]{64}$ ]]
parent="${base%@*}@$parent_digest"
check_platform "$parent" "$work/parent"
check_platform "$image" "$work/child"

jq -s '[.[] | if type == "array" then .[] else . end | .payload | @base64d | fromjson]' \
  "$work/envelopes.jsonl" > "$work/statements.json"
source=${workflow%%/.github/workflows/*}
jq -e --arg image "$image" --arg base "$base" --arg parent "$parent" \
  --arg workflow "$workflow" --arg revision "$revision" --arg source "$source" \
  -f "$script_dir/provenance-policy.jq" "$work/statements.json" > "$work/statement.json"

jq -er '.predicate.buildDefinition.resolvedDependencies[] | select(.name == "containerfile") | .content' \
  "$work/statement.json" | base64 --decode > "$work/Containerfile"
recipe_digest=$(jq -er '.predicate.buildDefinition.resolvedDependencies[] | select(.name == "containerfile") | .digest.sha256' "$work/statement.json")
check_digest "sha256:$recipe_digest" "$work/Containerfile"
"$script_dir/check-containerfile.sh" "$work/Containerfile" "$base"

cosign verify --key "$provider_key" "${provider_tlog[@]}" "$base" > /dev/null
cosign verify-attestation --type slsaprovenance02 --new-bundle-format=false \
  --key "$provenance_key" "${provider_tlog[@]}" "$base" > /dev/null

jq -n --arg image "$image" --arg base "$base" --arg parent "$parent" \
  --arg workflow "$workflow" --arg revision "$revision" --slurpfile statement "$work/statement.json" '{
    image: $image, base: $base, runtimeBase: $parent, workflow: $workflow, revision: $revision,
    anonymousRegistryAccess: true, customerProvenanceVerified: true,
    providerSignatureVerified: true, providerProvenanceVerified: true,
    verifiedStatement: $statement[0]
  }'
