// API client for Mimir AIP backend
// In production (Docker), use relative paths to avoid CORS issues
// In development, you can set NEXT_PUBLIC_API_URL to http://localhost:8080

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "";

// Generic fetch wrapper with error handling
async function apiFetch<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const url = `${API_BASE_URL}${endpoint}`;
  
  try {
    const response = await fetch(url, {
      ...options,
      headers: {
        "Content-Type": "application/json",
        ...options?.headers,
      },
    });

    if (!response.ok) {
      const errorText = await response.text();
      throw new Error(`API error (${response.status}): ${errorText || response.statusText}`);
    }

    // Handle empty responses (204 No Content, etc.)
    const contentType = response.headers.get("content-type");
    if (contentType && contentType.includes("application/json")) {
      return await response.json();
    }
    
    return {} as T;
  } catch (error) {
    if (error instanceof Error) {
      throw error;
    }
    throw new Error("Unknown API error");
  }
}

// ==================== PIPELINES ====================

export interface Pipeline {
  id: string;
  name: string;
  status?: string;
  steps?: unknown[];
  [key: string]: unknown;
}

/**
 * Get all pipelines
 * GET /api/v1/pipelines
 */
export async function getPipelines(): Promise<Pipeline[]> {
  return apiFetch<Pipeline[]>("/api/v1/pipelines");
}

/**
 * Get a single pipeline by ID
 * GET /api/v1/pipelines/:id
 */
export async function getPipeline(id: string): Promise<Pipeline> {
  return apiFetch<Pipeline>(`/api/v1/pipelines/${id}`);
}

/**
 * Execute a pipeline
 * POST /api/v1/pipelines/execute
 */
export async function executePipeline(id: string, body: Record<string, unknown>): Promise<unknown> {
  return apiFetch("/api/v1/pipelines/execute", {
    method: "POST",
    body: JSON.stringify({
      pipeline_id: id,
      ...body,
    }),
  });
}

/**
 * Create a new pipeline
 * POST /api/v1/pipelines
 */
export async function createPipeline(metadata: Record<string, unknown>, config: Record<string, unknown>): Promise<Pipeline> {
  return apiFetch("/api/v1/pipelines", {
    method: "POST",
    body: JSON.stringify({ metadata, config }),
  });
}

/**
 * Update an existing pipeline
 * PUT /api/v1/pipelines/:id
 */
export async function updatePipeline(id: string, metadata?: Record<string, unknown>, config?: Record<string, unknown>): Promise<Pipeline> {
  return apiFetch(`/api/v1/pipelines/${id}`, {
    method: "PUT",
    body: JSON.stringify({ metadata, config }),
  });
}

/**
 * Delete a pipeline
 * DELETE /api/v1/pipelines/:id
 */
export async function deletePipeline(id: string): Promise<{ message: string; id: string }> {
  return apiFetch(`/api/v1/pipelines/${id}`, {
    method: "DELETE",
  });
}

/**
 * Clone a pipeline
 * POST /api/v1/pipelines/:id/clone
 */
export async function clonePipeline(id: string, name: string): Promise<Pipeline> {
  return apiFetch(`/api/v1/pipelines/${id}/clone`, {
    method: "POST",
    body: JSON.stringify({ name }),
  });
}

/**
 * Validate a pipeline
 * POST /api/v1/pipelines/:id/validate
 */
export async function validatePipeline(id: string): Promise<{ valid: boolean; errors: string[]; pipeline_id: string }> {
  return apiFetch(`/api/v1/pipelines/${id}/validate`, {
    method: "POST",
  });
}

/**
 * Get pipeline execution history
 * GET /api/v1/pipelines/:id/history
 */
export async function getPipelineHistory(id: string): Promise<{ pipeline_id: string; history: ExecutionLog[] }> {
  return apiFetch(`/api/v1/pipelines/${id}/history`);
}

// ==================== JOBS ====================

