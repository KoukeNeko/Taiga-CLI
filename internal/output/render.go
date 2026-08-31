package output

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"
)

const ContractVersion = 1

type Renderer struct {
	Out    io.Writer
	Err    io.Writer
	JSON   bool
	Fields []string
	Quiet  bool
}

type Meta struct {
	Contract int `json:"contract"`
}

type DataEnvelope struct {
	Data any  `json:"data"`
	Meta Meta `json:"meta"`
}

type ListEnvelope struct {
	Items any  `json:"items"`
	Page  any  `json:"page,omitempty"`
	Meta  Meta `json:"meta"`
}

type ErrorBody struct {
	Code           string         `json:"code"`
	Message        string         `json:"message"`
	Retryable      bool           `json:"retryable"`
	Details        map[string]any `json:"details,omitempty"`
	UpstreamStatus int            `json:"upstream_status,omitempty"`
}

type ErrorEnvelope struct {
	Error ErrorBody `json:"error"`
	Meta  Meta      `json:"meta"`
}

type FieldError struct {
	Field     string
	Available []string
}

func (e *FieldError) Error() string {
	return fmt.Sprintf("unknown JSON field %q (available: %s)", e.Field, strings.Join(e.Available, ", "))
}

func (r Renderer) Data(value any) error {
	filtered, err := filterValue(value, r.Fields)
	if err != nil {
		return err
	}
	return writeJSON(r.Out, DataEnvelope{Data: filtered, Meta: Meta{Contract: ContractVersion}})
}

func (r Renderer) List(items, page any) error {
	filtered, err := filterList(items, r.Fields)
	if err != nil {
		return err
	}
	return writeJSON(r.Out, ListEnvelope{Items: filtered, Page: page, Meta: Meta{Contract: ContractVersion}})
}

func (r Renderer) Plan(plan any) error {
	return writeJSON(r.Out, map[string]any{"plan": plan, "meta": Meta{Contract: ContractVersion}})
}

func (r Renderer) Failure(body ErrorBody) error {
	if r.JSON {
		return writeJSON(r.Err, ErrorEnvelope{Error: body, Meta: Meta{Contract: ContractVersion}})
	}
	_, err := fmt.Fprintf(r.Err, "Error: %s\n", body.Message)
	return err
}

func writeJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(value)
}

func filterList(value any, fields []string) (any, error) {
	if len(fields) == 0 {
		return value, nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var items []map[string]any
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("filter list fields: %w", err)
	}
	available := reflectedJSONFields(value)
	if len(items) > 0 {
		available = mergeFields(available, sortedKeys(items[0]))
	}
	if err := validateFields(fields, available); err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return items, nil
	}
	filtered := make([]map[string]any, 0, len(items))
	for _, item := range items {
		filtered = append(filtered, selectFields(item, fields))
	}
	return filtered, nil
}

func filterValue(value any, fields []string) (any, error) {
	if len(fields) == 0 {
		return value, nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var item map[string]any
	if err := json.Unmarshal(data, &item); err != nil {
		return nil, fmt.Errorf("filter fields: %w", err)
	}
	available := mergeFields(reflectedJSONFields(value), sortedKeys(item))
	if err := validateFields(fields, available); err != nil {
		return nil, err
	}
	return selectFields(item, fields), nil
}

func validateFields(fields, available []string) error {
	set := map[string]struct{}{}
	for _, field := range available {
		set[field] = struct{}{}
	}
	for _, field := range normalizeFields(fields) {
		if _, ok := set[field]; !ok {
			return &FieldError{Field: field, Available: available}
		}
	}
	return nil
}

func selectFields(item map[string]any, fields []string) map[string]any {
	selected := map[string]any{}
	for _, field := range normalizeFields(fields) {
		selected[field] = item[field]
	}
	return selected
}

func normalizeFields(fields []string) []string {
	var result []string
	seen := map[string]struct{}{}
	for _, group := range fields {
		for _, field := range strings.Split(group, ",") {
			field = strings.TrimSpace(field)
			if field == "" {
				continue
			}
			if _, ok := seen[field]; ok {
				continue
			}
			seen[field] = struct{}{}
			result = append(result, field)
		}
	}
	return result
}

func sortedKeys(item map[string]any) []string {
	keys := make([]string, 0, len(item))
	for key := range item {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func reflectedJSONFields(value any) []string {
	typeOf := reflect.TypeOf(value)
	for typeOf != nil && (typeOf.Kind() == reflect.Pointer || typeOf.Kind() == reflect.Slice || typeOf.Kind() == reflect.Array) {
		typeOf = typeOf.Elem()
	}
	if typeOf == nil || typeOf.Kind() != reflect.Struct {
		return nil
	}
	fields := make([]string, 0, typeOf.NumField())
	for index := 0; index < typeOf.NumField(); index++ {
		field := typeOf.Field(index)
		if !field.IsExported() {
			continue
		}
		name := strings.Split(field.Tag.Get("json"), ",")[0]
		if name == "-" {
			continue
		}
		if name == "" {
			name = field.Name
		}
		fields = append(fields, name)
	}
	sort.Strings(fields)
	return fields
}

func mergeFields(groups ...[]string) []string {
	seen := map[string]struct{}{}
	for _, group := range groups {
		for _, field := range group {
			seen[field] = struct{}{}
		}
	}
	fields := make([]string, 0, len(seen))
	for field := range seen {
		fields = append(fields, field)
	}
	sort.Strings(fields)
	return fields
}
