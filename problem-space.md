# Problem space

## Problems with software development

People! People will tend to always do the easiest thing in a given situation. `toolshare`'s pervading philosophy is to make the easiest thing to do also a really good, if not the best, thing to do.

Examples:

1. removing manual steps from workflows (manual steps are prone to human error) by encoding "good" into easy workflow / UX.
1. advising optimisations for points where workflows interact with `toolshare` to get the best experience - for example, CI, release, compliance.

### {Up-front assumptions}

#### You use software from third parties in your workflows

Pretty fundamental. If you don't, you probably don't need `toolshare`.

#### You want to keep up with your software dependencies

There are lots of benefits to doing this; mainly, other people are working on the tools you're using to improve and fix them. You want to take advantage of those efforts.

#### You don't want to check binaries into source control

This is a workable solution but has some drawbacks especially with `git` version control. It solves the Works On My Machine and onboarding problems handily. Over time, your git repository will bloat and operations will appreciably and noticeably slow down.

#### You care about being able to reproduce builds at a point in history

It's common for CI and workstations to be capable of building at `HEAD` since that's where most work happens. But - if you want to hotfix at a point in history - you have more work to do to figure out which versions of which tools are in play at the time to recreate the environment.

#### You elect to read our "how to: get initially set up" guide

This describes how to install `toolshare`, get set up locally, and get set up inside build-scripts / CI.

The latter boils down to:

1. "run `toolshare install` as the first thing in your build-scripts"
1. "if you containerise your CI, insert a step at the front of your pipeline to update-and-push new containers when `toolshare` configuration changes in source control"

See [how to: get started].

[how to: get started]: docs/how-to.get-started.md

### Onboarding new people to team

Friction: "Here's a computer & source control credentials - just try to get it to build locally ok?"

Slightly better: "Here's the onboarding doc (last updated 9 months ago by new person) - just follow that and fill in any gaps as you find them to get it to build locally, ok?"

Good: "Here's a computer, `config-management-thing` should be wired up to have stuff installed for you; run this script and that should get you going - as long as you don't also work on product B which uses a different version of the compiler, in which case, um... Figure it out"

`toolshare`:

1. Check out source control
1. `cd {checkout}`
1. `toolshare install`

### Maintaining existing peoples' environments

#### ... when someone else has changed a tool or introduced a new one

Friction: "My build worked yesterday and doesn't now - what changed?" "Oh, um, not sure - go through each bit of the build script where tools are used and debug it!"

Slightly better: "Oh, Dave updated `test-runner-x` yesterday but he's out today - check out the new releases and figure out which one he used?"

Good: "Oh, Dave updated `test-runner-x` yesterday, recorded it in the onboarding doc, and pushed a new CI container - you just need to update your `tool-runner-x` to `1.2.3` and you're done."

`toolshare`:

