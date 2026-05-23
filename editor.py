#!/usr/bin/env python3
"""Space Haven Save Editor — busca recursos por nombre en español."""

import re
import sys
import shutil
from pathlib import Path
from datetime import datetime

SAVEGAMES_DIR = Path("/mnt/f/Steam/steamapps/common/SpaceHaven/savegames")

# eid → (nombre_ES, nombre_EN)
# Extraído de spacehaven.jar: library/haven + library/texts
RECURSOS: dict[int, tuple[str, str]] = {
    # Comida y bebida
    15:   ("Hortalizas de raíz",          "Root Vegetables"),
    16:   ("Agua",                          "Water"),
    71:   ("Biomateria",                    "Bio Matter"),
    706:  ("Frutas",                        "Fruits"),
    707:  ("Carne artificial",              "Artificial Meat"),
    712:  ("Comida espacial",               "Space Food"),
    984:  ("Carne de monstruo",             "Monster Meat"),
    985:  ("Carne humana",                  "Human Meat"),
    2657: ("Frutos secos",                  "Nuts and Seeds"),
    179:  ("Comida procesada",              "Processed Food"),
    # Bebidas
    3366: ("Bebida alcohólica suave",       "Mild Alcohol"),
    3378: ("Cereales y lúpulos",            "Grains and Hops"),
    # Combustible
    178:  ("Hipercombustible",              "Hyperfuel"),
    # Recursos básicos
    40:   ("Hielo",                         "Ice"),
    157:  ("Metales básicos",               "Base Metals"),
    158:  ("Energio",                       "Energium"),
    169:  ("Metales nobles",                "Noble Metals"),
    170:  ("Carbono",                       "Carbon"),
    171:  ("Sustancias químicas",           "Raw Chemicals"),
    172:  ("Hiperio",                       "Hyperium"),
    173:  ("Componentes electrónicos",      "Electronics Component"),
    174:  ("Barras de energía",             "Energy Rod"),
    175:  ("Plásticos",                     "Plastics"),
    176:  ("Productos químicos",            "Chemicals"),
    177:  ("Tejidos",                       "Fabrics"),
    3512: ("Mineral exótico",               "Exotic Ore"),
    3513: ("Mineral básico",                "Basic Ore"),
    # Bloques fabricados
    162:  ("Infrabloques",                  "Infrablock"),
    930:  ("Tecnobloques",                  "Techblock"),
    1759: ("Cascobloques",                  "Hull Block"),
    1919: ("Energibloques",                 "Energy Block"),
    1920: ("Superbloques",                  "Superblock"),
    1921: ("Blandibloques",                 "Soft Block"),
    1922: ("Placas de acero",               "Steel Plates"),
    # Componentes procesados
    1924: ("Componentes optoelectrónicos",  "Optronics Component"),
    1925: ("Componentes cuantrónicos",      "Quantronics Component"),
    1926: ("Células de energía",            "Energy Cell"),
    1932: ("Fibras",                        "Fibers"),
    2475: ("Fertilizante",                  "Fertilizer"),
    3419: ("Partes de aumentos",            "Augmentation Parts"),
    # Residuos
    1873: ("Infrarresiduos",                "Infra Scrap"),
    1874: ("Residuos blandos",              "Soft Scrap"),
    1886: ("Restos de casco",               "Hull Scrap"),
    1946: ("Tecnorresiduos",                "Tech Scrap"),
    1947: ("Residuos energéticos",          "Energy Scrap"),
    # Dinero
    1858: ("Créditos",                      "Credits"),
    # Médico y consumibles
    2053: ("Suministros médicos",           "Medical Supplies"),
    2058: ("Líquido intravenoso",           "IV Fluid"),
    3925: ("PMI",                           "ISP"),
    4005: ("Analgésicos",                   "Painkillers"),
    4006: ("Estimulante de combate",        "Combat Stimulant"),
    4007: ("Venda",                         "Bandage"),
    4020: ("Estimulante de ánimo",          "Mood Stimulant"),
    4022: ("CSP",                           "CSP"),
    4025: ("Enzima alienígena",             "Alien Enzyme"),
    4030: ("Vendaje de nanobots",           "Nano Wound Dressing"),
    4035: ("Jeringa sedante",               "Sedative Syringe"),
    # Armas
    725:  ("Rifle de asalto",               "Rifle"),
    728:  ("Subfusil",                      "SMG"),
    729:  ("Escopeta",                      "Shotgun"),
    746:  ("Granada",                       "Grenade"),
    760:  ("Pistola",                       "Pistol"),
    1021: ("Fusil de francotirador",        "Sniper Rifle"),
    1152: ("Torreta X1",                    "Sentry Gun X1"),
    2715: ("Munición explosiva",            "Explosive Ammunition"),
    3069: ("Rifle láser",                   "Laser Rifle"),
    3070: ("Pistola láser",                 "Laser Pistol"),
    3071: ("Clustergun de plasma",          "Plasma Clustergun"),
    3072: ("Rifle de plasma",               "Plasma Rifle"),
    3956: ("Subfusil (M2)",                 "SMG"),
    3960: ("Lanzallamas",                   "Flamethrower"),
    3961: ("Rifle aturdidor",               "Stun Rifle"),
    3962: ("Pistola aturdidora",            "Stun Pistol"),
    3967: ("Lanzagranadas explosivo",       "Explosive Grenade Launcher"),
    4040: ("Carga de brecha",               "Small Breaching Charge"),
    4076: ("Lanzagranadas incendiario",     "Incendiary Grenade Launcher"),
    4533: ("Subfusil (M3)",                 "SMG"),
    # Accesorios de armas
    3968: ("Mira básica",                   "Basic Scope"),
    3969: ("Empuñadura táctica",            "Tactical Grip"),
    3975: ("Autocargador de escopeta",      "Shotgun Autoloader"),
    # Equipo y ropa
    481:  ("Sombrero",                      "Hat"),
    488:  ("Gafas de sol",                  "Sunglasses"),
    3383: ("Chaleco antibalas",             "Bulletproof Vest"),
    3384: ("Chaleco blindado",              "Armored Vest"),
    3386: ("Control remoto",               "Remote Control"),
    3387: ("Generador de oxígeno portátil", "Oxygen Generator"),
    3388: ("Tanque de oxígeno",             "Oxygen Tank"),
    3630: ("Extensor de O2 para traje",     "Space Suit Oxygen Extender"),
    4065: ("Extensor de O2 para traje (M2)","Space Suit Oxygen Extender"),
    # Herramientas
    1733: ("Extintor",                      "Fire Extinguisher"),
    2419: ("Utensilio de limpieza",         "Cleaning Tool"),
    # Varios
    2797: ("Collar de esclavo",             "Slave Collar"),
    3728: ("Fragmento de código",           "Code Fragment"),
    4027: ("Órganos alienígenas",           "Alien Organs"),
    4028: ("Órganos humanos",               "Human Organs"),
}

