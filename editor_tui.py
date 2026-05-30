#!/usr/bin/env python3
"""Space Haven Save Editor — Textual TUI."""
from __future__ import annotations

from pathlib import Path

from textual import on
from textual.app import App, ComposeResult
from textual.binding import Binding
from textual.containers import Horizontal, ScrollableContainer, Vertical
from textual.screen import ModalScreen, Screen
from textual.widgets import (
    Button,
    ContentSwitcher,
    DataTable,
    Footer,
    Header,
    Input,
    Label,
    ListItem,
    ListView,
    Static,
)

from editor import (
    ARMAS_IDS,
    ATRIBUTOS,
    COMIDA_IDS,
    COMBUSTIBLE_IDS,
    EQUIPO_IDS,
    HABILIDADES,
    MEDICO_IDS,
    RASGOS,
    RECURSOS,
    TECNOLOGIAS,
    add_char_trait,
    backup_save,
    complete_tech,
    find_character_bounds,
    find_player_characters,
    find_player_ship_bounds,
    get_char_attributes,
    get_char_loadout,
    get_char_skills,
    get_char_stat,
    get_char_traits,
    get_credits,
    incomplete_tech,
    list_saves,
    load_save,
    nombre,
    parse_inventory,
    parse_research,
    remove_char_trait,
    SAVEGAMES_DIR,
    set_char_attribute,
    set_char_name,
    set_char_skill,
    set_char_stat,
    set_credits,
    set_resource,
)

ARMAS_PURAS: set[int] = ARMAS_IDS - {3968, 3969, 3975}
ACCESORIOS_ARMAS: set[int] = {3968, 3969, 3975}
ARMADURAS_IDS: set[int] = {3383, 3384}
HEADGEAR_IDS: set[int] = {481, 488}
SUPERVIVENCIA_IDS: set[int] = {3387, 3388, 3630, 4065}
DEFENSA_IDS: set[int] = ARMADURAS_IDS | HEADGEAR_IDS | SUPERVIVENCIA_IDS


def _slot_name(eid: int) -> str:
    return nombre(eid) if eid != 0 else "—"


# ─────────────────────────────────────────────────────────────────────────────
# Modals
# ─────────────────────────────────────────────────────────────────────────────


class EditValueModal(ModalScreen):
    DEFAULT_CSS = """
    EditValueModal { align: center middle; }
    #em-box {
        width: 54; height: auto;
        background: $surface; border: solid $primary; padding: 1 2;
    }
    #em-btns { margin-top: 1; height: auto; align-horizontal: center; }
    #em-btns Button { margin: 0 1; }
    """

    def __init__(self, item_name: str, current: int) -> None:
        super().__init__()
        self._name = item_name
        self._current = current

    def compose(self) -> ComposeResult:
        with Vertical(id="em-box"):
            yield Label(f"[bold]{self._name}[/bold]")
            yield Label(f"Actual: [cyan]{self._current:,}[/cyan]")
            yield Input(value=str(self._current), id="em-val", type="integer")
            with Horizontal(id="em-btns"):
                yield Button("Confirmar", variant="primary", id="em-ok")
                yield Button("Cancelar", id="em-cancel")

    def on_mount(self) -> None:
        inp = self.query_one("#em-val", Input)
        inp.focus()
        inp.action_select_all()

    @on(Button.Pressed, "#em-ok")
    def _ok(self) -> None:
        try:
            v = int(self.query_one("#em-val", Input).value)
            self.dismiss(v if v >= 0 else None)
        except ValueError:
            self.dismiss(None)

    @on(Button.Pressed, "#em-cancel")
    def _cancel(self) -> None:
        self.dismiss(None)

    def on_input_submitted(self) -> None:
        self._ok()


class ConfirmModal(ModalScreen):
    DEFAULT_CSS = """
    ConfirmModal { align: center middle; }
    #cf-box {
        width: 62; height: auto;
        background: $surface; border: solid $warning; padding: 1 2;
    }
    #cf-btns { margin-top: 1; height: auto; align-horizontal: center; }
    #cf-btns Button { margin: 0 1; }
    """

    def __init__(self, message: str) -> None:
        super().__init__()
        self._msg = message

    def compose(self) -> ComposeResult:
        with Vertical(id="cf-box"):
            yield Label(self._msg)
            with Horizontal(id="cf-btns"):
                yield Button("Sí, guardar", variant="warning", id="cf-yes")
                yield Button("Descartar cambios", variant="error", id="cf-no")
                yield Button("Cancelar", id="cf-cancel")

    @on(Button.Pressed, "#cf-yes")
    def _yes(self) -> None:
        self.dismiss("save")

    @on(Button.Pressed, "#cf-no")
    def _no(self) -> None:
        self.dismiss("discard")

    @on(Button.Pressed, "#cf-cancel")
    def _cancel(self) -> None:
        self.dismiss("cancel")


