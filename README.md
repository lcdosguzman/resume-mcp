# resume-mcp

Servidor MCP en Go (sin dependencias externas) que sirve tu perfil y el formato
de tu currículum, y le da a un modelo (Claude u otro cliente MCP) todo lo
necesario para generar un currículum en Markdown adaptado a una vacante
puntual.

**Importante:** este servidor NO genera el texto del currículum. Solo expone
tus datos (`data/perfil.json`), el formato deseado (`data/formato_cv.json`)
y guarda el resultado que el modelo cliente redacte. La "inteligencia" la
pone el LLM que consume el MCP (por ejemplo, vos hablando con Claude).

## 1. Requisitos

- Go 1.22 o superior instalado (`go version`).

No hace falta `go mod tidy` ni conexión a internet: el proyecto usa
únicamente la librería estándar de Go.

## 2. Completar tus datos

Antes de usarlo, editá estos dos archivos:

- `data/perfil.json`: tus datos reales (experiencia, educación, skills, etc).
  Viene con una plantilla y placeholders — reemplazalos y borrá el campo
  `"_nota"`.
- `data/formato_cv.json`: cómo querés que se vea el currículum generado
  (secciones, orden, reglas de estilo). Incluye una sección final obligatoria
  con la puntuación de ajuste a la vacante, fortalezas y brechas. Ya viene con
  un formato razonable, ajustalo si querés.

## 3. Compilar y correr

```bash
cd resume-mcp
go build -o resume-mcp-server .
./resume-mcp-server
```

Por defecto levanta en `http://127.0.0.1:8090/mcp`. Podés cambiar el puerto
con la variable de entorno `MCP_PORT`:

```bash
MCP_PORT=9000 ./resume-mcp-server
```

Vas a ver:
```
resume-mcp escuchando en http://127.0.0.1:8090/mcp (Ctrl+C para detener)
```

Podés chequear que está vivo con:
```bash
curl http://127.0.0.1:8090/health
```

## 4. Probar manualmente con curl

```bash
# Listar las tools disponibles
curl -s -X POST http://127.0.0.1:8090/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'

# Traer tu perfil
curl -s -X POST http://127.0.0.1:8090/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"obtener_perfil","arguments":{}}}'
```

## 5. Tools expuestas

| Tool | Descripción |
|---|---|
| `obtener_perfil` | Devuelve `perfil.json` completo. |
| `obtener_formato_cv` | Devuelve `formato_cv.json` completo. |
| `preparar_contexto_vacante` | Recibe `{ "vacante": "texto de la vacante" }` y devuelve perfil + formato + la vacante + instrucciones, listo para que el modelo redacte el CV. |
| `guardar_cv` | Recibe `{ "contenido": "...markdown...", "nombre_archivo": "opcional" }`, guarda el archivo `.md` en `output/` y devuelve un enlace de descarga. |

Flujo típico que sigue el modelo cliente:
1. Llama `obtener_perfil` y `obtener_formato_cv` (o directamente
   `preparar_contexto_vacante`, que ya incluye ambos).
2. Redacta el currículum en Markdown adaptado a la vacante.
3. Añade al final una evaluación honesta de ajuste con puntuación de 1 a 10,
  fortalezas y debilidades o brechas basadas en la evidencia disponible.
4. Llama `guardar_cv` con el contenido final → queda en `output/*.md` y la respuesta incluye un enlace de descarga.

Los archivos guardados se sirven en `http://127.0.0.1:8090/downloads/<archivo>`. Si el servidor
se publica con otro host o dominio, definí `MCP_PUBLIC_URL` antes de iniciarlo para que la tool
devuelva enlaces utilizables por el cliente, por ejemplo `MCP_PUBLIC_URL=https://cv.example.com`.

## 6. Conectarlo a un cliente MCP

Este servidor usa **Streamable HTTP** en modo simple (JSON-RPC 2.0 por POST,
sin streaming SSE, ya que no necesita enviar mensajes del servidor al
cliente por iniciativa propia). Esto es compatible con cualquier cliente MCP
que hable HTTP directo.

**Claude Desktop / Claude Code** (config nativa) esperan generalmente un
servidor por `stdio` (un comando que levantan ellos), no una URL HTTP suelta.
Para conectar este servidor HTTP tenés dos caminos:

**Opción A — Bridge con `mcp-remote` (recomendado, no requiere tocar el código):**

En la config de tu cliente (por ejemplo
`~/Library/Application Support/Claude/claude_desktop_config.json` en macOS,
o el equivalente en tu SO), agregá:

```json
{
  "mcpServers": {
    "resume-mcp": {
      "command": "npx",
      "args": ["-y", "mcp-remote", "http://127.0.0.1:8090/mcp"]
    }
  }
}
```

Esto requiere tener Node.js instalado (para `npx`). El servidor Go tiene que
estar corriendo (`./resume-mcp-server`) antes de abrir el cliente.

**Opción B — MCP Inspector (para debug/desarrollo):**

```bash
npx @modelcontextprotocol/inspector
```

Y apuntalo a `http://127.0.0.1:8090/mcp` como servidor HTTP. Útil para
probar las tools de forma interactiva sin depender de Claude Desktop.

## 7. Estructura del proyecto

```
resume-mcp/
├── go.mod
├── main.go              # wiring: rutas de datos, registro de tools, servidor HTTP
├── mcp/
│   ├── types.go          # tipos JSON-RPC 2.0 y MCP (initialize, tools/list, tools/call)
│   └── server.go          # dispatcher HTTP del protocolo MCP
├── data/
│   ├── perfil.json        # TUS DATOS (completar)
│   └── formato_cv.json    # formato/estructura del CV a generar
└── output/                 # currículums generados (.md)
```

## 8. Próximos pasos (fuera del alcance de esta v1)

- Exportar a PDF (por ejemplo con la lib de Go `gofpdf`, o convirtiendo el
  `.md` con `pandoc`).
- Soportar más de una vacante / batch.
- Guardar historial de postulaciones (qué vacante generó qué CV y cuándo).
- Autenticación básica si el servidor se expone más allá de `127.0.0.1`.
