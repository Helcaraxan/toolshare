# Reference: structure of toolshare

## Top-level footprint

This is the filesystem footprint.

It's designed to accomodate repo-local configuration, with user-level and system-level configuration and content.

```txt
{repo-root}/
  .toolsharerc/
    *.tool.yaml

  {there's no content kept in-repo}

{user-home}/
  .toolsharerc/
    *.tool.yaml               # control user-level tools (for example tools that different users use that are not repo-scoped)
  {content}/

{system-level}/
  toolsharerc/
    *.tool.yaml               # control system-level tools (for example system-singletons, persistent services)
  {content}/
```

## `content` footprint

This is the `content` footprint:

It's designed to accomodate

* simultaneous availability across multiple OS', architectures, and versions of binaries - in a Write-Once-Read-Many fashion so that once a tool is downloaded, it's not touched again.
* later extension-space for the ability to ship other content like configuration files.

```txt
{level}/content/
  v1/                       # hedge against breaking changes to layout
    {arch}/                 # architecture (arm, x86, etc)
      {os}/                 # operating system (darwin, linux, windows etc)
        {tool-name}/        # name of the tool
          {tool-version}/   # version of the tool
```
