package orm

import (
	"database/sql/driver"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStringArray_Scan(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected StringArray
		wantErr  bool
	}{
		{
			name:     "nil value",
			input:    nil,
			expected: nil,
			wantErr:  false,
		},
		{
			name:     "empty array string",
			input:    "{}",
			expected: StringArray{},
			wantErr:  false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: StringArray{},
			wantErr:  false,
		},
		{
			name:     "single element",
			input:    `{"hello"}`,
			expected: StringArray{"hello"},
			wantErr:  false,
		},
		{
			name:     "multiple elements",
			input:    `{"hello","world","test"}`,
			expected: StringArray{"hello", "world", "test"},
			wantErr:  false,
		},
		{
			name:     "elements with quotes",
			input:    `{"hello ""world""","test"}`,
			expected: StringArray{`hello "world"`, "test"},
			wantErr:  false,
		},
		{
			name:     "elements with commas",
			input:    `{"hello,world","test"}`,
			expected: StringArray{"hello,world", "test"},
			wantErr:  false,
		},
		{
			name:     "byte slice input",
			input:    []byte(`{"hello","world"}`),
			expected: StringArray{"hello", "world"},
			wantErr:  false,
		},
		{
			name:    "invalid type",
			input:   123,
			wantErr: true,
		},
		{
			name:    "invalid format",
			input:   "hello,world",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sa StringArray
			err := sa.Scan(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, sa)
			}
		})
	}
}

func TestStringArray_Value(t *testing.T) {
	tests := []struct {
		name     string
		input    StringArray
		expected driver.Value
		wantErr  bool
	}{
		{
			name:     "nil array",
			input:    nil,
			expected: nil,
			wantErr:  false,
		},
		{
			name:     "empty array",
			input:    StringArray{},
			expected: "{}",
			wantErr:  false,
		},
		{
			name:     "single element",
			input:    StringArray{"hello"},
			expected: `{"hello"}`,
			wantErr:  false,
		},
		{
			name:     "multiple elements",
			input:    StringArray{"hello", "world", "test"},
			expected: `{"hello","world","test"}`,
			wantErr:  false,
		},
		{
			name:     "elements with quotes",
			input:    StringArray{`hello "world"`, "test"},
			expected: `{"hello ""world""","test"}`,
			wantErr:  false,
		},
		{
			name:     "elements with commas",
			input:    StringArray{"hello,world", "test"},
			expected: `{"hello,world","test"}`,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := tt.input.Value()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, value)
			}
		})
	}
}

func TestStringArray_RoundTrip(t *testing.T) {
	tests := []StringArray{
		nil,
		{},
		{"hello"},
		{"hello", "world"},
		{`hello "world"`, "test"},
		{"hello,world", "test"},
		{"", "empty", ""},
	}

	for _, original := range tests {
		t.Run("", func(t *testing.T) {

			value, err := original.Value()
			require.NoError(t, err)

			// Scan back
			var result StringArray
			err = result.Scan(value)
			require.NoError(t, err)

			assert.Equal(t, original, result)
		})
	}
}
