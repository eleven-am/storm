package orm

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type JSONData struct {
	Data  *any
	Valid bool
}

func NewJSONData(data any) JSONData {
	return JSONData{
		Data:  &data,
		Valid: true,
	}
}

func NewNullJSONData() JSONData {
	return JSONData{
		Data:  nil,
		Valid: false,
	}
}

func (j JSONData) Value() (driver.Value, error) {
	if !j.Valid || j.Data == nil {
		return nil, nil
	}

	return json.Marshal(j.Data)
}

func (j *JSONData) Scan(value interface{}) error {
	if value == nil {
		j.Data = nil
		j.Valid = false
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into JSONData", value)
	}

	if len(bytes) == 0 {
		j.Data = nil
		j.Valid = false
		return nil
	}

	var data any
	if err := json.Unmarshal(bytes, &data); err != nil {
		return fmt.Errorf("failed to unmarshal JSONData: %w", err)
	}

	j.Data = &data
	j.Valid = true
	return nil
}

func (j *JSONData) Get(v interface{}) error {
	if !j.Valid || j.Data == nil {
		return fmt.Errorf("JSONData is null or invalid")
	}

	jsonBytes, err := json.Marshal(j.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal JSONData: %w", err)
	}

	if err := json.Unmarshal(jsonBytes, v); err != nil {
		return fmt.Errorf("failed to unmarshal JSONData into target type: %w", err)
	}

	return nil
}

func (j *JSONData) MustGet(v interface{}) error {
	if !j.Valid || j.Data == nil {
		panic("JSONData: MustGet called on null or invalid field")
	}

	if err := j.Get(v); err != nil {
		return err
	}

	return nil
}

func (j *JSONData) Set(data interface{}) {
	if data == nil {
		j.Data = nil
		j.Valid = false
	} else {
		j.Data = &data
		j.Valid = true
	}
}

func (j *JSONData) IsNull() bool {
	return !j.Valid || j.Data == nil
}

func (j JSONData) String() string {
	if !j.Valid || j.Data == nil {
		return "NULL"
	}
	bytes, _ := json.Marshal(j.Data)
	return string(bytes)
}
