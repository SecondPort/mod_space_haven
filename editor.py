#!/usr/bin/env python3
"""Space Haven Save Editor — cargo, characters, and research."""

import re
import sys
import shutil
import os
from pathlib import Path
from datetime import datetime

SPACEHAVEN_RELATIVE_SAVE_PATH = Path("steamapps/common/SpaceHaven/savegames")


def _candidate_save_dirs() -> list[Path]:
    candidates: list[Path] = []

    env_path = os.environ.get("SPACEHAVEN_SAVEGAMES_DIR", "").strip()
    if env_path:
        candidates.append(Path(env_path).expanduser())

    home = Path.home()
    candidates.extend([
        home / ".local/share/Steam" / SPACEHAVEN_RELATIVE_SAVE_PATH,
        home / ".steam/steam" / SPACEHAVEN_RELATIVE_SAVE_PATH,
        home / ".var/app/com.valvesoftware.Steam/.local/share/Steam" / SPACEHAVEN_RELATIVE_SAVE_PATH,
    ])

    mnt_root = Path("/mnt")
    if mnt_root.exists():
        for drive in sorted(mnt_root.iterdir()):
            candidates.append(drive / "Steam" / SPACEHAVEN_RELATIVE_SAVE_PATH)

    deduped: list[Path] = []
    seen: set[Path] = set()
    for path in candidates:
        if path not in seen:
            deduped.append(path)
            seen.add(path)
    return deduped


def detect_savegames_dir() -> Path | None:
    for candidate in _candidate_save_dirs():
        if candidate.exists():
            return candidate
    return None


def expected_savegames_dirs() -> list[Path]:
    return _candidate_save_dirs()


SAVEGAMES_DIR = detect_savegames_dir()

# eid → (nombre_ES, nombre_EN)
# Extracted from spacehaven.jar: library/haven + library/texts
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
    4022: ("PSN",                            "CSP"),
    4025: ("Enzima alienígena",             "Alien Enzyme"),
    4030: ("Vendaje de nanobots",           "Nano Wound Dressing"),
    4035: ("Jeringa sedante",               "Sedative Syringe"),
    # Armas
    725:  ("Fusil",                           "Rifle"),
    728:  ("Subfusil",                      "SMG"),
    729:  ("Escopeta",                      "Shotgun"),
    746:  ("Granada",                       "Grenade"),
    760:  ("Pistola",                       "Pistol"),
    1021: ("Fusil de francotirador",        "Sniper Rifle"),
    1152: ("Arma centinela X1",              "Sentry Gun X1"),
    2715: ("Munición explosiva",            "Explosive Ammunition"),
    3069: ("Fusil láser",                   "Laser Rifle"),
    3070: ("Pistola láser",                 "Laser Pistol"),
    3071: ("Fusil de racimo plasmático",    "Plasma Clustergun"),
    3072: ("Fusil de plasma",               "Plasma Rifle"),
    3956: ("Subfusil",                      "SMG"),
    3960: ("Lanzallamas",                   "Flamethrower"),
    3961: ("Fusil aturdidor",               "Stun Rifle"),
    3962: ("Pistola aturdidora",            "Stun Pistol"),
    3967: ("Lanzagranadas explosivo",       "Explosive Grenade Launcher"),
    4040: ("Carga explosiva pequeña",        "Small Breaching Charge"),
    4076: ("Lanzagranadas incendiario",     "Incendiary Grenade Launcher"),
    4533: ("Subfusil",                      "SMG"),
    # Accesorios de armas
    3968: ("Visor básico",                  "Basic Scope"),
    3969: ("Empuñadura táctica",            "Tactical Grip"),
    3975: ("Cargador automático de escopeta", "Shotgun Autoloader"),
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

# sk= (saveNR from Job$SkillClass bytecode) → Spanish skill name
# Source: javap -verbose Job$SkillClass.class (static initializer)
HABILIDADES: dict[int, str] = {
    2:  "Minería",
    3:  "Botánica",
    4:  "Construcción",
    5:  "Industria",
    6:  "Medicina",
    7:  "Artillero",
    8:  "Escudos",
    9:  "Operaciones",
    10: "Armas",
    12: "Logística",
    13: "Química",
    14: "Navegación",
    16: "Investigación",
    22: "Pilotaje",
}

