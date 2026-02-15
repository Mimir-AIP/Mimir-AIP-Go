# Entity Extraction Plan

## Overview
Entity extraction is the automated, deterministic process of identifying and structuring entities, attributes, and relationships from ingested data to facilitate ontology creation. This process ensures that ontologies are built consistently and accurately from raw data sources, without relying on machine learning models that could introduce variability. The extraction occurs after data ingestion pipelines have processed and uses rule-based algorithms to parse structured and semi-structured data formats.

## Extraction Process
The entity extraction process follows a sequential pipeline:

1. **Data Input**: Receives processed data from ingestion pipelines, this may have originated in a number of formats (e.g., CSV, JSON, XML, relational databases), however once processed, all data is normalized to a common internal representation.
2. **Preprocessing**: Data (represented in common internal format) is cleaned and standardized to ensure consistency. This includes handling missing values, normalizing text (e.g., case normalization, removing punctuation), and converting data types as necessary.
3. **Rule-Based Extraction**: A set of predefined rules and patterns are applied to the preprocessed data to identify entities, attributes, and relationships. These rules are designed to provide general accuracy across various origin data formats and domains, ensuring that the extraction process is deterministic and repeatable.
4. **Entity Structuring**: Extracted entities are structured into a format suitable for ontology creation. This involves defining entity types, attributes, and relationships in a way that can be easily integrated into the ontology framework.
5. **Output Generation**: The structured entities are outputted in a format that can be directly used for ontology creation; RDF triples and OWL classes and properties.

## Rules and Patterns
Initial pass will focus on extracting entities froom any ingested structured data(if none ingested this will be skipped). Subsequent passes will focus on unstructured data, then a final pass to combine and reconcile entities from both structured and unstructured sources.
### Tabular Data
Entity extraction from tabular data treats each row as a potential entity, with columns representing attributes. This deterministic process ensures consistent extraction across different tabular formats.

#### Algorithm Overview
1. **Header Identification**: Extract column headers to define attribute names.
2. **Entity Creation**: Create entities from each data row, using the first column as the primary identifier.
3. **Attribute Assignment**: Assign cell values as entity attributes based on column headers.
4. **Relationship Inference**: Infer relationships based on specific column patterns (e.g., hierarchical relationships).

#### Pseudocode for Tabular Entity Extraction

```pseudocode
function extractEntitiesFromTabularData(table_data):
    # Assume table_data is a list of lists, first row is headers
    if not table_data or len(table_data) < 2:
        return {'entities': [], 'relationships': []}

    headers = table_data[0]
    entities = []
    relationships = []

    for row in table_data[1:]:
        if not row or not row[0]:  # Skip empty rows or rows without primary identifier
            continue

        entity = {
            'name': normalizeText(row[0]),
            'attributes': {},
            'source': 'structured',
            'confidence': 0.9  # High confidence for structured data
        }

        # Extract attributes from remaining columns
        for i in range(1, min(len(headers), len(row))):
            if row[i]:  # Only add non-empty attributes
                attr_name = normalizeText(headers[i])
                entity['attributes'][attr_name] = row[i]

        entities.append(entity)

    # Infer relationships based on predefined patterns
    relationship_patterns = [
        {'column': 'Manager', 'relation': 'reports_to'},
        {'column': 'Supervisor', 'relation': 'reports_to'},
        {'column': 'Department', 'relation': 'belongs_to'},
        {'column': 'Location', 'relation': 'located_in'}
    ]

    for pattern in relationship_patterns:
        if pattern['column'] in headers:
            col_index = headers.index(pattern['column'])
            for entity in entities:
                if col_index < len(table_data[entities.index(entity) + 1]):
                    target_name = table_data[entities.index(entity) + 1][col_index]
                    if target_name:
                        target_entity = findEntityByName(entities, normalizeText(target_name))
                        if target_entity:
                            relationships.append({
                                'entity1': entity,
                                'entity2': target_entity,
                                'relation': pattern['relation'],
                                'confidence': 0.85
                            })

    return {'entities': entities, 'relationships': relationships}

function findEntityByName(entities, name):
    for entity in entities:
        if entity['name'] == name:
            return entity
    return None

function normalizeText(text):
    # Basic normalization: lowercase, remove extra spaces, handle common variations
    return text.lower().strip().replace('  ', ' ')
```

