
<!-- PREAMBLE BEGIN -->
> For project overview, contributing guidelines, code of conduct, security policy,
> and licensing information, see the
> [sporeos-dev organization README](https://github.com/sporeos-dev/.github).

> [!WARNING]
> **Alpha Software: Use at Your Own Risk.**
> Spore OS is currently in an alpha state.
> It is under active development, breaking changes are expected frequently.
> Do not use in production environments.
<!-- PREAMBLE FIN -->

# spore-os

A local-first IPC protocol. Applications expose capabilities; Spore OS routes calls.

> Capabilities work like functions; 
> apps like objects; 
> a PC like a single unit.

**Status:** Active development. All parts are subject to change.

---

## How it works

`spored` is a hub daemon. Applications connect to it via a Unix socket, declare a manifest describing what they expose, and route calls to each other using human-readable dot-notation subjects.

```
# open a file picker dialog
file.open ~h1

# the hub routes to spore-dialog; response comes back bound to the handle
~h1:file.open path="/home/user/notes.txt" ok capture=dev.sporeos.dialog
```

[Wire format and full specification →](https://github.com/sporeos-dev/spore-os-protocol)

---

## Development

Build, test, and deploy instructions: [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md).

---

## License

Apache-2.0 — see [LICENSE](LICENSE).
