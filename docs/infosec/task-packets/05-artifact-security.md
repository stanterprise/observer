# TP-05 - Authorize and Safely Deliver Artifacts

**Status:** Proposed; no owner assigned.
**Priority:** P1 for safe delivery; per-user/project grants are deferred.
**Accountable roles:** Application and data owners.
**Findings:** 3 and 7.
**Controls:** IAM-02, ING-03/05, DATA-01/03/04/05, NET-19; test V-08.
**Prerequisites:** D-05/D-07 for content policy; ownership and human
authorization from [TP-02](02-project-ownership.md) and
[TP-03](03-human-access-and-authorization.md). Coordinate size enforcement
with [TP-06](06-ingestion-limits.md) and deletion with
[TP-10](10-retention-and-deletion.md).

## Objective and current state

Deliver untrusted artifacts only to authorized users without treating their
content, filenames, keys, or URLs as trusted. The
[attachment handler](../../../pkg/api/attachments_postgres.go#L30) looks up by
storage key and supports inline content, local/S3 objects, a 15-minute signed URL,
and external URI redirects. Inline and proxied content can use stored MIME type
and an `inline` disposition; those paths do not set `nosniff`.

An external reporter can supply untrusted content even when the viewer is on a
private network. The P1 content and redirect defenses are independent of human
sign-in and should not wait for TP-03. Until human authorization exists, neither
a private network nor possession of a storage key proves an individual is
permitted to access an artifact; record that remaining gap in TP-16.

## Work and acceptance criteria

1. **P1:** Constrain storage to private objects and generated, validated keys.
   Prevent path traversal in every supported storage driver. Define an approved
   policy for existing external URIs and redirects; allow only approved
   destinations. If server-side fetching is retained or added, constrain schemes,
   resolved destinations, and every redirect to prevent SSRF. Enforce
   cross-project key restrictions where trusted ownership is available; complete
   human retrieval authorization in criterion 4.
2. **P1:** Treat content as hostile: enforce approved types, sizes,
   archive/decompression limits, and malware scanning or quarantine policy.
   Ensure scanning failures follow the approved fail-closed policy; never execute
   uploads. Document reporter-side secret/personal-data redaction and define
   additional server-side rejection or quarantine rules without claiming
   arbitrary sensitive content can always be detected.
3. **P1:** Prevent active content, including HTML and SVG, from executing on the
   trusted application origin.
   Use safe download headers and/or a separately isolated credential-free origin
   while preserving approved screenshot/report workflows. Set correct
   `Content-Type`, safe `Content-Disposition`, and `X-Content-Type-Options:
   nosniff` where the browser receives content, not merely on a redirect.
   Safely encode filenames and verify the actual storage-delivered response.
4. **Future access:** Resolve an attachment to its authoritative project/run and
   check the caller's grant before returning bytes, metadata, or any
   redirect/signed URL. An arbitrary storage key or filename is not a grant.
   Do not reveal object existence across projects.
5. If signed URLs remain, authorize first, restrict each URL to one object and
   the approved short lifetime, provide safe object response metadata, and keep
   URLs out of logs. Document expiry/revocation limitations and the actual
   browser-to-storage network flow. A signed URL can be usable outside the
   private web boundary by anyone holding it; a private bucket alone is not
   per-user authorization. Until authorization is implemented, disable or
   restrict that distributed delivery path, or record the applicable approved
   exception. Disable or proxy delivery if the selected backend cannot satisfy
   these controls. Surface signing and storage failures explicitly rather than
   silently treating them as successful downloads.
6. Preserve supported local screenshot/report and attachment workflows when
   shared handlers change; do not impose the distributed identity infrastructure
   on local demos.

## Verification

- For the P1 milestone, test traversal, malicious filenames, active HTML/SVG,
  wrong types, oversized files, redirects, allowed remote fetch destinations,
  signed URL restrictions, and valid local artifact workflows. Verify response
  headers on inline, proxy, and storage-delivered paths, and demonstrate in a
  browser that active content does not execute on the application origin.
- For full authorization, add two-project tests for unauthorized lookup, guessed
  keys, cross-project access, and signed URL authorization, scope, and expiry.
  Record this separately from the P1 content-safety milestone.
- Run `go test ./pkg/api ./pkg/storage ./internal/repository/postgres`; exercise
  V-08 with synthetic files against the selected object store and browser path.
  Record scanning and object-store policy evidence without retaining payloads or
  signed URLs; do not claim V-08 passes before the required authorization exists.