# attribute id= (name_tid from PersonalitySetting$AttributeType) → English name
# Source: javap -verbose PersonalitySetting$AttributeType.class
ATRIBUTOS: dict[int, str] = {
    210: "Bravery",
    212: "Zest",
    213: "Intelligence",
    214: "Perception",
}

# trait id= → English name
# Source: library/haven <trait id="X"><name tid="Y"/> cross-referenced with library/texts
RASGOS: dict[int, str] = {
    191:  "Hero",
    655:  "Wimp",
    656:  "Clumsy",
    1034: "Moody",
    1035: "Smart",
    1036: "Bloodlust",
    1037: "Antisocial",
    1038: "Needy",
    1039: "Fast learner",
    1040: "Lazy",
    1041: "Hard working",
    1042: "Psychopath",
    1043: "Peace-loving",
    1044: "Iron-willed",
    1045: "Spacefarer",
    1046: "Confident",
    1047: "Neurotic",
    1048: "Charming",
    1533: "Iron stomach",
    1534: "Nyctophilia",
    1535: "Minimalist",
    1560: "Talkative",
    1562: "Gourmand",
    2082: "Alien lover",
}

# tech id= → (English name, (labPoints_l1, labPoints_l2, labPoints_l3))
# Source: library/haven <tech id="X"> with labPoints + library/texts for names
TECNOLOGIAS: dict[int, tuple[str, tuple[int, int, int]]] = {
    2532: ("Scanner",                     (150,  40,  10)),
    2533: ("Shield Generator",            (180,  40,  12)),
    2534: ("Energy Turret",               (100,  50,  25)),
    2538: ("Large Storage",               (100,  30,  15)),
    2539: ("Autopsy Table",               ( 40,   0,   0)),
    2559: ("Medical Bed",                 (120,  60,  15)),
    2560: ("Grow Bed with Light",         ( 80,  15,   0)),
    2561: ("CO2 Producer",                ( 80,  12,   0)),
    2563: ("Arcade Machine",              (125,  60,  20)),
    2564: ("Basic Entertainment",         ( 60,  30,   0)),
    2565: ("Solar Panel",                 (180,  50,  10)),
    2566: ("X2 Power Generator",          ( 80,  20,  10)),
    2567: ("X3 Power Generator",          (100,  40,  30)),
    2568: ("Power Capacity Node",         (120,  60,   5)),
    2569: ("Item Fabricator",             (100,  30,   6)),
    2570: ("Water Collector",             ( 60,  25,   0)),
    2571: ("Assembler",                   ( 90,  10,   0)),
    2572: ("Micro-Weaver",                ( 50,  10,   0)),
    2573: ("Chemical Refinery",           ( 40,   0,   0)),
    2574: ("Metal Refinery",              (140,  40,   4)),
    2575: ("Energy Refinery",             (200,  80,  50)),
    2576: ("Composter",                   ( 85,  15,   0)),
    2577: ("Hypersleep Chamber",          (150,  50,  20)),
    2581: ("Basic",                       ( 40,   0,   0)),
    2583: ("Hyperium Hyperdrive",         (  1,   0,   0)),
    2584: ("X1 Hyperdrive",               ( 50,   0,   0)),
    2585: ("Advanced",                    (125,  25,   0)),
    2586: ("Optronic",                    (350, 120,  40)),
    2587: ("Quantum",                     (500, 250, 100)),
    2589: ("Navigation console",          (125,  15,   0)),
    2590: ("Weapons console",             (125,  15,   0)),
    2591: ("Rocket Turret",               (180,  75,  12)),
    2592: ("Energy Turret",               (160,  55,  10)),
    2594: ("X1 Power Generator",          ( 30,  10,   0)),
    2595: ("Recycler",                    ( 40,  20,   0)),
    2596: ("Advanced Assembler",          (  1,   0,   0)),
    2597: ("Optronics Fabricator",        (  1,   0,   0)),
    2598: ("Shields console",             (  1,   0,   0)),
    2599: ("Operations console",          (  1,   0,   0)),
    2600: ("Targeting Jammer",            (  1,   0,   0)),
    2601: ("Chemical",                    (400, 250, 120)),
    2602: ("Botany",                      (  1,   0,   0)),
    2604: ("Advanced Nutrition",          (  1,   0,   0)),
    2605: ("Laser Weapons",               (150,  50,  25)),
    2606: ("Plasma Weapons",              (180,  60,  30)),
    2607: ("Stun Weapons",                (180,  80,  40)),
    2609: ("Implanted Rebreather",        (  1,   0,   0)),
    2610: ("Ocular Implant",              (  1,   0,   0)),
    2611: ("Synthetic Stomach Lining",    (  1,   0,   0)),
    2612: ("Metal Refinery",              ( 80,  20,   0)),
    2613: ("Surgical Enhancement Facility",(1,   0,   0)),
    2614: ("Fabrics",                     (  1,   0,   0)),
    2617: ("Fibers",                      (  1,   0,   0)),
    2618: ("Bulletproof Vest",            ( 50,   0,   0)),
    2619: ("Armored Vest",                ( 80,  20,   0)),
    2622: ("Prefrontal Microcontroller",  ( 50,  20,   5)),
    2623: ("Botany",                      ( 40,   0,   0)),
    2626: ("Advanced Nutrition",          ( 20,   0,   0)),
    2627: ("Anatomical Augmentation",     (120,  50,  25)),
    2628: ("Neural Augmentation",         (140,  60,  20)),
    2629: ("Nanotech Augmentation",       (120,  25,  10)),
    2630: ("Alcohol Beverage Machine",    (100,  30,  10)),
    2694: ("Logistics Robot Station",     (400, 100,  50)),
    2696: ("Grains and Hops",             ( 60,  10,   0)),
    2847: ("Salvage Robot Station",       (140,  80,  25)),
    3024: ("Weapon Attachments 1",        (100,  50,  25)),
    3025: ("Sentry gun X1",               (150,  75,  50)),
    3112: ("Recycler",                    ( 50,   5,   0)),
    3114: ("Research Lab",                (  0,   0,   0)),
    3115: ("Research Workbench",          (100,  20,   5)),
    3116: ("Research Experiment Table",   (200,  50,  25)),
    3119: ("Navigation console",          ( 60,  20,   0)),
    3122: ("Shields console",             ( 60,  10,   0)),
    3124: ("Crawler",                     ( 25,   0,   0)),
    3125: ("Hauler",                      ( 30,   0,   0)),
    3126: ("Learning Computer",           (  0,   0,   0)),
    3127: ("Robotics 01",                 ( 60,  20,  10)),
    3128: ("Industry 01",                 ( 50,   5,   0)),
    3129: ("Industry 02",                 ( 80,  12,   0)),
    3130: ("Botany 02",                   ( 50,  20,   5)),
    3417: ("Anatomical Augmentation",     ( 60,  30,  12)),
    3420: ("Anatomical Augmentation",     ( 80,  50,  25)),
    3421: ("Neural Augmentation",         (100,  55,  35)),
    3422: ("Nanotech Augmentation",       ( 90,  60,  32)),
    3423: ("Prefrontal Microcontroller",  ( 75,  30,  20)),
    3464: ("Advanced Learning System",    (100,  40,  20)),
    3704: ("Alien Hive Core",             ( 25,   0,   0)),
    3705: ("Evolving Alien Core",         ( 25,   0,   0)),
    3706: ("Advanced Nutrition 02",       ( 30,   0,   0)),
    3707: ("Hamster (Flybot)",            ( 25,   0,   0)),
    3708: ("Chimp (Walkerbot)",           ( 25,   0,   0)),
    3709: ("Rogue bot Architecture",      (200,  80,  30)),
    3710: ("X2 Hypersleep Tank",          ( 50,  10,   0)),
    3970: ("Advanced Medical Bed",        (140,  80,  25)),
    3973: ("Alien Enzyme",                (150,  50,  25)),
    3974: ("Nano Wound Dressing",         (150,  50,  25)),
    4024: ("Alien Enzyme",                ( 50,  30,  10)),
    4032: ("Stimulants",                  (140,  80,  25)),
    4092: ("Advanced Disassembly",        ( 60,  30,   0)),
    4093: ("X1 Couch",                    ( 60,  20,   0)),
    4132: ("Learning Computer",           ( 40,   0,   0)),
    4134: ("Advanced Learning System",    (100,  20,   5)),
    4529: ("Combat Robot Station",        (150,  75,  50)),
}

