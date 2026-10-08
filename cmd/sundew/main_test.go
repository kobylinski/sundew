package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestVersionBinary(t *testing.T) {
	for _, tc := range []struct {
		name, ldflags, want string
	}{
		{"default", "", "dev\n"},
		{"release", "-X main.version=0.0.0-test", "0.0.0-test\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			binary := filepath.Join(t.TempDir(), "sundew")
			build := exec.Command("go", "build", "-ldflags", tc.ldflags, "-o", binary, ".")
			if output, err := build.CombinedOutput(); err != nil {
				t.Fatalf("build: %v\n%s", err, output)
			}
			cmd := exec.Command(binary, "--version")
			// --version must work without loading config or opening a listener.
			for _, value := range os.Environ() {
				if !strings.HasPrefix(value, "SUNDEW_ADDR=") {
					cmd.Env = append(cmd.Env, value)
				}
			}
			cmd.Env = append(cmd.Env, "SUNDEW_ADDR=invalid")
			output, err := cmd.CombinedOutput()
			if err != nil || string(output) != tc.want {
				t.Fatalf("--version = %q, %v; want %q, nil", output, err, tc.want)
			}
		})
	}
}
