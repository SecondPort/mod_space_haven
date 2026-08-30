<!--
Thanks for the pull request.

Title it as a Conventional Commit, e.g.
  fix(cargo): count containers behind a self-closing sibling
  feat(cli): add `set loadout`
-->

## What this changes

<!-- One or two sentences. What is different after this merges? -->

## Why

<!-- The problem being solved. Link the issue: "Closes #12". -->

Closes #

## How you verified it

<!--
Required. "It builds" is not verification.

Say what you ran and what you saw. For anything touching the save format, say
whether you tried it on a real save and whether the game loaded it afterwards.
-->

- [ ] `make check` passes (format, vet, tests)
- [ ] `make race` passes
- [ ] New tests cover the change, and they fail without it
- [ ] Tried against a real save, and the game loaded the result — or N/A because:

## Checklist

- [ ] One concern in this pull request. A fix and a refactor are two.
- [ ] Under roughly 400 changed lines, or split into a chain and explained below.
- [ ] Editing rules live in `internal/savegame`, not in `internal/tui` or
      `internal/cli`, so both front ends get them.
- [ ] Every save edit is a byte-range patch through `internal/xmldoc`. Nothing
      re-serializes the document.
- [ ] New user-facing strings are in `internal/tui/text.go`, in both Spanish and
      English.
- [ ] Any new reference-data ids say where they came from.
- [ ] Documentation updated if behaviour or flags changed.
- [ ] No real save files, personal paths, or screenshots of a real colony.

## Anything the reviewer should know

<!--
Trade-offs you made, alternatives you rejected, parts you are unsure about.
Saying "I am not sure about X" gets you a better review, not a worse one.
-->
