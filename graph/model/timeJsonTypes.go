package model

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// JSON implements a GraphQL scalar for arbitrary JSON data
type JSON struct {
	Data map[string]interface{}
}

// MarshalGQL implements the graphql.Marshaler interface
func (j JSON) MarshalGQL(w io.Writer) {
	bytes, err := json.Marshal(j.Data)
	if err != nil {
		io.WriteString(w, "{}")
		return
	}
	io.WriteString(w, string(bytes))
}

// UnmarshalGQL implements the graphql.Unmarshaler interface
func (j *JSON) UnmarshalGQL(v interface{}) error {
	switch v := v.(type) {
	case string:
		return json.Unmarshal([]byte(v), &j.Data)
	case map[string]interface{}:
		j.Data = v
		return nil
	default:
		return fmt.Errorf("invalid type for JSON: %T", v)
	}
}

// Add this below the JSON implementation in the same file

// Time wraps time.Time for GraphQL scalar
type Time struct {
	time.Time
}

// MarshalGQL implements the graphql.Marshaler interface
func (t Time) MarshalGQL(w io.Writer) {
	_, err := io.WriteString(w, t.Time.Format(time.RFC3339))
	if err != nil {
		io.WriteString(w, `""`)
	}
}

// UnmarshalGQL implements the graphql.Unmarshaler interface
func (t *Time) UnmarshalGQL(v interface{}) error {
	switch v := v.(type) {
	case string:
		parsed, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return err
		}
		t.Time = parsed
		return nil
	default:
		return fmt.Errorf("invalid type for Time: %T", v)
	}
}
