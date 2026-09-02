package cli

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// parseBatchOrders reads what a person typed on the command line and decides
// which records get reordered, so every way of getting it wrong is worth
// stating.
func TestParseBatchOrders(t *testing.T) {
	accepted := map[string]map[int64]int{
		"1=0":     {1: 0},
		"12=3":    {12: 3},
		" 7 = 2 ": {7: 2},
	}
	for input, want := range accepted {
		got, err := parseBatchOrders([]string{input})
		if err != nil {
			t.Errorf("%q: unexpected error %v", input, err)
			continue
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%q = %v, want %v", input, got, want)
		}
	}

	got, err := parseBatchOrders([]string{"4=1", "5=2"})
	if err != nil || !reflect.DeepEqual(got, map[int64]int{4: 1, 5: 2}) {
		t.Errorf("two entries = %v, %v", got, err)
	}

	rejected := map[string][]string{
		"no values":             {},
		"no separator":          {"41"},
		"missing order":         {"4="},
		"missing id":            {"=2"},
		"id is not a number":    {"four=1"},
		"order is not a number": {"4=one"},
		"zero id":               {"0=1"},
		"negative id":           {"-4=1"},
		"negative order":        {"4=-1"},
		"repeated id":           {"4=1", "4=2"},
	}
	for name, input := range rejected {
		if _, err := parseBatchOrders(input); err == nil {
			t.Errorf("%s: %v was accepted", name, input)
		}
	}

	// The cap exists so one command cannot reorder a whole project by accident.
	tooMany := make([]string, 0, maxBatchItems+1)
	for index := range maxBatchItems + 1 {
		tooMany = append(tooMany, strconv.Itoa(index+1)+"=0")
	}
	if _, err := parseBatchOrders(tooMany); err == nil {
		t.Errorf("%d entries was accepted, over the %d cap", len(tooMany), maxBatchItems)
	}
}

// An importer's response carries the credentials it was handed. None of them
// may reach output, including a dry run.
func TestRedactImporterResult(t *testing.T) {
	secret := "must-never-appear"
	fields := map[string]any{
		"token":         secret,
		"auth_token":    secret,
		"AUTH_CODE":     secret,
		"client_secret": secret,
		"Password":      secret,
		"code_verifier": secret,
		"project":       "visible",
		"count":         3,
	}
	redacted, ok := redactImporterResult(fields).(map[string]any)
	if !ok {
		t.Fatal("a map must come back as a map")
	}
	for key, value := range redacted {
		if value == secret {
			t.Errorf("%q still carries the credential", key)
		}
	}
	if redacted["project"] != "visible" || redacted["count"] != 3 {
		t.Errorf("non-secret fields were altered: %v", redacted)
	}
	if len(redacted) != len(fields) {
		t.Errorf("redaction dropped fields: %d of %d", len(redacted), len(fields))
	}
	// Anything that is not an object is passed through, since there are no
	// field names to judge.
	if got := redactImporterResult("plain"); got != "plain" {
		t.Errorf("non-object became %v", got)
	}
}

func TestPositiveID(t *testing.T) {
	for input, want := range map[string]int64{"1": 1, " 42 ": 42, "9007199254740993": 9007199254740993} {
		got, err := positiveID(input, "Project")
		if err != nil || got != want {
			t.Errorf("positiveID(%q) = %d, %v; want %d", input, got, err, want)
		}
	}
	for _, input := range []string{"", "0", "-1", "1.5", "one", "1e3", " "} {
		if _, err := positiveID(input, "Project"); err == nil {
			t.Errorf("positiveID(%q) was accepted", input)
		} else if !strings.Contains(err.Error(), "Project") {
			t.Errorf("positiveID(%q) error omits the name: %v", input, err)
		}
	}
}