COMIDA_IDS      = {15, 16, 71, 179, 706, 707, 712, 984, 985, 2657, 3366, 3378}
COMBUSTIBLE_IDS = {178}
MEDICO_IDS      = {2053, 2058, 3925, 4005, 4006, 4007, 4020, 4022, 4025, 4030, 4035}
ARMAS_IDS       = {725, 728, 729, 746, 760, 1021, 1152, 2715, 3069, 3070, 3071, 3072,
                   3956, 3960, 3961, 3962, 3967, 4040, 4076, 4533,
                   3968, 3969, 3975}
EQUIPO_IDS      = {481, 488, 3383, 3384, 3386, 3387, 3388, 3630, 4065, 1733, 2419}

# Tags that identify machine sub-elements containing production buffers.
# <inv> blocks nested directly inside these are NOT main cargo — they are
# input/output buffers of machines and must not be touched by the editor.
_MACHINE_TAGS = frozenset({
    'prod', 'grow', 'engine', 'medical', 'mine', 'refinery',
    'purifier', 'converter', 'research', 'workshop', 'logic',
    'extract', 'analyze',
})

_INV_RE = re.compile(r'<inv>(.*?)</inv>', re.DOTALL)


def nombre(eid: int) -> str:
    entry = RECURSOS.get(eid)
    if entry:
        return entry[0]
    return f"Elemento #{eid}"


def nombre_largo(eid: int) -> str:
    entry = RECURSOS.get(eid)
    if entry:
        return f"{entry[0]}  ({entry[1]})"
    return f"Elemento #{eid}"


def buscar_por_nombre(query: str) -> list[tuple[int, str, str]]:
    q = query.lower()
    return [
        (eid, es, en)
        for eid, (es, en) in RECURSOS.items()
        if q in es.lower() or q in en.lower() or q == str(eid)
    ]


