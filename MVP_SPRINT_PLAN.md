# Mimir AIP: 2-3 Week MVP Sprint Plan
## "Make It Autonomous" Edition

**Goal:** Demonstrate end-to-end autonomous pipeline in 2-3 weeks
**Team:** Solo developer + AI agents (Claude, GitHub Copilot, etc.)
**Strategy:** Focus HARD on integration gaps, leverage agents for boilerplate

---

## 🎯 MVP Definition: "The Autonomous Flow Demo"

**What We'll Build:**
```
User uploads CSV with customer data
    ↓ (automated)
Pipeline ingests and processes data
    ↓ (automated)
Entities/relationships extracted to ontology
    ↓ (automated)
ML model trained on ontology data
    ↓ (automated)
Digital twin created with model
    ↓ (automated)
Anomaly detected in new data
    ↓ (automated)
Alert pipeline sends email notification
```

**Zero manual steps after initial CSV upload!**

---

## 🚀 WEEK 1: The Integration Layer (Days 1-7)

### Day 1-2: Event Bus Foundation
**Goal:** Create event-driven architecture backbone

**Tasks:**
1. **Create Event Bus System** (`utils/event_bus.go`)
   ```go
   type EventBus struct {
       handlers map[string][]EventHandler
       mu       sync.RWMutex
   }

   type Event struct {
       Type      string
       Source    string
       Payload   map[string]any
       Timestamp time.Time
   }

   type EventHandler func(Event) error

   func (eb *EventBus) Publish(event Event)
   func (eb *EventBus) Subscribe(eventType string, handler EventHandler)
   ```

2. **Add Event Emitters**
   - Pipeline completion: `event_bus.Publish(Event{Type: "pipeline.completed"})`
   - Monitoring rules: `event_bus.Publish(Event{Type: "anomaly.detected"})`
   - Model training: `event_bus.Publish(Event{Type: "model.trained"})`

3. **Basic Event Handlers**
   - `OnPipelineComplete` → Trigger extraction job
   - `OnAnomalyDetected` → Execute alert pipeline
   - `OnModelTrained` → Create digital twin

**Agent Help:**
- Generate event bus boilerplate
- Create event handler templates
- Write unit tests for event system

**Success Criteria:**
- ✅ Event published in one component received in another
- ✅ Unit tests pass for event bus

---

### Day 3-4: Pipeline → Ontology Auto-Extraction
**Goal:** Pipeline completion automatically triggers entity extraction

**Tasks:**
1. **Add Pipeline Completion Hook** (modify `pipelines/pipeline.go`)
   ```go
   func (p *Pipeline) Execute(input Data) (Data, error) {
       // ... existing code ...

       // After successful execution:
       if p.AutoExtractEntities && p.TargetOntologyID != "" {
           go triggerExtractionJob(p.ID, p.TargetOntologyID, output)
       }

       eventBus.Publish(Event{
           Type: "pipeline.completed",
           Payload: map[string]any{
               "pipeline_id": p.ID,
               "output": output,
           },
       })

       return output, nil
   }
   ```

2. **Auto-Extraction Job Creator** (`handlers_extraction.go`)
   ```go
   func triggerExtractionJob(pipelineID, ontologyID string, data Data) {
       job := ExtractionJob{
           OntologyID: ontologyID,
           SourceType: "pipeline",
           SourceID:   pipelineID,
           Method:     "hybrid", // Use best extraction method
           Status:     "pending",
       }

       createExtractionJob(job)
       executeExtractionJob(job.ID) // Execute immediately
   }
   ```

3. **Add UI Toggle** (frontend: `src/app/pipelines/[id]/page.tsx`)
   - Checkbox: "Auto-extract entities to ontology"
   - Dropdown: Select target ontology

**Agent Help:**
- Modify pipeline execution to add hooks
- Generate extraction job creation code
- Create UI components for configuration

**Success Criteria:**
- ✅ Pipeline runs → Extraction job auto-created
- ✅ Entities appear in ontology after pipeline completes

---

### Day 5-7: Anomaly → Pipeline Trigger
**Goal:** Detected anomalies automatically execute response pipelines

**Tasks:**
1. **Alert Actions Table** (database schema in `pipelines/Storage/persistence.go`)
   ```go
   type AlertAction struct {
       ID              string
       RuleID          string    // Which monitoring rule triggers this
       ActionType      string    // "execute_pipeline", "send_email", "webhook"
       PipelineID      string    // Pipeline to execute (if action_type = execute_pipeline)
       EmailRecipients []string  // For email actions
       WebhookURL      string    // For webhook actions
       Enabled         bool
   }
   ```

