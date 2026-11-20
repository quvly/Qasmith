// Package lsp exposes compiler-owned language intelligence over LSP stdio.
package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"sync"
	"time"

	"github.com/quvly/Qasmith/pkg/qasmith"
)

type document struct {
	path, uri, source string
	version           int
}
type textDocument struct {
	URI     string `json:"uri"`
	Text    string `json:"text"`
	Version int    `json:"version"`
}
type documentParams struct {
	TextDocument   textDocument `json:"textDocument"`
	Position       Position     `json:"position"`
	ContentChanges []struct {
		Text  string `json:"text"`
		Range *Range `json:"range"`
	} `json:"contentChanges"`
	Text    *string `json:"text"`
	Context struct {
		IncludeDeclaration bool `json:"includeDeclaration"`
	} `json:"context"`
}
type workspaceFolder struct {
	URI string `json:"uri"`
}
type inbound struct {
	message    message
	parseError *rpcError
}
type findings struct {
	revision uint64
	analyses map[string]qasmith.EditorAnalysis
	failures map[string]error
}
type analysisJob struct {
	revision uint64
	docs     map[string]document
	ctx      context.Context
}

type server struct {
	out, log              io.Writer
	editor                qasmith.EditorSession
	docs                  map[string]document
	folders               []workspaceFolder
	previous              map[string]bool
	timer                 *time.Timer
	timerC                <-chan time.Time
	diagnostics           bool
	delay                 time.Duration
	initialized, shutdown bool
	hierarchicalSymbols   bool
	diagnosticVersions    bool
	revision              uint64
	cancelAnalysis        context.CancelFunc
	cache                 findings
	jobs                  chan analysisJob
	results               chan findings
	writeErr              error
	requestMu             sync.Mutex
	requests              map[string]context.CancelFunc
	cancellations         map[string]bool
}

// Serve runs a bounded stdio LSP endpoint. Protocol data goes only to out.
func Serve(input io.Reader, out, log io.Writer) error {
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	s := &server{out: out, log: log, docs: make(map[string]document), previous: make(map[string]bool),
		diagnostics: true, delay: 200 * time.Millisecond, jobs: make(chan analysisJob, 1), results: make(chan findings, 1),
		requests: make(map[string]context.CancelFunc), cancellations: make(map[string]bool)}
	defer s.stopAnalysis()
	messages := make(chan inbound, 32)
	readErrors := make(chan error, 1)
	go s.read(ctx, input, messages, readErrors)
	go s.analyze(ctx)
	for {
		// Process queued edits before considering results from an older revision.
		select {
		case m := <-messages:
			if done, err := s.handle(ctx, m); done {
				return err
			}
			if s.writeErr != nil {
				return s.writeErr
			}
			continue
		default:
		}
		select {
		case m := <-messages:
			if done, err := s.handle(ctx, m); done {
				return err
			}
		case err := <-readErrors:
			// Deferred EOF can follow an already queued exit notification.
			for len(messages) > 0 {
				if done, err := s.handle(ctx, <-messages); done {
					return err
				}
			}
			if errors.Is(err, io.EOF) {
				return nil
			}
			s.reply(nil, nil, &rpcError{-32700, "Invalid LSP framing"})
			return err
		case <-s.timerC:
			s.timerC = nil
			s.startAnalysis(ctx)
		case result := <-s.results:
			if result.revision == s.revision && !s.shutdown {
				s.cache = result
				if s.diagnostics {
					s.publish(result)
				}
			}
		}
		if s.writeErr != nil {
			return s.writeErr
		}
	}
}

func (s *server) read(ctx context.Context, input io.Reader, messages chan<- inbound, errs chan<- error) {
	r := bufio.NewReader(input)
	for {
		body, err := readFrame(r)
		if err != nil {
			errs <- err
			return
		}
		var m message
		var problem *rpcError
		switch {
		case !json.Valid(body):
			problem = &rpcError{-32700, "Parse error"}
		case json.Unmarshal(body, &m) != nil || m.Version != "2.0" || m.Method == "" || !validID(m.ID):
			problem = &rpcError{-32600, "Invalid Request"}
		}
		if problem == nil && m.Method == "$/cancelRequest" {
			var cancel struct {
				ID json.RawMessage `json:"id"`
			}
			if json.Unmarshal(m.Params, &cancel) == nil && validID(cancel.ID) && len(cancel.ID) > 0 {
				key := string(cancel.ID)
				s.requestMu.Lock()
				if stop := s.requests[key]; stop != nil {
					stop()
				} else if len(s.cancellations) < 128 {
					s.cancellations[key] = true
				}
				s.requestMu.Unlock()
			}
			continue
		}
		select {
		case messages <- inbound{m, problem}:
		case <-ctx.Done():
			return
		}
	}
}

func validID(id json.RawMessage) bool {
	return len(id) == 0 || id[0] == '"' || id[0] == '-' || id[0] >= '0' && id[0] <= '9'
}

func (s *server) emit(data any) {
	if s.writeErr == nil {
		s.writeErr = writeFrame(s.out, data)
	}
}

