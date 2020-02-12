package yarn

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/packit/fs"
	"github.com/cloudfoundry/packit/pexec"
)

//go:generate faux --interface Summer --output fakes/summer.go
type Summer interface {
	Sum(path string) (string, error)
}

type YarnInstallProcess struct {
	executable Executable
	summer     Summer
}

func NewYarnInstallProcess(executable Executable, summer Summer) YarnInstallProcess {
	return YarnInstallProcess{
		executable: executable,
		summer:     summer,
	}
}

//go:generate faux --interface Executable --output fakes/executable.go
type Executable interface {
	Execute(pexec.Execution) (stdout, stderr string, err error)
}

func (ip YarnInstallProcess) ShouldRun(workingDir string, metadata map[string]interface{}) (run bool, sha string, err error) {
	_, err = os.Stat(filepath.Join(workingDir, "yarn.lock"))
	if os.IsNotExist(err) {
		return true, "", nil
	} else if err != nil {
		return true, "", fmt.Errorf("unable to read yarn.lock file: %w", err)
	}

	sum, err := ip.summer.Sum(filepath.Join(workingDir, "yarn.lock"))
	if err != nil {
		return true, "", fmt.Errorf("unable to sum yarn.lock file: %w", err)
	}

	prevSHA, ok := metadata["cache_sha"].(string)
	if (ok && sum != prevSHA) || !ok {
		return true, sum, nil
	}

	return false, "", nil
}

func (ip YarnInstallProcess) Execute(workingDir, layerPath string) error {
	err := os.MkdirAll(filepath.Join(workingDir, "node_modules"), os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create node_modules directory: %w", err)
	}

	err = fs.Move(filepath.Join(workingDir, "node_modules"), filepath.Join(layerPath, "node_modules"))
	if err != nil {
		return fmt.Errorf("failed to move node_modules directory to layer: %w", err)
	}

	err = os.Symlink(filepath.Join(layerPath, "node_modules"), filepath.Join(workingDir, "node_modules"))
	if err != nil {
		return fmt.Errorf("failed to symlink node_modules into working directory: %w", err)
	}

	stdout := bytes.NewBuffer(nil)
	_, _, err = ip.executable.Execute(pexec.Execution{
		Args:   []string{"config", "get", "yarn-offline-mirror"},
		Stdout: stdout,
	})
	if err != nil {
		return fmt.Errorf("failed to execute yarn config: %w", err)
	}

	installArgs := []string{"install", "--pure-lockfile", "--ignore-engines"}

	offlineMirrorPath := strings.TrimSpace(stdout.String())

	fileInfo, err := os.Stat(offlineMirrorPath)
	if err == nil {
		if fileInfo.IsDir() {
			installArgs = append(installArgs, "--offline")
		}
	} else {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("failed to confirm existence of offline mirror directory: %w", err)
		}
	}

	_, _, err = ip.executable.Execute(pexec.Execution{
		Args: append(installArgs, "--modules-folder", filepath.Join(layerPath, "node_modules")),
	})
	if err != nil {
		return fmt.Errorf("failed to execute yarn install: %w", err)
	}

	return nil
}
