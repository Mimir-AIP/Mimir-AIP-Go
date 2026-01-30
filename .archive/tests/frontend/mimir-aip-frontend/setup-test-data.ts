/**
 * Setup script to create persistent test data for E2E tests
 * This creates ontologies, extraction jobs, and other data needed by tests
 * 
 * Run: npx ts-node e2e/setup-test-data.ts
 */

const API_BASE_URL = process.env.API_URL || 'http://localhost:8080';

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

:hasAge a owl:DatatypeProperty ;
    rdfs:domain :Person ;
    rdfs:range xsd:integer ;
    rdfs:label "has age" .

:worksAt a owl:ObjectProperty ;
    rdfs:domain :Person ;
    rdfs:range :Organization ;
    rdfs:label "works at" .
`.trim();

async function apiFetch(endpoint: string, options?: RequestInit) {
  const url = `${API_BASE_URL}${endpoint}`;
  const response = await fetch(url, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
  });
  
  if (!response.ok) {
    const text = await response.text();
    throw new Error(`API error (${response.status}): ${text}`);
  }
  
  return response.json();
}

async function createTestOntology() {
  console.log('Creating test ontology...');
  
  try {
    const response = await apiFetch('/api/v1/ontology', {
      method: 'POST',
      body: JSON.stringify({
        name: 'E2E Persistent Test Ontology',
        description: 'Persistent ontology for E2E testing - DO NOT DELETE',
        version: '1.0.0',
        format: 'turtle',
        ontology_data: testOntologyContent,
        created_by: 'e2e-setup-script',
      }),
    });
    
    // Handle different response formats
    let ontologyId = response.data?.ontology_id || response.ontology_id || response.id;
    
    // Handle map[value:xxx] format
    if (typeof ontologyId === 'object' && ontologyId.value) {
      ontologyId = ontologyId.value;
    }
    
    console.log('✓ Ontology created:', ontologyId);
    return ontologyId;
  } catch (err) {
    console.error('Failed to create ontology:', err);
    return null;
  }
}

async function createTestExtractionJob(ontologyId: string) {
  console.log('Creating test extraction job...');
  
  try {
    const response = await apiFetch('/api/v1/extraction/jobs', {
      method: 'POST',
      body: JSON.stringify({
        ontology_id: ontologyId,
        job_name: 'E2E Persistent Test Extraction',
        extraction_type: 'deterministic',
        source_type: 'text',
        data: {
          text: 'Alice Smith works at TechCorp. Bob Johnson is 35 years old and works at DataCo. Charlie Brown is a software engineer at TechCorp.',
        },
      }),
    });
    
    console.log('✓ Extraction job created:', response.data?.job_id || response.job_id);
    return response.data?.job_id || response.job_id;
  } catch (err) {
    console.error('Failed to create extraction job:', err);
    return null;
  }
}

async function createTestPipeline(index: number) {
  console.log(`Creating test pipeline ${index}...`);
  
  try {
    const response = await apiFetch('/api/v1/pipelines', {
      method: 'POST',
      body: JSON.stringify({
        metadata: {
          name: `E2E Test Pipeline ${index}`,
          description: `Persistent test pipeline ${index} - DO NOT DELETE`,
          enabled: true,
          tags: ['e2e-test', 'persistent'],
          created_by: 'e2e-setup-script',
        },
        config: {
          Name: `E2E Test Pipeline ${index}`,
          Description: `Persistent test pipeline ${index}`,
          Enabled: true,
          Steps: [
            {
              Name: 'test_step_1',
              Plugin: 'test/echo',
              Config: {
                message: `Test message from pipeline ${index}`,
              },
              Output: 'step1_output',
            },
            {
              Name: 'test_step_2',
              Plugin: 'test/transform',
              Config: {
                input: 'step1_output',
                transform: 'uppercase',
              },
              Output: 'final_output',
            },
          ],
        },
      }),
    });
    
    let pipelineId = response.metadata?.id || response.id || response.data?.pipeline_id;
    
    // Handle map[value:xxx] format
    if (typeof pipelineId === 'object' && pipelineId.value) {
      pipelineId = pipelineId.value;
    }
    
    console.log(`✓ Pipeline ${index} created:`, pipelineId);
    return pipelineId;
  } catch (err) {
    console.error(`Failed to create pipeline ${index}:`, err);
    return null;
  }
}

async function createTestDigitalTwin(ontologyId: string, index: number) {
  console.log(`Creating test digital twin ${index}...`);
  
  try {
    const response = await apiFetch('/api/v1/twin/create', {
      method: 'POST',
      body: JSON.stringify({
        name: `E2E Test Twin ${index}`,
        description: `Persistent test digital twin ${index} - DO NOT DELETE`,
        ontology_id: ontologyId,
        state: {
          temperature: 20 + index,
          pressure: 100 + (index * 10),
          status: 'active',
          last_updated: new Date().toISOString(),
        },
        properties: {
          type: 'test-device',
          location: `test-location-${index}`,
          manufacturer: 'E2E Test Co.',
        },
      }),
    });
    
    const twinId = response.data?.twin_id || response.twin_id || response.id;
    console.log(`✓ Digital twin ${index} created:`, twinId);
    return twinId;
  } catch (err) {
    console.error(`Failed to create digital twin ${index}:`, err);
    return null;
  }
}

async function createTestScenario(twinId: string, index: number) {
  console.log(`Creating test scenario for twin ${twinId}...`);
  
  try {
    const response = await apiFetch(`/api/v1/twin/${twinId}/scenarios`, {
      method: 'POST',
      body: JSON.stringify({
        name: `Test Scenario ${index}`,
        description: `Test scenario ${index} for E2E testing`,
        scenario_type: 'simulation',
        events: [
          {
            time: 0,
            action: 'set_temperature',
            value: 25,
          },
          {
            time: 10,
            action: 'increase_pressure',
            value: 150,
          },
        ],
        duration: 60,
      }),
    });
    
    const scenarioId = response.data?.scenario_id || response.scenario_id || response.id;
    console.log(`✓ Scenario created:`, scenarioId);
    return scenarioId;
  } catch (err) {
    console.error(`Failed to create scenario:`, err);
    return null;
  }
}

async function checkExistingData() {
  console.log('\nChecking existing test data...');
  
  try {
    // Check ontologies
    const ontologies = await apiFetch('/api/v1/ontology?status=active').catch(() => []);
    console.log(`Found ${ontologies.length} active ontologies`);
    
    // Check extraction jobs (may fail if plugin not available)
    let jobCount = 0;
    try {
      const jobs = await apiFetch('/api/v1/extraction/jobs');
      jobCount = jobs?.data?.jobs?.length || jobs?.jobs?.length || 0;
      console.log(`Found ${jobCount} extraction jobs`);
    } catch (err) {
      console.log('Extraction jobs check skipped (plugin not available)');
    }
    
    // Check pipelines
    const pipelines = await apiFetch('/api/v1/pipelines').catch(() => []);
    const pipelineCount = Array.isArray(pipelines) ? pipelines.length : 0;
    console.log(`Found ${pipelineCount} pipelines`);
    
    // Check digital twins
    let twinCount = 0;
    try {
      const twins = await apiFetch('/api/v1/twin');
      twinCount = twins?.data?.twins?.length || twins?.twins?.length || 0;
      console.log(`Found ${twinCount} digital twins`);
    } catch (err) {
      console.log('Digital twins check skipped (may not be available)');
    }
    
    // Check if we have a persistent test ontology
    const testOnt = ontologies.find((ont: any) => 
      ont.name.includes('E2E Persistent') || ont.description?.includes('DO NOT DELETE')
    );
    
    // Check if we have test pipelines
    const testPipelines = Array.isArray(pipelines) ? pipelines.filter((p: any) => 
      p.metadata?.name?.includes('E2E Test Pipeline') || 
      p.metadata?.description?.includes('DO NOT DELETE')
    ) : [];
    
    if (testOnt) {
      console.log(`\n✓ Found existing test ontology: ${testOnt.id}`);
      console.log(`✓ Found ${testPipelines.length} test pipelines`);
      return {
        ontologyId: testOnt.id,
        hasJobs: jobCount > 0,
        hasPipelines: testPipelines.length >= 3,
        hasTwins: twinCount > 0,
      };
    }
    
    return null;
  } catch (err) {
    console.error('Failed to check existing data:', err);
    return null;
  }
}

async function main() {
  console.log('=== E2E Test Data Setup ===\n');
  console.log(`API URL: ${API_BASE_URL}\n`);
  
  // Check if we already have test data
  const existing = await checkExistingData();
  
  if (existing?.ontologyId) {
    console.log('\n✓ Test data already exists!');
    console.log(`\nOntology ID: ${existing.ontologyId}`);
    
    let needsUpdate = false;
    
    if (!existing.hasJobs) {
      console.log('\n⚠ No extraction jobs found, creating one...');
      await createTestExtractionJob(existing.ontologyId);
      needsUpdate = true;
    }
    
    if (!existing.hasPipelines) {
      console.log('\n⚠ Insufficient test pipelines, creating 3...');
      for (let i = 1; i <= 3; i++) {
        await createTestPipeline(i);
        await new Promise(resolve => setTimeout(resolve, 500));
      }
      needsUpdate = true;
    }
    
    if (!existing.hasTwins) {
      console.log('\n⚠ No digital twins found, creating 2...');
      for (let i = 1; i <= 2; i++) {
        const twinId = await createTestDigitalTwin(existing.ontologyId, i);
        if (twinId) {
          await new Promise(resolve => setTimeout(resolve, 500));
          await createTestScenario(twinId, i);
        }
        await new Promise(resolve => setTimeout(resolve, 500));
      }
      needsUpdate = true;
    }
    
    if (!needsUpdate) {
      console.log('\n✓ All test data is complete!');
    }
    
    console.log('\n=== Setup Complete ===');
    return;
  }
  
  // Create new test data
  console.log('\nCreating new test data...\n');
  
  const ontologyId = await createTestOntology();
  
  if (!ontologyId) {
    console.error('\n❌ Failed to create ontology. Aborting setup.');
    return;
  }
  
  // Give the ontology a moment to be fully processed
  console.log('\nWaiting for ontology to be processed...');
  await new Promise(resolve => setTimeout(resolve, 2000));
  
  // Create extraction jobs (optional - plugin may not be loaded)
  console.log('\n--- Creating Extraction Jobs ---');
  const jobId = await createTestExtractionJob(ontologyId);
  if (jobId) {
    await new Promise(resolve => setTimeout(resolve, 1000));
  } else {
    console.log('⚠ Extraction jobs skipped (plugin not available)');
  }
  
  // Create pipelines
  console.log('\n--- Creating Pipelines ---');
  for (let i = 1; i <= 3; i++) {
    await createTestPipeline(i);
    await new Promise(resolve => setTimeout(resolve, 500));
  }
  
  // Create digital twins with scenarios
  console.log('\n--- Creating Digital Twins ---');
  let twinsCreated = 0;
  for (let i = 1; i <= 2; i++) {
    const twinId = await createTestDigitalTwin(ontologyId, i);
    if (twinId) {
      twinsCreated++;
      await new Promise(resolve => setTimeout(resolve, 500));
      await createTestScenario(twinId, i);
    }
    await new Promise(resolve => setTimeout(resolve, 500));
  }
  
  console.log('\n=== Setup Complete ===');
  console.log('\nTest data has been created:');
  console.log('  ✓ 1 Ontology');
  console.log(`  ${jobId ? '✓' : '⚠'} ${jobId ? '1+' : '0'} Extraction Job(s) ${!jobId ? '(plugin not available)' : ''}`);
  console.log('  ✓ 3 Pipelines');
  console.log(`  ${twinsCreated > 0 ? '✓' : '⚠'} ${twinsCreated} Digital Twin(s) with Scenarios`);
  console.log('\nYou can now run the E2E tests with better coverage.');
  console.log('\nNote: This data is persistent and will not be cleaned up automatically.');
  console.log('To remove ontology: curl -X DELETE http://localhost:8080/api/v1/ontology/<id>');
  console.log('To remove pipeline: curl -X DELETE http://localhost:8080/api/v1/pipelines/<id>');
  console.log('To remove twin: curl -X DELETE http://localhost:8080/api/v1/twin/<id>');
}

main().catch(console.error);
