package api

import (
	"bytes"
	"html/template"
	"io"
	"log"
	"os"
)

const (
	ENV_API_MODE        = "API_MODE"
	DebugMode    string = "debug"
	ReleaseMode  string = "release"
	TestMode     string = "test"
)
const (
	debugCode = iota
	releaseCode
	testCode
)

// DefaultWriter is the default io.Writer used the Api for debug output and
// middleware output like Logger() or Recovery().
// Note that both Logger and Recovery provides custom ways to configure their
// output io.Writer.
// To support coloring in Windows use:
// 		import "github.com/mattn/go-colorable"
// 		api.DefaultWriter = colorable.NewColorableStdout()
var DefaultWriter io.Writer = os.Stdout
var DefaultErrorWriter io.Writer = os.Stderr

var apiMode = debugCode
var modeName = DebugMode

func init() {
	log.SetFlags(0)

	mode := os.Getenv(ENV_API_MODE)
	if len(mode) == 0 {
		SetMode(DebugMode)
	} else {
		SetMode(mode)
	}
}

func SetMode(value string) {
	switch value {
	case DebugMode:
		apiMode = debugCode
	case ReleaseMode:
		apiMode = releaseCode
	case TestMode:
		apiMode = testCode
	default:
		panic("api mode unknown: " + value)
	}
	modeName = value
}

func Mode() string {
	return modeName
}

// IsDebugging returns true if the framework is running in debug mode.
// Use SetMode(Api.Release) to disable debug mode.
func IsDebugging() bool {
	return apiMode == debugCode
}

func debugPrintRoute(httpMethod, absolutePath string, handlers []Handler) {
	if IsDebugging() {
		nuHandlers := len(handlers)
		handlerName := FunctionName(LastHandler(handlers))
		debugPrint("%-6s %-25s --> %s (%d handlers)\n", httpMethod, absolutePath, handlerName, nuHandlers)
	}
}

func debugPrintLoadTemplate(tmpl *template.Template) {
	if IsDebugging() {
		var buf bytes.Buffer
		for _, tmpl := range tmpl.Templates() {
			buf.WriteString("\t- ")
			buf.WriteString(tmpl.Name())
			buf.WriteString("\n")
		}
		debugPrint("Loaded HTML Templates (%d): \n%s\n", len(tmpl.Templates()), buf.String())
	}
}

func debugPrint(format string, values ...interface{}) {
	if IsDebugging() {
		log.Printf("[Api-debug] "+format, values...)
	}
}

func debugPrintWARNINGNew() {
	debugPrint(`[WARNING] Running in "debug" mode. Switch to "release" mode in production.
 - using env:	export Api_MODE=release
 - using code:	Api.SetMode(Api.ReleaseMode)

`)
}

func debugPrintWARNINGSetHTMLTemplate() {
	debugPrint(`[WARNING] Since SetHTMLTemplate() is NOT thread-safe. It should only be called
at initialization. ie. before any route is registered or the router is listening in a socket:

	router := Api.Default()
	router.SetHTMLTemplate(template) // << good place

`)
}

func debugPrintError(err error) {
	if err != nil {
		debugPrint("[ERROR] %v\n", err)
	}
}