1. Update source control (contains team-member's change to tools)
1. `cd {checkout}`
1. {do regular things and it fails locally}
1. `toolshare install`
    * now you're in sync with whatever tool(s) were changed; if it still fails, it's _not_ tool drift that's causing this.

#### ... when you are updating tool versions to keep up with fixes and features

Friction A: "Oh, I don't know, there are tonnes of changes since the last time we updated I think..." {tool goes un-updated in this scenario - this can lead to bigger-bang updates in future which might be under pressure if for example there's a critical fix or security issue needed}

Friction B: "Oh, just try it out locally and if it works, awesome!" {CI and team go un-updated in this scenario}

Slightly better: "Try it out locally, push a new CI environment change, update onboarding docs" {Team all get potentially broken as soon as this change lands}

Good: "Try it out locally, tell everyone to update to the version you chose, push a new CI environment change to capture the new dependency, update the onboarding docs, and don't go on holiday tomorrow."

`toolshare`:

1. Make a new branch
1. Update `.toolshare/test-runner-x.tool.yaml` to use your new version
1. Commit, push, PR.

* Team-members will be notified to `toolshare install` (or their `direnv` or build-scripts does this up-front).
* CI (if you have followed our integration advice)

  * containerised: will (one-time) re-build containers to new unique hashes.
  * non-containerised: will `toolshare install` at the front of build-scripts.

#### ... where those people work on more than one thing on their computer

Friction: "Oh, just change `tool-runner-x` to be the version that product B wants when you're working there, then remember to swap it back for product A. Good luck!"

Slightly better: "Oh, just maintain n installs of `tool-runner-x` on your machine and set up `PATH` when you swap working between products."

Good: "Oh, [direnv] should have set up your `PATH` when you `cd` into product B so this should Just Work {and honestly, this is a pretty great experience!}"

`toolshare`:

1. _There's nothing to do_; `toolshare` routes your `tool-runner-x` invocations (at a cost of a couple milliseconds per invoke) to the correct binary depending on which product (actually, the current-working-directory) you're inside.

[direnv]: https://direnv.net

### Maintaining CI environments optimally

#### ... when containerised

Friction: "CI isn't containerised, it's whatever's on the build agents and they're all snowflakes... Just remote onto each one as an admin and install stuff."

Slightly better: "Build the container and push `latest` - the builds will settle down once CI and people pull it."

Good: "Build the container and choose a new version tag for it, then update all source control that uses that to the new tag. Don't miss any!"

`toolshare` (if you have followed our integration advice)

1. See [when you are updating tool versions to keep up with fixes and features] above.

    * since `toolshare ci hash` is used to resolve the container version to run in, the container holds all necessary tool dependencies.
    * since `toolshare install` is first-thing inside build-scripts, the tool dependencies are set-and-checked per build.

[when you are updating tool versions to keep up with fixes and features]: #when-you-are-updating-tool-versions-to-keep-up-with-fixes-and-features

#### ... when not containerised

Friction: "Oh, a different team maintains the build farm, file a ticket and they'll update each machine at some point."

Slightly better: "Oh, learn {configuration management tool} and then commit a change to this repository and roll it out; the build farm will update at some arbitrary later point and that might interrupt some builds, but it'll probably be fine. Don't make any mistakes!"

Good: "{as above} ... and try it out in a staging set of the build farm first."

`toolshare` (if you have followed our integration advice)

1. See [when you are updating tool versions to keep up with fixes and features] above.

    * since `toolshare install` is first-thing inside build-scripts, the tool dependencies are set-and-checked per build.

### TODO: Yanking tools

TODO: Managed `toolshare`.

### TODO: Spotting oddly-slow or often-failing-weirdly tools

TODO: Observable `toolshare` tools.

## Things `toolshare` is not

### Not: a software library manager

For example: python's `pip`, node's `npm`.

Most software ecosystems have at least one of these. `toolshare` manages binaries that are "whole" pieces of software, not the "library" constituent parts.

### Not: a software language manager

For example: [`rbenv`], [`rvm`], [`pyenv`], [`nvm`] et al.

These manage the language runtimes and standard libraries for a particular language's ecosystem. Existing tools do this well. `toolshare` doesn't do this.

[`rbenv`]: https://github.com/rbenv/rbenv
[`rvm`]: https://rvm.io/
[`pyenv`]: https://github.com/pyenv/pyenv
[`nvm`]: https://github.com/nvm-sh/nvm

## Things `toolshare` has similarities with

Non-exhaustive. Compare `toolshare` to prior/existing art.

|| Thing || Notes ||
| [`tfenv`] et al | These manage the installation of one particular tool. `toolshare` can replace these entirely; it can manage arbitrary tools. |
| [`homebrew`] | This is a software package manager. It lacks the ability to pin to a particular version without effort. [citation-homebrew] |
| [`nix`] | This is a software package manager. It lacks the ability to pin to a particular version without effort. [citation-nix] |
| [`anyenv`] | This is quite comparable to toolshare. It does not support Windows. |

[`tfenv`]: https://github.com/tfutils/tfenv
[`homebrew`]: https://brew.sh/
[citation-homebrew]: https://github.com/Homebrew/discussions/discussions/155
[`nix`]: https://github.com/NixOS/nix
[citation-nix]: https://nixos.wiki/wiki/FAQ/Pinning_Nixpkgs
