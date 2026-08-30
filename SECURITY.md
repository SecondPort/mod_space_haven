# Security Policy

## Supported versions

The latest release is the supported one. This is a small tool with a single
maintainer; fixes land on `main` and go out in the next tag.

| Version | Supported |
| --- | --- |
| latest release | yes |
| anything older | no — please update |

## What counts as a vulnerability here

This editor reads a file you already have and writes it back. It opens no
sockets, contacts no servers, and needs no credentials. So the realistic risks
are about the file and the machine, not the network:

- **Save corruption or data loss** — a path where the editor writes a broken
  save, or destroys the backup it promised to keep.
- **Path traversal** — a crafted `SPACEHAVEN_SAVEGAMES_DIR`, slot name, or save
  path that makes the editor read or write outside the savegames folder.
- **Crashes on malicious input** — a save file that panics the scanner, or drives
  it into unbounded memory or an endless loop.
- **Leaking local information** — anything that puts file paths or save contents
  somewhere the user did not ask for.
- **Supply chain** — a compromised dependency or release artifact.

A save file that simply fails to open with a clear error message is a bug, not a
vulnerability. Please report those as a normal issue.

## Reporting

**Do not open a public issue.**

Report privately through GitHub Security Advisories:

1. Go to the [Security tab](https://github.com/SecondPort/mod_space_haven/security/advisories/new).
2. Open a draft advisory.

Or email **lucasg.antigravity@gmail.com** with `SECURITY` in the subject.

Please include:

- what the problem is, and what an attacker gets out of it
- the smallest reproduction you can manage
- the version (`modhaven --version`), your OS, and your Go version if you built
  it yourself
- **no real save files.** Reduce the input to the few tags that trigger it.

## What to expect

| Stage | Timeline |
| --- | --- |
| Acknowledgement | within 7 days |
| Initial assessment | within 14 days |
| Fix or a plan, for a confirmed issue | within 30 days |

This is a spare-time project, so these are honest targets rather than a
guarantee. You will be told where things stand either way.

## Disclosure

Please give the fix a chance to ship before going public. Once a release is out,
say whatever you like — and you will be credited in the advisory unless you ask
not to be.

## For anyone using this tool

- Close the game before editing. It holds the save in memory and will overwrite
  your changes on its next write.
- Every write is preceded by a timestamped backup (`game.bak_YYYYMMDD_HHMMSS`)
  next to the save, and the new file is written to a temporary file and renamed
  into place. Keep those backups until you have loaded the edited save.
- Take binaries from the [releases page](https://github.com/SecondPort/mod_space_haven/releases)
  or build from source. Check the published `checksums.txt` against your
  download.
