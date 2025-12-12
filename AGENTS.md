# Agent Guidelines for Mimir-AIP-Go

## Build & Test Commands

**Backend (Go):**
- Build: `go build -o mimir-aip-server .`
- Test all: `go test ./...`
- Single test: `go test -run TestName ./tests` or `go test -run TestName ./path/to/package`
- Coverage: `go test -cover ./...`
- Benchmarks: `go test -bench=. ./...`
- Race detection: `go test -race ./...`

**Frontend (Next.js):**
- Dev: `cd mimir-aip-frontend && npm run dev`
- Build: `npm run build`
- Lint: `npm run lint`
- Test: `npm test` (Vitest)
- E2E: `npm run test:e2e` (Playwright)

**Docker (Unified Deployment):**
- Build unified container (backend + frontend): `./build-unified.sh`
- Start unified services: `docker-compose -f docker-compose.unified.yml up`
- The unified Docker image (244MB) serves both Go API and Next.js UI on port 8080 with reverse proxy
- Backend serves API on `/api/v1/*`, frontend serves UI on `/`, with proper CSP headers

## Code Style Guidelines

### Go
- **Imports**: Group stdlib, external, internal; use full module path `github.com/Mimir-AIP/Mimir-AIP-Go/...`
- **Naming**: PascalCase for exports, camelCase for private; descriptive names (e.g., `ExecutionLogger`, `handleUpdateJob`)
- **Error Handling**: Always check errors; use `fmt.Errorf("context: %w", err)` for wrapping; return early on errors
- **Types**: Use explicit types; prefer `any` over `interface{}`; define structs with JSON tags for API responses
- **Concurrency**: Use `sync.Mutex` for shared state; prefer channels for goroutine communication; always use context for cancellation
- **Logging**: Use `utils.GetLogger()` singleton; structured logging with component tags; levels: DEBUG, INFO, WARN, ERROR, FATAL

### TypeScript/React
- **Imports**: Group React, external libs, internal (`@/lib`, `@/components`)
- **Types**: Explicit TypeScript types; export interfaces from `@/lib/api.ts`; avoid `any`
- **Naming**: PascalCase for components; camelCase for functions/variables; prefix handlers with `handle` (e.g., `handleDelete`)
- **State**: Use React hooks; useState for local, useEffect for side effects
- **Error Handling**: Try-catch with toast notifications; always set loading states
- **Components**: Functional components with TypeScript; extract reusable UI to `/components`

### General Conventions
- **Comments**: Document exported functions/types; explain "why" not "what"
- **File Organization**: One component per file; group related functionality
- **API Design**: RESTful endpoints; consistent response format `{message, data}` or `{error}`
- **Testing**: Table-driven tests (Go); unit tests for components (Vitest); use `testify/assert` for Go assertions