# ─────────────────────────────────────────────────────────────────────────────
# Save selection screen
# ─────────────────────────────────────────────────────────────────────────────


class SaveSelectScreen(Screen):
    DEFAULT_CSS = """
    SaveSelectScreen { align: center middle; }
    #ss-box {
        width: 74; height: auto;
        background: $surface; border: solid $primary; padding: 1 2;
    }
    #ss-list { height: auto; max-height: 16; }
    .ss-note { color: $text-muted; margin-top: 1; }
    """
    BINDINGS = [Binding("q", "quit", "Salir")]

    def __init__(self) -> None:
        super().__init__()
        self._saves: list[tuple[Path, str, str]] = []

    def compose(self) -> ComposeResult:
        with Vertical(id="ss-box"):
            yield Label("[bold]Space Haven — Editor de Save[/bold]")
            yield Label("[yellow]IMPORTANTE: Cerrá el juego antes de editar.[/yellow]")
            yield Static("")
            yield Label("Seleccioná un save:")
            yield ListView(id="ss-list")
            yield Label("Enter: seleccionar   q: salir", classes="ss-note")

    def on_mount(self) -> None:
        self._saves = list_saves(SAVEGAMES_DIR)
        lv = self.query_one("#ss-list", ListView)
        if not self._saves:
            lv.append(ListItem(Label("(No se encontraron saves)")))
        else:
            for i, (path, sname, date) in enumerate(self._saves):
                slot = path.parent.parent.name
                label = f"{slot:<12}  {sname:<26}  {date}"
                lv.append(ListItem(Label(label), id=f"sv-{i}"))
        lv.focus()

    @on(ListView.Selected)
    def _selected(self, event: ListView.Selected) -> None:
        if not self._saves:
            return
        self.dismiss(self._saves[event.index][0])

    def action_quit(self) -> None:
        self.dismiss(None)


# ─────────────────────────────────────────────────────────────────────────────
# Content panels
# ─────────────────────────────────────────────────────────────────────────────


class CargoPanel(Static):
    DEFAULT_CSS = """
    CargoPanel {
        height: 1fr;
        padding: 0;
    }
    CargoPanel DataTable { height: 1fr; }
    """

    def compose(self) -> ComposeResult:
        with ScrollableContainer():
            yield DataTable(id="cargo-tbl", cursor_type="row", zebra_stripes=True)

    def on_mount(self) -> None:
        t = self.query_one("#cargo-tbl", DataTable)
        t.add_column("ID", width=7)
        t.add_column("Nombre", width=38)
        t.add_column("Cantidad", width=12)
        self.refresh_data()

    def refresh_data(self) -> None:
        app: SpaceHavenEditorApp = self.app  # type: ignore[assignment]
        inv = app.inventario
        t = self.query_one("#cargo-tbl", DataTable)
        t.clear()

        known = COMIDA_IDS | COMBUSTIBLE_IDS | MEDICO_IDS | ARMAS_IDS | EQUIPO_IDS
        categories: list[tuple[str, dict[int, int]]] = [
            ("COMIDA / BEBIDA", {k: inv.get(k, 0) for k in COMIDA_IDS if k in inv}),
            ("COMBUSTIBLE", {k: inv.get(k, 0) for k in COMBUSTIBLE_IDS if k in inv}),
            ("MÉDICO / CONSUMIBLES", {k: inv.get(k, 0) for k in MEDICO_IDS if k in inv}),
            ("ARMAS Y MUNICIÓN", {k: inv.get(k, 0) for k in ARMAS_IDS if k in inv}),
            ("EQUIPO Y HERRAMIENTAS", {k: inv.get(k, 0) for k in EQUIPO_IDS if k in inv}),
            ("MATERIALES Y RECURSOS", {k: v for k, v in inv.items() if k not in known}),
        ]
        for cat, items in categories:
            if not items:
                continue
            t.add_row(f"── {cat}", "", "")
            for eid, qty in sorted(items.items(), key=lambda x: nombre(x[0])):
                t.add_row(str(eid), nombre(eid), f"{qty:,}", key=f"c-{eid}")

    @on(DataTable.RowSelected, "#cargo-tbl")
    def _row(self, event: DataTable.RowSelected) -> None:
        key = str(event.row_key.value or "")
        if not key.startswith("c-"):
            return
        eid = int(key[2:])
        cur = self.app.inventario.get(eid, 0)  # type: ignore[union-attr]
        self.app.push_screen(  # type: ignore[union-attr]
            EditValueModal(nombre(eid), cur),
            lambda v: self._apply(eid, cur, v),
        )

    def _apply(self, eid: int, old: int, new_val: int | None) -> None:
        if new_val is None:
            return
        app: SpaceHavenEditorApp = self.app  # type: ignore[assignment]
        new_content, inserted = set_resource(
            app.content, app.ship_start, app.ship_end, eid, new_val,
        )
        app.apply_edit(new_content)
        action = "insertado" if inserted else "actualizado"
        app.notify(f"{nombre(eid)} {action}: {old:,} → {new_val:,}")
        self.refresh_data()


