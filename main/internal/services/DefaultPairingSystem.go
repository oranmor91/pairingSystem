package services

import (
	"fmt"
	"sync"

	"pairingSystem/internal/filters"
	"pairingSystem/internal/models"
	"pairingSystem/internal/storage"
)

// DefaultPairingSystem implements the PairingSystem interface
type DefaultPairingSystem struct {
	processor *Processor
	mu        sync.RWMutex
}

// NewDefaultPairingSystem creates a new default pairing system
func NewDefaultPairingSystem(providerStorage *storage.ProviderStorage, cacheCapacity int) *DefaultPairingSystem {
	return &DefaultPairingSystem{
		processor: NewProcessor(providerStorage, cacheCapacity),
	}
}

// Ensure DefaultPairingSystem implements PairingSystem interface
var _ PairingSystem = (*DefaultPairingSystem)(nil)

// FilterProviders filters providers based on consumer policy requirements
// This method applies location, feature, and stake filters sequentially
func (d *DefaultPairingSystem) FilterProviders(providers []*models.Provider, policy *models.ConsumerPolicy) []*models.Provider {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if len(providers) == 0 {
		return providers
	}

	if policy == nil {
		return []*models.Provider{}
	}

	// Validate policy
	if err := filters.ValidatePolicy(policy); err != nil {
		return []*models.Provider{}
	}

	// Use processor to filter providers
	return d.processor.FilterProviders(providers, policy)
}

// RankProviders ranks providers based on consumer policy preferences
// This method calculates scores for stake, location match, and feature completeness
func (d *DefaultPairingSystem) RankProviders(providers []*models.Provider, policy *models.ConsumerPolicy) []*models.PairingScore {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if len(providers) == 0 {
		return []*models.PairingScore{}
	}

	if policy == nil {
		return []*models.PairingScore{}
	}

	// Validate policy
	if err := filters.ValidatePolicy(policy); err != nil {
		return []*models.PairingScore{}
	}

	// Use processor to rank providers
	return d.processor.RankProviders(providers, policy)
}

// GetPairingList returns the top 5 providers for a given policy
// This method combines filtering and ranking to provide the best matches
func (d *DefaultPairingSystem) GetPairingList(providers []*models.Provider, policy *models.ConsumerPolicy) ([]*models.Provider, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if len(providers) == 0 {
		return []*models.Provider{}, nil
	}

	if policy == nil {
		return nil, fmt.Errorf("consumer policy cannot be nil")
	}

	// Validate policy
	if err := filters.ValidatePolicy(policy); err != nil {
		return nil, fmt.Errorf("invalid policy: %w", err)
	}

	// Process policy through complete pipeline
	result := d.processor.ProcessPolicy(policy)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to process policy: %w", result.Error)
	}

	return result.TopProviders, nil
}

// GetPairingListWithDetails returns detailed results including scores and processing info
func (d *DefaultPairingSystem) GetPairingListWithDetails(providers []*models.Provider, policy *models.ConsumerPolicy) (*ProcessorResult, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if policy == nil {
		return nil, fmt.Errorf("consumer policy cannot be nil")
	}

	// Validate policy
	if err := filters.ValidatePolicy(policy); err != nil {
		return nil, fmt.Errorf("invalid policy: %w", err)
	}

	// Process policy through complete pipeline
	result := d.processor.ProcessPolicy(policy)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to process policy: %w", result.Error)
	}

	return result, nil
}

// GetPairingListConcurrently processes multiple policies concurrently
func (d *DefaultPairingSystem) GetPairingListConcurrently(providers []*models.Provider, policies []*models.ConsumerPolicy, maxConcurrency int) ([]*ProcessorResult, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if len(policies) == 0 {
		return []*ProcessorResult{}, nil
	}

	// Validate all policies
	for i, policy := range policies {
		if policy == nil {
			return nil, fmt.Errorf("policy at index %d is nil", i)
		}
		if err := filters.ValidatePolicy(policy); err != nil {
			return nil, fmt.Errorf("invalid policy at index %d: %w", i, err)
		}
	}

	// Process policies concurrently
	results := d.processor.ProcessPoliciesConcurrently(policies, maxConcurrency)

	// Check for errors in results
	for i, result := range results {
		if result.Error != nil {
			return nil, fmt.Errorf("failed to process policy at index %d: %w", i, result.Error)
		}
	}

	return results, nil
}

// SetTopN sets the number of top providers to return
func (d *DefaultPairingSystem) SetTopN(n int) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.processor.SetTopN(n)
}

// GetTopN returns the current number of top providers being returned
func (d *DefaultPairingSystem) GetTopN() int {
	d.mu.RLock()
	defer d.mu.RUnlock()

	stats := d.processor.GetStats()
	if topN, exists := stats["top_n"]; exists {
		if n, ok := topN.(int); ok {
			return n
		}
	}
	return 5 // Default value
}

// SetCachingEnabled enables or disables caching
func (d *DefaultPairingSystem) SetCachingEnabled(enabled bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.processor.SetCachingEnabled(enabled)
}

// IsCachingEnabled returns whether caching is enabled
func (d *DefaultPairingSystem) IsCachingEnabled() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	stats := d.processor.GetStats()
	if cachingEnabled, exists := stats["caching_enabled"]; exists {
		if enabled, ok := cachingEnabled.(bool); ok {
			return enabled
		}
	}
	return true // Default value
}

// UpdateScoringWeights updates the weights used in the scoring system
func (d *DefaultPairingSystem) UpdateScoringWeights(stake, location, feature float64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.processor.UpdateScoringWeights(stake, location, feature)
}

// GetStats returns comprehensive statistics about the pairing system
func (d *DefaultPairingSystem) GetStats() map[string]interface{} {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return d.processor.GetStats()
}

// GetFilterStats returns detailed filtering statistics for a specific policy
func (d *DefaultPairingSystem) GetFilterStats(policy *models.ConsumerPolicy) map[string]interface{} {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return d.processor.GetFilterStats(policy)
}

// GetScoringStats returns detailed scoring statistics for a specific policy
func (d *DefaultPairingSystem) GetScoringStats(policy *models.ConsumerPolicy) map[string]interface{} {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return d.processor.GetScoringStats(policy)
}

// ResetStats resets all statistics
func (d *DefaultPairingSystem) ResetStats() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.processor.ResetStats()
}

// ValidateConfiguration validates the pairing system configuration
func (d *DefaultPairingSystem) ValidateConfiguration() error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.processor == nil {
		return fmt.Errorf("processor is nil")
	}

	return d.processor.ValidateConfiguration()
}

// GetProcessor returns the internal processor (for testing/debugging)
func (d *DefaultPairingSystem) GetProcessor() *Processor {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return d.processor
}
