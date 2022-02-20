# Reference: Command Line Interface (CLI) structure

This is intended as a sketch so we can play with what goes where.

## Assumptions / conventions

1. named flags for all parameters - no positional args
    * ... since those are harder UX (less discoverable) and increase risk of breaking changes between versions.

## Structure

```txt
toolshare
  init             # make a .toolsharerc/ in current working directory when it's missing
                   # ? warn if {repo-root}/.toolsharerc is missing? Having that there is the 80/20 case; putting things elsewhere is a bit edge-case

    add            # add a new tool dependency; .toolsharerc/{tool}.tool.yaml
      --tool-name [required]
      --tool-version [required]

  install          # resolve tools based on configuration on disk; install any tools that are missing from local toolshare; ensure shims
                   # question: `sync` instead? We might yank tools later on in a managed setup.

  invoke           # invoke a tool at a particular version, passing along flags.
                   # Example: toolshare invoke --toolshare-invoke-tool-name clang-format --toolshare-invoke-tool-version 6.5.4
                   # Note - not really intended for human use, should only appear within shims.
                   # `--toolshare-invoke-*` are disambiguation for us; they'll be stripped out of the ones passed to the tool itself.
    --toolshare-invoke-tool-name [required]
    --toolshare-invoke-tool-version [required]
    --toolshare-invoke-priority [normal]
```