class WeaponsPanel(Static):
    """Weapons & defense panel: cargo quantities + character loadouts."""

    DEFAULT_CSS = """
    WeaponsPanel { height: 1fr; layout: vertical; }
    #wp-top { height: 1fr; }
    #wp-weapons { width: 1fr; border-right: solid $primary; }
    #wp-defense { width: 1fr; }
    #wp-loadouts {
        height: 14;
        border-top: solid $primary;
    }
    .wp-title {
        background: $primary;
        color: $text;
        text-style: bold;
        padding: 0 1;
    }
    WeaponsPanel DataTable { height: 1fr; }
    """

    def compose(self) -> ComposeResult:
        with Horizontal(id="wp-top"):
            with Vertical(id="wp-weapons"):
                yield Static(" ARMAS EN CARGO", classes="wp-title")
                yield DataTable(id="wp-wtbl", cursor_type="row", zebra_stripes=True)
            with Vertical(id="wp-defense"):
                yield Static(" ARMADURAS Y EQUIPO DE DEFENSA", classes="wp-title")
                yield DataTable(id="wp-dtbl", cursor_type="row", zebra_stripes=True)
        with Vertical(id="wp-loadouts"):
            yield Static(" LOADOUTS DE TRIPULACIÓN  [dim](solo lectura — los personajes auto-equipan del cargo)[/dim]", classes="wp-title")
            yield DataTable(id="wp-ltbl", cursor_type="row", show_cursor=False, zebra_stripes=True)

    def on_mount(self) -> None:
        # Weapons table columns
        wt = self.query_one("#wp-wtbl", DataTable)
        wt.add_column("ID", width=6)
        wt.add_column("Arma", width=34)
        wt.add_column("En Cargo", width=10)

        # Defense table columns
        dt = self.query_one("#wp-dtbl", DataTable)
        dt.add_column("ID", width=6)
        dt.add_column("Item", width=32)
        dt.add_column("En Cargo", width=10)

        # Loadouts table columns
        lt = self.query_one("#wp-ltbl", DataTable)
        lt.add_column("Personaje", width=22)
        lt.add_column("Primaria", width=22)
        lt.add_column("Secundaria", width=22)
        lt.add_column("Armadura", width=22)
        lt.add_column("Casco", width=18)

        self.refresh_data()

    def refresh_data(self) -> None:
        app: SpaceHavenEditorApp = self.app  # type: ignore[assignment]
        inv = app.inventario

        # --- Weapons ---
        wt = self.query_one("#wp-wtbl", DataTable)
        wt.clear()
        armas_sorted = sorted(ARMAS_PURAS, key=lambda e: nombre(e))
        for eid in armas_sorted:
            qty = inv.get(eid, 0)
            qty_str = f"[green]{qty:,}[/green]" if qty > 0 else "[dim]0[/dim]"
            wt.add_row(str(eid), nombre(eid), qty_str, key=f"w-{eid}")

        accs_sorted = sorted(ACCESORIOS_ARMAS, key=lambda e: nombre(e))
        if accs_sorted:
            wt.add_row("── ACCESORIOS", "", "")
            for eid in accs_sorted:
                qty = inv.get(eid, 0)
                qty_str = f"[green]{qty:,}[/green]" if qty > 0 else "[dim]0[/dim]"
                wt.add_row(str(eid), nombre(eid), qty_str, key=f"w-{eid}")

        # --- Defense ---
        dt = self.query_one("#wp-dtbl", DataTable)
        dt.clear()

        def _def_section(title: str, ids: set[int]) -> None:
            dt.add_row(f"── {title}", "", "")
            for eid in sorted(ids, key=lambda e: nombre(e)):
                qty = inv.get(eid, 0)
                qty_str = f"[green]{qty:,}[/green]" if qty > 0 else "[dim]0[/dim]"
                dt.add_row(str(eid), nombre(eid), qty_str, key=f"d-{eid}")

        _def_section("ARMADURAS", ARMADURAS_IDS)
        _def_section("CASCO / HEADGEAR", HEADGEAR_IDS)
        _def_section("EQUIPO DE SUPERVIVENCIA", SUPERVIVENCIA_IDS)

        # --- Loadouts ---
        lt = self.query_one("#wp-ltbl", DataTable)
        lt.clear()
        for char in app.characters:
            ent_id = char["entId"]
            c_start, c_end = find_character_bounds(app.content, ent_id)
            if c_start == -1:
                continue
            char_block = app.content[c_start:c_end]
            lo = get_char_loadout(char_block)
            full_name = f"{char['name']} {char['lname']}"
            lt.add_row(
                full_name,
                _slot_name(lo.get("primary", 0)),
                _slot_name(lo.get("secondary", 0)),
                _slot_name(lo.get("armor", 0)),
                _slot_name(lo.get("headgear", 0)),
            )

    @on(DataTable.RowSelected, "#wp-wtbl")
    def _weapon_row(self, event: DataTable.RowSelected) -> None:
        key = str(event.row_key.value or "")
        if not key.startswith("w-"):
            return
        eid = int(key[2:])
        cur = self.app.inventario.get(eid, 0)  # type: ignore[union-attr]
        self.app.push_screen(  # type: ignore[union-attr]
            EditValueModal(nombre(eid), cur),
            lambda v: self._apply(eid, cur, v),
        )

    @on(DataTable.RowSelected, "#wp-dtbl")
    def _defense_row(self, event: DataTable.RowSelected) -> None:
        key = str(event.row_key.value or "")
        if not key.startswith("d-"):
            return
        eid = int(key[2:])
        cur = self.app.inventario.get(eid, 0)  # type: ignore[union-attr]
        self.app.push_screen(  # type: ignore[union-attr]
            EditValueModal(nombre(eid), cur),
            lambda v: self._apply(eid, cur, v),
        )

    def _apply(self, eid: int, old: int, new_val: int | None) -> None:
        if new_val is None:
            return
        app: SpaceHavenEditorApp = self.app  # type: ignore[assignment]
        new_content, inserted = set_resource(
            app.content, app.ship_start, app.ship_end, eid, new_val,
        )
        app.apply_edit(new_content)
        action = "insertado" if inserted else "actualizado"
        app.notify(f"{nombre(eid)} {action}: {old:,} → {new_val:,}")
        self.refresh_data()


