# Architecture overview

## Dataflows

```mermaid
flowchart TD
    subgraph sources[Sources]
        s_fs[Filesystem]
        s_gcs[GCS]
        s_gh[GitHub]
        s_https[HTTPS]
        s_s3[S3]
    end
    subgraph Storage
        subgraph remote_cache[Remote cache]
            r_fs[Filesystem]
            r_gcs[GCS]
            r_https[HTTPS]
            r_s3[S3]
        end
        subgraph local_cache[Local cache]
            l_fs[Filesystem]
        end
    end
    subgraph state[State]
        st_gcs[GCS]
        st_git[Git]
        st_http[HTTPS]
        st_s3[S3]
    end
    client[Toolshare client]

    sources -->|Get| client
    client <-.->|Get / Store| remote_cache
    client <-->|Run / Store| local_cache
    state <-..->|Fetch / Update| client
```

## Type relationships

```mermaid
classDiagram
    class Backend {
        <<Interface>>
        +Fetch(Binary) Bytes
        +Store(Binary, Bytes)
    }
    class BinaryStore {
        <<Interface>>
        +Path(Binary) String
    }
    class Binary {
        <<Struct>>
        +Tool Tool
        +Version Version
        +OS OS
        +Arch Arch
    }

    class Filesystem {
        <<Struct>>
        -Source
        +New(Source) Filesystem
    }
    class GCS {
        <<Struct>>
        -Source
        +New(Source) GCS
    }
    class GitHub {
        <<Struct>>
        -Source
        +New(Source) GitHub
    }
    class S3 {
        <<Struct>>
        -Source
        +New(Source) S3
    }

    Backend <-- GCS: implements
    Backend <-- GitHub: implements
    Backend <-- S3: implements
    Backend <-- Filesystem: implements

    BinaryStore <-- Filesystem: implements

    class Source {
        <<Struct>>
        +String Type
        +String ArchivePathTemplate
    }
    class GitHubSource {
        <<Struct>>
        +String Slug
        +String ReleaseAssetTemplate
        +String BaseURL
    }
    class URLSource {
        <<Struct>>
        +String URLTemplate
    }

    Source --> "0..1" GitHubSource: contains
    Source --> "0..1" URLSource: contains
```

## Sources

Source are providers of tool binaries on a one-source-per-binary basis. They are strictly read-only resources. A source
is the main abstraction and translation layer that allows to bridge the gap between the widly varying storage and naming
schemes employed by each individual tool and `toolshare`'s well-defined internal schemes.

## Local cache

Because nobody wants to have to download a tool each time you run it, `toolshare` caches binaries locally in a folder
tree on a write-once basis.

## Remote cache

A remote cache is similar to a source in that it provides tool binaries. The difference lies in the fact that a remote
cache stores binaries for many different tools using `toolshare`'s well-defined internal schemes. It can be seen as a
copy of the local cache but available over a network rather than on a local disk.

Using a remote cache does not replace the local cache. It is instead used as a secondary cache. If a tool is not
available in the local cache it is fetched from the remote cache. If it's also not available in the remote cache it is
fetched from the source, if one is specified.

> NOTE: Using a remote cache is entirely optional and is mainly intended for use in the context of an organisation-wide
> deployment of `toolshare`. In such cases the remote cache may be:
>
> * stored on an office-based NFS share or server to improve download speed and latency.
> * used to store internal tools that are not published to sources available over the public internet.
> * used because the internet-connectivity necessary to reach sources is restricted or unavailable (air-gapped
>   environment).
> * controlled by a team that performs validation of tools before allowing their use and disables the use of public
>   sources via a centrally managed system-level configuration of `toolshare`.

## State

A toolshare state allows for central management of:

* recommended versions when a tool is not pinned
* a deny-list of specific tool versions (e.g due to known vulnerabilities)

Read [the detailed documentation](./state.md) for more information.

> NOTE: Using a state is entirely optional and is mainly intended for use in the context of an organisation-wide
> deployment of `toolshare`.
