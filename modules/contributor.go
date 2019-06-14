package modules

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/cloudfoundry/libcfbuildpack/build"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"github.com/cloudfoundry/libcfbuildpack/layers"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"path/filepath"
)

const (
	Dependency  = "modules"
	Dir         = "node_modules"
	DirMetadata = "Node Dependencies"
	cacheLayer  = "modules_cache"
	lockFile    = "yarn.lock"
)

var (
	cacheDir = filepath.Join(".cache", "yarn")
)

type PackageManager interface {
	Install(modulesDir, cacheDir string) error
	Check(appDir string) error
	SetConfig(location, key, value string) error
}

type Metadata struct {
	Name string
	Hash string
}

func (m Metadata) Identity() (name string, version string) {
	return m.Name, m.Hash
}

type Contributor struct {
	context            build.Build
	metadata           Metadata
	buildContribution  bool
	launchContribution bool
	pkgManager         PackageManager
	modulesLayer       layers.Layer
	cacheLayer         layers.Layer
}

func NewContributor(context build.Build, pkgManager PackageManager) (Contributor, bool, error) {
	plan, shouldInstallModules := context.BuildPlan[Dependency]
	if !shouldInstallModules {
		return Contributor{}, false, nil
	}

	if exists, err := helper.FileExists(filepath.Join(context.Application.Root, lockFile)); err != nil {
		return Contributor{}, true, nil
	} else if !exists {
		return Contributor{}, false, errors.New("yarn.lock not found")
	}

	contributor := Contributor{
		context:      context,
		pkgManager:   pkgManager,
		modulesLayer: context.Layers.Layer(Dependency),
		cacheLayer:   context.Layers.Layer(cacheLayer),
	}

	if _, ok := plan.Metadata["build"]; ok {
		contributor.buildContribution = true
	}

	if _, ok := plan.Metadata["launch"]; ok {
		contributor.launchContribution = true
	}

	return contributor, true, nil
}

func (c *Contributor) Contribute() error {
	if err := c.setMetadata(); err != nil {
		return err
	}

	if err := c.modulesLayer.Contribute(c.metadata, c.contributeNodeModules, c.flags()...); err != nil {
		return err
	}

	return c.context.Layers.WriteApplicationMetadata(layers.Metadata{
		Processes: []layers.Process{{"web", "yarn start"}},
	})
}

func (c *Contributor) contributeNodeModules(layer layers.Layer) error {
	appModulesDir := filepath.Join(c.context.Application.Root, Dir)
	layerModulesDir := filepath.Join(c.modulesLayer.Root, Dir)
	layerCacheDir := filepath.Join(c.cacheLayer.Root, cacheDir)

	if vendored, err := helper.FileExists(appModulesDir); err != nil {
		return err
	} else if vendored {
		if err := moveNodeModulesToLayer(appModulesDir, layerModulesDir); err != nil {
			return err
		}
	}

	if err := c.enableCaching(); err != nil {
		return errors.Wrap(err, "failed to enable caching")
	}

	if err := c.pkgManager.Install(layerModulesDir, layerCacheDir); err != nil {
		return fmt.Errorf("failed to install node_modules: %s", err.Error())
	}

	if err := c.pkgManager.Check(c.context.Application.Root); err != nil {
		return fmt.Errorf("failed to yarn check installed modules %s", err.Error())
	}

	if err := os.Symlink(layerModulesDir, appModulesDir); err != nil {
		return err
	}

	if err := layer.OverrideSharedEnv("NODE_PATH", layerModulesDir); err != nil {
		return err
	}

	if err := layer.AppendPathSharedEnv("PATH", filepath.Join(layerModulesDir, ".bin")); err != nil {
		return err
	}

	return layer.OverrideSharedEnv("npm_config_nodedir", os.Getenv("NODE_HOME"))
}

func (c *Contributor) enableCaching() error {
	appCacheDir := filepath.Join(c.context.Application.Root, cacheDir)
	layerCacheDir := filepath.Join(c.cacheLayer.Root, cacheDir)

	c.cacheLayer.Touch()

	if err := os.MkdirAll(layerCacheDir, 0777); err != nil {
		return err
	}
	if exists, err := helper.FileExists(appCacheDir); err != nil {
		return err
	} else if exists {
		return helper.CopyDirectory(appCacheDir, layerCacheDir)
	}

	return c.cacheLayer.WriteMetadata(Metadata{Name: "Modules Cache"}, layers.Cache)
}

func (c *Contributor) setMetadata() error {
	lockFile := filepath.Join(c.context.Application.Root, lockFile)
	if exists, err := helper.FileExists(lockFile); err != nil {
		return err
	} else if !exists {
		return errors.New(fmt.Sprintf("failed to find %s", lockFile))
	}

	f, err := ioutil.ReadFile(lockFile)
	if err != nil {
		return err
	}

	hash := sha256.Sum256(f)
	c.metadata = Metadata{DirMetadata, hex.EncodeToString(hash[:])}
	return nil
}

func (c *Contributor) flags() []layers.Flag {
	flags := []layers.Flag{layers.Cache}

	if c.buildContribution {
		flags = append(flags, layers.Build)
	}

	if c.launchContribution {
		flags = append(flags, layers.Launch)
	}

	return flags
}

func moveNodeModulesToLayer(nodeModules, modulesLayer string) error {
	nodeModulesExist, err := helper.FileExists(nodeModules)
	if err != nil {
		return fmt.Errorf("unable to stat node_modules: %s", err.Error())
	}

	if nodeModulesExist {
		if err := helper.CopyDirectory(nodeModules, modulesLayer); err != nil {
			return fmt.Errorf(`unable to copy "%s" to "%s": %s`, nodeModules, modulesLayer, err.Error())
		}

		if err := os.RemoveAll(nodeModules); err != nil {
			return fmt.Errorf("unable to remove node_modules from the app dir: %s", err.Error())
		}
	}

	return nil
}