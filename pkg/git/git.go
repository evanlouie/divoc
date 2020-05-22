package git

import (
	"fmt"
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
	// Check that git is on user PATH
	if _, err := exec.LookPath("git"); err != nil {
		return err
	}

	// prepare clone command
	cloneCmd := exec.Command("git", "clone", options.Repo)
	if options.Dir != "" {
		cloneCmd.Args = append(cloneCmd.Args, options.Dir)
	}
	if options.Depth != 0 {
		cloneCmd.Args = append(cloneCmd.Args, "--depth", fmt.Sprintf("%d", options.Depth))
	}

	// Execute command
	cloneCmd.Stdout = os.Stdout
	cloneCmd.Stderr = os.Stderr
	logger.Debug(fmt.Sprintf("Running: %v", cloneCmd))
	if err := cloneCmd.Run(); err != nil {
		logger.Error(err)
		return fmt.Errorf("failed cloning git repository %s to directory %s", options.Repo, options.Dir)
	}

	return err
}
