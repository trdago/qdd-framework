package mejoras

import "testing"

func TestRepoDeRemote_SSHYHTTPS(t *testing.T) {
	casos := map[string]string{
		"git@github.com:trdago/qdd-framework.git":     "trdago/qdd-framework",
		"https://github.com/trdago/qdd-framework.git": "trdago/qdd-framework",
		"https://github.com/trdago/qdd-framework":     "trdago/qdd-framework",
		"ssh://git@github.com/trdago/qdd.blueprints/": "trdago/qdd.blueprints",
	}
	for remote, quiere := range casos {
		if got, ok := RepoDeRemote(remote); !ok || got != quiere {
			t.Errorf("%s → %q %v, quiere %q", remote, got, ok, quiere)
		}
	}
	if _, ok := RepoDeRemote("git@gitlab.com:x/y.git"); ok {
		t.Error("un remote que no es de GitHub no aplica")
	}
}

func TestIssuesDeAPI_SinPullRequestsYConQuienLoEscribio(t *testing.T) {
	api := `[
	 {"number":5,"title":"Mejora","body":"detalle","html_url":"https://github.com/o/r/issues/5","created_at":"2026-11-03T10:00:00Z",
	  "author_association":"OWNER","user":{"login":"trdago"},"labels":[{"name":"bug"}]},
	 {"number":6,"title":"Un PR","pull_request":{"url":"x"},"user":{"login":"a"},"created_at":"2026-11-03T10:00:00Z"}]`
	issues, err := IssuesDeAPI([]byte(api))
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 1 {
		t.Fatalf("los PR no son issues: %+v", issues)
	}
	is := issues[0]
	if is.Numero != 5 || is.Autor != "trdago" || is.Asociacion != "OWNER" || len(is.Etiquetas) != 1 || is.Etiquetas[0] != "bug" || is.Creado.IsZero() {
		t.Fatalf("%+v", is)
	}
	if _, err := IssuesDeAPI([]byte("no es json")); err == nil {
		t.Fatal("una respuesta ilegible es un error que el ciclo avisa")
	}
}