export interface Job {
  id: string;
  name?: string;
  pipeline?: string;
  cron_expr?: string;
  enabled?: boolean;
  next_run?: string;
  last_run?: string;
  created_at?: string;
  updated_at?: string;
  // Computed properties for UI
  status?: string;
  pipelineId?: string;
  createdAt?: string;
  [key: string]: unknown;
}

/**
 * Get all jobs (scheduler jobs)
 * GET /api/v1/jobs or /api/v1/scheduler/jobs
 */
export async function getJobs(): Promise<Job[]> {
  try {
    // Try /api/v1/scheduler/jobs first (more specific endpoint)
    const jobs = await apiFetch<Job[]>("/api/v1/scheduler/jobs");
    // Map backend fields to frontend expectations and compute status
    return jobs.map(job => ({
      ...job,
      status: job.enabled ? 'enabled' : 'disabled',
      pipelineId: job.pipeline,
      createdAt: job.created_at,
    }));
  } catch {
    // Fallback to /api/v1/jobs
    const jobs = await apiFetch<Job[]>("/api/v1/jobs");
    return jobs.map(job => ({
      ...job,
      status: job.enabled ? 'enabled' : 'disabled',
      pipelineId: job.pipeline,
      createdAt: job.created_at,
    }));
  }
}

/**
 * Get running jobs (filtered from all jobs)
 */
export async function getRunningJobs(): Promise<Job[]> {
  const jobs = await getJobs();
  return jobs.filter((job) => job.enabled === true);
}

/**
 * Get recent jobs (last 10 jobs, sorted by creation date)
 */
export async function getRecentJobs(): Promise<Job[]> {
  const jobs = await getJobs();
  // Sort by created_at descending and take first 10
  return jobs
    .sort((a, b) => {
      const dateA = a.created_at ? new Date(a.created_at).getTime() : 0;
      const dateB = b.created_at ? new Date(b.created_at).getTime() : 0;
      return dateB - dateA;
    })
    .slice(0, 10);
}

/**
 * Get a single scheduled job
 * GET /api/v1/scheduler/jobs/:id
 */
export async function getJob(id: string): Promise<Job> {
  return apiFetch<Job>(`/api/v1/scheduler/jobs/${id}`);
}

/**
 * Create a new scheduled job
 * POST /api/v1/scheduler/jobs
 */
export async function createJob(id: string, name: string, pipeline: string, cronExpr: string): Promise<{ message: string; job_id: string }> {
  return apiFetch("/api/v1/scheduler/jobs", {
    method: "POST",
    body: JSON.stringify({ id, name, pipeline, cron_expr: cronExpr }),
  });
}

/**
 * Update a scheduled job
 * PUT /api/v1/scheduler/jobs/:id
 */
export async function updateJob(id: string, updates: { name?: string; pipeline?: string; cron_expr?: string }): Promise<{ message: string; job_id: string }> {
  return apiFetch(`/api/v1/scheduler/jobs/${id}`, {
    method: "PUT",
    body: JSON.stringify(updates),
  });
}

/**
 * Delete a scheduled job
 * DELETE /api/v1/scheduler/jobs/:id
 */
export async function deleteJob(id: string): Promise<{ message: string; job_id: string }> {
  return apiFetch(`/api/v1/scheduler/jobs/${id}`, {
    method: "DELETE",
  });
}

/**
 * Enable a scheduled job
 * POST /api/v1/scheduler/jobs/:id/enable
 */
export async function enableJob(id: string): Promise<{ message: string; job_id: string }> {
  return apiFetch(`/api/v1/scheduler/jobs/${id}/enable`, {
    method: "POST",
  });
}

/**
 * Disable a scheduled job
 * POST /api/v1/scheduler/jobs/:id/disable
 */
export async function disableJob(id: string): Promise<{ message: string; job_id: string }> {
  return apiFetch(`/api/v1/scheduler/jobs/${id}/disable`, {
    method: "POST",
  });
}

// ==================== EXECUTION LOGS ====================

export interface ExecutionLogEntry {
  timestamp: string;
  level: string;
  message: string;
  step_name?: string;
  plugin_name?: string;
  data?: Record<string, unknown>;
}

