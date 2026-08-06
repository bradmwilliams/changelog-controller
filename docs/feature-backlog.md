# Feature Backlog

A running list of feature improvements for the changelog controller.

**Detailed package change history**

Currently, the RPM diff shows which packages changed between two releases, along with their old and new versions, but not *what* changed within each package. It would be nice to have visibility into the underlying changes for each package (e.g. CVE fixes, commit-level details) between releases.

In order to get there, we would likely need to run a container against the RHCOS image and extract per-package change data (e.g. via `rpm -q --changelog`), though the approach would need further investigation to work out the details.

**Multi-kernel variant reporting**

Starting with OCP 4.21, release payloads ship two base OS images: `rhel-coreos` (RHEL 9) and `rhel-coreos-10` (RHEL 10), each with its own kernel version. The `/kernel` endpoint currently returns only the primary kernel (`rhel-coreos`). It could be extended to report all kernel variants, either via an optional query parameter (e.g., `?os=rhel-coreos-10`) or by including a `kernelVariants` map in the response alongside the primary kernel. This would require extracting RPMs from multiple OS images per release, which doubles extraction time and requires a storage schema change to support multiple RPM lists per snapshot.