# Tags that identify machine sub-elements containing production buffers.
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
# Character read / write
# ---------------------------------------------------------------------------

def find_player_characters(content: str) -> list[dict]:
    """Return list of player-side characters using entId as unique identifier."""
    chars = []
    for m in re.finditer(
        r'<c\s+cid="\d+"[^>]*\bentId="(\d+)"[^>]*\bside="Player"[^>]*\bname="([^"]+)"[^>]*\blname="([^"]*)"',
        content
    ):
        chars.append({'entId': m.group(1), 'name': m.group(2), 'lname': m.group(3), 'start': m.start()})
    return chars


def _all_character_positions(content: str) -> list[int]:
    """Positions of all <c cid=...> character tags in the save."""
    return [m.start() for m in re.finditer(r'<c\s+cid=', content)]


def find_character_bounds(content: str, ent_id: str) -> tuple[int, int]:
    """Find start and end positions of the character block by entId.

    Characters are siblings inside <characters>. A character's block ends just
    before the next sibling <c cid= tag (or </characters>), so we don't need
    XML depth tracking.
    """
    start_m = re.search(r'<c\s+cid="\d+"[^>]*\bentId="' + re.escape(ent_id) + r'"[^>]*>', content)
    if not start_m:
        return -1, -1
    start = start_m.start()
    all_positions = _all_character_positions(content)
    idx = next((i for i, p in enumerate(all_positions) if p == start), -1)
    if idx == -1:
        return -1, -1
    if idx + 1 < len(all_positions):
        next_sibling = all_positions[idx + 1]
        end = content.rfind('</c>', start, next_sibling)
        return (start, end + len('</c>')) if end != -1 else (start, next_sibling)
    else:
        end_m = re.search(r'</characters>', content[start:])
        if end_m:
            end = content.rfind('</c>', start, start + end_m.start())
            return (start, end + len('</c>')) if end != -1 else (start, start + end_m.start())
        return start, len(content)


