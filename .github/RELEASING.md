## Release Process

[Semantic Versioning 2.0.0 reference](https://github.com/semver/semver/blob/master/semver.md)

Recommended versioning for the next release line:

- Start the next stable release at `v0.10.0`.
- Keep the binary's embedded version in source as a normal semver such as `0.10.0`.
- Let merge-driven builds update a single rolling `prerelease` tag.
- Reserve semver tags like `v0.10.x` for stable releases.

### 1. New Feature or Breaking‑Change Release (Minor/Major)

1. **Merge & Verify**

- Merge all feature or breaking‑change PRs into `master`.
- Ensure CI (tests, linter, codegen) all pass on `master`.

1. **Determine Version Bump**

- **Major** (`X.0.0`) when you make incompatible changes
- **Minor** (`0.Y.0`) when you add functionality in a backward compatible manner
- **Patch** (`0.0.Z`) when you make backward compatible bug fixes

1. **Merge to `master`**
   - Merging to `master` triggers the `Release` workflow automatically.

   Or, for a manual test without merging:
   - Run the `Release` workflow with `workflow_dispatch`.
   - Optionally set `release_tag` if you want the run to upload assets to a
     specific existing or new release tag instead of the default prerelease tag.
   - Run `Release Latest Assets` when you want the Actions UI path that updates
     the repo's current latest GitHub release tag without any manual input.

2. **Monitor Release**
   - GitHub Actions will:
     - Run `go generate ./...`
     - Build per-platform binaries with `main.version` set from `main.go`
     - Replace the rolling `prerelease` tag and GitHub prerelease
     - Archive `_datafiles` as `gomud-ALL-datafiles-prerelease.zip`
     - Generate `gomud-prerelease-SHA256SUMS.txt`
     - Leave the release unmarked as `Latest`

3. **Announce**
   - After review, a repo owner can edit the release in GitHub and promote it to
     `Latest`.
   - Share the release link with the team or via configured notifications.

---

### 2. Merge-Driven Prerelease Policy

1. **Pull requests do not publish release binaries**
   - PRs should run normal CI only.

2. **Merges to `master` do publish release binaries**
   - A push to `master` runs the `Release` workflow and replaces the rolling
     `prerelease`.

3. **Manual test runs can also publish prereleases**
   - `workflow_dispatch` can be used to refresh the rolling `prerelease`
     without merging.
   - If you provide `release_tag`, the run publishes to that tag instead.
   - `Release Latest Assets` is the no-input manual workflow that resolves the
     current latest GitHub release tag and publishes to it.

4. **Rolling release naming**
   - The release tag is always `prerelease`.
   - The release notes record the commit SHA and publish time for the current build.
   - Numbered releases such as `v0.10.0` remain the permanent download history.

---

### 3. Manual Test Release Flow

1. **Run the release workflow manually**
   - Use `workflow_dispatch` when you want a test release
     without merging to `master`.
   - Leave `release_tag` blank to use the default manual prerelease naming.
   - Set `release_tag` when you want the run to upload assets to a specific
     existing or new release tag.
   - Run `Release Latest Assets` when you want the no-input Actions UI path
     that uploads assets to whichever tag GitHub currently considers the latest
     release.

2. **Verify the GitHub release**
   - Confirm the workflow succeeds.
   - Confirm the targeted release now points at the expected commit.
   - Confirm the per-platform binaries are attached.
   - Confirm the `_datafiles` zip asset is attached.
   - Confirm the checksum manifest asset is attached.
   - For default manual prerelease runs, confirm GitHub marks the release as a
     prerelease and does not mark it as `Latest`.

3. **Clean up if needed**
   - No cleanup is normally required for the rolling `prerelease` path because
     the next successful run replaces it.
   - Tag-targeted manual runs use the tag you requested, so cleanup is only
     needed if you created a temporary release tag for testing.

---

### FAQ / Guidelines

- **Does every merge to `master` trigger a release?**
  Yes - every push to `master` runs the release workflow and publishes a prerelease.
  That prerelease is the rolling `prerelease` entry, not a newly named release.

- **Is auto-tagging enabled?**
  Stable semver tags are not generated automatically. The merge-driven workflow
  updates the rolling `prerelease` tag instead.

- **Can I create a test release without merging to `master`?**
  Yes - run the `Release` workflow manually with `workflow_dispatch`. That keeps PR
  submissions clean while still allowing an on-demand refresh of `prerelease`.
  Set `release_tag` when you want the run to publish assets to a specific tag.
  Run `Release Latest Assets` when you want the no-input path that targets the
  current latest release.

- **Are workflow-created releases stable releases?**
  No - the workflow creates prereleases. A repo owner must manually promote a release
  to `Latest` in GitHub when it is approved.

- **What assets should a release include?**
  Each release should include separate per-platform binaries, a `_datafiles` zip,
  and a checksum manifest so testers can download only what they need and still
  verify the assets.

- **What tag format should we use going forward?**
  Keep the source/binary version on the `0.10.x` line. Use `prerelease` for the
  rolling master build and semver tags for stable releases.

- **When should I bump minor vs. patch?**
  - **Minor** for new, backward‑compatible features.
  - **Patch** for bug fixes or documentation tweaks.

- **What about `go generate` directives?**
  The workflow runs `go generate ./...` automatically before each build.
