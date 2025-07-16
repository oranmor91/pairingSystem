package filters

import (
	"fmt"
	"strings"

	"pairingSystem/internal/models"
)

// Filter interface defines the contract for provider filtering
type Filter interface {
	Apply(providers []*models.Provider, policy *models.ConsumerPolicy) []*models.Provider
	Name() string
}

// LocationFilter filters providers based on their location
type LocationFilter struct{}

func (lf *LocationFilter) Name() string {
	return "LocationFilter"
}

// Apply filters providers that match the required location
// If no location is specified in policy, all providers pass
func (lf *LocationFilter) Apply(providers []*models.Provider, policy *models.ConsumerPolicy) []*models.Provider {
	if policy.RequiredLocation == "" {
		return providers
	}

	var filtered []*models.Provider
	for _, provider := range providers {
		if provider.Location == policy.RequiredLocation {
			filtered = append(filtered, provider)
		}
	}
	return filtered
}

// FeatureFilter filters providers that support all required features
type FeatureFilter struct{}

func (ff *FeatureFilter) Name() string {
	return "FeatureFilter"
}

// Apply filters providers that support all required features
// If no features are required, all providers pass
func (ff *FeatureFilter) Apply(providers []*models.Provider, policy *models.ConsumerPolicy) []*models.Provider {
	if len(policy.RequiredFeatures) == 0 {
		return providers
	}

	var filtered []*models.Provider
	for _, provider := range providers {
		if ff.hasAllRequiredFeatures(provider, policy.RequiredFeatures) {
			filtered = append(filtered, provider)
		}
	}
	return filtered
}

// hasAllRequiredFeatures checks if provider supports all required features
func (ff *FeatureFilter) hasAllRequiredFeatures(provider *models.Provider, requiredFeatures []string) bool {
	// Create a set of provider features for O(1) lookup
	providerFeatures := make(map[string]bool)
	for _, feature := range provider.Features {
		providerFeatures[strings.ToLower(feature)] = true
	}

	// Check if all required features are supported
	for _, required := range requiredFeatures {
		if !providerFeatures[strings.ToLower(required)] {
			return false
		}
	}
	return true
}

// StakeFilter filters providers that meet the minimum stake requirement
type StakeFilter struct{}

func (sf *StakeFilter) Name() string {
	return "StakeFilter"
}

// Apply filters providers that meet the minimum stake requirement
// If no minimum stake is specified, all providers pass
func (sf *StakeFilter) Apply(providers []*models.Provider, policy *models.ConsumerPolicy) []*models.Provider {
	if policy.MinStake <= 0 {
		return providers
	}

	var filtered []*models.Provider
	for _, provider := range providers {
		if provider.Stake >= policy.MinStake {
			filtered = append(filtered, provider)
		}
	}
	return filtered
}

// FilterChain applies multiple filters in sequence
type FilterChain struct {
	filters []Filter
}

// NewFilterChain creates a new filter chain with the standard filters
func NewFilterChain() *FilterChain {
	return &FilterChain{
		filters: []Filter{
			&LocationFilter{},
			&FeatureFilter{},
			&StakeFilter{},
		},
	}
}

// ApplyFilters applies all filters in the chain sequentially
func (fc *FilterChain) ApplyFilters(providers []*models.Provider, policy *models.ConsumerPolicy) []*models.Provider {
	if len(providers) == 0 {
		return providers
	}

	filtered := providers
	for _, filter := range fc.filters {
		filtered = filter.Apply(filtered, policy)
		// Early exit if no providers remain after filtering
		if len(filtered) == 0 {
			break
		}
	}
	return filtered
}

// GetFilterStats returns statistics about filtering results
func (fc *FilterChain) GetFilterStats(providers []*models.Provider, policy *models.ConsumerPolicy) map[string]int {
	stats := make(map[string]int)
	stats["initial"] = len(providers)

	filtered := providers
	for _, filter := range fc.filters {
		filtered = filter.Apply(filtered, policy)
		stats[filter.Name()] = len(filtered)
	}

	return stats
}

// ValidatePolicy validates that the consumer policy is properly formatted
func ValidatePolicy(policy *models.ConsumerPolicy) error {
	if policy == nil {
		return fmt.Errorf("policy cannot be nil")
	}

	if policy.MinStake < 0 {
		return fmt.Errorf("minimum stake cannot be negative")
	}

	// Validate required features are not empty strings
	for _, feature := range policy.RequiredFeatures {
		if strings.TrimSpace(feature) == "" {
			return fmt.Errorf("required features cannot contain empty strings")
		}
	}

	return nil
}

// ValidateProvider validates that the provider is properly formatted
func ValidateProvider(provider *models.Provider) error {
	if provider == nil {
		return fmt.Errorf("provider cannot be nil")
	}

	if provider.Address == "" {
		return fmt.Errorf("provider address cannot be empty")
	}

	if provider.Stake < 0 {
		return fmt.Errorf("provider stake cannot be negative")
	}

	if provider.Location == "" {
		return fmt.Errorf("provider location cannot be empty")
	}

	// Validate features are not empty strings
	for _, feature := range provider.Features {
		if strings.TrimSpace(feature) == "" {
			return fmt.Errorf("provider features cannot contain empty strings")
		}
	}

	return nil
}