def get_char_stat(char_block: str, stat: str) -> int:
    m = re.search(r'<' + re.escape(stat) + r'\s+v="(\d+)"', char_block)
    return int(m.group(1)) if m else -1


def set_char_stat(char_block: str, stat: str, value: int) -> str:
    return re.sub(
        r'(<' + re.escape(stat) + r'\s+v=")(\d+)(")',
        lambda m: f'{m.group(1)}{value}{m.group(3)}',
        char_block, count=1
    )


def get_char_skills(char_block: str) -> dict[int, dict]:
    skills = {}
    for m in re.finditer(r'<s sk="(\d+)"([^/]*)/>', char_block):
        sk = int(m.group(1))
        attrs = m.group(2)
        level_m = re.search(r'level="(\d+)"', attrs)
        mxn_m = re.search(r'mxn="(\d+)"', attrs)
        if level_m and mxn_m:
            skills[sk] = {'level': int(level_m.group(1)), 'mxn': int(mxn_m.group(1))}
    return skills


def set_char_skill(char_block: str, sk: int, level: int, mxn: int) -> str:
    def replacer(m: re.Match) -> str:
        s = m.group(0)
        s = re.sub(r'level="\d+"', f'level="{level}"', s)
        s = re.sub(r'mxn="\d+"', f'mxn="{mxn}"', s)
        return s
    return re.sub(r'<s sk="' + str(sk) + r'"[^/]*/>', replacer, char_block, count=1)


def get_char_attributes(char_block: str) -> dict[int, int]:
    attrs = {}
    for m in re.finditer(r'<a\s+points="(\d+)"\s+id="(\d+)"', char_block):
        attrs[int(m.group(2))] = int(m.group(1))
    return attrs


def set_char_attribute(char_block: str, attr_id: int, points: int) -> str:
    return re.sub(
        r'(<a\s+points=")(\d+)("\s+id="' + str(attr_id) + r'")',
        lambda m: f'{m.group(1)}{points}{m.group(3)}',
        char_block, count=1
    )


def get_char_traits(char_block: str) -> list[int]:
    traits_m = re.search(r'<traits>(.*?)</traits>', char_block, re.DOTALL)
    if not traits_m:
        return []
    return [int(m.group(1)) for m in re.finditer(r'<t\s+id="(\d+)"', traits_m.group(1))]