export interface ExecutionLog {
  id: string;
  job_id: string;
  pipeline_id: string;
  started_at: string;
  ended_at?: string;
  status: string;
  entries: ExecutionLogEntry[];
}

/**
 * Get execution logs for a specific execution
 * GET /api/v1/logs/executions/:id
 */
export async function getExecutionLog(executionId: string): Promise<ExecutionLog> {
  return apiFetch<ExecutionLog>(`/api/v1/logs/executions/${executionId}`);
}

/**
 * List execution logs with optional filtering
 * GET /api/v1/logs/executions?job_id=&pipeline_id=&limit=
 */
export async function listExecutionLogs(options?: { jobId?: string; pipelineId?: string; limit?: number }): Promise<ExecutionLog[]> {
  const params = new URLSearchParams();
  if (options?.jobId) params.append("job_id", options.jobId);
  if (options?.pipelineId) params.append("pipeline_id", options.pipelineId);
  if (options?.limit) params.append("limit", options.limit.toString());
  
  const query = params.toString() ? `?${params.toString()}` : "";
  return apiFetch<ExecutionLog[]>(`/api/v1/logs/executions${query}`);
}

/**
 * Get logs for a specific pipeline
 * GET /api/v1/pipelines/:id/logs?limit=
 */
export async function getPipelineLogs(pipelineId: string, limit = 50): Promise<{ pipeline_id: string; logs: ExecutionLog[] }> {
  return apiFetch<{ pipeline_id: string; logs: ExecutionLog[] }>(`/api/v1/pipelines/${pipelineId}/logs?limit=${limit}`);
}

/**
 * Get logs for a specific job
 * GET /api/v1/scheduler/jobs/:id/logs?limit=
 */
export async function getJobLogs(jobId: string, limit = 50): Promise<{ job_id: string; logs: ExecutionLog[] }> {
  return apiFetch<{ job_id: string; logs: ExecutionLog[] }>(`/api/v1/scheduler/jobs/${jobId}/logs?limit=${limit}`);
}

// ==================== PLUGINS ====================

export interface Plugin {
  name: string;
  type?: string;
  description?: string;
  [key: string]: unknown;
}

/**
 * Get all plugins
 * GET /api/v1/plugins
 */
export async function getPlugins(): Promise<Plugin[]> {
  return apiFetch<Plugin[]>("/api/v1/plugins");
}

/**
 * Get plugins by type
 * GET /api/v1/plugins/:type
 */
export async function getPluginsByType(type: string): Promise<Plugin[]> {
  return apiFetch<Plugin[]>(`/api/v1/plugins/${type}`);
}

/**
 * Get a specific plugin
 * GET /api/v1/plugins/:type/:name
 */
export async function getPlugin(type: string, name: string): Promise<Plugin> {
  return apiFetch<Plugin>(`/api/v1/plugins/${type}/${name}`);
}

// ==================== CONFIG ====================

export interface Config {
  [key: string]: unknown;
}

/**
 * Get current configuration
 * GET /api/v1/config
 */
export async function getConfig(): Promise<Config> {
  return apiFetch<Config>("/api/v1/config");
}

/**
 * Update configuration
 * PUT /api/v1/config
 */
export async function updateConfig(config: Config): Promise<unknown> {
  return apiFetch("/api/v1/config", {
    method: "PUT",
    body: JSON.stringify(config),
  });
}

/**
 * Reload configuration from file
 * POST /api/v1/config/reload
 */
export async function reloadConfig(): Promise<{ message: string; file: string }> {
  return apiFetch("/api/v1/config/reload", {
    method: "POST",
  });
}

/**
 * Save configuration to file
 * POST /api/v1/config/save
 */
export async function saveConfig(filePath?: string, format?: "yaml" | "json"): Promise<{ message: string; file: string; format: string }> {
  return apiFetch("/api/v1/config/save", {
    method: "POST",
    body: JSON.stringify({ file_path: filePath, format }),
  });
}

// ==================== PERFORMANCE ====================

export interface PerformanceMetrics {
  [key: string]: unknown;
}

