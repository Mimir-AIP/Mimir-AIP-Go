# Storage

Backend Orchestrater server/container will use an abstract CIR-based storage interface and user can use different modular plugins to determine where and how their data is stored(this could be SQL, S3, Mongo, Supabase etc., neo4j etc.) Backend orchestrater should not need to 'care' about how the data is actually being stored the conversion from abstract to specifics for both storage and retrieval is handled by the plugin. The backend orchestrater will use the project's ontology to structure CIR data for storage. Plugins may override or amend this structure as needed to fit the specific storage type, but the backend orchestrater will always interact with the storage through the abstract CIR interface. When pipelines are executed and new data is ingested, this will be stored in the storage as CIR data structured according to the ontology. When ML models are being trained or making inferences, they will retrieve data from storage in CIR format. Finally when building the digital twin, it will also retrieve data from storage in CIR format to populate the entities, attributes and relationships defined in the ontology.(Digital twins do not store a entire copy of the data rather they query and reference the data, and if data modification it will store only the changed values seperately and reference the original data for unchanged values)

## Common internal representation
When data ingested from ingestion pipelines, it will be converted into a common internal representation (CIR). The CIR is a standardized JSON schema that normalizes ingested data from various sources (APIs, files, databases) into a uniform format. This allows downstream processing components, such as entity extraction and ontology generation, to operate consistently regardless of the original data source or format. The CIR serves as the bridge between raw data ingestion and structured storage, ensuring that all data passes through the same processing pipeline before being stored according to the project's ontology.

### CIR Schema
The CIR follows a flexible JSON schema that can accommodate both structured and unstructured data:

```json
{
  "version": "1.0",
  "source": {
    "type": "string",  // e.g., "api", "file", "database", "stream"
    "uri": "string",   // Source identifier (URL, file path, etc.)
    "timestamp": "string",  // ISO 8601 timestamp of ingestion
    "format": "string",     // Original format: "csv", "json", "xml", "text", "binary"
    "parameters": "object"  // Optional ingestion parameters
  },
  "data": "any",  // Flexible data container - can be object, array, string, etc.
  "metadata": {
    "size": "number",       // Data size in bytes
    "encoding": "string",   // Character encoding if applicable
    "record_count": "number", // Number of records/items (for structured data)
    "schema_inference": "object", // Optional inferred schema information
    "quality_metrics": "object"   // Optional data quality indicators
  }
}
```

### How CIR Fits with Pipeline Plan
Ingestion pipelines, as defined in the Pipeline Plan, are responsible for fetching raw data and converting it into CIR format. Each ingestion pipeline culminates in a step that outputs data in CIR schema, ensuring consistency. The CIR is then automatically passed to entity extraction processes for ontology generation. This integration means:

- Pipelines handle source-specific logic (authentication, parsing, error handling)
- CIR provides a standardized interface for all downstream processing
- Entity extraction algorithms can assume CIR input format
- Storage plugins receive CIR-converted data for persistence

### Examples

#### Example 1: Structured CSV Data Ingestion
**Pipeline Steps:**
1. Fetch CSV file from a URL or local path
2. Parse CSV into array of objects
3. Convert to CIR format

**Sample Pipeline YAML:**
```yaml
name: csv_employee_data_ingestion
type: ingestion
steps:
  - name: fetch_csv
    plugin: default
    parameters:
      url: https://example.com/employees.csv
      method: GET
    action: http_request
    output:
      csv_content: "{{response.body}}"
  - name: parse_csv
    plugin: default
    parameters:
      csv_data: "{{context.fetch_csv.csv_content}}"
      has_header: true
    action: parse_csv
    output:
      parsed_data: "{{parsed}}"
  - name: create_cir
    plugin: default
    parameters:
      source_type: "api"
      source_uri: "https://example.com/employees.csv"
      data: "{{context.parse_csv.parsed_data}}"
      format: "csv"
    action: create_cir
    output:
      cir_output: "{{cir}}"
```

**Resulting CIR:**
```json
{
  "version": "1.0",
  "source": {
    "type": "api",
    "uri": "https://example.com/employees.csv",
    "timestamp": "2026-02-15T22:51:36Z",
    "format": "csv",
    "parameters": {}
  },
  "data": [
    {
      "id": "1",
      "name": "John Doe",
      "department": "Engineering",
      "manager": "Jane Smith",
      "salary": "75000"
    },
    {
      "id": "2",
      "name": "Jane Smith",
      "department": "Engineering",
      "manager": "Bob Johnson",
      "salary": "85000"
    }
  ],
  "metadata": {
    "size": 1024,
    "encoding": "utf-8",
    "record_count": 2,
    "schema_inference": {
      "columns": ["id", "name", "department", "manager", "salary"],
      "types": ["string", "string", "string", "string", "string"]
    }
  }
}
```

