package cli

import (
	"context"
	"encoding/json"
	"testing"
)

func TestVersionCommandJSON(t *testing.T) {
	app, out, stderr, _ := testApp(t, nil)
	if code := app.Execute(context.Background(), []string{"--json", "version", "--fields", "version,commit,build_date"}); code != ExitSuccess {
		t.Fatalf("exit = %d, stderr = %s", code, stderr.String())
	}
	var result struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Data["version"] == nil || result.Data["commit"] == nil || result.Data["build_date"] == nil {
		t.Fatalf("result = %#v", result)
	}
}
