package hclparser

// SchemaVersion is incremented when the JSON representation changes in an
// incompatible way.
const SchemaVersion = "3"

const RootModuleAddress = "root"

type Result struct {
	SchemaVersion string        `json:"schemaVersion"`
	Module        Module        `json:"module"`
	Modules       []Module      `json:"modules"`
	Files         []File        `json:"files"`
	Declarations  []Declaration `json:"declarations"`
	Diagnostics   []Diagnostic  `json:"diagnostics"`
}

type Module struct {
	Address string `json:"address"`
	Path    string `json:"path"`
	Source  string `json:"source,omitempty"`
	Parent  string `json:"parent,omitempty"`
}

type File struct {
	Path          string `json:"path"`
	Bytes         int64  `json:"bytes"`
	ModuleAddress string `json:"moduleAddress"`
	SourcePath    string `json:"-"`
}

type DeclarationKind string

const (
	DeclarationResource DeclarationKind = "resource"
	DeclarationData     DeclarationKind = "data"
	DeclarationModule   DeclarationKind = "module"
	DeclarationVariable DeclarationKind = "variable"
	DeclarationLocal    DeclarationKind = "local"
	DeclarationOutput   DeclarationKind = "output"
)

type Declaration struct {
	Kind           DeclarationKind `json:"kind"`
	Address        string          `json:"address"`
	ModuleAddress  string          `json:"moduleAddress"`
	ModuleSource   string          `json:"moduleSource,omitempty"`
	Name           string          `json:"name"`
	Type           string          `json:"type,omitempty"`
	File           string          `json:"file"`
	Range          Range           `json:"range"`
	SelectionRange Range           `json:"selectionRange"`
	References     []Reference     `json:"references"`
}

// Reference is a syntactic HCL traversal found inside a declaration. The
// parser records it without deciding what Terraform symbol it refers to.
type Reference struct {
	Text          string   `json:"text"`
	Traversal     []string `json:"traversal"`
	AttributePath string   `json:"attributePath"`
	Range         Range    `json:"range"`
}

type Diagnostic struct {
	Severity string `json:"severity"`
	Summary  string `json:"summary"`
	Detail   string `json:"detail,omitempty"`
	Subject  *Range `json:"subject,omitempty"`
}

type Range struct {
	File  string   `json:"file,omitempty"`
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type Position struct {
	Line   int `json:"line"`
	Column int `json:"column"`
	Byte   int `json:"byte"`
}
