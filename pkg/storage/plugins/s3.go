package plugins

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/storage"
)

// S3Plugin implements the StoragePlugin interface for AWS S3 (and S3-compatible) storage
type S3Plugin struct {
	client      *s3.Client
	bucket      string
	prefix      string
	initialized bool
}

// NewS3Plugin creates a new S3 storage plugin
func NewS3Plugin() *S3Plugin {
	return &S3Plugin{}
}

// Initialize initializes the S3 plugin with configuration
func (s *S3Plugin) Initialize(cfg *models.PluginConfig) error {
	if cfg.Options == nil {
		return fmt.Errorf("options are required for s3 storage (bucket)")
	}

	bucketVal, ok := cfg.Options["bucket"]
	if !ok {
		return fmt.Errorf("bucket is required in options for s3 storage")
	}
	bucket, ok := bucketVal.(string)
	if !ok || bucket == "" {
		return fmt.Errorf("bucket must be a non-empty string")
	}

	prefix := "mimir"
	if p, ok := cfg.Options["prefix"].(string); ok && p != "" {
		prefix = p
	}

	region := "us-east-1"
	accessKey := ""
	secretKey := ""

	if cfg.Credentials != nil {
		if r, ok := cfg.Credentials["region"].(string); ok && r != "" {
			region = r
		}
		if ak, ok := cfg.Credentials["access_key_id"].(string); ok {
			accessKey = ak
		}
		if sk, ok := cfg.Credentials["secret_access_key"].(string); ok {
			secretKey = sk
		}
	}

	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	s3Opts := []func(*s3.Options){}

	if endpoint, ok := cfg.Options["endpoint"].(string); ok && endpoint != "" {
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(endpoint)
			o.UsePathStyle = true
		})
	}

	s.client = s3.NewFromConfig(awsCfg, s3Opts...)
	s.bucket = bucket
	s.prefix = prefix
	s.initialized = true
	return nil
}

// CreateSchema is a no-op for S3 (bucket structure is implicit)
func (s *S3Plugin) CreateSchema(ontology *models.OntologyDefinition) error {
	if !s.initialized {
		return fmt.Errorf("plugin not initialized")
	}
	// S3 uses a flat key namespace; no schema creation needed
	return nil
}

// Store stores CIR data into S3
func (s *S3Plugin) Store(cir *models.CIR) (*models.StorageResult, error) {
	if !s.initialized {
		return nil, fmt.Errorf("plugin not initialized")
	}

	if err := cir.Validate(); err != nil {
		return nil, fmt.Errorf("invalid CIR: %w", err)
	}

	entityType := s.inferEntityType(cir)
	affectedItems := 0

	if arr, err := cir.GetDataAsArray(); err == nil {
		for _, item := range arr {
			itemCIR := &models.CIR{
				Version:  cir.Version,
				Source:   cir.Source,
				Data:     item,
				Metadata: cir.Metadata,
			}
			if err := s.putObject(entityType, itemCIR); err != nil {
				return nil, fmt.Errorf("failed to store item: %w", err)
			}
			affectedItems++
		}
	} else {
		if err := s.putObject(entityType, cir); err != nil {
			return nil, fmt.Errorf("failed to store item: %w", err)
		}
		affectedItems = 1
	}

	return &models.StorageResult{
		Success:       true,
		AffectedItems: affectedItems,
	}, nil
}

func (s *S3Plugin) putObject(entityType string, cir *models.CIR) error {
	data, err := json.Marshal(cir)
	if err != nil {
		return fmt.Errorf("failed to marshal CIR: %w", err)
	}

	key := fmt.Sprintf("%s/%s/%s.json", s.prefix, entityType, uuid.New().String())
	contentType := "application/json"

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	return err
}

