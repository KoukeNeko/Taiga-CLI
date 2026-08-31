package cli

import "sort"

const jsonSchemaDraft = "https://json-schema.org/draft/2020-12/schema"

type Safety string
type Idempotency string

const (
	SafetyRead    Safety      = "read"
	SafetyWrite   Safety      = "write"
	SafetyLocal   Safety      = "local"
	Idempotent    Idempotency = "idempotent"
	NonIdempotent Idempotency = "non_idempotent"
)

type Descriptor struct {
	Command     string         `json:"command"`
	Description string         `json:"description"`
	Safety      Safety         `json:"safety"`
	Idempotency Idempotency    `json:"idempotency"`
	Input       map[string]any `json:"input_schema"`
	Output      map[string]any `json:"output_schema"`
}

func objectSchema(properties map[string]any, required ...string) map[string]any {
	schema := map[string]any{"$schema": jsonSchemaDraft, "type": "object", "properties": properties, "additionalProperties": false}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func descriptors() map[string]Descriptor {
	stringProperty := map[string]any{"type": "string"}
	integerProperty := map[string]any{"type": "integer"}
	booleanProperty := map[string]any{"type": "boolean"}
	commonOutput := objectSchema(map[string]any{"data": map[string]any{"type": "object"}, "meta": map[string]any{"type": "object"}}, "data", "meta")
	listOutput := objectSchema(map[string]any{"items": map[string]any{"type": "array"}, "page": map[string]any{"type": "object"}, "meta": map[string]any{"type": "object"}}, "items", "meta")
	return map[string]Descriptor{
		"doctor":        {Command: "doctor", Description: "Diagnose a Taiga endpoint and current context", Safety: SafetyRead, Idempotency: Idempotent, Input: objectSchema(map[string]any{"host": stringProperty, "api_url": stringProperty}), Output: commonOutput},
		"auth login":    {Command: "auth login", Description: "Authenticate and save credentials", Safety: SafetyLocal, Idempotency: NonIdempotent, Input: objectSchema(map[string]any{"host": stringProperty, "api_url": stringProperty, "profile": stringProperty, "with_token": booleanProperty}), Output: commonOutput},
		"auth logout":   {Command: "auth logout", Description: "Remove saved credentials", Safety: SafetyLocal, Idempotency: Idempotent, Input: objectSchema(map[string]any{"profile": stringProperty}), Output: commonOutput},
		"auth status":   {Command: "auth status", Description: "Show current authentication status", Safety: SafetyRead, Idempotency: Idempotent, Input: objectSchema(map[string]any{}), Output: commonOutput},
		"config get":    {Command: "config get", Description: "Read a configuration value", Safety: SafetyRead, Idempotency: Idempotent, Input: objectSchema(map[string]any{"key": stringProperty, "local": booleanProperty}, "key"), Output: commonOutput},
		"config set":    {Command: "config set", Description: "Write a configuration value", Safety: SafetyLocal, Idempotency: Idempotent, Input: objectSchema(map[string]any{"key": stringProperty, "value": stringProperty, "local": booleanProperty}, "key", "value"), Output: commonOutput},
		"config list":   {Command: "config list", Description: "List non-secret configuration", Safety: SafetyRead, Idempotency: Idempotent, Input: objectSchema(map[string]any{"local": booleanProperty}), Output: commonOutput},
		"project list":  {Command: "project list", Description: "List projects", Safety: SafetyRead, Idempotency: Idempotent, Input: objectSchema(map[string]any{"page": integerProperty, "limit": integerProperty}), Output: listOutput},
		"project view":  {Command: "project view", Description: "View a project", Safety: SafetyRead, Idempotency: Idempotent, Input: objectSchema(map[string]any{"slug": stringProperty}, "slug"), Output: commonOutput},
		"project use":   {Command: "project use", Description: "Select a default project", Safety: SafetyLocal, Idempotency: Idempotent, Input: objectSchema(map[string]any{"slug": stringProperty, "local": booleanProperty}, "slug"), Output: commonOutput},
	}
}

func descriptorNames(values map[string]Descriptor) []string {
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
