# toolshare (name TBD)

## Why does this exist?

Working on software products in modern times requires lots of tools - just like building houses does. `toolshare` aims to remove friction from the process of iterating on your product by making it easy for everyone on the team to get set up, and be using the same version of the same tools.

### 2 minutes

You can use `toolshare` - a "tool dependency manager" - to ease or eliminate the onboarding and Works On My Machine problems when developing software, by

1. simplifying onboarding of new people (`toolshare onboard`).
    * problem: onboarding new people takes effort and time and is error-prone.
    * "how do I get set up to work on this?" "oh, just run through this out-of-date wiki onboarding document created by the person who left last month".
1. eliminating drift of tool versions (`toolshare sync`) between team-members.
    * problem: differently set up environments causes different results between workstations/CI. Sometimes those are obvious, sometimes subtle enough to be unnoticed.
    * "my `test-runner` is giving different results than your `test-runner` and CI's - wtf?" "oh, easy, you need version 1.5.6... No, that's not written down anywhere"

### More than 2 minutes

The above are the main, but not only, problems that `toolshare` tries to solve. For more detail and problems, check out [problem-space.md].

## How to: contribute

Please see [contributing.md]

[contributing.md]: contributing.md