#### Example 2: Unstructured Text from API
**Pipeline Steps:**
1. Make API request to fetch raw text
2. Validate response
3. Convert to CIR format

**Sample Pipeline YAML:**
```yaml
name: api_text_ingestion
type: ingestion
steps:
  - name: fetch_text
    plugin: default
    parameters:
      url: https://api.example.com/articles/latest
      method: GET
      headers:
        Authorization: "Bearer {{api_key}}"
    action: http_request
    output:
      api_response: "{{response.body}}"
  - name: validate_response
    plugin: default
    parameters:
      condition: "{{context.fetch_text.api_response}}"
      if_false: "error_handling"
    action: if_else
    output:
      validation_result: "{{result}}"
  - name: create_cir
    plugin: default
    parameters:
      source_type: "api"
      source_uri: "https://api.example.com/articles/latest"
      data: "{{context.fetch_text.api_response}}"
      format: "text"
    action: create_cir
    output:
      cir_output: "{{cir}}"
```

**Resulting CIR:**
```json
{
  "version": "1.0",
  "source": {
    "type": "api",
    "uri": "https://api.example.com/articles/latest",
    "timestamp": "2026-02-15T22:51:36Z",
    "format": "text",
    "parameters": {
      "headers": {
        "Authorization": "Bearer [REDACTED]"
      }
    }
  },
  "data": "John Doe works for TechCorp in the Engineering department. He reports to Jane Smith, who is the director of the department. TechCorp is located in San Francisco and has been operating since 2010. The company specializes in AI solutions and has partnerships with several major clients including GlobalTech and InnovateLabs.",
  "metadata": {
    "size": 387,
    "encoding": "utf-8",
    "record_count": 1,
    "quality_metrics": {
      "word_count": 78,
      "sentence_count": 4
    }
  }
}
```

The CIR representation ensures that both structured data and unstructured text are uniformly represented, allowing entity extraction algorithms to process them consistently for ontology generation and subsequent storage.

## Storage Plugins

The backend orchestrater interacts with storage through an abstract CIR-based interface, treating all data as CIR objects regardless of the underlying storage technology. This abstraction allows the orchestrater to perform operations like storing ingested data, querying for ML training, and retrieving data for digital twin population without needing to understand storage-specific details. When the orchestrater needs to store data, it provides CIR objects structured according to the project's ontology. For retrieval, it issues queries and expects results as CIR objects. The plugins handle the bidirectional translation between this abstract CIR representation and the concrete storage system's native format.

### Plugin Schema

Storage plugins must implement a standardized interface to enable bidirectional translation between the abstract CIR format and the specific storage technology. The plugin schema defines the contract that all storage plugins must follow, ensuring compatibility with the orchestrater's CIR abstraction layer.

```typescript
interface StoragePlugin {
  // Initialize the plugin with configuration
  initialize(config: PluginConfig): Promise<void>;

  // Create or update the storage schema based on the ontology definition
  createSchema(ontology: OntologyDefinition): Promise<void>;

  // Store CIR data into the storage system
  store(cir: CIR): Promise<StorageResult>;

  // Retrieve data using queries and return as CIR objects
  retrieve(query: CIRQuery): Promise<CIR[]>;

  // Update existing CIR data
  update(query: CIRQuery, updates: CIRUpdate): Promise<StorageResult>;

  // Delete CIR data
  delete(query: CIRQuery): Promise<StorageResult>;

  // Get storage-specific metadata
  getMetadata(): Promise<StorageMetadata>;

  // Health check and connection validation
  healthCheck(): Promise<boolean>;
}

// Supporting types
interface PluginConfig {
  connectionString: string;
  credentials?: Record<string, any>;
  options?: Record<string, any>;
}

interface OntologyDefinition {
  entities: EntityDefinition[];
  relationships: RelationshipDefinition[];
}

interface EntityDefinition {
  name: string;
  attributes: AttributeDefinition[];
  primaryKey?: string[];
}

interface AttributeDefinition {
  name: string;
  type: 'string' | 'number' | 'boolean' | 'date' | 'json';
  nullable?: boolean;
  defaultValue?: any;
}

interface RelationshipDefinition {
  name: string;
  fromEntity: string;
  toEntity: string;
  type: 'one-to-one' | 'one-to-many' | 'many-to-many';
}

interface CIR {
  version: string;
  source: {
    type: string;
    uri: string;
    timestamp: string;
    format: string;
    parameters?: Record<string, any>;
  };
  data: any;
  metadata: {
    size: number;
    encoding?: string;
    record_count?: number;
    schema_inference?: Record<string, any>;
    quality_metrics?: Record<string, any>;
  };
}

interface CIRQuery {
  entityType?: string;
  filters?: CIRCondition[];
  orderBy?: OrderByClause[];
  limit?: number;
  offset?: number;
}

interface CIRCondition {
  attribute: string;
  operator: 'eq' | 'neq' | 'gt' | 'gte' | 'lt' | 'lte' | 'in' | 'like';
  value: any;
}

interface OrderByClause {
  attribute: string;
  direction: 'asc' | 'desc';
}

interface CIRUpdate {
  filters: CIRCondition[];
  updates: Record<string, any>;
}

interface StorageResult {
  success: boolean;
  affectedItems?: number;
  error?: string;
}

interface StorageMetadata {
  storageType: string;
  version: string;
  capabilities: string[];
}
```

