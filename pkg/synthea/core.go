package synthea

import (
	"divoc/pkg/dependency"
	"divoc/pkg/git"
	"divoc/pkg/logger"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"runtime"
)

type syntheaState struct {
	installPath string
}

// Package state variable to hold runtime state
var state = syntheaState{}

// Clone the Synthea repository locally to a temporary directory
func Clone() (err error) {
	// Clone Synthea into a temp dir
	tempDir, err := ioutil.TempDir("", "synthea")
	if err != nil {
		return err
	}
	state.installPath = tempDir

	// Execute cloning
	logger.Info(fmt.Sprintf("Cloning Synthea repository to %s", tempDir))
	return git.Clone(git.CloneOptions{
		Repo:  "https://github.com/synthetichealth/synthea",
		Dir:   tempDir,
		Depth: 1, // shallow clone -- repository is large
	})
}

// Clean the temporary clone directory created from Clone()
func Clean() (err error) {
	logger.Info(fmt.Sprintf("Cleaning temporary directory %s", state.installPath))
	return os.RemoveAll(state.installPath)
}

type Options map[string]string

// SetOptions appends new config options to `src/main/resources/synthea.properties` file in the
// cloned repository created from Clone()
func SetOptions(options Options) error {
	// Append to src/main/resources/synthea.properties to set feature flags
	var propertiesPath = path.Join(state.installPath, "src", "main", "resources", "synthea.properties")
	pFile, err := os.OpenFile(propertiesPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer pFile.Close()

	// append each option to the end of the properties file
	for property, value := range options {
		logger.Info(fmt.Sprintf("Setting Synthea %s => %s", property, value))
		if _, err := pFile.WriteString(fmt.Sprintf("%s = %s\n", property, value)); err != nil {
			logger.Error(err)
			return fmt.Errorf("failed to update Synthea properties file %s", propertiesPath)
		}
	}
	return nil
}

type CliArgs struct {
	Seed           int
	PopulationSize int
	ModuleFilter   string
	State          string
	City           string
}

// Run the Synthea in a child process.
// Uses `sh` on all windows hosts.
// Uses `cmd.exe` on windows.
func Run(args CliArgs) error {
	if state.installPath == "" {
		return fmt.Errorf("zero-length path to synthea set")
	}

	// use `sh` as shell script executor be default, use `cmd.exe` if host is windows
	executor := "sh"
	if runtime.GOOS == "windows" {
		executor = "cmd.exe"
	}

	var cmd = exec.Command(executor, "run_synthea", "-p", "128", "California", "San Francisco")
	if args.Seed != 0 {
		cmd.Args = append(cmd.Args, "-s", fmt.Sprintf("%d", args.Seed))
	}
	if args.PopulationSize != 0 {
		cmd.Args = append(cmd.Args, "-p", fmt.Sprintf("%d", args.PopulationSize))
	}
	if args.ModuleFilter != "" {
		cmd.Args = append(cmd.Args, "-m", args.ModuleFilter)
	}
	cmd.Args = append(cmd.Args, args.State, args.City)
	cmd.Dir = state.installPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// GetInstallPath returns the temporary directory Synthea was cloned to.
// Returns an error if the the installPath has not been set in package state.
func GetInstallPath() (string, error) {
	if state.installPath == "" {
		return "", errors.New("Synthea not installed -- must run synthea.Clone()")
	}
	return state.installPath, nil
}

func init() {
	// Check for host dependencies
	if errs := dependency.IsInstalled("java"); errs != nil {
		logger.Fatal(errs)
	}
}
