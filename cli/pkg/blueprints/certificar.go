package blueprints

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Rule: Prohibido usar 'else'
// Certificación ciega (pedido del usuario, 2026-10-31): antes de publicar una receta, alguien que no ve el original
// la construye desde cero, solo con la receta, en dos variantes: con los valores sugeridos y con otros valores de
// negocio. QDD corre sus pruebas y exige una prueba que pase por cada escenario de cada variante: así lo técnico se
// replica al 100 % y lo de negocio se adapta bien (los escenarios de la variante ya traen sus valores). QDD prepara
// las carpetas y verifica; construir es de un agente nuevo, sin la conversación ni el código original.

const (
	RecetaFile        = "RECETA.md"
	ConstruccionFile  = "construccion.yaml"
	CertificacionFile = "certificacion.yaml"
	VarianteSugerida  = "sugerida"
	VarianteAdaptada  = "adaptada"
	maxPruebaCiega    = 10 * time.Minute
)

// Kit son las carpetas de una certificación, una por variante.
type Kit struct {
	Ref       string
	Huella    string
	Variantes []Variante
}

// Variante es una construcción a ciegas con sus respuestas y los escenarios que debe cumplir.
type Variante struct {
	Nombre     string            `yaml:"nombre"`
	Respuestas map[string]string `yaml:"respuestas"`
	Escenarios []string          `yaml:"escenarios"`
	Carpeta    string            `yaml:"-"`
	Lenguaje   string            `yaml:"lenguaje,omitempty"`
	Comando    string            `yaml:"comando,omitempty"`
	Resultado  string            `yaml:"resultado"`
	Faltan     []string          `yaml:"faltan,omitempty"`
	Aprobada   bool              `yaml:"aprobada"`
}

// Certificacion es el resultado que queda junto al blueprint (certificacion.yaml).
type Certificacion struct {
	Ref       string     `yaml:"blueprint"`
	Huella    string     `yaml:"huella"` // sha256 de blueprint.yaml sin su línea version:
	Fecha     string     `yaml:"fecha"`
	Aprobada  bool       `yaml:"aprobada"`
	Variantes []Variante `yaml:"variantes"`
}

// huellaDe identifica la receta sin su versión, que QDD fija al proponer.
func huellaDe(b Blueprint) (string, error) {
	data, err := os.ReadFile(filepath.Join(b.Dir, ManifestName))
	if err != nil {
		return "", fmt.Errorf("la receta se certifica desde su carpeta: %w", err)
	}
	sum := sha256.Sum256(versionLine.ReplaceAll(data, []byte("version:")))
	return hex.EncodeToString(sum[:]), nil
}

// PrepararCertificacion deja, fuera del blueprint y del proyecto, una carpeta por variante con solo su receta. Lo que
// un agente ya construyó en ellas se conserva.
func PrepararCertificacion(b Blueprint) (Kit, error) {
	huella, err := huellaDe(b)
	if err != nil {
		return Kit{}, err
	}
	k := Kit{Ref: b.Ref(), Huella: huella}
	for _, v := range variantesDe(b) {
		dir, err := centralDir("certificaciones", b.ID+"-"+huella[:12], v.Nombre)
		if err != nil {
			return Kit{}, err
		}
		v.Carpeta = dir
		if err := os.WriteFile(filepath.Join(dir, RecetaFile), []byte(recetaParaConstruir(b, v)), 0644); err != nil {
			return Kit{}, err
		}
		k.Variantes = append(k.Variantes, v)
	}
	return k, nil
}

// variantesDe: la sugerida (valores por defecto) y, si la receta tiene algo que adaptar, otra con otros valores.
func variantesDe(b Blueprint) []Variante {
	sugerida := variante(b, VarianteSugerida, respuestasPorDefecto(b))
	adaptada := variante(b, VarianteAdaptada, respuestasAdaptadas(b))
	if fmt.Sprint(sugerida.Respuestas) == fmt.Sprint(adaptada.Respuestas) {
		return []Variante{sugerida}
	}
	return []Variante{sugerida, adaptada}
}

func variante(b Blueprint, nombre string, respuestas map[string]string) Variante {
	aplicadas, _, _ := b.Resolver(respuestas)
	_, escenarios := b.componer(aplicadas)
	v := Variante{Nombre: nombre, Respuestas: aplicadas}
	for _, e := range escenarios {
		v.Escenarios = append(v.Escenarios, e.ID)
	}
	return v
}

