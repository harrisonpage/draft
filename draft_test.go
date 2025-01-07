package main

import (
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
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
		{"TooLongName", "a" + makeFakeString(300), true}, // Too long input
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

func makeFakeString(n int) string {
	result := make([]rune, n)
	for i := 0; i < n; i++ {
		result[i] = 'a'
	}
	return string(result)
}

func sorted(input string) string {
	lines := strings.Split(input, "\n")
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

func TestValidateHeaders(t *testing.T) {
	tests := []struct {
		name          string
		frontMatter   FrontMatter
		filePath      string
		shouldFail    bool
		expectedError string
	}{
		// Test 1: Valid case
		{
			name: "All required headers present",
			frontMatter: FrontMatter{
				Title:       "My Post",
				Link:        "https://example.com",
				Published:   "2024-12-17",
				Template:    "default",
				Description: "A post description",
				Status:      "public",
			},
			filePath:   "post1.yaml",
			shouldFail: false,
		},
		// Test 2: Missing multiple required headers
		{
			name: "Missing required headers",
			frontMatter: FrontMatter{
				Title:  "My Post",
				Link:   "https://example.com",
				Status: "public",
			},
			filePath:   "post2.yaml",
			shouldFail: true,
			expectedError: `Post post2.yaml has the following issues:
missing a required header: description
missing a required header: published
missing a required header: template`,
		},
		// Test 3: Unknown header present and missing a required header
		{
			name: "Unknown header present",
			frontMatter: FrontMatter{
				Title:     "My Post",
				Link:      "https://example.com",
				Published: "2024-12-17",
				Status:    "public",
			},
			filePath:   "post3.yaml",
			shouldFail: true,
			expectedError: `Post post3.yaml has the following issues:
missing a required header: description
missing a required header: template`,
		},
		// Test 4: All required headers plus optional headers
		{
			name: "Extra non-required headers",
			frontMatter: FrontMatter{
				Title:       "My Post",
				Link:        "https://example.com",
				Published:   "2024-12-17",
				Template:    "default",
				Description: "A description",
				Tags:        []string{"go", "testing"},
				Favicon:     "icon.png",
				Status:      "public",
			},
			filePath:   "post4.yaml",
			shouldFail: false,
		},
		// Test 5: Empty headers
		{
			name:        "Empty headers",
			frontMatter: FrontMatter{},
			filePath:    "post5.yaml",
			shouldFail:  true,
			expectedError: `Post post5.yaml has the following issues:
missing a required header: description
missing a required header: link
missing a required header: published
missing a required header: template
missing a required header: title
Invalid value for status: `,
		},
		// Test 6: Empty knownHeaders map
		{
			name: "Empty known headers",
			frontMatter: FrontMatter{
				Title: "My Post",
			},
			filePath:   "post6.yaml",
			shouldFail: true,
			expectedError: `Post post6.yaml has the following issues:
missing a required header: description
missing a required header: link
missing a required header: published
missing a required header: template
Invalid value for status: `,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateHeaders(tt.frontMatter, tt.filePath)

			if tt.shouldFail {
				if err == nil {
					t.Errorf("Test #%d: Expected an error but got nil for test case: %s", 1+i, tt.name)
				} else {
					expected := sorted(tt.expectedError)
					got := sorted(err.Error())
					if expected != "" && expected != got {
						t.Errorf("Test #%d: Header mismatch:\n%s", 1+i, cmp.Diff(expected, got))
					}
				}
			} else {
				if err != nil {
					t.Errorf("Test #%d: Unexpected error for test case %s: %v", 1+i, tt.name, err)
				}
			}
		})
	}
}
