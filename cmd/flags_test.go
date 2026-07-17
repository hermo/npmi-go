package cmd

import (
	"flag"
	"io"
	"os"
	"runtime"
	"testing"
)

func TestParseFlagsTarExtractThreadsFromEnv(t *testing.T) {
	resetFlagsForTest(t, []string{"npmi-go"})
	t.Setenv("NPMI_TAR_EXTRACT_THREADS", "7")

	options, err := ParseFlags()
	if err != nil {
		t.Fatalf("ParseFlags returned error: %v", err)
	}

	if options.TarExtractThreads != 7 {
		t.Fatalf("TarExtractThreads=%d, want=7", options.TarExtractThreads)
	}
}

func TestParseFlagsTarExtractThreadsFlagOverridesEnv(t *testing.T) {
	resetFlagsForTest(t, []string{"npmi-go", "-tar-extract-threads", "11"})
	t.Setenv("NPMI_TAR_EXTRACT_THREADS", "7")

	options, err := ParseFlags()
	if err != nil {
		t.Fatalf("ParseFlags returned error: %v", err)
	}

	if options.TarExtractThreads != 11 {
		t.Fatalf("TarExtractThreads=%d, want=11", options.TarExtractThreads)
	}
}

func TestParseFlagsTarExtractThreadsDefault(t *testing.T) {
	resetFlagsForTest(t, []string{"npmi-go"})
	t.Setenv("NPMI_TAR_EXTRACT_THREADS", "")

	options, err := ParseFlags()
	if err != nil {
		t.Fatalf("ParseFlags returned error: %v", err)
	}

	want := runtime.GOMAXPROCS(0)
	if options.TarExtractThreads != want {
		t.Fatalf("TarExtractThreads=%d, want=%d", options.TarExtractThreads, want)
	}
}

func resetFlagsForTest(t *testing.T, args []string) {
	t.Helper()

	originalArgs := os.Args
	originalCommandLine := flag.CommandLine

	os.Args = args
	flag.CommandLine = flag.NewFlagSet(args[0], flag.ContinueOnError)
	flag.CommandLine.SetOutput(io.Discard)

	t.Cleanup(func() {
		os.Args = originalArgs
		flag.CommandLine = originalCommandLine
	})
}