2. **Alert Action Executor** (`pipelines/ML/monitoring_rules.go`)
   ```go
   func (mr *MonitoringRules) CheckRule(rule MonitoringRule, value float64) {
       if rule.IsViolated(value) {
           alert := CreateAlert(rule, value)

           // Trigger all actions for this rule
           actions := GetAlertActionsForRule(rule.ID)
           for _, action := range actions {
               go executeAlertAction(action, alert)
           }
       }
   }

   func executeAlertAction(action AlertAction, alert Alert) error {
       switch action.ActionType {
       case "execute_pipeline":
           // Execute the response pipeline
           executePipeline(action.PipelineID, map[string]any{
               "alert_id": alert.ID,
               "rule_id": alert.RuleID,
               "value": alert.Value,
           })
       case "send_email":
           sendEmailAlert(action.EmailRecipients, alert)
       case "webhook":
           postToWebhook(action.WebhookURL, alert)
       }
   }
   ```

3. **Simple Email Sender** (`utils/notifications.go`)
   ```go
   func sendEmailAlert(recipients []string, alert Alert) error {
       // Use standard Go smtp package
       // Read SMTP config from env vars
       subject := fmt.Sprintf("Alert: %s", alert.RuleName)
       body := fmt.Sprintf("Anomaly detected!\nValue: %v\nThreshold: %v",
           alert.Value, alert.Threshold)

       return sendEmail(recipients, subject, body)
   }
   ```

4. **UI for Alert Actions** (frontend: `src/app/monitoring/rules/[id]/page.tsx`)
   - Add "Actions" tab to monitoring rules
   - Configure: "When this rule triggers, execute pipeline X"
   - Configure: "Send email to: user@example.com"

**Agent Help:**
- Generate database migration for alert_actions table
- Create SMTP email sender (lots of boilerplate)
- Build UI for action configuration

**Success Criteria:**
- ✅ Anomaly detected → Pipeline executes automatically
- ✅ Anomaly detected → Email sent
- ✅ UI allows configuring alert actions

**🎉 END OF WEEK 1 MILESTONE:**
- Events flow through system
- Pipelines auto-trigger extraction
- Anomalies auto-trigger responses

---

## ⚡ WEEK 2: The Autonomous ML Pipeline (Days 8-14)

### Day 8-10: Ontology → Training Data Extraction
**Goal:** ML can extract training data directly from knowledge graph

**Tasks:**
1. **Training Data Extractor** (`pipelines/ML/training_data_extractor.go`)
   ```go
   type TrainingDataExtractor struct {
       rdfStore *RDFStore
   }

   func (tde *TrainingDataExtractor) ExtractDataset(
       ontologyID string,
       targetProperty string,
   ) (*Dataset, error) {
       // 1. Query for all entities with target property
       query := fmt.Sprintf(`
           SELECT ?entity ?%s ?feature1 ?feature2 ...
           WHERE {
               ?entity rdf:type ?type ;
                      :%s ?%s ;
                      :feature1 ?feature1 ;
                      :feature2 ?feature2 .
           }
       `, targetProperty, targetProperty, targetProperty)

       results := tde.rdfStore.Query(query)

       // 2. Convert SPARQL results to training dataset
       dataset := &Dataset{
           Features: [][]float64{},
           Target:   []float64{},
           FeatureNames: []string{},
       }

       for _, row := range results {
           features := extractFeatures(row)
           target := extractTarget(row, targetProperty)

           dataset.Features = append(dataset.Features, features)
           dataset.Target = append(dataset.Target, target)
       }

       return dataset, nil
   }

   func extractFeatures(row map[string]any) []float64 {
       // Convert RDF values to numeric features
       // Handle: strings → encoding, dates → timestamps, etc.
   }
   ```

2. **Feature Engineering from Relationships**
   ```go
   func (tde *TrainingDataExtractor) ExtractRelationshipFeatures(
       entityURI string,
   ) map[string]float64 {
       // Example: Count relationships of each type
       query := fmt.Sprintf(`
           SELECT ?predicate (COUNT(?object) AS ?count)
           WHERE {
               <%s> ?predicate ?object .
           }
           GROUP BY ?predicate
       `, entityURI)

       // Returns: {"hasProduct": 5, "hasOrder": 10, ...}
       // These become features for ML
   }
   ```

