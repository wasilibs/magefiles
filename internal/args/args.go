package args

import (
	"fmt"

	"github.com/wasilibs/magefiles/internal/meta"
)

type BenchMode int

const (
	BenchModeWazero BenchMode = iota
	BenchModeCGO
	BenchModeDefault
)

func BenchArgs(pkg string, count int, mode BenchMode) []string {
	args := []string{"test", "-bench=.", "-run=^$", "-v", "-timeout=60m"}
	if count > 0 {
		args = append(args, fmt.Sprintf("-count=%d", count))
	}
	switch mode {
	case BenchModeCGO:
		args = append(args, fmt.Sprintf("-tags=%s_cgo", meta.LibraryName))
	case BenchModeDefault:
		args = append(args, fmt.Sprintf("-tags=%s_bench_default", meta.LibraryName))
	}
	args = append(args, pkg)

	return args
}
