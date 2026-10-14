# How verification works

The provider supplies a signed base image, AAA.
The customer builds and publishes an extended image, BBB.
A third party wants to verify that BBB uses a base image from the provider.

Think of two signed receipts:

- The provider's signature says, "I supplied this exact base image."
- The customer's build provenance says, "This workflow built this exact customer image using that base."

SHA-256 digests identify the exact images. Repository names tell the verifier where to retrieve the evidence.
Anyone can copy a digest into a false statement. The verifier must also trust the workflow that records the build.

## What the verifier does

1. Retrieve BBB's provenance using its repository and digest.
2. Verify the provenance signature against the approved GitHub workflow identity and commit.
3. Check that the provenance identifies BBB and the expected base image.
4. Retrieve the base image's signature using its repository and digest.
5. Verify that signature with the trusted provider public key.
6. Retrieve and verify the base image's original build provenance separately.

```text
Customer image BBB
  -> signed customer build provenance
  -> exact core-runtime base reference
  -> provider signature and original provider provenance
```

BBB's provenance contains base references and digests.
It does not contain the base image's signature or original provenance.
The provider does not sign or endorse BBB.
The consumer needs no customer credentials. All required image and attestation data must remain publicly accessible.

## Real core-runtime example

These examples use a verified image from [this successful demo run](https://github.com/p5/base-image-provenance-verification/actions/runs/38643680421).
They show selected fields. Full attestation bundles also include other fields and signature material.

BBB is the published customer image:

```text
ghcr.io/p5/base-image-provenance-verification@sha256:37a642ce3f0696981a946bf2beac10f1a9b15c21a9fb7e739d612fb7bb0b10c4
```

Its signed SLSA v1 provenance includes:

```json
{
  "_type": "https://in-toto.io/Statement/v1",
  "subject": [{
    "name": "ghcr.io/p5/base-image-provenance-verification",
    "digest": {
      "sha256": "37a642ce3f0696981a946bf2beac10f1a9b15c21a9fb7e739d612fb7bb0b10c4"
    }
  }],
  "predicateType": "https://slsa.dev/provenance/v1",
  "predicate": {
    "buildDefinition": {
      "externalParameters": {
        "base": "registry.access.redhat.com/hi/core-runtime@sha256:e4b8f4ccc7a1ef92c469f53405de43292132f5da648ec5d2f95024f09762f407",
        "platform": "linux/amd64"
      },
      "resolvedDependencies": [
        {
          "name": "provider-base",
          "uri": "oci://registry.access.redhat.com/hi/core-runtime@sha256:e4b8f4ccc7a1ef92c469f53405de43292132f5da648ec5d2f95024f09762f407",
          "digest": {
            "sha256": "e4b8f4ccc7a1ef92c469f53405de43292132f5da648ec5d2f95024f09762f407"
          }
        },
        {
          "name": "runtime-base",
          "uri": "oci://registry.access.redhat.com/hi/core-runtime@sha256:07247ee02238b21022a5eb877e8e44c4560999627bf5e9dd9ab54922d2d74ae4",
          "digest": {
            "sha256": "07247ee02238b21022a5eb877e8e44c4560999627bf5e9dd9ab54922d2d74ae4"
          }
        }
      ]
    }
  }
}
```

The two base digests identify different objects:

| Digest prefix | Object |
|---|---|
| `e4b8…` | The signed core-runtime index, which lists platform-specific images. |
| `0724…` | The AMD64 image selected from that index for this build. |

The names `provider-base` and `runtime-base` are roles defined by this demo's build type.
They are not universal SLSA fields.
Other builders can record base information differently, including through a digest-linked SBOM.
SLSA provenance alone does not guarantee that every builder records the base image.

The verifier checks the provider's signature on the index at this reference:

```text
registry.access.redhat.com/hi/core-runtime@sha256:e4b8f4ccc7a1ef92c469f53405de43292132f5da648ec5d2f95024f09762f407
```

It also retrieves core-runtime's original provenance at that reference.
That provenance uses SLSA v0.2 and includes this source input:

```json
{
  "predicateType": "https://slsa.dev/provenance/v0.2",
  "predicate": {
    "builder": {
      "id": "https://tekton.dev/chains/v2"
    },
    "buildType": "tekton.dev/v1/PipelineRun",
    "materials": [{
      "uri": "git+https://gitlab.com/redhat/hummingbird/containers.git",
      "digest": {
        "sha1": "65b713b7d65822f7b35292f27266d40c1cfd314e"
      }
    }]
  }
}
```

Here, `sha1` identifies the Git source commit. The image references use SHA-256.
The original provenance can retain build-repository names after publication to another registry.
Verification must bind the statement to the expected image digest.

## Verification and rebuilding are different

The customer provenance identifies the base image used by the trusted workflow.
The base image's own provenance describes how the provider built that base.
That information can help someone rebuild it.

Rebuilding also requires access to the source, dependencies, build tools, and necessary build settings.
Provenance does not guarantee that all inputs remain available.
It does not guarantee an identical output digest. That also requires a reproducible build.

The demo verifies origin and the recorded build relationship. It does not rebuild either image.
It does not check that customer layers preserve provider files or hardening.

See the [verification reference](verification.md) for key sources, trust settings, and implementation limits.

## References

- [SLSA build provenance](https://slsa.dev/spec/v1.2/build-provenance).
- [SLSA artifact verification](https://slsa.dev/spec/v1.2/verifying-artifacts).
- [Cosign attestation verification](https://docs.sigstore.dev/cosign/verifying/attestation/).
