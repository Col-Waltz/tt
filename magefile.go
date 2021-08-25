// +build mage

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/apex/log"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

const (
	goPackageName = "github.com/tarantool/tt/cli"

	asmflags = "all=-trimpath=${PWD}"
	gcflags  = "all=-trimpath=${PWD}"

	packagePath = "./cli"

	defaultLinuxConfigPath  = "/etc/tarantool"
	defaultDarwinConfigPath = "/usr/local/etc/tarantool"
)

var (
	ldflags = []string{
		"-s", "-w",
		"-X ${PACKAGE}/version.gitTag=${GIT_TAG}",
		"-X ${PACKAGE}/version.gitCommit=${GIT_COMMIT}",
		"-X ${PACKAGE}/version.versionLabel=${VERSION_LABEL}",
		"-X ${PACKAGE}/configure.defaultConfigPath=${CONFIG_PATH}",
	}

	goExecutableName     = "go"
	pythonExecutableName = "python3"
	ttExecutableName     = "tt"
)

func init() {
	var err error

	if specifiedGoExe := os.Getenv("GOEXE"); specifiedGoExe != "" {
		goExecutableName = specifiedGoExe
	}

	if specifiedTTExe := os.Getenv("TTEXE"); specifiedTTExe != "" {
		ttExecutableName = specifiedTTExe
	} else {
		if ttExecutableName, err = filepath.Abs(ttExecutableName); err != nil {
			panic(err)
		}
	}

	// We want to use Go 1.11 modules even if the source lives inside GOPATH.
	// The default is "auto".
	os.Setenv("GO111MODULE", "on")
}

// Building tt executable.
func Build() error {
	fmt.Println("Building tt...")

	err := sh.RunWith(
		getBuildEnvironment(), goExecutableName, "build",
		"-o", ttExecutableName,
		"-ldflags", strings.Join(ldflags, " "),
		"-asmflags", asmflags,
		"-gcflags", gcflags,
		packagePath,
	)

	if err != nil {
		return fmt.Errorf("Failed to build tt executable: %s", err)
	}

	return nil
}

// Run golang and python linters.
func Lint() error {
	fmt.Println("Running go vet...")

	if err := sh.RunV(goExecutableName, "vet", packagePath); err != nil {
		return err
	}

	fmt.Println("Running flake8...")

	if err := sh.RunV(pythonExecutableName, "-m", "flake8", "test"); err != nil {
		return err
	}

	return nil
}

// Run unit tests.
func Unit() error {
	fmt.Println("Running unit tests...")

	if mg.Verbose() {
		return sh.RunV(goExecutableName, "test", "-v", fmt.Sprintf("%s/...", packagePath))
	}

	return sh.RunV(goExecutableName, "test", fmt.Sprintf("%s/...", packagePath))
}

// Run integration tests.
func Integration() error {
	fmt.Println("Running integration tests...")

	return sh.RunV(pythonExecutableName, "-m", "pytest", "test/integration")
}

// Run all tests together.
func Test() {
	mg.SerialDeps(Lint, Unit, Integration)
}

// Cleanup directory.
func Clean() {
	fmt.Println("Cleaning directory...")

	os.Remove(ttExecutableName)
}

// getDefaultConfigPath returns the path to the configuration file,
// determining it based on the OS.
func getDefaultConfigPath() string {
	switch runtime.GOOS {
	case "linux":
		return defaultLinuxConfigPath
	case "darwin":
		return defaultDarwinConfigPath

	}

	log.Fatalf("Trying to get default config path file on an unsupported OS")
	return ""
}

// getBuildEnvironment return map with build environment variables.
func getBuildEnvironment() map[string]string {
	var err error

	var currentDir string
	var gitTag string
	var gitCommit string

	if currentDir, err = os.Getwd(); err != nil {
		log.Warnf("Failed to get current directory: %s", err)
	}

	if _, err := exec.LookPath("git"); err == nil {
		gitTag, _ = sh.Output("git", "describe", "--tags")
		gitCommit, _ = sh.Output("git", "rev-parse", "--short", "HEAD")
	}

	return map[string]string{
		"PACKAGE":       goPackageName,
		"GIT_TAG":       gitTag,
		"GIT_COMMIT":    gitCommit,
		"VERSION_LABEL": os.Getenv("VERSION_LABEL"),
		"PWD":           currentDir,
		"CONFIG_PATH":   getDefaultConfigPath(),
	}
}