This schema enables bidirectional translation by requiring plugins to convert between the orchestrater's CIR format and the storage system's native representation. For example, a SQL plugin would translate CIR queries to SQL statements, while a graph database plugin might convert CIR data to nodes and relationships.

### Neo4j Storage Plugin Pseudocode Example

Neo4j is a graph database that stores data as nodes, relationships, and properties. The plugin translates the ontology and CIR data into a graph model where:

- Each entity type becomes a node label
- Entity instances from CIR.data become nodes with that label
- Attributes become node properties
- Relationships between entities are represented as graph edges based on the ontology

```python
class Neo4jStoragePlugin(StoragePlugin):
    def __init__(self):
        self.driver = None
        self.entity_mappings = {}  # Maps entity names to node labels

    async def initialize(self, config: PluginConfig):
        self.driver = GraphDatabase.driver(
            config.connectionString,
            auth=(config.credentials.username, config.credentials.password)
        )
        # Validate connection
        await self.healthCheck()

    async def createSchema(self, ontology: OntologyDefinition):
        with self.driver.session() as session:
            for entity in ontology.entities:
                # Create constraints for primary keys
                if entity.primaryKey:
                    pk_constraint = f"CREATE CONSTRAINT {entity.name}_pk IF NOT EXISTS FOR (n:{entity.name}) REQUIRE ({', '.join(f'n.{attr}' for attr in entity.primaryKey)}) IS NODE KEY"
                    await session.run(pk_constraint)

                # Create indexes for attributes
                for attr in entity.attributes:
                    if attr.name in (entity.primaryKey or []):
                        continue  # Already indexed by constraint
                    idx_query = f"CREATE INDEX {entity.name}_{attr.name} IF NOT EXISTS FOR (n:{entity.name}) ON (n.{attr.name})"
                    await session.run(idx_query)

                # Store mapping for later use
                self.entity_mappings[entity.name] = entity.name

            # Create relationship constraints/indexes
            for rel in ontology.relationships:
                # For many-to-many, we might create relationship indexes
                pass

    async def store(self, cir: CIR) -> StorageResult:
        with self.driver.session() as session:
            # Assume cir.data is an array of entity objects or a single entity
            if isinstance(cir.data, list):
                entities = cir.data
            else:
                entities = [cir.data]

            total_created = 0
            for entity in entities:
                entity_type = self._infer_entity_type(entity)
                if not entity_type:
                    continue

                # Create node
                properties = {k: v for k, v in entity.items() if k != 'relationships'}
                query = f"CREATE (n:{entity_type}) SET n = $props"
                result = await session.run(query, props=properties)
                total_created += result.consume().counters.nodes_created

                # Handle relationships if present
                if 'relationships' in entity:
                    await self._create_relationships(session, entity, entity_type)

            return StorageResult(success=True, affectedItems=total_created)

    async def retrieve(self, query: CIRQuery) -> List[CIR]:
        with self.driver.session() as session:
            # Build Cypher query from CIR query
            cypher_query = self._build_cypher_query(query)
            result = await session.run(cypher_query)

            # Convert results back to CIR format
            cir_objects = []
            async for record in result:
                node = record['n']
                entity_data = dict(node)
                # Add relationships if needed
                # For simplicity, just return entity data
                cir_obj = CIR(
                    version="1.0",
                    source={
                        "type": "storage",
                        "uri": f"neo4j:{query.entityType}",
                        "timestamp": datetime.now().isoformat(),
                        "format": "graph"
                    },
                    data=entity_data,
                    metadata={
                        "size": len(str(entity_data)),
                        "record_count": 1
                    }
                )
                cir_objects.append(cir_obj)

            return cir_objects

    def _build_cypher_query(self, query: CIRQuery) -> str:
        # Build MATCH clause
        entity_type = query.entityType or "Entity"
        match_clause = f"MATCH (n:{entity_type})"

        # Build WHERE clause
        where_clauses = []
        for condition in query.filters or []:
            if condition.operator == 'eq':
                where_clauses.append(f"n.{condition.attribute} = ${condition.attribute}")
            elif condition.operator == 'gt':
                where_clauses.append(f"n.{condition.attribute} > ${condition.attribute}")
            # Add other operators as needed...

        where_clause = f"WHERE {' AND '.join(where_clauses)}" if where_clauses else ""

        # Build RETURN clause
        return_clause = "RETURN n"

        # Build ORDER BY and LIMIT
        order_by = ""
        if query.orderBy:
            order_parts = [f"n.{clause.attribute} {clause.direction.upper()}" for clause in query.orderBy]
            order_by = f"ORDER BY {', '.join(order_parts)}"

        limit_clause = f"LIMIT {query.limit}" if query.limit else ""

        # Combine all parts
        full_query = f"{match_clause} {where_clause} {return_clause} {order_by} {limit_clause}".strip()
        return full_query

    async def update(self, query: CIRQuery, updates: CIRUpdate) -> StorageResult:
        with self.driver.session() as session:
            # Build WHERE clause from query filters
            where_clauses = []
            params = {}
            for condition in (query.filters or []) + (updates.filters or []):
                where_clauses.append(f"n.{condition.attribute} = ${condition.attribute}")
                params[condition.attribute] = condition.value

            where_clause = f"WHERE {' AND '.join(where_clauses)}"

            # Build SET clause
            set_parts = [f"n.{key} = ${key}" for key in updates.updates.keys()]
            set_clause = f"SET {', '.join(set_parts)}"
            params.update(updates.updates)

            entity_type = query.entityType or "Entity"
            cypher_query = f"MATCH (n:{entity_type}) {where_clause} {set_clause}"
            result = await session.run(cypher_query, params)

            return StorageResult(success=True, affectedItems=result.consume().counters.properties_set)

    async def delete(self, query: CIRQuery) -> StorageResult:
        with self.driver.session() as session:
            # Build WHERE clause
            where_clauses = []
            params = {}
            for condition in query.filters or []:
                where_clauses.append(f"n.{condition.attribute} = ${condition.attribute}")
                params[condition.attribute] = condition.value

            where_clause = f"WHERE {' AND '.join(where_clauses)}" if where_clauses else ""

            entity_type = query.entityType or "Entity"
            cypher_query = f"MATCH (n:{entity_type}) {where_clause} DELETE n"
            result = await session.run(cypher_query, params)

            return StorageResult(success=True, affectedItems=result.consume().counters.nodes_deleted)

    def _infer_entity_type(self, entity: dict) -> str:
        # Simple inference based on attributes present
        # In practice, this would use the ontology to determine entity type
        if 'name' in entity and 'department' in entity:
            return 'Employee'
        elif 'name' in entity and 'location' in entity:
            return 'Company'
        return 'Entity'  # Default

    async def _create_relationships(self, session, entity: dict, entity_type: str):
        # Create relationships based on entity data
        # This is simplified; in practice, would use ontology relationships
        if 'manager' in entity:
            query = """
            MATCH (e:{entity_type} {{name: $entity_name}})
            MATCH (m:{manager_type} {{name: $manager_name}})
            CREATE (e)-[:REPORTS_TO]->(m)
            """.format(entity_type=entity_type, manager_type='Employee')
            await session.run(query, entity_name=entity['name'], manager_name=entity['manager'])

    async def getMetadata(self) -> StorageMetadata:
        # Query Neo4j for version and capabilities
        with self.driver.session() as session:
            result = await session.run("CALL dbms.components() YIELD name, versions, edition")
            components = await result.single()
            return StorageMetadata(
                storageType="neo4j",
                version=components["versions"][0],
                capabilities=["graph", "cypher", "acid", "indexing"]
            )

    async def healthCheck(self) -> bool:
        try:
            with self.driver.session() as session:
                await session.run("RETURN 1")
            return True
        except Exception:
            return False
```

This Neo4j plugin demonstrates how a graph database can be adapted to work with the CIR abstraction. While Neo4j stores data as a graph, the plugin presents a CIR interface to the orchestrater, allowing it to store and query data using entity-based concepts while leveraging Neo4j's graph capabilities for complex relationship traversals and pattern matching.