# ---------------------------------------------------------------------------
# Cargo container classification
# ---------------------------------------------------------------------------

def _last_opening_tag(text: str) -> str:
    """Name of the last opening or self-closing XML tag in text, '' if none."""
    m = re.search(r'<([a-zA-Z][a-zA-Z0-9]*)\b[^>]*/?>$', text.rstrip())
    return m.group(1).lower() if m else ''


def _is_cargo_inv(ship_block: str, inv_match: re.Match) -> bool:
    """True when <inv> is a direct child of <feat> (main cargo container).

    Space Haven XML structure:
      Cargo container:  <feat ...> <inv> ...   ← direct child, edit allowed
      Machine buffer:   <feat ...> <prod> <inv> ← nested inside machine, leave alone
    """
    pre = ship_block[:inv_match.start()]
    return _last_opening_tag(pre) == 'feat'


def _cargo_inv_ranges(ship_block: str) -> list[tuple[int, int]]:
    """(start, end) spans of all cargo <inv> blocks within ship_block."""
    return [
        (m.start(), m.end())
        for m in _INV_RE.finditer(ship_block)
        if _is_cargo_inv(ship_block, m)
    ]


def _in_cargo(pos: int, ranges: list[tuple[int, int]]) -> bool:
    return any(s <= pos < e for s, e in ranges)


# ---------------------------------------------------------------------------
# Save I/O
# ---------------------------------------------------------------------------

def load_save(path: Path) -> str:
    return path.read_text(encoding="utf-8")


def get_credits(content: str) -> int:
    m = re.search(r'<playerBank\s[^>]*\bca="(\d+)"', content)
    return int(m.group(1)) if m else 0


def set_credits(content: str, amount: int) -> str:
    return re.sub(
        r'(<playerBank\s[^>]*\bca=")(\d+)(")',
        lambda m: f'{m.group(1)}{amount}{m.group(3)}',
        content
    )


def backup_save(path: Path) -> Path:
    ts = datetime.now().strftime("%Y%m%d_%H%M%S")
    backup = path.with_name(f"game.bak_{ts}")
    shutil.copy2(path, backup)
    return backup


def find_player_ship_id(content: str) -> str:
    m = re.search(r'isPlayer="true".*?createdShipId="(\d+)"', content, re.DOTALL)
    return m.group(1) if m else "38"


def find_player_ship_bounds(content: str) -> tuple[int, int]:
    sid = find_player_ship_id(content)
    start_match = re.search(r'<ship\s[^>]*sid="' + sid + r'"[^>]*>', content)
    if not start_match:
        raise ValueError(f"Nave del jugador (sid={sid}) no encontrada.")
    start = start_match.start()
    pos = start_match.end()
    depth = 1
    while depth > 0 and pos < len(content):
        next_open = content.find("<ship", pos)
        next_close = content.find("</ship>", pos)
        if next_close == -1:
            break
        if next_open != -1 and next_open < next_close:
            depth += 1
            pos = next_open + 1
        else:
            depth -= 1
            pos = next_close + len("</ship>")
    return start, pos


# ---------------------------------------------------------------------------
# Inventory read / write
# ---------------------------------------------------------------------------

def parse_inventory(ship_content: str) -> dict[int, int]:
    """Sum inStorage for all items in CARGO containers only (not machine buffers)."""
    totals: dict[int, int] = {}
    cargo_ranges = _cargo_inv_ranges(ship_content)
    for m in re.finditer(r'<s\s+elementaryId="(\d+)"\s+inStorage="(\d+)"', ship_content):
        if _in_cargo(m.start(), cargo_ranges):
            eid, amt = int(m.group(1)), int(m.group(2))
            totals[eid] = totals.get(eid, 0) + amt
    return totals


def _insert_resource(ship_block: str, eid: int, value: int) -> str:
    """Insert a new <s> tag into the largest cargo container."""
    cargo_invs = [
        m for m in _INV_RE.finditer(ship_block)
        if _is_cargo_inv(ship_block, m)
    ]
    if not cargo_invs:
        return ship_block

    best = max(cargo_invs, key=lambda m: len(re.findall(r'elementaryId="\d+"', m.group(1))))
    s_m = re.search(r'(\s+)<s\s+elementaryId=', best.group(1))
    indent = s_m.group(1) if s_m else '\n\t\t\t\t\t\t\t\t'
    new_tag = f'{indent}<s elementaryId="{eid}" inStorage="{value}" onTheWayIn="0" onTheWayOut="0"/>'
    insert_at = best.end(1)
    return ship_block[:insert_at] + new_tag + ship_block[insert_at:]


