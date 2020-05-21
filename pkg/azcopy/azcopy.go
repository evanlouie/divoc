package azcopy

import (
	"divoc/pkg/logger"
	"fmt"
	"os"
	"os/exec"
)

type ServicePrincipal struct {
	ApplicationId string
	Password      string
	Tenant        string
	//DisplayName   string
	//Name          string
}

// Login to azcopy via the provided service principal
func Login(principal ServicePrincipal) error {
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

// Copy data from one place to another
func Copy(from string, to string) error {
	cmd := exec.Command("azcopy", "copy", from, to, "--recursive")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}
