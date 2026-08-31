package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
)

func TestDataEnvelopeAndFields(t *testing.T) {
	var out bytes.Buffer
	renderer := Renderer{Out: &out, Fields: []string{"id,name"}}
	if err := renderer.Data(struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
		Slug string `json:"slug"`
	}{ID: 1, Name: "Demo", Slug: "demo"}); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	data := result["data"].(map[string]any)
	if len(data) != 2 || data["slug"] != nil {
		t.Fatalf("data = %#v", data)
	}
	if result["meta"].(map[string]any)["contract"] != float64(1) {
		t.Fatalf("result = %#v", result)
	}
}

func TestUnknownField(t *testing.T) {
	var out bytes.Buffer
	err := (Renderer{Out: &out, Fields: []string{"missing"}}).Data(map[string]any{"id": 1})
	var fieldErr *FieldError
	if !errors.As(err, &fieldErr) {
		t.Fatalf("error = %#v", err)
	}
}

func TestUnknownFieldOnEmptyTypedList(t *testing.T) {
	var out bytes.Buffer
	type item struct {
		ID int `json:"id"`
	}
	err := (Renderer{Out: &out, Fields: []string{"missing"}}).List([]item{}, nil)
	var fieldErr *FieldError
	if !errors.As(err, &fieldErr) {
		t.Fatalf("error = %#v", err)
	}
}

func TestJSONFailureWritesOnlyStderr(t *testing.T) {
	var out, stderr bytes.Buffer
	renderer := Renderer{Out: &out, Err: &stderr, JSON: true}
	if err := renderer.Failure(ErrorBody{Code: "not_found", Message: "missing"}); err != nil {
		t.Fatal(err)
	}
	if out.Len() != 0 {
		t.Fatalf("stdout = %q", out.String())
	}
	if !json.Valid(stderr.Bytes()) {
		t.Fatalf("stderr is not JSON: %q", stderr.String())
	}
}