def set_resource(content: str, ship_start: int, ship_end: int,
                 eid: int, new_value: int) -> tuple[str, bool]:
    """Set the total inStorage for eid across all CARGO containers.

    Machine buffer amounts are never modified.
    """
    ship_block = content[ship_start:ship_end]
    cargo_ranges = _cargo_inv_ranges(ship_block)
    pattern = re.compile(
        r'(<s\s+elementaryId="{}")\s+(inStorage=")(\d+)(")'.format(eid)
    )
    matches = [
        m for m in pattern.finditer(ship_block)
        if _in_cargo(m.start(), cargo_ranges)
    ]

    if not matches:
        new_ship = _insert_resource(ship_block, eid, new_value)
        return content[:ship_start] + new_ship + content[ship_end:], True

    count = len(matches)
    base = new_value // count
    remainder = new_value % count
    new_ship = ship_block
    offset = 0
    for i, m in enumerate(matches):
        slot_value = base + (1 if i < remainder else 0)
        replacement = f'{m.group(1)} {m.group(2)}{slot_value}{m.group(4)}'
        s = m.start() + offset
        e = m.end() + offset
        new_ship = new_ship[:s] + replacement + new_ship[e:]
        offset += len(replacement) - (m.end() - m.start())
    return content[:ship_start] + new_ship + content[ship_end:], False


# ---------------------------------------------------------------------------
# UI helpers
# ---------------------------------------------------------------------------

def print_section(titulo: str, inventario: dict[int, int], filter_ids: set[int] | None = None):
    print(f"\n  {'─' * 42}")
    print(f"  {titulo}")
    print(f"  {'─' * 42}")
    mostrar = {k: v for k, v in inventario.items()
               if filter_ids is None or k in filter_ids}
    if not mostrar:
        print("  (ninguno)")
        return
    for eid, qty in sorted(mostrar.items(), key=lambda x: nombre(x[0])):
        label = nombre(eid)
        print(f"  [{eid:>5}]  {label:<35}  {qty:>6}")


def edit_loop(content: str, ship_start: int, ship_end: int,
              inventario: dict[int, int]) -> tuple[str, bool]:
    changed = False
    print()
    print("  ID para editar | 'b' buscar | 'c' créditos | '0' terminar")

    while True:
        print("  > ", end="")
        try:
            entrada = input().strip()
        except EOFError:
            break

        if entrada == "0":
            break

        if entrada.lower() == "c":
            current_credits = get_credits(content)
            print(f"  Créditos — actual: {current_credits:,}")
            print(f"  Nuevo valor: ", end="")
            try:
                new_credits = int(input().strip())
            except (ValueError, EOFError):
                print("  Entrada inválida. Sin cambios.")
                continue
            if new_credits < 0:
                print("  El valor debe ser >= 0.")
                continue
            content = set_credits(content, new_credits)
            print(f"  ✓ Créditos: {current_credits:,} → {new_credits:,}")
            changed = True
            continue

        if entrada.lower() == "b":
            print()
            print("  Buscar por nombre (ej: fertilizante, acero): ", end="")
            try:
                query = input().strip()
            except EOFError:
                continue
            results = buscar_por_nombre(query)
            if not results:
                print("  Sin resultados.")
            else:
                for eid, es, en in sorted(results, key=lambda x: x[1]):
                    print(f"  [{eid:>5}]  {es}  ({en})")
            print("  (escribí el ID para editarlo)")
            continue

        try:
            choice = int(entrada)
        except ValueError:
            print("  Entrada inválida. Usá un número o 'b'.")
            continue

        label = nombre_largo(choice)
        current = inventario.get(choice, 0)
        if choice not in inventario:
            print(f"  {label}  —  no está en cargo (se insertará)")
        else:
            print(f"  {label}  —  actual: {current}")

        print(f"  Nuevo valor: ", end="")
        try:
            new_val = int(input().strip())
        except (ValueError, EOFError):
            print("  Entrada inválida. Sin cambios.")
            continue

        if new_val < 0:
            print("  El valor debe ser >= 0.")
            continue

        content, inserted = set_resource(content, ship_start, ship_end, choice, new_val)
        inventario[choice] = new_val
        accion = "insertado" if inserted else "actualizado"
        print(f"  ✓ {nombre(choice)} {accion}: {current} → {new_val}")
        changed = True

    return content, changed


