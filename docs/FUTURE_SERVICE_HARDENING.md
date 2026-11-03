# Future service hardening

This is deferred design work. It is not required for the current all-local,
machine-global model, but it should be completed before treating `spored` as a
security boundary on shared or hostile machines.

## Goal

Run the daemon with the minimum operating-system privileges and filesystem
access required to operate. This limits damage if `spored`, a privileged node,
or its installation state is compromised.

This protects the **daemon**. It must not be applied indiscriminately to
user-space nodes: those need their own explicit sandbox and permission model.

## Linux: systemd

Run as a dedicated, non-login service account with no supplementary groups.
The service unit should:

- prevent privilege gain (`NoNewPrivileges=true`);
- make the host filesystem read-only by default (`ProtectSystem=strict`);
- grant write access only to Spore data and log directories;
- isolate temporary files and devices (`PrivateTmp=true`, `PrivateDevices=true`);
- remove unneeded Linux capabilities and network address families;
- apply sensible process, memory, and file-descriptor resource limits.

Validate each restriction against the intended node-launch and IPC model before
enabling it. Linux hardening belongs primarily in the installed systemd unit,
not in daemon application code.

## macOS: launchd and platform controls

Use a dedicated non-login account, restrictive ownership and modes on Spore
state, and a minimal launchd service definition. Launchd does not automatically
provide a full sandbox for a system daemon. If a macOS sandbox profile or other
system policy is used later, explicitly allow only required paths and IPC.

Code-signing and hardened-runtime policy are separate from service sandboxing;
they can strengthen publisher and runtime-code assurance later.

## Windows: service identity and ACLs

When Windows support is implemented, run under a dedicated low-privilege
service identity (or a restricted service token). Use ACLs so only that identity
can modify registry, permissions, logs, and service configuration. Prefer a
Service SID and restrict the service's filesystem, registry, network, and
process privileges. Use a named-pipe ACL rather than a globally accessible
endpoint.

## Deferred design decisions

Before enabling this work, define:

1. Which filesystem paths the daemon may read and write.
2. Whether the daemon may launch any processes, and under which user identity.
3. Which local users may connect to the IPC endpoint.
4. Whether containers or sandboxes may host nodes.
5. The recovery/upgrade procedure when a restrictive policy blocks startup.