// respuestasAdaptadas elige en cada pregunta otra opción que la sugerida (la que más trae) y en cada parámetro otro
// valor: el primer ejemplo distinto o, sin ejemplos, el doble del número o el texto marcado como otro.
func respuestasAdaptadas(b Blueprint) map[string]string {
	out := map[string]string{}
	mayores := respuestasMayores(b)
	for _, p := range b.Preguntas {
		out[p.ID] = mayores[p.ID]
		if out[p.ID] != primero(p.PorDefecto, primeraOpcion(p)) {
			continue
		}
		out[p.ID] = otraOpcion(p, out[p.ID])
	}
	for _, p := range b.Parametros {
		out[p.ID] = otroValor(p)
	}
	return out
}

func otraOpcion(p Pregunta, actual string) string {
	for _, o := range p.Opciones {
		if o.Valor != actual {
			return o.Valor
		}
	}
	return actual
}

func otroValor(p Parametro) string {
	for _, e := range p.Ejemplos {
		if strings.TrimSpace(e) != strings.TrimSpace(p.PorDefecto) {
			return strings.TrimSpace(e)
		}
	}
	n, err := strconv.ParseFloat(strings.Replace(p.PorDefecto, ",", ".", 1), 64)
	if p.Tipo == tipoNumero && err == nil {
		return strconv.FormatFloat(n*2+1, 'f', -1, 64)
	}
	return "otro " + p.PorDefecto
}

// recetaParaConstruir es lo único que ve quien construye a ciegas: la receta de su variante, ya resuelta.
func recetaParaConstruir(b Blueprint, v Variante) string {
	gen, escenarios := b.componer(v.Respuestas)
	var s strings.Builder
	fmt.Fprintf(&s, "# Receta: %s (%s) · variante %s\n\n", b.Nombre, b.Ref(), v.Nombre)
	s.WriteString("Construye esto desde cero en esta carpeta, en el lenguaje que elijas, solo con esta receta (no busques otra implementación).\n")
	s.WriteString("1. Sigue los pasos en orden.\n2. Por cada escenario, una prueba automática que lleve su id en el nombre (p. ej. `" + primero(append(v.Escenarios, "X-01")...) + "`).\n")
	fmt.Fprintf(&s, "3. Deja en `%s` el lenguaje y el comando que corre todas las pruebas:\n\n```yaml\nlenguaje: <el que elegiste>\ncomando: \"<p. ej. go test ./...>\"\n```\n", ConstruccionFile)
	fmt.Fprintf(&s, "\n## Problema\n%s\n\n**Criterio de éxito:** %s\n", strings.TrimSpace(b.Ficha.ObjetivoOperativo.Problema), strings.TrimSpace(b.Ficha.ObjetivoOperativo.CriterioDeExito))
	fmt.Fprintf(&s, "\n## Alcance\n")
	escribirLista(&s, "- ✓ ", b.Alcance.Incluye)
	escribirLista(&s, "- ✗ ", b.Alcance.Excluye)
	fmt.Fprintf(&s, "\n## Arquitectura\n%s\n", strings.TrimSpace(b.Ficha.ObjetivoTecnologico.Arquitectura))
	s.WriteString("\n## Pasos, decisiones y vocabulario\n")
	escribirLista(&s, "- ", gen.Decisiones)
	s.WriteString("\n## Invariantes (no se rompen)\n")
	escribirLista(&s, "- ", gen.Invariantes)
	s.WriteString("\n## Escenarios (cada uno, una prueba)\n")
	for _, e := range escenarios {
		dado := ""
		if strings.TrimSpace(e.Dado) != "" {
			dado = "dado " + strings.TrimSpace(e.Dado) + "; "
		}
		fmt.Fprintf(&s, "- **%s** %s: %scuando %s → %s\n", e.ID, e.Titulo, dado, strings.TrimSpace(e.Cuando), strings.TrimSpace(e.Entonces))
	}
	return s.String()
}

// VerificarCertificacion corre las pruebas de cada variante construida y deja el resultado en certificacion.yaml.
func VerificarCertificacion(b Blueprint) (Certificacion, error) {
	k, err := PrepararCertificacion(b)
	if err != nil {
		return Certificacion{}, err
	}
	c := Certificacion{Ref: b.Ref(), Huella: k.Huella, Fecha: time.Now().Format("2006-01-02 15:04"), Aprobada: true}
	for _, v := range k.Variantes {
		v = verificarVariante(v)
		c.Aprobada = c.Aprobada && v.Aprobada
		c.Variantes = append(c.Variantes, v)
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return c, err
	}
	cabecera := "# Certificación ciega de 'qdd blueprint certify': cada variante la construyó, solo con la receta, alguien que no vio el original.\n"
	return c, os.WriteFile(filepath.Join(b.Dir, CertificacionFile), append([]byte(cabecera), data...), 0644)
}

