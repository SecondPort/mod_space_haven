# Contributing

Thanks for wanting to help. This is a save editor: a bug here costs somebody
their colony, so the bar for correctness is high and the rules below exist to
keep it that way.

## Before you start

- **Open an issue first** for anything beyond a typo or an obvious one-line fix.
  A short conversation about the approach saves you from writing code that gets
  turned down.
- Bugs go in the bug report template, ideas in the feature request template.
  Both ask for the things a maintainer needs to act; filling them in properly is
  the difference between a fix this week and a fix never.
- **Never attach a real save file** to an issue or a pull request. Saves contain
  your colony and your file paths. Describe the shape of the XML instead, or
  reduce it to a handful of tags in a fenced block.

## Getting set up

You need Go 1.24 or newer. Nothing else.

```bash
git clone https://github.com/SecondPort/mod_space_haven.git
cd mod_space_haven
make check      # format, vet, test — this must pass before you push
```

Useful targets:

| Command | What it does |
| --- | --- |
| `make build` | build into `bin/modhaven` |
| `make test` | run the suite |
| `make race` | run the suite under the race detector |
| `make cover` | write `coverage.html` |
| `make check` | format, vet and test — run this before pushing |
| `make release` | cross-compile every supported platform |

## How the code is laid out

```
cmd/modhaven          launcher: flags, then the interface or a command
internal/xmldoc       offset-preserving XML scanner and byte-range patch sets
internal/savegame     the domain: ship, cargo, crew, research — no file I/O
internal/catalog      embedded reference tables
internal/library      the filesystem: discovery, listing, backup, atomic write
internal/tui          the Bubble Tea interface
internal/cli          the scriptable commands
```

Three boundaries hold this together. Please do not cross them:

1. **`internal/savegame` never touches the filesystem.** It edits bytes it was
   handed. That is what lets the editing rules be tested against fixtures alone.
2. **`internal/library` never knows the editing rules.** It finds files, backs
   them up and writes them.
3. **The interface and the commands are two front ends over the same
   operations.** A capability added to one belongs in `internal/savegame` so the
   other gets it too. Logic that lives only in `internal/tui` or only in
   `internal/cli` is how the two drift apart.

## The rule that matters most

**A save is edited by replacing byte ranges. It is never re-serialized.**

Space Haven cares about attribute order, indentation and self-closing style. Any
decode/encode round trip silently rewrites all three and corrupts the file. So:

- Read through `internal/xmldoc`, which reports where each tag and attribute
  value lives.
- Express every change as an `xmldoc.Patch` against the **original** buffer and
  hand the whole batch to `PatchSet.Apply`.
- Do not reach for `encoding/xml`, a DOM, or string replacement over the whole
  document.

If a change makes an edited save differ from the original anywhere other than
the values you meant to change, it is a bug, even if the game still loads it.

## Tests

Tests come first here, and a pull request without them will be sent back.

- Write the failing test before the fix. It is the only proof the bug was real.
- Every package has a fixture at `testdata/sample_save.xml`. Extend it rather
  than inventing a second one — a shared fixture is how a change in one package
  reveals a break in another. Keep the copies in sync.
- Name tests after the behaviour, not the function:
  `TestCargoSeesContainersBehindASelfClosingSibling`, not `TestCargo2`.
- Assert on outcomes a person would notice. `internal/tui` is tested by driving
  the model through `Update` with key messages — no terminal required.
- Regressions get a permanent test. If a fixture had to grow a strange shape to
  reproduce the bug, say so in a comment so nobody "tidies" it away later.

Coverage sits around 80% per package. Please do not take it downwards.

## Style

- `gofmt` decides formatting. `make check` enforces it, and so does CI.
- Comments explain **why**, not what. The code already says what.
- Errors read like sentences and name what failed:
  `savegame: character %q has no skill sk=%d`.
- Code, identifiers, comments and documentation are in English.
- User-facing interface strings live in `internal/tui/text.go` and exist in both
  Spanish and English. Adding a string means adding both. Never inline a
  user-facing string in a layout function.
- Table cells rendered by `bubbles/table` must be plain text. The table clips
  cells by rune width without understanding ANSI, so a styled cell gets cut
  mid-escape and leaks the raw sequence into the view. Style the table through
  `table.Styles`, never the content.

## Commits and pull requests

Commits follow [Conventional Commits](https://www.conventionalcommits.org):

```
feat(cargo): add resources the ship is not carrying yet
fix(xmldoc): stop truncating an element at a nested tag of the same name
test(savegame): cover containers behind a self-closing sibling
docs(readme): document the scripting commands
chore(ci): run the suite on macOS
```

Types in use: `feat`, `fix`, `docs`, `test`, `refactor`, `perf`, `chore`, `ci`.
A breaking change gets a `!` (`feat!:`) and a `BREAKING CHANGE:` footer.

For the pull request itself:

- **One concern per pull request.** A fix and a refactor in the same branch are
  two pull requests.
- Keep it under roughly 400 changed lines where you can. Beyond that, split it
  into a chain — reviews get worse as they get longer, and so do the bugs that
  survive them.
- Fill in the template. The "how did you verify this" box is not optional.
- Tests and documentation ship with the code, not in a follow-up.
- CI must be green: `gofmt`, `go vet`, and the suite under `-race` on Linux,
  macOS and Windows.
- The maintainer is a required reviewer through `CODEOWNERS`. Expect questions;
  they are about the code, never about you.

## Reference data

The tables in `internal/catalog/data` come from the game's own files
(`library/haven`, `library/texts`, decompiled class constants). If you add
entries, say in the pull request where each id came from. An id nobody can trace
is a guess, and a guess in a save editor eventually costs somebody a colony.

Unknown ids are not a failure: the editor falls back to `Elemento #1234` on
purpose, so a save from a newer game version still opens. Keep that fallback
working.

## Reporting a vulnerability

Do not open an issue. See [SECURITY.md](SECURITY.md).
