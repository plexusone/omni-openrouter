package auth

import (
	"context"
	"fmt"
	"html"
	"net"
	"net/http"
	"time"
)

// CallbackResult holds the result of an OAuth callback.
type CallbackResult struct {
	Code  string
	Error string
}

// CallbackServer manages a local HTTP server for OAuth callbacks.
type CallbackServer struct {
	listener net.Listener
	server   *http.Server
	result   chan CallbackResult
}

// NewCallbackServer creates a new callback server on a random available port.
func NewCallbackServer() (*CallbackServer, error) {
	// Find an available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("failed to create listener: %w", err)
	}

	cs := &CallbackServer{
		listener: listener,
		result:   make(chan CallbackResult, 1),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", cs.handleCallback)

	cs.server = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	return cs, nil
}

// Start starts the callback server in a goroutine.
func (cs *CallbackServer) Start() {
	go func() {
		_ = cs.server.Serve(cs.listener)
	}()
}

// Port returns the port the server is listening on.
func (cs *CallbackServer) Port() int {
	return cs.listener.Addr().(*net.TCPAddr).Port
}

// RedirectURI returns the full redirect URI for OAuth.
func (cs *CallbackServer) RedirectURI() string {
	return fmt.Sprintf("http://127.0.0.1:%d/callback", cs.Port())
}

// WaitForCallback waits for the OAuth callback with a timeout.
func (cs *CallbackServer) WaitForCallback(ctx context.Context) (CallbackResult, error) {
	select {
	case result := <-cs.result:
		return result, nil
	case <-ctx.Done():
		return CallbackResult{}, ctx.Err()
	}
}

// Close shuts down the callback server.
func (cs *CallbackServer) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return cs.server.Shutdown(ctx)
}

// handleCallback processes the OAuth callback.
func (cs *CallbackServer) handleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	errMsg := r.URL.Query().Get("error")

	if errMsg != "" {
		cs.result <- CallbackResult{Error: errMsg}
		_, _ = fmt.Fprintf(w, `
<!DOCTYPE html>
<html>
<head><title>Authentication Failed</title></head>
<body>
<h1>Authentication Failed</h1>
<p>Error: %s</p>
<p>You can close this window.</p>
</body>
</html>`, html.EscapeString(errMsg))
		return
	}

	if code == "" {
		cs.result <- CallbackResult{Error: "no authorization code received"}
		_, _ = w.Write([]byte(`
<!DOCTYPE html>
<html>
<head><title>Authentication Failed</title></head>
<body>
<h1>Authentication Failed</h1>
<p>No authorization code received.</p>
<p>You can close this window.</p>
</body>
</html>`))
		return
	}

	cs.result <- CallbackResult{Code: code}
	_, _ = w.Write([]byte(`
<!DOCTYPE html>
<html>
<head><title>Authentication Successful</title></head>
<body>
<h1>Authentication Successful!</h1>
<p>You have successfully logged in to OpenRouter.</p>
<p>You can close this window and return to the terminal.</p>
<script>setTimeout(function() { window.close(); }, 2000);</script>
</body>
</html>`))
}
