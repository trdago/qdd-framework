package consola

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Rule: Prohibido usar 'else'
// Segmento de recursos de la barra (blueprint consola-claude-code v4.0.0, pregunta recursos; RFC 0006 §3.8): una
// segunda línea «D N% · G0 N% · G1 — · R N% · C N%» con el disco de la carpeta del proyecto, el % de uso de cada GPU, la
// RAM y la CPU. Solo lee contadores locales; la GPU se consulta como mucho cada 5 s con tope de 300 ms. Lo que existe sin
// permiso para medirlo va con «—» y lo que el sistema no expone se omite: nunca un valor inventado.

const (
	cadaGPU     = 5 * time.Second
	topeGPU     = 300 * time.Millisecond
	topeLinea   = 400 * time.Millisecond
	vigenciaCPU = time.Minute
)

// Medidas son los recursos de un momento. Un nil se omite; en GPUs, un nil es una GPU sin lectura («—»).
type Medidas struct {
	Disco *float64
	GPUs  []*float64
	RAM   *float64
	CPU   *float64
}

// LineaRecursos arma la segunda línea de la barra; sin medidas, "".
func LineaRecursos(m Medidas) string {
	var partes []string
	if m.Disco != nil {
		partes = append(partes, "D "+porcentaje(*m.Disco))
	}
	for i, g := range m.GPUs {
		v := "—"
		if g != nil {
			v = porcentaje(*g)
		}
		partes = append(partes, fmt.Sprintf("G%d %s", i, v))
	}
	if m.RAM != nil {
		partes = append(partes, "R "+porcentaje(*m.RAM))
	}
	if m.CPU != nil {
		partes = append(partes, "C "+porcentaje(*m.CPU))
	}
	return strings.Join(partes, " · ")
}

// muestraCPU son los tiempos acumulados de la CPU (en las unidades del sistema): el uso sale de dos muestras.
type muestraCPU struct {
	Total  uint64 `json:"total"`
	Ocioso uint64 `json:"ocioso"`
}

func usoCPU(antes, ahora muestraCPU) (float64, bool) {
	if ahora.Total <= antes.Total || ahora.Ocioso < antes.Ocioso {
		return 0, false
	}
	dt := float64(ahora.Total - antes.Total)
	di := float64(ahora.Ocioso - antes.Ocioso)
	if di > dt {
		return 0, false
	}
	return (1 - di/dt) * 100, true
}

// sensores son las lecturas del sistema; un campo nil es un recurso que este sistema no expone.
type sensores struct {
	disco  func(dir string) (float64, bool)
	ram    func() (float64, bool)
	cpu    func() (muestraCPU, bool)
	nvidia func() (string, error)
	otras  func() []*float64
}

func sensoresDelSistema() sensores {
	return sensores{disco: usoDisco, ram: usoRAM, cpu: leerCPU, nvidia: consultarNvidia, otras: otrasGPUs}
}

// cacheRecursos guarda entre una barra y la siguiente lo que necesita dos lecturas (CPU) o es caro (GPU).
type cacheRecursos struct {
	CPU   *muestraCPU `json:"cpu,omitempty"`
	CPUEn int64       `json:"cpu_en,omitempty"`
	GPU   []*float64  `json:"gpu"`
	GPUEn int64       `json:"gpu_en,omitempty"`
}

func leerCache(ruta string) cacheRecursos {
	var c cacheRecursos
	data, err := os.ReadFile(ruta)
	if err != nil {
		return c
	}
	_ = json.Unmarshal(data, &c)
	return c
}

func guardarCache(ruta string, c cacheRecursos) {
	if ruta == "" {
		return
	}
	data, err := json.Marshal(c)
	if err != nil {
		return
	}
	_ = escribir(ruta, data)
}

func (s sensores) medir(dir, ruta string, now time.Time) Medidas {
	var m Medidas
	c := leerCache(ruta)
	if s.disco != nil && dir != "" {
		if p, ok := s.disco(dir); ok {
			m.Disco = &p
		}
	}
	if s.ram != nil {
		if p, ok := s.ram(); ok {
			m.RAM = &p
		}
	}
	if s.cpu != nil {
		m.CPU, c = s.medirCPU(c, now)
	}
	m.GPUs, c = s.medirGPUs(c, now)
	guardarCache(ruta, c)
	return m
}

