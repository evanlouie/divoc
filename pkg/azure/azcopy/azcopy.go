package azcopy

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io/ioutil"
	"microsoft.com/divoc/pkg/azure/auth"
	"microsoft.com/divoc/pkg/logger"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

type Context struct {
	BinPath     string // path to the azcopy binary on host
	tempInstall bool   // if azcopy was install by this cli -- signals Clean() to remove it
}

// install azcopy to a temporary directory in os.TempDir
func install() (context Context, err error) {
	// Determine host OS
	var azcopyOS string
	var azcopyBinName string
	switch hostOS := runtime.GOOS; hostOS {
	case "darwin":
		logger.Debug("MacOS detected")
		azcopyOS = "mac"
		azcopyBinName = "azcopy"
	case "linux":
		logger.Debug("Linux detected")
		azcopyOS = "linux"
		azcopyBinName = "azcopy"
	case "windows":
		logger.Debug("Windows detected")
		azcopyOS = "windows"
		azcopyBinName = "azcopy.exe"
	default:
		return context, fmt.Errorf("unsupported OS: %s", hostOS)
	}

	////////////////////////////////////////////////////////////////////////////////
	// Download
	////////////////////////////////////////////////////////////////////////////////
	// Download latest azcopy v10
	downloadURL := fmt.Sprintf("https://aka.ms/downloadazcopy-v10-%s", azcopyOS)
	logger.Infof("Downloading AzCopy from: %s", downloadURL)
	resp, err := http.Get(downloadURL)
	if err != nil {
		return context, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return context, err
	}

	////////////////////////////////////////////////////////////////////////////////
	// Unzip to temporary directory
	////////////////////////////////////////////////////////////////////////////////
	// Unzip the body in memory
	zipReader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return context, fmt.Errorf("failed to create ZipReader: %s", err)
	}

	// Create temp directory to store azcopy
	azcopyTempDir, err := ioutil.TempDir("", "azcopy")
	if err != nil {
		return context, err
	}

	// Unzip everything flatly to temporary directory - ignore folders
	for _, zipFile := range zipReader.File {
		// skip directories
		if zipFile.FileInfo().IsDir() {
			continue
		}

		// target is only file Base name -- ignore parent directories
		targetPath := path.Join(azcopyTempDir, filepath.Base(zipFile.Name))
		logger.Infof("Decompressing %s to %s", zipFile.Name, targetPath)
		fileBytes, err := readZipFile(zipFile)
		if err != nil {
			return context, err
		}
		if err := ioutil.WriteFile(targetPath, fileBytes, 0777); err != nil {
			return context, err
		}
	}

	// Set BinPath to the temporary binary
	azcopyBinPath := path.Join(azcopyTempDir, azcopyBinName)
	if _, err := os.Stat(azcopyBinPath); os.IsNotExist(err) {
		return context, err
	}
	context.BinPath = azcopyBinPath
	context.tempInstall = true

	return context, nil
}

// Reads a zip.File and returns its []byte.
// Ensures the file is closed upon completion.
func readZipFile(zf *zip.File) ([]byte, error) {
	f, err := zf.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return ioutil.ReadAll(f)
}

// Login to azcopy via the provided service principal
func (ctx Context) Login(sp auth.ServicePrincipal) error {
	cmd := exec.Command(ctx.BinPath,
		"login",
		"--service-principal",
		"--application-id", sp.ApplicationId,
		"--tenant-id", sp.Tenant)
	cmd.Env = append(cmd.Env, fmt.Sprintf("AZCOPY_SPA_CLIENT_SECRET=%s", sp.Password))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	home, err := os.UserHomeDir() // must run in user home because azcopy stores its login credentials in ~/.azcopy
	if err != nil {
		return err
	}
	cmd.Dir = home
	logger.Debug(fmt.Sprintf("Running: %v", cmd))
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

// Copy data from one location to another.
// Shells out to azcopy under the hood, so it supports any `from` and `to` that binary does.
// Must run Login() prior to usage unless host has already logged in by other means.
func (ctx Context) Copy(from string, to string) (err error) {
	// if `from` does not start with "http:" it is a filesystem path
	// convert `from` to absolute path if is a filesystem path
	if !strings.EqualFold(from[0:4], "http:") {
		absFrom, err := filepath.Abs(from)
		if err != nil {
			logger.Error(err)
			return fmt.Errorf("failed to calculate absolute path for: %s", from)
		}
		from = absFrom
	}

	cmd := exec.Command(ctx.BinPath, "copy", from, to, "--recursive")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	home, err := os.UserHomeDir() // must run in user home because azcopy stores its login credentials in ~/.azcopy
	if err != nil {
		return err
	}
	cmd.Dir = home
	logger.Debugf("Running: %v", cmd)
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

// Remove temporary azcopy installation if installed during init()
func (ctx Context) Cleanup() error {
	// Ensure that the bin to cleanup is in the temporary directory
	relPathToBinFromTemp, err := filepath.Rel(os.TempDir(), ctx.BinPath)
	if err != nil {
		return err
	}
	// Is in temp dir if rel path to os.TempDir doesn't start with ".."
	inTempDir := !strings.HasPrefix(relPathToBinFromTemp, "..")
	if ctx.tempInstall && inTempDir {
		azcopyDir := path.Dir(ctx.BinPath)
		logger.Infof("Cleaning temporary AzCopy installation at: %s", azcopyDir)
		return os.RemoveAll(azcopyDir)
	}

	return nil
}

// Install azcopy to a temporary directory if it is not found on the users PATH.
// Sets BinPath to the path to wherever the azcopy binary is (either in PATH or in os.TempDir).
// Sets inPath according to whether the file was found in user PATH -- effects Cleanup().
// Returns a usable Context to interact with azcopy.
func InstallIfNotPresentAndGetCtx() (ctx Context, err error) {
	// set BinPath if found on host
	found, err := exec.LookPath("azcopy")
	if err == nil {
		logger.Infof("AzCopy binary found on host PATH: %s", found)
		absFound, err := filepath.Abs(found)
		if err != nil {
			return ctx, err
		}
		ctx.BinPath = absFound
		return ctx, err
	} else {
		logger.Info("AzCopy binary not found on host PATH, installing to temporary directory...")
		return install()
	}
}
