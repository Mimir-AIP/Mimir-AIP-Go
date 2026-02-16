# Overall Plan for Mimir AIP
Mimir AIP will be an ontology-backed data platform for data aggregation, processing, analysis, digital twin creation, management and use(digital twins will be a ontology backed clone of a project encompassing the ontology, ML models, anomaly detection, sparql based querying, outputs[via triggering a output type pipeline to generate reports, send push notifications etc.])
^ All of above will be developed with the ability to trigger via mcp tools, at a later point I will include an AI chat functionality which if users choose can be the primary means on interacting with the system, all system functionality exposed as tools to allow agents(either within the integrated agent chat page OR by using the mcp tools with existing coding-agent systems such as claude code, opencode etc.)

## Languages:
### Backend: Go
### Frontend: Simple static site using a small number of primitive components(7 max) which will call backend orchestrator server via a REST API

## Development:
explained in: [DevelopmentPlan.md](DevelopmentPlan.md)

## Infrastructure:
explained in: [Infrastructure/InfrastructurePlan.md](Infrastructure/InfrastructurePlan.md)

## Projects:
explained in: [Projects/ProjectsPlan.md](Projects/ProjectsPlan.md)

## Ontologies:
explained in: [Ontologies/OntologyPlan.md](Ontologies/OntologyPlan.md)

### Entity Extraction:
explained in: [Ontologies/EntityExtractionPlan.md](Ontologies/EntityExtractionPlan.md)

## Pipelines:
explained in: [Pipelines/PipelinePlan.md](Pipelines/PipelinePlan.md)

### Jobs:
explained in: [Pipelines/JobsPlan.md](Pipelines/JobsPlan.md)

## Storage:
explained in: [Storage/StoragePlan.md](Storage/StoragePlan.md)

## ML Models:
explained in: [MLModels/MLModelPlan.md](MLModels/MLModelPlan.md)

## Digital Twin:
explained in: [DigitalTwins/DigitalTwinPlan.md](DigitalTwins/DigitalTwinPlan.md)



## Heirarchy:
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
