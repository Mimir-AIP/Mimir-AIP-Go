/**
 * Test data fixtures for e2e tests
 */

export const testUsers = {
  admin: {
    username: 'admin',
    password: 'admin123',
    apiKey: 'test-api-key-admin',
  },
  user: {
    username: 'testuser',
    password: 'test123',
    apiKey: 'test-api-key-user',
  },
};

export const testOntology = {
  name: 'Test Ontology',
  description: 'E2E test ontology',
  version: '1.0.0',
  format: 'turtle',
  status: 'active',
  content: `@prefix : <http://example.org/test#> .
@prefix owl: <http://www.w3.org/2002/07/owl#> .
@prefix rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .

:TestOntology a owl:Ontology ;
    rdfs:label "Test Ontology" ;
    rdfs:comment "A test ontology for E2E testing" .

:Person a owl:Class ;
    rdfs:label "Person" ;
    rdfs:comment "A person entity" .

:Organization a owl:Class ;
    rdfs:label "Organization" ;
    rdfs:comment "An organization entity" .

:name a owl:DatatypeProperty ;
    rdfs:label "name" ;
    rdfs:domain :Person ;
    rdfs:range xsd:string .

:worksFor a owl:ObjectProperty ;
    rdfs:label "works for" ;
    rdfs:domain :Person ;
    rdfs:range :Organization .
`,
};

export const testSPARQLQueries = {
  countTriples: `SELECT (COUNT(*) AS ?count) WHERE { ?s ?p ?o }`,
  listClasses: `PREFIX owl: <http://www.w3.org/2002/07/owl#>
PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
SELECT DISTINCT ?class ?label WHERE {
  ?class a owl:Class .
  OPTIONAL { ?class rdfs:label ?label }
}
ORDER BY ?class
LIMIT 100`,
  listProperties: `PREFIX owl: <http://www.w3.org/2002/07/owl#>
PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
SELECT DISTINCT ?property ?label WHERE {
  { ?property a owl:ObjectProperty } UNION { ?property a owl:DatatypeProperty }
  OPTIONAL { ?property rdfs:label ?label }
}
ORDER BY ?property
LIMIT 100`,
};

export const testPipeline = {
  name: 'Test Pipeline',
  description: 'E2E test pipeline',
  yaml: `version: "1.0"
name: test-pipeline
description: A test pipeline for E2E testing
steps:
  - name: input_step
    plugin: input/http
    config:
      url: https://api.example.com/data
  - name: output_step
    plugin: output/json
    config:
      file: test_output.json
`,
};

export const testDigitalTwin = {
  name: 'Test Digital Twin',
  description: 'E2E test digital twin',
  ontologyId: '', // Will be set dynamically
  initialState: {
    temperature: 20,
    humidity: 60,
  },
};

export const testScenario = {
  name: 'Test Scenario',
  description: 'E2E test scenario',
  parameters: {
    duration: 60,
    temperature_change: 5,
  },
};
