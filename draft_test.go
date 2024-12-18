package main

import (
	"testing"
)

func TestValidateLinkName(t *testing.T) {
	tests := []struct {
		name       string // Test case name
		input      string // Input to validateLinkName
		shouldFail bool   // Whether validation should fail
	}{
		// Valid inputs
		{"ValidName", "about", false},
		{"ValidNameWithUnderscore", "good_name_123", false},
		{"ValidNameWithDash", "valid-link", false},

		// Invalid inputs
		{"DotAsName", ".", true},
		{"DoubleDotAsName", "..", true},
		{"ContainsPathTraversal", "abc/../def", true},
		{"ContainsIllegalCharacters", "invalid<link>", true},
		{"LeadingWhitespace", "  leading-space", true},
		{"TrailingWhitespace", "trailing-space  ", true},
		{"ContainsNewline", "newline\n", true},
		{"TooLongName", "a" + makeString(300), true}, // Too long input
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateLinkName(tt.input)
			if tt.shouldFail && err == nil {
				t.Errorf("validateLinkName(%q) = nil; expected error", tt.input)
			} else if !tt.shouldFail && err != nil {
				t.Errorf("validateLinkName(%q) = %v; expected nil", tt.input, err)
			}
		})
	}
}

// makeString generates a string of 'n' 'a' characters for testing long inputs.
func makeString(n int) string {
	result := make([]rune, n)
	for i := 0; i < n; i++ {
		result[i] = 'a'
	}
	return string(result)
}

func TestValidateHeaders(t *testing.T) {
	knownHeaders := map[string]bool{
		"title":       true,
		"link":        true,
		"published":   true,
		"template":    true,
		"description": true,
		"tags":        false,
		"favicon":     false,
		"author":      false,
	}

	tests := []struct {
		name          string
		headers       map[string]string
		knownHeaders  map[string]bool
		filePath      string
		shouldFail    bool
		expectedError string
	}{
		// Valid case
		{
			name: "All required headers present",
			headers: map[string]string{
				"title":       "My Post",
				"link":        "https://example.com",
				"published":   "2024-12-17",
				"template":    "default",
				"description": "A post description",
			},
			knownHeaders: knownHeaders,
			filePath:     "post1.yaml",
			shouldFail:   false,
		},
		// Missing multiple required headers
		{
			name: "Missing required headers",
			headers: map[string]string{
				"title": "My Post",
				"link":  "https://example.com",
			},
			knownHeaders: knownHeaders,
			filePath:     "post2.yaml",
			shouldFail:   true,
			expectedError: `Post post2.yaml has the following issues:
missing a required header: description
missing a required header: published
missing a required header: template`,
		},
		// Unknown header present and missing a required header
		{
			name: "Unknown header present",
			headers: map[string]string{
				"title":     "My Post",
				"link":      "https://example.com",
				"published": "2024-12-17",
				"unknown":   "invalid",
			},
			knownHeaders: knownHeaders,
			filePath:     "post3.yaml",
			shouldFail:   true,
			expectedError: `Post post3.yaml has the following issues:
missing a required header: description
missing a required header: template
contains unknown header: unknown`,
		},
		// All required headers plus optional headers
		{
			name: "Extra non-required headers",
			headers: map[string]string{
				"title":       "My Post",
				"link":        "https://example.com",
				"published":   "2024-12-17",
				"template":    "default",
				"description": "A description",
				"tags":        "go, testing",
				"favicon":     "icon.png",
			},
			knownHeaders: knownHeaders,
			filePath:     "post4.yaml",
			shouldFail:   false,
		},
		// Empty headers map
		{
			name:         "Empty headers",
			headers:      map[string]string{},
			knownHeaders: knownHeaders,
			filePath:     "post5.yaml",
			shouldFail:   true,
			expectedError: `Post post5.yaml has the following issues:
missing a required header: description
missing a required header: link
missing a required header: published
missing a required header: template
missing a required header: title`,
		},
		// Empty knownHeaders map
		{
			name: "Empty known headers",
			headers: map[string]string{
				"title": "My Post",
			},
			knownHeaders: map[string]bool{},
			filePath:     "post6.yaml",
			shouldFail:   true,
			expectedError: `Post post6.yaml has the following issues:
contains unknown header: title`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateHeaders(tt.headers, tt.knownHeaders, tt.filePath)

			if tt.shouldFail {
				if err == nil {
					t.Errorf("Expected an error but got nil for test case: %s", tt.name)
				} else if tt.expectedError != "" && err.Error() != tt.expectedError {
					t.Errorf("Unexpected error: got %q, want %q", err.Error(), tt.expectedError)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for test case %s: %v", tt.name, err)
				}
			}
		})
	}
}
