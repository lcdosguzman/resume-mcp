package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	mcpserver "resume-mcp/mcp"
)

const (
	defaultPort   = "8090"
	perfilPath    = "perfil.json"
	formatoCVPath = "data/formato_cv.json"
	outputDir     = "output"
)

func main() {
	port := os.Getenv("MCP_PORT")
	if port == "" {
		port = defaultPort
	}
	publicURL := os.Getenv("MCP_PUBLIC_URL")
	if publicURL == "" {
		publicURL = "http://127.0.0.1:" + port
	}
	downloadBaseURL = strings.TrimRight(publicURL, "/") + "/downloads/"

	srv := mcpserver.NewServer("resume-mcp", "0.1.0")
	registerTools(srv)

	mux := http.NewServeMux()
	mux.Handle("/mcp", srv.Handler())
	mux.Handle("/downloads/", http.StripPrefix("/downloads/", http.FileServer(http.Dir(outputDir))))
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	addr := "127.0.0.1:" + port
	log.Printf("resume-mcp escuchando en http://%s/mcp (Ctrl+C para detener)", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("error iniciando servidor: %v", err)
	}
}

func registerTools(srv *mcpserver.Server) {
	srv.RegisterTool(mcpserver.Tool{
		Name:        "obtener_perfil",
		Description: "Devuelve los datos personales, experiencia, educación y habilidades del usuario en formato JSON. Usar esto primero para conocer la información disponible antes de adaptar un currículum.",
		InputSchema: mcpserver.InputSchema{Type: "object"},
	}, toolObtenerPerfil)

	srv.RegisterTool(mcpserver.Tool{
		Name:        "obtener_formato_cv",
		Description: "Devuelve la estructura/formato que debe seguir el currículum generado (secciones, orden, estilo, longitud). Usar junto con obtener_perfil.",
		InputSchema: mcpserver.InputSchema{Type: "object"},
	}, toolObtenerFormatoCV)

	srv.RegisterTool(mcpserver.Tool{
		Name:        "preparar_contexto_vacante",
		Description: "Recibe el texto de una vacante y devuelve, junto con el perfil y el formato del CV, instrucciones para generar el currículum adaptado a esa vacante. El modelo que llama a esta tool debe usar el resultado para redactar el currículum en Markdown y luego guardarlo con la tool 'guardar_cv'.",
		InputSchema: mcpserver.InputSchema{
			Type: "object",
			Properties: map[string]mcpserver.Property{
				"vacante": {Type: "string", Description: "Texto completo de la vacante/oferta laboral (descripción, requisitos, responsabilidades)."},
			},
			Required: []string{"vacante"},
		},
	}, toolPrepararContextoVacante)

	srv.RegisterTool(mcpserver.Tool{
		Name:        "guardar_cv",
		Description: "Guarda el currículum adaptado (ya redactado en Markdown por el modelo) como un archivo .md dentro de la carpeta output/ del servidor.",
		InputSchema: mcpserver.InputSchema{
			Type: "object",
			Properties: map[string]mcpserver.Property{
				"contenido":      {Type: "string", Description: "Contenido completo del currículum en formato Markdown."},
				"nombre_archivo": {Type: "string", Description: "Nombre de archivo opcional (sin extensión). Si no se indica, se genera uno con fecha/hora."},
			},
			Required: []string{"contenido"},
		},
	}, toolGuardarCV)
}

// ---------- Handlers ----------

func toolObtenerPerfil(_ json.RawMessage) (mcpserver.CallToolResult, error) {
	data, err := os.ReadFile(perfilPath)
	if err != nil {
		return errResult(fmt.Errorf("no se pudo leer %s: %w", perfilPath, err))
	}
	return textResult(string(data)), nil
}

func toolObtenerFormatoCV(_ json.RawMessage) (mcpserver.CallToolResult, error) {
	data, err := os.ReadFile(formatoCVPath)
	if err != nil {
		return errResult(fmt.Errorf("no se pudo leer %s: %w", formatoCVPath, err))
	}
	return textResult(string(data)), nil
}

type prepararArgs struct {
	Vacante string `json:"vacante"`
}