3. **Update AutoTrainer** (modify `pipelines/ML/auto_trainer.go`)
   ```go
   func (at *AutoTrainer) TrainFromOntology(
       ontologyID string,
       targetProperty string,
   ) (*Model, error) {
       // OLD: Requires manual CSV upload
       // NEW: Auto-extract from ontology

       extractor := NewTrainingDataExtractor(at.rdfStore)
       dataset, err := extractor.ExtractDataset(ontologyID, targetProperty)
       if err != nil {
           return nil, err
       }

       // Validate dataset quality
       if len(dataset.Features) < 100 {
           return nil, errors.New("insufficient training data")
       }

       // Train model
       model := at.trainModel(dataset)

       // Emit event
       eventBus.Publish(Event{
           Type: "model.trained",
           Payload: map[string]any{
               "model_id": model.ID,
               "ontology_id": ontologyID,
           },
       })

       return model, nil
   }
   ```

**Agent Help:**
- Generate SPARQL query templates
- Create feature extraction logic
- Handle data type conversions

**Success Criteria:**
- ✅ Click "Train Model" → No CSV upload needed
- ✅ Model trained using knowledge graph data
- ✅ Model accuracy reasonable (>60% for demo)

---

### Day 11-12: Model → Digital Twin Integration
**Goal:** Trained models automatically create digital twins

**Tasks:**
1. **Auto-Twin Creator** (`handlers_digital_twin.go`)
   ```go
   func createTwinFromModel(modelID string) (*DigitalTwin, error) {
       model := getModel(modelID)

       // Query ontology for entities related to model
       entities := queryRelatedEntities(model.OntologyID, model.TargetProperty)

       // Create digital twin with initial state from entities
       twin := &DigitalTwin{
           Name:       fmt.Sprintf("Twin for %s", model.Name),
           ModelID:    modelID,
           State:      buildInitialState(entities),
           Entities:   entities,
           CreatedAt:  time.Now(),
       }

       saveTwin(twin)

       // Set up anomaly detection rules for this twin
       createAnomalyRulesForTwin(twin, model)

       return twin, nil
   }
   ```

2. **Event Handler for Model Training**
   ```go
   // In event_bus initialization
   eventBus.Subscribe("model.trained", func(event Event) error {
       modelID := event.Payload["model_id"].(string)

       // Auto-create digital twin
       twin, err := createTwinFromModel(modelID)
       if err != nil {
           return err
       }

       log.Printf("Auto-created digital twin %s for model %s", twin.ID, modelID)
       return nil
   })
   ```

3. **Anomaly Detection Rules for Twins**
   ```go
   func createAnomalyRulesForTwin(twin *DigitalTwin, model *Model) {
       // Create monitoring rule that checks if twin state
       // deviates significantly from model predictions

       rule := MonitoringRule{
           Name:       fmt.Sprintf("Anomaly detection for %s", twin.Name),
           TwinID:     twin.ID,
           RuleType:   "z_score",
           Threshold:  3.0, // 3 standard deviations
           Enabled:    true,
       }

       saveMonitoringRule(rule)

       // Set up alert action: Send email on anomaly
       action := AlertAction{
           RuleID:          rule.ID,
           ActionType:      "send_email",
           EmailRecipients: []string{"admin@example.com"},
           Enabled:         true,
       }

       saveAlertAction(action)
   }
   ```

**Agent Help:**
- Generate twin creation from model metadata
- Create initial state building logic
- Write anomaly rule templates

**Success Criteria:**
- ✅ Model trains → Digital twin auto-created
- ✅ Twin has monitoring rules configured
- ✅ Twin state reflects model predictions

---

### Day 13-14: Agent Tools Fix (Top 5 Priority)
**Goal:** Make agent actually useful for orchestration

**Focus on 5 most critical tools:**

1. **`pipeline.create`** - Create new pipeline
   ```go
   func (ms *MCPServer) handlePipelineCreate(params map[string]any) (any, error) {
       // Parse pipeline config from params
       config := parsePipelineConfig(params)

       // Create pipeline via internal API
       pipeline := createPipeline(config)

       return map[string]any{
           "pipeline_id": pipeline.ID,
           "message": "Pipeline created successfully",
       }, nil
   }
   ```

