package waf

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/magefile/mage/sh"

	"github.com/wasilibs/magefiles/internal/args"
	"github.com/wasilibs/magefiles/internal/versions"
)

// WAFBench runs benchmarks in the default configuration for a Go app, using wazero.
func WAFBench() error {
	return sh.RunV("go", args.BenchArgs("./wafbench", 1, args.BenchModeWazero)...)
}

// WAFBenchCGO runs benchmarks with cgo instead of wasm. A C++ toolchain and the library must be installed to run.
func WAFBenchCGO() error {
	return sh.RunV("go", args.BenchArgs("./wafbench", 1, args.BenchModeCGO)...)
}

// WAFBenchDefault runs benchmarks using the reference library for comparison.
func WAFBenchDefault() error {
	return sh.RunV("go", args.BenchArgs("./wafbench", 1, args.BenchModeDefault)...)
}

// WAFBenchAll runs all benchmark types and outputs with benchstat. A C++ toolchain and libinjection must be installed to run.
func WAFBenchAll() error {
	if err := os.MkdirAll("build", 0o755); err != nil {
		return err
	}

	fmt.Println("Executing wazero benchmarks")
	wazero, err := sh.Output("go", args.BenchArgs("./wafbench", 5, args.BenchModeWazero)...)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join("build", "wafbench.txt"), []byte(wazero), 0o644); err != nil {
		return err
	}

	fmt.Println("Executing cgo benchmarks")
	cgo, err := sh.Output("go", args.BenchArgs("./wafbench", 5, args.BenchModeCGO)...)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join("build", "wafbench_cgo.txt"), []byte(cgo), 0o644); err != nil {
		return err
	}

	fmt.Println("Executing default benchmarks")
	def, err := sh.Output("go", args.BenchArgs("./wafbench", 5, args.BenchModeDefault)...)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join("build", "wafbench_default.txt"), []byte(def), 0o644); err != nil {
		return err
	}

	return sh.RunV("go", "run", fmt.Sprintf("golang.org/x/perf/cmd/benchstat@%s", versions.GolangPerf),
		"build/wafbench_default.txt", "build/wafbench.txt", "build/wafbench_cgo.txt")
}