def add_char_trait(char_block: str, trait_id: int) -> str:
    traits_m = re.search(r'(<traits>)(.*?)(</traits>)', char_block, re.DOTALL)
    if not traits_m:
        return char_block
    inner = traits_m.group(2)
    # Detect indentation from existing entries
    indent_m = re.search(r'(\s+)<t\s+id=', inner)
    indent = indent_m.group(1) if indent_m else '\n\t\t\t\t\t\t'
    new_entry = f'{indent}<t id="{trait_id}"/>'
    new_block = traits_m.group(1) + inner + new_entry + traits_m.group(3)
    return char_block[:traits_m.start()] + new_block + char_block[traits_m.end():]


def remove_char_trait(char_block: str, trait_id: int) -> str:
    return re.sub(r'\s*<t\s+id="' + str(trait_id) + r'"\s*/>', '', char_block, count=1)


def set_char_name(char_block: str, name: str, lname: str) -> str:
    s = re.sub(r'(\bname=")([^"]*)', lambda m: f'{m.group(1)}{name}', char_block, count=1)
    s = re.sub(r'(\blname=")([^"]*)', lambda m: f'{m.group(1)}{lname}', s, count=1)
    return s


# ---------------------------------------------------------------------------
# Character loadout read
# ---------------------------------------------------------------------------

def get_char_loadout(char_block: str) -> dict[str, int]:
    """Parse <loadout> slot assignments from a character block."""
    m = re.search(r'<loadout\s+([^/]*)/?>', char_block)
    if not m:
        return {}
    attrs_str = m.group(1)
    result: dict[str, int] = {}
    for key in ('headgear', 'armor', 'primary', 'attachment',
                'secondary', 'pocket1', 'pocket2', 'pocket3'):
        km = re.search(rf'{key}="(\d+)"', attrs_str)
        result[key] = int(km.group(1)) if km else 0
    return result


# ---------------------------------------------------------------------------
# Research read / write
# ---------------------------------------------------------------------------

def parse_research(content: str) -> dict[int, bool]:
    """Return {techId: is_done} for all research entries in the save."""
    result = {}
    for m in re.finditer(r'<l\s+techId="(\d+)"[^>]*>.*?done="(true|false)"', content, re.DOTALL):
        result[int(m.group(1))] = m.group(2) == 'true'
    return result


def complete_tech(content: str, tech_id: int) -> str:
    """Mark a research tech as done and set blocksDone to labPoints values."""
    info = TECNOLOGIAS.get(tech_id)
    l1, l2, l3 = info[1] if info else (1, 0, 0)

    def replace_stage(m: re.Match) -> str:
        s = m.group(0)
        s = re.sub(r'done="(?:true|false)"', 'done="true"', s)
        s = re.sub(
            r'blocksDone\s+level1="\d+"\s+level2="\d+"\s+level3="\d+"',
            f'blocksDone level1="{l1}" level2="{l2}" level3="{l3}"',
            s
        )
        return s

    pattern = re.compile(
        r'<l\s+techId="' + str(tech_id) + r'".*?</l>',
        re.DOTALL
    )
    return pattern.sub(replace_stage, content, count=1)


def incomplete_tech(content: str, tech_id: int) -> str:
    """Reset a research tech to not done."""
    def replace_stage(m: re.Match) -> str:
        s = m.group(0)
        s = re.sub(r'done="(?:true|false)"', 'done="false"', s)
        s = re.sub(
            r'blocksDone\s+level1="\d+"\s+level2="\d+"\s+level3="\d+"',
            'blocksDone level1="0" level2="0" level3="0"',
            s
        )
        return s

    pattern = re.compile(
        r'<l\s+techId="' + str(tech_id) + r'".*?</l>',
        re.DOTALL
    )
    return pattern.sub(replace_stage, content, count=1)


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
    print("  ID para editar | 'b' buscar | 'c' créditos | '0' volver")

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


