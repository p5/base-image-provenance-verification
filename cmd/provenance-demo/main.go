package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/p5/base-image-provenance-verification/internal/provenance"
	"os"
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func writeJSON(filename string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if filename == "-" {
		_, err = os.Stdout.Write(data)
		return err
	}
	return os.WriteFile(filename, data, 0600)
}

func run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: provenance-demo prepare|record|verify [flags]")
	}
	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	base := fs.String("base", provenance.DefaultBase, "expected digest-pinned provider image")
	platform := fs.String("platform", "linux/amd64", "target platform")
	output := fs.String("output", "-", "JSON output file, or - for stdout")
	switch args[0] {
	case "prepare":
		file := fs.String("containerfile", "Containerfile", "single-stage build recipe")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		recipe, err := os.ReadFile(*file)
		if err != nil {
			return err
		}
		inputs, err := provenance.Prepare(ctx, *base, *platform, recipe)
		if err != nil {
			return err
		}
		return writeJSON(*output, inputs)
	case "record":
		input := fs.String("inputs", "results/inputs.json", "resolved build inputs")
		source := fs.String("source", "https://github.com/p5/base-image-provenance-verification", "source repository")
		revision := fs.String("revision", "", "source and workflow commit")
		workflow := fs.String("workflow", provenance.DefaultWorkflow, "signing workflow identity")
		invocation := fs.String("invocation", "", "GitHub Actions run URL")
		version := fs.String("buildah-version", "", "Buildah version")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		data, err := os.ReadFile(*input)
		if err != nil {
			return err
		}
		var inputs provenance.Inputs
		if err := json.Unmarshal(data, &inputs); err != nil {
			return err
		}
		current, err := os.ReadFile("Containerfile")
		if err != nil {
			return err
		}
		if provenance.Digest(current) != provenance.Digest(inputs.Containerfile) {
			return fmt.Errorf("Containerfile changed after input resolution")
		}
		p, err := provenance.Record(inputs, *source, *revision, *workflow, *invocation, *version)
		if err != nil {
			return err
		}
		return writeJSON(*output, p)
	case "verify":
		var o provenance.Options
		fs.StringVar(&o.Image, "image", "", "distributed image pinned by digest")
		fs.StringVar(&o.Workflow, "workflow", provenance.DefaultWorkflow, "approved signing workflow identity")
		fs.StringVar(&o.Revision, "workflow-sha", "", "approved 40-character workflow and source commit")
		fs.StringVar(&o.ProviderKey, "provider-key", "keys/redhat-release3.pub", "trusted provider public key")
		fs.StringVar(&o.ProviderProvenanceKey, "provider-provenance-key", "keys/hummingbird.pub", "trusted original provenance public key")
		fs.BoolVar(&o.ProviderIgnoreTlog, "provider-ignore-tlog", false, "skip only provider transparency-log checks")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		o.Base = *base
		o.Platform = *platform
		result, err := provenance.Verify(ctx, o)
		if err != nil {
			return err
		}
		return writeJSON(*output, result)
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}
