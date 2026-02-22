# Ontologies

## Overview
Ontologies will be used to define the entities, attributes and relationships for a specific project(ideally to ensure retrievability and queryability of all data, a project will have a single all encompassing ontology, however we will alllow for multiple ontologies to be used within a project if users want to keep different data separate).

## Definition Format
Ontologies will be defined using the standard OWL format, serialized in RDF.

## Automatic Ontology Creation
When the user has built their ingestion pipeline, mimir will automatically run this, then feed the ingested data into a extraction process(which will be defined below) to extract the entities, attributes and relationships from the ingested data, this will then be used to automatically build the ontology for the project.

## Manual Editing
Additionally if the user prefers they can then manually edit and overide this to either correct any errors, or use an existing ontology they have built elsewhere.

## Ontology Usage
This ontology will be used as a basis for how mimir will then structure and process data(e.g. when the ingestion pipelines execute in future and ingest new data, it will be stored in the storage object in the structure defined by the ontology, additionally when building ML models, the ontology will be used to determine what data is available and how it is structured which will inform the model architecture and training process, finally the ontology will also be used when building the digital twin as this will determine what entities, attributes and relationships are represented in the digital twin).

## Ontology Maintenance
To ensure the ontology is always up to date with the ingested data, whenever new data is ingested via the pipelines, mimir will automatically compare the new data with the existing ontology, if it differs mimir will stop and the user will be prompted to either re-run the ontology creation process to update the ontology, or ignore the changes, identifying what extraction failed to correctly identify and what it should have identified instead. This automated re-checking and updating of the ontology will ensure that the ontology always accurately represents the ingested data, which is crucial for ensuring that the data can be effectively used for analysis, ML model training and digital twin creation.

## Mimir AIP Ontology Syntax
Mimir AIP uses ontologies defined exclusively in Turtle format (.ttl), following OWL 2 specifications. This section provides a comprehensive reference for the syntax used to define entities, attributes, and relationships. Turtle is a human-readable RDF serialization that uses prefixes, triples, and semicolons for concise representation.

### Basic Turtle Syntax
- **Prefixes**: Define namespace shortcuts to avoid long URIs.
  - Example: `@prefix : <http://example.org/mimir#> .`
  - `@prefix owl: <http://www.w3.org/2002/07/owl#> .`
  - `@prefix rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .`
  - `@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .`
  - `@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .`

- **Triples**: The core structure is subject-predicate-object, separated by spaces and ended with a period.
  - Example: `:User a owl:Class .`

- **Multiple predicates**: Use semicolons to chain predicates for the same subject.
  - Example: `:User a owl:Class ; rdfs:label "User" .`

- **Multiple objects**: Use commas to list multiple objects for the same predicate.
  - Example: `:User rdfs:subClassOf :Person , :AccountHolder .`

### Defining Entities (Classes)
Entities in Mimir AIP are represented as OWL classes.
- Declare a class: `<entity_name> a owl:Class .`
- Add labels: `<entity_name> rdfs:label "Entity Label" .`
- Subclass relationships: `<subclass> rdfs:subClassOf <superclass> .`

### Defining Attributes (Datatype Properties)
Attributes are datatype properties linking entities to literal values.
- Declare a property: `<property_name> a owl:DatatypeProperty .`
- Domain (entity it applies to): `<property_name> rdfs:domain <entity_name> .`
- Range (data type): `<property_name> rdfs:range xsd:<type> .`
  - Common types: xsd:string, xsd:int, xsd:float, xsd:boolean, xsd:dateTime.

### Defining Relationships (Object Properties)
Relationships between entities are object properties.
- Declare a property: `<property_name> a owl:ObjectProperty .`
- Domain and range: `<property_name> rdfs:domain <source_entity> ; rdfs:range <target_entity> .`
- Inverse properties: `<property> owl:inverseOf <inverse_property> .`
- Cardinality restrictions: Use owl:Restriction for min/max cardinality.

### Individuals (Instances)
- Declare an individual: `<individual_name> a <class_name> .`
- Assign properties: `<individual_name> <property> <value> .`

### Axioms and Restrictions
- Equivalent classes: `<class1> owl:equivalentClass <class2> .`
- Disjoint classes: `<class1> owl:disjointWith <class2> .`
- Functional properties: `<property> a owl:FunctionalProperty .`
- Restrictions: Use blank nodes for complex constraints, e.g., `_:restriction a owl:Restriction ; owl:onProperty <property> ; owl:someValuesFrom <class> .`

### Best Practices for Mimir AIP
- Use descriptive URIs with a consistent base (e.g., http://example.org/mimir#).
- Always include rdfs:label for human-readable names.
- Define entities first, then properties, then relationships.
- Use comments with # for clarity.
- Validate ontologies using OWL reasoners like HermiT or Pellet.

This syntax ensures ontologies are machine-readable for automated processing while being maintainable by developers.

## Ontology Example
An example ontology in Turtle format:

```turtle
@prefix : <http://example.org/mimir#> .
@prefix owl: <http://www.w3.org/2002/07/owl#> .
@prefix rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .

:User a owl:Class ;
    rdfs:label "User" .

:Product a owl:Class ;
    rdfs:label "Product" .

:id a owl:DatatypeProperty ;
    rdfs:domain :User ;
    rdfs:range xsd:int .

:name a owl:DatatypeProperty ;
    rdfs:domain :User ;
    rdfs:range xsd:string .

:email a owl:DatatypeProperty ;
    rdfs:domain :User ;
    rdfs:range xsd:string .

:title a owl:DatatypeProperty ;
    rdfs:domain :Product ;
    rdfs:range xsd:string .

:price a owl:DatatypeProperty ;
    rdfs:domain :Product ;
    rdfs:range xsd:float .

:purchases a owl:ObjectProperty ;
    rdfs:domain :User ;
    rdfs:range :Product .
```
