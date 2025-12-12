# Integration Test Suite

This directory contains comprehensive integration tests for the Mimir-AIP-Go project. These tests verify that complex components work together correctly in end-to-end workflows.

## Test Files

### 1. `integration_http_api_test.go` - HTTP API Integration Tests
Tests complete HTTP API workflows including:
- **Complete Pipeline Execution Flow**: End-to-end API request/response cycles
- **Error Handling**: Invalid requests, non-existent endpoints, method not allowed
- **Concurrent Request Handling**: Tests system behavior under load (50 concurrent requests)
- **Middleware Chain Integration**: CORS, API versioning, security headers
- **Performance Metrics**: Response time tracking and analysis
- **Authentication**: Public endpoint access patterns

**Key Test Functions:**
- `TestHTTPAPICompleteWorkflow()` - Full workflow validation
- `TestHTTPAPIErrorHandling()` - Error scenarios
- `TestHTTPAPIConcurrentRequests()` - Concurrency testing
- `TestHTTPAPIMiddlewareChain()` - Middleware integration
- `TestHTTPAPIResponseTiming()` - Performance characteristics

### 2. `integration_pipeline_test.go` - Pipeline Integration Tests
Tests multi-step pipeline execution with real plugins:
- **Sequential Step Execution**: Multiple steps with data flow
- **Context Passing**: Data preservation and passing between steps
- **Error Handling**: Pipeline failure scenarios and recovery
- **Performance Testing**: Execution timing and consistency
- **Concurrent Execution**: Multiple pipelines running simultaneously
- **Complex Workflows**: Real-world data aggregation scenarios

**Key Test Functions:**
- `TestPipelineMultiStepExecution()` - Multi-step workflows
- `TestPipelineContextPassing()` - Context management
- `TestPipelineErrorHandling()` - Error scenarios
- `TestPipelinePerformance()` - Performance analysis
- `TestPipelineConcurrentExecution()` - Concurrent pipeline tests
- `TestPipelineComplexWorkflow()` - Complex real-world scenarios

### 3. `integration_mcp_server_test.go` - MCP Server Integration Tests
Tests MCP (Model Context Protocol) server functionality:
- **Tool Discovery**: Plugin enumeration and metadata
- **Tool Execution**: Actual plugin invocation via MCP
- **Plugin Registry Integration**: Real plugin registration and access
- **End-to-End Workflows**: Complete MCP client interaction
- **Concurrent Tool Execution**: Multiple simultaneous tool calls
- **Error Recovery**: Failure handling and system stability
- **Response Format Compliance**: MCP format validation

**Key Test Functions:**
- `TestMCPServerToolDiscovery()` - Tool discovery endpoint
- `TestMCPServerToolExecution()` - Tool execution scenarios
- `TestMCPServerPluginRegistry()` - Plugin integration
- `TestMCPServerEndToEnd()` - Complete workflows
- `TestMCPServerConcurrentToolExecution()` - Concurrency tests
- `TestMCPServerErrorRecovery()` - Error handling

### 4. `integration_scheduler_test.go` - Scheduler Integration Tests
Tests cron-based job scheduling:
- **Job Management**: Add, list, enable, disable, remove jobs
- **Cron Expression Parsing**: Various cron formats
- **Job Execution Tracking**: Monitoring job runs
- **Concurrent Operations**: Thread-safe job management
- **Lifecycle Management**: Start, stop, graceful shutdown
- **Real Pipeline Integration**: Actual scheduled pipeline execution

**Key Test Functions:**
- `TestSchedulerIntegration()` - Basic job management
- `TestSchedulerJobExecution()` - Execution tracking
- `TestSchedulerCronExpressions()` - Cron parsing validation
- `TestSchedulerConcurrency()` - Concurrent operations
- `TestSchedulerStartStop()` - Lifecycle management
- `TestSchedulerGracefulShutdown()` - Shutdown behavior
- `TestSchedulerWithRealPipeline()` - Real execution tests

### 5. `integration_storage_test.go` - Storage Integration Tests
Located in `pipelines/Storage/storage_integration_test.go`, tests vector storage:
- **CRUD Operations**: Create, read, update, delete documents
- **Query Operations**: Semantic search and similarity queries
- **Batch Operations**: Bulk document processing
- **Collection Management**: Create, list, manage collections
- **Error Handling**: Invalid operations and recovery
- **Embedding Services**: OpenAI, Ollama integration
- **Performance Benchmarking**: Storage operation benchmarks

## Running the Tests

### Run All Integration Tests
```bash
go test ./tests/... -v
```

### Run Specific Test Suite
```bash
# HTTP API tests
go test ./tests/... -v -run TestHTTPAPI

# Pipeline tests
go test ./tests/... -v -run TestPipeline

# MCP Server tests
go test ./tests/... -v -run TestMCPServer

# Scheduler tests
go test ./tests/... -v -run TestScheduler

# Storage tests
go test ./pipelines/Storage/... -v -run TestStorage
```

