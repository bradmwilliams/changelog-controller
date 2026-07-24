# Feature Backlog

A running list of feature improvements for the changelog controller.

**Kernel version lookup per release**

Currently, determining the kernel version associated with a given release requires retrieving the full changelog and searching through it. It would be nice to have a dedicated way to query the kernel version for a release directly, without requiring a full changelog or RPM diff lookup.

This could be implemented by adding an endpoint such as `/release/{tag}/kernel` that looks up the kernel package entry (e.g. `kernel` or `kernel-rt`) from the RPM package list already extracted and stored for that release, and returns its version directly.

**Detailed package change history**

Currently, the RPM diff shows which packages changed between two releases, along with their old and new versions, but not *what* changed within each package. It would be nice to have visibility into the underlying changes for each package (e.g. CVE fixes, commit-level details) between releases.

In order to get there, we would likely need to run a container against the RHCOS image and extract per-package change data (e.g. via `rpm -q --changelog`), though the approach would need further investigation to work out the details.
