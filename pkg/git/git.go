package git

import (
	"divoc/pkg/logger"
	"fmt"
	"os"
	"os/exec"
)

type CloneOptions struct {
	Repo  string
	Dir   string
	Depth int
}

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
