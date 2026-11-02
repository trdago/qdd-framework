// materializar.js: reproduce, en el reloj del panel, los cuadros que qdd calculó ('qdd consola materializar') para lo
// que cambió. No decide nada: avanza un cuadro cada `ms` y, en el último, le avisa al panel ({ fin: clave }) para que
// vuelva a dibujar el texto de siempre. Cada línea es una fila de segmentos con su color, negrita y atenuado.
export default function Materializar(props, surface) {
  const { Box, Text } = surface.elements
  const cuadros = Array.isArray(props?.cuadros) ? props.cuadros : []
  if (surface.state === undefined) {
    surface.setState({ i: 0, avisado: false })
    surface.every(Math.max(16, Number(props?.ms) || 40), () => {
      const s = surface.state || { i: 0, avisado: false }
      if (s.i + 1 < cuadros.length) return surface.setState({ ...s, i: s.i + 1 })
      if (s.avisado) return
      surface.setState({ ...s, avisado: true })
      surface.post({ fin: String(props?.clave || '') })
    })
  }
  const cuadro = cuadros[Math.min(surface.state?.i ?? 0, cuadros.length - 1)] || []
  return Box({
    flexDirection: 'column',
    children: cuadro.map((linea, r) =>
      Box({
        key: 'l' + r,
        flexDirection: 'row',
        children: (linea || []).map((s, j) =>
          Text({
            key: 's' + j,
            ...(s.color ? { color: s.color } : {}),
            ...(s.negrita ? { weight: 'bold' } : {}),
            ...(s.tenue ? { dimColor: true } : {}),
            children: [String(s.texto ?? '')],
          }),
        ),
      }),
    ),
  })
}
