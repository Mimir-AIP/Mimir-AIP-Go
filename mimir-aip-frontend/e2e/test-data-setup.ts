/**
 * Reusable test data setup utilities for E2E tests
 * Use in test.beforeAll() to ensure test data exists
 */

import { APIRequestContext } from '@playwright/test';

// Simple test ontology content
const testOntologyContent = `
@prefix : <http://example.org/test#> .
@prefix owl: <http://www.w3.org/2002/07/owl#> .
@prefix rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .

:TestOntology a owl:Ontology ;
    rdfs:label "E2E Test Ontology" ;
    rdfs:comment "Persistent ontology for E2E testing" .

:Person a owl:Class ;
    rdfs:label "Person" ;
    rdfs:comment "A human being" .

:Organization a owl:Class ;
    rdfs:label "Organization" ;
    rdfs:comment "A business or company" .

:hasName a owl:DatatypeProperty ;
    rdfs:domain :Person ;
    rdfs:range xsd:string ;
    rdfs:label "has name" .

:worksAt a owl:ObjectProperty ;
    rdfs:domain :Person ;
    rdfs:range :Organization ;
    rdfs:label "works at" .
`.trim();

export interface TestDataContext {
  ontologyId?: string;
  pipelineId?: string;
  extractionJobId?: string;
  modelId?: string;
  twinId?: string;
}

/**
 * Ensure at least one test ontology exists
 * Returns the ID of an existing or newly created ontology
 */
export async function ensureTestOntology(request: APIRequestContext): Promise<string | undefined> {
  try {
    // Check for existing ontologies
    const listResp = await request.get('/api/v1/ontology');
    if (listResp.ok()) {
      const ontologies = await listResp.json();
      if (ontologies && ontologies.length > 0) {
        return ontologies[0].id;
      }
    }

    // Create new test ontology
    const createResp = await request.post('/api/v1/ontology', {
      data: {
        name: `E2E Test Ontology ${Date.now()}`,
        version: '1.0.0',
        description: 'Test ontology for E2E testing',
        format: 'turtle',
        content: testOntologyContent,
      },
    });

    if (createResp.ok()) {
      const result = await createResp.json();
      // Handle both direct ID and nested data structure
      return result.ontology_id || result.data?.ontology_id || result.id || undefined;
    }

    return undefined;
  } catch (error) {
    console.error('Failed to ensure test ontology:', error);
    return undefined;
  }
}

/**
 * Ensure at least one test pipeline exists
 * Returns the ID of an existing or newly created pipeline
 */
export async function ensureTestPipeline(request: APIRequestContext): Promise<string | undefined> {
  try {
    // Check for existing pipelines
    const listResp = await request.get('/api/v1/pipelines');
    if (listResp.ok()) {
      const pipelines = await listResp.json();
      if (pipelines && pipelines.length > 0) {
        return pipelines[0].id;
      }
    }

    // Create new test pipeline
    const createResp = await request.post('/api/v1/pipelines', {
      data: {
        name: `E2E Test Pipeline ${Date.now()}`,
        description: 'Test pipeline for E2E testing',
        steps: [
          {
            name: 'test_step',
            plugin: 'Input.CSV',
            config: {
              file_path: 'test.csv',
            },
          },
        ],
      },
    });

    if (createResp.ok()) {
      const result = await createResp.json();
      return result.pipeline_id || result.data?.pipeline_id || result.id || undefined;
    }

    return undefined;
  } catch (error) {
    console.error('Failed to ensure test pipeline:', error);
    return undefined;
  }
}

/**
 * Create an extraction job for testing
 * Requires an existing ontology ID
 */
export async function createTestExtractionJob(
  request: APIRequestContext,
  ontologyId: string
): Promise<string | undefined> {
  try {
    const createResp = await request.post('/api/v1/extraction/jobs', {
      data: {
        ontology_id: ontologyId,
        job_name: `E2E Test Extraction ${Date.now()}`,
        extraction_type: 'deterministic',
        source_type: 'text',
        data: {
          text: 'Alice works at TechCorp. Bob is a software engineer at DataCo. Charlie manages the team.',
        },
      },
    });

    if (createResp.ok()) {
      const result = await createResp.json();
      return result.data?.job_id || result.job_id || undefined;
    }

    return undefined;
  } catch (error) {
    console.error('Failed to create test extraction job:', error);
    return undefined;
  }
}

/**
 * Setup comprehensive test data for a test suite
 * Call this in test.beforeAll() and it will return IDs of created/existing resources
 */
export async function setupTestData(
  request: APIRequestContext,
  options: {
    needsOntology?: boolean;
    needsPipeline?: boolean;
    needsExtractionJob?: boolean;
  } = {}
): Promise<TestDataContext> {
  const context: TestDataContext = {};

  if (options.needsOntology !== false) {
    // Default to true if not specified
    context.ontologyId = await ensureTestOntology(request);
  }

  if (options.needsPipeline) {
    context.pipelineId = await ensureTestPipeline(request);
  }

  if (options.needsExtractionJob && context.ontologyId) {
    context.extractionJobId = await createTestExtractionJob(request, context.ontologyId);
  }

  return context;
}

/**
 * Cleanup test data (optional - usually better to keep for performance)
 */
export async function cleanupTestData(
  request: APIRequestContext,
  context: TestDataContext
): Promise<void> {
  // Optionally implement cleanup
  // For now, we keep test data for performance
  // Real cleanup should happen via DB reset or container restart
}