class CharactersPanel(Static):
    DEFAULT_CSS = """
    CharactersPanel { height: 1fr; }
    CharactersPanel DataTable { height: 1fr; }
    """

    def compose(self) -> ComposeResult:
        yield DataTable(id="ch-tbl", cursor_type="row", zebra_stripes=True)

    def on_mount(self) -> None:
        t = self.query_one("#ch-tbl", DataTable)
        t.add_column("Personaje", width=26)
        t.add_column("Health", width=8)
        t.add_column("Mood", width=8)
        t.add_column("Rest", width=8)
        t.add_column("Skills", width=40)
        self.refresh_data()

    def refresh_data(self) -> None:
        app: SpaceHavenEditorApp = self.app  # type: ignore[assignment]
        t = self.query_one("#ch-tbl", DataTable)
        t.clear()
        for char in app.characters:
            ent_id = char["entId"]
            c_start, c_end = find_character_bounds(app.content, ent_id)
            if c_start == -1:
                continue
            block = app.content[c_start:c_end]
            hp = get_char_stat(block, "Health")
            mood = get_char_stat(block, "Mood")
            rest = get_char_stat(block, "Rest")
            skills = get_char_skills(block)
            top_skills = sorted(
                ((sk, d["level"]) for sk, d in skills.items() if d["level"] > 0),
                key=lambda x: -x[1],
            )[:4]
            skill_str = "  ".join(
                f"{HABILIDADES.get(sk, f'sk{sk}')}:{lvl}"
                for sk, lvl in top_skills
            )
            full_name = f"{char['name']} {char['lname']}"
            t.add_row(full_name, str(hp), str(mood), str(rest), skill_str, key=f"ch-{ent_id}")

    @on(DataTable.RowSelected, "#ch-tbl")
    def _row(self, event: DataTable.RowSelected) -> None:
        key = str(event.row_key.value or "")
        if not key.startswith("ch-"):
            return
        ent_id = key[3:]
        self.app.push_screen(  # type: ignore[union-attr]
            CharacterDetailScreen(ent_id),
            lambda _: self.refresh_data(),
        )