// Retrieve retrieves CIR data from S3 using a query
func (s *S3Plugin) Retrieve(query *models.CIRQuery) ([]*models.CIR, error) {
	if !s.initialized {
		return nil, fmt.Errorf("plugin not initialized")
	}

	if err := storage.ValidateCIRQuery(query); err != nil {
		return nil, fmt.Errorf("invalid query: %w", err)
	}

	entityType := query.EntityType
	if entityType == "" {
		entityType = "default"
	}

	prefix := fmt.Sprintf("%s/%s/", s.prefix, entityType)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(prefix),
	})

	results := make([]*models.CIR, 0)

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list objects in s3: %w", err)
		}

		for _, obj := range page.Contents {
			if obj.Key == nil || !strings.HasSuffix(*obj.Key, ".json") {
				continue
			}

			getCtx, getCancel := context.WithTimeout(context.Background(), 10*time.Second)
			resp, err := s.client.GetObject(getCtx, &s3.GetObjectInput{
				Bucket: aws.String(s.bucket),
				Key:    obj.Key,
			})
			getCancel()

			if err != nil {
				continue
			}

			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				continue
			}

			var cir models.CIR
			if err := json.Unmarshal(body, &cir); err != nil {
				continue
			}

			if s.matchesFilters(&cir, query.Filters) {
				results = append(results, &cir)
			}
		}
	}

	// Apply offset and limit
	if query.Offset > 0 && query.Offset < len(results) {
		results = results[query.Offset:]
	}
	if query.Limit > 0 && query.Limit < len(results) {
		results = results[:query.Limit]
	}

	return results, nil
}

// Update updates existing CIR data in S3
func (s *S3Plugin) Update(query *models.CIRQuery, updates *models.CIRUpdate) (*models.StorageResult, error) {
	if !s.initialized {
		return nil, fmt.Errorf("plugin not initialized")
	}

	entityType := query.EntityType
	if entityType == "" {
		entityType = "default"
	}

	prefix := fmt.Sprintf("%s/%s/", s.prefix, entityType)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(prefix),
	})

	affectedItems := 0

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list objects in s3: %w", err)
		}

		for _, obj := range page.Contents {
			if obj.Key == nil || !strings.HasSuffix(*obj.Key, ".json") {
				continue
			}

			getCtx, getCancel := context.WithTimeout(context.Background(), 10*time.Second)
			resp, err := s.client.GetObject(getCtx, &s3.GetObjectInput{
				Bucket: aws.String(s.bucket),
				Key:    obj.Key,
			})
			getCancel()

			if err != nil {
				continue
			}

			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				continue
			}

			var cir models.CIR
			if err := json.Unmarshal(body, &cir); err != nil {
				continue
			}

			if !s.matchesFilters(&cir, query.Filters) {
				continue
			}

			dataMap, err := cir.GetDataAsMap()
			if err != nil {
				continue
			}

			for key, value := range updates.Updates {
				dataMap[key] = value
			}
			cir.Data = dataMap
			cir.UpdateSize()

			updatedData, err := json.Marshal(&cir)
			if err != nil {
				continue
			}

			putCtx, putCancel := context.WithTimeout(context.Background(), 10*time.Second)
			_, putErr := s.client.PutObject(putCtx, &s3.PutObjectInput{
				Bucket:      aws.String(s.bucket),
				Key:         obj.Key,
				Body:        bytes.NewReader(updatedData),
				ContentType: aws.String("application/json"),
			})
			putCancel()

			if putErr != nil {
				continue
			}
			affectedItems++
		}
	}

	return &models.StorageResult{
		Success:       true,
		AffectedItems: affectedItems,
	}, nil
}

