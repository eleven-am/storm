package orm_generator

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetadataTemplate(t *testing.T) {

	assert.NotEmpty(t, metadataTemplate)
	assert.Contains(t, metadataTemplate, "{{ .Model.Name }}")
	assert.Contains(t, metadataTemplate, "{{ .Package }}")
}

func TestColumnTemplate(t *testing.T) {

	assert.NotEmpty(t, columnTemplate)
	assert.Contains(t, columnTemplate, "Column")
}

func TestRepositoryTemplate(t *testing.T) {

	assert.NotEmpty(t, repositoryTemplate)
	assert.Contains(t, repositoryTemplate, "Repository")
}

func TestTemplateHelperFunctions(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
		function func(string) string
	}{
		{
			name:     "toSnakeCase",
			input:    "TestUserName",
			expected: "test_user_name",
			function: toSnakeCase,
		},
		{
			name:     "toCamelCase",
			input:    "test_user_name",
			expected: "testUserName",
			function: toCamelCase,
		},
		{
			name:     "toPascalCase",
			input:    "test_user_name",
			expected: "TestUserName",
			function: toPascalCase,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.function(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestTemplateValidation(t *testing.T) {

	templates := []string{metadataTemplate, columnTemplate, repositoryTemplate}

	for _, template := range templates {
		assert.NotEmpty(t, template)

		assert.Contains(t, template, "{{")
		assert.Contains(t, template, "}}")
	}
}

// Test helper functions (don't redefine existing functions)
func testToTitle(s string) string {
	return strings.Title(s)
}