class ResearchPanel(Static):
    DEFAULT_CSS = """
    ResearchPanel { height: 1fr; layout: vertical; }
    #rp-controls { height: 3; padding: 0 1; }
    #rp-controls Button { margin-right: 1; }
    ResearchPanel DataTable { height: 1fr; }
    """

    def compose(self) -> ComposeResult:
        with Horizontal(id="rp-controls"):
            yield Button("Completar TODAS", variant="warning", id="rp-all")
            yield Button("Resetear TODAS", variant="error", id="rp-reset")
        yield DataTable(id="rp-tbl", cursor_type="row", zebra_stripes=True)

    def on_mount(self) -> None:
        t = self.query_one("#rp-tbl", DataTable)
        t.add_column("ID", width=7)
        t.add_column("Tecnología", width=42)
        t.add_column("Estado", width=14)
        self.refresh_data()

    def refresh_data(self) -> None:
        app: SpaceHavenEditorApp = self.app  # type: ignore[assignment]
        t = self.query_one("#rp-tbl", DataTable)
        t.clear()
        for tech_id in sorted(app.research.keys()):
            info = TECNOLOGIAS.get(tech_id)
            tname = info[0] if info else f"Tech #{tech_id}"
            done = app.research[tech_id]
            status = "[green]✓ Completa[/green]" if done else "[dim]Pendiente[/dim]"
            t.add_row(str(tech_id), tname, status, key=f"t-{tech_id}")

    @on(DataTable.RowSelected, "#rp-tbl")
    def _row(self, event: DataTable.RowSelected) -> None:
        key = str(event.row_key.value or "")
        if not key.startswith("t-"):
            return
        tech_id = int(key[2:])
        app: SpaceHavenEditorApp = self.app  # type: ignore[assignment]
        done = app.research.get(tech_id, False)
        if done:
            app.content = incomplete_tech(app.content, tech_id)
            app.research[tech_id] = False
            app.changed = True
            app.notify(f"Tech {tech_id} reseteada.")
        else:
            app.content = complete_tech(app.content, tech_id)
            app.research[tech_id] = True
            app.changed = True
            info = TECNOLOGIAS.get(tech_id)
            tname = info[0] if info else f"Tech #{tech_id}"
            app.notify(f"✓ {tname} completada.")
        self.refresh_data()

    @on(Button.Pressed, "#rp-all")
    def _complete_all(self) -> None:
        app: SpaceHavenEditorApp = self.app  # type: ignore[assignment]
        count = 0
        for tech_id, done in app.research.items():
            if not done:
                app.content = complete_tech(app.content, tech_id)
                app.research[tech_id] = True
                count += 1
        if count:
            app.changed = True
            app.notify(f"✓ {count} tecnologías completadas.")
            self.refresh_data()

    @on(Button.Pressed, "#rp-reset")
    def _reset_all(self) -> None:
        app: SpaceHavenEditorApp = self.app  # type: ignore[assignment]
        count = 0
        for tech_id, done in app.research.items():
            if done:
                app.content = incomplete_tech(app.content, tech_id)
                app.research[tech_id] = False
                count += 1
        if count:
            app.changed = True
            app.notify(f"{count} tecnologías reseteadas.")
            self.refresh_data()


# ─────────────────────────────────────────────────────────────────────────────
# Character detail screen
# ─────────────────────────────────────────────────────────────────────────────


