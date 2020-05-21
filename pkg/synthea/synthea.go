package synthea

import (
	"divoc/pkg/git"
	"divoc/pkg/logger"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
)

type syntheaState struct {
	installPath string
}

var state = syntheaState{}

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

func Clean() (err error) {
	logger.Info(fmt.Sprintf("Cleaning temporary directory %s", state.installPath))
	return os.RemoveAll(state.installPath)
}

type Options map[string]string

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

func Run(args CliArgs) error {
	if state.installPath == "" {
		return fmt.Errorf("zero-length path to synthea set")
	}

	var cmd = exec.Command("sh", "run_synthea", "-p", "128", "California", "San Francisco")
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

func GetInstallPath() (string, error) {
	if state.installPath == "" {
		return "", errors.New("Synthea not installed -- must run synthea.Clone()")
	}
	return state.installPath, nil
}