# ---------------------------------------------------------------------------
# Save selection
# ---------------------------------------------------------------------------

def list_saves(base: Path) -> list[tuple[Path, str, str]]:
    saves = []
    for slot in sorted(base.iterdir()):
        game = slot / "save" / "game"
        info = slot / "save" / "info"
        if not game.exists():
            continue
        content = game.read_text(encoding="utf-8", errors="replace")
        sname_m = re.search(r'isPlayer="true".*?shn="([^"]+)"', content, re.DOTALL)
        sname = sname_m.group(1) if sname_m else "Nave desconocida"
        date_str = "?"
        if info.exists():
            ts_m = re.search(r'realTimeDate="(\d+)"', info.read_text())
            if ts_m:
                ts = int(ts_m.group(1)) / 1000
                date_str = datetime.fromtimestamp(ts).strftime("%Y-%m-%d %H:%M")
        saves.append((game, sname, date_str))
    return saves


def select_save() -> Path:
    if not SAVEGAMES_DIR.exists():
        print(f"Directorio de saves no encontrado: {SAVEGAMES_DIR}")
        sys.exit(1)
    saves = list_saves(SAVEGAMES_DIR)
    if not saves:
        print("No se encontraron saves.")
        sys.exit(1)

    print("\n" + "=" * 55)
    print("  Space Haven — Editor de Save")
    print("=" * 55)
    print()
    print("  IMPORTANTE: Cerrá el juego antes de editar.")
    print()
    print("  Seleccioná un save:\n")
    for i, (path, sname, date) in enumerate(saves, 1):
        slot = path.parent.parent.name
        print(f"  [{i}]  {slot:<12}  {sname:<22}  {date}")
    print()

    while True:
        print(f"  Opción (1-{len(saves)}): ", end="")
        try:
            choice = int(input().strip())
        except (ValueError, EOFError):
            continue
        if 1 <= choice <= len(saves):
            return saves[choice - 1][0]
        print("  Opción inválida.")


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main():
    save_path = select_save()
    content = load_save(save_path)
    ship_start, ship_end = find_player_ship_bounds(content)
    inventario = parse_inventory(content[ship_start:ship_end])

    sname_m = re.search(r'isPlayer="true".*?shn="([^"]+)"', content, re.DOTALL)
    ship_name = sname_m.group(1) if sname_m else "Nave del jugador"

    credits = get_credits(content)

    print("\n" + "=" * 48)
    print(f"  {ship_name} — Cargo")
    print("=" * 48)
    print(f"\n  {'─' * 42}")
    print(f"  CRÉDITOS")
    print(f"  {'─' * 42}")
    print(f"  {'Créditos':<35}  {credits:>10,}")
    known = COMIDA_IDS | COMBUSTIBLE_IDS | MEDICO_IDS | ARMAS_IDS | EQUIPO_IDS
    print_section("COMIDA / BEBIDA", inventario, COMIDA_IDS)
    print_section("COMBUSTIBLE", inventario, COMBUSTIBLE_IDS)
    print_section("MÉDICO / CONSUMIBLES", inventario, MEDICO_IDS)
    print_section("ARMAS Y MUNICIÓN", inventario, ARMAS_IDS)
    print_section("EQUIPO Y HERRAMIENTAS", inventario, EQUIPO_IDS)
    print_section("MATERIALES Y RECURSOS", inventario,
                  {k for k in inventario if k not in known})

    content, changed = edit_loop(content, ship_start, ship_end, inventario)

    if not changed:
        print("\n  Sin cambios. Saliendo.")
        return

    print("\n  ¿Guardar cambios? (s/n): ", end="")
    try:
        answer = input().strip().lower()
    except EOFError:
        answer = "n"

    if answer == "s":
        backup = backup_save(save_path)
        print(f"  Backup creado: {backup.name}")
        save_path.write_text(content, encoding="utf-8")
        print("  ✓ Save actualizado.")
        print("  Cerrá el juego si está abierto, luego cargá el save.")
    else:
        print("  Cambios descartados.")

    print("\nListo.\n")


if __name__ == "__main__":
    main()