func (s sensores) medirCPU(c cacheRecursos, now time.Time) (*float64, cacheRecursos) {
	ahora, ok := s.cpu()
	if !ok {
		return nil, c
	}
	antes, en := c.CPU, time.UnixMilli(c.CPUEn)
	c.CPU, c.CPUEn = &ahora, now.UnixMilli()
	if antes == nil || now.Sub(en) > vigenciaCPU {
		return nil, c
	}
	p, ok := usoCPU(*antes, ahora)
	if !ok {
		return nil, c
	}
	return &p, c
}

func (s sensores) medirGPUs(c cacheRecursos, now time.Time) ([]*float64, cacheRecursos) {
	nvidia := c.GPU
	if s.nvidia != nil && now.Sub(time.UnixMilli(c.GPUEn)) >= cadaGPU {
		nvidia = nil
		if out, err := s.nvidia(); err == nil {
			nvidia = parseNvidia(out)
		}
		c.GPU, c.GPUEn = nvidia, now.UnixMilli()
	}
	if s.nvidia == nil {
		nvidia = nil
	}
	gpus := append([]*float64{}, nvidia...)
	if s.otras != nil {
		gpus = append(gpus, s.otras()...)
	}
	return gpus, c
}

// consultarNvidia pide el % de uso de cada GPU NVIDIA, una por línea, con tope de 300 ms.
func consultarNvidia() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), topeGPU)
	defer cancel()
	out, err := exec.CommandContext(ctx, "nvidia-smi", "--query-gpu=utilization.gpu", "--format=csv,noheader,nounits").Output()
	return string(out), err
}

func parseNvidia(out string) []*float64 {
	var gpus []*float64
	for _, l := range strings.Split(out, "\n") {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		v, err := strconv.ParseFloat(l, 64)
		if err != nil {
			gpus = append(gpus, nil)
			continue
		}
		gpus = append(gpus, &v)
	}
	return gpus
}

func parseMeminfo(s string) (float64, bool) {
	var total, disponible float64
	for _, l := range strings.Split(s, "\n") {
		campos := strings.Fields(l)
		if len(campos) < 2 {
			continue
		}
		v, err := strconv.ParseFloat(campos[1], 64)
		if err != nil {
			continue
		}
		switch campos[0] {
		case "MemTotal:":
			total = v
		case "MemAvailable:":
			disponible = v
		}
	}
	if total <= 0 || disponible <= 0 || disponible > total {
		return 0, false
	}
	return (total - disponible) / total * 100, true
}

// parseProcStat lee la línea «cpu» de /proc/stat: total = user…steal; ocioso = idle + iowait.
func parseProcStat(s string) (muestraCPU, bool) {
	linea, _, _ := strings.Cut(s, "\n")
	campos := strings.Fields(linea)
	if len(campos) < 9 || campos[0] != "cpu" {
		return muestraCPU{}, false
	}
	var v [8]uint64
	for i := range v {
		n, err := strconv.ParseUint(campos[i+1], 10, 64)
		if err != nil {
			return muestraCPU{}, false
		}
		v[i] = n
	}
	var m muestraCPU
	for _, n := range v {
		m.Total += n
	}
	m.Ocioso = v[3] + v[4]
	return m, true
}

func porcentajeUsado(total, libre, disponible uint64) (float64, bool) {
	if total == 0 || libre > total {
		return 0, false
	}
	usado := total - libre
	if usado+disponible == 0 {
		return 0, false
	}
	return float64(usado) / float64(usado+disponible) * 100, true
}

// barraCon agrega a la barra la línea de recursos, pegada a la derecha, si s no es nil. La medición tiene un tope total: si se pasa, la barra
// sale sin la segunda línea.
func barraCon(datos []byte, now time.Time, modo string, ancho int, s *sensores, ruta string) string {
	linea := Barra(datos, now, modo, ancho)
	if linea == "" || s == nil {
		return linea
	}
	var in entrada
	_ = json.Unmarshal(datos, &in)
	dir := primero(in.Workspace.ProjectDir, in.Workspace.CurrentDir, in.Cwd)
	resultado := make(chan string, 1)
	go func() {
		defer func() { _ = recover() }()
		resultado <- LineaRecursos(s.medir(dir, ruta, now))
	}()
	select {
	case r := <-resultado:
		if r == "" {
			return linea
		}
		return linea + "\n" + aLaDerecha(r, ancho)
	case <-time.After(topeLinea):
		return linea
	}
}

// BarraConRecursos es la barra con la línea de recursos del sistema abajo, a la derecha; la caché va en dirCache.
func BarraConRecursos(datos []byte, now time.Time, modo string, ancho int, dirCache string) string {
	s := sensoresDelSistema()
	return barraCon(datos, now, modo, ancho, &s, filepath.Join(dirCache, "recursos.json"))
}