2. **`pipeline.execute`** - Run a pipeline
   ```go
   func (ms *MCPServer) handlePipelineExecute(params map[string]any) (any, error) {
       pipelineID := params["pipeline_id"].(string)

       // Execute pipeline
       output, err := executePipeline(pipelineID, params["input"])

       return map[string]any{
           "status": "success",
           "output": output,
       }, nil
   }
   ```

3. **`model.train`** - Train ML model
   ```go
   func (ms *MCPServer) handleModelTrain(params map[string]any) (any, error) {
       ontologyID := params["ontology_id"].(string)
       targetProperty := params["target_property"].(string)

       // Trigger model training
       model, err := autoTrainer.TrainFromOntology(ontologyID, targetProperty)

       return map[string]any{
           "model_id": model.ID,
           "accuracy": model.Accuracy,
       }, nil
   }
   ```

4. **`ontology.query`** - Query knowledge graph
   ```go
   func (ms *MCPServer) handleOntologyQuery(params map[string]any) (any, error) {
       query := params["query"].(string)

       // Execute SPARQL query
       results := rdfStore.Query(query)

       return map[string]any{
           "results": results,
           "count": len(results),
       }, nil
   }
   ```

5. **`twin.simulate`** - Run simulation
   ```go
   func (ms *MCPServer) handleTwinSimulate(params map[string]any) (any, error) {
       twinID := params["twin_id"].(string)
       scenario := parseScenario(params["scenario"])

       // Run simulation
       result := runSimulation(twinID, scenario)

       return map[string]any{
           "outcome": result.FinalState,
           "impact": result.ImpactAnalysis,
       }, nil
   }
   ```

**Agent Help:**
- Replace stub implementations with real code
- Add parameter validation
- Create response formatting

**Success Criteria:**
- ✅ Agent can create pipeline via chat
- ✅ Agent can train model via chat
- ✅ Agent can query ontology via chat
- ✅ Agent tools actually execute (not stubs)

**🎉 END OF WEEK 2 MILESTONE:**
- ML trains on ontology data (no CSV!)
- Models auto-create digital twins
- Agent has working tools

---

## 🏁 WEEK 3: Integration & Demo (Days 15-21)

### Day 15-16: Fix Frontend & E2E Tests
**Goal:** Get frontend working so we can demo

**Tasks:**
1. **Fix Page Crashes** (currently preventing E2E tests)
   - Investigate Next.js runtime errors
   - Fix font loading issues
   - Ensure all pages load without crashing

2. **Run E2E Tests**
   - Execute all 12 E2E test files
   - Fix any failures
   - Verify UI works end-to-end

3. **Add Missing UI Components**
   - Alert actions configuration UI
   - Pipeline auto-extraction toggle
   - Digital twin anomaly detection display

**Agent Help:**
- Debug Next.js errors
- Fix failing E2E tests
- Generate missing UI components

---

### Day 17-18: End-to-End Integration Test
**Goal:** Prove the autonomous flow works

**The Demo Scenario:**
1. **Upload customer churn dataset** (CSV with customer data)
2. **Create ingestion pipeline** (via UI or agent)
3. **Enable auto-extraction** (toggle on pipeline settings)
4. **Wait for automation:**
   - Pipeline processes CSV → Stores in DB
   - Extraction job auto-created → Entities in ontology
   - Click "Train Model" → Model trains on ontology data
   - Digital twin auto-created → Monitoring rules set up
5. **Simulate new data with anomaly**
   - High churn risk customer appears
   - Anomaly detected → Email sent automatically

**Tasks:**
- Test each step manually
- Fix any integration bugs
- Ensure zero manual intervention after step 3

---

### Day 19-20: Documentation & Demo Prep
**Goal:** Make it demonstrable

**Tasks:**
1. **Create Demo Video** (5 minutes)
   - Show: "Upload CSV → Everything happens automatically"
   - Narrate: Explain what's happening autonomously
   - Show: Email alert triggered by anomaly

2. **Update README.md**
   - Add "Autonomous Features" section
   - List what's automated vs. manual
   - Include demo video link

3. **Write Quick Start Guide**
   - "Set up your first autonomous pipeline in 10 minutes"
   - Step-by-step with screenshots
   - Example datasets provided

4. **Create Architecture Diagram**
   - Show event flow through system
   - Highlight autonomous components
   - Explain integration points

