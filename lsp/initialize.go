package lsp

type InitializeRequest struct {
	Request
	Params InitializeRequestParams `json:"params"`
}

type InitializeRequestParams struct {
	ClientInfo       *ClientInfo       `json:"clientInfo"`
	RootUri          string            `json:"rootUri"`
	WorkspaceFolders []WorkspaceFolder `json:"workspaceFolders"`
	// there's tons more that goes here
}

type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type WorkspaceFolder struct {
	URI  string `json:"uri"`
	Name string `json:"name"`
}

type InitializeResponse struct {
	Response
	Result InitializeResponseResult `json:"result"`
}

type InitializeResponseResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
	ServerInfo   *ServerInfo        `json:"serverInfo"`
}

type ServerCapabilities struct {
	TextDocumentSync   TextDocumentSyncKind `json:"textDocumentSync"`
	HoverProvider      bool                 `json:"hoverProvider"`
	DefinitionProvider bool                 `json:"definitionProvider"`
	CodeActionProvider bool                 `json:"codeActionProvider"`
	CompletionProvider map[string]any       `json:"completionProvider"`
}

type TextDocumentSyncKind int

const (
	None TextDocumentSyncKind = iota
	Full
	Incremental
)

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func NewInitializeResponse(id int) InitializeResponse {
	return InitializeResponse{
		Response: Response{
			RPC: "2.0",
			ID:  &id,
		},
		Result: InitializeResponseResult{
			Capabilities: ServerCapabilities{
				TextDocumentSync:   Full,
				HoverProvider:      true,
				DefinitionProvider: true,
				CodeActionProvider: true,
				CompletionProvider: map[string]any{},
			},
			ServerInfo: &ServerInfo{
				Name:    "educationalsp",
				Version: "0.0.0.0.0.0-beta1.final",
			},
		},
	}
}
