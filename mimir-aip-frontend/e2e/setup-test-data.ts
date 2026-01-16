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
    
    console.log('✓ Ontology created:', response.data?.ontology_id || response.ontology_id);
    return response.data?.ontology_id || response.ontology_id;
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

async function checkExistingData() {
  console.log('\nChecking existing test data...');
  
  try {
    // Check ontologies
    const ontologies = await apiFetch('/api/v1/ontology?status=active');
    console.log(`Found ${ontologies.length} active ontologies`);
    
    // Check extraction jobs
    const jobs = await apiFetch('/api/v1/extraction/jobs');
    const jobCount = jobs?.data?.jobs?.length || jobs?.jobs?.length || 0;
    console.log(`Found ${jobCount} extraction jobs`);
    
    // Check if we have a persistent test ontology
    const testOnt = ontologies.find((ont: any) => 
      ont.name.includes('E2E Persistent') || ont.description?.includes('DO NOT DELETE')
    );
    
    if (testOnt) {
      console.log(`\n✓ Found existing test ontology: ${testOnt.id}`);
      return {
        ontologyId: testOnt.id,
        hasJobs: jobCount > 0,
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
    
    if (!existing.hasJobs) {
      console.log('\n⚠ No extraction jobs found, creating one...');
      await createTestExtractionJob(existing.ontologyId);
    }
    
    console.log('\n=== Setup Complete ===');
    return;
  }
  
  // Create new test data
  console.log('\nCreating new test data...\n');
  
  const ontologyId = await createTestOntology();
  
  if (ontologyId) {
    // Give the ontology a moment to be fully processed
    await new Promise(resolve => setTimeout(resolve, 2000));
    
    await createTestExtractionJob(ontologyId);
  }
  
  console.log('\n=== Setup Complete ===');
  console.log('\nTest data has been created. You can now run the E2E tests.');
  console.log('Note: This data is persistent and will not be cleaned up automatically.');
  console.log('To remove: curl -X DELETE http://localhost:8080/api/v1/ontology/<id>');
}

main().catch(console.error);