func (s *server) reply(id json.RawMessage, result any, err *rpcError) {
	if len(id) == 0 {
		id = json.RawMessage("null")
	}
	m := map[string]any{"jsonrpc": "2.0", "id": id}
	if err != nil {
		m["error"] = err
	} else {
		m["result"] = result
	}
	s.emit(m)
}
func (s *server) notify(method string, params any) {
	s.emit(map[string]any{"jsonrpc": "2.0", "method": method, "params": params})
}

func (s *server) handle(ctx context.Context, in inbound) (bool, error) {
	m := in.message
	if in.parseError != nil {
		s.reply(nil, nil, in.parseError)
		return false, nil
	}
	request := len(m.ID) > 0
	respond := func(result any, err *rpcError) {
		if request {
			s.reply(m.ID, result, err)
		}
	}
	if m.Method == "exit" {
		if !s.shutdown {
			return true, fmt.Errorf("exit before shutdown")
		}
		return true, nil
	}
	if s.shutdown {
		respond(nil, &rpcError{-32600, "Server is shut down"})
		return false, nil
	}
	if m.Method == "initialize" {
		if !request {
			return false, nil
		}
		if s.initialized {
			respond(nil, &rpcError{-32600, "Server already initialized"})
			return false, nil
		}
		var init struct {
			WorkspaceFolders      []workspaceFolder `json:"workspaceFolders"`
			InitializationOptions json.RawMessage   `json:"initializationOptions"`
			Capabilities          struct {
				TextDocument struct {
					PublishDiagnostics struct {
						VersionSupport bool `json:"versionSupport"`
					} `json:"publishDiagnostics"`
					DocumentSymbol struct {
						Hierarchical bool `json:"hierarchicalDocumentSymbolSupport"`
					} `json:"documentSymbol"`
				} `json:"textDocument"`
			} `json:"capabilities"`
		}
		if len(m.Params) == 0 || string(m.Params) == "null" || json.Unmarshal(m.Params, &init) != nil {
			respond(nil, &rpcError{-32602, "Invalid initialization parameters"})
			return false, nil
		}
		s.initialized = true
		s.folders = init.WorkspaceFolders
		s.hierarchicalSymbols = init.Capabilities.TextDocument.DocumentSymbol.Hierarchical
		s.diagnosticVersions = init.Capabilities.TextDocument.PublishDiagnostics.VersionSupport
		s.configure(init.InitializationOptions)
		respond(map[string]any{"serverInfo": map[string]string{"name": "Qasmith", "version": "0.9.0"},
			"capabilities": map[string]any{
				"positionEncoding": "utf-16", "textDocumentSync": map[string]any{"openClose": true, "change": 1, "save": map[string]bool{"includeText": true}},
				"completionProvider": map[string]any{"triggerCharacters": []string{"{", ":", ">"}},
				"hoverProvider":      true, "definitionProvider": true, "referencesProvider": true, "documentSymbolProvider": true,
				"signatureHelpProvider": map[string]any{"triggerCharacters": []string{"(", ","}},
				"workspace":             map[string]any{"workspaceFolders": map[string]bool{"supported": true, "changeNotifications": true}},
			}}, nil)
		return false, nil
	}
	if !s.initialized {
		respond(nil, &rpcError{-32002, "Server not initialized"})
		return false, nil
	}
	switch m.Method {
	case "shutdown":
		if request {
			s.shutdown = true
			s.stopAnalysis()
			respond(nil, nil)
		}
	case "initialized", "$/setTrace":
	case "workspace/didChangeConfiguration":
		var p struct {
			Settings json.RawMessage `json:"settings"`
		}
		if json.Unmarshal(m.Params, &p) == nil {
			s.configure(p.Settings)
			s.invalidate()
		}
	case "workspace/didChangeWatchedFiles":
		s.invalidate()
	case "workspace/didChangeWorkspaceFolders":
		var p struct {
			Event struct{ Added, Removed []workspaceFolder }
		}
		if json.Unmarshal(m.Params, &p) == nil {
			for _, removed := range p.Event.Removed {
				s.folders = slices.DeleteFunc(s.folders, func(f workspaceFolder) bool { return f == removed })
			}
			s.folders = append(s.folders, p.Event.Added...)
			s.invalidate()
		}
	case "textDocument/didOpen", "textDocument/didChange", "textDocument/didSave", "textDocument/didClose":
		var p documentParams
		if json.Unmarshal(m.Params, &p) == nil {
			s.update(m.Method, p)
		}
	case "textDocument/completion", "textDocument/hover", "textDocument/definition", "textDocument/references", "textDocument/signatureHelp", "textDocument/documentSymbol":
		if !request {
			break
		}
		var p documentParams
		if json.Unmarshal(m.Params, &p) != nil || p.Position.Line < 0 || p.Position.Character < 0 {
			respond(nil, &rpcError{-32602, "Invalid position or document parameters"})
			break
		}
		job, stop := context.WithCancel(ctx)
		key := string(m.ID)
		s.requestMu.Lock()
		s.requests[key] = stop
		if s.cancellations[key] {
			stop()
			delete(s.cancellations, key)
		}
		s.requestMu.Unlock()
		result, err := s.feature(job, m.Method, p)
		s.requestMu.Lock()
		delete(s.requests, key)
		s.requestMu.Unlock()
		switch {
		case job.Err() != nil:
			respond(nil, &rpcError{-32800, "Request cancelled"})
		case err != nil:
			respond(nil, &rpcError{-32602, err.Error()})
		default:
			respond(result, nil)
		}
		stop()
	default:
		respond(nil, &rpcError{-32601, "Method not found"})
	}
	return false, nil
}

