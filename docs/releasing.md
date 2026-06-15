# Releasing Wanderer + keeping the ExApp in sync

Wanderer's version comes from `git describe --tags` (Makefile →
`-ldflags -X main.Version`). Tagging is the release act; a published
release also notifies the downstream Nextcloud ExApp
(`MWest2020/wanderer-exapp`) to rebuild its image pinned to the tag.

## Cut a release

```sh
# 1. land the CHANGELOG: move [Unreleased] entries under a new
#    [X.Y.Z] - <date> heading, leave an empty [Unreleased] on top.
# 2. tag + push
git tag -a vX.Y.Z -m "Wanderer vX.Y.Z"
git push origin vX.Y.Z
# 3. publish a GitHub release (this is what fires the ExApp sync)
gh release create vX.Y.Z --title "vX.Y.Z" --notes-from-tag
```

`make build` then reports `vX.Y.Z` via `wanderer version`.

## One-time: the ExApp dispatch secret

The `release-dispatch` workflow calls `repository_dispatch` on
`wanderer-exapp` so it rebuilds against the new tag. The default
`GITHUB_TOKEN` **cannot** dispatch to another repo, so add a token
secret once:

1. **Create a token** (either works):
   - Classic PAT with the `repo` scope, **or**
   - Fine-grained PAT scoped to `MWest2020/wanderer-exapp` with
     **Contents: Read and write** (the permission GitHub's
     `POST /repos/{owner}/{repo}/dispatches` endpoint requires).

   GitHub → Settings → Developer settings → Personal access tokens.

2. **Store it as a secret on this repo** (run it yourself — never
   paste a PAT into a tool):

   ```sh
   gh secret set EXAPP_DISPATCH_TOKEN --repo MWest2020/wanderer
   # paste the token when prompted
   ```

   Or via the UI: `wanderer` → Settings → Secrets and variables →
   Actions → New repository secret → name `EXAPP_DISPATCH_TOKEN`.

That's it. On the next published release the workflow dispatches
`event_type=wanderer-release` with the tag; `wanderer-exapp`'s
`sync-wanderer` workflow rebuilds + pushes
`ghcr.io/mwest2020/wanderer-exapp:latest` (+ `core-<tag>`).

Without the secret the workflow fails loudly with a clear message —
it never silently no-ops.

## Manual trigger (without a release)

```sh
gh workflow run release-dispatch.yml -f version=vX.Y.Z   # this repo
# or directly on the downstream repo:
gh workflow run sync-wanderer.yml -f version=vX.Y.Z --repo MWest2020/wanderer-exapp
```
