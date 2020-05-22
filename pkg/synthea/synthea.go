package synthea

import (
	"errors"
	"fmt"
	"io/ioutil"
	"microsoft.com/divoc/pkg/git"
	"microsoft.com/divoc/pkg/logger"
	"os"
	"os/exec"
	"path"
)

// Path to cloned synthea repo on host
// if nil, it has not been cloned or set by user
var InstallPath *string

// Clone the Synthea repository locally to a temporary directory
func Clone() (err error) {
	// Clone Synthea into a temp dir
	tempDir, err := ioutil.TempDir("", "synthea")
	if err != nil {
		return err
	}
	InstallPath = &tempDir

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
	logger.Info(fmt.Sprintf("Cleaning temporary Synthea directory %s", *InstallPath))
	return os.RemoveAll(*InstallPath)
}

type Options map[string]string

// SetOptions appends new config options to `src/main/resources/synthea.properties` file in the
// cloned repository created from Clone()
func SetOptions(options Options) error {
	// Append to src/main/resources/synthea.properties to set feature flags
	var propertiesPath = path.Join(*InstallPath, "src", "main", "resources", "synthea.properties")
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

// Run the run_synthea script in a child process.
func Run(args CliArgs) error {
	if *InstallPath == "" {
		return fmt.Errorf("zero-length path to synthea set")
	}
	// Ensure Java is on user PATH
	if _, err := exec.LookPath("java"); err != nil {
		return err
	}

	var cmdArgs []string
	if args.Seed != 0 {
		cmdArgs = append(cmdArgs, "-s", fmt.Sprintf("%d", args.Seed))
	}
	if args.PopulationSize != 0 {
		cmdArgs = append(cmdArgs, "-p", fmt.Sprintf("%d", args.PopulationSize))
	}
	if args.ModuleFilter != "" {
		cmdArgs = append(cmdArgs, "-m", args.ModuleFilter)
	}
	cmdArgs = append(cmdArgs, args.State, args.City)
	syntheaBin := path.Join(*InstallPath, "run_synthea")

	cmd := exec.Command(syntheaBin, cmdArgs...)
	cmd.Dir = *InstallPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	logger.Debug(fmt.Sprintf("Running: %v", cmd))
	return cmd.Run()
}

// GetInstallPath returns the temporary directory Synthea was cloned to.
// Returns an error if the the InstallPath has not been set in package state.
func GetInstallPath() (string, error) {
	if InstallPath == nil {
		return "", errors.New("synthea not installed -- must run synthea.Clone()")
	}
	return *InstallPath, nil
}
