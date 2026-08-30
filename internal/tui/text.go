package tui

import "github.com/SecondPort/mod_space_haven/internal/catalog"

// uiText holds every string the interface shows, so a new language is one
// struct literal rather than a hunt through the layout code.
type uiText struct {
	AppTitle    string
	PickTitle   string
	PickWarning string
	NoSaves     string
	NoSaveDir   string
	SearchedIn  string
	DirTip      string

	ColSlot    string
	ColShip    string
	ColDate    string
	ColID      string
	ColName    string
	ColAmount  string
	ColTech    string
	ColStatus  string
	ColCrew    string
	ColHealth  string
	ColMood    string
	ColRest    string
	ColSkills  string
	ColSkill   string
	ColLevel   string
	ColMax     string
	ColAttr    string
	ColPoints  string
	ColTrait   string
	ColWeapon  string
	ColItem    string
	ColInCargo string
	ColPrimary string
	ColSecond  string
	ColArmor   string
	ColHelmet  string

	NavCargo    string
	NavWeapons  string
	NavCrew     string
	NavResearch string

	SecWeapons     string
	SecAttachments string
	SecDefense     string
	SecArmor       string
	SecHeadgear    string
	SecSurvival    string
	SecLoadouts    string
	SecFood        string
	SecFuel        string
	SecMedical     string
	SecWeaponry    string
	SecGear        string
	SecMaterials   string
	SecSkills      string
	SecAttributes  string
	SecTraits      string

	LoadoutNote string
	EmptySlot   string

	Credits   string
	Research  string
	CrewCount string
	Unsaved   string

	PromptNewValue string
	PromptCredits  string
	PromptFirst    string
	PromptLast     string
	PromptSearch   string
	SearchTitle    string

	Done    string
	Pending string

	ConfirmQuitTitle string
	ConfirmQuitBody  string
	ConfirmSave      string
	ConfirmDiscard   string
	ConfirmCancel    string

	StatusSaved     string
	StatusNoChanges string
	StatusUpdated   string
	StatusInserted  string
	StatusAdded     string
	StatusRemoved   string
	StatusCompleted string
	StatusReset     string
	StatusAllDone   string
	StatusAllReset  string
	StatusRenamed   string

	KeysGlobal string
	KeysCargo  string
	KeysResear string
	KeysCrew   string
	KeysDetail string
	KeysPicker string
	KeysModal  string
}

var textES = uiText{
	AppTitle:    "Space Haven — Editor de Save",
	PickTitle:   "Elegí un save",
	PickWarning: "IMPORTANTE: cerrá el juego antes de editar.",
	NoSaves:     "No se encontraron saves en esta carpeta.",
	NoSaveDir:   "No se encontró la carpeta de saves.",
	SearchedIn:  "Rutas buscadas:",
	DirTip:      "Configurá SPACEHAVEN_SAVEGAMES_DIR con la ruta correcta.",

	ColSlot:    "Slot",
	ColShip:    "Nave",
	ColDate:    "Fecha",
	ColID:      "ID",
	ColName:    "Nombre",
	ColAmount:  "Cantidad",
	ColTech:    "Tecnología",
	ColStatus:  "Estado",
	ColCrew:    "Personaje",
	ColHealth:  "Salud",
	ColMood:    "Ánimo",
	ColRest:    "Descanso",
	ColSkills:  "Habilidades",
	ColSkill:   "Habilidad",
	ColLevel:   "Nivel",
	ColMax:     "Máx",
	ColAttr:    "Atributo",
	ColPoints:  "Puntos",
	ColTrait:   "Rasgo",
	ColWeapon:  "Arma",
	ColItem:    "Ítem",
	ColInCargo: "En cargo",
	ColPrimary: "Primaria",
	ColSecond:  "Secundaria",
	ColArmor:   "Armadura",
	ColHelmet:  "Casco",

	NavCargo:    "Cargo",
	NavWeapons:  "Armas",
	NavCrew:     "Tripulación",
	NavResearch: "Investigación",

	SecWeapons:     "ARMAS",
	SecAttachments: "ACCESORIOS",
	SecDefense:     "DEFENSA Y EQUIPO",
	SecArmor:       "ARMADURAS",
	SecHeadgear:    "CASCOS",
	SecSurvival:    "SUPERVIVENCIA",
	SecLoadouts:    "EQUIPAMIENTO DE LA TRIPULACIÓN",
	SecFood:        "COMIDA Y BEBIDA",
	SecFuel:        "COMBUSTIBLE",
	SecMedical:     "MÉDICO Y CONSUMIBLES",
	SecWeaponry:    "ARMAS Y MUNICIÓN",
	SecGear:        "EQUIPO Y HERRAMIENTAS",
	SecMaterials:   "MATERIALES Y RECURSOS",
	SecSkills:      "HABILIDADES",
	SecAttributes:  "ATRIBUTOS",
	SecTraits:      "RASGOS",

	LoadoutNote: "solo lectura — la tripulación se equipa sola desde el cargo",
	EmptySlot:   "—",

	Credits:   "Créditos",
	Research:  "Investigación",
	CrewCount: "Tripulación",
	Unsaved:   "sin guardar",

	PromptNewValue: "Nuevo valor",
	PromptCredits:  "Créditos",
	PromptFirst:    "Nombre",
	PromptLast:     "Apellido",
	PromptSearch:   "Buscar por nombre o ID…",
	SearchTitle:    "Agregar recurso al cargo",

	Done:    "✓ completa",
	Pending: "pendiente",

	ConfirmQuitTitle: "Cambios sin guardar",
	ConfirmQuitBody:  "Tenés cambios que todavía no están en el archivo.",
	ConfirmSave:      "guardar y salir",
	ConfirmDiscard:   "salir sin guardar",
	ConfirmCancel:    "cancelar",

	StatusSaved:     "Save guardado. Backup: %s",
	StatusNoChanges: "No hay cambios para guardar.",
	StatusUpdated:   "%s actualizado: %s → %s",
	StatusInserted:  "%s agregado al cargo: %s",
	StatusAdded:     "Rasgo '%s' agregado.",
	StatusRemoved:   "Rasgo '%s' eliminado.",
	StatusCompleted: "'%s' marcada como completa.",
	StatusReset:     "'%s' reseteada.",
	StatusAllDone:   "%d tecnologías completadas.",
	StatusAllReset:  "%d tecnologías reseteadas.",
	StatusRenamed:   "Nombre actualizado: %s",

	KeysGlobal: "tab secciones · c créditos · s guardar · q salir",
	KeysCargo:  "enter editar · a agregar recurso",
	KeysResear: "enter alternar · A completar todo · R resetear todo",
	KeysCrew:   "enter abrir personaje",
	KeysDetail: "tab panel · enter editar · n renombrar · esc volver",
	KeysPicker: "enter abrir · q salir",
	KeysModal:  "enter confirmar · esc cancelar",
}

