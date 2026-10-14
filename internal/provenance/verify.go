package provenance

import (
	"context"
	"fmt"
)

type Options struct {
	Image                 string
	Base                  string
	Platform              string
	Workflow              string
	Revision              string
	ProviderKey           string
	ProviderProvenanceKey string
	ProviderIgnoreTlog    bool
}

type Result struct {
	Image                      string    `json:"image"`
	Base                       string    `json:"base"`
	RuntimeBase                string    `json:"runtimeBase"`
	Workflow                   string    `json:"workflow"`
	Revision                   string    `json:"revision"`
	AnonymousRegistryAccess    bool      `json:"anonymousRegistryAccess"`
	CustomerProvenanceVerified bool      `json:"customerProvenanceVerified"`
	ProviderSignatureVerified  bool      `json:"providerSignatureVerified"`
	ProviderProvenanceVerified bool      `json:"providerProvenanceVerified"`
	Statement                  Statement `json:"verifiedStatement"`
}

func Verify(ctx context.Context, o Options) (Result, error) {
	result := Result{Image: o.Image, Base: o.Base, Workflow: o.Workflow, Revision: o.Revision}
	for _, ref := range []string{o.Image, o.Base} {
		if _, _, err := SplitReference(ref); err != nil {
			return result, err
		}
	}
	if !revisionPattern.MatchString(o.Revision) || o.Workflow == "" || o.ProviderKey == "" || o.ProviderProvenanceKey == "" {
		return result, fmt.Errorf("approved workflow commit, workflow identity, and provider keys are required")
	}
	runner, err := NewRunner()
	if err != nil {
		return result, err
	}
	defer runner.Close()
	output, err := runner.Run(ctx, "cosign", "verify-attestation", "--type", "slsaprovenance1",
		"--certificate-identity", o.Workflow, "--certificate-oidc-issuer", "https://token.actions.githubusercontent.com",
		"--certificate-github-workflow-sha", o.Revision, o.Image)
	if err != nil {
		return result, fmt.Errorf("customer provenance: %w", err)
	}
	statements, err := DecodeStatements(output)
	if err != nil {
		return result, err
	}
	parent, err := runner.Resolve(ctx, o.Base, o.Platform)
	if err != nil {
		return result, err
	}
	if child, err := runner.Resolve(ctx, o.Image, o.Platform); err != nil {
		return result, err
	} else if child != o.Image {
		return result, fmt.Errorf("demo expects a single platform output manifest")
	}
	var accepted bool
	var policyError error
	for _, s := range statements {
		policyError = CheckStatement(s, o.Image, o.Base, parent, o.Platform, o.Workflow, o.Revision)
		if policyError == nil {
			result.Statement = s
			accepted = true
			break
		}
	}
	if !accepted {
		return result, fmt.Errorf("ancestry policy: %w", policyError)
	}
	args := []string{"verify", "--key", o.ProviderKey}
	if o.ProviderIgnoreTlog {
		args = append(args, "--insecure-ignore-tlog")
	}
	if _, err := runner.Run(ctx, "cosign", append(args, o.Base)...); err != nil {
		return result, fmt.Errorf("provider signature: %w", err)
	}
	args = []string{"verify-attestation", "--type", "slsaprovenance02", "--new-bundle-format=false", "--key", o.ProviderProvenanceKey}
	if o.ProviderIgnoreTlog {
		args = append(args, "--insecure-ignore-tlog")
	}
	if _, err := runner.Run(ctx, "cosign", append(args, o.Base)...); err != nil {
		return result, fmt.Errorf("original provider provenance: %w", err)
	}
	result.RuntimeBase = parent
	result.AnonymousRegistryAccess = true
	result.CustomerProvenanceVerified = true
	result.ProviderSignatureVerified = true
	result.ProviderProvenanceVerified = true
	return result, nil
}
