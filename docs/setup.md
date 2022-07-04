# Tool Binary Distribution setup

## Initialisation

**Typically any initialisation is done automatically and silently when `toolshare` is invoked for the first time on a
new system**.

This mainly involves setting up the folder and file structure that is used for binary caching.  Alternatively it is
possible to invoke `toolshare init` to run or rerun the initialisation explicitly. This might help recover a defectuous
setup if manual changes have been made to the files that `toolshare` uses.

When used as `toolshare init --force` it results in a clean-slate setup and the deletion of any pre-existing files and
folders. A manual confirmation will be required as this might result in the deletion of a large amount of cached assets,
or in the case of a script one can pass the `--assume-yes` flag to bypass the confirmation.

## Environments

A `toolshare` environment designates a folder-tree where the root contains a `.toolshare` configuration file. Typically
an environment will correspond to a single VCS repository, but this is not a requirement. The `.toolshare` file contains
the list of all the tools for which `toolshare` should manage the version when invoked from a path within the
environment.

Just like folder-trees can be nested, environments can be too. As a general rule of thumb the configurations of nested
environments are merged, with _the innermost taking piority_, in order to determine what action `toolshare` should take
when invoking a tool from a given path.

Optionally one can also configure a special system-wide environment that will be always be used. When merging this
environment is considered as the outermost one, even when a `.toolshare` file exists at the root of the current filesystem.

## Configuration

The exact content of a `.toolshare` file will depend on whether you are in a _stateless_ or _stateful_ setup, with
**_stateless_ by far being the most common and default situation**. You can read more about _stateless_ vs. _stateful_
setups in the [dedicated documentation](./internals/state.md).

### Stateless-mode

In _stateless_ mode, to configure a tool for use with `toolshare`, two elements are required:

* A version pin, meaning the version that should be used when invoking the tool
* A source for the tool's binaries

Example:

```yaml
pins:
  kubectl: "1.20.1"

sources:
  kubectl:
    type: github
    slug: kubernetes/kubectl
    assetTemplate: kubectl-{version}-{os}-{arch}.tar.gz
    archivePathTemplate: ./kubectl{exe}
```

Binaries, when not yet locally cached nor available in any configured remote storage, need to be fetched from a source.
`toolshare` supports multiple sources types.

#### GitHub

#### GCS or S3 cloud storage buckets

Because the different `.toolshare` files of nested environments are merged it is perfectly possible to only define a
source in, for example, the special system-wide environment and only define a pin in a local environment's `.toolshare`
file. In such a setup:

* When invoked from within the local environment `toolshare` will fetch the binary for the pinned version from the
  configured system-wide source.
* If the tool is invoked from outside any local environment this will result in an error as there will be no pin to
  reference to determine the version that should be used.

### Stateful-mode

In _stateful_ mode, to configure a tool for use with `toolshare`, only one **optional** element comes into play:

* A version pin, meaning the version that should be used when invoking the tool

Example:

```yaml
pins:
  kubectl: "1.20.1"
```

See the [stateful documentation](./internals/state.md) for additional information.
