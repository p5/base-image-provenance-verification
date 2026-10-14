package provenance

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

const (
	DefaultBase     = "registry.access.redhat.com/hi/core-runtime@sha256:e4b8f4ccc7a1ef92c469f53405de43292132f5da648ec5d2f95024f09762f407"
	DefaultWorkflow = "https://github.com/p5/base-image-provenance-verification/.github/workflows/demo.yml@refs/heads/main"
	BuildType       = "https://github.com/p5/base-image-provenance-verification/blob/main/docs/build-type.md"
	PredicateType   = "https://slsa.dev/provenance/v1"
)

var digestPattern = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)
var revisionPattern = regexp.MustCompile(`^[a-f0-9]{40}$`)

type Resource struct {
	Name    string            `json:"name"`
	URI     string            `json:"uri,omitempty"`
	Digest  map[string]string `json:"digest"`
	Content []byte            `json:"content,omitempty"`
}

type Parameters struct {
	Base          string `json:"base"`
	Platform      string `json:"platform"`
	Source        string `json:"source"`
	Revision      string `json:"revision"`
	Containerfile string `json:"containerfile"`
}

type Predicate struct {
	BuildDefinition struct {
		BuildType            string     `json:"buildType"`
		ExternalParameters   Parameters `json:"externalParameters"`
		ResolvedDependencies []Resource `json:"resolvedDependencies"`
	} `json:"buildDefinition"`
	RunDetails struct {
		Builder struct {
			ID      string            `json:"id"`
			Version map[string]string `json:"version"`
		} `json:"builder"`
		Metadata struct {
			InvocationID string `json:"invocationId"`
		} `json:"metadata"`
	} `json:"runDetails"`
}

type Statement struct {
	Type          string     `json:"_type"`
	Subject       []Resource `json:"subject"`
	PredicateType string     `json:"predicateType"`
	Predicate     Predicate  `json:"predicate"`
}

type Inputs struct {
	Base          string `json:"base"`
	Platform      string `json:"platform"`
	RuntimeBase   string `json:"runtimeBase"`
	Containerfile []byte `json:"containerfile"`
}

func Digest(data []byte) string {
	hash := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(hash[:])
}

func SplitReference(ref string) (string, string, error) {
	repo, digest, ok := strings.Cut(ref, "@")
	if !ok || !strings.Contains(repo, "/") || strings.ContainsAny(repo, " \t\n") || !digestPattern.MatchString(digest) {
		return "", "", fmt.Errorf("require a registry image pinned by SHA-256: %q", ref)
	}
	return repo, digest, nil
}

// CheckRecipe limits the demo to one literal FROM and COPY or LABEL instructions.
func CheckRecipe(data []byte, base string) error {
	if _, _, err := SplitReference(base); err != nil {
		return err
	}
	from := 0
	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			return fmt.Errorf("Containerfile directives and comments are outside this demo policy")
		}
		if strings.ContainsAny(line, "\\\r") || strings.Contains(line, "<<") {
			return fmt.Errorf("multiline Containerfile instructions are outside this demo policy")
		}
		fields := strings.Fields(line)
		switch strings.ToUpper(fields[0]) {
		case "FROM":
			from++
			if from != 1 || len(fields) != 2 || fields[1] != base {
				return fmt.Errorf("require one literal FROM %s", base)
			}
		case "COPY":
			if from != 1 || len(fields) < 3 || strings.Contains(strings.ToLower(line), "--from") {
				return fmt.Errorf("require COPY from the local build context")
			}
		case "LABEL":
			if from != 1 || len(fields) < 2 {
				return fmt.Errorf("LABEL must follow FROM")
			}
		default:
			return fmt.Errorf("unsupported Containerfile instruction: %s", fields[0])
		}
	}
	if from != 1 {
		return fmt.Errorf("missing runtime FROM")
	}
	return nil
}

func resource(name, ref string) Resource {
	_, digest, _ := SplitReference(ref)
	return Resource{Name: name, URI: "oci://" + ref, Digest: map[string]string{"sha256": strings.TrimPrefix(digest, "sha256:")}}
}

