# plugin-docker-buildx

Woodpecker CI plugin to build multiarch Docker images with [buildx](https://docs.docker.com/buildx/working-with-buildx/).

This is a fork of [woodpecker-ci/plugin-docker-buildx](https://github.com/woodpecker-ci/plugin-docker-buildx) maintained by GowerStreet.
See [upstream docs](https://woodpecker-ci.org/plugins/Docker%20Buildx) for the full settings reference.

## Why we forked

The upstream plugin had two issues that caused disk exhaustion on the CI host:

1. **`purge` was a no-op.** The setting was documented and defaulted to `true`, but the `settings.Cleanup` field was never read by `Execute()`. Every job created a new `buildx_buildkit_*` container and a ~4 GB state volume and left both running permanently. On a busy day this accumulated 80–90 GB of orphaned builders.

2. **No BuildKit GC policy.** Even if builders had been cleaned up, their caches had no upper bound. A single builder could grow to fill the disk on long-running pipelines.

## What we changed

### 1. `purge` now works (`purge: true` is the default)

`Execute()` captures the builder name from `docker buildx create --use` output, then runs `docker buildx rm <name>` after the build completes. The builder container and its state volume are removed at the end of every job.

Set `purge: false` only if you need to inspect the builder after a failed build.

### 2. BuildKit GC defaults are always applied

The auto-generated buildkit config now always includes a `[worker.oci]` section:

```toml
[worker.oci]
gc = true
gckeepstorage = 1000        # 1 GB max local cache

[[worker.oci.gcpolicy]]
keepBytes = 536870912       # evict above 512 MiB
keepDuration = 7200         # evict layers older than 2 hours
```

This bounds the local cache footprint of each builder while it is alive, acting as a backstop in case purge ever fails. If you set `buildkit_config` explicitly these defaults are not applied — your config is used as-is.

### 3. Registry cache is enabled by default (`auto_cache: true`)

When `auto_cache` is enabled (the default) and no explicit `cache_from`, `cache_to`, or `cache_images` settings are provided, the plugin automatically derives a registry cache image from the first `repo` value:

```
repo: gowerstreet/myapp  →  cache image: gowerstreet/myapp:buildcache
```

This gives every job free layer caching with zero per-pipeline config. The same credentials used to push the image cover the cache tag. The cache is written with `mode=max` so all intermediate layers are stored, making layer-cache hits as effective as possible.

Auto-cache is suppressed when:
- `dry_run: true` (no push, so no cache write)
- Any of `cache_from`, `cache_to`, or `cache_images` is set explicitly

Opt out globally with `auto_cache: false`.

## Image

```
gowerstreet/plugin-docker-buildx:latest
```

## Minimal example

```yaml
steps:
  publish:
    image: gowerstreet/plugin-docker-buildx
    settings:
      repo: gowerstreet/myapp
      tags: ${CI_COMMIT_SHA}
      username:
        from_secret: docker_username
      password:
        from_secret: docker_password
```

This will:
- Build and push `gowerstreet/myapp:<sha>`
- Pull cache from `gowerstreet/myapp:buildcache` (first run: miss; subsequent runs: fast)
- Push updated cache to `gowerstreet/myapp:buildcache`
- Remove the ephemeral BuildKit builder and its state volume on completion

## Keeping in sync with upstream

```bash
git remote add upstream https://github.com/woodpecker-ci/plugin-docker-buildx.git
git fetch upstream
git merge upstream/main
```

Our changes are confined to `plugin/impl.go`, `cmd/docker-buildx/config.go`, and `plugin/impl_test.go`, so merges are usually clean.

## License

Apache-2.0 — see [LICENSE](./LICENSE).
