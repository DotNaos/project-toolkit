package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/DotNaos/repo-kit/cli/internal/checker"
	"github.com/DotNaos/repo-kit/cli/internal/config"
	"github.com/DotNaos/repo-kit/cli/internal/syncer"
)

const defaultSourceURL = "https://github.com/DotNaos/repo-kit"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "init":
		if err := runInit(os.Args[2:]); err != nil {
			die(err)
		}
	case "sync":
		if err := runSync(os.Args[2:]); err != nil {
			die(err)
		}
	case "check":
		if err := runCheck(os.Args[2:]); err != nil {
			die(err)
		}
	case "update":
		if err := runUpdate(os.Args[2:]); err != nil {
			die(err)
		}
	default:
		usage()
		os.Exit(1)
	}
}

func runInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	target := fs.String("target", ".", "target repository path")
	manifest := fs.String("manifest", "default", "manifest name")
	sourceURL := fs.String("source-url", defaultSourceURL, "source repository URL")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg := config.RepositoryConfig{Version: 1, Manifest: *manifest, SourceURL: *sourceURL}
	if err := config.Write(filepath.Clean(*target), cfg); err != nil {
		return err
	}
	fmt.Println("initialized .repo-kit/config.yaml")
	return nil
}

func runSync(args []string) error {
	fs := flag.NewFlagSet("sync", flag.ContinueOnError)
	target := fs.String("target", ".", "target repository path")
	kitRoot := fs.String("kit-root", ".", "repo-kit source path")
	if err := fs.Parse(args); err != nil {
		return err
	}

	return syncer.Sync(filepath.Clean(*target), filepath.Clean(*kitRoot))
}

func runCheck(args []string) error {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	target := fs.String("target", ".", "target repository path")
	if err := fs.Parse(args); err != nil {
		return err
	}

	drift, err := checker.Drift(filepath.Clean(*target))
	if err != nil {
		return err
	}
	if len(drift) == 0 {
		fmt.Println("no drift detected")
		return nil
	}
	for _, d := range drift {
		fmt.Println(d)
	}
	return fmt.Errorf("drift detected")
}

func runUpdate(args []string) error {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	target := fs.String("target", ".", "target repository path")
	kitRoot := fs.String("kit-root", ".", "repo-kit source path")
	if err := fs.Parse(args); err != nil {
		return err
	}

	return syncer.Sync(filepath.Clean(*target), filepath.Clean(*kitRoot))
}

func die(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}

func usage() {
	fmt.Println("repo-kit <init|sync|check|update>")
}