/**
 * Get performance metrics
 * GET /api/v1/performance/metrics
 */
export async function getPerformanceMetrics(): Promise<PerformanceMetrics> {
  return apiFetch<PerformanceMetrics>("/api/v1/performance/metrics");
}

/**
 * Get performance statistics (includes system stats)
 * GET /api/v1/performance/stats
 */
export async function getPerformanceStats(): Promise<{
  performance: PerformanceMetrics;
  system: {
    go_version: string;
    num_cpu: number;
    num_goroutines: number;
  };
}> {
  return apiFetch("/api/v1/performance/stats");
}

// ==================== JOB EXECUTION MONITORING ====================

export interface JobExecution {
  id: string;
  pipeline_id?: string;
  status: string;
  started_at?: string;
  completed_at?: string;
  error?: string;
  [key: string]: unknown;
}

/**
 * List all job executions
 * GET /api/v1/jobs
 */
export async function getJobExecutions(): Promise<JobExecution[]> {
  return apiFetch<JobExecution[]>("/api/v1/jobs");
}

/**
 * Get a specific job execution
 * GET /api/v1/jobs/:id
 */
export async function getJobExecution(id: string): Promise<JobExecution> {
  return apiFetch<JobExecution>(`/api/v1/jobs/${id}`);
}

/**
 * Get running job executions
 * GET /api/v1/jobs/running
 */
export async function getRunningJobExecutions(): Promise<JobExecution[]> {
  return apiFetch<JobExecution[]>("/api/v1/jobs/running");
}

/**
 * Get recent job executions
 * GET /api/v1/jobs/recent?limit=10
 */
export async function getRecentJobExecutions(limit = 10): Promise<JobExecution[]> {
  return apiFetch<JobExecution[]>(`/api/v1/jobs/recent?limit=${limit}`);
}

/**
 * Stop a running job execution
 * POST /api/v1/jobs/:id/stop
 */
export async function stopJobExecution(id: string): Promise<{ message: string; id: string }> {
  return apiFetch(`/api/v1/jobs/${id}/stop`, {
    method: "POST",
  });
}

/**
 * Get job statistics
 * GET /api/v1/jobs/statistics
 */
export async function getJobStatistics(): Promise<Record<string, unknown>> {
  return apiFetch("/api/v1/jobs/statistics");
}

/**
 * Export job data
 * GET /api/v1/jobs/export
 */
export async function exportJobs(): Promise<unknown> {
  return apiFetch("/api/v1/jobs/export");
}

// ==================== AUTHENTICATION ====================

export interface User {
  id: string;
  username: string;
  roles: string[];
  active: boolean;
}

export interface LoginResponse {
  token: string;
  user: string;
  roles: string[];
  expires_in: number;
}

/**
 * Login with username and password
 * POST /api/v1/auth/login
 */
export async function login(username: string, password: string): Promise<LoginResponse> {
  return apiFetch("/api/v1/auth/login", {
    method: "POST",
    body: JSON.stringify({ username, password }),
  });
}

/**
 * Refresh authentication token
 * POST /api/v1/auth/refresh
 */
export async function refreshToken(token: string): Promise<{ token: string; expires_in: number }> {
  return apiFetch("/api/v1/auth/refresh", {
    method: "POST",
    body: JSON.stringify({ token }),
  });
}

/**
 * Get current user info
 * GET /api/v1/auth/me
 */
export async function getMe(): Promise<User> {
  return apiFetch<User>("/api/v1/auth/me");
}

/**
 * List all users (admin only)
 * GET /api/v1/auth/users
 */
export async function getUsers(): Promise<{ users: User[] }> {
  return apiFetch<{ users: User[] }>("/api/v1/auth/users");
}

/**
 * Create API key
 * POST /api/v1/auth/apikeys
 */
export async function createAPIKey(name: string): Promise<{
  key: string;
  name: string;
  user_id: string;
  created: string;
}> {
  return apiFetch("/api/v1/auth/apikeys", {
    method: "POST",
    body: JSON.stringify({ name }),
  });
}