def edit_character_loop(content: str, characters: list[dict]) -> tuple[str, bool]:
    changed = False

    print()
    print("  Personajes:\n")
    for i, c in enumerate(characters, 1):
        print(f"  [{i}]  {c['name']} {c['lname']}")
    print()
    print("  Seleccioná un personaje ('0' para volver): ", end="")

    try:
        choice = int(input().strip())
    except (ValueError, EOFError):
        return content, False

    if choice < 1 or choice > len(characters):
        return content, False

    char_info = characters[choice - 1]
    ent_id = char_info['entId']
    c_start, c_end = find_character_bounds(content, ent_id)
    if c_start == -1:
        print("  Personaje no encontrado.")
        return content, False

    while True:
        char_block = content[c_start:c_end]
        full_name = f"{char_info['name']} {char_info['lname']}"

        health = get_char_stat(char_block, 'Health')
        mood   = get_char_stat(char_block, 'Mood')
        rest   = get_char_stat(char_block, 'Rest')
        skills = get_char_skills(char_block)
        attrs  = get_char_attributes(char_block)
        traits = get_char_traits(char_block)

        print(f"\n  {'─' * 48}")
        print(f"  {full_name}")
        print(f"  {'─' * 48}")
        print(f"  [1] Stats:    Health={health}  Mood={mood}  Rest={rest}")
        print(f"  [2] Skills:")
        for sk, data in sorted(skills.items()):
            name = HABILIDADES.get(sk, f"sk={sk}")
            print(f"       sk={sk:<2}  {name:<14}  level={data['level']}  max={data['mxn']}")
        print(f"  [3] Attributes:")
        for attr_id, pts in sorted(attrs.items()):
            name = ATRIBUTOS.get(attr_id, f"id={attr_id}")
            print(f"       id={attr_id}  {name:<14}  points={pts}")
        print(f"  [4] Traits:   {', '.join(RASGOS.get(t, str(t)) for t in traits) or '(none)'}")
        print(f"  [5] Name")
        print(f"  [0] Volver")
        print()
        print("  Opción: ", end="")

        try:
            opt = input().strip()
        except EOFError:
            break

        if opt == "0":
            break

        elif opt == "1":
            for stat_name, current_val in [('Health', health), ('Mood', mood), ('Rest', rest)]:
                print(f"  {stat_name} (actual: {current_val}, 0-100, Enter para saltar): ", end="")
                try:
                    raw = input().strip()
                except EOFError:
                    break
                if not raw:
                    continue
                try:
                    val = max(0, min(100, int(raw)))
                except ValueError:
                    print("  Valor inválido, saltando.")
                    continue
                char_block = set_char_stat(char_block, stat_name, val)
                print(f"  ✓ {stat_name}: {current_val} → {val}")
                changed = True
            content = content[:c_start] + char_block + content[c_end:]

        elif opt == "2":
            print("  sk= para editar (Enter para saltar cada uno):")
            for sk in sorted(skills.keys()):
                name = HABILIDADES.get(sk, f"sk={sk}")
                cur = skills[sk]
                print(f"  {name} (sk={sk}) level={cur['level']} max={cur['mxn']} — nuevo nivel (0-10): ", end="")
                try:
                    raw = input().strip()
                except EOFError:
                    break
                if not raw:
                    continue
                try:
                    level = max(0, min(10, int(raw)))
                except ValueError:
                    print("  Valor inválido, saltando.")
                    continue
                mxn = max(level, cur['mxn'])
                char_block = set_char_skill(char_block, sk, level, mxn)
                print(f"  ✓ {name}: level={cur['level']} → {level}, max={cur['mxn']} → {mxn}")
                changed = True
            content = content[:c_start] + char_block + content[c_end:]

        elif opt == "3":
            print("  Atributos (total recomendado: 12 puntos):")
            for attr_id in sorted(attrs.keys()):
                name = ATRIBUTOS.get(attr_id, f"id={attr_id}")
                cur = attrs[attr_id]
                print(f"  {name} (id={attr_id}) actual={cur} — nuevo valor (1-8, Enter para saltar): ", end="")
                try:
                    raw = input().strip()
                except EOFError:
                    break
                if not raw:
                    continue
                try:
                    val = max(1, min(8, int(raw)))
                except ValueError:
                    print("  Valor inválido, saltando.")
                    continue
                char_block = set_char_attribute(char_block, attr_id, val)
                print(f"  ✓ {name}: {cur} → {val}")
                changed = True
            content = content[:c_start] + char_block + content[c_end:]

        elif opt == "4":
            print()
            print("  Rasgos actuales:", ', '.join(RASGOS.get(t, str(t)) for t in traits) or '(none)')
            print()
            print("  Rasgos disponibles:")
            for tid, tname in sorted(RASGOS.items(), key=lambda x: x[1]):
                marker = "✓" if tid in traits else " "
                print(f"  [{marker}] {tid:<5}  {tname}")
            print()
            print("  ID de rasgo para agregar/quitar ('0' para volver): ", end="")
            try:
                raw = input().strip()
            except EOFError:
                continue
            if raw == "0":
                continue
            try:
                trait_id = int(raw)
            except ValueError:
                print("  ID inválido.")
                continue
            if trait_id not in RASGOS and trait_id not in traits:
                print(f"  Rasgo {trait_id} no reconocido. ¿Continuar? (s/n): ", end="")
                try:
                    confirm = input().strip().lower()
                except EOFError:
                    continue
                if confirm != "s":
                    continue
            if trait_id in traits:
                char_block = remove_char_trait(char_block, trait_id)
                print(f"  ✓ Rasgo '{RASGOS.get(trait_id, trait_id)}' eliminado.")
            else:
                char_block = add_char_trait(char_block, trait_id)
                print(f"  ✓ Rasgo '{RASGOS.get(trait_id, trait_id)}' agregado.")
            content = content[:c_start] + char_block + content[c_end:]
            changed = True

        elif opt == "5":
            print(f"  Nombre actual: {char_info['name']} {char_info['lname']}")
            print(f"  Nuevo nombre (Enter para saltar): ", end="")
            try:
                new_name = input().strip()
            except EOFError:
                continue
            if not new_name:
                continue
            print(f"  Nuevo apellido (Enter para mantener '{char_info['lname']}'): ", end="")
            try:
                new_lname = input().strip()
            except EOFError:
                new_lname = char_info['lname']
            if not new_lname:
                new_lname = char_info['lname']
            char_block = set_char_name(char_block, new_name, new_lname)
            content = content[:c_start] + char_block + content[c_end:]
            char_info['name'] = new_name
            char_info['lname'] = new_lname
            print(f"  ✓ Nombre actualizado: {new_name} {new_lname}")
            changed = True

    return content, changed


