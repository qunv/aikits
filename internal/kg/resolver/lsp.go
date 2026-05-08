package resolver

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// LSPClient manages a JSON-RPC LSP process.
type LSPClient struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  *bufio.Reader
	mu      sync.Mutex
	seq     int
	logFile *os.File // stderr log; nil if not capturing
}

type jsonrpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type jsonrpcNotification struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type jsonrpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *jsonrpcError   `json:"error,omitempty"`
}

type jsonrpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// rawFrame is used to distinguish responses from unsolicited server messages.
type rawFrame struct {
	ID     *json.Number    `json:"id"`
	Method string          `json:"method"`
	Result json.RawMessage `json:"result"`
	Error  *jsonrpcError   `json:"error,omitempty"`
}

// startProcess launches the given LSP executable and returns a connected client.
// The initialize handshake has NOT been performed yet.
// If logDir is non-empty, stderr is captured to logDir/<basename(name)>.log (truncated on each run).
func startProcess(name string, args []string, logDir string) (*LSPClient, error) {
	cmd := exec.Command(name, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("lsp stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("lsp stdout pipe: %w", err)
	}

	var logFile *os.File
	if logDir != "" {
		if mkErr := os.MkdirAll(logDir, 0o755); mkErr == nil {
			logName := filepath.Base(name) + ".log"
			if f, fErr := os.Create(filepath.Join(logDir, logName)); fErr == nil {
				logFile = f
				cmd.Stderr = f
			}
		}
	}

	if err := cmd.Start(); err != nil {
		if logFile != nil {
			_ = logFile.Close()
		}
		return nil, fmt.Errorf("lsp start: %w", err)
	}
	return &LSPClient{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  bufio.NewReader(stdout),
		logFile: logFile,
	}, nil
}

// Start launches the LSP process and performs the initialize handshake.
// logDir is the directory for stderr capture; empty string disables logging.
func Start(name string, args []string, rootURI, logDir string) (*LSPClient, error) {
	c, err := startProcess(name, args, logDir)
	if err != nil {
		return nil, err
	}
	initParams := map[string]any{
		"processId":             nil,
		"rootUri":               rootURI,
		"capabilities":          map[string]any{},
		"initializationOptions": map[string]any{},
	}
	if _, err := c.Call("initialize", initParams); err != nil {
		_ = c.Shutdown()
		return nil, fmt.Errorf("lsp initialize: %w", err)
	}
	if err := c.Notify("initialized", map[string]any{}); err != nil {
		_ = c.Shutdown()
		return nil, fmt.Errorf("lsp initialized notify: %w", err)
	}
	return c, nil
}

// Shutdown sends shutdown + exit to the LSP process and closes the log file.
func (c *LSPClient) Shutdown() error {
	_, _ = c.Call("shutdown", nil)
	_ = c.Notify("exit", nil)
	if c.cmd.Process != nil {
		_ = c.cmd.Process.Kill()
	}
	err := c.cmd.Wait()
	if c.logFile != nil {
		_ = c.logFile.Close()
	}
	return err
}

// Call sends a JSON-RPC request and returns the result bytes.
func (c *LSPClient) Call(method string, params any) (json.RawMessage, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.seq++
	seq := c.seq
	req := jsonrpcRequest{
		JSONRPC: "2.0",
		ID:      seq,
		Method:  method,
		Params:  params,
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	msg := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body)
	if _, err := io.WriteString(c.stdin, msg); err != nil {
		return nil, fmt.Errorf("write request: %w", err)
	}

	resp, err := c.readResponse(seq)
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("lsp error %d: %s", resp.Error.Code, resp.Error.Message)
	}
	return resp.Result, nil
}

// Notify sends a JSON-RPC notification (no response expected).
func (c *LSPClient) Notify(method string, params any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	notif := jsonrpcNotification{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}
	body, err := json.Marshal(notif)
	if err != nil {
		return fmt.Errorf("marshal notification: %w", err)
	}
	msg := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body)
	_, err = io.WriteString(c.stdin, msg)
	return err
}

// readResponse reads from stdout, skipping notifications and server-to-client
// requests, until a response with the given ID is found.
// jdtls (and other chatty servers) frequently send $/progress, window/logMessage,
// etc. while processing requests; those must be discarded here.
func (c *LSPClient) readResponse(expectedID int) (*jsonrpcResponse, error) {
	for {
		buf, err := c.readFrame()
		if err != nil {
			return nil, err
		}

		// Peek at the frame to decide whether it's a notification, a
		// server→client request, or an actual response.
		var frame rawFrame
		if err := json.Unmarshal(buf, &frame); err != nil {
			return nil, fmt.Errorf("unmarshal lsp frame: %w", err)
		}

		// Notifications have no id; server→client requests have method+id.
		// Both can be silently discarded — we don't handle server requests.
		if frame.Method != "" {
			continue
		}

		// Skip responses with an unexpected ID (shouldn't happen in practice).
		if frame.ID != nil {
			id, _ := frame.ID.Int64()
			if int(id) != expectedID {
				continue
			}
		}

		var resp jsonrpcResponse
		if err := json.Unmarshal(buf, &resp); err != nil {
			return nil, fmt.Errorf("unmarshal lsp response: %w", err)
		}
		return &resp, nil
	}
}

// readFrame reads one Content-Length-framed message from stdout.
func (c *LSPClient) readFrame() ([]byte, error) {
	var contentLen int
	for {
		line, err := c.stdout.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("read lsp header: %w", err)
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		if strings.HasPrefix(line, "Content-Length: ") {
			val := strings.TrimPrefix(line, "Content-Length: ")
			contentLen, _ = strconv.Atoi(strings.TrimSpace(val))
		}
	}

	if contentLen == 0 {
		return nil, fmt.Errorf("no content-length in lsp response")
	}

	buf := make([]byte, contentLen)
	if _, err := io.ReadFull(c.stdout, buf); err != nil {
		return nil, fmt.Errorf("read lsp body: %w", err)
	}
	return buf, nil
}

