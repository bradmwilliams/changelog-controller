# Release Payload Changelog Service

---

## Overview

The goal of this project is to extract changelog and diff generation out of `release-controller-api` into a dedicated service that can be reliably queried by both the release controller UI and programmatic consumers. Currently, the changelog feature is embedded in the release controller and cannot be consumed reliably in an API-first way.

One area where a programmatic diff API would be valuable is automated regression analysis. For example, the payload agent (`claude-payload-agent`) analyzes rejected release payloads to identify which PR caused a regression, but currently has no reliable way to programmatically query structured diff data between builds.

Additionally, because changelog generation runs inline on the request path with only an in-process cache, pod restarts cause repeated cold starts and two types of failures:

- **504 Gateway Timeout:** On a cache miss, generating a changelog requires cloning git repositories for every changed component (often 30 to 50 repos) before the response can be returned. This can take several minutes, causing the request to exceed the API server's response timeout and return a 504 to the caller.
- **429 Too Many Requests:** RPM diff operations are expensive and capped at 16 concurrent `oc` calls per pod. When the cache is cold (e.g., after a pod restart), many requests arrive at once, all 16 slots fill up, and any further requests are immediately rejected with a 429 until slots free up.O

---

## Current Architecture

The existing implementation lives in `openshift/release-controller`:


