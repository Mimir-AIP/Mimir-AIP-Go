package storage

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

func (s *Service) AggregateByOntology(projectID, ontologyID string, req *models.OntologyAggregateRequest) (*models.OntologyAggregateResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("aggregate request is required")
	}
	if req.Metric.Property == "" && strings.ToLower(req.Metric.Function) != "count" {
		return nil, fmt.Errorf("metric.property is required for non-count aggregations")
	}
	compiled, err := s.store.GetCompiledOntology(ontologyID)
	if err != nil {
		return nil, fmt.Errorf("compiled ontology not found: %w", err)
	}
	if compiled.ProjectID != projectID {
		return nil, fmt.Errorf("ontology %s belongs to project %s, not %s", ontologyID, compiled.ProjectID, projectID)
	}
	classID := resolveClassIDForRetrieval(req.ClassID, compiled)
	if strings.TrimSpace(req.ClassID) != "" && classID == "" {
		return nil, fmt.Errorf("ontology class not found: %s", req.ClassID)
	}
	function := strings.ToLower(strings.TrimSpace(req.Metric.Function))
	if function == "" {
		function = "count"
	}
	if !validAggregateFunction(function) {
		return nil, fmt.Errorf("unsupported aggregation function: %s", function)
	}
	metricPropertyID := ""
	if req.Metric.Property != "" {
		ids, err := resolvePropertyIDsForRetrieval([]string{req.Metric.Property}, classID, compiled)
		if err != nil {
			return nil, err
		}
		for id := range ids {
			metricPropertyID = id
		}
	}
	groupIDs, err := resolvePropertyIDsForRetrieval(req.GroupBy, classID, compiled)
	if err != nil {
		return nil, err
	}
	retrieveReq := &models.OntologyRetrieveRequest{ProjectID: projectID, ClassID: req.ClassID, Filters: req.Filters, StorageIDs: req.StorageIDs, Limit: req.Limit}
	if metricPropertyID != "" {
		retrieveReq.Properties = append(retrieveReq.Properties, metricPropertyID)
	}
	for id := range groupIDs {
		retrieveReq.Properties = append(retrieveReq.Properties, id)
	}
	retrieved, err := s.RetrieveByOntology(projectID, ontologyID, retrieveReq)
	if err != nil {
		return nil, err
	}
	buckets := map[string]*aggregateBucket{}
	for _, result := range retrieved.Results {
		key := groupKey(result.Properties, groupIDs)
		bucket := buckets[key.canonical]
		if bucket == nil {
			bucket = &aggregateBucket{key: key.values, min: math.Inf(1), max: math.Inf(-1)}
			buckets[key.canonical] = bucket
		}
		bucket.count++
		if function == "count" {
			continue
		}
		value, ok := toFloat(result.Properties[metricPropertyID])
		if !ok {
			continue
		}
		bucket.sum += value
		bucket.valueCount++
		if value < bucket.min {
			bucket.min = value
		}
		if value > bucket.max {
			bucket.max = value
		}
	}
	groups := make([]models.OntologyAggregateGroup, 0, len(buckets))
	for _, bucket := range buckets {
		value := bucketValue(bucket, function)
		groups = append(groups, models.OntologyAggregateGroup{Key: bucket.key, Value: value, Count: bucket.count})
	}
	sort.Slice(groups, func(i, j int) bool { return fmt.Sprint(groups[i].Key) < fmt.Sprint(groups[j].Key) })
	return &models.OntologyAggregateResponse{OntologyID: ontologyID, ContentHash: compiled.ContentHash, ClassID: classID, PropertyID: metricPropertyID, Function: function, Groups: groups, Count: len(groups)}, nil
}

type aggregateBucket struct {
	key        map[string]interface{}
	count      int
	valueCount int
	sum        float64
	min        float64
	max        float64
}

type aggregateKey struct {
	canonical string
	values    map[string]interface{}
}

func groupKey(properties map[string]interface{}, groupIDs map[string]struct{}) aggregateKey {
	if len(groupIDs) == 0 {
		return aggregateKey{canonical: "__all__", values: map[string]interface{}{}}
	}
	ids := make([]string, 0, len(groupIDs))
	for id := range groupIDs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	values := map[string]interface{}{}
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		value := properties[id]
		values[id] = value
		parts = append(parts, id+"="+fmt.Sprint(value))
	}
	return aggregateKey{canonical: strings.Join(parts, "|"), values: values}
}

func validAggregateFunction(function string) bool {
	switch function {
	case "count", "sum", "avg", "min", "max":
		return true
	default:
		return false
	}
}

func bucketValue(bucket *aggregateBucket, function string) float64 {
	switch function {
	case "count":
		return float64(bucket.count)
	case "sum":
		return bucket.sum
	case "avg":
		if bucket.valueCount == 0 {
			return 0
		}
		return bucket.sum / float64(bucket.valueCount)
	case "min":
		if bucket.valueCount == 0 {
			return 0
		}
		return bucket.min
	case "max":
		if bucket.valueCount == 0 {
			return 0
		}
		return bucket.max
	default:
		return 0
	}
}
