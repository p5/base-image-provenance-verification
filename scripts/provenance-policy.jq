# Apply this policy only to statements that Cosign has already verified.
[
  .[]
  | select(._type == "https://in-toto.io/Statement/v1" or ._type == "https://in-toto.io/Statement/v0.1")
  | select(.predicateType == "https://slsa.dev/provenance/v1")
  | select((.subject | length) == 1 and .subject[0].digest.sha256 == ($image | split("@sha256:")[1]))
  | select(.predicate.runDetails.builder.id == $workflow)
  | select(.predicate.buildDefinition.buildType == "https://github.com/p5/base-image-provenance-verification/blob/main/docs/build-type.md")
  | select(.predicate.buildDefinition.externalParameters == {
      base: $base,
      platform: "linux/amd64",
      source: $source,
      revision: $revision,
      containerfile: "Containerfile"
    })
  | .predicate.buildDefinition.resolvedDependencies as $deps
  | select(($deps | length) == 4 and ($deps | map(.name) | unique | length) == 4)
  | ($deps | INDEX(.name)) as $by_name
  | select($by_name["provider-base"].uri == ("oci://" + $base))
  | select($by_name["provider-base"].digest == {sha256: ($base | split("@sha256:")[1])})
  | select($by_name["runtime-base"].uri == ("oci://" + $parent))
  | select($by_name["runtime-base"].digest == {sha256: ($parent | split("@sha256:")[1])})
  | select($by_name.source.uri == $source and $by_name.source.digest == {gitCommit: $revision})
  | select($by_name.containerfile.digest.sha256 | type == "string" and test("^[a-f0-9]{64}$"))
  | select($by_name.containerfile.content | type == "string" and length > 0)
]
| if length > 0 then .[0] else error("no signed statement satisfies the ancestry policy") end
