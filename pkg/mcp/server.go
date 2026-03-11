package mcp

import (
	"net/http"
	"strings"

	"github.com/mark3labs/mcp-go/server"
	"github.com/mimir-aip/mimir-aip-go/pkg/analysis"
	"github.com/mimir-aip/mimir-aip-go/pkg/connectors"
	"github.com/mimir-aip/mimir-aip-go/pkg/digitaltwin"
	"github.com/mimir-aip/mimir-aip-go/pkg/extraction"
	"github.com/mimir-aip/mimir-aip-go/pkg/mlmodel"
	"github.com/mimir-aip/mimir-aip-go/pkg/ontology"
	"github.com/mimir-aip/mimir-aip-go/pkg/pipeline"
	"github.com/mimir-aip/mimir-aip-go/pkg/project"
	"github.com/mimir-aip/mimir-aip-go/pkg/queue"
	"github.com/mimir-aip/mimir-aip-go/pkg/scheduler"
	"github.com/mimir-aip/mimir-aip-go/pkg/storage"
)

// MimirMCPServer exposes Mimir platform capabilities via the Model Context Protocol.
type MimirMCPServer struct {
	projectSvc    *project.Service
	pipelineSvc   *pipeline.Service
	connectorSvc  *connectors.Service
	analysisSvc   *analysis.Service
	mlSvc         *mlmodel.Service
	dtSvc         *digitaltwin.Service
	storageSvc    *storage.Service
	ontologySvc   *ontology.Service
	extractionSvc *extraction.Service
	schedulerSvc  *scheduler.Service
	queue         *queue.Queue
}

// New creates a new MimirMCPServer.
func New(
	projectSvc *project.Service,
	pipelineSvc *pipeline.Service,
	connectorSvc *connectors.Service,
	analysisSvc *analysis.Service,
	mlSvc *mlmodel.Service,
	dtSvc *digitaltwin.Service,
	storageSvc *storage.Service,
	ontologySvc *ontology.Service,
	extractionSvc *extraction.Service,
	schedulerSvc *scheduler.Service,
	q *queue.Queue,
) *MimirMCPServer {
	return &MimirMCPServer{
		projectSvc:    projectSvc,
		pipelineSvc:   pipelineSvc,
		connectorSvc:  connectorSvc,
		analysisSvc:   analysisSvc,
		mlSvc:         mlSvc,
		dtSvc:         dtSvc,
		storageSvc:    storageSvc,
		ontologySvc:   ontologySvc,
		extractionSvc: extractionSvc,
		schedulerSvc:  schedulerSvc,
		queue:         q,
	}
}

// SSEHandler builds and returns an HTTP handler for the MCP SSE transport.
// It serves GET /mcp/sse (SSE stream) and POST /mcp/message.
// Mount it at "/mcp/" in your HTTP mux.
func (m *MimirMCPServer) SSEHandler(baseURL string) http.Handler {
	s := server.NewMCPServer("Mimir AIP", "1.0.0", server.WithToolCapabilities(true))

	registerSystemTools(s, m)
	registerProjectTools(s, m)
	registerPipelineTools(s, m)
	registerConnectorTools(s, m)
	registerAnalysisTools(s, m)
	registerMLModelTools(s, m)
	registerDigitalTwinTools(s, m)
	registerOntologyTools(s, m)
	registerStorageTools(s, m)
	registerTaskTools(s, m)
	registerScheduleTools(s, m)

	// WithBaseURL tells the SSE server its canonical base so it advertises the
	// correct message endpoint to clients.  We append "/mcp" so the advertised
	// message endpoint becomes <baseURL>/mcp/message.
	// http.StripPrefix strips the "/mcp" path prefix before handing requests
	// to the SSE server, so it sees "/sse" and "/message" as expected.
	sseServer := server.NewSSEServer(s, server.WithBaseURL(baseURL+"/mcp"))
	return http.StripPrefix("/mcp", sseServer)
}

// splitCSV splits a comma-separated string into a trimmed slice.
func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
