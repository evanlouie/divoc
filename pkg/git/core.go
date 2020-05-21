package git

import (
	"fmt"
	"microsoft.com/divoc/pkg/dependency"
	"microsoft.com/divoc/pkg/logger"
	"os"
	"os/exec"
)

type CloneOptions struct {
	Repo  string
	Dir   string
	Depth int
}

// Clone a git repository based on the provided CloneOptions
func Clone(options CloneOptions) (err error) {
	// prepare clone command
	cloneCmd := exec.Command("git", "clone", options.Repo)
	if options.Dir != "" {
		cloneCmd.Args = append(cloneCmd.Args, options.Dir)
	}
	if options.Depth > 0 {
		cloneCmd.Args = append(cloneCmd.Args, "--depth", fmt.Sprintf("%d", options.Depth))
	}

	// Execute command
	cloneCmd.Stdout = os.Stdout
	cloneCmd.Stderr = os.Stderr
	err = cloneCmd.Run()
	if err != nil {
		logger.Error(err)
		return fmt.Errorf("failed cloning git repository %s to directory %s", options.Repo, options.Dir)
	}

	return err
}

func init() {
	// Check for host dependencies
	if errs := dependency.IsInstalled("git"); errs != nil {
		logger.Fatal(errs)
	}
}