func (s *server) update(method string, p documentParams) {
	path, ok := pathFromURI(p.TextDocument.URI)
	if !ok {
		return
	}
	old, exists := s.docs[path]
	switch method {
	case "textDocument/didOpen":
		if len(s.docs) >= 64 && !exists || len(p.TextDocument.Text) > 1<<20 {
			fmt.Fprintln(s.log, "Qasmith document limit exceeded")
			return
		}
		s.docs[path] = document{path, p.TextDocument.URI, p.TextDocument.Text, p.TextDocument.Version}
	case "textDocument/didChange":
		if !exists || p.TextDocument.Version <= old.version || len(p.ContentChanges) == 0 {
			return
		}
		for _, change := range p.ContentChanges {
			if change.Range != nil || len(change.Text) > 1<<20 {
				return
			}
			old.source = change.Text
		}
		old.version = p.TextDocument.Version
		s.docs[path] = old
	case "textDocument/didSave":
		if !exists {
			return
		}
		if p.Text != nil && len(*p.Text) <= 1<<20 {
			old.source = *p.Text
			s.docs[path] = old
		}
	case "textDocument/didClose":
		delete(s.docs, path)
		s.notify("textDocument/publishDiagnostics", map[string]any{"uri": p.TextDocument.URI, "diagnostics": []any{}})
		delete(s.previous, path)
	}
	s.invalidate()
}

func (s *server) configure(raw json.RawMessage) {
	var cfg struct {
		Diagnostics struct {
			Enabled    *bool `json:"enabled"`
			DebounceMs *int  `json:"debounceMs"`
		} `json:"diagnostics"`
		Qasmith json.RawMessage `json:"qasmith"`
	}
	if json.Unmarshal(raw, &cfg) != nil {
		return
	}
	if len(cfg.Qasmith) > 0 {
		json.Unmarshal(cfg.Qasmith, &cfg)
	}
	if cfg.Diagnostics.Enabled != nil {
		s.diagnostics = *cfg.Diagnostics.Enabled
	}
	if cfg.Diagnostics.DebounceMs != nil {
		s.delay = time.Duration(min(2000, max(0, *cfg.Diagnostics.DebounceMs))) * time.Millisecond
	}
}

func (s *server) stopAnalysis() {
	if s.timer != nil {
		s.timer.Stop()
		s.timerC = nil
	}
	if s.cancelAnalysis != nil {
		s.cancelAnalysis()
		s.cancelAnalysis = nil
	}
}

func (s *server) invalidate() {
	s.stopAnalysis()
	s.revision++
	s.cache = findings{}
	if !s.diagnostics {
		for path := range s.previous {
			s.notify("textDocument/publishDiagnostics", map[string]any{"uri": s.uri(path), "diagnostics": []any{}})
		}
		clear(s.previous)
		return
	}
	s.timer = time.NewTimer(s.delay)
	s.timerC = s.timer.C
}

func (s *server) startAnalysis(ctx context.Context) {
	job, stop := context.WithCancel(ctx)
	s.cancelAnalysis = stop
	payload := analysisJob{s.revision, cloneDocs(s.docs), job}
	select {
	case <-s.jobs:
	default:
	}
	s.jobs <- payload
}

func cloneDocs(docs map[string]document) map[string]document {
	out := make(map[string]document, len(docs))
	for path, doc := range docs {
		out[path] = doc
	}
	return out
}
func overlays(docs map[string]document) map[string]string {
	out := make(map[string]string, len(docs))
	for path, doc := range docs {
		out[path] = doc.source
	}
	return out
}
func (s *server) analyze(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case job := <-s.jobs:
			out := findings{job.revision, make(map[string]qasmith.EditorAnalysis), make(map[string]error)}
			source := overlays(job.docs)
			for path := range job.docs {
				result, err := s.editor.Analyze(job.ctx, path, source)
				if job.ctx.Err() != nil {
					break
				}
				if err != nil {
					out.failures[path] = err
				} else {
					out.analyses[path] = result
				}
			}
			if job.ctx.Err() != nil {
				continue
			}
			select {
			case s.results <- out:
			case <-job.ctx.Done():
			}
		}
	}
}

func (s *server) uri(path string) string {
	if doc, ok := s.docs[path]; ok {
		return doc.uri
	}
	return fileURI(path)
}
func (s *server) source(path string) string {
	if doc, ok := s.docs[path]; ok {
		return doc.source
	}
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	data, _ := io.ReadAll(io.LimitReader(f, 1<<20))
	return string(data)
}