TODO: handle other structured data formats (e.g., JSON, XML)
### Unstructured Data
Entity extraction from unstructured text involves a multi-stage, rule-based pipeline that identifies entities, attributes, and relationships while incorporating confidence scoring mechanisms. The process uses linguistic analysis to extract meaningful components and applies iterative scoring to refine extraction quality through backpropagation and score boosting.

#### Algorithm Overview
The extraction algorithm processes unstructured text through the following stages:

1. **Text Preprocessing**: Tokenize the input text, perform part-of-speech (POS) tagging, and generate dependency parse trees to understand syntactic structure.
2. **Entity Candidate Identification**: Extract potential entities from nouns and noun phrases, assigning initial confidence scores based on linguistic features (e.g., proper nouns receive higher scores).
3. **Attribute Extraction**: Identify attributes associated with entities using adjectives, possessive constructions, and descriptive phrases.
4. **Relationship Extraction**: Infer relationships between entities based on syntactic patterns, semantic proximity, and predefined linguistic templates.
5. **Confidence Scoring and Refinement**: Apply initial confidence scores, then use iterative backpropagation to adjust scores based on entity interconnections and contextual boosting.

#### Pseudocode for Entity Extraction Algorithm

```pseudocode
function extractEntitiesFromUnstructuredText(text):
    # Stage 1: Preprocessing
    tokens = tokenize(text)
    pos_tags = posTag(tokens)
    dependency_tree = dependencyParse(tokens, pos_tags)

    # Stage 2: Entity Candidate Identification
    entity_candidates = []
    for i in range(len(tokens)):
        if pos_tags[i] in ['NN', 'NNS', 'NNP', 'NNPS']:  # Nouns and proper nouns
            candidate = {
                'text': tokens[i],
                'start_pos': i,
                'confidence': calculateInitialEntityConfidence(pos_tags[i], tokens, i)
            }
            entity_candidates.append(candidate)

    # Extract noun phrases using dependency tree
    noun_phrases = extractNounPhrases(dependency_tree, tokens, pos_tags)
    for phrase in noun_phrases:
        candidate = {
            'text': phrase['text'],
            'start_pos': phrase['start'],
            'confidence': 0.8  # Higher initial confidence for phrases
        }
        entity_candidates.append(candidate)

    # Stage 3: Attribute Extraction
    attributes = []
    for candidate in entity_candidates:
        candidate_attributes = extractAttributes(candidate, tokens, pos_tags, dependency_tree)
        attributes.extend(candidate_attributes)

    # Stage 4: Relationship Extraction
    relationships = []
    for i in range(len(entity_candidates)):
        for j in range(i+1, len(entity_candidates)):
            rel = extractRelationship(entity_candidates[i], entity_candidates[j], tokens, dependency_tree)
            if rel is not None:
                relationships.append(rel)

    # Stage 5: Confidence Scoring and Refinement
    # Initial scoring already done, now apply iterative refinement
    max_iterations = 10
    convergence_threshold = 0.01

    for iteration in range(max_iterations):
        previous_confidences = [c['confidence'] for c in entity_candidates]

        # Backpropagation: Adjust entity confidence based on connected relationships
        for entity in entity_candidates:
            connected_rels = [r for r in relationships if r['entity1'] == entity or r['entity2'] == entity]
            if connected_rels:
                avg_rel_confidence = sum(r['confidence'] for r in connected_rels) / len(connected_rels)
                entity['confidence'] = updateConfidence(entity['confidence'], avg_rel_confidence, 0.3)  # Backprop factor

        # Score boosting: Boost confidence for entities with multiple relationships
        relationship_counts = {}
        for rel in relationships:
            relationship_counts[rel['entity1']] = relationship_counts.get(rel['entity1'], 0) + 1
            relationship_counts[rel['entity2']] = relationship_counts.get(rel['entity2'], 0) + 1

        for entity in entity_candidates:
            rel_count = relationship_counts.get(entity, 0)
            if rel_count > 1:
                boost_factor = min(1.0, 0.1 * rel_count)  # Boost up to 10% per additional relationship
                entity['confidence'] = min(1.0, entity['confidence'] * (1 + boost_factor))

        # Check for convergence
        current_confidences = [c['confidence'] for c in entity_candidates]
        if max(abs(a - b) for a, b in zip(previous_confidences, current_confidences)) < convergence_threshold:
            break

    # Filter low-confidence extractions
    high_confidence_entities = [e for e in entity_candidates if e['confidence'] > 0.5]
    high_confidence_relationships = [r for r in relationships if r['confidence'] > 0.5]

    return {
        'entities': high_confidence_entities,
        'attributes': attributes,
        'relationships': high_confidence_relationships
    }

function calculateInitialEntityConfidence(pos_tag, tokens, position):
    base_confidence = 0.6
    if pos_tag in ['NNP', 'NNPS']:  # Proper nouns
        base_confidence += 0.3
    if position == 0:  # Sentence start
        base_confidence += 0.1
    # Add context-based adjustments (e.g., capitalization, surrounding words)
    return min(1.0, base_confidence)

function extractAttributes(entity, tokens, pos_tags, dependency_tree):
    attributes = []
    # Find adjectives modifying the entity
    for i in range(len(tokens)):
        if pos_tags[i] in ['JJ', 'JJR', 'JJS'] and isModifierOf(i, entity['start_pos'], dependency_tree):
            attributes.append({
                'entity': entity,
                'attribute': tokens[i],
                'type': 'descriptive',
                'confidence': 0.7
            })
    return attributes

function extractRelationship(entity1, entity2, tokens, dependency_tree):
    # Check for common relationship patterns
    patterns = [
        {'words': ['works', 'for'], 'relation': 'employment', 'confidence': 0.8},
        {'words': ['located', 'in'], 'relation': 'location', 'confidence': 0.7},
        {'words': ['part', 'of'], 'relation': 'composition', 'confidence': 0.75}
    ]

    text_between = getTextBetweenEntities(entity1, entity2, tokens)
    for pattern in patterns:
        if all(word in text_between for word in pattern['words']):
            return {
                'entity1': entity1,
                'entity2': entity2,
                'relation': pattern['relation'],
                'confidence': pattern['confidence']
            }
    return None

function updateConfidence(current_confidence, influence_confidence, backprop_factor):
    # Weighted average for backpropagation
    return current_confidence * (1 - backprop_factor) + influence_confidence * backprop_factor
```

