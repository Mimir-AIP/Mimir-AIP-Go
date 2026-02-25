# ML Models

ML models will be created in a semi-automated way, once mimir has automatically generated the ontology for a project based on the ingested data, it will then use this ontology to determine what data is available and how it is structured, based on a set of deterministic rules it will then recommend a model type best suited to the data, user can view this recommendation and either accept it or choose a different model type. Once the model type is chosen, mimir will then use the data which has been ingested and stored, seperately for training and testing, then begin the training process. During training the user can view various metrics and visualisations to monitor the training process, once training is complete the model will be stored and can be used for inference within the digital twin or for other purposes. Similar to the ontology, mimir will continuously monitor the ml model results and if it detects a significant drop in performance it will alert the user and recommend retraining the model with new data. Additionally if the ontology is upadated(wither via the automated process after new data ingestion or via manual editing) mimir will also check if the changes to the ontology have any impact on the ml model performance, if it does it will alert the user and recommend retraining the model with the updated ontology and any new data that has been ingested and stored.

## Model Types
Initially mimir will support the following model types:
- Decision Trees
- Random Forests
- Regression Models
- Neural Networks

## Model recommendation rules
Based on the ontology and data available for that ontology, mimir will recommend a model type based on the following rules:

The model recommendation system analyzes the project's ontology structure and ingested data characteristics to suggest the most appropriate ML model type from the available options: Decision Trees, Random Forests, Regression Models, and Neural Networks. The recommendation prioritizes model suitability based on data complexity, size, and structure derived from the ontology.

Key factors considered:
- **Ontology Complexity**: Number of entities (classes), attributes (datatype properties), and relationships (object properties). Higher complexity suggests more sophisticated models.
- **Data Types**: Prevalence of numerical vs. categorical attributes, as defined by OWL datatype ranges (e.g., xsd:int/float vs. xsd:string).
- **Data Volume**: Estimated dataset size from ingested data, influencing whether simple or scalable models are preferred.
- **Structural Patterns**: Presence of hierarchical relationships or graph-like structures in the ontology, which may benefit from ensemble or deep learning approaches.

Recommendation Logic:
1. **Decision Trees**: Recommended for small to medium datasets with mixed categorical/numerical data and moderate ontology complexity. Best for interpretable models where understanding feature importance is valuable.
2. **Random Forests**: Preferred for datasets with many categorical features, complex relationships, or when ensemble methods can improve accuracy over single trees. Suitable for medium to large datasets.
3. **Regression Models**: Selected when the ontology indicates primarily numerical attributes and predictive tasks focused on continuous outputs. Assumes linear or simple non-linear relationships.
4. **Neural Networks**: Recommended for large datasets, highly complex ontologies with many entities/relationships, or when capturing non-linear patterns is crucial. Also suitable for unstructured data components if present.

The system applies a scoring mechanism where each model type receives points based on matching criteria, with the highest-scoring model recommended. Users can override the recommendation if needed.

### Pseudocode Implementation
```pseudocode
function recommendModelType(ontology, dataSummary):
    # ontology: parsed OWL ontology with classes, properties, etc.
    # dataSummary: object with data size, types, etc. from ingested data

    # Extract ontology features
    numEntities = countClasses(ontology)
    numAttributes = countDatatypeProperties(ontology)
    numRelationships = countObjectProperties(ontology)
    dataTypes = analyzeDataTypes(ontology)  # {numerical: count, categorical: count}

    # Extract data features
    dataSize = dataSummary.size  # e.g., 'small', 'medium', 'large'
    hasUnstructured = dataSummary.hasUnstructured  # boolean

    # Initialize scores for each model type
    scores = {
        'DecisionTree': 0,
        'RandomForest': 0,
        'Regression': 0,
        'NeuralNetwork': 0
    }

    # Scoring rules based on ontology complexity
    if numEntities < 10 and numRelationships < 20:
        scores['DecisionTree'] += 2
    elif numEntities >= 10 and numRelationships >= 20:
        scores['RandomForest'] += 2
        scores['NeuralNetwork'] += 1

    # Scoring based on data types
    numericalRatio = dataTypes.numerical / (dataTypes.numerical + dataTypes.categorical)
    if numericalRatio > 0.7:
        scores['Regression'] += 3
        scores['NeuralNetwork'] += 1
    elif numericalRatio < 0.3:
        scores['DecisionTree'] += 2
        scores['RandomForest'] += 2

    # Scoring based on data size
    if dataSize == 'small':
        scores['DecisionTree'] += 2
    elif dataSize == 'medium':
        scores['RandomForest'] += 2
        scores['Regression'] += 1
    elif dataSize == 'large':
        scores['NeuralNetwork'] += 3
        scores['RandomForest'] += 1

    # Scoring for unstructured data
    if hasUnstructured:
        scores['NeuralNetwork'] += 2

    # Additional scoring for complex relationships
    if numRelationships > numEntities:
        scores['RandomForest'] += 1
        scores['NeuralNetwork'] += 1

    # Find the model with highest score
    recommendedModel = max(scores, key=scores.get)

    # Handle ties by preferring simpler models
    if scores[recommendedModel] == scores.get('DecisionTree', 0) and recommendedModel != 'DecisionTree':
        recommendedModel = 'DecisionTree'
    elif scores[recommendedModel] == scores.get('RandomForest', 0) and recommendedModel not in ['DecisionTree', 'RandomForest']:
        recommendedModel = 'RandomForest'

    return recommendedModel

function countClasses(ontology):
    # Count owl:Class declarations
    return len([c for c in ontology.classes])

function countDatatypeProperties(ontology):
    # Count owl:DatatypeProperty declarations
    return len([p for p in ontology.properties if p.type == 'DatatypeProperty'])

function countObjectProperties(ontology):
    # Count owl:ObjectProperty declarations
    return len([p for p in ontology.properties if p.type == 'ObjectProperty'])

function analyzeDataTypes(ontology):
    numerical = 0
    categorical = 0
    for prop in ontology.properties:
        if prop.type == 'DatatypeProperty':
            rangeType = prop.range
            if rangeType in ['xsd:int', 'xsd:float', 'xsd:double']:
                numerical += 1
            else:
                categorical += 1
    return {'numerical': numerical, 'categorical': categorical}
```

## Model Training and Monitoring
Once recommendation complete and user opts to proceed with either it or a different model type, mimir will handle the training process. It will split the ingested data into training and testing sets, then train the selected model type. During training, mimir will provide real-time metrics such as accuracy, loss, precision, recall, etc., depending on the model type and task. Visualizations like learning curves, feature importance plots, or confusion matrices may also be available. 

After training, the model will be stored for inference. Mimir will continuously monitor model performance in production and alert the user if performance degrades, recommending retraining with new data or updated ontology as needed.