func toolPrepararContextoVacante(raw json.RawMessage) (mcpserver.CallToolResult, error) {
	var args prepararArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return errResult(fmt.Errorf("argumentos inválidos: %w", err))
	}
	if strings.TrimSpace(args.Vacante) == "" {
		return errResult(fmt.Errorf("el campo 'vacante' no puede estar vacío"))
	}

	perfil, err := os.ReadFile(perfilPath)
	if err != nil {
		return errResult(fmt.Errorf("no se pudo leer %s: %w", perfilPath, err))
	}
	formato, err := os.ReadFile(formatoCVPath)
	if err != nil {
		return errResult(fmt.Errorf("no se pudo leer %s: %w", formatoCVPath, err))
	}

	var sb strings.Builder
	sb.WriteString("# Contexto para generar currículum adaptado\n\n")
	sb.WriteString("## Instrucciones\n")
	sb.WriteString("Redactá un currículum en Markdown adaptado a la vacante de abajo, usando ")
	sb.WriteString("únicamente información real presente en el perfil (no inventar experiencia, ")
	sb.WriteString("títulos ni fechas). Priorizá y reordená la experiencia, logros y habilidades ")
	sb.WriteString("que sean más relevantes para esta vacante puntual. Seguí la estructura indicada ")
	sb.WriteString("en 'formato_cv'. Incluí obligatoriamente la sección final de evaluación de ajuste ")
	sb.WriteString("con una puntuación entera de 1 a 10, fortalezas y debilidades o brechas, ")
	sb.WriteString("distinguiendo la experiencia demostrada de la no evidenciada. Cuando termines, ")
	sb.WriteString("guardá el resultado usando la tool 'guardar_cv'.\n\n")

	sb.WriteString("## Vacante\n```\n")
	sb.WriteString(args.Vacante)
	sb.WriteString("\n```\n\n")

	sb.WriteString("## Perfil del candidato (perfil.json)\n```json\n")
	sb.Write(perfil)
	sb.WriteString("\n```\n\n")

	sb.WriteString("## Formato requerido del currículum (formato_cv.json)\n```json\n")
	sb.Write(formato)
	sb.WriteString("\n```\n")

	return textResult(sb.String()), nil
}

type guardarArgs struct {
	Contenido     string `json:"contenido"`
	NombreArchivo string `json:"nombre_archivo"`
}

var nombreArchivoSeguro = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)
var downloadBaseURL string

func toolGuardarCV(raw json.RawMessage) (mcpserver.CallToolResult, error) {
	var args guardarArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return errResult(fmt.Errorf("argumentos inválidos: %w", err))
	}
	if strings.TrimSpace(args.Contenido) == "" {
		return errResult(fmt.Errorf("el campo 'contenido' no puede estar vacío"))
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return errResult(fmt.Errorf("no se pudo crear carpeta de salida: %w", err))
	}

	nombre := strings.TrimSpace(args.NombreArchivo)
	if nombre == "" {
		nombre = "cv"
	} else {
		nombre = nombreArchivoSeguro.ReplaceAllString(nombre, "_")
	}
	nombre = strings.Trim(nombre, "_")
	if nombre == "" {
		nombre = "cv"
	}
	nombre += "_" + time.Now().Format("20060102_150405")

	path := filepath.Join(outputDir, nombre+".md")
	if err := os.WriteFile(path, []byte(args.Contenido), 0o644); err != nil {
		return errResult(fmt.Errorf("no se pudo escribir el archivo: %w", err))
	}

	abs, _ := filepath.Abs(path)
	downloadURL := downloadBaseURL + url.PathEscape(filepath.Base(path))
	return textResult(fmt.Sprintf("Currículum guardado en: %s\nDescarga: %s", abs, downloadURL)), nil
}

// ---------- Helpers ----------

func textResult(text string) mcpserver.CallToolResult {
	return mcpserver.CallToolResult{
		Content: []mcpserver.ContentBlock{{Type: "text", Text: text}},
	}
}

func errResult(err error) (mcpserver.CallToolResult, error) {
	return mcpserver.CallToolResult{}, err
}
