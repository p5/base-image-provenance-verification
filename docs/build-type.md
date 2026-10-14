# Build type for the Buildah demo

Build type identifier:

```text
https://github.com/p5/base-image-provenance-verification/blob/main/docs/build-type.md
```

The signed predicate uses `https://slsa.dev/provenance/v1`.
Cosign wraps the predicate in an in-toto statement whose subject is the pushed customer image digest.

## Build procedure

1. Check out the workflow's source commit on a GitHub-hosted runner.
2. Check the restricted Containerfile against the expected provider image reference.
3. Resolve the provider index to exactly one `linux/amd64` image manifest.
4. Check the provider index hash.
5. Run Buildah with the literal Containerfile, `--pull=always`, and `--platform linux/amd64`.
6. Push the customer image with Buildah and read its `--digestfile` output.
7. Record the inputs and Containerfile contents in a SLSA v1 predicate.
8. Pass the predicate to a separate signing job in the same workflow run.
9. Sign the output subject with Cosign and the GitHub workflow identity.
10. Verify the published evidence with anonymous registry access.

The workflow has no user-supplied build parameters.
Repository content and workflow changes require approval through the verifier's expected workflow commit.

## Predicate fields

`buildDefinition.externalParameters` contains:

| Field | Meaning |
|---|---|
| `base` | Digest-pinned provider image reference, which can identify an image index. |
| `platform` | The requested platform, currently `linux/amd64`. |
| `source` | Source repository URL. |
| `revision` | Source and workflow commit. |
| `containerfile` | Recipe filename, currently `Containerfile`. |

`buildDefinition.resolvedDependencies` contains exactly four named resources:

| Name | Meaning |
|---|---|
| `provider-base` | Provider reference and SHA-256 digest. The verifier checks the provider signature on this digest. |
| `runtime-base` | Selected platform manifest reference and SHA-256 digest. |
| `containerfile` | Recipe bytes in standard base64 `content`, with their SHA-256 digest. |
| `source` | Repository URL and `gitCommit` digest. |

The names `provider-base` and `runtime-base` have meanings defined by this build type.
They are not general SLSA fields that other build types must support.

`runDetails.builder.id` identifies the signing GitHub workflow.
`runDetails.builder.version.buildah` records the Buildah version.
`runDetails.metadata.invocationId` identifies the workflow run and attempt.

## Verification rules

Cosign checks the output digest, signing identity, OIDC issuer, and approved workflow commit.
Its CUE policy checks the build parameters and all four dependency roles.
The recorded provider index and runtime platform must match the approved pins in the verifier.
The build job checks the Containerfile. The verifier trusts that check through the approved workflow commit.
The verifier does not parse the recipe or traverse image indexes.
Finally, Cosign verifies the provider image signature and original provider provenance.

The workflow generates these claims. Buildah itself does not generate the SLSA predicate.
The workflow's integrity is part of the verifier's trust decision.