### Run Tests with Coverage
```bash
go test ./tests/... -v -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Run Only Short Tests
```bash
go test ./tests/... -v -short
```

### Run Specific Test
```bash
go test ./tests/... -v -run TestHTTPAPICompleteWorkflow
```

## Test Environment

### Prerequisites
- Go 1.21 or higher
- Internet connection (for external API calls in tests)
- Optional: OpenAI API key for embedding tests (set `OPENAI_API_KEY`)
- Optional: Ollama running locally for embedding tests

### Test Data
Integration tests create temporary directories and files:
- Pipeline configurations (YAML)
- Test data files
- Temporary storage backends

All temporary data is automatically cleaned up after tests complete.

## Test Coverage

The integration tests cover:

### HTTP API Layer
- ✅ Request/response cycles
- ✅ Middleware chain execution
- ✅ Error handling and recovery
- ✅ CORS and security headers
- ✅ API versioning
- ✅ Concurrent request handling
- ✅ Performance metrics

### Pipeline Execution
- ✅ Multi-step execution
- ✅ Context passing between steps
- ✅ Plugin integration
- ✅ Error propagation
- ✅ Concurrent pipeline execution
- ✅ Performance characteristics
- ✅ Complex workflows

### MCP Server
- ✅ Tool discovery
- ✅ Tool execution
- ✅ Plugin registry integration
- ✅ Concurrent tool invocation
- ✅ Error handling
- ✅ Response format compliance

### Scheduler
- ✅ Job lifecycle management
- ✅ Cron expression parsing
- ✅ Job execution tracking
- ✅ Concurrent operations
- ✅ Graceful shutdown
- ✅ Thread safety

### Storage Backend
- ✅ Document CRUD operations
- ✅ Semantic search
- ✅ Batch operations
- ✅ Collection management
- ✅ Embedding services
- ✅ Error handling

## Performance Characteristics

Integration tests measure and validate:

### HTTP API
- Average response time: < 100ms for health checks
- Max response time: < 500ms
- Concurrent request success rate: > 95%

### Pipeline Execution
- Average execution time: < 5s for test pipelines
- Execution time variance: < 2s
- Concurrent execution success rate: > 80%

### MCP Server
- Tool discovery: < 50ms
- Tool execution: varies by plugin
- Concurrent tool execution success rate: > 70%

### Scheduler
- Graceful shutdown: < 35s
- Job management operations: < 10ms

## Best Practices

1. **Test Isolation**: Each test creates its own temporary environment
2. **Cleanup**: All tests clean up resources in defer statements
3. **Error Checking**: All errors are checked with `require.NoError()`
4. **Assertions**: Use `assert` for non-critical checks, `require` for critical ones
5. **Logging**: Tests log performance statistics and diagnostic information
6. **Concurrency**: Tests verify thread-safety of critical components
7. **Real Data**: Tests use real HTTP endpoints when possible for authenticity

## Common Issues

### Test Failures Due to Network

If tests fail due to network issues:
- Check internet connectivity
- Verify `httpbin.org` is accessible
- Consider using mock HTTP responses for offline testing

### Test Timeouts

If tests timeout:
- Increase timeout values in specific tests
- Check system load and resource availability
- Use `-timeout` flag: `go test -timeout 5m`

### Race Conditions

To detect race conditions:
```bash
go test ./tests/... -race -v
```

## Contributing

When adding new integration tests:

1. Follow the existing naming convention: `Test[Component][Scenario]()`
2. Add comprehensive documentation comments
3. Clean up all resources in defer statements
4. Test both success and failure scenarios
5. Include performance assertions where appropriate
6. Add concurrency tests for thread-safe components
7. Update this README with new test descriptions

## Test Metrics

Track these metrics when running integration tests:

- **Total Tests**: Number of test functions
- **Coverage**: Percentage of code covered
- **Duration**: Total time to run all tests
- **Success Rate**: Percentage of passing tests
- **Concurrency**: Number of concurrent operations tested
- **Performance**: Response times and throughput

## Continuous Integration

Integration tests are designed to run in CI/CD pipelines:

```yaml
# Example GitHub Actions workflow
- name: Run Integration Tests
  run: |
    go test ./tests/... -v -coverprofile=coverage.out
    go test ./pipelines/Storage/... -v
```

## Future Enhancements

Planned improvements to the integration test suite:

- [ ] Add chaos engineering tests (random failures)
- [ ] Add load testing with configurable scenarios
- [ ] Add database integration tests
- [ ] Add authentication/authorization integration tests
- [ ] Add webhook integration tests
- [ ] Add metrics and tracing validation
- [ ] Add multi-region deployment tests
- [ ] Add backup/restore integration tests

## Support

For issues with integration tests:
1. Check test output for detailed error messages
2. Review logs in temporary test directories
3. Run with `-v` flag for verbose output
4. Check GitHub issues for known problems
5. Create new issue with test failure details
