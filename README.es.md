# resume-mcp

Servidor MCP en Go que expone el perfil de un candidato y el formato de un CV a un LLM, y luego guarda el currículum personalizado generado por el modelo en un archivo Markdown.

Este proyecto está pensado como ejemplo educativo para entender cómo se construyen servidores Model Context Protocol (MCP), cómo se exponen herramientas al modelo, cómo se prepara el contexto para el LLM y cómo se guarda el resultado final en una salida útil.

La idea no es que sea una solución de producción compleja, sino una base clara y fácil de entender para que otros desarrolladores puedan reutilizarla como referencia para construir sus propios MCP.

El servidor no genera el contenido del CV por sí mismo. El LLM conectado es responsable de analizar la descripción del puesto, redactar el currículum y llamar a la herramienta `save_resume`.

## ¿Qué problema resuelve?

Este proyecto demuestra un patrón muy útil:

- un cliente o agente llama a herramientas expuestas por un servidor MCP
- el servidor devuelve contexto estructurado y relevante
- el modelo genera una respuesta adaptada a un caso concreto
- la salida se guarda en disco para reutilizarla o descargarla

En este caso, el caso de uso real es adaptar un CV a un puesto concreto usando información del perfil del candidato y la estructura del documento esperada.

## ¿Cómo funciona?

La aplicación sigue una estructura muy simple y clara:

1. Capa de servidor
   - registra herramientas y prompts MCP
   - maneja solicitudes JSON-RPC 2.0
   - valida entradas y despacha la llamada

2. Capa de negocio
   - lee el perfil del candidato
   - lee el formato del CV
   - prepara el contexto para el LLM

3. Capa de almacenamiento
   - guarda los CV generados en `output/`
   - devuelve una URL de descarga para cada archivo

4. Capa de configuración
   - maneja variables de entorno como el puerto y la URL pública
   - valida la configuración antes de arrancar la app

## Estructura del proyecto

```text
resume-mcp/
├── cmd/
│   └── resume-mcp/
│       └── main.go
├── internal/
│   ├── config/
│   ├── profile/
│   └── resume/
├── mcp/
│   ├── server.go
│   ├── tools.go
│   ├── prompts.go
│   ├── types.go
│   └── server_test.go
├── data/
│   ├── profile.json
│   └── resume_format.json
├── output/
├── go.mod
├── README.md
├── README.es.md
└── resume-mcp-server
```

## Herramientas MCP disponibles

El servidor expone estas herramientas:

- `get_profile`
  - devuelve el contenido completo del perfil del candidato

- `get_resume_format`
  - devuelve el formato y la estructura del CV

- `prepare_job_context`
  - prepara el contexto completo para adaptar el CV a un puesto concreto

- `save_resume`
  - guarda el CV generado en Markdown y devuelve una URL de descarga

## Prompts disponibles

También expone el prompt `tailor_resume`, que guía al modelo a través del flujo completo:

1. leer la descripción del puesto
2. obtener el contexto con `prepare_job_context`
3. redactar un CV adaptado en Markdown
4. guardar el resultado con `save_resume`

## Requisitos

- Go 1.22 o superior
- Git
- Node.js y `npx` solo si se usa el puente `mcp-remote`

## Inicio rápido

Clona el repositorio y entra en la carpeta:

```bash
git clone https://github.com/lcdosguzman/resume-mcp.git
cd resume-mcp
```

Comprueba que Go está instalado y ejecuta la suite de pruebas:

```bash
go version
go test ./...
```

Inicia el servidor:

```bash
go run ./cmd/resume-mcp
```

Comprueba el estado del servicio:

```bash
curl http://127.0.0.1:8090/health
```

El endpoint MCP por defecto es:

```text
http://127.0.0.1:8090/mcp
```

## Configuración

Puedes personalizar el puerto y la URL pública con variables de entorno:

```bash
MCP_PORT=9000 go run ./cmd/resume-mcp
MCP_PORT=9000 MCP_PUBLIC_URL=http://127.0.0.1:9000 go run ./cmd/resume-mcp
```

## Datos del perfil y formato del CV

Antes de usar el proyecto con un LLM real, modifica estos archivos:

- `data/profile.json`
- `data/resume_format.json`

El perfil debe contener información real y útil del candidato, sin secretos ni datos sensibles. El formato define la estructura del CV y las secciones que se deben respetar.

## Ejemplo de uso manual

Listar herramientas:

```bash
curl -s -X POST http://127.0.0.1:8090/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
```

Leer el perfil:

```bash
curl -s -X POST http://127.0.0.1:8090/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"get_profile","arguments":{}}}'
```

## ¿Por qué este proyecto es útil como base?

Porque muestra, de forma concreta y legible, cómo construir un MCP en Go con:

- herramientas
- prompts
- validación de entrada
- configuración
- acceso a archivos
- flujo con LLM
- salida persistente

Es un buen punto de partida para aprender el patrón sin entrar en una complejidad innecesaria.

## Buenas prácticas que ya incluye

- separación de responsabilidades por capas
- validación de entradas y archivos
- control de nombres de archivo
- configuración centralizada
- pruebas básicas de comportamiento

## Mejoras recomendadas

Si quieres llevar este repo un paso más allá como referencia pública, estas son las mejoras más interesantes:

- mejorar el logging y la trazabilidad
- centralizar aún más la validación
- mejorar los errores para clientes externos
- añadir más pruebas de integración
- hacer más explícita la documentación del flujo MCP

## Nota final

Este repositorio está pensado como un ejemplo claro de trabajo con MCP, útil para aprender, experimentar y reutilizar. No pretende ser un producto final de producción, sino una base sólida y legible que otros desarrolladores puedan usar para entender el patrón y construir sobre él.

---

Este documento está escrito en español como complemento del README principal en inglés. Está pensado para facilitar la comprensión del proyecto para una audiencia hispanohablante, manteniendo el proyecto base en inglés como referencia principal para GitHub.
