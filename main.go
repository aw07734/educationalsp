package main

import (
	"bufio"
	"educationalsp/analysis"
	"educationalsp/lsp"
	"educationalsp/rpc"
	"encoding/json"
	"io"
	"log"
	"os"
)

func main() {
	logger := getLogger("E:/My/educationalsp/output.log")
	logger.Println("Hey, I started!")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Split(rpc.Split)

	state := analysis.NewState()
	writer := os.Stdout

	for scanner.Scan() {
		msg := scanner.Bytes()
		method, contents, err := rpc.DecodeMessage(msg)
		if err != nil {
			logger.Fatalf("Got an error: %s\n", err)
			continue
		}
		handleMessage(logger, writer, &state, method, contents)
	}
	if err := scanner.Err(); err != nil {
		logger.Fatalln("error!")
	}
}

func handleMessage(logger *log.Logger, writer io.Writer, state *analysis.State, method string, contents []byte) {
	logger.Printf("Received msg with method: %s\n", method)
	switch method {
	case "initialize":
		var request lsp.InitializeRequest
		if err := json.Unmarshal(contents, &request); err != nil {
			logger.Fatalf("Hey, we couldn't parse this: %s\n", err)
			return
		}
		logger.Printf("Connected: %s\n", request.Params.RootUri)
		// Hey, let's reply!
		msg := lsp.NewInitializeResponse(request.ID)
		writeResponse(writer, msg)
		logger.Println("Sent the reply")
	case "textDocument/didOpen":
		var request lsp.DidOpenTextDocumentNotification
		if err := json.Unmarshal(contents, &request); err != nil {
			logger.Fatalf("textDocument/didOpen: %s\n", err)
			return
		}
		logger.Printf("Connected: %s\n", request.Params.TextDocument.Uri)
		diagnostics := state.OpenDocument(request.Params.TextDocument.Uri, request.Params.TextDocument.Text)
		writeResponse(writer, lsp.PublishDiagnosticsNotification{
			Notification: lsp.Notification{
				RPC:    "2.0",
				Method: "textDocument/publishDiagnostics",
			},
			Params: lsp.PublishDiagnosticsParams{
				Uri:         request.Params.TextDocument.Uri,
				Diagnostics: diagnostics,
			},
		})
	case "textDocument/didChange":
		var request lsp.TextDocumentDidChangeNotification
		if err := json.Unmarshal(contents, &request); err != nil {
			logger.Fatalf("textDocument/didChange: %s\n", err)
			return
		}
		logger.Printf("Changed: %s\n", request.Params.TextDocument.Uri)
		var diagnostics []lsp.Diagnostic
		for _, change := range request.Params.ContentChanges {
			diagnostics = state.UpdateDocument(request.Params.TextDocument.Uri, change.Text)
		}
		writeResponse(writer, lsp.PublishDiagnosticsNotification{
			Notification: lsp.Notification{
				RPC:    "2.0",
				Method: "textDocument/publishDiagnostics",
			},
			Params: lsp.PublishDiagnosticsParams{
				Uri:         request.Params.TextDocument.Uri,
				Diagnostics: diagnostics,
			},
		})
	case "textDocument/hover":
		var request lsp.HoverRequest
		if err := json.Unmarshal(contents, &request); err != nil {
			logger.Fatalf("textDocument/hover: %s\n", err)
			return
		}
		response := state.Hover(request.ID, request.Params.TextDocument.Uri, request.Params.Position)
		writeResponse(writer, response)
	case "textDocument/definition":
		var request lsp.DefinitionRequest
		if err := json.Unmarshal(contents, &request); err != nil {
			logger.Fatalf("textDocument/definition: %s\n", err)
			return
		}
		response := state.Definition(request.ID, request.Params.TextDocument.Uri, request.Params.Position)
		writeResponse(writer, response)
	case "textDocument/codeAction":
		var request lsp.CodeActionRequest
		if err := json.Unmarshal(contents, &request); err != nil {
			logger.Fatalf("textDocument/codeAction: %s\n", err)
			return
		}
		response := state.TextDocumentCodeAction(request.ID, request.Params.TextDocument.Uri)
		writeResponse(writer, response)
	case "textDocument/completion":
		var request lsp.CompletionRequest
		if err := json.Unmarshal(contents, &request); err != nil {
			logger.Fatalf("textDocument/completion: %s\n", err)
			return
		}
		response := state.TextDocumentCompletion(request.ID, request.Params.TextDocument.Uri, request.Params.Position)
		writeResponse(writer, response)
	case "exit":
		logger.Println("Exiting...")
		os.Exit(0)
	}
}

func writeResponse(writer io.Writer, msg any) {
	reply := rpc.EncodeMessage(msg)
	writer.Write([]byte(reply))
}

func getLogger(filename string) *log.Logger {
	logFile, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	if err != nil {
		panic("Hey, you didn't give me a good file")
	}
	return log.New(logFile, "[educationalsp]", log.Ldate|log.Ltime|log.Lshortfile)
}
