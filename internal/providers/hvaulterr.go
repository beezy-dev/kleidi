package providers

import (
	"fmt"
	"regexp"
	"strings"
	"strconv"
)

// The hVaultErr struct wraps the Vault string-based error,
// providing structured fields for reliable error matching and inspection.
// It implements the `error` interface.
type hVaultErr struct {
	// originalError holds the full, original error string for context.
	originalError string

	// Extracted fields from the error string.
	StatusCode int
	Method     string
	URL        string
	Namespace  string
	Messages   []string
}

// Error implements the `error` interface, returning the original error string.
func (e *hVaultErr) Error() string {
	return e.originalError
}

// Unwrap returns a new error with the original wrapped error.
func (e *hVaultErr) Unwrap() error {
	return fmt.Errorf("%s", e.originalError)
}

// Is allows errors.Is to work directly with the hVaultErr type.
// It checks if the wrapped error matches the target sentinel error based on
// its status code and message.
func (e *hVaultErr) Is(target error) bool {
	// Check if the target is a *hVaultErr.
	targetErr, ok := target.(*hVaultErr)
	if !ok {
		return false
	}

	// Check if the status codes match.
	if e.StatusCode != targetErr.StatusCode {
		return false
	}

	// If the target error doesn't have an original message,
	// a status code match is sufficient.
	if targetErr.originalError == "" {
		return true
	}

	// If the target error has an original message,
	// check if any of the parsed messages in the receiver error
	// contain the target's original message.
	for _, msg := range e.Messages {
		if strings.Contains(strings.ToLower(msg), strings.ToLower(targetErr.originalError)) {
			return true
		}
	}

	return false
}

// Pre-defined constant errors for specific conditions.
var (
	ErrInvalidToken = &hVaultErr{StatusCode: 403, originalError: "invalid token"}
	ErrVaultSealed  = &hVaultErr{StatusCode: 503, originalError: "Vault is sealed"}
)

// WrapVaultError parses a raw Vault error string and wraps it
// in a structured hVaultErr. It uses regular expressions to
// extract key information.
func WrapVaultError(errString string) error {
	re := regexp.MustCompile(
		`URL: (\S+) (\S+)\s*` +
			`Code: (\d+)\. .*?:\s*` +
			`(?s)(.*)`)

	match := re.FindStringSubmatch(errString)
	if len(match) < 4 {
		// If parsing fails, just return a generic wrapped error.
		return fmt.Errorf("failed to parse Vault error string: %w", fmt.Errorf("%s", errString))
	}

	// Extract the components from the regex match.
	method := match[1]
	url := match[2]
	statusCode, err := strconv.Atoi(match[3])
	if err != nil {
		// If parsing fails, default the status code to 0.
		// This is a safer alternative to a potential Sscanf panic or unexpected behavior.
		statusCode = 0
	}

	errorBody := match[4]

	// Extract the namespace, which is optional.
	namespaceRe := regexp.MustCompile(`Namespace: (.+)\n`)
	namespaceMatch := namespaceRe.FindStringSubmatch(errString)
	namespace := ""
	if len(namespaceMatch) > 1 {
		namespace = strings.TrimSpace(namespaceMatch[1])
	}

	// Parse the individual error messages from the error body.
	var messages []string
	if strings.Contains(errorBody, "* ") {
		// Handles the case with multiple errors
		messageLines := strings.Split(strings.TrimSpace(errorBody), "\n")
		// Filter out the "* n error(s) occurred:" line and trim "* "
		for _, line := range messageLines {
			trimmedLine := strings.TrimSpace(line)
			if trimmedLine == "" {
				continue
			}
			if strings.HasPrefix(trimmedLine, "* ") {
				// Check for the "n errors occurred" or "1 error occurred" line and skip it
				if strings.HasSuffix(trimmedLine, " errors occurred:") || strings.HasSuffix(trimmedLine, " error occurred:")  {
					continue
				}
				messages = append(messages, strings.TrimPrefix(trimmedLine, "* "))
			} else {
				messages = append(messages, trimmedLine)
			}
		}
	} else {
		// Handles the case with a single raw message
		messages = append(messages, strings.TrimSpace(errorBody))
	}

	return &hVaultErr{
		originalError: errString,
		StatusCode:    statusCode,
		Method:        method,
		URL:           url,
		Namespace:     namespace,
		Messages:      messages,
	}
}
