package blueprints

import (
	"fmt"
	"strings"
)

// Un repositorio de blueprints público lo usa cualquiera: por HTTPS no hace
// falta llave SSH. QDD lo sugiere cuando el acceso por SSH falla y, al
// publicar, reintenta por HTTPS si SSH no tiene permiso de escritura.

// httpsEquivalente traduce una URL SSH de git (usuario@host:ruta o
// ssh://usuario@host[:puerto]/ruta) a la HTTPS del mismo repositorio. Es una
// variable para que las pruebas apunten la «URL HTTPS» a un repositorio local.
var httpsEquivalente = func(repo string) (string, bool) {
	host, ruta, ok := partesSSH(repo)
	if !ok {
		return "", false
	}
	return "https://" + host + "/" + ruta, true
}

func partesSSH(repo string) (string, string, bool) {
	if resto, ok := strings.CutPrefix(repo, "ssh://"); ok {
		hostPuerto, ruta, _ := strings.Cut(resto, "/")
		host, _, _ := strings.Cut(hostPuerto[strings.LastIndex(hostPuerto, "@")+1:], ":")
		return host, ruta, host != "" && ruta != ""
	}
	if strings.Contains(repo, "://") {
		return "", "", false
	}
	// usuario@host:ruta (sintaxis scp); exige el usuario para no confundirse
	// con una ruta de Windows como C:/repos.
	usuarioHost, ruta, ok := strings.Cut(repo, ":")
	if !ok || !strings.Contains(usuarioHost, "@") || strings.Contains(usuarioHost, "/") {
		return "", "", false
	}
	host := usuarioHost[strings.LastIndex(usuarioHost, "@")+1:]
	ruta = strings.TrimPrefix(ruta, "/")
	return host, ruta, host != "" && ruta != ""
}

// sugerenciaHTTPS es lo que se agrega a un error de acceso a un repositorio
// SSH: si es público, la URL HTTPS funciona sin llave (FND-068).
func sugerenciaHTTPS(repo string) string {
	url, ok := httpsEquivalente(repo)
	if !ok {
		return ""
	}
	return fmt.Sprintf(" Si el repositorio es público no necesita llave SSH: usa %s", url)
}

// sinAcceso es el error de no poder clonar o actualizar un repositorio.
func sinAcceso(accion, repo string, err error) error {
	return fmt.Errorf("no se pudo %s '%s' (¿tienes acceso al repositorio?): %v%s", accion, repo, err, sugerenciaHTTPS(repo))
}

// empujar publica ref en el origin del clon. Si el origin es SSH y el push
// falla (p. ej. una deploy key de solo lectura), reintenta por la URL HTTPS del
// mismo repositorio, que usa las credenciales de git (como las de gh); el aviso
// lo dice (FND-063).
func empujar(espacio, repo, ref string, flags ...string) (string, error) {
	args := append(append([]string{"push", "--quiet"}, flags...), "origin", ref)
	_, errSSH := git(espacio, args...)
	if errSSH == nil {
		return "", nil
	}
	url, ok := httpsEquivalente(repo)
	if !ok {
		return "", errSSH
	}
	if _, err := git(espacio, "push", "--quiet", url, ref); err != nil {
		return "", fmt.Errorf("%v (por HTTPS, %s, tampoco: %v)", errSSH, url, err)
	}
	return fmt.Sprintf("se publicó por HTTPS (%s) porque por SSH no hubo permiso: %s", url, primeraLinea(errSSH.Error())), nil
}

func primeraLinea(s string) string {
	linea, _, _ := strings.Cut(strings.TrimSpace(s), "\n")
	return linea
}
