#!/usr/bin/env python3
"""Launcher for the Space Haven save editor."""

from __future__ import annotations

import argparse
import sys


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Launch the Space Haven save editor in TUI or CLI mode."
    )
    parser.add_argument(
        "--cli",
        action="store_true",
        help="run the classic interactive terminal mode",
    )
    parser.add_argument(
        "--tui",
        action="store_true",
        help="force Textual TUI mode",
    )
    args = parser.parse_args()

    if args.cli and args.tui:
        parser.error("Choose only one mode: --cli or --tui")

    if args.cli:
        from editor import main as cli_main

        cli_main()
        return

    try:
        from editor_tui import main as tui_main

        tui_main()
    except ModuleNotFoundError as exc:
        if exc.name != "textual" or args.tui:
            raise

        print(
            "Textual is not installed. Falling back to CLI mode.\n"
            "Install full dependencies with: pip install -e .",
            file=sys.stderr,
        )
        from editor import main as cli_main

        cli_main()


if __name__ == "__main__":
    main()
