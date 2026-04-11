package api

import (
	"fmt"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/queue"
)

const recentFailureWindow = time.Hour

type ProjectStateProvider struct {
	store metadatastore.MetadataStore
	queue *queue.Queue
}

func NewProjectStateProvider(store metadatastore.MetadataStore, q *queue.Queue) *ProjectStateProvider {
	return &ProjectStateProvider{store: store, queue: q}
}

func (p *ProjectStateProvider) Summary(projectID string) (*models.ProjectStateSummary, error) {
	if projectID == "" {
		return nil, fmt.Errorf("project id is required")
	}
	if _, err := p.store.GetProject(projectID); err != nil {
		return nil, err
	}

	pipelines, err := p.store.ListPipelinesByProject(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list project pipelines: %w", err)
	}
	ontologies, err := p.store.ListOntologiesByProject(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list project ontologies: %w", err)
	}
	modelsList, err := p.store.ListMLModelsByProject(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list project ml models: %w", err)
	}
	twins, err := p.store.ListDigitalTwinsByProject(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list project digital twins: %w", err)
	}
	storageConfigs, err := p.store.ListStorageConfigsByProject(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list project storage configs: %w", err)
	}
	reviews, err := p.store.ListReviewItems(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list project review items: %w", err)
	}
	insights, err := p.store.ListInsightsByProject(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list project insights: %w", err)
	}
	plugins, err := p.store.ListPlugins()
	if err != nil {
		return nil, fmt.Errorf("failed to list plugins: %w", err)
	}
	tasks, err := p.queue.ListWorkTasks()
	if err != nil {
		return nil, fmt.Errorf("failed to list work tasks: %w", err)
	}
	queueLength, err := p.queue.QueueLength()
	if err != nil {
		return nil, fmt.Errorf("failed to read queue length: %w", err)
	}

	pipelineByID := make(map[string]*models.Pipeline, len(pipelines))
	for _, pipeline := range pipelines {
		pipelineByID[pipeline.ID] = pipeline
	}

	now := time.Now().UTC()
	activeTasks := 0
	recentFailedTasks := 0
	activePipelineTasks := 0
	activeIngestionTasks := 0
	recentFailedPipelineTasks := 0
	activeMLTasks := 0
	recentFailedMLTasks := 0
	activeTwinTasks := 0
	recentFailedTwinTasks := 0
	for _, task := range tasks {
		if task.ProjectID != projectID {
			continue
		}
		if isActiveTask(task.Status) {
			activeTasks++
		}
		if isRecentFailure(task, now) {
			recentFailedTasks++
		}
		switch task.Type {
		case models.WorkTaskTypePipelineExecution:
			if isActiveTask(task.Status) {
				activePipelineTasks++
				if pipeline := pipelineByID[task.TaskSpec.PipelineID]; pipeline != nil && pipeline.Type == models.PipelineTypeIngestion {
					activeIngestionTasks++
				}
			}
			if isRecentFailure(task, now) {
				recentFailedPipelineTasks++
			}
		case models.WorkTaskTypeMLTraining:
			if isActiveTask(task.Status) {
				activeMLTasks++
			}
			if isRecentFailure(task, now) {
				recentFailedMLTasks++
			}
		case models.WorkTaskTypeDigitalTwinProcessing:
			if isActiveTask(task.Status) {
				activeTwinTasks++
			}
			if isRecentFailure(task, now) {
				recentFailedTwinTasks++
			}
		}
	}

	pendingReviews := 0
	for _, review := range reviews {
		if review.Status == models.ReviewItemStatusPending {
			pendingReviews++
		}
	}

	activeOntologies := 0
	ontologiesInProgress := 0
	for _, ontology := range ontologies {
		switch ontology.Status {
		case "active":
			activeOntologies++
		case "draft":
			ontologiesInProgress++
		}
	}

	degradedModels := 0
	failedModels := 0
	for _, model := range modelsList {
		switch model.Status {
		case models.ModelStatusDegraded:
			degradedModels++
		case models.ModelStatusFailed:
			failedModels++
		}
	}

	pendingApprovals := 0
	twinsSyncing := 0
	twinsErrored := 0
	for _, twin := range twins {
		switch twin.Status {
		case "syncing":
			twinsSyncing++
		case "error":
			twinsErrored++
		}
		alerts, err := p.store.ListAlertEventsByDigitalTwin(twin.ID, 200)
		if err != nil {
			return nil, fmt.Errorf("failed to list alert events for digital twin %s: %w", twin.ID, err)
		}
		for _, alert := range alerts {
			if alert.ApprovalStatus == models.AlertApprovalStatusPending {
				pendingApprovals++
			}
		}
	}

	sections := map[string]models.ProjectSectionState{
		"Projects": {
			Status: models.ProjectSectionStateComplete,
			Detail: "Project loaded",
			Count:  1,
		},
		"Pipelines":         summarizeTaskBackedSection(len(pipelines), activePipelineTasks, recentFailedPipelineTasks, "pipeline", "pipelines configured"),
		"Ontologies":        summarizeOntologySection(len(ontologies), activeOntologies, ontologiesInProgress),
		"ML Models":         summarizeMLSection(len(modelsList), activeMLTasks, recentFailedMLTasks, degradedModels, failedModels),
		"Digital Twins":     summarizeTwinSection(len(twins), activeTwinTasks, recentFailedTwinTasks, pendingApprovals, twinsSyncing, twinsErrored),
		"Storage":           summarizeStorageSection(len(storageConfigs), activeIngestionTasks),
		"Insights & Review": summarizeInsightsSection(len(insights), pendingReviews),
		"Plugins":           summarizeStaticSection(len(plugins), "plugins installed"),
		"Work Queue":        summarizeQueueSection(len(tasks), queueLength, activeTasks, recentFailedTasks),
	}

	return &models.ProjectStateSummary{
		ProjectID:   projectID,
		GeneratedAt: now,
		QueueLength: queueLength,
		ActiveTasks: activeTasks,
		Sections:    sections,
	}, nil
}