def edit_research_loop(content: str) -> tuple[str, bool]:
    changed = False
    research = parse_research(content)

    while True:
        done_ids   = {tid for tid, done in research.items() if done}
        undone_ids = {tid for tid, done in research.items() if not done}

        print(f"\n  {'─' * 48}")
        print(f"  INVESTIGACIÓN")
        print(f"  {'─' * 48}")
        print(f"  Completadas: {len(done_ids)} / {len(research)}")
        print()
        print("  [1] Ver todas las tecnologías")
        print("  [2] Completar una tecnología")
        print("  [3] Completar TODAS las tecnologías")
        print("  [4] Resetear una tecnología")
        print("  [0] Volver")
        print()
        print("  Opción: ", end="")

        try:
            opt = input().strip()
        except EOFError:
            break

        if opt == "0":
            break

        elif opt == "1":
            print()
            print(f"  {'ID':<6}  {'Nombre':<40}  {'Estado'}")
            print(f"  {'─'*6}  {'─'*40}  {'─'*10}")
            for tech_id in sorted(research.keys()):
                info = TECNOLOGIAS.get(tech_id)
                name = info[0] if info else f"Tech #{tech_id}"
                estado = "✓ Completa" if research[tech_id] else "  Pendiente"
                print(f"  {tech_id:<6}  {name:<40}  {estado}")

        elif opt == "2":
            print("  ID de tecnología a completar: ", end="")
            try:
                raw = input().strip()
            except EOFError:
                continue
            try:
                tech_id = int(raw)
            except ValueError:
                print("  ID inválido.")
                continue
            if tech_id not in research:
                print(f"  Tech {tech_id} no encontrada en el save.")
                continue
            if research[tech_id]:
                print(f"  Ya está completa.")
                continue
            info = TECNOLOGIAS.get(tech_id)
            name = info[0] if info else f"Tech #{tech_id}"
            content = complete_tech(content, tech_id)
            research[tech_id] = True
            print(f"  ✓ '{name}' marcada como completa.")
            changed = True

        elif opt == "3":
            print(f"  ¿Completar las {len(undone_ids)} tecnologías pendientes? (s/n): ", end="")
            try:
                confirm = input().strip().lower()
            except EOFError:
                continue
            if confirm != "s":
                continue
            count = 0
            for tech_id in sorted(undone_ids):
                content = complete_tech(content, tech_id)
                research[tech_id] = True
                count += 1
            print(f"  ✓ {count} tecnologías completadas.")
            changed = True

        elif opt == "4":
            print("  ID de tecnología a resetear: ", end="")
            try:
                raw = input().strip()
            except EOFError:
                continue
            try:
                tech_id = int(raw)
            except ValueError:
                print("  ID inválido.")
                continue
            if tech_id not in research:
                print(f"  Tech {tech_id} no encontrada en el save.")
                continue
            info = TECNOLOGIAS.get(tech_id)
            name = info[0] if info else f"Tech #{tech_id}"
            content = incomplete_tech(content, tech_id)
            research[tech_id] = False
            print(f"  ✓ '{name}' reseteada.")
            changed = True

    return content, changed


