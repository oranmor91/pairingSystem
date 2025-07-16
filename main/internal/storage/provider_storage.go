package storage

import (
	"sync"

	"pairingSystem/internal/models"

	"github.com/google/uuid"
)

// ProviderStorage manages all providers with location-based indexing
type ProviderStorage struct {
	// Static collection of all providers
	providers map[string]*models.Provider
	// Location-based indexing for fast location queries
	providersByLocation map[string][]*models.Provider
	// Thread-safe access
	mu sync.RWMutex
}

// NewProviderStorage creates a new provider storage instance
func NewProviderStorage() *ProviderStorage {
	return &ProviderStorage{
		providers:           make(map[string]*models.Provider),
		providersByLocation: make(map[string][]*models.Provider),
	}
}

// AddProvider adds a new provider to the storage
func (ps *ProviderStorage) AddProvider(provider *models.Provider) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	// Generate unique ID if not provided
	if provider.Address == "" {
		provider.Address = uuid.New().String()
	}

	ps.providers[provider.Address] = provider

	// Add to location-based index
	ps.providersByLocation[provider.Location] = append(
		ps.providersByLocation[provider.Location],
		provider,
	)
}

// GetAllProviders returns all providers in the storage
func (ps *ProviderStorage) GetAllProviders() []*models.Provider {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	providers := make([]*models.Provider, 0, len(ps.providers))
	for _, provider := range ps.providers {
		providers = append(providers, provider)
	}
	return providers
}

// GetProvidersByLocation returns providers filtered by location
func (ps *ProviderStorage) GetProvidersByLocation(location string) []*models.Provider {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	if providers, exists := ps.providersByLocation[location]; exists {
		// Return a copy to avoid race conditions
		result := make([]*models.Provider, len(providers))
		copy(result, providers)
		return result
	}
	return []*models.Provider{}
}

// GetProvider returns a specific provider by address
func (ps *ProviderStorage) GetProvider(address string) (*models.Provider, bool) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	provider, exists := ps.providers[address]
	return provider, exists
}

// GetProviderCount returns the total number of providers
func (ps *ProviderStorage) GetProviderCount() int {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	return len(ps.providers)
}

// GetLocationStats returns statistics about providers by location
func (ps *ProviderStorage) GetLocationStats() map[string]int {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	stats := make(map[string]int)
	for location, providers := range ps.providersByLocation {
		stats[location] = len(providers)
	}
	return stats
}

// RemoveProvider removes a provider from the storage
func (ps *ProviderStorage) RemoveProvider(address string) bool {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	provider, exists := ps.providers[address]
	if !exists {
		return false
	}

	// Remove from main storage
	delete(ps.providers, address)

	// Remove from location index
	locationProviders := ps.providersByLocation[provider.Location]
	for i, p := range locationProviders {
		if p.Address == address {
			ps.providersByLocation[provider.Location] = append(
				locationProviders[:i],
				locationProviders[i+1:]...,
			)
			break
		}
	}

	// Clean up empty location entries
	if len(ps.providersByLocation[provider.Location]) == 0 {
		delete(ps.providersByLocation, provider.Location)
	}

	return true
}
