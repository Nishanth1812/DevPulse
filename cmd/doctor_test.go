package cmd

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Nishanth1812/devpulse/internal/config"
)

func writeDoctorConfig(t *testing.T, registered string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.toml")
	content := ""
	if registered != "" {
		content = "[registered_repos]\nmissing = \"" + filepath.ToSlash(registered) + "\"\n"
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write doctor config: %v", err)
	}
	t.Setenv(config.ConfigEnv, path)
	loaded, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	manager = loaded
	noColor = true
	t.Cleanup(func() {
		manager = nil
		noColor = false
	})
	return path
}

func TestRunDoctorReturnsFailureForMissingRepository(t *testing.T) {
	writeDoctorConfig(t, filepath.Join(t.TempDir(), "deleted-repository"))
	var output bytes.Buffer
	doctorCmd.SetOut(&output)
	t.Cleanup(func() { doctorCmd.SetOut(nil) })

	err := runDoctor(doctorCmd, nil)
	var failure doctorFailure
	if !errors.As(err, &failure) {
		t.Fatalf("runDoctor error = %v, want doctorFailure", err)
	}
	if failure.FailCount != 1 {
		t.Fatalf("failure count = %d, want 1", failure.FailCount)
	}
	if !strings.Contains(output.String(), "[FAIL]") {
		t.Fatalf("doctor output missing failure marker:\n%s", output.String())
	}
}

func TestRunDoctorWarningOnlyReturnsSuccess(t *testing.T) {
	writeDoctorConfig(t, "")
	var output bytes.Buffer
	doctorCmd.SetOut(&output)
	t.Cleanup(func() { doctorCmd.SetOut(nil) })

	if err := runDoctor(doctorCmd, nil); err != nil {
		t.Fatalf("warning-only doctor returned error: %v\n%s", err, output.String())
	}
	if strings.Contains(output.String(), "[FAIL]") {
		t.Fatalf("warning-only doctor output contains failure:\n%s", output.String())
	}
}

func TestDoctorSubprocessExit(t *testing.T) {
	if os.Getenv("DEVPULSE_DOCTOR_HELPER") == "1" {
		rootCmd.SetArgs([]string{"doctor", "--no-color"})
		Execute()
		return
	}

	configPath := filepath.Join(t.TempDir(), "config.toml")
	missing := filepath.Join(t.TempDir(), "missing-repository")
	if err := os.WriteFile(configPath, []byte("[registered_repos]\nmissing = \""+filepath.ToSlash(missing)+"\"\n"), 0o600); err != nil {
		t.Fatalf("write subprocess config: %v", err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestDoctorSubprocessExit", "-test.v")
	cmd.Env = append(os.Environ(), config.ConfigEnv+"="+configPath, "DEVPULSE_DOCTOR_HELPER=1")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("doctor subprocess unexpectedly succeeded:\n%s", output)
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
		t.Fatalf("doctor subprocess error = %v, want exit code 1\n%s", err, output)
	}
	if !bytes.Contains(output, []byte("[FAIL]")) {
		t.Fatalf("doctor subprocess output missing failure:\n%s", output)
	}
}
