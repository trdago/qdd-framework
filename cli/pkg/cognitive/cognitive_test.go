package cognitive

import (
	"context"
	"errors"
	"testing"
)

// fakeBackend is a hand-written test double (not a mocking-framework mock —
// production code under this repo's Zero-Mocks policy never imports one; this
// is a plain struct implementing the real Backend interface, confined to
// _test.go).
type fakeBackend struct {
	name      string
	available bool
	response  string
	err       error
}

func (f *fakeBackend) Name() string           { return f.name }
func (f *fakeBackend) Available() bool        { return f.available }
func (f *fakeBackend) Ask(context.Context, string) (string, error) {
	return f.response, f.err
}

func TestAskUsing_ReturnsFirstAvailableBackendsAnswer(t *testing.T) {
	backends := []Backend{
		&fakeBackend{name: "unavailable", available: false},
		&fakeBackend{name: "primary", available: true, response: "hola"},
		&fakeBackend{name: "secondary", available: true, response: "nunca llega aquí"},
	}

	resp, name, err := askUsing(context.Background(), "ping", backends)
	if err != nil {
		t.Fatalf("askUsing returned unexpected error: %v", err)
	}
	if name != "primary" {
		t.Errorf("backend used = %q, want %q", name, "primary")
	}
	if resp != "hola" {
		t.Errorf("response = %q, want %q", resp, "hola")
	}
}

func TestAskUsing_FallsBackWhenFirstAvailableBackendFails(t *testing.T) {
	backends := []Backend{
		&fakeBackend{name: "flaky", available: true, err: errors.New("boom")},
		&fakeBackend{name: "reliable", available: true, response: "ok"},
	}

	resp, name, err := askUsing(context.Background(), "ping", backends)
	if err != nil {
		t.Fatalf("askUsing returned unexpected error: %v", err)
	}
	if name != "reliable" {
		t.Errorf("backend used = %q, want %q", name, "reliable")
	}
	if resp != "ok" {
		t.Errorf("response = %q, want %q", resp, "ok")
	}
}

func TestAskUsing_ErrorsWhenNoBackendIsAvailable(t *testing.T) {
	backends := []Backend{
		&fakeBackend{name: "a", available: false},
		&fakeBackend{name: "b", available: false},
	}

	_, _, err := askUsing(context.Background(), "ping", backends)
	if err == nil {
		t.Error("expected an error when no backend is available")
	}
}

func TestExtractTagged(t *testing.T) {
	cases := []struct {
		name string
		text string
		want string
		ok   bool
	}{
		{"present", "prefix [VERDICT_START]{\"a\":1}[VERDICT_END] suffix", `{"a":1}`, true},
		{"missing start", "no tags here", "", false},
		{"missing end", "[VERDICT_START]{\"a\":1}", "", false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := ExtractTagged(c.text, "[VERDICT_START]", "[VERDICT_END]")
			if ok != c.ok {
				t.Fatalf("ok = %v, want %v", ok, c.ok)
			}
			if got != c.want {
				t.Errorf("got = %q, want %q", got, c.want)
			}
		})
	}
}