var textEN = uiText{
	AppTitle:    "Space Haven — Save Editor",
	PickTitle:   "Pick a save",
	PickWarning: "IMPORTANT: close the game before editing.",
	NoSaves:     "No saves were found in this folder.",
	NoSaveDir:   "The savegames folder was not found.",
	SearchedIn:  "Looked in:",
	DirTip:      "Point SPACEHAVEN_SAVEGAMES_DIR at the right folder.",

	ColSlot:    "Slot",
	ColShip:    "Ship",
	ColDate:    "Date",
	ColID:      "ID",
	ColName:    "Name",
	ColAmount:  "Amount",
	ColTech:    "Technology",
	ColStatus:  "Status",
	ColCrew:    "Crew member",
	ColHealth:  "Health",
	ColMood:    "Mood",
	ColRest:    "Rest",
	ColSkills:  "Skills",
	ColSkill:   "Skill",
	ColLevel:   "Level",
	ColMax:     "Max",
	ColAttr:    "Attribute",
	ColPoints:  "Points",
	ColTrait:   "Trait",
	ColWeapon:  "Weapon",
	ColItem:    "Item",
	ColInCargo: "In cargo",
	ColPrimary: "Primary",
	ColSecond:  "Secondary",
	ColArmor:   "Armor",
	ColHelmet:  "Headgear",

	NavCargo:    "Cargo",
	NavWeapons:  "Weapons",
	NavCrew:     "Crew",
	NavResearch: "Research",

	SecWeapons:     "WEAPONS",
	SecAttachments: "ATTACHMENTS",
	SecDefense:     "DEFENSE AND GEAR",
	SecArmor:       "ARMOR",
	SecHeadgear:    "HEADGEAR",
	SecSurvival:    "SURVIVAL",
	SecLoadouts:    "CREW LOADOUTS",
	SecFood:        "FOOD AND DRINK",
	SecFuel:        "FUEL",
	SecMedical:     "MEDICAL AND CONSUMABLES",
	SecWeaponry:    "WEAPONS AND AMMO",
	SecGear:        "GEAR AND TOOLS",
	SecMaterials:   "MATERIALS AND RESOURCES",
	SecSkills:      "SKILLS",
	SecAttributes:  "ATTRIBUTES",
	SecTraits:      "TRAITS",

	LoadoutNote: "read-only — the crew equip themselves from cargo",
	EmptySlot:   "—",

	Credits:   "Credits",
	Research:  "Research",
	CrewCount: "Crew",
	Unsaved:   "unsaved",

	PromptNewValue: "New value",
	PromptCredits:  "Credits",
	PromptFirst:    "First name",
	PromptLast:     "Last name",
	PromptSearch:   "Search by name or ID…",
	SearchTitle:    "Add a resource to cargo",

	Done:    "✓ done",
	Pending: "pending",

	ConfirmQuitTitle: "Unsaved changes",
	ConfirmQuitBody:  "You have changes that are not in the file yet.",
	ConfirmSave:      "save and quit",
	ConfirmDiscard:   "quit without saving",
	ConfirmCancel:    "cancel",

	StatusSaved:     "Save written. Backup: %s",
	StatusNoChanges: "Nothing to save.",
	StatusUpdated:   "%s updated: %s → %s",
	StatusInserted:  "%s added to cargo: %s",
	StatusAdded:     "Trait '%s' added.",
	StatusRemoved:   "Trait '%s' removed.",
	StatusCompleted: "'%s' marked as done.",
	StatusReset:     "'%s' reset.",
	StatusAllDone:   "%d technologies completed.",
	StatusAllReset:  "%d technologies reset.",
	StatusRenamed:   "Renamed to %s",

	KeysGlobal: "tab sections · c credits · s save · q quit",
	KeysCargo:  "enter edit · a add resource",
	KeysResear: "enter toggle · A complete all · R reset all",
	KeysCrew:   "enter open crew member",
	KeysDetail: "tab panel · enter edit · n rename · esc back",
	KeysPicker: "enter open · q quit",
	KeysModal:  "enter confirm · esc cancel",
}

func textFor(lang catalog.Language) uiText {
	if lang == catalog.English {
		return textEN
	}
	return textES
}
