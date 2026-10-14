# Customer demo walkthrough

This walkthrough verifies an existing public build. It does not require a GitHub account or registry login.
Allow about ten minutes. Registry access requires a network connection.

## Explain the purpose

Say:

> You distribute your own image. Your users want evidence that it uses our signed base image.
> Your build publishes a signed statement with the exact base digest.
> Users verify your build statement, then verify our signature on that base digest.
> Our signature stays with our base image. Your statement links your image to that base.

For this demo, Red Hat is the base provider. This repository acts as the customer's build repository.

## Explain CUE

Cosign verifies signatures and the identity of the signer.
CUE defines which statements the verifier accepts.
Cosign applies the CUE policy through its built-in `--policy` option.

Our policy requires the approved base digests, workflow, source commit, platform, and build input fields.
A valid signature alone does not establish that the statement names the approved base.
CUE rejects a signed statement that fails these rules.

Say:

> Cosign checks who signed this statement. CUE checks whether the statement meets our rules.
> Neither tool can prove that a dishonest builder told the truth. We must trust the approved build workflow.

Consumers do not install CUE. Cosign includes CUE policy support.
We could implement the same rules with another policy tool. CUE avoids a separate verification application here.

## Prepare the terminal

Use Bash. Install Git, Cosign 3.1.3, and `jq` before the demonstration.
The verification script does not require `jq`. These display commands use it to show the verified statement.
Buildah is only necessary to build a new image.

Check the tools:

```bash
command -v git cosign jq
cosign version
```

Expect paths for all three tools and `GitVersion: v3.1.3`.

Clone the repository into a new directory:

```bash
git clone https://github.com/p5/base-image-provenance-verification.git
cd base-image-provenance-verification

WORKFLOW_COMMIT=73d3646142299e32b227b12495bc661823b67409
IMAGE=ghcr.io/p5/base-image-provenance-verification@sha256:d0a59cb700225b9ff3454551b005b4362df17c725d2fd57d1d80e49c8b7414e8
git checkout "$WORKFLOW_COMMIT"
```

Git reports a detached HEAD. This is expected because the demo uses an exact commit.
Keep this terminal open. The subsequent commands use these two variables.

These values identify [a successful build](https://github.com/p5/base-image-provenance-verification/actions/runs/38648276123).
The image digest identifies the customer image. The commit identifies the approved build code.
Review the workflow before you approve its commit.
Confirm the [provider keys](verification.md#base-image-and-keys) through provider-controlled channels.

## Show what the customer builds

Type:

```bash
cat Containerfile
cat customer/message.txt
```

The `FROM` instruction names the pinned `core-runtime` base.
The `COPY` instruction adds `/customer/message.txt`.
The workflow uses direct Buildah commands to build and publish this image.
A separate job signs the build statement with Cosign and the GitHub workflow identity.

Say:

> This customer adds a file to the provider's image.
> The build records the provider's exact image digest, rather than the changing `latest` tag.

This demo restricts the recipe to one base and local file additions.
Package installation requires broader recipe support. It does not require a different signature model.

## Verify as a user

Type:

```bash
scripts/verify.sh "$IMAGE" "$WORKFLOW_COMMIT" > verified.jsonl
```

Expect exit status zero and this final message:

```text
Verified customer provenance and signed provider base for ghcr.io/p5/base-image-provenance-verification@sha256:d0a59cb700225b9ff3454551b005b4362df17c725d2fd57d1d80e49c8b7414e8
```

The script uses an empty registry credential configuration. Your existing registry login does not affect this check.
It retrieves the evidence from public registries.
It checks the customer statement's signature, output digest, workflow identity, workflow commit, and CUE policy.
It then checks the provider's base signature and original build provenance with the published provider keys.

Cosign prints warnings about skipped provider transparency-log checks and a deprecated attestation format flag.
The existing provider evidence uses public-key verification and a legacy attestation format.
These provider checks skip transparency-log verification. Customer certificate and transparency-log checks remain enabled.
The warnings do not mean that the script skips provider signature verification.

Say:

> A third party can run this command without an account.
> The checks bind this exact customer image to an approved build statement and the signed provider base.

## Show the verified base reference

Type:

```bash
jq -r '.payload | @base64d | fromjson | {
  customer: .subject[0].digest,
  base: .predicate.buildDefinition.externalParameters.base,
  inputs: [.predicate.buildDefinition.resolvedDependencies[]
    | select(.name == "provider-base" or .name == "runtime-base")
    | {name, uri}],
  builder: .predicate.runDetails.builder.id
}' verified.jsonl
```

Expect the customer digest to start with `d0a59cb7` and the provider base digest to start with `e4b8f4cc`.
The `runtime-base` digest starts with `07247ee0`.

The provider base is a multi-platform image index. The runtime base is its selected `linux/amd64` manifest.
The build checks that relationship. The verifier requires both approved digests.

Say:

> This is the data from the verified customer statement.
> It contains the base reference and digest. It does not contain the provider's signature.
> The verifier retrieves that signature separately, using the base reference.

Decoding JSON does not verify a signature. This command displays evidence that the preceding verification command accepted.

## Show why CUE matters

Keep the same image and signing identity. Change the acceptance rule to require a different base.

Type:

```bash
cat > wrong-base.cue <<'EOF'
predicate: buildDefinition: externalParameters: base: "registry.access.redhat.com/hi/core-runtime@sha256:0000000000000000000000000000000000000000000000000000000000000000"
EOF

cosign verify-attestation --type slsaprovenance1 \
  --certificate-identity https://github.com/p5/base-image-provenance-verification/.github/workflows/demo.yml@refs/heads/main \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --certificate-github-workflow-sha "$WORKFLOW_COMMIT" \
  --policy wrong-base.cue \
  "$IMAGE"
```

Expect a nonzero exit status and a policy error with `conflicting values` for the base field.
This command deliberately fails. Run it without a shell's `set -e` option so the terminal remains available.
This small policy demonstrates one rule. The normal script applies the complete repository policy.

Say:

> The signature still belongs to the approved workflow. The image has not changed.
> We asked for a different base, so CUE rejects the statement.

## Explain the trust boundary

Say:

> Users trust the provider's public keys and the customer's approved build workflow.
> A compromised approved builder can sign false claims. Signatures alone cannot prevent that.
> These checks establish base origin. They do not check whether customer layers preserve the base files or hardening.

The provider does not endorse the customer image.
The provider's original provenance can help a rebuild. It does not guarantee available inputs or identical rebuild output.

## Build a fresh image before a demonstration

The preceding demonstration uses fixed, tested evidence. To publish fresh evidence, start the workflow from the repository's Actions page.
Alternatively, use GitHub CLI with an account that can run this repository's workflow:

```bash
gh workflow run demo.yml --repo p5/base-image-provenance-verification --ref main
```

Wait for all four jobs to pass. Open the run summary.
Copy its image reference and workflow commit into `IMAGE` and `WORKFLOW_COMMIT`.
Check out that commit before you repeat the verification commands.
Do not combine an image from one run with a commit from another run.