This algorithm provides a deterministic, rule-based approach to entity extraction while incorporating confidence scoring mechanisms. The iterative refinement process allows for score boosting based on entity interconnectedness and backpropagation of confidence through relationships, improving overall extraction quality without relying on machine learning models.

## Final pass: Entity Reconciliation
After extracting entities from both structured and unstructured data, a final reconciliation pass is performed to merge duplicate entities and resolve conflicts. This involves comparing entity names, attributes, and relationships to identify potential matches and applying deterministic rules to merge them into a single, unified entity representation. This ensures that the final ontology is coherent and free of redundancies, providing a solid foundation for further ontology development and reasoning.

#### Algorithm Overview
The reconciliation process follows these steps:

1. **Entity Grouping**: Group entities from both sources by normalized name similarity to identify potential duplicates.
2. **Entity Merging**: For each group, merge entities into a single representation using deterministic rules (e.g., prefer higher confidence entities, average confidences, resolve attribute conflicts).
3. **Relationship Reconciliation**: Update relationship references to point to merged entities and deduplicate relationships based on entity pairs and relation types.
4. **Attribute Assignment**: Assign extracted attributes from unstructured data to the appropriate reconciled entities.

#### Pseudocode for Entity Reconciliation

```pseudocode
function reconcileEntities(structured_results, unstructured_results):
    # Combine all extracted data
    all_entities = structured_results['entities'] + unstructured_results['entities']
    all_relationships = structured_results['relationships'] + unstructured_results['relationships']
    all_attributes = unstructured_results.get('attributes', [])

    # Step 1: Group entities by normalized name
    entity_groups = groupEntitiesByNormalizedName(all_entities)

    # Step 2: Merge entity groups
    reconciled_entities = []
    entity_mapping = {}  # Maps original entities to their reconciled versions

    for group in entity_groups:
        merged_entity = mergeEntityGroup(group)
        reconciled_entities.append(merged_entity)
        for original_entity in group:
            entity_mapping[original_entity] = merged_entity

    # Step 3: Reconcile relationships
    reconciled_relationships = []
    seen_relationships = set()  # To deduplicate relationships

    for relationship in all_relationships:
        # Map relationship entities to reconciled entities
        reconciled_entity1 = entity_mapping.get(relationship['entity1'], relationship['entity1'])
        reconciled_entity2 = entity_mapping.get(relationship['entity2'], relationship['entity2'])

        # Create a unique key for deduplication
        rel_key = (reconciled_entity1['name'], reconciled_entity2['name'], relationship['relation'])

        if rel_key not in seen_relationships:
            reconciled_relationship = {
                'entity1': reconciled_entity1,
                'entity2': reconciled_entity2,
                'relation': relationship['relation'],
                'confidence': relationship['confidence']
            }
            reconciled_relationships.append(reconciled_relationship)
            seen_relationships.add(rel_key)

    # Step 4: Assign attributes to reconciled entities
    for attribute in all_attributes:
        reconciled_entity = entity_mapping.get(attribute['entity'], attribute['entity'])
        if reconciled_entity:
            attr_name = attribute['attribute']
            # Only add if not already present to avoid conflicts
            if attr_name not in reconciled_entity.get('attributes', {}):
                reconciled_entity.setdefault('attributes', {})[attr_name] = attribute

    return {
        'entities': reconciled_entities,
        'relationships': reconciled_relationships
    }

function groupEntitiesByNormalizedName(entities):
    groups = {}
    for entity in entities:
        normalized_name = normalizeEntityName(entity['name'])
        if normalized_name not in groups:
            groups[normalized_name] = []
        groups[normalized_name].append(entity)
    return list(groups.values())

function mergeEntityGroup(entity_group):
    if len(entity_group) == 1:
        return entity_group[0]

    # Select primary entity (highest confidence, prefer structured data)
    primary_entity = max(entity_group, key=lambda e: (
        e.get('confidence', 0),
        1 if e.get('source') == 'structured' else 0
    ))

    # Merge attributes
    merged_attributes = {}
    for entity in entity_group:
        for attr_name, attr_value in entity.get('attributes', {}).items():
            if attr_name not in merged_attributes:
                merged_attributes[attr_name] = attr_value
            elif not merged_attributes[attr_name] and attr_value:
                # Prefer non-empty values
                merged_attributes[attr_name] = attr_value
            # If both have values and differ, keep the primary entity's value

    # Calculate average confidence
    total_confidence = sum(e.get('confidence', 0) for e in entity_group)
    avg_confidence = total_confidence / len(entity_group)

    # Create merged entity
    merged_entity = {
        'name': primary_entity['name'],
        'attributes': merged_attributes,
        'confidence': avg_confidence,
        'sources': list(set(e.get('source', 'unknown') for e in entity_group))
    }

    return merged_entity

function normalizeEntityName(name):
    # Comprehensive normalization for entity name matching
    normalized = name.lower().strip()
    # Remove common punctuation and articles
    normalized = normalized.replace(',', '').replace('.', '').replace('the ', '').replace('a ', '').replace('an ', '')
    # Handle common abbreviations or variations
    normalized = normalized.replace('&', 'and').replace('corp', 'corporation').replace('inc', 'incorporated')
    # Remove extra spaces
    normalized = ' '.join(normalized.split())
    return normalized
```