| File                                           | Description                                                                                                                                                                                                                                                                                                                                                      |
| ---------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `cmd/release-controller-api/http.go`           | Registers all HTTP routes for the API server, including the `/changelog` endpoint. Entry point for understanding which URLs map to which handlers.                                                                                                                                                                                                               |
| `cmd/release-controller-api/http_changelog.go` | The main handler for changelog requests. When a request comes in with `from` and `to` version tags (e.g. `4.15.0`, `4.16.0`), this file looks up the actual container image references those tags point to, calls the appropriate `oc` functions to generate the diff, and formats the result as JSON, markdown, or HTML depending on what the caller requested. |
| `pkg/release-controller/release_info.go`       | The layer that actually runs `oc` subprocesses. Contains all functions for fetching image metadata, generating changelogs, and diffing RPM packages. Also defines the in-process cache (`groupcache`) and the concurrency semaphore that caps RPM operations at 16 concurrent calls.                                                                             |
| `pkg/release-controller/types.go`              | Carries a local copy of the changelog data schema (`ChangeLog`, `ChangeLogImageInfo`, `CommitInfo`, `RpmDiff`, etc.). The canonical source for this schema is `oc` itself ([openshift/oc/pkg/cli/admin/release/info.go](https://github.com/openshift/oc/blob/main/pkg/cli/admin/release/info.go)); release-controller maintains its own copy of the structures. |
| `pkg/rhcos/rhcos.go`                           | Post-processing for CoreOS-specific content. Enriches the JSON struct with CoreOS component metadata and rewrites CoreOS version strings in markdown output into clickable links.                                                                                                                                                                                |
| `pkg/rhcos/node_image.go`                      | Generates the Node Image Info section of a changelog, which lists RPM package changes on the OpenShift node image (the CoreOS-based OS that runs on cluster nodes).                                                                                                                                                                                              |


### Request flow today

The primary consumer API is `GET /api/v1/releasestream/{stream}/release/{tag}`, which returns the combined `ReleaseInfo` and `ChangeLog` as a single JSON response. The `/changelog` endpoint described below is mostly a UI convenience and is not heavily used by programmatic consumers.

A single `GET /changelog?from=4.15.0&to=4.16.0&format=json` triggers all of this inline on the HTTP response goroutine:

1. Resolve `from` and `to` version tags to container image pull specs by looking them up in the release ImageStream on the `app.ci` cluster (this requires the full API server to be running with a live Kubernetes connection)
2. `oc image info <from>` and `oc image info <to>` -- fetch metadata for both images
3. `oc adm release info --changelog=/tmp/git/ <from> <to> --output=json` -- clones the git repository for every changed component (often 30 to 50 repos) and runs `git log` to get the commit history; this is the slow step
4. `oc adm release info -o json <to>` -- list the CoreOS streams present in the release
5. Per CoreOS stream: `oc adm release info --rpmdb <from>` and `--rpmdb <to>` -- fetch the full RPM package list for each release image
6. Per CoreOS stream: `oc adm release info --rpmdb-diff <from> <to>` -- diff the RPM packages between the two releases
7. `oc image extract <extensions-image>` -- extract node extensions metadata

### Current output formats

The `/changelog` endpoint accepts a `format` query parameter. The separate endpoint `/api/v1/releasestream/{stream}/release/{tag}` provides the combined `ReleaseInfo` and `ChangeLog` data and is the primary API that consumers use.


| Endpoint                                       | `format`             | Content-Type       | Description                                                                                                       |
| ---------------------------------------------- | -------------------- | ------------------ | ----------------------------------------------------------------------------------------------------------------- |
| `/changelog`                                   | `markdown` (default) | `text/plain`       | Raw `oc` output with CoreOS links rewritten; used by the release controller web UI                                |
| `/changelog`                                   | `html`               | `text/html`        | Same markdown content rendered to HTML; displayed in the browser                                                  |
| `/changelog`                                   | `json`               | `application/json` | Structured JSON matching the `ChangeLog` schema; intended for programmatic consumers                              |
| `/api/v1/releasestream/{stream}/release/{tag}` | n/a                  | `application/json` | Full release metadata response that includes a `changeLogJson` field with embedded changelog data for the release |


---

## Proposed Architecture

Currently, all the work to produce a changelog happens at request time: when a caller asks for a diff, the service runs `oc` to fetch metadata for both releases, clone git repositories, and compute RPM diffs from scratch, which is why cold requests are slow.

We should shift most of this work to happen earlier. As each release enters the Ready phase (when it begins verification), the service should automatically extract and store its component manifest, RHCOS image digests, and RPM package list. When a diff is later requested between two releases, the component and RPM data is already stored. The only remaining step is fetching the source commits for changed components (what `oc adm release info --changelog` does), which saves significant time compared to running the full pipeline on demand.

1. **Proactive extraction.** When a release enters the Ready phase, the service should immediately run `oc` to extract its component manifest and RPM package list and store the results, rather than waiting until a diff is requested. This way the expensive `oc` work happens once per release and is kept in process memory, rather than being re-run on every diff request. The service detects ready releases by polling the new `/api/v1/releasestreams/ready` endpoint on `release-controller-api`.
2. **Persistent storage.** Extracted results should be written to disk rather than kept only in process memory, so data survives pod restarts and is reloaded on startup, eliminating cold starts. The storage backend is configurable:
  - **Filesystem:** Stores each changelog result as a file on disk, named by the image digest of the release pair. This is the default and simplest option, requiring no additional infrastructure.
  - **SQLite:** Stores results in a local database file, which allows the service to query stored changelogs by release tag, date, or other fields rather than only looking up by exact digest.
  - **BigQuery:** Writes results to a cloud-hosted database, making it possible to run queries across all historical changelogs from any machine, not just the server running the service.
3. **Async handling for cache misses.** Even with proactive extraction, data may not be available for every request, for example when the service first starts up or a release pair has never been seen before. Rather than making the caller wait while `oc` runs, the service should return `202 Accepted` immediately and queue the work in the background. The caller can retry the request once the data is ready.
4. **Tag resolution.** Callers pass version tags like `4.15.0`, but `oc` requires full container image pull specs (e.g. `quay.io/openshift-release-dev/ocp-release@sha256:...`). Currently, `release-controller-api` resolves tags using in-process Kubernetes informers that watch the release ImageStreams on the `app.ci` cluster. Since the changelog service is a separate process, it does not have those informers. Instead, it should resolve tags by calling `release-controller-api`'s existing `/api/v1/releasestream/{stream}/release/{tag}` endpoint over HTTP, which returns the pull spec for a given tag.
5. **Structured JSON only.** The service should deal exclusively in structured JSON. Rendering into other formats (markdown, HTML) is a presentation concern that belongs in a separate helper library or package, not in the service itself. This keeps the service focused on data production and allows consumers to choose their own rendering.
6. **Per-package RPM changelogs.** The current RPM diff shows which packages were added, removed, or updated between two releases, but it would be useful to also know what changed within each package. This can be deferred but is worth revisiting later. Additionally, extracting per-package changelogs requires running containers inside containers (`podman run` against the RHCOS image), which presents security implications that need to be considered before implementation.

### Request flow

**Per-release extraction (triggered when a release enters Ready):**

1. Release watcher polls `/api/v1/releasestreams/ready` on `release-controller-api` and detects a newly ready release
2. `oc adm release info -o json <pull-spec>` -- extract component manifest
3. `oc adm release info --rpmdb <pull-spec>` -- extract RHCOS image digest and RPM package list
4. Store per-release record keyed by image digest

**Diff request (both releases already extracted):**

1. `GET /changelog?from=4.15.0&to=4.16.0&arch=amd64&format=json`
2. Resolve version tags to pull specs via `release-controller-api`
3. Look up both per-release records from in-memory cache or storage backend
4. `oc adm release info --changelog=/tmp/git/ <from> <to> --output=json` -- fetch git commit data (only remaining slow step; only on first request for a given pair)
5. Combine component manifest diff, RPM diff, and git commits into a `ChangeLog` struct
6. Write to cache and storage, return `200 OK`

**Cache miss (neither release has been pre-extracted):**

1. Return `202 Accepted`, enqueue full extraction for both releases
2. Worker runs per-release extraction for each, then the diff step
3. Next request for the same pair is served from cache

---

## Implementation Details

The new service will be built on top of the existing `changelog-controller` codebase, which already has the core service infrastructure in place. The remaining work is the actual changelog generation logic and several surrounding pieces.

### Already implemented


| Component                                                | File                                  |
| -------------------------------------------------------- | ------------------------------------- |
| HTTP server and routing                                  | `pkg/cmd/changelog-controller/cmd.go` |
| Request validation (semver, arch whitelist)              | Same file                             |
| Tag to pull spec resolution via `release-controller-api` | `pkg/release-controller/client.go`    |
| Async work queue (rate-limited, deduplicated)            | `controller.go`                       |
| In-memory cache with hit/miss stats                      | `controller.go`                       |
| Cache warming from storage on startup                    | `cmd.go:warmCache()`                  |
| Filesystem, SQLite, and BigQuery storage backends        | `pkg/storage/`                        |
| Storage abstraction interface                            | `pkg/storage/interface.go`            |


### Not yet implemented

**1. Core changelog generation in `sync()`**

`controller.go:sync()` is currently a no-op (lines 359-375, TODO comment). It needs to call `oc adm release info --changelog` and unmarshal the result.

Reference: `ExecReleaseInfo.ChangeLog()` in `release-controller/pkg/release-controller/release_info.go` (line 473). Handles image reference validation, `oc` binary lookup, subprocess execution, and returns raw JSON.

**2. Per-release extraction**

A new method that runs at release acceptance time rather than at diff request time:

- `oc adm release info -o json <pull-spec>` to extract the component manifest
- `oc adm release info --rpmdb <pull-spec>` to extract the RPM package list
- Store the result keyed by image digest

Reference: `ExecReleaseInfo.RpmListForStream()` (line 651) and `ExecReleaseInfo.RpmDiffForStream()` (line 767) in `release_info.go`.

**3. Release watcher**

A goroutine that polls `release-controller-api` for newly accepted releases and enqueues per-release extraction jobs. This is what makes the proactive model work.

**4. RPM diff generation**

Called during diff request processing, using stored per-release data where possible.

Reference: `ExecReleaseInfo.RpmDiff()` (line 740) and `ExecReleaseInfo.RpmDiffForStream()` (line 767) in `release_info.go`. Data types: `RpmDiff`, `RpmList`, `RpmChangedDiff` (line 794).

The existing release-controller uses a fixed concurrency semaphore (16 slots) that blocks requests when full. For the new service, we should avoid this fixed cap and instead support dynamic scaling to handle concurrent RPM operations without hard-blocking callers.

**5. JSON schema**

The response schema originates from `oc` ([openshift/oc/pkg/cli/admin/release/info.go](https://github.com/openshift/oc/blob/main/pkg/cli/admin/release/info.go)). This service should maintain its own copy of the struct definitions rather than importing from `release-controller` or `oc`:

```
ChangeLog
  from, to       ChangeLogReleaseInfo    (name, creation time, digest)
  components     []ChangeLogComponentInfo (version changes with GitHub diff URLs)
  newImages      []ChangeLogImageInfo
  removedImages  []ChangeLogImageInfo
  rebuiltImages  []ChangeLogImageInfo
  updatedImages  []ChangeLogImageInfo
    commits      []CommitInfo            (subject, PR URL, bug IDs)
```

`NodeImageStreams` (RPM diff data) currently lives on `APIReleaseInfo` (line 42), not on `ChangeLog`. It should be added directly to `ChangeLog` so the full diff is one unified struct. An OpenAPI spec should be written from this struct before significant consumer integration begins.

**6. Rendering (separate package)**

Rendering logic should be kept separate from the changelog generation logic. The service itself deals exclusively in structured JSON. A separate helper library/package should handle transforming the JSON data into other formats (markdown, HTML) for consumers that need them. The existing transforms in `release-controller/pkg/rhcos/` (`TransformJsonOutput`, `TransformMarkDownOutput`, `NodeImageSectionMarkdown`) can serve as reference for this helper package.

**7. Response formatting**

`writeChangelogResponse()` in `cmd.go` currently escapes the raw `oc` string. It needs to unmarshal into `ChangeLog` and return properly formatted JSON via `json.MarshalIndent()`.

**8. Missing endpoints**

- `GET /healthz`: returns `200` if the service is running
- `GET /readyz`: returns `200` when `oc` is on PATH; `503` otherwise
- `GET /metrics`: request latency, cache hit rate, queue depth, `oc` subprocess duration

**9. RPM package changelogs**

Can be deferred but worth revisiting later. Extracting per-package RPM changelogs requires running containers inside containers (`podman run` against the RHCOS image), which presents security implications that need to be addressed.

**10. Tests and deployment**

- Unit tests for handlers and storage backends
- Integration tests against a known release pair
- Kubernetes Deployment, Route, ConfigMap, pull secret
- Git cache volume (`/tmp/git/`) and RPM cache volume (`/tmp/rpmdb/`)

---

## Resolved Questions

1. **Extraction trigger.** The changelog service should poll the new `/api/v1/releasestreams/ready` endpoint to detect payloads entering the verification window and begin extraction immediately.

2. **JSON schema ownership.** The service should maintain its own copy of the types independently. No imports from `release-controller`.

3. **Tag resolution.** Support both tags and pull specs. When tags are provided, the service makes an additional API call to `release-controller-api` to resolve the pull spec.

4. **RPM package changelogs.** Can be deferred but worth revisiting later.

