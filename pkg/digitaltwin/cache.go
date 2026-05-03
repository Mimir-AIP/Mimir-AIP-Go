package digitaltwin

import (
	"context"
	"log"
	"time"
)

func (s *Service) StartCacheEviction(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			twins, err := s.store.ListDigitalTwins()
			if err != nil {
				log.Printf("cache eviction: failed to list digital twins: %v", err)
				continue
			}
			for _, twin := range twins {
				if err := s.store.DeleteExpiredPredictions(twin.ID); err != nil {
					log.Printf("cache eviction: error for twin %s: %v", twin.ID, err)
				}
			}
			log.Printf("cache eviction: completed tick for %d digital twins", len(twins))
		}
	}
}

// GetRelatedEntities returns entities related to the given entity via the specified relationship type.
