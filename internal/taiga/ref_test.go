package taiga

import "testing"

func TestParseItemRef(t *testing.T) {
	tests := []struct {
		name, input, project string
		want                 ItemRef
		wantErr              bool
	}{
		{name: "bare", input: "42", project: "example-project", want: ItemRef{Project: "example-project", Ref: 42}},
		{name: "qualified", input: "demo#7", want: ItemRef{Project: "demo", Ref: 7}},
		{name: "url", input: "https://example.test/taiga/project/demo/issue/9", want: ItemRef{Project: "demo", Ref: 9}},
		{name: "missing project", input: "42", wantErr: true},
		{name: "wrong url", input: "https://example.test/projects", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ParseItemRef(test.input, test.project)
			if (err != nil) != test.wantErr {
				t.Fatalf("ParseItemRef() error = %v, wantErr %v", err, test.wantErr)
			}
			if got != test.want {
				t.Fatalf("ParseItemRef() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func FuzzParseItemRef(f *testing.F) {
	f.Add("demo#42", "fallback")
	f.Add("https://example.test/project/demo/issue/1", "")
	f.Fuzz(func(t *testing.T, value, project string) {
		_, _ = ParseItemRef(value, project)
	})
}

func TestParseStoryRefURL(t *testing.T) {
	got, err := ParseStoryRef("https://example.test/taiga/project/demo/us/13", "")
	if err != nil {
		t.Fatal(err)
	}
	if got != (ItemRef{Project: "demo", Ref: 13}) {
		t.Fatalf("ParseStoryRef() = %#v", got)
	}
}
