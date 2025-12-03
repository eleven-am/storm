package orm

import (
	"database/sql/driver"
	"testing"
)

func TestJSONData_Set(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		wantErr bool
	}{
		{
			name: "set struct",
			input: struct {
				Name string `json:"name"`
				Age  int    `json:"age"`
			}{Name: "John", Age: 30},
			wantErr: false,
		},
		{
			name:    "set nil",
			input:   nil,
			wantErr: false,
		},
		{
			name:    "set map",
			input:   map[string]interface{}{"key": "value"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := &JSONData{}
			j.Set(tt.input)

			if tt.input == nil {
				if j.Valid || j.Data != nil {
					t.Errorf("Set(nil) should set Valid=false and Data=nil")
				}
			} else {
				if !j.Valid || j.Data == nil {
					t.Errorf("Set(%v) should set Valid=true and Data!=nil", tt.input)
				}
			}
		})
	}
}

func TestJSONData_Get(t *testing.T) {
	type testStruct struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	j := &JSONData{}
	testData := testStruct{Name: "John", Age: 30}
	j.Set(testData)

	var result testStruct
	err := j.Get(&result)
	if err != nil {
		t.Errorf("Get() should not return error: %v", err)
	}
	if result.Name != "John" || result.Age != 30 {
		t.Errorf("Get() should return correct data: got %+v", result)
	}

	nullJ := NewNullJSONData()
	var nullResult testStruct
	err = nullJ.Get(&nullResult)
	if err == nil {
		t.Errorf("Get() on null JSONData should return error")
	}
}

func TestJSONData_MustGet(t *testing.T) {
	type testStruct struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	j := &JSONData{}
	testData := testStruct{Name: "John", Age: 30}
	j.Set(testData)

	var result testStruct
	err := j.MustGet(&result)
	if err != nil {
		t.Errorf("MustGet() should not return error: %v", err)
	}
	if result.Name != "John" || result.Age != 30 {
		t.Errorf("MustGet() should return correct data: got %+v", result)
	}

	nullJ := NewNullJSONData()
	var nullResult testStruct

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("MustGet() on null JSONData should panic")
		}
	}()

	nullJ.MustGet(&nullResult)
}

func TestJSONData_Scan(t *testing.T) {
	tests := []struct {
		name    string
		value   interface{}
		wantErr bool
	}{
		{
			name:    "scan bytes",
			value:   []byte(`{"key":"value"}`),
			wantErr: false,
		},
		{
			name:    "scan string",
			value:   `{"key":"value"}`,
			wantErr: true,
		},
		{
			name:    "scan nil",
			value:   nil,
			wantErr: false,
		},
		{
			name:    "scan unsupported type",
			value:   123,
			wantErr: true,
		},
		{
			name:    "scan empty bytes",
			value:   []byte{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := &JSONData{}
			err := j.Scan(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("Scan() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				if tt.value == nil || (tt.value != nil && len(tt.value.([]byte)) == 0) {

					if j.Valid || j.Data != nil {
						t.Errorf("Scan(nil/empty) should set Valid=false and Data=nil")
					}
				} else {

					if !j.Valid || j.Data == nil {
						t.Errorf("Scan(%v) should set Valid=true and Data!=nil", tt.value)
					}
				}
			}
		})
	}
}

func TestJSONData_Value(t *testing.T) {
	tests := []struct {
		name    string
		data    JSONData
		want    driver.Value
		wantErr bool
	}{
		{
			name:    "value with data",
			data:    NewJSONData(map[string]interface{}{"key": "value"}),
			want:    []byte(`{"key":"value"}`),
			wantErr: false,
		},
		{
			name:    "value with null data",
			data:    NewNullJSONData(),
			want:    nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.data.Value()
			if (err != nil) != tt.wantErr {
				t.Errorf("Value() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got == nil && tt.want == nil {
				return
			}
			if got != nil && tt.want != nil {
				gotBytes := got.([]byte)
				wantBytes := tt.want.([]byte)
				if string(gotBytes) != string(wantBytes) {
					t.Errorf("Value() = %v, want %v", got, tt.want)
				}
			} else if got != tt.want {
				t.Errorf("Value() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJSONData_IsNull(t *testing.T) {
	tests := []struct {
		name string
		data JSONData
		want bool
	}{
		{
			name: "null data",
			data: NewNullJSONData(),
			want: true,
		},
		{
			name: "valid data",
			data: NewJSONData(map[string]interface{}{"key": "value"}),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.data.IsNull(); got != tt.want {
				t.Errorf("IsNull() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJSONData_String(t *testing.T) {
	tests := []struct {
		name string
		data JSONData
		want string
	}{
		{
			name: "null data",
			data: NewNullJSONData(),
			want: "NULL",
		},
		{
			name: "valid data",
			data: NewJSONData(map[string]interface{}{"key": "value"}),
			want: `{"key":"value"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.data.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewJSONData(t *testing.T) {
	testData := map[string]interface{}{"key": "value"}
	j := NewJSONData(testData)

	if !j.Valid {
		t.Errorf("NewJSONData() should set Valid=true")
	}
	if j.Data == nil {
		t.Errorf("NewJSONData() should set Data!=nil")
	}
	if j.IsNull() {
		t.Errorf("NewJSONData() should not be null")
	}
}

func TestNewNullJSONData(t *testing.T) {
	j := NewNullJSONData()

	if j.Valid {
		t.Errorf("NewNullJSONData() should set Valid=false")
	}
	if j.Data != nil {
		t.Errorf("NewNullJSONData() should set Data=nil")
	}
	if !j.IsNull() {
		t.Errorf("NewNullJSONData() should be null")
	}
}
