import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import * as api from '../api'

// Mock fetch globally
const mockFetch = vi.fn()
global.fetch = mockFetch

describe('API Client', () => {
  beforeEach(() => {
    mockFetch.mockClear()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('Pipeline API', () => {
    describe('getPipelines', () => {
      it('should fetch all pipelines successfully', async () => {
        const mockPipelines = [
          { id: '1', name: 'Pipeline 1', status: 'active' },
          { id: '2', name: 'Pipeline 2', status: 'inactive' },
        ]

        mockFetch.mockResolvedValueOnce({
          ok: true,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => mockPipelines,
        })

        const result = await api.getPipelines()

        expect(mockFetch).toHaveBeenCalledWith(
          'http://localhost:8080/api/v1/pipelines',
          expect.objectContaining({
            headers: expect.objectContaining({
              'Content-Type': 'application/json',
            }),
          })
        )
        expect(result).toEqual(mockPipelines)
      })

      it('should handle API errors', async () => {
        mockFetch.mockResolvedValueOnce({
          ok: false,
          status: 500,
          statusText: 'Internal Server Error',
          text: async () => 'Server error',
        })

        await expect(api.getPipelines()).rejects.toThrow('API error (500): Server error')
      })
    })

    describe('getPipeline', () => {
      it('should fetch a single pipeline by ID', async () => {
        const mockPipeline = { id: 'test-1', name: 'Test Pipeline', status: 'active' }

        mockFetch.mockResolvedValueOnce({
          ok: true,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => mockPipeline,
        })

        const result = await api.getPipeline('test-1')

        expect(mockFetch).toHaveBeenCalledWith(
          'http://localhost:8080/api/v1/pipelines/test-1',
          expect.any(Object)
        )
        expect(result).toEqual(mockPipeline)
      })
    })

    describe('executePipeline', () => {
      it('should execute a pipeline with input data', async () => {
        const mockResponse = { execution_id: 'exec-123', status: 'running' }
        const inputData = { key: 'value' }

        mockFetch.mockResolvedValueOnce({
          ok: true,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => mockResponse,
        })

        const result = await api.executePipeline('pipeline-1', inputData)

        expect(mockFetch).toHaveBeenCalledWith(
          'http://localhost:8080/api/v1/pipelines/execute',
          expect.objectContaining({
            method: 'POST',
            body: JSON.stringify({ pipeline_id: 'pipeline-1', ...inputData }),
          })
        )
        expect(result).toEqual(mockResponse)
      })
    })

    describe('createPipeline', () => {
      it('should create a new pipeline', async () => {
        const metadata = { name: 'New Pipeline' }
        const config = { steps: [] }
        const mockResponse = { id: 'new-1', ...metadata, ...config }

        mockFetch.mockResolvedValueOnce({
          ok: true,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => mockResponse,
        })

        const result = await api.createPipeline(metadata, config)

        expect(mockFetch).toHaveBeenCalledWith(
          'http://localhost:8080/api/v1/pipelines',
          expect.objectContaining({
            method: 'POST',
            body: JSON.stringify({ metadata, config }),
          })
        )
        expect(result).toEqual(mockResponse)
      })
    })

    describe('updatePipeline', () => {
      it('should update an existing pipeline', async () => {
        const metadata = { name: 'Updated Pipeline' }
        const config = { steps: [{ type: 'input' }] }
        const mockResponse = { id: 'update-1', ...metadata, ...config }

        mockFetch.mockResolvedValueOnce({
          ok: true,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => mockResponse,
        })

        const result = await api.updatePipeline('update-1', metadata, config)

        expect(mockFetch).toHaveBeenCalledWith(
          'http://localhost:8080/api/v1/pipelines/update-1',
          expect.objectContaining({
            method: 'PUT',
            body: JSON.stringify({ metadata, config }),
          })
        )
        expect(result).toEqual(mockResponse)
      })
    })

    describe('deletePipeline', () => {
      it('should delete a pipeline', async () => {
        mockFetch.mockResolvedValueOnce({
          ok: true,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => ({}),
        })

        await api.deletePipeline('delete-1')

        expect(mockFetch).toHaveBeenCalledWith(
          'http://localhost:8080/api/v1/pipelines/delete-1',
          expect.objectContaining({
            method: 'DELETE',
          })
        )
      })
    })

    describe('clonePipeline', () => {
      it('should clone a pipeline', async () => {
        const mockResponse = { id: 'cloned-1', name: 'Cloned Pipeline' }

        mockFetch.mockResolvedValueOnce({
          ok: true,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => mockResponse,
        })

        const result = await api.clonePipeline('original-1', 'Cloned Pipeline')

        expect(mockFetch).toHaveBeenCalledWith(
          'http://localhost:8080/api/v1/pipelines/original-1/clone',
          expect.objectContaining({
            method: 'POST',
            body: JSON.stringify({ name: 'Cloned Pipeline' }),
          })
        )
        expect(result).toEqual(mockResponse)
      })
    })

    describe('validatePipeline', () => {
      it('should validate a pipeline configuration', async () => {
        const pipelineId = 'pipeline-1'
        const mockResponse = { valid: true, errors: [], pipeline_id: pipelineId }

        mockFetch.mockResolvedValueOnce({
          ok: true,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => mockResponse,
        })

        const result = await api.validatePipeline(pipelineId)

        expect(mockFetch).toHaveBeenCalledWith(
          'http://localhost:8080/api/v1/pipelines/pipeline-1/validate',
          expect.objectContaining({
            method: 'POST',
          })
        )
        expect(result).toEqual(mockResponse)
      })
    })

    describe('getPipelineHistory', () => {
      it('should fetch pipeline execution history', async () => {
        const mockHistory = [
          { id: 'exec-1', timestamp: '2023-01-01', status: 'completed' },
          { id: 'exec-2', timestamp: '2023-01-02', status: 'running' },
        ]

        mockFetch.mockResolvedValueOnce({
          ok: true,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => mockHistory,
        })

        const result = await api.getPipelineHistory('pipeline-1')

        expect(mockFetch).toHaveBeenCalledWith(
          'http://localhost:8080/api/v1/pipelines/pipeline-1/history',
          expect.any(Object)
        )
        expect(result).toEqual(mockHistory)
      })
    })
  })

  describe('Job API', () => {
    describe('getJobs', () => {
      it('should fetch all scheduled jobs', async () => {
        const mockJobs = [
          { id: 'job-1', name: 'Daily Job', status: 'enabled' },
          { id: 'job-2', name: 'Weekly Job', status: 'disabled' },
        ]

        mockFetch.mockResolvedValueOnce({
          ok: true,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => mockJobs,
        })

        const result = await api.getJobs()

        expect(mockFetch).toHaveBeenCalledWith(
          'http://localhost:8080/api/v1/scheduler/jobs',
          expect.any(Object)
        )
        expect(result).toEqual(mockJobs)
      })
    })

    describe('getJob', () => {
      it('should fetch a single job by ID', async () => {
        const mockJob = { id: 'job-1', name: 'Test Job', status: 'enabled' }

        mockFetch.mockResolvedValueOnce({
          ok: true,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => mockJob,
        })

        const result = await api.getJob('job-1')

        expect(mockFetch).toHaveBeenCalledWith(
          'http://localhost:8080/api/v1/scheduler/jobs/job-1',
          expect.any(Object)
        )
        expect(result).toEqual(mockJob)
      })
    })

    describe('createJob', () => {
      it('should create a new scheduled job', async () => {
        const mockResponse = { message: 'Job created', job_id: 'new-job-1' }

        mockFetch.mockResolvedValueOnce({
          ok: true,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => mockResponse,
        })

        const result = await api.createJob('job-1', 'New Job', 'pipeline-1', '0 0 * * *')

        expect(mockFetch).toHaveBeenCalledWith(
          'http://localhost:8080/api/v1/scheduler/jobs',
          expect.objectContaining({
            method: 'POST',
            body: JSON.stringify({
              id: 'job-1',
              name: 'New Job',
              pipeline: 'pipeline-1',
              cron_expr: '0 0 * * *',
            }),
          })
        )
        expect(result).toEqual(mockResponse)
      })
    })

    describe('deleteJob', () => {
      it('should delete a job', async () => {
        mockFetch.mockResolvedValueOnce({
          ok: true,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => ({}),
        })

        await api.deleteJob('delete-job-1')

        expect(mockFetch).toHaveBeenCalledWith(
          'http://localhost:8080/api/v1/scheduler/jobs/delete-job-1',
          expect.objectContaining({
            method: 'DELETE',
          })
        )
      })
    })

    describe('enableJob', () => {
      it('should enable a job', async () => {
        mockFetch.mockResolvedValueOnce({
          ok: true,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => ({}),
        })

        await api.enableJob('enable-job-1')

        expect(mockFetch).toHaveBeenCalledWith(
          'http://localhost:8080/api/v1/scheduler/jobs/enable-job-1/enable',
          expect.objectContaining({
            method: 'POST',
          })
        )
      })
    })

    describe('disableJob', () => {
      it('should disable a job', async () => {
        mockFetch.mockResolvedValueOnce({
          ok: true,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => ({}),
        })

        await api.disableJob('disable-job-1')

        expect(mockFetch).toHaveBeenCalledWith(
          'http://localhost:8080/api/v1/scheduler/jobs/disable-job-1/disable',
          expect.objectContaining({
            method: 'POST',
          })
        )
      })
    })
  })

  describe('Job Execution API', () => {
    describe('getJobExecutions', () => {
      it('should fetch all job executions', async () => {
        const mockExecutions = [
          { id: 'exec-1', job_id: 'job-1', status: 'completed' },
          { id: 'exec-2', job_id: 'job-2', status: 'running' },
        ]

        mockFetch.mockResolvedValueOnce({
          ok: true,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => mockExecutions,
        })

        const result = await api.getJobExecutions()

        expect(mockFetch).toHaveBeenCalledWith(
          'http://localhost:8080/api/v1/jobs',
          expect.any(Object)
        )
        expect(result).toEqual(mockExecutions)
      })
    })

    describe('getJobExecution', () => {
      it('should fetch a single job execution', async () => {
        const mockExecution = { id: 'exec-1', job_id: 'job-1', status: 'completed' }

        mockFetch.mockResolvedValueOnce({
          ok: true,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => mockExecution,
        })

        const result = await api.getJobExecution('exec-1')

        expect(mockFetch).toHaveBeenCalledWith(
          'http://localhost:8080/api/v1/jobs/exec-1',
          expect.any(Object)
        )
        expect(result).toEqual(mockExecution)
      })
    })

    describe('stopJobExecution', () => {
      it('should stop a running job execution', async () => {
        mockFetch.mockResolvedValueOnce({
          ok: true,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => ({}),
        })

        await api.stopJobExecution('stop-exec-1')

        expect(mockFetch).toHaveBeenCalledWith(
          'http://localhost:8080/api/v1/jobs/stop-exec-1/stop',
          expect.objectContaining({
            method: 'POST',
          })
        )
      })
    })

    describe('getJobStatistics', () => {
      it('should fetch job statistics', async () => {
        const mockStats = {
          total_jobs: 10,
          active_jobs: 5,
          total_executions: 100,
          success_rate: 0.95,
        }

        mockFetch.mockResolvedValueOnce({
          ok: true,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => mockStats,
        })

        const result = await api.getJobStatistics()

        expect(mockFetch).toHaveBeenCalledWith(
          'http://localhost:8080/api/v1/jobs/statistics',
          expect.any(Object)
        )
        expect(result).toEqual(mockStats)
      })
    })
  })

  describe('Plugin API', () => {
    describe('getPlugins', () => {
      it('should fetch all plugins', async () => {
        const mockPlugins = [
          { name: 'Plugin 1', type: 'input', version: '1.0' },
          { name: 'Plugin 2', type: 'output', version: '2.0' },
        ]

        mockFetch.mockResolvedValueOnce({
          ok: true,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => mockPlugins,
        })

        const result = await api.getPlugins()

        expect(mockFetch).toHaveBeenCalledWith(
          'http://localhost:8080/api/v1/plugins',
          expect.any(Object)
        )
        expect(result).toEqual(mockPlugins)
      })
    })

    describe('getPluginsByType', () => {
      it('should fetch plugins by type', async () => {
        const mockPlugins = [{ name: 'Input Plugin', type: 'input', version: '1.0' }]

        mockFetch.mockResolvedValueOnce({
          ok: true,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => mockPlugins,
        })

        const result = await api.getPluginsByType('input')

        expect(mockFetch).toHaveBeenCalledWith(
          'http://localhost:8080/api/v1/plugins/input',
          expect.any(Object)
        )
        expect(result).toEqual(mockPlugins)
      })
    })

    describe('getPlugin', () => {
      it('should fetch a single plugin by type and name', async () => {
        const mockPlugin = { name: 'Test Plugin', type: 'input', version: '1.0' }

        mockFetch.mockResolvedValueOnce({
          ok: true,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => mockPlugin,
        })

        const result = await api.getPlugin('input', 'Test Plugin')

        expect(mockFetch).toHaveBeenCalledWith(
          'http://localhost:8080/api/v1/plugins/input/Test Plugin',
          expect.any(Object)
        )
        expect(result).toEqual(mockPlugin)
      })
    })
  })

  describe('Config API', () => {
    describe('getConfig', () => {
      it('should fetch configuration', async () => {
        const mockConfig = { key: 'value', nested: { prop: 'data' } }

        mockFetch.mockResolvedValueOnce({
          ok: true,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => mockConfig,
        })

        const result = await api.getConfig()

        expect(mockFetch).toHaveBeenCalledWith(
          'http://localhost:8080/api/v1/config',
          expect.any(Object)
        )
        expect(result).toEqual(mockConfig)
      })
    })

    describe('saveConfig', () => {
      it('should save configuration', async () => {
        mockFetch.mockResolvedValueOnce({
          ok: true,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => ({ message: 'Config saved', file: 'config.yaml', format: 'yaml' }),
        })

        await api.saveConfig('config.yaml', 'yaml')

        expect(mockFetch).toHaveBeenCalledWith(
          'http://localhost:8080/api/v1/config/save',
          expect.objectContaining({
            method: 'POST',
            body: JSON.stringify({ file_path: 'config.yaml', format: 'yaml' }),
          })
        )
      })
    })

    describe('reloadConfig', () => {
      it('should reload configuration from file', async () => {
        mockFetch.mockResolvedValueOnce({
          ok: true,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => ({}),
        })

        await api.reloadConfig()

        expect(mockFetch).toHaveBeenCalledWith(
          'http://localhost:8080/api/v1/config/reload',
          expect.objectContaining({
            method: 'POST',
          })
        )
      })
    })
  })

  describe('Performance API', () => {
    describe('getPerformanceStats', () => {
      it('should fetch performance statistics', async () => {
        const mockStats = {
          performance: {
            cpu_usage: 45.2,
            memory_usage: 1024,
          },
          system: {
            go_version: '1.23',
            num_cpu: 8,
            num_goroutines: 150,
          },
        }

        mockFetch.mockResolvedValueOnce({
          ok: true,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => mockStats,
        })

        const result = await api.getPerformanceStats()

        expect(mockFetch).toHaveBeenCalledWith(
          'http://localhost:8080/api/v1/performance/stats',
          expect.any(Object)
        )
        expect(result).toEqual(mockStats)
      })
    })
  })

  describe('Error Handling', () => {
    it('should handle non-JSON responses', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        headers: new Headers({ 'content-type': 'text/plain' }),
        json: async () => { throw new Error('Not JSON') },
      })

      const result = await api.getConfig()
      expect(result).toEqual({})
    })

    it('should throw error for network failures', async () => {
      mockFetch.mockRejectedValueOnce(new Error('Network error'))

      await expect(api.getConfig()).rejects.toThrow('Network error')
    })

    it('should handle 404 errors', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 404,
        statusText: 'Not Found',
        text: async () => 'Resource not found',
      })

      await expect(api.getPipeline('nonexistent')).rejects.toThrow('API error (404): Resource not found')
    })

    it('should handle 401 unauthorized errors', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 401,
        statusText: 'Unauthorized',
        text: async () => 'Authentication required',
      })

      await expect(api.getConfig()).rejects.toThrow('API error (401): Authentication required')
    })
  })

  describe('Digital Twins API', () => {
    describe('listDigitalTwins', () => {
      it('should fetch all digital twins from correct endpoint', async () => {
        const mockTwins = [
          { id: 'twin-1', name: 'Manufacturing Plant', ontology_id: 'ont-1' },
          { id: 'twin-2', name: 'Supply Chain', ontology_id: 'ont-2' },
        ]

        mockFetch.mockResolvedValueOnce({
          ok: true,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => ({
            success: true,
            data: {
              twins: mockTwins,
              count: 2
            }
          }),
        })

        const result = await api.listDigitalTwins()

        // Should call correct endpoint (not /twin)
        expect(mockFetch).toHaveBeenCalledWith(
          'http://localhost:8080/api/v1/twins',
          expect.objectContaining({
            headers: expect.objectContaining({
              'Content-Type': 'application/json',
            }),
          })
        )
        
        // Should parse response correctly
        expect(result).toEqual(mockTwins)
        expect(result).toHaveLength(2)
        expect(result[0].name).toBe('Manufacturing Plant')
      })

      it('should handle empty twins list', async () => {
        mockFetch.mockResolvedValueOnce({
          ok: true,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => ({
            success: true,
            data: {
              twins: [],
              count: 0
            }
          }),
        })

        const result = await api.listDigitalTwins()

        expect(result).toEqual([])
        expect(result).toHaveLength(0)
      })

      it('should handle malformed response gracefully', async () => {
        mockFetch.mockResolvedValueOnce({
          ok: true,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => ({
            success: true,
            data: null // Malformed - no twins array
          }),
        })

        const result = await api.listDigitalTwins()

        // Should return empty array instead of crashing
        expect(result).toEqual([])
      })

      it('should handle API errors', async () => {
        mockFetch.mockResolvedValueOnce({
          ok: false,
          status: 500,
          statusText: 'Internal Server Error',
          text: async () => 'Database connection failed',
        })

        await expect(api.listDigitalTwins()).rejects.toThrow('API error (500): Database connection failed')
      })
    })

    describe('getDigitalTwin', () => {
      it('should fetch a single digital twin by ID', async () => {
        const mockTwin = {
          id: 'twin-1',
          name: 'Manufacturing Plant',
          ontology_id: 'ont-1',
          description: 'Main production facility',
          created_at: '2024-01-01T00:00:00Z'
        }

        mockFetch.mockResolvedValueOnce({
          ok: true,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => ({
            success: true,
            data: mockTwin
          }),
        })

        const result = await api.getDigitalTwin('twin-1')

        expect(mockFetch).toHaveBeenCalledWith(
          'http://localhost:8080/api/v1/twin/twin-1',
          expect.any(Object)
        )
        expect(result).toEqual(mockTwin)
        expect(result.name).toBe('Manufacturing Plant')
      })

      it('should handle not found errors', async () => {
        mockFetch.mockResolvedValueOnce({
          ok: false,
          status: 404,
          statusText: 'Not Found',
          text: async () => 'Digital twin not found',
        })

        await expect(api.getDigitalTwin('nonexistent')).rejects.toThrow('API error (404): Digital twin not found')
      })
    })

    describe('getTwinState', () => {
      it('should fetch current state of a digital twin', async () => {
        const mockState = {
          twin_id: 'twin-1',
          state: {
            temperature: 75,
            pressure: 101.3,
            status: 'operational'
          },
          entity_states: {
            'sensor-1': {
              entity_id: 'sensor-1',
              properties: { reading: 75 }
            }
          }
        }

        mockFetch.mockResolvedValueOnce({
          ok: true,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => ({
            success: true,
            data: mockState
          }),
        })

        const result = await api.getTwinState('twin-1')

        expect(mockFetch).toHaveBeenCalledWith(
          'http://localhost:8080/api/v1/twin/twin-1/state',
          expect.any(Object)
        )
        expect(result).toEqual(mockState)
        expect(result.state.temperature).toBe(75)
      })
    })

    describe('createScenario', () => {
      it('should create a scenario for a digital twin', async () => {
        const scenarioRequest = {
          name: 'High Load Test',
          description: 'Test system under high load',
          parameters: {
            load_factor: 2.0,
            duration: 3600
          }
        }

        const mockResponse = {
          scenario_id: 'scenario-123',
          message: 'Scenario created successfully'
        }

        mockFetch.mockResolvedValueOnce({
          ok: true,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => ({
            success: true,
            data: mockResponse
          }),
        })

        const result = await api.createScenario('twin-1', scenarioRequest)

        expect(mockFetch).toHaveBeenCalledWith(
          'http://localhost:8080/api/v1/twin/twin-1/scenarios',
          expect.objectContaining({
            method: 'POST',
            body: JSON.stringify(scenarioRequest),
          })
        )
        expect(result).toEqual(mockResponse)
        expect(result.scenario_id).toBe('scenario-123')
      })
    })
  })

  describe('Ontology API', () => {
    describe('listOntologies', () => {
      it('should fetch all ontologies successfully', async () => {
        const mockOntologies = [
          { ontology_id: 'ont-1', name: 'Ontology 1', status: 'active' },
          { ontology_id: 'ont-2', name: 'Ontology 2', status: 'active' },
        ]

        mockFetch.mockResolvedValueOnce({
          ok: true,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => mockOntologies,
        })

        const result = await api.listOntologies()

        expect(mockFetch).toHaveBeenCalledWith(
          'http://localhost:8080/api/v1/ontology',
          expect.any(Object)
        )
        expect(result).toEqual(mockOntologies)
      })

      it('should filter ontologies by status', async () => {
        const mockOntologies = [
          { ontology_id: 'ont-1', name: 'Ontology 1', status: 'active' },
        ]

        mockFetch.mockResolvedValueOnce({
          ok: true,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => mockOntologies,
        })

        await api.listOntologies('active')

        expect(mockFetch).toHaveBeenCalledWith(
          'http://localhost:8080/api/v1/ontology?status=active',
          expect.any(Object)
        )
      })

      it('should handle API errors', async () => {
        mockFetch.mockResolvedValueOnce({
          ok: false,
          status: 500,
          statusText: 'Internal Server Error',
          text: async () => 'Server error',
        })

        await expect(api.listOntologies()).rejects.toThrow('API error (500): Server error')
      })
    })

    describe('getOntology', () => {
      it('should fetch a single ontology by ID', async () => {
        const mockResponse = {
          success: true,
          data: {
            ontology: { ontology_id: 'ont-1', name: 'Test Ontology' },
          },
        }

        mockFetch.mockResolvedValueOnce({
          ok: true,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => mockResponse,
        })

        const result = await api.getOntology('ont-1')

        expect(mockFetch).toHaveBeenCalledWith(
          'http://localhost:8080/api/v1/ontology/ont-1?include_content=false',
          expect.any(Object)
        )
        expect(result).toEqual(mockResponse)
      })

      it('should include content when requested', async () => {
        const mockResponse = {
          success: true,
          data: {
            ontology: { ontology_id: 'ont-1', name: 'Test Ontology' },
            content: '@prefix : <http://example.org/> .',
          },
        }

        mockFetch.mockResolvedValueOnce({
          ok: true,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => mockResponse,
        })

        const result = await api.getOntology('ont-1', true)

        expect(mockFetch).toHaveBeenCalledWith(
          'http://localhost:8080/api/v1/ontology/ont-1?include_content=true',
          expect.any(Object)
        )
        expect(result.data.content).toBeDefined()
      })
    })

    describe('uploadOntology', () => {
      it('should upload a new ontology', async () => {
        const uploadRequest = {
          name: 'New Ontology',
          description: 'Test ontology',
          format: 'turtle',
          ontology_data: '@prefix : <http://example.org/> .',
        }

        const mockResponse = {
          success: true,
          data: { ontology_id: 'new-ont-1', message: 'Ontology uploaded' },
        }

        mockFetch.mockResolvedValueOnce({
          ok: true,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => mockResponse,
        })

        const result = await api.uploadOntology(uploadRequest)

        expect(mockFetch).toHaveBeenCalledWith(
          'http://localhost:8080/api/v1/ontology',
          expect.objectContaining({
            method: 'POST',
            body: JSON.stringify(uploadRequest),
          })
        )
        expect(result.success).toBe(true)
        expect(result.data.ontology_id).toBe('new-ont-1')
      })
    })

    describe('deleteOntology', () => {
      it('should delete an ontology', async () => {
        const mockResponse = {
          success: true,
          data: { ontology_id: 'ont-1', status: 'deleted', message: 'Success' },
        }

        mockFetch.mockResolvedValueOnce({
          ok: true,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => mockResponse,
        })

        const result = await api.deleteOntology('ont-1')

        expect(mockFetch).toHaveBeenCalledWith(
          'http://localhost:8080/api/v1/ontology/ont-1',
          expect.objectContaining({
            method: 'DELETE',
          })
        )
        expect(result.success).toBe(true)
      })
    })

    describe('exportOntology', () => {
      it('should export ontology in turtle format', async () => {
        const mockContent = '@prefix : <http://example.org/> .'

        mockFetch.mockResolvedValueOnce({
          ok: true,
          headers: new Headers({ 'content-type': 'text/plain' }),
          text: async () => mockContent,
        })

        const result = await api.exportOntology('ont-1', 'turtle')

        expect(mockFetch).toHaveBeenCalledWith(
          'http://localhost:8080/api/v1/ontology/ont-1/export?format=turtle',
          expect.any(Object)
        )
        expect(result).toBe(mockContent)
      })
    })
  })

  describe('Extraction API', () => {
    describe('listExtractionJobs', () => {
      it('should fetch all extraction jobs successfully', async () => {
        const mockJobs = {
          success: true,
          data: {
            jobs: [
              { job_id: 'job-1', job_name: 'Job 1', status: 'completed' },
              { job_id: 'job-2', job_name: 'Job 2', status: 'running' },
            ],
          },
        }

        mockFetch.mockResolvedValueOnce({
          ok: true,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => mockJobs,
        })

        const result = await api.listExtractionJobs()

        expect(mockFetch).toHaveBeenCalledWith(
          'http://localhost:8080/api/v1/extraction/jobs',
          expect.any(Object)
        )
        expect(result).toEqual(mockJobs)
      })

      it('should filter jobs by ontology ID', async () => {
        const mockJobs = {
          success: true,
          data: { jobs: [] },
        }

        mockFetch.mockResolvedValueOnce({
          ok: true,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => mockJobs,
        })

        await api.listExtractionJobs({ ontology_id: 'ont-1' })

        expect(mockFetch).toHaveBeenCalledWith(
          'http://localhost:8080/api/v1/extraction/jobs?ontology_id=ont-1',
          expect.any(Object)
        )
      })

      it('should handle API errors', async () => {
        mockFetch.mockResolvedValueOnce({
          ok: false,
          status: 500,
          statusText: 'Internal Server Error',
          text: async () => 'Server error',
        })

        await expect(api.listExtractionJobs()).rejects.toThrow('API error (500): Server error')
      })
    })

    describe('getExtractionJob', () => {
      it('should fetch a single extraction job by ID', async () => {
        const mockResponse = {
          success: true,
          data: {
            job: { job_id: 'job-1', job_name: 'Test Job', status: 'completed' },
            entities: [
              { entity_id: 'ent-1', type: 'Person', text: 'John' },
            ],
          },
        }

        mockFetch.mockResolvedValueOnce({
          ok: true,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => mockResponse,
        })

        const result = await api.getExtractionJob('job-1')

        expect(mockFetch).toHaveBeenCalledWith(
          'http://localhost:8080/api/v1/extraction/jobs/job-1',
          expect.any(Object)
        )
        expect(result).toEqual(mockResponse)
        expect(result.data.job.job_id).toBe('job-1')
        expect(result.data.entities).toHaveLength(1)
      })
    })

    describe('createExtractionJob', () => {
      it('should create a new extraction job', async () => {
        const jobData = {
          ontology_id: 'ont-1',
          job_name: 'New Job',
          extraction_type: 'deterministic' as const,
          source_type: 'text' as const,
          data: { text: 'Test data' },
        }

        const mockResponse = {
          success: true,
          data: { job_id: 'new-job-1', message: 'Job created' },
        }

        mockFetch.mockResolvedValueOnce({
          ok: true,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => mockResponse,
        })

        const result = await api.createExtractionJob(jobData)

        expect(mockFetch).toHaveBeenCalledWith(
          'http://localhost:8080/api/v1/extraction/jobs',
          expect.objectContaining({
            method: 'POST',
            body: JSON.stringify(jobData),
          })
        )
        expect(result.success).toBe(true)
        expect(result.data.job_id).toBe('new-job-1')
      })
    })
  })
})
