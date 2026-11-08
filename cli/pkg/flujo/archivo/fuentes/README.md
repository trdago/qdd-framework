# JetBrains Mono para el archivo de un flujo

La familia de la marca de QDD (qdd-movile `docs/marca/tokens.json`, `semantico.fuente.familia`) en sus tres pesos
cargados: 400, 500 y 700. Licencia SIL OFL 1.1 en `LICENCIA-JetBrainsMono.txt`.

Recortada a latín (español), puntuación, flechas, cajas y figuras geométricas, y los símbolos de la marca, para que
el archivo HTML no pese de más. Se regenera desde los TTF de qdd-movile `assets/fonts` con fonttools:

    pyftsubset JetBrainsMono-<Peso>.ttf --flavor=woff2 --layout-features='*' \
      --unicodes="U+0020-007E,U+00A0-017F,U+2010-205E,U+20AC,U+2190-21FF,U+2500-25FF,U+2713,U+2715,U+2717" \
      --output-file=JetBrainsMono-<Peso>.woff2