func isActiveTask(status models.WorkTaskStatus) bool {
	switch status {
	case models.WorkTaskStatusQueued, models.WorkTaskStatusScheduled, models.WorkTaskStatusSpawned, models.WorkTaskStatusExecuting:
		return true
	default:
		return false
	}
}

func isRecentFailure(task *models.WorkTask, now time.Time) bool {
	if task == nil {
		return false
	}
	switch task.Status {
	case models.WorkTaskStatusFailed, models.WorkTaskStatusTimeout, models.WorkTaskStatusCancelled:
		at := task.SubmittedAt
		if task.CompletedAt != nil {
			at = *task.CompletedAt
		}
		return now.Sub(at) <= recentFailureWindow
	default:
		return false
	}
}

func summarizeTaskBackedSection(total, active, failed int, singular, completeDetail string) models.ProjectSectionState {
	if failed > 0 {
		return models.ProjectSectionState{Status: models.ProjectSectionStateError, Detail: fmt.Sprintf("%d recent %s failure(s)", failed, singular), Count: failed}
	}
	if active > 0 {
		return models.ProjectSectionState{Status: models.ProjectSectionStateInProgress, Detail: fmt.Sprintf("%d active %s task(s)", active, singular), Count: active}
	}
	if total > 0 {
		return models.ProjectSectionState{Status: models.ProjectSectionStateComplete, Detail: fmt.Sprintf("%d %s", total, completeDetail), Count: total}
	}
	return models.ProjectSectionState{Status: models.ProjectSectionStateInactive, Detail: fmt.Sprintf("No %s", completeDetail)}
}

func summarizeOntologySection(total, active, inProgress int) models.ProjectSectionState {
	if inProgress > 0 {
		return models.ProjectSectionState{Status: models.ProjectSectionStateInProgress, Detail: fmt.Sprintf("%d ontology draft/review item(s)", inProgress), Count: inProgress}
	}
	if active > 0 {
		return models.ProjectSectionState{Status: models.ProjectSectionStateComplete, Detail: fmt.Sprintf("%d active ontology(ies)", active), Count: active}
	}
	if total > 0 {
		return models.ProjectSectionState{Status: models.ProjectSectionStateComplete, Detail: fmt.Sprintf("%d ontology resource(s)", total), Count: total}
	}
	return models.ProjectSectionState{Status: models.ProjectSectionStateInactive, Detail: "No ontologies configured"}
}

