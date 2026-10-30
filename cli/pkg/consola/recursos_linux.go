package consola

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Rule: Prohibido usar 'else'
// Linux: RAM de /proc/meminfo, CPU de /proc/stat y las GPU que no son NVIDIA de /sys/class/drm (gpu_busy_percent, que
// exponen p. ej. las AMD; una GPU sin ese contador, como la integrada de Intel sin root, queda con «—»).

func usoRAM() (float64, bool) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, false
	}
	return parseMeminfo(string(data))
}

func leerCPU() (muestraCPU, bool) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return muestraCPU{}, false
	}
	return parseProcStat(string(data))
}

var tarjeta = regexp.MustCompile(`^card(\d+)$`)

const vendorNvidia = "0x10de"

func otrasGPUs() []*float64 {
	return gpusDRM("/sys/class/drm")
}

// gpusDRM recorre las tarjetas en orden de número; las NVIDIA las mide nvidia-smi.
func gpusDRM(raiz string) []*float64 {
	entradas, err := os.ReadDir(raiz)
	if err != nil {
		return nil
	}
	type tarjetaDRM struct {
		n   int
		dir string
	}
	var tarjetas []tarjetaDRM
	for _, e := range entradas {
		m := tarjeta.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		n, _ := strconv.Atoi(m[1])
		tarjetas = append(tarjetas, tarjetaDRM{n, filepath.Join(raiz, e.Name(), "device")})
	}
	sort.Slice(tarjetas, func(i, j int) bool { return tarjetas[i].n < tarjetas[j].n })
	var gpus []*float64
	for _, t := range tarjetas {
		vendor, err := os.ReadFile(filepath.Join(t.dir, "vendor"))
		if err != nil || strings.TrimSpace(string(vendor)) == vendorNvidia {
			continue
		}
		gpus = append(gpus, ocupada(filepath.Join(t.dir, "gpu_busy_percent")))
	}
	return gpus
}

func ocupada(ruta string) *float64 {
	data, err := os.ReadFile(ruta)
	if err != nil {
		return nil
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(string(data)), 64)
	if err != nil {
		return nil
	}
	return &v
}