// ==================== VISUALIZATION ====================

/**
 * Visualize a pipeline (ASCII art)
 * POST /api/v1/visualize/pipeline
 */
export async function visualizePipeline(pipelineFile: string): Promise<string> {
  const response = await fetch(`${API_BASE_URL}/api/v1/visualize/pipeline`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ pipeline_file: pipelineFile }),
  });
  return response.text();
}

/**
 * Visualize system status (ASCII art)
 * GET /api/v1/visualize/status
 */
export async function visualizeStatus(): Promise<string> {
  const response = await fetch(`${API_BASE_URL}/api/v1/visualize/status`);
  return response.text();
}

/**
 * Visualize scheduler jobs (ASCII art)
 * GET /api/v1/visualize/scheduler
 */
export async function visualizeScheduler(): Promise<string> {
  const response = await fetch(`${API_BASE_URL}/api/v1/visualize/scheduler`);
  return response.text();
}

/**
 * Visualize plugins (ASCII art)
 * GET /api/v1/visualize/plugins
 */
export async function visualizePlugins(): Promise<string> {
  const response = await fetch(`${API_BASE_URL}/api/v1/visualize/plugins`);
  return response.text();
}

// ==================== HEALTH ====================

/**
 * Health check
 * GET /health
 */
export async function getHealth(): Promise<{ status: string }> {
  return apiFetch<{ status: string }>("/health");
}

// ==================== ONTOLOGY ====================

export interface Ontology {
  id: string;
  name: string;
  description?: string;
  version: string;
  file_path: string;
  tdb2_graph: string;
  format: string;
  status: string;
  created_at: string;
  updated_at: string;
  created_by?: string;
  metadata?: string;
}

export interface OntologyUploadRequest {
  name: string;
  description?: string;
  version: string;
  format?: string;
  ontology_data: string;
  created_by?: string;
}

export interface OntologyUploadResponse {
  ontology_id: string;
  ontology_name: string;
  ontology_version: string;
  tdb2_graph: string;
  status: string;
  message?: string;
}

export interface SPARQLQueryRequest {
  query: string;
}

export interface SPARQLQueryResult {
  query_type: string;
  variables: string[];
  bindings: Record<string, unknown>[];
  duration?: number;
  boolean?: boolean;
}

export interface KnowledgeGraphStats {
  total_triples: number;
  total_subjects: number;
  total_predicates: number;
  total_objects: number;
  named_graphs: string[];
  last_updated: string;
  size_bytes: number;
}

/**
 * List all ontologies
 * GET /api/v1/ontology
 */
export async function listOntologies(status?: string): Promise<Ontology[]> {
  const params = status ? `?status=${encodeURIComponent(status)}` : "";
  return apiFetch<Ontology[]>(`/api/v1/ontology${params}`);
}

/**
 * Upload an ontology
 * POST /api/v1/ontology
 */
export async function uploadOntology(request: OntologyUploadRequest): Promise<{ success: boolean; data: OntologyUploadResponse }> {
  return apiFetch("/api/v1/ontology", {
    method: "POST",
    body: JSON.stringify(request),
  });
}

/**
 * Get an ontology by ID
 * GET /api/v1/ontology/:id
 */
export async function getOntology(id: string, includeContent = false): Promise<{ success: boolean; data: { ontology: Ontology; content?: string } }> {
  const params = includeContent ? "?include_content=true" : "";
  return apiFetch(`/api/v1/ontology/${id}${params}`);
}

/**
 * Delete an ontology
 * DELETE /api/v1/ontology/:id
 */
export async function deleteOntology(id: string): Promise<{ success: boolean; data: { ontology_id: string; status: string; message: string } }> {
  return apiFetch(`/api/v1/ontology/${id}`, {
    method: "DELETE",
  });
}

/**
 * Validate an ontology
 * POST /api/v1/ontology/validate
 */