# ---------------------------------------------------------------------------
# Save selection
# ---------------------------------------------------------------------------

def list_saves(base: Path) -> list[tuple[Path, str, str]]:
    if not base.exists():
        return []
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
    if SAVEGAMES_DIR is None:
        print("Directorio de saves no encontrado.")
        print("\nRutas buscadas:")
        for path in expected_savegames_dirs():
            print(f"  - {path}")
        print("\nTip: configurá SPACEHAVEN_SAVEGAMES_DIR con la ruta correcta.")
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
    characters = find_player_characters(content)

    sname_m = re.search(r'isPlayer="true".*?shn="([^"]+)"', content, re.DOTALL)
    ship_name = sname_m.group(1) if sname_m else "Nave del jugador"
    credits = get_credits(content)
    research = parse_research(content)
    done_count = sum(1 for v in research.values() if v)

    changed = False

    while True:
        print("\n" + "=" * 55)
        print(f"  {ship_name}")
        print("=" * 55)
        print(f"  Créditos: {credits:,}")
        print(f"  Cargo: {len(inventario)} tipos de ítem")
        print(f"  Tripulación: {len(characters)} personajes")
        print(f"  Investigación: {done_count}/{len(research)} completadas")
        print()
        print("  [1] Cargo / Recursos")
        print("  [2] Personajes")
        print("  [3] Investigación")
        print("  [0] Salir")
        print()
        print("  Opción: ", end="")

        try:
            opt = input().strip()
        except EOFError:
            break

        if opt == "0":
            break

        elif opt == "1":
            known = COMIDA_IDS | COMBUSTIBLE_IDS | MEDICO_IDS | ARMAS_IDS | EQUIPO_IDS
            print("\n" + "=" * 48)
            print(f"  {ship_name} — Cargo")
            print("=" * 48)
            print(f"\n  {'─' * 42}")
            print(f"  CRÉDITOS")
            print(f"  {'─' * 42}")
            print(f"  {'Créditos':<35}  {credits:>10,}")
            print_section("COMIDA / BEBIDA", inventario, COMIDA_IDS)
            print_section("COMBUSTIBLE", inventario, COMBUSTIBLE_IDS)
            print_section("MÉDICO / CONSUMIBLES", inventario, MEDICO_IDS)
            print_section("ARMAS Y MUNICIÓN", inventario, ARMAS_IDS)
            print_section("EQUIPO Y HERRAMIENTAS", inventario, EQUIPO_IDS)
            print_section("MATERIALES Y RECURSOS", inventario, {k for k in inventario if k not in known})
            content, sub_changed = edit_loop(content, ship_start, ship_end, inventario)
            if sub_changed:
                changed = True
                credits = get_credits(content)

        elif opt == "2":
            if not characters:
                print("  No hay personajes del jugador en este save.")
                continue
            content, sub_changed = edit_character_loop(content, characters)
            if sub_changed:
                changed = True

        elif opt == "3":
            content, sub_changed = edit_research_loop(content)
            if sub_changed:
                changed = True
                research = parse_research(content)
                done_count = sum(1 for v in research.values() if v)

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
