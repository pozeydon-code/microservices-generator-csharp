package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/pozeydon-code/microservices-generator-csharp/internal/packaging"
)

func main() {
	version := flag.String("version", "", "explicit GitHub Release tag, for example v1.2.3")
	checksumsPath := flag.String("checksums", "", "path to checksums.txt downloaded from that exact GitHub Release")
	outputDir := flag.String("output", "", "directory for rendered package metadata")
	flag.Parse()
	if *version == "" || *checksumsPath == "" || *outputDir == "" {
		fail("--version, --checksums, and --output are required")
	}

	checksums, err := os.Open(*checksumsPath)
	if err != nil {
		fail("open checksums.txt: %v", err)
	}
	defer checksums.Close()
	release, err := packaging.ParseRelease(*version, checksums)
	if err != nil {
		fail("validate release: %v", err)
	}

	if _, err := checksums.Seek(0, 0); err != nil {
		fail("rewind checksums.txt: %v", err)
	}
	bundle, err := packaging.Render(*version, checksums)
	if err != nil {
		fail("render package metadata: %v", err)
	}
	if err := packaging.Validate(bundle, release); err != nil {
		fail("validate package metadata: %v", err)
	}
	if err := bundle.Write(*outputDir, release); err != nil {
		fail("write package metadata: %v", err)
	}
	fmt.Printf("rendered package metadata for %s in %s\n", release.Tag, *outputDir)
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "package-manifests: "+format+"\n", args...)
	os.Exit(2)
}
