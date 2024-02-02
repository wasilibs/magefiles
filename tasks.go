package magefiles

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/google/go-github/github"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"

	"github.com/wasilibs/magefiles/internal/args"
	"github.com/wasilibs/magefiles/internal/meta"
	"github.com/wasilibs/magefiles/internal/versions"
)

// SetLibraryName sets the library name of the importing project.
// It is used to determine things like build tags.
func SetLibraryName(name string) {
	meta.LibraryName = name
}

// SetLibraryRepo sets the github repository of the upstream library being built.
func SetLibraryRepo(repo string) {
	meta.LibraryRepo = repo
}

// Test runs unit tests - by default, it uses wazero; set WASI_TEST_MODE=cgo or WASI_TEST_MODE=tinygo to use either
func Test() error {
	mode := strings.ToLower(os.Getenv("WASI_TEST_MODE"))

	if mode != "tinygo" {
		return sh.RunV("go", "test", "-v", "-timeout=20m", "-tags", buildTags(), "./...")
	}

	return sh.RunV("tinygo", "test", "-target=wasi", "-v", "-tags", buildTags(), "./...")
}

// Format autoformats the code.
func Format() error {
	if err := sh.RunV("go", "run", fmt.Sprintf("mvdan.cc/gofumpt@%s", versions.GoFumpt), "-l", "-w", "."); err != nil {
		return err
	}
	if err := sh.RunV("go", "run", fmt.Sprintf("github.com/rinchsan/gosimports/cmd/gosimports@%s", versions.GosImports), "-w",
		"-local", "github.com/wasilibs",
		"."); err != nil {
		return nil
	}
	return nil
}

// Lint runs lint checks.
func Lint() error {
	return sh.RunV("go", "run", fmt.Sprintf("github.com/golangci/golangci-lint/cmd/golangci-lint@%s", versions.GolangCILint), "run", "--build-tags", buildTags(), "--timeout", "30m")
}

// Check runs lint and tests.
func Check() {
	mg.SerialDeps(Lint, Test)
}

// Bench runs benchmarks in the default configuration for a Go app, using wazero.
func Bench() error {
	return sh.RunV("go", args.BenchArgs("./...", 1, args.BenchModeWazero)...)
}

// BenchCGO runs benchmarks with cgo instead of wasm. A C++ toolchain and the library must be installed to run.
func BenchCGO() error {
	return sh.RunV("go", args.BenchArgs("./...", 1, args.BenchModeCGO)...)
}

// BenchDefault runs benchmarks using the reference library for comparison.
func BenchDefault() error {
	return sh.RunV("go", args.BenchArgs("./...", 1, args.BenchModeDefault)...)
}

// BenchAll runs all benchmark types and outputs with benchstat. A C++ toolchain and libinjection must be installed to run.
func BenchAll() error {
	if err := os.MkdirAll("build", 0o755); err != nil {
		return err
	}

	fmt.Println("Executing wazero benchmarks")
	wazero, err := sh.Output("go", args.BenchArgs("./...", 5, args.BenchModeWazero)...)
	if err != nil {
		return fmt.Errorf("error running wazero benchmarks: %w", err)
	}
	if err := os.WriteFile(filepath.Join("build", "bench.txt"), []byte(wazero), 0o644); err != nil {
		return err
	}

	fmt.Println("Executing cgo benchmarks")
	cgo, err := sh.Output("go", args.BenchArgs("./...", 5, args.BenchModeCGO)...)
	if err != nil {
		fmt.Println("Error running cgo benchmarks:")
		return fmt.Errorf("error running cgo benchmarks: %w", err)
	}
	if err := os.WriteFile(filepath.Join("build", "bench_cgo.txt"), []byte(cgo), 0o644); err != nil {
		return err
	}

	fmt.Println("Executing default benchmarks")
	def, err := sh.Output("go", args.BenchArgs("./...", 5, args.BenchModeDefault)...)
	if err != nil {
		return fmt.Errorf("error running default benchmarks: %w", err)
	}
	if err := os.WriteFile(filepath.Join("build", "bench_default.txt"), []byte(def), 0o644); err != nil {
		return err
	}

	return sh.RunV("go", "run", fmt.Sprintf("golang.org/x/perf/cmd/benchstat@%s", versions.GolangPerf),
		"build/bench_default.txt", "build/bench.txt", "build/bench_cgo.txt")
}

// UpdateLibs updates the precompiled wasm libraries.
func UpdateLibs() error {
	if err := sh.RunV("docker", "build", "-t", "wasilibs-build", "-f", filepath.Join("buildtools", "wasm", "Dockerfile"), "."); err != nil {
		return err
	}
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	wasmDir := filepath.Join(wd, "internal", "wasm")
	if err := os.MkdirAll(wasmDir, 0o755); err != nil {
		return err
	}
	return sh.RunV("docker", "run", "--rm", "-v", fmt.Sprintf("%s:/out", wasmDir), "wasilibs-build")
}

// UpdateUpstream sets the upstream version in buildtools/wasm/version.txt to the latest.
func UpdateUpstream() error {
	currBytes, err := os.ReadFile(filepath.Join("buildtools", "wasm", "version.txt"))
	if err != nil {
		return err
	}
	curr := strings.TrimSpace(string(currBytes))

	gh, err := api.DefaultRESTClient()
	if err != nil {
		return err
	}

	var release *github.RepositoryRelease
	if err := gh.Get(fmt.Sprintf("repos/%s/releases/latest", meta.LibraryRepo), &release); err != nil {
		return err
	}

	if release == nil {
		return errors.New("could not find releases")
	}

	latest := release.GetTagName()
	if latest == curr {
		fmt.Println("up to date")
		return nil
	}

	fmt.Println("updating to", latest)
	if err := os.WriteFile(filepath.Join("buildtools", "wasm", "version.txt"), []byte(latest), 0o644); err != nil {
		return err
	}

	mg.Deps(UpdateLibs)

	return nil
}

func buildTags() string {
	mode := strings.ToLower(os.Getenv("WASI_TEST_MODE"))

	var tags []string
	if mode == "cgo" {
		tags = append(tags, fmt.Sprintf("%s_cgo", meta.LibraryName))
	}

	return strings.Join(tags, ",")
}
