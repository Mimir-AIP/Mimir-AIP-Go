package digitaltwin

import (
	"log"

	automationpkg "github.com/mimir-aip/mimir-aip-go/pkg/automation"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

// AutomationListener bridges completed ingestion pipelines into explicit twin-processing runs.
type AutomationListener struct {
	automationService *automationpkg.Service
	processor         *Processor
}

func NewAutomationListener(automationService *automationpkg.Service, processor *Processor) *AutomationListener {
	return &AutomationListener{automationService: automationService, processor: processor}
}

func (l *AutomationListener) OnPipelineCompleted(evt automationpkg.PipelineCompletedEvent) {
	if evt.PipelineType != models.PipelineTypeIngestion {
		return
	}
	automations, err := l.automationService.MatchPipelineCompleted(evt)
	if err != nil {
		log.Printf("Twin automation listener: failed to match automations for pipeline %s: %v", evt.PipelineID, err)
		return
	}
	for _, automation := range automations {
		if automation.TargetType != models.AutomationTargetTypeDigitalTwin || automation.ActionType != models.AutomationActionTypeProcessTwin {
			continue
		}
		if _, err := l.processor.RequestRun(automation.TargetID, &models.TwinProcessingRunCreateRequest{
			TriggerType:  models.TwinProcessingTriggerTypePipelineCompleted,
			TriggerRef:   evt.WorkTaskID,
			AutomationID: automation.ID,
		}); err != nil {
			log.Printf("Twin automation listener: failed to request run for automation %s: %v", automation.ID, err)
		}
	}
}
