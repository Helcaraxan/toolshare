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
    github_slug: kubernetes/kubectl
    github_release_asset_template: kubectl-{version}-{platform}-{arch}.tar.gz
    archive_path_template: ./kubectl{exe}
```

Because the different `.toolshare` files of nested environments are merged it is perfectly possible to only define a
source in, for example, the special system-wide environment and only define a pin in a local environment's `.toolshare`
file. In such a setup:

* When invoked from within the local environment `toolshare` will fetch the binary for the pinned version from the
  configured system-wide source.
* If the tool is invoked from outside any local environment this will result in an error as there will be no pin to
  reference to determine the version that should be used.

Binaries, when not yet locally cached nor available in any configured remote storage, need to be fetched from a source.
`toolshare` supports multiple sources types.

#### Path templates

Regardless of the source from where a tool binary may be fetched, you will need to tell Toolshare a path at which the
specific binary for a given OS and tool version may be found. This is achieved through templated paths such as:

```text
my-tool/{version}/{platform}_{arch}{exe}
```

The elements in `{brackets}` will be replaced at runtime with the appropriate value for the specific tool binary that
needs to be fetched. Supported `{items}` are:

* `{arch}` - replaced with `x86_32`, `x86_64`, `arm32` or `arm64`.
* `{exe}` - replaced with `.exe` on Windows and with an empty string on all other platforms.
* `{platform}` - replaced with `darwin` (Mac), `linux` or `windows`.
* `{tool}` - replaced with the tool's name which corresponds to name of the source mapping in the configuration file.
* `{version}` - replaced with the tool version that needs to be fetched.

Because many third-party tool sources use their own variations on platform and architecture names it is possible to
customize the default mappings. Use the `template_mappings` setting in any given source configuration.

Example:

```yaml
sources:
  my-tool:
    # [...] Source-specific configuration
    template_mappings:
      x86_64: amd64  # Will result in '{arch}' being replaced with 'amd64' instead of 'x86_64' in path templates.
      darwin: macos  # Will result in '{platform}' being replaced with 'macos' instead of 'darwin' in path templates.
```

#### GitHub sources

To fetch a tool from GitHub there are two mandatory source configuration elements: the repo slug and the release-asset
template. The former to know the GitHub project from which to fetch the tool binaries, the second to define what release-asset to fetch for a given version of the tool and OS.

Example with the `kubectx` tool:

```yaml
sources:
  kubectl:
    github_slug: ahmetb/kubectx
    github_release_asset_template: kubectx_{version}_{platform}_{arch}.tar.gz
    archive_path_template: kubectx
    # Please note that this will not work on Windows as 'kubectx' is not supported on that platform.
```

One can also observe the use of the `archive_path_template` configuration which must be used, if the downloaded asset is
in archive form, to specify the path within the archive where to retrieve the tool binary.

An additional optional GitHub-specific configuration is the `github_base_url` setting to point Toolshare to a
self-hosted GitHub Enterprise server.

#### HTTPS sources

To fetch tool binaries from a URL one can use an HTTPS source. An example with the `terraform` tool that can be fetched
from the official website in the form of an archive containing a single binary.

```yaml
sources:
  terraform:
    https_url_template: https://releases.hashicorp.com/terraform/{version}/terraform_{version}_{platform}_{arch}.zip
    archive_path_template: terraform{exe}
    template_mappings:
      arm32: arm
      x86_32: "386"
      x86_64: amd64
```

#### Filesystem sources

In certain cases a tool binary might be shared via a read-only, possibly non-executable and / or network-mounted
filesystem. In such a case, to allow for network-less low-latency execution of the tool Toolshare can be pointed to such
a filesystem source:

```yaml
sources:
  private_tool:
    file_path_template: file:///tool-sources/private_tool/{version}/{platform}_{arch}{exe}
```

#### GCS or S3 cloud storage bucket sources

Beyond (network-mounted) filesystem sources tool binaries may also be shared via cloud buckets. There is support for
both GCS and S3 backends:

```yaml
sources:
  gcs_tool:
    gcs_bucket: our-gcs-bucket
    gcs_path_template: tool-sources/gcs_tool/{version}/{platform}_{arch}{exe}

  s3_tool:
    s3_bucket: our-s3-bucket
    s3_path_template: tool-sources/s3_tool/{version}/{platform}/{arch}/s3_tool{exe}
```

Authentication for both cloud providers are fetched from their default locations as stored by `gcloud auth login` and
in the AWS CLI configuration file.

### Stateful-mode

In _stateful_ mode, to configure a tool for use with `toolshare`, only one **optional** element comes into play:

* A version pin, meaning the version that should be used when invoking the tool

Example:

```yaml
pins:
  kubectl: "1.20.1"
```

See the [stateful documentation](./internals/state.md) for additional information.