class CharacterDetailScreen(Screen):
    DEFAULT_CSS = """
    CharacterDetailScreen { padding: 1 2; }
    #cd-title { text-style: bold; margin-bottom: 1; }
    #cd-stats-row { height: 3; margin-bottom: 1; }
    #cd-stats-row Button { margin-right: 1; }
    #cd-panels { height: 1fr; }
    #cd-skills { width: 1fr; border-right: solid $primary; }
    #cd-right { width: 1fr; }
    #cd-attrs { height: 1fr; border-bottom: solid $primary; }
    #cd-traits { height: 1fr; }
    .cd-sec {
        background: $primary; color: $text;
        text-style: bold; padding: 0 1;
    }
    CharacterDetailScreen DataTable { height: 1fr; }
    """
    BINDINGS = [
        Binding("escape", "go_back", "Volver"),
        Binding("s", "save_file", "Guardar"),
    ]

    def __init__(self, ent_id: str) -> None:
        super().__init__()
        self._ent_id = ent_id

    def _get_char_info(self) -> dict | None:
        app: SpaceHavenEditorApp = self.app  # type: ignore[assignment]
        return next((c for c in app.characters if c["entId"] == self._ent_id), None)

    def _get_block(self) -> tuple[str, int, int]:
        app: SpaceHavenEditorApp = self.app  # type: ignore[assignment]
        c_start, c_end = find_character_bounds(app.content, self._ent_id)
        return app.content[c_start:c_end], c_start, c_end

    def compose(self) -> ComposeResult:
        yield Header()
        char = self._get_char_info()
        full_name = f"{char['name']} {char['lname']}" if char else "Personaje"
        yield Label(f"[bold]{full_name}[/bold]", id="cd-title")
        with Horizontal(id="cd-stats-row"):
            yield Button("Editar Stats", id="cd-edit-stats")
            yield Button("Editar Nombre", id="cd-edit-name")
            yield Button("Volver", id="cd-back")
        with Horizontal(id="cd-panels"):
            with Vertical(id="cd-skills"):
                yield Static(" HABILIDADES  (Enter: editar)", classes="cd-sec")
                yield DataTable(id="cd-sk-tbl", cursor_type="row", zebra_stripes=True)
            with Vertical(id="cd-right"):
                with Vertical(id="cd-attrs"):
                    yield Static(" ATRIBUTOS  (Enter: editar)", classes="cd-sec")
                    yield DataTable(id="cd-at-tbl", cursor_type="row", zebra_stripes=True)
                with Vertical(id="cd-traits"):
                    yield Static(" RASGOS  (Enter: toggle)", classes="cd-sec")
                    yield DataTable(id="cd-tr-tbl", cursor_type="row", zebra_stripes=True)
        yield Footer()

    def on_mount(self) -> None:
        sk_t = self.query_one("#cd-sk-tbl", DataTable)
        sk_t.add_column("sk", width=5)
        sk_t.add_column("Habilidad", width=18)
        sk_t.add_column("Nivel", width=6)
        sk_t.add_column("Máx", width=6)

        at_t = self.query_one("#cd-at-tbl", DataTable)
        at_t.add_column("id", width=5)
        at_t.add_column("Atributo", width=16)
        at_t.add_column("Pts", width=6)

        tr_t = self.query_one("#cd-tr-tbl", DataTable)
        tr_t.add_column("", width=3)
        tr_t.add_column("id", width=6)
        tr_t.add_column("Rasgo", width=22)

        self.refresh_data()

    def refresh_data(self) -> None:
        block, _, _ = self._get_block()

        # Skills
        sk_t = self.query_one("#cd-sk-tbl", DataTable)
        sk_t.clear()
        skills = get_char_skills(block)
        for sk in sorted(skills.keys()):
            d = skills[sk]
            sk_t.add_row(str(sk), HABILIDADES.get(sk, f"sk{sk}"),
                         str(d["level"]), str(d["mxn"]), key=f"sk-{sk}")

        # Attributes
        at_t = self.query_one("#cd-at-tbl", DataTable)
        at_t.clear()
        attrs = get_char_attributes(block)
        for attr_id in sorted(attrs.keys()):
            at_t.add_row(str(attr_id), ATRIBUTOS.get(attr_id, f"id{attr_id}"),
                         str(attrs[attr_id]), key=f"at-{attr_id}")

        # Traits
        tr_t = self.query_one("#cd-tr-tbl", DataTable)
        tr_t.clear()
        current_traits = get_char_traits(block)
        for trait_id in sorted(RASGOS.keys(), key=lambda t: RASGOS[t]):
            marker = "[green]✓[/green]" if trait_id in current_traits else " "
            tr_t.add_row(marker, str(trait_id), RASGOS[trait_id], key=f"tr-{trait_id}")

    @on(DataTable.RowSelected, "#cd-sk-tbl")
    def _skill_row(self, event: DataTable.RowSelected) -> None:
        key = str(event.row_key.value or "")
        if not key.startswith("sk-"):
            return
        sk = int(key[3:])
        block, _, _ = self._get_block()
        skills = get_char_skills(block)
        cur_level = skills.get(sk, {}).get("level", 0)
        sk_name = HABILIDADES.get(sk, f"sk{sk}")
        self.app.push_screen(
            EditValueModal(f"{sk_name} (0-10)", cur_level),
            lambda v: self._apply_skill(sk, v),
        )

    def _apply_skill(self, sk: int, new_level: int | None) -> None:
        if new_level is None:
            return
        new_level = max(0, min(10, new_level))
        app: SpaceHavenEditorApp = self.app  # type: ignore[assignment]
        block, c_start, c_end = self._get_block()
        skills = get_char_skills(block)
        cur = skills.get(sk, {})
        new_mxn = max(new_level, cur.get("mxn", 0))
        new_block = set_char_skill(block, sk, new_level, new_mxn)
        app.content = app.content[:c_start] + new_block + app.content[c_end:]
        app.changed = True
        app._update_subtitle()
        app.notify(f"{HABILIDADES.get(sk, f'sk{sk}')}: {cur.get('level', 0)} → {new_level}")
        self.refresh_data()

    @on(DataTable.RowSelected, "#cd-at-tbl")
    def _attr_row(self, event: DataTable.RowSelected) -> None:
        key = str(event.row_key.value or "")
        if not key.startswith("at-"):
            return
        attr_id = int(key[3:])
        block, _, _ = self._get_block()
        attrs = get_char_attributes(block)
        cur = attrs.get(attr_id, 1)
        aname = ATRIBUTOS.get(attr_id, f"id{attr_id}")
        self.app.push_screen(
            EditValueModal(f"{aname} (1-8, total rec. 12)", cur),
            lambda v: self._apply_attr(attr_id, v),
        )

    def _apply_attr(self, attr_id: int, new_pts: int | None) -> None:
        if new_pts is None:
            return
        new_pts = max(1, min(8, new_pts))
        app: SpaceHavenEditorApp = self.app  # type: ignore[assignment]
        block, c_start, c_end = self._get_block()
        new_block = set_char_attribute(block, attr_id, new_pts)
        app.content = app.content[:c_start] + new_block + app.content[c_end:]
        app.changed = True
        app._update_subtitle()
        app.notify(f"{ATRIBUTOS.get(attr_id, f'id{attr_id}')}: → {new_pts}")
        self.refresh_data()

    @on(DataTable.RowSelected, "#cd-tr-tbl")
    def _trait_row(self, event: DataTable.RowSelected) -> None:
        key = str(event.row_key.value or "")
        if not key.startswith("tr-"):
            return
        trait_id = int(key[3:])
        app: SpaceHavenEditorApp = self.app  # type: ignore[assignment]
        block, c_start, c_end = self._get_block()
        traits = get_char_traits(block)
        tname = RASGOS.get(trait_id, str(trait_id))
        if trait_id in traits:
            new_block = remove_char_trait(block, trait_id)
            app.notify(f"Rasgo '{tname}' eliminado.")
        else:
            new_block = add_char_trait(block, trait_id)
            app.notify(f"Rasgo '{tname}' agregado.")
        app.content = app.content[:c_start] + new_block + app.content[c_end:]
        app.changed = True
        app._update_subtitle()
        self.refresh_data()

    @on(Button.Pressed, "#cd-edit-stats")
    def _edit_stats(self) -> None:
        block, _, _ = self._get_block()
        hp = get_char_stat(block, "Health")
        self.app.push_screen(
            EditValueModal("Health (0-200)", hp),
            lambda v: self._apply_stat("Health", hp, v),
        )

    def _apply_stat(self, stat: str, old: int, new_val: int | None) -> None:
        if new_val is None:
            return
        app: SpaceHavenEditorApp = self.app  # type: ignore[assignment]
        block, c_start, c_end = self._get_block()
        new_block = set_char_stat(block, stat, new_val)

        # Chain: after Health ask Mood then Rest
        if stat == "Health":
            app.content = app.content[:c_start] + new_block + app.content[c_end:]
            app.changed = True
            app._update_subtitle()
            app.notify(f"Health: {old} → {new_val}")
            mood = get_char_stat(new_block, "Mood")
            app.push_screen(
                EditValueModal("Mood (0-100)", mood),
                lambda v: self._apply_stat("Mood", mood, v),
            )
        elif stat == "Mood":
            app.content = app.content[:c_start] + new_block + app.content[c_end:]
            app.changed = True
            app._update_subtitle()
            app.notify(f"Mood: {old} → {new_val}")
            rest = get_char_stat(new_block, "Rest")
            app.push_screen(
                EditValueModal("Rest (0-200)", rest),
                lambda v: self._apply_stat("Rest", rest, v),
            )
        elif stat == "Rest":
            app.content = app.content[:c_start] + new_block + app.content[c_end:]
            app.changed = True
            app._update_subtitle()
            app.notify(f"Rest: {old} → {new_val}")
        self.refresh_data()

    @on(Button.Pressed, "#cd-edit-name")
    def _edit_name(self) -> None:
        self.app.notify(
            "Edición de nombre: usá el editor CLI (`python editor.py`).",
            severity="warning",
            timeout=4,
        )

    @on(Button.Pressed, "#cd-back")
    def action_go_back(self) -> None:
        self.dismiss(None)

    def action_save_file(self) -> None:
        app: SpaceHavenEditorApp = self.app  # type: ignore[assignment]
        app.action_save_file()


