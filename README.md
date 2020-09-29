# Yarn Install Cloud Native Buildpack

The Yarn Install CNB generates and provides application dependencies for node
applications that use the [yarn](https://yarnpkg.com) package manager.

## Integration

The Yarn Install CNB provides `node_modules` as a dependency. Downstream
buildpacks can require the `node_modules` dependency by generating a [Build
Plan TOML](https://github.com/buildpacks/spec/blob/master/buildpack.md#build-plan-toml)
file that looks like the following:

```toml
[[requires]]

  # The name of the Yarn Install dependency is "node_modules". This value is
  # considered # part of the public API for the buildpack and will not change
  # without a plan # for deprecation.
  name = "node_modules"

  # Note: The version field is unsupported as there is no version for a set of
  # node_modules.

  # The Yarn Install buildpack supports some non-required metadata options.
  [requires.metadata]

    # Setting the build flag to true will ensure that the node modules
    # are available for subsequent buildpacks during their build phase.
    # If you are writing a buildpack that needs a node module during
    # its build process, this flag should be set to true.
    build = true

    # Setting the launch flag to true will ensure that the packages
    # managed by Yarn are available for the running application. If you
    # are writing an application that needs node modules at runtime,
    # this flag should be set to true.
    launch = true

```

## Usage

To package this buildpack for consumption:

```
$ ./scripts/package.sh --version <version-number>
```

This will create a `buildpackage.cnb` file under the `build` directory which you
can use to build your app as follows:
```
pack build <app-name> -p <path-to-app> -b <path/to/node-engine.cnb> -b <path/to/yarn.cnb> /
-b build/buildpackage.cnb
```

## `buildpack.yml` Configurations

The `yarn-install` buildpack does not support configurations using `buildpack.yml`.

## Run Tests

To run all unit tests, run:
```
./scripts/unit.sh
```

To run all integration tests, run:
```
/scripts/integration.sh
```
