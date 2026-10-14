# Verify a customer image built from a signed base

This demo builds an image from Red Hat's signed `core-runtime` base with Buildah in GitHub Actions.
It adds a customer file and publishes the image and signed build provenance to GHCR.
Third parties can verify the base image's origin without registry credentials.

```text
Trusted customer build -> exact core-runtime base -> provider signature
```

The customer provenance records the base digest. The verifier retrieves the provider's signature separately.
The provider does not sign or endorse the customer image.

## Run the demo

Push to `main`, or start the **Build and verify public provenance** workflow from [Actions](https://github.com/p5/base-image-provenance-verification/actions).
It builds with direct Buildah commands, signs provenance with Cosign, and checks anonymous verification.
If GHCR creates a private package, make the demo package public before you rerun verification.

## Verify an image

Install Cosign 3.1.3. The script also uses Bash and standard file utilities.
Review and approve the workflow commit before you trust its build claims.
Get the image reference and workflow commit from the run summary.

```sh
git clone https://github.com/p5/base-image-provenance-verification.git
cd base-image-provenance-verification
git checkout APPROVED_WORKFLOW_COMMIT

scripts/verify.sh \
  ghcr.io/p5/base-image-provenance-verification@sha256:IMAGE_DIGEST \
  APPROVED_WORKFLOW_COMMIT
```

Replace both placeholders with values from the approved run. No registry login is required.
The demo pins one `linux/amd64` base and verifies its recorded use by the approved workflow.
Customer transparency-log checks remain enabled. Existing provider evidence uses public-key checks with transparency-log checks disabled.

## Details

- [Customer demo walkthrough](docs/demo.md): exact commands, explanations, and expected results.
- [How it works](docs/how-it-works.md): simple explanation, real core-runtime data, and rebuilding limits.
- [Verification reference](docs/verification.md): keys, policy, requirements, tests, and supported scope.
- [Build type](docs/build-type.md): provenance fields and the build procedure.
- [Workflow](.github/workflows/demo.yml) and [verification script](scripts/verify.sh).
