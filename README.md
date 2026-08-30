# Mod Haven

[![CI](https://github.com/SecondPort/mod_space_haven/actions/workflows/ci.yml/badge.svg)](https://github.com/SecondPort/mod_space_haven/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/SecondPort/mod_space_haven.svg)](https://pkg.go.dev/github.com/SecondPort/mod_space_haven)
[![Go Report Card](https://goreportcard.com/badge/github.com/SecondPort/mod_space_haven)](https://goreportcard.com/report/github.com/SecondPort/mod_space_haven)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

Terminal save editor for Space Haven, written in Go. One binary, no runtime to
install, no services — it reads a save file on your machine, backs it up, and
writes it back.

```
 Space Haven — Editor de Save
 Nostromo  ·  Créditos: 999.999  ·  Investigación: 1/3  ·  Tripulación: 2
  1  Cargo          │  ID       Nombre                     Cantidad
  2  Armas          │ ─────────────────────────────────────────────
  3  Tripulación    │           ── COMIDA Y BEBIDA
  4  Investigación  │  16       Agua                       40
                    │           ── MATERIALES Y RECURSOS
                    │  157      Metales básicos            150
```

## Install

### From source

```bash
git clone https://github.com/SecondPort/mod_space_haven.git
cd mod_space_haven
make build      # produces bin/modhaven
```

Or install it onto your `PATH`:

```bash
go install github.com/SecondPort/mod_space_haven/cmd/modhaven@latest
```

Requires Go 1.24 or newer to build. The finished binary requires nothing.

### Prebuilt binaries

Grab one from the [releases page](https://github.com/SecondPort/mod_space_haven/releases)
and check it against the published `checksums.txt`. Or build them all yourself:

```bash
make release    # dist/modhaven_{linux,darwin,windows}_{amd64,arm64}
```

## Use

Start the terminal interface:

```bash
modhaven
```

It finds your saves, lists them, and opens the one you pick.

| Key | Action |
| --- | --- |
| `1`–`4`, `tab` | switch section |
| `↑` `↓` | move through a list |
| `enter` | edit the selected row |
| `a` | add a resource to cargo (searchable) |
| `c` | edit credits |
| `←` `→` | switch pane in Weapons |
| `A` / `R` | complete / reset all research (Research section) |
| `n` | rename a crew member (crew detail) |
| `H` `M` `R` | edit health, mood, rest (crew detail) |
| `s` | write the save |
| `esc` | back |
| `q` | quit (asks if there is unsaved work) |

Nothing reaches disk until you press `s`.

### Scripting

Every edit the interface performs is also a command, so a save can be adjusted
from a script or a shell one-liner:

```bash
modhaven list                        # the saves the editor can see
modhaven info                        # summary of the most recent save
modhaven cargo                       # the player ship's cargo
modhaven crew                        # crew with stats and skills
modhaven research                    # research progress

modhaven set credits 250000
modhaven set cargo 157 5000          # base metals
modhaven set research 2532 done
modhaven set research all done
```

Commands act on the most recently written save unless you pass `--save`.

### Flags

| Flag | Meaning |
| --- | --- |
| `--lang es\|en` | interface language (default `es`) |
| `--dir PATH` | savegames folder, overriding detection |
| `--save PATH` | open one save directly, e.g. `.../slot1/save/game` |
| `--version` | print the version |

## What it edits

- credits
- cargo and resources, including items the ship is not carrying yet
- weapons, armour and survival gear stock
- crew stats, skills, attributes, traits and names
- research progress, one technology at a time or all at once

Crew loadouts are shown but not editable: characters equip themselves from
cargo, so stocking the hold is what actually changes what they carry.

## Finding your saves

The editor checks, in order:

1. `$SPACEHAVEN_SAVEGAMES_DIR`
2. the Steam library for your platform
   - Linux: `~/.local/share/Steam`, `~/.steam/steam`, the Flatpak path
   - macOS: `~/Library/Application Support/Steam`
   - Windows: `C:\Program Files (x86)\Steam`, `C:\Program Files\Steam`, `%USERPROFILE%\Steam`
3. Steam libraries on other drives (`/mnt`, `/media`, `/run/media`, `/Volumes`, `D:`–`F:`)

If yours lives elsewhere:

```bash
export SPACEHAVEN_SAVEGAMES_DIR="/path/to/your/savegames"
```

## Safety

- Close the game before editing. The game holds the save in memory and will
  overwrite your changes on its next write.
- Every write is preceded by a timestamped backup, `game.bak_YYYYMMDD_HHMMSS`,
  next to the save.
- The new save is written to a temporary file and renamed into place, so an
  interrupted write cannot leave a half-finished save behind.
- Edits are byte-range replacements against the original document: an edited
  save differs from the original only in the attribute values you changed. The
  editor never re-serializes the XML, which is what would silently reorder
  attributes or rewrite self-closing tags and break the file.
- This edits existing saves. It does not repair corrupted ones.

## How it is put together

```
cmd/modhaven          launcher: flags, then the interface or a command
internal/xmldoc       offset-preserving XML scanner and byte-range patch sets
internal/savegame     the domain: ship, cargo, crew, research — no file I/O
internal/catalog      embedded reference tables (resources, skills, traits, techs)
internal/library      the filesystem: discovery, listing, backup, atomic write
internal/tui          the Bubble Tea interface
internal/cli          the scriptable commands
```

`internal/savegame` knows the rules and touches no disk, so it is tested against
fixtures alone. `internal/library` knows the disk and none of the rules. The
interface and the commands are two front ends over the same operations, which is
why they cannot drift apart.

The reference tables were extracted from the game's own files (`library/haven`,
`library/texts`, and decompiled class constants) and are compiled into the
binary with `go:embed`.

## Contributing

Contributions are welcome, and the bar is high on purpose: a bug in a save
editor costs somebody their colony.

```bash
make check      # format, vet, test — run this before you push
make test
make race
make cover      # writes coverage.html
make help
```

Start with **[CONTRIBUTING.md](CONTRIBUTING.md)**. It covers the three
architectural boundaries, the byte-range editing rule that everything else hangs
off, and what a pull request needs to get merged.

- Bugs, ideas and save-format mismatches go through the
  [issue templates](https://github.com/SecondPort/mod_space_haven/issues/new/choose).
- Never attach a real save file to an issue or a pull request. A handful of tags
  is enough, and saves carry your colony and your file paths.
- Behaviour in project spaces is covered by the
  [Code of Conduct](CODE_OF_CONDUCT.md).
- Security problems go through a
  [private advisory](https://github.com/SecondPort/mod_space_haven/security/advisories/new),
  never a public issue. See [SECURITY.md](SECURITY.md).

## Repository status

- no API keys, tokens or private keys
- no save files or game files
- no cloud or server-side components
- local backups and tooling metadata are gitignored

## License

[MIT](LICENSE) © SecondPort
