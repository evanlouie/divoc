package dependency

import (
	"fmt"
	"microsoft.com/divoc/pkg/logger"
	"os/exec"
)

// IsInstalled will check to see if all provided binaries are found on the host PATH.
// If any are not, a single error is returned listing them.
// Missing dependencies are logged as errors as they are searched for.
func IsInstalled(bins ...string) error {
	var notInstalled []string
	for _, bin := range bins {
		_, err := exec.LookPath(bin)
		if err != nil {
			logger.Error(err)
			notInstalled = append(notInstalled, bin)
		}
	}
	if len(notInstalled) != 0 {
		return fmt.Errorf("missing host depenencies: %v", notInstalled)
	}
	return nil
}
