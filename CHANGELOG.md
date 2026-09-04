
## Unreleased

FEATURES:

* **New Resource:** `myjfrog_private_link`: Resource to manage MyJFrog PrivateLink connections for JFrog cloud instances.
* **New Data Source:** `myjfrog_private_links`: Data source to list PrivateLink connections for a JFrog cloud server.

## 1.0.3 (August 20, 2026)

SECURITY:

* Remediate CVE-2026-39821 (9.6 Critical).
* Remediate CVE-2026-56865 (8.4 High).
* Remediate CVE-2026-56864 (7.5 High).
* Remediate CVE-2026-33818 (7.5 High).
* Remediate CVE-2026-46600 (7.5 High).
* Remediate CVE-2026-56862 (7.5 High).
* Remediate CVE-2026-56859 (7.5 High).
* Remediate CVE-2026-56860 (7.5 High).
* Remediate CVE-2026-56858 (6.1 Medium).
* Remediate CVE-2026-56853 (5.3 Medium).
* Remediate CVE-2026-25680 (6.5 Medium).
* Remediate CVE-2026-42506 (6.1 Medium).
* Remediate CVE-2026-42502 (6.1 Medium).
* Remediate CVE-2026-25681 (6.1 Medium).
* Remediate CVE-2026-27136 (6.1 Medium).
* Remediate CVE-2026-46595 (10.0 Critical).
* Remediate CVE-2026-42508 (9.1 Critical).
* Remediate CVE-2026-39834 (9.1 Critical).
* Remediate CVE-2026-39833 (9.1 Critical).
* Remediate CVE-2026-39832 (9.1 Critical).
* Remediate CVE-2026-39831 (9.1 Critical).
* Remediate CVE-2026-39830 (9.1 Critical).
* Remediate CVE-2026-39829 (7.5 High).
* Remediate CVE-2026-46597 (7.5 High).
* Remediate CVE-2026-39828 (6.3 Medium).
* Remediate CVE-2026-39827 (6.5 Medium).
* Remediate CVE-2026-39835 (5.3 Medium).
* Remediate CVE-2026-46598 (5.3 Medium).
* Remediate CVE-2025-47914 (5.3 Medium).
* Remediate CVE-2025-58181 (5.3 Medium).
* Remediate CVE-2026-1229 (2.9 Low).

## 1.0.2 (Nov 7, 2025). Tested on Artifactory  with Terraform 1.13.5 and OpenTofu 1.10.7

BUG FIXES:

* Fix Custom Domain Name update fails after resource import and apply.

## 1.0.1 (January 23, 2024)

BUG FIXES:

* Fix provider name typo in documentation example, and README.md. PR: [#38](https://github.com/jfrog/terraform-provider-myjfrog/pull/38)

## 1.0.0 (July 19, 2024). Tested on Artifactory  with Terraform 1.9.2 and OpenTofu 1.7.3

FEATURES:

* **New Resource:** `myjfrog_ip_allowlist`: Resource to manage MyJFrog IP allowlist.
* **New Resource:** `myjfrog_custom_domain_name`: Resource to manage MyJFrog IP allowlist. PR: [#2](https://github.com/jfrog/terraform-provider-myjfrog/pull/2)