export async function validateOntology(ontologyData: string, format?: string): Promise<{ success: boolean; data: { valid: boolean; errors: unknown[]; warnings: unknown[] } }> {
  return apiFetch("/api/v1/ontology/validate", {
    method: "POST",
    body: JSON.stringify({ ontology_data: ontologyData, format }),
  });
}

/**
 * Get ontology statistics
 * GET /api/v1/ontology/:id/stats
 */
export async function getOntologyStats(id: string): Promise<{ success: boolean; data: { stats: unknown; ontology_name: string } }> {
  return apiFetch(`/api/v1/ontology/${id}/stats`);
}

/**
 * Export an ontology
 * GET /api/v1/ontology/:id/export
 */
export async function exportOntology(id: string, format = "turtle"): Promise<string> {
  const response = await fetch(`${API_BASE_URL}/api/v1/ontology/${id}/export?format=${format}`);
  return response.text();
}

/**
 * Execute a SPARQL query
 * POST /api/v1/kg/query
 */
export async function executeSPARQLQuery(query: string): Promise<{ success: boolean; data: SPARQLQueryResult }> {
  return apiFetch("/api/v1/kg/query", {
    method: "POST",
    body: JSON.stringify({ query }),
  });
}

/**
 * Get knowledge graph statistics
 * GET /api/v1/kg/stats
 */
export async function getKnowledgeGraphStats(): Promise<{ success: boolean; data: KnowledgeGraphStats }> {
  return apiFetch("/api/v1/kg/stats");
}

/**
 * Get subgraph for visualization
 * GET /api/v1/kg/subgraph
 */
export async function getSubgraph(rootUri: string, depth = 1): Promise<{ success: boolean; data: { nodes: unknown[]; edges: unknown[]; stats: { node_count: number; edge_count: number } } }> {
  return apiFetch(`/api/v1/kg/subgraph?root_uri=${encodeURIComponent(rootUri)}&depth=${depth}`);
}

// ==================== ENTITY EXTRACTION ====================

export interface ExtractionJob {
  id: string;
  ontology_id: string;
  job_name: string;
  status: "pending" | "running" | "completed" | "failed";
  extraction_type: "deterministic" | "llm" | "hybrid";
  source_type: string;
  entities_extracted: number;
  triples_generated: number;
  error_message?: string;
  created_at: string;
  started_at?: string;
  completed_at?: string;
}

export interface ExtractedEntity {
  id: number;
  job_id: string;
  entity_uri: string;
  entity_type: string;
  entity_label?: string;
  confidence?: number;
  source_text?: string;
  properties?: Record<string, unknown>;
  created_at: string;
}

export interface ExtractionJobDetails extends ExtractionJob {
  entities: ExtractedEntity[];
}

/**
 * Create a new extraction job
 * POST /api/v1/extraction/jobs
 */
