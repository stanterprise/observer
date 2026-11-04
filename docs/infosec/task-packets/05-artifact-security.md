# TP-05 - Authorize and Safely Deliver Artifacts

**Status:** Proposed; no owner assigned.
**Priority:** P0.
**Accountable roles:** Application and data owners.
**Findings:** 3 and 7.
**Controls:** IAM-02, ING-03/05, DATA-01/03/04/05, NET-19; test V-08.
**Prerequisites:** D-05/D-07; ownership and human authorization from
[TP-02](02-project-ownership.md) and
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

## Work and acceptance criteria

1. Resolve an attachment to its authoritative project/run and check the caller's
   grant before returning bytes, metadata, or any redirect/signed URL. An
   arbitrary storage key or filename is not a grant. Do not reveal object
   existence across projects.
2. Constrain storage to private objects and generated, validated keys. Prevent
   path traversal and cross-project key use in every supported storage driver.
   Define an approved policy for existing external URIs and redirects; allow
   only approved destinations. If server-side fetching is retained or added,
   constrain schemes, resolved destinations, and every redirect to prevent SSRF.
3. Treat content as hostile: enforce approved types, sizes, archive/decompression
   limits, and malware scanning or quarantine policy. Ensure scanning failures
   follow the approved fail-closed policy; never execute uploads. Document
   reporter-side secret/personal-data redaction and define additional server-side
   rejection or quarantine rules without claiming arbitrary sensitive content
   can always be detected.
4. Prevent active content from executing on the trusted application origin.
   Use safe download headers and/or a separately isolated credential-free origin
   while preserving approved screenshot/report workflows. Set correct
   `Content-Type`, safe `Content-Disposition`, and `X-Content-Type-Options:
   nosniff` where the browser receives content, not merely on a redirect.
   Safely encode filenames.
5. If signed URLs remain, authorize first, restrict each URL to one object and
   the approved short lifetime, provide safe object response metadata, and keep
   URLs out of logs. Document expiry/revocation limitations and the actual
   browser-to-storage network flow. Disable or proxy delivery if the selected
   backend cannot satisfy these controls. Surface signing and storage failures
   explicitly rather than silently treating them as successful downloads.

## Verification

- Add tests for two projects; unauthorized lookup, guessed keys, traversal,
  malicious filenames, active HTML/SVG, wrong types, oversized files, signed URL
  scope/expiry, redirects, and any allowed remote fetch/redirect destinations.
  Verify response headers on inline, proxy, and storage-delivered paths.
- Run `go test ./pkg/api ./pkg/storage ./internal/repository/postgres`; exercise
  V-08 with synthetic files against the selected object store and browser path.
  Record scanning and object-store policy evidence without retaining payloads or
  signed URLs.
