import "strings"

// The caller supplies these trusted values, not the image publisher.
_trust: {
	base:     =~"@sha256:[a-f0-9]{64}$"
	parent:   =~"@sha256:[a-f0-9]{64}$"
	workflow: string
	revision: =~"^[a-f0-9]{40}$"
}
_source: strings.Split(_trust.workflow, "/.github/workflows/")[0]
_baseDigest: strings.Split(_trust.base, "@sha256:")[1]
_parentDigest: strings.Split(_trust.parent, "@sha256:")[1]

"_type"!: "https://in-toto.io/Statement/v1" | "https://in-toto.io/Statement/v0.1"
predicateType!: "https://slsa.dev/provenance/v1"
subject!: [{digest!: {sha256!: =~"^[a-f0-9]{64}$"}}]
predicate!: {
	buildDefinition!: {
		buildType!: "https://github.com/p5/base-image-provenance-verification/blob/main/docs/build-type.md"
		externalParameters!: close({
			base!:          _trust.base
			platform!:      "linux/amd64"
			source!:        _source
			revision!:      _trust.revision
			containerfile!: "Containerfile"
		})
		resolvedDependencies!: [...{
			name!: "provider-base" | "runtime-base" | "containerfile" | "source"
			if name == "provider-base" {
				uri!: "oci://\(_trust.base)"
				digest!: {sha256!: _baseDigest}
			}
			if name == "runtime-base" {
				uri!: "oci://\(_trust.parent)"
				digest!: {sha256!: _parentDigest}
			}
			if name == "containerfile" {
				digest!: {sha256!: =~"^[a-f0-9]{64}$"}
				content!: string & !=""
			}
			if name == "source" {
				uri!: _source
				digest!: {gitCommit!: _trust.revision}
			}
		}]
	}
	runDetails!: {builder!: {id!: _trust.workflow}}
}

// Dependency order has no meaning. Require one resource for each role.
_count: 4 & len(predicate.buildDefinition.resolvedDependencies)
_provider: 1 & len([for d in predicate.buildDefinition.resolvedDependencies if d.name == "provider-base" {d}])
_runtime: 1 & len([for d in predicate.buildDefinition.resolvedDependencies if d.name == "runtime-base" {d}])
_recipe: 1 & len([for d in predicate.buildDefinition.resolvedDependencies if d.name == "containerfile" {d}])
_code: 1 & len([for d in predicate.buildDefinition.resolvedDependencies if d.name == "source" {d}])
