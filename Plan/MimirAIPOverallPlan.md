Overall Plan for Mimir AIP
---
Mimir AIP will be an ontology-backed data platform for data aggregation, processing, analysis, digital twin creation, management and use(digital twins will be a ontology backed clone of a project encompassing the ontology, ML models, anomaly detection, sparql based querying, outputs[via triggering a output type pipeline to generate reports, send push notifications etc.])
^ All of above will be developed with the ability to trigger via mcp tools, at a later point I will include an AI chat functionality which if users choose can be the primary means on interacting with the system, all system functionality exposed as tools to allow agents(either within the integrated agent chat page OR by using the mcp tools with existing coding-agent systems such as claude code, opencode etc.)

Languages:
Backend: Go
Frontend: Simple static site using a small number of primitive components(7 max) which will call backend server via a REST API

Infrastructure:
Frontend single kubernetes container
Backend Orchestrater single kubernetes container
Backend workers(can be used to run pipelines, ml models(inference & training), digital twin jobs etc.) scalable kubernetes workers(one worker per job; orchestrater spins this up, worker completes job, returns results to orchestrator and closes)
^Initially during dev this will all be on single system but in theory the workers should be able to scale across a cluster of systems or even remote systems for distributed sclaing.

Storage:
Backend Orchestrater server/container will use an abstract, tabular storage interface and user can use different modular plugins to determine where and how their data is stored(this could be SQL, S3, Mongo, Supabase etc., neo4j etc.) Backend orchestrater should not need to 'care' about how the data is actually being stored the conversion from abstract to specifics for both storage and retrieval is handled by the plugin. 

Heirarchy:
```mermaid
flowchart TD
	subgraph Project
		direction TB
		ProjectNode((Project))
		Storage((Storage))
		subgraph IngestionPipelines[Ingestion Pipelines]
			direction TB
			Pipeline1((Pipeline 1))
			Pipeline2((Pipeline 2))
			PipelineN((Pipeline N))
		end
		subgraph Ontologies
			direction TB
			Ontology1((Ontology 1))
			Ontology2((Ontology 2))
			OntologyN((Ontology N))
		end
		subgraph MLModels[ML Models]
			direction TB
			MLModel1((ML Model 1))
			MLModel2((ML Model 2))
			MLModelN((ML Model N))
		end
		DigitalTwin((Digital Twin))
		subgraph OutputPipelines[Output Pipelines]
			direction TB
			OutPipeline1((Output Pipeline 1))
			OutPipeline2((Output Pipeline 2))
			OutPipelineN((Output Pipeline N))
		end
		ProjectNode -->|1-1| Storage
		Pipeline1 --> Ontology1
		Pipeline2 --> Ontology1
		Pipeline2 --> Ontology2
		PipelineN --> OntologyN
		Ontology1 --> DigitalTwin
		Ontology2 --> DigitalTwin
		OntologyN --> DigitalTwin
		Ontology1 --> MLModel1
		Ontology2 --> MLModel2
		OntologyN --> MLModelN
		MLModel1 --> DigitalTwin
		MLModel2 --> DigitalTwin
		MLModelN --> DigitalTwin
		DigitalTwin --> OutPipeline1
		DigitalTwin --> OutPipeline2
		DigitalTwin --> OutPipelineN
	end
	classDef project fill:#f9f,stroke:#333,stroke-width:2px;
	class ProjectNode project;
```
