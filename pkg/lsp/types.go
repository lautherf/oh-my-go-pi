package lsp

import "encoding/json"

type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

type Diagnostic struct {
	Range    Range   `json:"range"`
	Severity int     `json:"severity,omitempty"`
	Code     string  `json:"code,omitempty"`
	Source   string  `json:"source,omitempty"`
	Message  string  `json:"message"`
}

type DiagnosticSeverity int

const (
	SeverityError       DiagnosticSeverity = 1
	SeverityWarning     DiagnosticSeverity = 2
	SeverityInformation DiagnosticSeverity = 3
	SeverityHint        DiagnosticSeverity = 4
)

type TextDocumentItem struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

type VersionedTextDocumentIdentifier struct {
	URI     string `json:"uri"`
	Version int    `json:"version"`
}

type TextDocumentContentChangeEvent struct {
	Text string `json:"text"`
}

type InitializeParams struct {
	ProcessID             *int                   `json:"processId,omitempty"`
	ClientInfo            *ClientInfo            `json:"clientInfo,omitempty"`
	Capabilities          ClientCapabilities     `json:"capabilities"`
	Trace                 string                 `json:"trace,omitempty"`
}

type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

type ClientCapabilities struct {
	TextDocument *TextDocumentClientCapabilities `json:"textDocument,omitempty"`
	Workspace    *WorkspaceClientCapabilities    `json:"workspace,omitempty"`
}

type TextDocumentClientCapabilities struct {
	Completion         *CompletionCapability         `json:"completion,omitempty"`
	Hover              *HoverCapability              `json:"hover,omitempty"`
	Definition         *DefinitionCapability         `json:"definition,omitempty"`
	References         *ReferencesCapability         `json:"references,omitempty"`
	DocumentSymbol     *DocumentSymbolCapability     `json:"documentSymbol,omitempty"`
	Rename             *RenameCapability             `json:"rename,omitempty"`
	CodeAction         *CodeActionCapability         `json:"codeAction,omitempty"`
}

type CompletionCapability struct {
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`
}

type HoverCapability struct {
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`
	ContentFormat       []string `json:"contentFormat,omitempty"`
}

type DefinitionCapability struct {
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`
}

type ReferencesCapability struct {
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`
}

type DocumentSymbolCapability struct {
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`
	HierarchicalDocumentSymbolSupport *bool `json:"hierarchicalDocumentSymbolSupport,omitempty"`
}

type RenameCapability struct {
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`
	PrepareSupport      *bool `json:"prepareSupport,omitempty"`
}

type CodeActionCapability struct {
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`
	CodeActionLiteralSupport *struct{} `json:"codeActionLiteralSupport,omitempty"`
}

type WorkspaceClientCapabilities struct {
	DidChangeConfiguration *struct{} `json:"didChangeConfiguration,omitempty"`
}

type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
	ServerInfo   *ServerInfo        `json:"serverInfo,omitempty"`
}

type ServerCapabilities struct {
	TextDocumentSync   *TextDocumentSyncOptions `json:"textDocumentSync,omitempty"`
	HoverProvider      bool                     `json:"hoverProvider,omitempty"`
	DefinitionProvider bool                     `json:"definitionProvider,omitempty"`
	ReferencesProvider bool                     `json:"referencesProvider,omitempty"`
	CompletionProvider *CompletionOptions       `json:"completionProvider,omitempty"`
	RenameProvider     bool                     `json:"renameProvider,omitempty"`
	CodeActionProvider bool                     `json:"codeActionProvider,omitempty"`
	DocumentSymbolProvider bool                 `json:"documentSymbolProvider,omitempty"`
}

type TextDocumentSyncOptions struct {
	OpenClose bool `json:"openClose,omitempty"`
	Change    int  `json:"change,omitempty"`
	Save      *SaveOptions `json:"save,omitempty"`
}

type SaveOptions struct {
	IncludeText bool `json:"includeText,omitempty"`
}

type CompletionOptions struct {
	TriggerCharacters []string `json:"triggerCharacters,omitempty"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

type CompletionItem struct {
	Label         string             `json:"label"`
	Kind          int                `json:"kind,omitempty"`
	Detail        string             `json:"detail,omitempty"`
	Documentation string             `json:"documentation,omitempty"`
	InsertText    string             `json:"insertText,omitempty"`
}

type Hover struct {
	Contents MarkupContent `json:"contents"`
	Range    *Range        `json:"range,omitempty"`
}

type MarkupContent struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

type SymbolInformation struct {
	Name          string   `json:"name"`
	Kind          int      `json:"kind"`
	Location      Location `json:"location"`
	ContainerName string   `json:"containerName,omitempty"`
}

type DocumentSymbol struct {
	Name           string     `json:"name"`
	Detail         string     `json:"detail,omitempty"`
	Kind           int        `json:"kind"`
	Range          Range      `json:"range"`
	SelectionRange Range      `json:"selectionRange"`
	Children       []DocumentSymbol `json:"children,omitempty"`
}

type TextEdit struct {
	Range   Range  `json:"range"`
	NewText string `json:"newText"`
}

type WorkspaceEdit struct {
	Changes map[string][]TextEdit `json:"changes,omitempty"`
}

type CodeAction struct {
	Title   string         `json:"title"`
	Kind    string         `json:"kind,omitempty"`
	Edit    *WorkspaceEdit `json:"edit,omitempty"`
	Command *Command       `json:"command,omitempty"`
}

type Command struct {
	Title     string `json:"title"`
	Command   string `json:"command"`
	Arguments []json.RawMessage `json:"arguments,omitempty"`
}

type PublishDiagnosticsParams struct {
	URI         string       `json:"uri"`
	Diagnostics []Diagnostic `json:"diagnostics"`
	Version     *int         `json:"version,omitempty"`
}

type DidChangeTextDocumentParams struct {
	TextDocument   VersionedTextDocumentIdentifier      `json:"textDocument"`
	ContentChanges []TextDocumentContentChangeEvent      `json:"contentChanges"`
}

type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

type DidCloseTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

type DidSaveTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Text         string                 `json:"text,omitempty"`
}

type ReferenceParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
	Context      ReferenceContext       `json:"context"`
}

type ReferenceContext struct {
	IncludeDeclaration bool `json:"includeDeclaration"`
}

type RenameParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
	NewName      string                 `json:"newName"`
}

type CodeActionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Range        Range                  `json:"range"`
	Context      CodeActionContext       `json:"context"`
}

type CodeActionContext struct {
	Diagnostics []Diagnostic `json:"diagnostics"`
	Only        []string     `json:"only,omitempty"`
}

type DocumentSymbolParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

type CompletionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
	Context      *CompletionContext     `json:"context,omitempty"`
}

type CompletionContext struct {
	TriggerKind    int    `json:"triggerKind"`
	TriggerCharacter string `json:"triggerCharacter,omitempty"`
}

type CompletionList struct {
	IsIncomplete bool             `json:"isIncomplete"`
	Items        []CompletionItem `json:"items"`
}
