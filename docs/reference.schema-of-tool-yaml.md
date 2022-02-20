# Schema of `*.tool.yaml`

## Minimal

```yaml
---
name: test-runner-x
version: '1.1.2'     # accept v1.1.2 on the basis of snipping the 'v'; YAML treats naked 1.1.1 as a decimal otherwise, not a string...?
```

## (sketch) Install strategy

Totally grain of salt. Supposed to reflect:

* To fetch tool; how to build download URL (parameterised by `arch`, `OS`, `major/minor/patch` version).
* To verify tool; how to fetch hash and verify download.
* To unpackage tool; how to unpackage tool into local share.

```yaml
install_strategy:
  [github_releases]:
    url: ...
    hash: sha1|...
    hash_file: ...
    package_file: test-runner-x-{arch}-{os}-{major}-{minor}-{patch}
    package_type: naked|zip|tgz|...
  [naked_url]:
    url: ...
```

Supposed we might want to support CI builds for those that publish those?
