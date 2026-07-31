# Feature Backlog

A running list of feature improvements for the changelog controller.

**Detailed package change history**

Currently, the RPM diff shows which packages changed between two releases, along with their old and new versions, but not *what* changed within each package. It would be nice to have visibility into the underlying changes for each package (e.g. CVE fixes, commit-level details) between releases.

In order to get there, we would likely need to run a container against the RHCOS image and extract per-package change data (e.g. via `rpm -q --changelog`), though the approach would need further investigation to work out the details.
