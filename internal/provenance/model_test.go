package provenance

import (
	"encoding/json"
	"strings"
	"testing"
)

const testRevision = "0123456789abcdef0123456789abcdef01234567"

var testImage = "ghcr.io/p5/demo@sha256:" + strings.Repeat("b", 64)
var testParent = "registry.access.redhat.com/hi/core-runtime@sha256:" + strings.Repeat("a", 64)

func fixture(t *testing.T) Statement {
	t.Helper()
	p, err := Record(Inputs{DefaultBase, "linux/amd64", testParent, []byte("FROM " + DefaultBase + "\nCOPY customer/message.txt /message.txt\n")},
		"https://github.com/p5/base-image-provenance-verification", testRevision, DefaultWorkflow,
		"https://github.com/p5/base-image-provenance-verification/actions/runs/1/attempts/1", "buildah version test")
	if err != nil {
		t.Fatal(err)
	}
	_, digest, _ := SplitReference(testImage)
	return Statement{Type: "https://in-toto.io/Statement/v1", PredicateType: PredicateType, Subject: []Resource{{Digest: map[string]string{"sha256": strings.TrimPrefix(digest, "sha256:")}}}, Predicate: p}
}

func TestAncestryPolicy(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Statement)
		reject bool
	}{
		{"valid", func(*Statement) {}, false},
		{"wrong output", func(s *Statement) { s.Subject[0].Digest["sha256"] = strings.Repeat("c", 64) }, true},
		{"wrong schema", func(s *Statement) { s.PredicateType = "https://slsa.dev/provenance/v0.2" }, true},
		{"wrong build type", func(s *Statement) { s.Predicate.BuildDefinition.BuildType = "other" }, true},
		{"wrong builder", func(s *Statement) { s.Predicate.RunDetails.Builder.ID = "other" }, true},
		{"wrong revision", func(s *Statement) { s.Predicate.BuildDefinition.ExternalParameters.Revision = strings.Repeat("c", 40) }, true},
		{"wrong base parameter", func(s *Statement) { s.Predicate.BuildDefinition.ExternalParameters.Base = testParent }, true},
		{"wrong platform", func(s *Statement) { s.Predicate.BuildDefinition.ExternalParameters.Platform = "linux/arm64" }, true},
		{"wrong parent", func(s *Statement) {
			s.Predicate.BuildDefinition.ResolvedDependencies[1].Digest["sha256"] = strings.Repeat("c", 64)
		}, true},
		{"wrong provider digest", func(s *Statement) { s.Predicate.BuildDefinition.ResolvedDependencies[0].URI = "oci://" + testParent }, true},
		{"duplicate parent", func(s *Statement) {
			s.Predicate.BuildDefinition.ResolvedDependencies = append(s.Predicate.BuildDefinition.ResolvedDependencies, s.Predicate.BuildDefinition.ResolvedDependencies[1])
		}, true},
		{"missing parent", func(s *Statement) {
			s.Predicate.BuildDefinition.ResolvedDependencies = s.Predicate.BuildDefinition.ResolvedDependencies[1:]
		}, true},
		{"changed recipe", func(s *Statement) {
			s.Predicate.BuildDefinition.ResolvedDependencies[2].Content = []byte("FROM scratch\n")
		}, true},
		{"wrong source commit", func(s *Statement) {
			s.Predicate.BuildDefinition.ResolvedDependencies[3].Digest["gitCommit"] = strings.Repeat("c", 40)
		}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := fixture(t)
			tt.mutate(&s)
			err := CheckStatement(s, testImage, DefaultBase, testParent, "linux/amd64", DefaultWorkflow, testRevision)
			if (err != nil) != tt.reject {
				t.Fatalf("reject=%v, error=%v", tt.reject, err)
			}
		})
	}
}

func TestRejectAmbiguousRecipes(t *testing.T) {
	for _, recipe := range []string{
		"", "FROM scratch\n", "FROM " + DefaultBase + " AS runtime\n", "ARG BASE\nFROM ${BASE}\n",
		"FROM " + DefaultBase + "\nFROM scratch\n", "FROM " + DefaultBase + "\nCOPY --from=builder /x /x\n",
		"# syntax=custom/frontend\nFROM " + DefaultBase + "\n", "FROM " + DefaultBase + "\nONBUILD RUN echo bad\n",
		"FROM " + DefaultBase + "\nCOPY <<EOF /file\n", "FROM " + DefaultBase + "\\\n",
	} {
		if err := CheckRecipe([]byte(recipe), DefaultBase); err == nil {
			t.Errorf("accepted recipe %q", recipe)
		}
	}
}

func TestDecodeVerifiedEnvelopes(t *testing.T) {
	s := fixture(t)
	statement, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	envelope, err := json.Marshal(struct {
		Payload []byte `json:"payload"`
	}{statement})
	if err != nil {
		t.Fatal(err)
	}
	for _, input := range []string{string(envelope), "[" + string(envelope) + "]", string(envelope) + "\n" + string(envelope)} {
		got, err := DecodeStatements([]byte(input))
		if err != nil || len(got) == 0 {
			t.Fatalf("decode: %v", err)
		}
		if err := CheckStatement(got[0], testImage, DefaultBase, testParent, "linux/amd64", DefaultWorkflow, testRevision); err != nil {
			t.Fatal(err)
		}
	}
	for _, input := range []string{"", "{}", "[]", "{", `{"payload":"not base64"}`} {
		if _, err := DecodeStatements([]byte(input)); err == nil {
			t.Errorf("accepted %q", input)
		}
	}
}

func TestRejectMutableReferences(t *testing.T) {
	for _, ref := range []string{"ghcr.io/p5/demo:latest", "ghcr.io/p5/demo@sha256:bad", "@sha256:" + strings.Repeat("a", 64)} {
		if _, _, err := SplitReference(ref); err == nil {
			t.Errorf("accepted %q", ref)
		}
	}
}
