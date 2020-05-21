package azcopy

import (
	"divoc/pkg/azure/auth"
	"divoc/pkg/dependency"
	"divoc/pkg/logger"
	"fmt"
	"os"
	"os/exec"
)

// Login to azcopy via the provided service principal
func Login(principal auth.ServicePrincipal) error {
	cmd := exec.Command("azcopy",
		"login",
		"--service-principal",
		"--application-id", principal.ApplicationId,
		"--tenant-id", principal.Tenant)
	cmd.Env = append(cmd.Env, fmt.Sprintf("AZCOPY_SPA_CLIENT_SECRET=%s", principal.Password))
	out, err := cmd.CombinedOutput()
	logger.Info(string(out))
	if err != nil {
		return err
	}

	return nil
}

// Copy data from one location to another.
// Shells out to azcopy under the hood, so it supports any `from` and `to` that binary does.
// Must run Login() prior to usage unless host has already logged in by other means.
func Copy(from string, to string) error {
	cmd := exec.Command("azcopy", "copy", from, to, "--recursive")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
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
