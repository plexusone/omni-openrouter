package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/plexusone/omnivault"
	"github.com/plexusone/omnivault/vault"
)

const (
	// OpenRouter OAuth endpoints
	AuthorizeURL = "https://openrouter.ai/auth"
	TokenURL     = "https://openrouter.ai/api/v1/auth/keys" //nolint:gosec // G101: This is a URL, not a credential

	// VaultKey is the key used to store the API key in omnivault.
	VaultKey = "openrouter:default"

	// AuthTimeout is the maximum time to wait for user authentication.
	AuthTimeout = 5 * time.Minute
)

// TokenResponse represents the response from OpenRouter's token endpoint.
type TokenResponse struct {
	Key string `json:"key"`
}

// Login performs the OAuth PKCE flow to authenticate with OpenRouter.
// It returns the API key on success.
func Login(ctx context.Context) (string, error) {
	// Generate PKCE values
	verifier, err := GenerateCodeVerifier()
	if err != nil {
		return "", fmt.Errorf("failed to generate code verifier: %w", err)
	}
	challenge := GenerateCodeChallenge(verifier)

	// Start callback server
	server, err := NewCallbackServer()
	if err != nil {
		return "", fmt.Errorf("failed to start callback server: %w", err)
	}
	defer func() { _ = server.Close() }()
	server.Start()

	// Build authorization URL
	authURL := buildAuthURL(challenge, server.RedirectURI())

	// Open browser
	if err := openBrowser(authURL); err != nil {
		return "", fmt.Errorf("failed to open browser: %w\nPlease visit: %s", err, authURL)
	}

	fmt.Println("Opening browser for OpenRouter authentication...")
	fmt.Println("If the browser doesn't open, visit:", authURL)

	// Wait for callback with timeout
	authCtx, cancel := context.WithTimeout(ctx, AuthTimeout)
	defer cancel()

	result, err := server.WaitForCallback(authCtx)
	if err != nil {
		return "", fmt.Errorf("authentication timed out: %w", err)
	}

	if result.Error != "" {
		return "", fmt.Errorf("authentication failed: %s", result.Error)
	}

	// Exchange code for API key
	apiKey, err := exchangeCode(ctx, result.Code, verifier)
	if err != nil {
		return "", fmt.Errorf("failed to exchange code: %w", err)
	}

	// Store in omnivault
	if err := storeAPIKey(ctx, apiKey); err != nil {
		// Log warning but don't fail - the key was still obtained
		fmt.Printf("Warning: failed to store API key in vault: %v\n", err)
	}

	return apiKey, nil
}

// LoadAPIKey retrieves the stored API key from omnivault.
func LoadAPIKey(ctx context.Context) (string, error) {
	client, err := omnivault.NewClient(omnivault.Config{
		Provider: omnivault.ProviderFile,
	})
	if err != nil {
		return "", fmt.Errorf("failed to open vault: %w", err)
	}
	defer func() { _ = client.Close() }()

	secret, err := client.Get(ctx, VaultKey)
	if err != nil {
		return "", fmt.Errorf("API key not found: %w", err)
	}

	return secret.String(), nil
}

// buildAuthURL constructs the OpenRouter authorization URL.
func buildAuthURL(challenge, redirectURI string) string {
	params := url.Values{
		"callback_url":          {redirectURI},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
	}
	return AuthorizeURL + "?" + params.Encode()
}

// exchangeCode exchanges the authorization code for an API key.
func exchangeCode(ctx context.Context, code, verifier string) (string, error) {
	data := url.Values{
		"code":          {code},
		"code_verifier": {verifier},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("token exchange failed (status %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if tokenResp.Key == "" {
		return "", fmt.Errorf("empty API key in response")
	}

	return tokenResp.Key, nil
}

// storeAPIKey stores the API key in omnivault.
func storeAPIKey(ctx context.Context, apiKey string) error {
	client, err := omnivault.NewClient(omnivault.Config{
		Provider: omnivault.ProviderFile,
	})
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()

	return client.Set(ctx, VaultKey, &vault.Secret{Value: apiKey})
}

// openBrowser opens the default browser to the given URL.
func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}
