# Windows Security Review

Review date: 2026-10-26

## Scope

This review covers the Windows-specific implementation in this repository,
with comparisons to Linux and macOS where their platform abstraction code
provides a direct comparison. It focuses on local peer authentication,
authorization, and named socket access. External Hyphae and shell
implementations, installers, service configuration, and deployed file-system
ACLs were outside the available source scope.

## Findings

| # | Severity | Affected code | Summary |
|---|----------|---------------|---------|
| 1 | High | `spored/internal/pal/windows.go`, `spored/internal/nodes/node.go`, `spored/internal/bus/responses.go` | A Windows client that cannot have its process image queried is provisionally activated before fallback verification completes. Its responses are accepted without binding them to an authenticated node identity, allowing it to forge pending binary-hash or confirmation responses and impersonate a permitted node. |
| 2 | High | `spored/internal/nodes/binary.go`, `spored/internal/pal/windows.go` | A configured `trust: developer` binary is treated as verified without validating the connecting process. Because the Windows endpoint permits authenticated users to connect, another local account can claim that node identity and use its granted permissions. |

### 1. Provisional peer activation permits response forgery

On Windows, access denial while opening a peer process is treated as a
`RequiresUserSpace` condition. The connection is activated while binary hash
verification is pending. The endpoint permits `Authenticated Users` to
connect, and response messages are processed before verification. Since
responses are matched only by request handle rather than the authorized
responder identity, an unverified peer can forge the expected binary-hash
response. It can then claim the target node's permissions and forge subsequent
confirmation responses.

Linux and macOS do not activate a peer before their respective executable path
lookups complete, making this specific provisional-activation path Windows
specific.

Recommended remediation:

1. Do not activate or process traffic from a connection until peer
   verification succeeds.
2. Bind every pending request to its authorized responder connection or node
   identity, and reject responses from other peers.
3. Limit endpoint access to the intended user or service principals rather
   than all authenticated users.

### 2. Developer-trusted nodes have no peer authentication

`trust: developer` causes a configured existing binary to be considered
verified without checking the connecting process's PID, path, or checksum.
Another authenticated local Windows user can therefore present the developer
node's ID and inherit its permission grants.

This is not exclusively a Windows issue: the Linux and macOS socket
implementations use mode `0777`, which provides similarly broad local access.
The Windows ACL does not mitigate the issue because `Authenticated Users`
includes other local accounts.

Recommended remediation:

1. Verify the peer identity for developer-trusted nodes, including its
   installing user or SID where applicable.
2. Restrict endpoint access to that identity and avoid treating developer
   trust as authorization for arbitrary local principals to claim the node.

## Existing Windows Mitigations

- The Windows implementation obtains the peer PID from the kernel rather than
  accepting a client-supplied PID.
- When process-image lookup succeeds, the implementation validates the peer's
  executable path and SHA-256 checksum before registering permissions.
- The endpoint ACL gives `Authenticated Users` read/write access rather than
  full control, although that remains too broad for peer authentication.

## Assessment Limits

The available repository does not include Windows installer code, service
configuration, or deployed ACL inspection. Consequently, inherited ACLs for
the data directory, manifests, registries, and binaries were not assessed.
The external Hyphae and shell implementations were also unavailable, so their
independent authentication was not reviewed.