func Record(inputs Inputs, source, revision, workflow, invocation, buildahVersion string) (Predicate, error) {
	var p Predicate
	if !revisionPattern.MatchString(revision) || source == "" || workflow == "" || invocation == "" || buildahVersion == "" {
		return p, fmt.Errorf("source, 40-character revision, workflow, invocation, and Buildah version are required")
	}
	if err := CheckRecipe(inputs.Containerfile, inputs.Base); err != nil {
		return p, err
	}
	baseRepo, _, err := SplitReference(inputs.Base)
	if err != nil {
		return p, err
	}
	parentRepo, _, err := SplitReference(inputs.RuntimeBase)
	if err != nil || baseRepo != parentRepo {
		return p, fmt.Errorf("runtime base must belong to the provider repository")
	}
	p.BuildDefinition.BuildType = BuildType
	p.BuildDefinition.ExternalParameters = Parameters{inputs.Base, inputs.Platform, source, revision, "Containerfile"}
	p.BuildDefinition.ResolvedDependencies = []Resource{
		resource("provider-base", inputs.Base),
		resource("runtime-base", inputs.RuntimeBase),
		{Name: "containerfile", Digest: map[string]string{"sha256": strings.TrimPrefix(Digest(inputs.Containerfile), "sha256:")}, Content: inputs.Containerfile},
		{Name: "source", URI: source, Digest: map[string]string{"gitCommit": revision}},
	}
	p.RunDetails.Builder.ID = workflow
	p.RunDetails.Builder.Version = map[string]string{"buildah": buildahVersion}
	p.RunDetails.Metadata.InvocationID = invocation
	return p, nil
}

func CheckStatement(s Statement, image, base, runtimeBase, platform, workflow, revision string) error {
	_, imageDigest, err := SplitReference(image)
	if err != nil {
		return err
	}
	if s.Type != "https://in-toto.io/Statement/v1" && s.Type != "https://in-toto.io/Statement/v0.1" {
		return fmt.Errorf("unsupported statement type")
	}
	if s.PredicateType != PredicateType || s.Predicate.BuildDefinition.BuildType != BuildType {
		return fmt.Errorf("unexpected provenance type or build type")
	}
	if len(s.Subject) != 1 || s.Subject[0].Digest["sha256"] != strings.TrimPrefix(imageDigest, "sha256:") {
		return fmt.Errorf("provenance subject does not match the distributed image")
	}
	p := s.Predicate
	if p.RunDetails.Builder.ID != workflow {
		return fmt.Errorf("unexpected builder identity")
	}
	params := p.BuildDefinition.ExternalParameters
	expectedSource, _, ok := strings.Cut(workflow, "/.github/workflows/")
	if !ok || params.Base != base || params.Platform != platform || params.Revision != revision || params.Containerfile != "Containerfile" || params.Source != expectedSource {
		return fmt.Errorf("unexpected build parameters")
	}
	deps := make(map[string]Resource)
	for _, dep := range p.BuildDefinition.ResolvedDependencies {
		if _, exists := deps[dep.Name]; exists {
			return fmt.Errorf("duplicate dependency: %s", dep.Name)
		}
		deps[dep.Name] = dep
	}
	if len(deps) != 4 {
		return fmt.Errorf("require provider base, runtime base, Containerfile, and source dependencies")
	}
	for name, ref := range map[string]string{"provider-base": base, "runtime-base": runtimeBase} {
		want := resource(name, ref)
		got := deps[name]
		if got.URI != want.URI || got.Digest["sha256"] != want.Digest["sha256"] {
			return fmt.Errorf("unexpected %s dependency", name)
		}
	}
	recipe := deps["containerfile"]
	if recipe.Digest["sha256"] != strings.TrimPrefix(Digest(recipe.Content), "sha256:") {
		return fmt.Errorf("Containerfile content does not match its digest")
	}
	if err := CheckRecipe(recipe.Content, base); err != nil {
		return err
	}
	if deps["source"].URI != params.Source || deps["source"].Digest["gitCommit"] != revision {
		return fmt.Errorf("source does not match the approved workflow commit")
	}
	return nil
}

func DecodeStatements(data []byte) ([]Statement, error) {
	var statements []Statement
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	for decoder.More() {
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			return nil, err
		}
		var envelopes []struct {
			Payload []byte `json:"payload"`
		}
		if len(raw) > 0 && raw[0] == '[' {
			if err := json.Unmarshal(raw, &envelopes); err != nil {
				return nil, err
			}
		} else {
			var envelope struct {
				Payload []byte `json:"payload"`
			}
			if err := json.Unmarshal(raw, &envelope); err != nil {
				return nil, err
			}
			envelopes = append(envelopes, envelope)
		}
		for _, envelope := range envelopes {
			var s Statement
			if err := json.Unmarshal(envelope.Payload, &s); err != nil {
				return nil, err
			}
			statements = append(statements, s)
		}
	}
	if len(statements) == 0 {
		return nil, fmt.Errorf("no verified statements")
	}
	return statements, nil
}
