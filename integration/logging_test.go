package integration_test

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/paketo-buildpacks/occam"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
	. "github.com/paketo-buildpacks/occam/matchers"
)

func testLogging(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		pack   occam.Pack
		docker occam.Docker
	)

	it.Before(func() {
		pack = occam.NewPack()
		docker = occam.NewDocker()
	})

	context("when app is NOT vendored", func() {
		var (
			image occam.Image

			name string
		)

		it.Before(func() {
			var err error
			name, err = occam.RandomName()
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
			Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
		})

		it("should build a working OCI image for a simple app", func() {
			var err error
			var logs fmt.Stringer
			image, logs, err = pack.WithNoColor().Build.
				WithBuildpacks(nodeURI, yarnURI).
				WithNoPull().
				Execute(name, filepath.Join("testdata", "simple_app"))
			Expect(err).NotTo(HaveOccurred())

			buildpackVersion, err := GetGitVersion()
			Expect(err).ToNot(HaveOccurred())

			Expect(logs).To(ContainLines(
				fmt.Sprintf("%s %s", buildpackInfo.Buildpack.Name, buildpackVersion),
				"  Executing build process",
				MatchRegexp(`    Installing Yarn 1\.\d+\.\d+`),
				MatchRegexp(`      Completed in (\d+)(\.\d+)?(ms|s)`),
				"",
				"  Resolving installation process",
				"    Process inputs:",
				"      yarn.lock -> Found",
				"",
				"    Selected default build process: 'yarn install'",
				"",
				"  Executing build process",
				fmt.Sprintf("    Running yarn install --ignore-engines --frozen-lockfile --modules-folder /layers/%s/modules/node_modules", strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_")),
				MatchRegexp(`      Completed in (\d+)(\.\d+)?(ms|s)`),
				"",
				"  Configuring environment",
				fmt.Sprintf(`    PATH -> "$PATH:/layers/%s/modules/node_modules/.bin"`, strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_")),
			))
		})
	})

	context("when the app is vendored", func() {
		var (
			image occam.Image

			name string
		)

		it.Before(func() {
			var err error
			name, err = occam.RandomName()
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
			Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
		})

		it("should build a working OCI image for a simple app", func() {
			var err error
			var logs fmt.Stringer
			image, logs, err = pack.WithNoColor().Build.
				WithBuildpacks(nodeURI, yarnURI).
				WithNoPull().
				Execute(name, filepath.Join("testdata", "vendored"))
			Expect(err).NotTo(HaveOccurred())

			buildpackVersion, err := GetGitVersion()
			Expect(err).ToNot(HaveOccurred())

			Expect(logs).To(ContainLines(
				fmt.Sprintf("%s %s", buildpackInfo.Buildpack.Name, buildpackVersion),
				"  Executing build process",
				MatchRegexp(`    Installing Yarn 1\.\d+\.\d+`),
				MatchRegexp(`      Completed in (\d+)(\.\d+)?(ms|s)`),
				"",
				"  Resolving installation process",
				"    Process inputs:",
				"      yarn.lock -> Found",
				"",
				"    Selected default build process: 'yarn install'",
				"",
				"  Executing build process",
				fmt.Sprintf("    Running yarn install --ignore-engines --frozen-lockfile --offline --modules-folder /layers/%s/modules/node_modules", strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_")),
				MatchRegexp(`      Completed in (\d+)(\.\d+)?(ms|s)`),
				"",
				"  Configuring environment",
				fmt.Sprintf(`    PATH -> "$PATH:/layers/%s/modules/node_modules/.bin"`, strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_")),
			))
		})
	})
}
