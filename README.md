# Mode Haven

Python terminal save editor for Space Haven.

This project is meant to be usable from a local terminal on a regular computer, with a simple `.venv` setup and no external services.

## Quick start

### Option A: install dependencies with `requirements.txt`

```bash
git clone https://github.com/SecondPort/mode_space_haven.git
cd mode_space_haven
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
python main.py
```

CLI mode:

```bash
python main.py --cli
```

### Option B: install as a local command

```bash
git clone https://github.com/SecondPort/mode_space_haven.git
cd mode_space_haven
python3 -m venv .venv
source .venv/bin/activate
pip install -e .
mode-haven
```

CLI mode:

```bash
mode-haven --cli
```

## What it does

The editor opens an existing Space Haven save, creates a backup before writing, and lets you modify common game data from the terminal.

You can edit:

- credits
- cargo and resources
- character stats
- character skills
- character attributes
- character traits
- character names
- character loadout
- research progress

## Save folder detection

The app tries to detect the save directory automatically in this order:

1. `SPACEHAVEN_SAVEGAMES_DIR`
2. `~/.local/share/Steam/steamapps/common/SpaceHaven/savegames`
3. `~/.steam/steam/steamapps/common/SpaceHaven/savegames`
4. `~/.var/app/com.valvesoftware.Steam/.local/share/Steam/steamapps/common/SpaceHaven/savegames`
5. `/mnt/*/Steam/steamapps/common/SpaceHaven/savegames`

If your saves live somewhere else:

```bash
export SPACEHAVEN_SAVEGAMES_DIR="/path/to/your/savegames"
python main.py
```

## Run modes

### TUI mode

Default mode. Uses Textual for a richer terminal interface.

```bash
python main.py
```

### Classic CLI mode

Simple interactive terminal flow.

```bash
python main.py --cli
```

## Requirements

- Python 3.10 or newer
- a local terminal
- Space Haven save files on your machine

Direct dependency:

- `textual==8.2.7`

## Safety

- Close the game before editing saves.
- The editor creates a timestamped backup before writing changes.
- Backups use the format `game.bak_YYYYMMDD_HHMMSS`.
- This project edits existing saves only. It does not repair corrupted save files.

## Project structure

- `main.py` — launcher for TUI and CLI modes
- `editor.py` — classic terminal editor
- `editor_tui.py` — Textual terminal UI
- `requirements.txt` — dependency install for `.venv` workflow
- `pyproject.toml` — package metadata and console script definition

## Public repository status

I reviewed the repository to prepare it for public use.

Current public-safety status:

- no API keys are required
- no tokens are required
- no private keys are included
- no save files are included
- local agent metadata was removed from tracked files
- local backups and environment files are ignored in `.gitignore`

## What this repository does not include

- Space Haven game files
- personal save files
- cloud services
- server-side components

## Recommended local setup

```bash
python3 -m venv .venv
source .venv/bin/activate
pip install --upgrade pip
pip install -r requirements.txt
python main.py
```

## Notes

- If `textual` is missing, install dependencies again inside the active virtual environment.
- If the save folder is not detected, set `SPACEHAVEN_SAVEGAMES_DIR` manually.
- If you want the command `mode-haven`, install with `pip install -e .`.