// Delete deletes CIR data from S3
func (s *S3Plugin) Delete(query *models.CIRQuery) (*models.StorageResult, error) {
	if !s.initialized {
		return nil, fmt.Errorf("plugin not initialized")
	}

	entityType := query.EntityType
	if entityType == "" {
		entityType = "default"
	}

	prefix := fmt.Sprintf("%s/%s/", s.prefix, entityType)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(prefix),
	})

	affectedItems := 0

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list objects in s3: %w", err)
		}

		for _, obj := range page.Contents {
			if obj.Key == nil || !strings.HasSuffix(*obj.Key, ".json") {
				continue
			}

			getCtx, getCancel := context.WithTimeout(context.Background(), 10*time.Second)
			resp, err := s.client.GetObject(getCtx, &s3.GetObjectInput{
				Bucket: aws.String(s.bucket),
				Key:    obj.Key,
			})
			getCancel()

			if err != nil {
				continue
			}

			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				continue
			}

			var cir models.CIR
			if err := json.Unmarshal(body, &cir); err != nil {
				continue
			}

			if !s.matchesFilters(&cir, query.Filters) {
				continue
			}

			delCtx, delCancel := context.WithTimeout(context.Background(), 10*time.Second)
			_, delErr := s.client.DeleteObject(delCtx, &s3.DeleteObjectInput{
				Bucket: aws.String(s.bucket),
				Key:    obj.Key,
			})
			delCancel()

			if delErr != nil {
				continue
			}
			affectedItems++
		}
	}

	return &models.StorageResult{
		Success:       true,
		AffectedItems: affectedItems,
	}, nil
}

// GetMetadata returns metadata about the S3 storage
func (s *S3Plugin) GetMetadata() (*models.StorageMetadata, error) {
	return &models.StorageMetadata{
		StorageType: "s3",
		Version:     "1.0.0",
		Capabilities: []string{
			"store",
			"retrieve",
			"update",
			"delete",
			"object_storage",
			"scalable",
		},
	}, nil
}

// HealthCheck checks if the S3 bucket is accessible
func (s *S3Plugin) HealthCheck() (bool, error) {
	if !s.initialized {
		return false, fmt.Errorf("plugin not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s.bucket),
	})
	if err != nil {
		return false, fmt.Errorf("s3 health check failed: %w", err)
	}

	return true, nil
}

// inferEntityType attempts to infer the entity type from CIR data
func (s *S3Plugin) inferEntityType(cir *models.CIR) string {
	if entityType, ok := cir.GetParameter("entity_type"); ok {
		if typeStr, ok := entityType.(string); ok {
			return typeStr
		}
	}

	if dataMap, err := cir.GetDataAsMap(); err == nil {
		if _, hasName := dataMap["name"]; hasName {
			if _, hasDept := dataMap["department"]; hasDept {
				return "Employee"
			}
			if _, hasLoc := dataMap["location"]; hasLoc {
				return "Company"
			}
		}
	}

	return "default"
}

// matchesFilters checks if a CIR object matches the query filters
func (s *S3Plugin) matchesFilters(cir *models.CIR, filters []models.CIRCondition) bool {
	if len(filters) == 0 {
		return true
	}

	dataMap, err := cir.GetDataAsMap()
	if err != nil {
		return false
	}

	for _, filter := range filters {
		value, exists := dataMap[filter.Attribute]
		if !exists {
			return false
		}

		if !s.evaluateCondition(value, filter.Operator, filter.Value) {
			return false
		}
	}

	return true
}

// evaluateCondition evaluates a filter condition
func (s *S3Plugin) evaluateCondition(value interface{}, operator string, expected interface{}) bool {
	switch operator {
	case "eq":
		return fmt.Sprintf("%v", value) == fmt.Sprintf("%v", expected)
	case "neq":
		return fmt.Sprintf("%v", value) != fmt.Sprintf("%v", expected)
	case "like":
		valueStr := fmt.Sprintf("%v", value)
		expectedStr := fmt.Sprintf("%v", expected)
		return strings.Contains(strings.ToLower(valueStr), strings.ToLower(expectedStr))
	case "gt":
		return toFloat(value) > toFloat(expected)
	case "gte":
		return toFloat(value) >= toFloat(expected)
	case "lt":
		return toFloat(value) < toFloat(expected)
	case "lte":
		return toFloat(value) <= toFloat(expected)
	case "in":
		if arr, ok := expected.([]interface{}); ok {
			valueStr := fmt.Sprintf("%v", value)
			for _, v := range arr {
				if fmt.Sprintf("%v", v) == valueStr {
					return true
				}
			}
		}
		return false
	default:
		return false
	}
}
