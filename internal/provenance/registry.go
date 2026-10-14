package provenance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Runner gives registry tools an empty credential configuration.
type Runner struct{ Directory string }

func NewRunner() (Runner, error) {
	dir, err := os.MkdirTemp("", "public-provenance-*")
	if err != nil {
		return Runner{}, err
	}
	if err := os.WriteFile(dir+"/config.json", []byte(`{"auths":{}}`), 0600); err != nil {
		os.RemoveAll(dir)
		return Runner{}, err
	}
	return Runner{dir}, nil
}

func (r Runner) Close() { _ = os.RemoveAll(r.Directory) }

func (r Runner) Run(ctx context.Context, tool string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, tool, args...)
	for _, value := range os.Environ() {
		name, _, _ := strings.Cut(value, "=")
		if name != "DOCKER_CONFIG" && name != "REGISTRY_AUTH_FILE" && name != "COSIGN_REPOSITORY" {
			command.Env = append(command.Env, value)
		}
	}
	command.Env = append(command.Env, "DOCKER_CONFIG="+r.Directory, "REGISTRY_AUTH_FILE="+r.Directory+"/config.json")
	var stderr bytes.Buffer
	command.Stderr = &stderr
	data, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w: %s", tool, args[0], err, stderr.String())
	}
	return data, nil
}

type descriptor struct {
	Digest   string `json:"digest"`
	Size     int64  `json:"size"`
	Platform struct {
		OS           string `json:"os"`
		Architecture string `json:"architecture"`
	} `json:"platform"`
}
type manifest struct {
	Manifests []descriptor `json:"manifests"`
	Config    descriptor   `json:"config"`
}

func (r Runner) fetch(ctx context.Context, ref string, blob bool) ([]byte, error) {
	_, digest, err := SplitReference(ref)
	if err != nil {
		return nil, err
	}
	args := []string{"manifest", "fetch", "--registry-config", r.Directory + "/config.json", ref}
	if blob {
		args = []string{"blob", "fetch", "--registry-config", r.Directory + "/config.json", "--output", "-", ref}
	}
	data, err := r.Run(ctx, "oras", args...)
	if err != nil {
		return nil, err
	}
	if Digest(data) != digest {
		return nil, fmt.Errorf("registry content does not match %s", digest)
	}
	return data, nil
}

func (r Runner) Resolve(ctx context.Context, ref, platform string) (string, error) {
	if platform != "linux/amd64" {
		return "", fmt.Errorf("demo supports only linux/amd64")
	}
	for range 5 {
		repo, _, err := SplitReference(ref)
		if err != nil {
			return "", err
		}
		data, err := r.fetch(ctx, ref, false)
		if err != nil {
			return "", err
		}
		var m manifest
		if err := json.Unmarshal(data, &m); err != nil {
			return "", err
		}
		if len(m.Manifests) > 0 {
			var matches []descriptor
			for _, child := range m.Manifests {
				if child.Platform.OS == "linux" && child.Platform.Architecture == "amd64" {
					matches = append(matches, child)
				}
			}
			if len(matches) != 1 {
				return "", fmt.Errorf("require exactly one linux/amd64 manifest")
			}
			ref = repo + "@" + matches[0].Digest
			continue
		}
		config, err := r.fetch(ctx, repo+"@"+m.Config.Digest, true)
		if err != nil {
			return "", err
		}
		var identity struct {
			OS           string `json:"os"`
			Architecture string `json:"architecture"`
		}
		if err := json.Unmarshal(config, &identity); err != nil {
			return "", err
		}
		if identity.OS != "linux" || identity.Architecture != "amd64" {
			return "", fmt.Errorf("image config has the wrong platform")
		}
		return ref, nil
	}
	return "", fmt.Errorf("image index nesting is too deep")
}

func Prepare(ctx context.Context, base, platform string, recipe []byte) (Inputs, error) {
	if err := CheckRecipe(recipe, base); err != nil {
		return Inputs{}, err
	}
	runner, err := NewRunner()
	if err != nil {
		return Inputs{}, err
	}
	defer runner.Close()
	parent, err := runner.Resolve(ctx, base, platform)
	if err != nil {
		return Inputs{}, err
	}
	return Inputs{base, platform, parent, recipe}, nil
}
