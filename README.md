# Mode Haven

Terminal-first save editor for **Space Haven** written in Python.

It lets you open an existing save, create a backup, and edit common data such as cargo, credits, characters, and research.

## Quick start

### 1. Clone the repository

```bash
git clone https://github.com/SecondPort/mode_space_haven.git
cd mode_space_haven
```

### 2. Create a virtual environment

```bash
python3 -m venv .venv
source .venv/bin/activate
```

### 3. Install the project

```bash
pip install -e .
```

### 4. Run it

Text UI mode:

```bash
mode-haven
```

Classic terminal mode:

```bash
mode-haven --cli
```

You can also run it directly with Python:

```bash
python main.py
python main.py --cli
```

## What it can edit

- Credits
- Cargo and resources
- Character stats
- Character skills
- Character attributes
- Character traits
- Character names
- Character loadout
- Research progress

## How save discovery works

The tool tries to find the save directory automatically.

Current built-in search paths are:

- `SPACEHAVEN_SAVEGAMES_DIR` if the environment variable is set
- `~/.local/share/Steam/steamapps/common/SpaceHaven/savegames`
- `~/.steam/steam/steamapps/common/SpaceHaven/savegames`
- `~/.var/app/com.valvesoftware.Steam/.local/share/Steam/steamapps/common/SpaceHaven/savegames`
- `/mnt/*/Steam/steamapps/common/SpaceHaven/savegames`

If your saves are somewhere else, set the environment variable before running the app:

```bash
export SPACEHAVEN_SAVEGAMES_DIR="/path/to/your/savegames"
mode-haven
```

## Safety

- Close the game before editing a save.
- The editor creates a timestamped backup before writing changes.
- Backup files use the format `game.bak_YYYYMMDD_HHMMSS`.

## Requirements

- Python 3.10+
- A terminal
- `textual` for the TUI mode, installed automatically by `pip install -e .`

## Project files

- `main.py` — launcher for TUI and CLI modes
- `editor.py` — classic terminal editor
- `editor_tui.py` — Textual-based terminal UI

## Known limitations

- Save auto-discovery is focused on the paths listed above.
- The project edits existing saves; it does not validate or repair corrupted save files.
- This repository does not ship any game assets or saves.

## Public repository note

This repository is intended to be safe to publish:

- no secrets are required to run it
- no API keys are used
- no save files are included
- local agent metadata is ignored from version control

If you plan to contribute, avoid committing personal save files, virtual environments, or backup files.