func verificarVariante(v Variante) Variante {
	var hecho struct {
		Lenguaje string `yaml:"lenguaje"`
		Comando  string `yaml:"comando"`
	}
	data, err := os.ReadFile(filepath.Join(v.Carpeta, ConstruccionFile))
	if err != nil || yaml.Unmarshal(data, &hecho) != nil || strings.TrimSpace(hecho.Comando) == "" {
		v.Resultado = "sin construir: falta " + ConstruccionFile + " con su comando"
		return v
	}
	v.Lenguaje, v.Comando = hecho.Lenguaje, hecho.Comando
	v.Faltan = escenariosSinPrueba(v)
	codigo, salida := correrPruebas(v.Carpeta, hecho.Comando)
	v.Aprobada = codigo == 0 && len(v.Faltan) == 0
	v.Resultado = fmt.Sprintf("pruebas %s · escenarios con prueba %d/%d", resultadoDe(codigo, salida), len(v.Escenarios)-len(v.Faltan), len(v.Escenarios))
	return v
}

func resultadoDe(codigo int, salida string) string {
	if codigo == 0 {
		return "✅"
	}
	return fmt.Sprintf("❌ (código %d: %s)", codigo, recortar(strings.TrimSpace(salida), 80))
}

func correrPruebas(dir, comando string) (int, string) {
	shell, flag := "sh", "-c"
	if runtime.GOOS == "windows" {
		shell, flag = "cmd", "/C"
	}
	ctx, cancel := context.WithTimeout(context.Background(), maxPruebaCiega)
	defer cancel()
	cmd := exec.CommandContext(ctx, shell, flag, comando)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err == nil {
		return 0, string(out)
	}
	if exit, ok := err.(*exec.ExitError); ok {
		return exit.ExitCode(), string(out)
	}
	return -1, err.Error()
}

// escenariosSinPrueba: cada escenario de la variante tiene que aparecer por su id en lo construido (sin contar la
// receta ni construccion.yaml).
func escenariosSinPrueba(v Variante) []string {
	var construido strings.Builder
	_ = filepath.Walk(v.Carpeta, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() && (info.Name() == ".git" || info.Name() == "node_modules" || info.Name() == "target" || info.Name() == "vendor") {
			return filepath.SkipDir
		}
		if info.IsDir() || info.Size() > 1<<20 || info.Name() == RecetaFile || info.Name() == ConstruccionFile {
			return nil
		}
		data, _ := os.ReadFile(path)
		construido.Write(data)
		return nil
	})
	texto := construido.String()
	var faltan []string
	for _, id := range v.Escenarios {
		if strings.Contains(texto, id) {
			continue
		}
		faltan = append(faltan, id)
	}
	sort.Strings(faltan)
	return faltan
}

// CertificacionVigente lee la certificación del blueprint; vale si está aprobada y es de esta misma receta.
func CertificacionVigente(b Blueprint) (Certificacion, bool) {
	var c Certificacion
	data, err := os.ReadFile(filepath.Join(b.Dir, CertificacionFile))
	if err != nil || yaml.Unmarshal(data, &c) != nil {
		return Certificacion{}, false
	}
	huella, err := huellaDe(b)
	return c, err == nil && c.Aprobada && c.Huella == huella
}

// LineaCertificacion resume la certificación: ✅/❌ y cada variante con su lenguaje y lo que le falta.
func LineaCertificacion(c Certificacion) string {
	partes := make([]string, 0, len(c.Variantes))
	aprobadas := 0
	for _, v := range c.Variantes {
		marca := "❌"
		if v.Aprobada {
			marca, aprobadas = "✅", aprobadas+1
		}
		parte := fmt.Sprintf("%s %s", marca, v.Nombre)
		if v.Lenguaje != "" {
			parte += " (" + v.Lenguaje + ")"
		}
		parte += ": " + v.Resultado
		if len(v.Faltan) > 0 {
			parte += " · sin prueba " + strings.Join(v.Faltan, ", ")
		}
		partes = append(partes, parte)
	}
	return fmt.Sprintf("certificación ciega %d/%d · %s", aprobadas, len(c.Variantes), strings.Join(partes, " · "))
}

// CarpetaDeAporte es la carpeta del blueprint en su aporte en curso, sin cambiar de rama: un id o una carpeta.
func CarpetaDeAporte(root, pedido string) (Blueprint, error) {
	if exists(filepath.Join(pedido, ManifestName)) {
		return ParseDir(pedido)
	}
	repo, subdir, err := origenDe(root, pedido, "")
	if err != nil {
		return Blueprint{}, err
	}
	espacio, err := espacioPath(repo)
	if err != nil {
		return Blueprint{}, err
	}
	dir := filepath.Join(espacio, filepath.FromSlash(subdir))
	if !exists(filepath.Join(dir, ManifestName)) {
		return Blueprint{}, fmt.Errorf("no hay un aporte de '%s': qdd blueprint contribute %s (o distill desde un módulo)", pedido, pedido)
	}
	return ParseDir(dir)
}