# ─────────────────────────────────────────────────────────────────────────────
# Main application
# ─────────────────────────────────────────────────────────────────────────────


class SpaceHavenEditorApp(App):
    TITLE = "Space Haven Save Editor"
    CSS = """
    SpaceHavenEditorApp {
        background: $surface;
    }
    #app-main {
        height: 1fr;
    }
    #sidebar {
        width: 22;
        background: $panel;
        border-right: solid $primary;
    }
    #sidebar-title {
        background: $primary;
        color: $text;
        text-style: bold;
        padding: 0 1;
        height: 1;
    }
    #nav-list {
        background: $panel;
        height: 1fr;
    }
    #content-area {
        width: 1fr;
        padding: 0 1;
    }
    ContentSwitcher {
        height: 1fr;
    }
    """
    BINDINGS = [
        Binding("s", "save_file", "Guardar"),
        Binding("q", "request_quit", "Salir"),
        Binding("c", "edit_credits", "Créditos"),
    ]

    def __init__(self) -> None:
        super().__init__()
        self.save_path: Path | None = None
        self.content: str = ""
        self.ship_start: int = 0
        self.ship_end: int = 0
        self.inventario: dict[int, int] = {}
        self.characters: list[dict] = []
        self.research: dict[int, bool] = {}
        self.changed: bool = False
        self._panel_order = ["cargo", "weapons", "characters", "research"]
        self._panel_idx = 0

    def on_mount(self) -> None:
        self.push_screen(SaveSelectScreen(), self._after_select)

    def _after_select(self, path: Path | None) -> None:
        if path is None:
            self.exit()
            return
        self.load_game(path)

    def load_game(self, path: Path) -> None:
        self.save_path = path
        self.content = load_save(path)
        self.ship_start, self.ship_end = find_player_ship_bounds(self.content)
        self.inventario = parse_inventory(self.content[self.ship_start:self.ship_end])
        self.characters = find_player_characters(self.content)
        self.research = parse_research(self.content)
        self.changed = False
        self._update_subtitle()
        self.call_after_refresh(self._refresh_all_panels)

    def _refresh_all_panels(self) -> None:
        for panel_id in ("cargo", "weapons", "characters", "research"):
            try:
                panel = self.query_one(f"#{panel_id}")
                if hasattr(panel, "refresh_data"):
                    panel.refresh_data()  # type: ignore[union-attr]
            except Exception:
                pass

    def apply_edit(self, new_content: str) -> None:
        self.content = new_content
        self.ship_start, self.ship_end = find_player_ship_bounds(new_content)
        self.inventario = parse_inventory(new_content[self.ship_start:self.ship_end])
        self.changed = True
        self._update_subtitle()

    def _update_subtitle(self) -> None:
        credits = get_credits(self.content)
        done = sum(1 for v in self.research.values() if v)
        total = len(self.research)
        marker = " [yellow]●[/yellow]" if self.changed else ""
        self.sub_title = (
            f"Créditos: {credits:,}   "
            f"Research: {done}/{total}   "
            f"Tripulación: {len(self.characters)}{marker}"
        )

    def compose(self) -> ComposeResult:
        yield Header()
        with Horizontal(id="app-main"):
            with Vertical(id="sidebar"):
                yield Static("MENÚ", id="sidebar-title")
                yield ListView(
                    ListItem(Label("📦  Cargo / Recursos"), id="nav-cargo"),
                    ListItem(Label("🔫  Armas y Defensa"), id="nav-weapons"),
                    ListItem(Label("👤  Personajes"), id="nav-characters"),
                    ListItem(Label("🔬  Investigación"), id="nav-research"),
                    id="nav-list",
                )
            with Vertical(id="content-area"):
                with ContentSwitcher(initial="cargo"):
                    yield CargoPanel(id="cargo")
                    yield WeaponsPanel(id="weapons")
                    yield CharactersPanel(id="characters")
                    yield ResearchPanel(id="research")
        yield Footer()

    @on(ListView.Selected, "#nav-list")
    def _nav(self, event: ListView.Selected) -> None:
        panel_id = event.item.id.replace("nav-", "")
        self.query_one(ContentSwitcher).current = panel_id
        try:
            panel = self.query_one(f"#{panel_id}")
            if hasattr(panel, "refresh_data"):
                panel.refresh_data()  # type: ignore[union-attr]
        except Exception:
            pass

    def action_save_file(self) -> None:
        if not self.changed:
            self.notify("Sin cambios.", title="Info")
            return
        if self.save_path is None:
            return
        backup = backup_save(self.save_path)
        self.save_path.write_text(self.content, encoding="utf-8")
        self.changed = False
        self._update_subtitle()
        self.notify(f"✓ Save guardado. Backup: {backup.name}", title="Guardado")

    def action_request_quit(self) -> None:
        if not self.changed:
            self.exit()
            return
        self.push_screen(ConfirmModal("Hay cambios sin guardar."), self._quit_result)

    def _quit_result(self, result: str) -> None:
        if result == "save":
            self.action_save_file()
            self.exit()
        elif result == "discard":
            self.exit()

    def action_edit_credits(self) -> None:
        cur = get_credits(self.content)
        self.push_screen(
            EditValueModal("Créditos", cur),
            lambda v: self._apply_credits(cur, v),
        )

    def _apply_credits(self, old: int, new_val: int | None) -> None:
        if new_val is None:
            return
        self.content = set_credits(self.content, new_val)
        self.changed = True
        self._update_subtitle()
        self.notify(f"Créditos: {old:,} → {new_val:,}")


def main() -> None:
    app = SpaceHavenEditorApp()
    app.run()


if __name__ == "__main__":
    main()
