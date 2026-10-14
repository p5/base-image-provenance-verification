# Verification reference

This demo builds a customer image with Buildah in GitHub Actions.
It adds a customer file to a signed Red Hat `core-runtime` image.
It publishes the image and signed SLSA v1 provenance to GHCR.
Third parties can verify the chain without a GitHub account or registry credentials.

```text
Trusted GitHub workflow and commit
  -> signed provenance for the customer image digest
  -> recorded runtime base digest
  -> provider signature on the base image
```

The provider does not sign or endorse the customer image.
The verifier also checks the provider's original build provenance.
The customer provenance references base digests. It does not embed the provider's signature or provenance.

## Run the demo

Push a commit to `main`, or run the **Build and verify public provenance** workflow.
The workflow has four jobs:

1. Test the verifier and check the workflow.
2. Run `buildah build` and `buildah push`, then record resolved build inputs.
3. Sign the provenance with a GitHub OIDC identity through Cosign.
4. Verify the public image and evidence without registry credentials.

Buildah runs directly in `run` steps. No Buildah action or Docker command is used.
Only the signing job receives `id-token: write`.
The build job records provenance through repository code. Buildah does not generate this SLSA statement itself.

New GHCR packages can start private, even when their source repository is public.
Set the package visibility to **Public** in its GitHub package settings.
Then rerun failed jobs if the public verification job failed.
The public verification job deliberately fails when anonymous access is unavailable.

The run summary contains the immutable image reference, workflow commit, and verification command.
The `public-verification` run artifact contains the Cosign-verified attestation envelopes.
That artifact is a convenience. Verification retrieves the signed evidence from the public registry.

## Verify as a third party

Install Cosign 3.1.3. The verifier uses Bash and standard file utilities.
Cosign applies [the CUE policy](../policy/base.cue) through its built-in `--policy` option.
Verification does not require ORAS, `jq`, or the CUE CLI.
Buildah is required to build images. It is not required to verify them.
No Go application or Go toolchain is required.

Review and approve the workflow commit before you trust its claims.
Copy the image reference and workflow commit from the run summary.

```sh
git clone https://github.com/p5/base-image-provenance-verification.git
cd base-image-provenance-verification
git checkout APPROVED_WORKFLOW_COMMIT

scripts/verify.sh \
  ghcr.io/p5/base-image-provenance-verification@sha256:IMAGE_DIGEST \
  APPROVED_WORKFLOW_COMMIT
```

Replace the two placeholders with the approved commit and published image digest.
No registry login is needed. The verifier uses a new, empty registry credential configuration for each run.
It also removes `COSIGN_REPOSITORY` to prevent an environment variable from redirecting signature discovery.

The default workflow identity is:

```text
https://github.com/p5/base-image-provenance-verification/.github/workflows/demo.yml@refs/heads/main
```

Cosign checks that identity, the GitHub OIDC issuer, and the exact approved workflow commit.
Customer certificate and transparency-log checks remain enabled.
The verifier rejects mutable image tags.

This fixed demo checks provider signatures and provenance with the provider's published public keys.
Only those provider checks skip transparency-log verification, because the existing provider evidence does not require those records.
Customer certificate and transparency-log checks remain enabled.
The original provider provenance uses Cosign's explicit legacy attestation format selection.
The script emits Cosign's verified attestation envelopes as JSON on standard output.
It reports success only after all provider checks pass.

## Base image and keys

The demo pins this provider image:

```text
registry.access.redhat.com/hi/core-runtime@sha256:e4b8f4ccc7a1ef92c469f53405de43292132f5da648ec5d2f95024f09762f407
```

The demo also pins the base's AMD64 manifest:

```text
sha256:07247ee02238b21022a5eb877e8e44c4560999627bf5e9dd9ab54922d2d74ae4
```

The build job checks the provider index hash and selects its AMD64 member.
The verifier requires the recorded provider index and runtime base to match these approved pins.
It does not traverse arbitrary image indexes or inspect customer files.
To update the base, review and update both pins in `scripts/verify.sh` and the workflow's base reference.

Provider key sources:

- [Red Hat release key](https://security.access.redhat.com/data/63405576.txt), copied to `keys/redhat-release3.pub`.
- [Hummingbird build key](https://gitlab.com/redhat/hummingbird/containers/-/blob/main/ci/key.pub), copied to `keys/hummingbird.pub`.

Confirm those keys through provider-controlled channels before you trust them.
Customer evidence cannot supply or replace your trusted provider keys.

## What this proves

Verification establishes that the approved workflow signed a build statement for this exact customer image.
The statement records the signed provider image and its selected runtime platform as build inputs.
The trusted workflow uses that literal base reference in its Buildah build.

A signature authenticates a statement. Accuracy depends on the trusted workflow and build environment.
An attacker who controls an approved workflow can sign a false statement.
Approving a workflow commit requires reviewing its code, actions, and build process.

This demo does not claim a SLSA assurance level.
It does not check that customer layers preserve provider files or hardening.
It does not require a separate customer image signature; signed provenance binds the output digest.

## Supported scope

The demo supports one `linux/amd64` image and a restricted, single-stage Containerfile.
The recipe permits one literal, digest-pinned `FROM`, local `COPY` instructions, and `LABEL` instructions.
It rejects build arguments, custom frontends, multiple stages, and copies from other images.
This restriction makes the runtime-base relationship explicit.

See [the build type](build-type.md) for the provenance fields and their meaning.
Support for package installation or general Containerfiles needs a broader recipe policy and matching tests.
The build job enforces the recipe restriction. The verifier trusts that job through the approved workflow commit.

If you copy the image to another registry, copy its signed provenance as well.
Repeat anonymous verification against the destination reference.

## Local checks

Install the CUE CLI 0.14.1, `jq`, ShellCheck, and actionlint for development checks.
Consumers do not need these tools.

```sh
scripts/test-policy.sh
shellcheck scripts/*.sh
actionlint
```

The 27 checks cover CUE policy constraints and the build job's recipe restrictions.
Cosign checks the output digest, workflow identity, and approved commit.
The Actions job also rejects a wrong provider key and a wrong workflow commit against real signed evidence.

## References

- [Buildah build](https://github.com/containers/buildah/blob/main/docs/buildah-build.1.md).
- [SLSA provenance](https://slsa.dev/spec/v1.2/provenance).
- [Cosign attestation verification](https://docs.sigstore.dev/cosign/verifying/attestation/).
