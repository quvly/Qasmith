package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const maxMessage = 8 << 20

type message struct {
	Version string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// readFrame handles the bounded LSP stdio envelope. JSON-RPC dispatch stays
// in Server so invalid requests never enter the document or compiler API.
func readFrame(r *bufio.Reader) ([]byte, error) {
	length := -1
	for i := 0; i < 32; i++ {
		line, err := r.ReadSlice('\n')
		if err != nil {
			return nil, err
		}
		if len(line) < 2 || line[len(line)-2] != '\r' {
			return nil, fmt.Errorf("expected CRLF in LSP header")
		}
		text := string(line[:len(line)-2])
		if text == "" {
			if length < 0 {
				return nil, fmt.Errorf("missing Content-Length")
			}
			buf := make([]byte, length)
			_, err := io.ReadFull(r, buf)
			return buf, err
		}
		key, value, ok := strings.Cut(text, ":")
		if !ok {
			return nil, fmt.Errorf("malformed LSP header")
		}
		if strings.EqualFold(strings.TrimSpace(key), "Content-Length") {
			if length >= 0 {
				return nil, fmt.Errorf("duplicate Content-Length")
			}
			n, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil || n < 0 || n > maxMessage {
				return nil, fmt.Errorf("invalid Content-Length")
			}
			length = n
		}
	}
	return nil, fmt.Errorf("too many LSP headers")
}

func writeFrame(w io.Writer, data any) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Content-Length: %d\r\n\r\n", len(payload)); err != nil {
		return err
	}
	_, err = w.Write(payload)
	return err
}