**Agent Help:**
- Generate documentation from code
- Create architecture diagrams (Mermaid)
- Write tutorial content

---

### Day 21: Polish & Buffer
**Goal:** Handle unexpected issues, polish rough edges

**Tasks:**
- Fix any critical bugs found during testing
- Improve error messages
- Add loading states for async operations
- Performance optimization if needed

---

## 🎯 MVP Success Criteria

At the end of 3 weeks, you should be able to:

### ✅ Demo Autonomous Flow
1. Upload customer CSV
2. Create pipeline (optionally via agent: "Ingest this CSV for me")
3. Enable auto-extraction
4. **Everything else is automatic:**
   - Data processed
   - Entities extracted
   - Model trained
   - Twin created
   - Anomaly detected
   - Email sent

### ✅ Working Agent
- Agent can create pipelines
- Agent can train models
- Agent can query ontology
- Agent can run simulations
- Agent provides useful responses (not stubs)

### ✅ Demonstrable
- 5-minute demo video
- Documentation explaining autonomous features
- Quick start guide for new users
- Working E2E tests

---

## 🤖 How to Leverage AI Agents Maximally

### For Code Generation (80% of your time)
**Claude/GPT-4:**
```
Prompt Examples:
- "Generate event bus system in Go with pub/sub pattern"
- "Create SPARQL query to extract training data for property X"
- "Build React component for alert action configuration"
- "Write unit tests for pipeline completion hook"
```

**GitHub Copilot:**
- Let it autocomplete boilerplate
- Use for test generation
- Generate type definitions

### For Architecture Decisions (10% of your time)
**Claude:**
```
- "How should I structure event handlers?"
- "Best way to convert SPARQL results to ML features?"
- "Database schema for alert actions?"
```

### For Debugging (10% of your time)
**Claude with error logs:**
```
- "Why is my Next.js page crashing? Here's the error:"
- "SPARQL query not returning results. Query: ..."
- "Event not being received. Code: ..."
```

---

## 🚨 What to CUT / DEFER

**Not in MVP (do later):**
- ❌ Multiple ML algorithms (Decision Tree is fine)
- ❌ Hyperparameter tuning (use defaults)
- ❌ Complex feature engineering (basic features OK)
- ❌ Model performance monitoring
- ❌ A/B testing
- ❌ Distributed execution
- ❌ Advanced security features
- ❌ All Phase 4 features (scalability, lineage, etc.)

**MVP Focus:**
- ✅ Events flow between components
- ✅ Pipeline → Ontology (automated)
- ✅ Ontology → Model training (automated)
- ✅ Anomaly → Action (automated)
- ✅ Agent tools work (not stubs)

---

## 📅 Daily Checklist

### Every Day:
- [ ] Commit and push changes (work in public!)
- [ ] Update this doc with progress
- [ ] Test integration between components
- [ ] Ask agents for help when stuck (don't spin wheels!)

### Red Flags (Get help immediately if):
- Stuck on a problem for >2 hours
- Integration not working after multiple attempts
- Unclear how to implement a feature
- Performance issues blocking progress

---

## 🎉 Success Looks Like:

**Week 1 End:**
- Can trigger pipeline from monitoring rule ✅
- Can auto-extract entities from pipeline ✅
- Can send email on anomaly ✅

**Week 2 End:**
- Can train model without CSV upload ✅
- Can create digital twin from model ✅
- Agent tools work for 5 core operations ✅

**Week 3 End:**
- Full demo scenario works end-to-end ✅
- Documentation shows autonomous features ✅
- Video demonstrates the vision ✅

---

## 🚀 START HERE (Day 1 Morning)

1. **Read this entire document**
2. **Set up development environment**
   - Ensure Go, Node.js, PostgreSQL running
   - Run existing tests to confirm setup
3. **Create Day 1 branch: `feature/event-bus-foundation`**
4. **Prompt Claude:**
   ```
   "I need to create an event bus system in Go with pub/sub pattern.
   Here's my project structure: [paste key files]
   Help me implement the EventBus struct with Publish/Subscribe methods."
   ```
5. **Start coding!**

Let's make Mimir truly autonomous! 🤖✨

---

## Questions? Get Unstuck:

- Post in GitHub Discussions
- Ask Claude for specific implementation help
- Review existing code for patterns
- Check the original IMPLEMENTATION_ROADMAP.md for detailed specs

**Remember:** Progress > Perfection. Ship the MVP, iterate later!
