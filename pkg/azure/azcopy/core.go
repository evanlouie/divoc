package azcopy

import (
	"fmt"
	"microsoft.com/divoc/pkg/azure/auth"
	"microsoft.com/divoc/pkg/dependency"
	"microsoft.com/divoc/pkg/logger"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Login to azcopy via the provided service principal
func Login(principal auth.ServicePrincipal) error {
	cmd := exec.Command("azcopy",
		"login",
		"--service-principal",
		"--application-id", principal.ApplicationId,
		"--tenant-id", principal.Tenant)
	cmd.Env = append(cmd.Env, fmt.Sprintf("AZCOPY_SPA_CLIENT_SECRET=%s", principal.Password))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	home, _ := os.UserHomeDir() // must run in ~/ because azcopy stores its login credentials in ~/.azcopy
	cmd.Dir = home
	err := cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

// Copy data from one location to another.
// Shells out to azcopy under the hood, so it supports any `from` and `to` that binary does.
// Must run Login() prior to usage unless host has already logged in by other means.
func Copy(from string, to string) error {
	// convert `from` to absolute path if is a filesystem path
	if !(strings.HasPrefix(from, "http") || strings.HasPrefix(from, "HTTP")) {
		from, err := filepath.Abs(from)
		if err != nil {
			logger.Error(err)
			return fmt.Errorf("failed to calculate absolute path for: %s", from)
		}
	}

	cmd := exec.Command("azcopy", "copy", from, to, "--recursive")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	home, _ := os.UserHomeDir() // must run in ~/ because azcopy stores its login credentials in ~/.azcopy
	cmd.Dir = home
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func init() {
	// Check for host dependencies
	if errs := dependency.IsInstalled("git"); errs != nil {
		logger.Fatal(errs)
	}
}
