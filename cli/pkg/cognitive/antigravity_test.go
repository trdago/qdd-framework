package cognitive

import (
	"os"
	"path/filepath"
	"testing"
)

func TestModelContentFromLine(t *testing.T) {
	cases := []struct {
		name string
		line string
		want string
		ok   bool
	}{
		{"model turn with content", `{"source":"MODEL","content":"hola"}`, "hola", true},
		{"non-model turn ignored", `{"source":"USER","content":"hola"}`, "", false},
		{"model turn with empty content", `{"source":"MODEL","content":""}`, "", false},
		{"invalid json", `not json`, "", false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := modelContentFromLine(c.line)
			if ok != c.ok {
				t.Fatalf("ok = %v, want %v", ok, c.ok)
			}
			if got != c.want {
				t.Errorf("got = %q, want %q", got, c.want)
			}
		})
	}
}

func TestReadModelResponse_FindsModelLineAmongOthers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "transcript.jsonl")
	content := "{\"source\":\"USER\",\"content\":\"pregunta\"}\n{\"source\":\"MODEL\",\"content\":\"[VERDICT_START]{}[VERDICT_END]\"}\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write transcript: %v", err)
	}

	got, ok := readModelResponse(path)
	if !ok {
		t.Fatal("expected to find a MODEL response line")
	}
	if got != "[VERDICT_START]{}[VERDICT_END]" {
		t.Errorf("got = %q, want the MODEL line's content", got)
	}
}

func TestReadModelResponse_MissingFile(t *testing.T) {
	_, ok := readModelResponse(filepath.Join(t.TempDir(), "does-not-exist.jsonl"))
	if ok {
		t.Error("expected ok=false for a transcript file that doesn't exist yet")
	}
}
