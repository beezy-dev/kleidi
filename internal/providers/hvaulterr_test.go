package providers

import (
	"fmt"
	"errors"
	"strings"
	"testing"
)

// Test cases for the WrapVaultError function.
func TestWrapVaultError(t *testing.T) {
	testCases := []struct {
		name       string
		input      string
		expectErr  bool
		statusCode int
		method     string
		url        string
		namespace  string
		messages   []string
	}{
		{
			name: "403 Invalid Token Error",
			input: `Error making API request.

URL: GET https://127.0.0.1:8200/v1/kms/transit/keys/kms-key-aes256
Code: 403. Errors:

* 2 errors occurred:
        * permission denied
        * invalid token`,
			expectErr:  false,
			statusCode: 403,
			method:     "GET",
			url:        "https://127.0.0.1:8200/v1/kms/transit/keys/kms-key-aes256",
			namespace:  "",
			messages:   []string{"permission denied", "invalid token"},
		},
		{
			name: "403 Invalid Token Error with namespace",
			input: `Error making API request.

Namespace: my-namespace
URL: GET https://127.0.0.1:8200/v1/kms/transit/keys/kms-key-aes256
Code: 403. Errors:

* 2 errors occurred:
        * permission denied
        * invalid token`,
			expectErr:  false,
			statusCode: 403,
			method:     "GET",
			url:        "https://127.0.0.1:8200/v1/kms/transit/keys/kms-key-aes256",
			namespace:  "my-namespace",
			messages:   []string{"permission denied", "invalid token"},
		},
		{
			name: "503 Vault is sealed",
			input: `Error making API request.

URL: GET https://127.0.0.1:8200/v1/kms/transit/keys/kms-key-aes256
Code: 503. Errors:

* Vault is sealed`,
			expectErr:  false,
			statusCode: 503,
			method:     "GET",
			url:        "https://127.0.0.1:8200/v1/kms/transit/keys/kms-key-aes256",
			namespace:  "",
			messages:   []string{"Vault is sealed"},
		},
		{
			name: "503 Vault is sealed with namespace",
			input: `Error making API request.

Namespace: my-namespace
URL: GET https://127.0.0.1:8200/v1/kms/transit/keys/kms-key-aes256
Code: 503. Errors:

* Vault is sealed`,
			expectErr:  false,
			statusCode: 503,
			method:     "GET",
			url:        "https://127.0.0.1:8200/v1/kms/transit/keys/kms-key-aes256",
			namespace:  "my-namespace",
			messages:   []string{"Vault is sealed"},
		},
		{
			name: "403 lookup-self policy missing with namespace",
			input: `Error making API request.

Namespace: root
URL: GET https://127.0.0.1:8200/v1/auth/token/lookup-self
Code: 403. Errors:

* 1 error occurred:
        * permission denied`,
			expectErr:  false,
			statusCode: 403,
			method:     "GET",
			url:        "https://127.0.0.1:8200/v1/auth/token/lookup-self",
			namespace:  "root",
			messages:   []string{"permission denied"},
		},
		{
			name: "403 renew-self policy missing with namespace",
			input: `Error making API request.

Namespace: root
URL: PUT https://127.0.0.1:8200/v1/auth/token/renew-self
Code: 403. Errors:

* 1 error occurred:
        * permission denied`,
			expectErr:  false,
			statusCode: 403,
			method:     "PUT",
			url:        "https://127.0.0.1:8200/v1/auth/token/renew-self",
			namespace:  "root",
			messages:   []string{"permission denied"},
		},
		{
			name: "Unparsable Error String",
			input: `This is a completely different error format.
It should not be parsed correctly.`,
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := WrapVaultError(tc.input)

			if tc.expectErr {
				if err == nil {
					t.Fatalf("expected an error, but got nil")
				}
				// We expect a generic wrapped error here
				if !strings.Contains(err.Error(), "failed to parse") {
					t.Errorf("expected a parsing error, but got: %v", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("expected no error, but got nil")
			}

			hErr, ok := err.(*hVaultErr)
			if !ok {
				t.Fatalf("expected error of type *hVaultErr, but got %T", err)
			}

			if hErr.StatusCode != tc.statusCode {
				t.Errorf("expected status code %d, but got %d", tc.statusCode, hErr.StatusCode)
			}
			if hErr.Method != tc.method {
				t.Errorf("expected method %s, but got %s", tc.method, hErr.Method)
			}
			if hErr.URL != tc.url {
				t.Errorf("expected URL %s, but got %s", tc.url, hErr.URL)
			}
			if hErr.Namespace != tc.namespace {
				t.Errorf("expected namespace %s, but got %s", tc.namespace, hErr.Namespace)
			}
			if len(hErr.Messages) != len(tc.messages) {
				t.Fatalf("expected %d: \"%v\" messages, but got %d: \"%v\"", len(tc.messages), tc.messages, len(hErr.Messages), hErr.Messages)
			}
			for i, msg := range tc.messages {
				if hErr.Messages[i] != msg {
					t.Errorf("message at index %d: expected %s, but got %s", i, msg, hErr.Messages[i])
				}
			}
		})
	}
}

// TestErrorsIs is a test function to verify how errors.Is works with hVaultErr.
func TestErrorsIs(t *testing.T) {
	// Wrapped errors from the parser.
	wrappedInvalidTokenErr := WrapVaultError(`Error making API request.

Namespace: root
URL: GET https://127.0.0.1:8200/v1/kms/transit/keys/kms-key-aes256
Code: 403. Errors:

* 2 errors occurred:
        * permission denied
        * invalid token`)
	
	wrappedVaultSealedErr := WrapVaultError(`Error making API request.

Namespace: root
URL: GET https://127.0.0.1:8200/v1/kms/transit/keys/kms-key-aes256
Code: 503. Errors:

* Vault is sealed`)

	// A deeply wrapped error.
	deeplyWrappedErr := fmt.Errorf("a layer of wrapping: %w", wrappedInvalidTokenErr)

	// A non-constant hVaultErr instance with matching status and message.
	customSealedErr := &hVaultErr{StatusCode: 503, originalError: "Vault is sealed"}

	// A completely different error.
	nonMatchingErr := errors.New("a completely different error")

	t.Run("Direct comparison with errors.Is", func(t *testing.T) {
		if !errors.Is(wrappedInvalidTokenErr, ErrInvalidToken) {
			t.Errorf("errors.Is should have returned true for a direct match, but it returned false")
		}
	})

	t.Run("Deeply wrapped comparison with errors.Is", func(t *testing.T) {
		if !errors.Is(deeplyWrappedErr, ErrInvalidToken) {
			t.Errorf("errors.Is should have unwrapped and found the correct error, but it returned false")
		}
	})

	t.Run("Another direct comparison with errors.Is", func(t *testing.T) {
		if !errors.Is(wrappedVaultSealedErr, ErrVaultSealed) {
			t.Errorf("errors.Is should have returned true for a direct match, but it returned false")
		}
	})
	
	t.Run("Comparison with a non-constant matching error", func(t *testing.T) {
		if !errors.Is(wrappedVaultSealedErr, customSealedErr) {
			t.Errorf("errors.Is should have returned true for a non-constant match, but it returned false")
		}
	})

	t.Run("Non-matching error comparison", func(t *testing.T) {
		if errors.Is(wrappedInvalidTokenErr, nonMatchingErr) {
			t.Errorf("errors.Is should have returned false for a non-matching error, but it returned true")
		}
	})
}