// The aliases exist so that a name a person is likely to type reaches the same
// resource as the name the API uses.
func TestNormaliseAliases(t *testing.T) {
	for input, want := range map[string]string{
		"userstory": "story", "user-story": "story", "us": "story",
		"wikipage": "wiki", "wiki-page": "wiki",
		"  ISSUE  ": "issue", "epic": "epic", "unknown": "unknown",
	} {
		if got := normalizeSearchKind(input); got != want {
			t.Errorf("normalizeSearchKind(%q) = %q, want %q", input, got, want)
		}
	}
	for input, want := range map[string]string{
		"current-profile": "profile", "current_profile": "profile",
		"api_url": "api-url", "apiurl": "api-url", "project": "project",
	} {
		if got := normalizeConfigKey(input); got != want {
			t.Errorf("normalizeConfigKey(%q) = %q, want %q", input, got, want)
		}
	}
	for input, want := range map[string]string{
		"story": "story", "userstory": "story", "user-story": "story",
		" TASK ": "task", "issue": "issue",
	} {
		got, err := normalizeDueDateResource(input)
		if err != nil || got != want {
			t.Errorf("normalizeDueDateResource(%q) = %q, %v; want %q", input, got, err, want)
		}
	}
	for _, input := range []string{"epic", "wiki", "", "sprint"} {
		if _, err := normalizeDueDateResource(input); err == nil {
			t.Errorf("normalizeDueDateResource(%q) was accepted", input)
		}
	}
}

func TestValidKinds(t *testing.T) {
	for _, kind := range []string{"all", "epic", "story", "task", "issue", "wiki"} {
		if !validSearchKind(kind) {
			t.Errorf("validSearchKind(%q) = false", kind)
		}
	}
	// validSearchKind runs on an already-normalised value, so an alias is not
	// one of its inputs and must not be accepted here.
	for _, kind := range []string{"userstory", "Story", "", "sprint"} {
		if validSearchKind(kind) {
			t.Errorf("validSearchKind(%q) = true", kind)
		}
	}
	for _, kind := range []string{"text", "multiline", "richtext", "date", "url", "dropdown", "checkbox", "number", " TEXT "} {
		if !validCustomFieldType(kind) {
			t.Errorf("validCustomFieldType(%q) = false", kind)
		}
	}
	for _, kind := range []string{"", "string", "boolean", "integer"} {
		if validCustomFieldType(kind) {
			t.Errorf("validCustomFieldType(%q) = true", kind)
		}
	}
}

// A custom field value is whatever the person typed. Anything that parses as
// JSON keeps its type so a number stays a number, and everything else is the
// string itself rather than an error.
func TestParseCustomFieldValue(t *testing.T) {
	cases := []struct {
		raw  string
		want any
	}{
		{"42", float64(42)},
		{"true", true},
		{"null", nil},
		{`"quoted"`, "quoted"},
		{"not json", "not json"},
		{"", ""},
		{"2026-09-03", "2026-09-03"},
	}
	for _, testCase := range cases {
		if got := parseCustomFieldValue(testCase.raw); !reflect.DeepEqual(got, testCase.want) {
			t.Errorf("parseCustomFieldValue(%q) = %#v, want %#v", testCase.raw, got, testCase.want)
		}
	}
	if got := parseCustomFieldValue(`{"a":1}`); !reflect.DeepEqual(got, map[string]any{"a": float64(1)}) {
		t.Errorf("an object was not preserved: %#v", got)
	}
}

func TestAuthRequiredIsAnAuthError(t *testing.T) {
	err := authRequired("run `aihki auth login` first")
	var known *contractError
	if !errors.As(err, &known) {
		t.Fatalf("authRequired returned %T, want a contract error", err)
	}
	if known.ExitCode != ExitAuth {
		t.Errorf("exit code = %d, want %d", known.ExitCode, ExitAuth)
	}
	if known.Code != "authentication_required" {
		t.Errorf("code = %q, want authentication_required", known.Code)
	}
	if !strings.Contains(known.Error(), "auth login") {
		t.Errorf("message = %q, want it to name the command to run", known.Error())
	}
}