func summarizeMLSection(total, active, failedTasks, degraded, failedModels int) models.ProjectSectionState {
	if failedTasks > 0 || failedModels > 0 {
		count := failedTasks + failedModels
		return models.ProjectSectionState{Status: models.ProjectSectionStateError, Detail: fmt.Sprintf("%d model issue(s)", count), Count: count}
	}
	if active > 0 {
		return models.ProjectSectionState{Status: models.ProjectSectionStateInProgress, Detail: fmt.Sprintf("%d training task(s) running", active), Count: active}
	}
	if degraded > 0 {
		return models.ProjectSectionState{Status: models.ProjectSectionStateError, Detail: fmt.Sprintf("%d degraded model(s)", degraded), Count: degraded}
	}
	if total > 0 {
		return models.ProjectSectionState{Status: models.ProjectSectionStateComplete, Detail: fmt.Sprintf("%d model(s) available", total), Count: total}
	}
	return models.ProjectSectionState{Status: models.ProjectSectionStateInactive, Detail: "No ML models configured"}
}

func summarizeTwinSection(total, active, failed, pendingApprovals, syncing, errored int) models.ProjectSectionState {
	if pendingApprovals > 0 {
		return models.ProjectSectionState{Status: models.ProjectSectionStateAttention, Detail: fmt.Sprintf("%d manual approval(s) waiting", pendingApprovals), Count: pendingApprovals, Pulse: true}
	}
	if failed > 0 || errored > 0 {
		count := failed + errored
		return models.ProjectSectionState{Status: models.ProjectSectionStateError, Detail: fmt.Sprintf("%d twin issue(s)", count), Count: count}
	}
	if active > 0 || syncing > 0 {
		count := active + syncing
		return models.ProjectSectionState{Status: models.ProjectSectionStateInProgress, Detail: fmt.Sprintf("%d twin job(s) active", count), Count: count}
	}
	if total > 0 {
		return models.ProjectSectionState{Status: models.ProjectSectionStateComplete, Detail: fmt.Sprintf("%d twin(s) ready", total), Count: total}
	}
	return models.ProjectSectionState{Status: models.ProjectSectionStateInactive, Detail: "No digital twins configured"}
}

func summarizeStorageSection(total, activeIngestion int) models.ProjectSectionState {
	if activeIngestion > 0 {
		return models.ProjectSectionState{Status: models.ProjectSectionStateInProgress, Detail: fmt.Sprintf("%d ingestion task(s) active", activeIngestion), Count: activeIngestion}
	}
	if total > 0 {
		return models.ProjectSectionState{Status: models.ProjectSectionStateComplete, Detail: fmt.Sprintf("%d storage config(s)", total), Count: total}
	}
	return models.ProjectSectionState{Status: models.ProjectSectionStateInactive, Detail: "No storage configured"}
}

func summarizeInsightsSection(totalInsights, pendingReviews int) models.ProjectSectionState {
	if pendingReviews > 0 {
		return models.ProjectSectionState{Status: models.ProjectSectionStateInProgress, Detail: fmt.Sprintf("%d review item(s) pending", pendingReviews), Count: pendingReviews}
	}
	if totalInsights > 0 {
		return models.ProjectSectionState{Status: models.ProjectSectionStateComplete, Detail: fmt.Sprintf("%d insight(s) available", totalInsights), Count: totalInsights}
	}
	return models.ProjectSectionState{Status: models.ProjectSectionStateInactive, Detail: "No insights yet"}
}

func summarizeStaticSection(total int, detail string) models.ProjectSectionState {
	if total > 0 {
		return models.ProjectSectionState{Status: models.ProjectSectionStateComplete, Detail: fmt.Sprintf("%d %s", total, detail), Count: total}
	}
	return models.ProjectSectionState{Status: models.ProjectSectionStateInactive, Detail: fmt.Sprintf("No %s", detail)}
}

func summarizeQueueSection(total int, queueLength int64, active, failed int) models.ProjectSectionState {
	if failed > 0 {
		return models.ProjectSectionState{Status: models.ProjectSectionStateError, Detail: fmt.Sprintf("%d recent queue failure(s)", failed), Count: failed}
	}
	if active > 0 || queueLength > 0 {
		count := active
		if int(queueLength) > count {
			count = int(queueLength)
		}
		return models.ProjectSectionState{Status: models.ProjectSectionStateInProgress, Detail: fmt.Sprintf("%d queued/active task(s)", count), Count: count}
	}
	if total > 0 {
		return models.ProjectSectionState{Status: models.ProjectSectionStateComplete, Detail: "Queue idle", Count: total}
	}
	return models.ProjectSectionState{Status: models.ProjectSectionStateInactive, Detail: "Queue idle"}
}
