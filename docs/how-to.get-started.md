# How to: get started

## Install locally

## Set up within build-scripts / CI

### Basic

### Optimal

To reduce repeated work (wait-time & compute spend):

1. Insert a step into your CI pipeline that

    1. computes a unique hash to tag your containers with

        * run `toolshare ci hash` in your repository root directory

    1. rebuilds your CI container(s) if the hash is different.
    1. pushes them to your registry tagged with that unique hash.
    1. (To avoid _every_ CI step doing this all at once, every time; this should block subsequent CI steps).

1. Make each CI step resolve the container hash before running work.