export async function createExtractionJob(data: {
  ontology_id: string;
  job_name?: string;
  source_type: string;
  extraction_type?: "deterministic" | "llm" | "hybrid";
  data: unknown;
}): Promise<{ success: boolean; data: { job_id: string; status: string; entities_extracted: number; triples_generated: number; confidence: number; warnings: string[] } }> {
  return apiFetch("/api/v1/extraction/jobs", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

/**
 * List extraction jobs
 * GET /api/v1/extraction/jobs
 */
export async function listExtractionJobs(params?: {
  ontology_id?: string;
  status?: string;
}): Promise<{ success: boolean; data: { jobs: ExtractionJob[] } }> {
  const searchParams = new URLSearchParams();
  if (params?.ontology_id) searchParams.set("ontology_id", params.ontology_id);
  if (params?.status) searchParams.set("status", params.status);
  
  const queryString = searchParams.toString();
  const endpoint = queryString ? `/api/v1/extraction/jobs?${queryString}` : "/api/v1/extraction/jobs";
  
  return apiFetch(endpoint);
}

/**
 * Get extraction job details
 * GET /api/v1/extraction/jobs/:id
 */
export async function getExtractionJob(id: string): Promise<{ success: boolean; data: { job: ExtractionJob; entities: ExtractedEntity[] } }> {
  return apiFetch(`/api/v1/extraction/jobs/${id}`);
}

// ==================== NATURAL LANGUAGE QUERIES ====================

export interface NLQueryResult {
  question: string;
  sparql_query: string;
  explanation: string;
  results: SPARQLQueryResult;
}

/**
 * Execute a natural language query
 * POST /api/v1/kg/nl-query
 */
export async function executeNLQuery(question: string, ontologyId?: string): Promise<{ success: boolean; data: NLQueryResult }> {
  return apiFetch("/api/v1/kg/nl-query", {
    method: "POST",
    body: JSON.stringify({
      question,
      ontology_id: ontologyId,
    }),
  });
}

// ==================== ONTOLOGY VERSIONING ====================

export interface OntologyVersion {
  id: number;
  ontology_id: string;
  version: string;
  previous_version?: string;
  changelog: string;
  migration_strategy?: string;
  created_at: string;
  created_by?: string;
}

export interface OntologyChange {
  id: number;
  version_id: number;
  change_type: string;
  entity_type: string;
  entity_uri: string;
  old_value?: string;
  new_value?: string;
  description?: string;
  created_at: string;
}

export interface VersionDiff {
  from_version: string;
  to_version: string;
  changes: OntologyChange[];
  summary: {
    classes_added: number;
    classes_removed: number;
    classes_modified: number;
    properties_added: number;
    properties_removed: number;
    properties_modified: number;
    total_changes: number;
  };
}

/**
 * Create a new version of an ontology
 * POST /api/v1/ontology/:id/versions
 */
export async function createOntologyVersion(
  ontologyId: string,
  version: string,
  changelog: string,
  createdBy?: string
): Promise<{ success: boolean; data: OntologyVersion }> {
  return apiFetch(`/api/v1/ontology/${ontologyId}/versions`, {
    method: "POST",
    body: JSON.stringify({
      version,
      changelog,
      created_by: createdBy,
    }),
  });
}

/**
 * List all versions of an ontology
 * GET /api/v1/ontology/:id/versions
 */
export async function listOntologyVersions(ontologyId: string): Promise<{ success: boolean; data: OntologyVersion[] }> {
  return apiFetch(`/api/v1/ontology/${ontologyId}/versions`);
}

/**
 * Get a specific version with changes
 * GET /api/v1/ontology/:id/versions/:vid
 */
export async function getOntologyVersion(
  ontologyId: string,
  versionId: number
): Promise<{ success: boolean; data: { version: OntologyVersion; changes: OntologyChange[] } }> {
  return apiFetch(`/api/v1/ontology/${ontologyId}/versions/${versionId}`);
}

/**
 * Compare two versions
 * GET /api/v1/ontology/:id/versions/compare?v1=...&v2=...
 */
export async function compareOntologyVersions(
  ontologyId: string,
  v1: string,
  v2: string
): Promise<{ success: boolean; data: VersionDiff }> {
  return apiFetch(`/api/v1/ontology/${ontologyId}/versions/compare?v1=${encodeURIComponent(v1)}&v2=${encodeURIComponent(v2)}`);
}

/**
 * Delete a version
 * DELETE /api/v1/ontology/:id/versions/:vid
 */
export async function deleteOntologyVersion(ontologyId: string, versionId: number): Promise<void> {
  return apiFetch(`/api/v1/ontology/${ontologyId}/versions/${versionId}`, {
    method: "DELETE",
  });
}

// ========================================
// Drift Detection Types & Functions
// ========================================

export interface DriftDetection {
  id: number;
  ontology_id: string;
  detection_type: string;
  data_source: string;
  suggestions_generated: number;
  status: "running" | "completed" | "failed";
  started_at: string;
  completed_at?: string;
  error_message?: string;
}

export interface DriftDetectionRequest {
  source: "extraction_job" | "data" | "knowledge_graph";
  job_id?: string;
  data?: any;
  data_source?: string;
}

/**
 * Trigger drift detection for an ontology
 * POST /api/v1/ontology/:id/drift/detect
 */
export async function triggerDriftDetection(
  ontologyId: string,
  request: DriftDetectionRequest
): Promise<{ success: boolean; data: { message: string; suggestions_generated: number; ontology_id: string } }> {
  return apiFetch(`/api/v1/ontology/${ontologyId}/drift/detect`, {
    method: "POST",
    body: JSON.stringify(request),
  });
}

/**
 * Get drift detection history
 * GET /api/v1/ontology/:id/drift/history
 */
export async function getDriftHistory(ontologyId: string): Promise<{ success: boolean; data: DriftDetection[] }> {
  return apiFetch(`/api/v1/ontology/${ontologyId}/drift/history`);
}

// ========================================
// Suggestion Management Types & Functions
// ========================================

export interface OntologySuggestion {
  id: number;
  ontology_id: string;
  suggestion_type: "add_class" | "add_property" | "modify_class" | "modify_property" | "deprecate";
  entity_type: string;
  entity_uri?: string;
  confidence: number;
  reasoning: string;
  status: "pending" | "approved" | "rejected" | "applied";
  risk_level: "low" | "medium" | "high" | "critical";
  created_at: string;
  reviewed_at?: string;
  reviewed_by?: string;
  review_decision?: string;
  review_notes?: string;
}

export interface SuggestionReview {
  reviewed_by: string;
  review_notes?: string;
}

export interface SuggestionSummary {
  ontology_id: string;
  summary: string;
}

/**
 * List suggestions for an ontology
 * GET /api/v1/ontology/:id/suggestions
 */
export async function listSuggestions(
  ontologyId: string,
  status?: string
): Promise<{ success: boolean; data: OntologySuggestion[] }> {
  const queryString = status ? `?status=${encodeURIComponent(status)}` : "";
  const endpoint = queryString ? `/api/v1/ontology/${ontologyId}/suggestions${queryString}` : `/api/v1/ontology/${ontologyId}/suggestions`;
  return apiFetch(endpoint);
}

/**
 * Get a specific suggestion
 * GET /api/v1/ontology/:id/suggestions/:sid
 */
export async function getSuggestion(
  ontologyId: string,
  suggestionId: number
): Promise<{ success: boolean; data: OntologySuggestion }> {
  return apiFetch(`/api/v1/ontology/${ontologyId}/suggestions/${suggestionId}`);
}

/**
 * Approve a suggestion
 * POST /api/v1/ontology/:id/suggestions/:sid/approve
 */
export async function approveSuggestion(
  ontologyId: string,
  suggestionId: number,
  review: SuggestionReview
): Promise<{ success: boolean; data: { message: string; suggestion_id: number; status: string } }> {
  return apiFetch(`/api/v1/ontology/${ontologyId}/suggestions/${suggestionId}/approve`, {
    method: "POST",
    body: JSON.stringify(review),
  });
}

/**
 * Reject a suggestion
 * POST /api/v1/ontology/:id/suggestions/:sid/reject
 */
export async function rejectSuggestion(
  ontologyId: string,
  suggestionId: number,
  review: SuggestionReview
): Promise<{ success: boolean; data: { message: string; suggestion_id: number; status: string } }> {
  return apiFetch(`/api/v1/ontology/${ontologyId}/suggestions/${suggestionId}/reject`, {
    method: "POST",
    body: JSON.stringify(review),
  });
}

/**
 * Apply an approved suggestion to the ontology
 * POST /api/v1/ontology/:id/suggestions/:sid/apply
 */
export async function applySuggestion(
  ontologyId: string,
  suggestionId: number
): Promise<{ success: boolean; data: { message: string; suggestion_id: number; status: string } }> {
  return apiFetch(`/api/v1/ontology/${ontologyId}/suggestions/${suggestionId}/apply`, {
    method: "POST",
  });
}

/**
 * Get suggestion summary
 * GET /api/v1/ontology/:id/suggestions/summary
 */
export async function getSuggestionSummary(ontologyId: string): Promise<{ success: boolean; data: SuggestionSummary }> {
  return apiFetch(`/api/v1/ontology/${ontologyId}/suggestions/summary`);
}
