package testdata

import (
	"fmt"
	"math/rand"
	"time"

	"pairingSystem/internal/models"
	"pairingSystem/internal/storage"
)

// FakeDataGenerator generates fake providers and policies for testing
type FakeDataGenerator struct {
	Rand *rand.Rand
}

// NewFakeDataGenerator creates a new fake data generator
func NewFakeDataGenerator() *FakeDataGenerator {
	return &FakeDataGenerator{
		Rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Predefined data for realistic fake generation
var (
	locations = []string{
		"US-East-1", "US-West-1", "US-West-2", "EU-West-1", "EU-Central-1",
		"AP-Southeast-1", "AP-Northeast-1", "AP-South-1", "CA-Central-1", "SA-East-1",
		"US-East-2", "EU-West-2", "EU-West-3", "AP-Southeast-2", "AP-Northeast-2",
	}

	features = []string{
		"HTTP", "HTTPS", "WebSocket", "gRPC", "GraphQL", "REST", "SOAP",
		"TCP", "UDP", "MQTT", "Redis", "PostgreSQL", "MongoDB", "MySQL",
		"Load_Balancing", "SSL_Termination", "Caching", "Compression", "Rate_Limiting",
		"Authentication", "Authorization", "Monitoring", "Logging", "Metrics",
	}

	providerAddressPrefixes = []string{
		"lava", "provider", "node", "rpc", "service", "endpoint", "api", "gateway",
	}
)

// GenerateRandomProvider generates a random provider with realistic data
func (fdg *FakeDataGenerator) GenerateRandomProvider() *models.Provider {
	// Generate random address
	prefix := providerAddressPrefixes[fdg.Rand.Intn(len(providerAddressPrefixes))]
	address := fmt.Sprintf("%s_%d_%x", prefix, fdg.Rand.Intn(10000), fdg.Rand.Uint32())

	// Generate random stake - weighted towards lower stakes to better match policies
	// 70% chance of stake between 1000-50000, 30% chance of higher stakes
	var stake int64
	if fdg.Rand.Float32() < 0.7 {
		stake = int64(fdg.Rand.Intn(49000) + 1000) // 1000-50000
	} else {
		stake = int64(fdg.Rand.Intn(950000) + 50000) // 50000-1000000
	}

	// Generate random location
	location := locations[fdg.Rand.Intn(len(locations))]

	// Generate random features (3-12 features) - increased minimum to improve matching
	numFeatures := fdg.Rand.Intn(10) + 3
	selectedFeatures := make([]string, 0, numFeatures)
	usedFeatures := make(map[string]bool)

	for len(selectedFeatures) < numFeatures {
		feature := features[fdg.Rand.Intn(len(features))]
		if !usedFeatures[feature] {
			selectedFeatures = append(selectedFeatures, feature)
			usedFeatures[feature] = true
		}
	}

	return &models.Provider{
		Address:  address,
		Stake:    stake,
		Location: location,
		Features: selectedFeatures,
	}
}

// GenerateRandomProviders generates multiple random providers
func (fdg *FakeDataGenerator) GenerateRandomProviders(count int) []*models.Provider {
	providers := make([]*models.Provider, count)
	for i := 0; i < count; i++ {
		providers[i] = fdg.GenerateRandomProvider()
	}
	return providers
}

// GenerateRandomPolicy generates a random consumer policy
func (fdg *FakeDataGenerator) GenerateRandomPolicy() *models.ConsumerPolicy {
	// 50% chance of having a location requirement
	var requiredLocation string
	if fdg.Rand.Float32() < 0.5 {
		requiredLocation = locations[fdg.Rand.Intn(len(locations))]
	}

	// Generate random minimum stake (0-25000) - reduced from 50000
	minStake := int64(fdg.Rand.Intn(25000))

	// Generate random required features (0-3 features) - reduced from 6
	numFeatures := fdg.Rand.Intn(4)
	requiredFeatures := make([]string, 0, numFeatures)
	usedFeatures := make(map[string]bool)

	for len(requiredFeatures) < numFeatures {
		feature := features[fdg.Rand.Intn(len(features))]
		if !usedFeatures[feature] {
			requiredFeatures = append(requiredFeatures, feature)
			usedFeatures[feature] = true
		}
	}

	return &models.ConsumerPolicy{
		RequiredLocation: requiredLocation,
		RequiredFeatures: requiredFeatures,
		MinStake:         minStake,
	}
}

// GenerateRandomPolicies generates multiple random consumer policies
func (fdg *FakeDataGenerator) GenerateRandomPolicies(count int) []*models.ConsumerPolicy {
	policies := make([]*models.ConsumerPolicy, count)
	for i := 0; i < count; i++ {
		policies[i] = fdg.GenerateRandomPolicy()
	}
	return policies
}

// GenerateSpecificProvider generates a provider with specific characteristics
func (fdg *FakeDataGenerator) GenerateSpecificProvider(location string, minStake int64, requiredFeatures []string) *models.Provider {
	// Generate random address
	prefix := providerAddressPrefixes[fdg.Rand.Intn(len(providerAddressPrefixes))]
	address := fmt.Sprintf("%s_%d_%x", prefix, fdg.Rand.Intn(10000), fdg.Rand.Uint32())

	// Use provided location or random
	if location == "" {
		location = locations[fdg.Rand.Intn(len(locations))]
	}

	// Generate stake equal to or greater than minimum
	stake := minStake + int64(fdg.Rand.Intn(100000))

	// Include required features plus some random ones
	allFeatures := make([]string, 0)
	allFeatures = append(allFeatures, requiredFeatures...)

	// Add some random features
	numExtraFeatures := fdg.Rand.Intn(5)
	usedFeatures := make(map[string]bool)
	for _, feature := range requiredFeatures {
		usedFeatures[feature] = true
	}

	for len(allFeatures)-len(requiredFeatures) < numExtraFeatures {
		feature := features[fdg.Rand.Intn(len(features))]
		if !usedFeatures[feature] {
			allFeatures = append(allFeatures, feature)
			usedFeatures[feature] = true
		}
	}

	return &models.Provider{
		Address:  address,
		Stake:    stake,
		Location: location,
		Features: allFeatures,
	}
}

// GenerateMatchingProviders generates providers that match a specific policy
func (fdg *FakeDataGenerator) GenerateMatchingProviders(policy *models.ConsumerPolicy, count int) []*models.Provider {
	providers := make([]*models.Provider, count)
	for i := 0; i < count; i++ {
		providers[i] = fdg.GenerateSpecificProvider(
			policy.RequiredLocation,
			policy.MinStake,
			policy.RequiredFeatures,
		)
	}
	return providers
}

// GenerateDistributedProviders generates providers distributed across locations
func (fdg *FakeDataGenerator) GenerateDistributedProviders(totalCount int) []*models.Provider {
	providers := make([]*models.Provider, 0, totalCount)
	locationsCount := len(locations)

	// Distribute providers across locations
	for i := 0; i < totalCount; i++ {
		location := locations[i%locationsCount]

		// Generate provider for this location
		provider := fdg.GenerateRandomProvider()
		provider.Location = location

		providers = append(providers, provider)
	}

	return providers
}

// GenerateHighStakeProviders generates providers with high stakes
func (fdg *FakeDataGenerator) GenerateHighStakeProviders(count int) []*models.Provider {
	providers := make([]*models.Provider, count)
	for i := 0; i < count; i++ {
		provider := fdg.GenerateRandomProvider()
		// Set high stake (500000 to 1000000)
		provider.Stake = int64(fdg.Rand.Intn(500000) + 500000)
		providers[i] = provider
	}
	return providers
}

// GenerateFeatureRichProviders generates providers with many features
func (fdg *FakeDataGenerator) GenerateFeatureRichProviders(count int) []*models.Provider {
	providers := make([]*models.Provider, count)
	for i := 0; i < count; i++ {
		provider := fdg.GenerateRandomProvider()
		// Set many features (15-20 features)
		numFeatures := fdg.Rand.Intn(6) + 15
		selectedFeatures := make([]string, 0, numFeatures)
		usedFeatures := make(map[string]bool)

		for len(selectedFeatures) < numFeatures && len(selectedFeatures) < len(features) {
			feature := features[fdg.Rand.Intn(len(features))]
			if !usedFeatures[feature] {
				selectedFeatures = append(selectedFeatures, feature)
				usedFeatures[feature] = true
			}
		}

		provider.Features = selectedFeatures
		providers[i] = provider
	}
	return providers
}

// GenerateStrictPolicy generates a policy with strict requirements
func (fdg *FakeDataGenerator) GenerateStrictPolicy() *models.ConsumerPolicy {
	// Generate 2-4 unique features instead of 5 potentially duplicate ones
	numFeatures := fdg.Rand.Intn(3) + 2 // 2-4 features
	selectedFeatures := make([]string, 0, numFeatures)
	usedFeatures := make(map[string]bool)

	for len(selectedFeatures) < numFeatures {
		feature := features[fdg.Rand.Intn(len(features))]
		if !usedFeatures[feature] {
			selectedFeatures = append(selectedFeatures, feature)
			usedFeatures[feature] = true
		}
	}

	// Reduce minimum stake to more reasonable levels
	return &models.ConsumerPolicy{
		RequiredLocation: locations[fdg.Rand.Intn(len(locations))],
		RequiredFeatures: selectedFeatures,
		MinStake:         int64(fdg.Rand.Intn(30000) + 5000), // 5000-35000 instead of 20000-100000
	}
}

// GenerateRelaxedPolicy generates a policy with relaxed requirements
func (fdg *FakeDataGenerator) GenerateRelaxedPolicy() *models.ConsumerPolicy {
	return &models.ConsumerPolicy{
		RequiredLocation: "", // No location requirement
		RequiredFeatures: []string{
			features[fdg.Rand.Intn(len(features))],
		},
		MinStake: int64(fdg.Rand.Intn(5000)),
	}
}

// PopulateProviderStorage populates provider storage with fake data
func (fdg *FakeDataGenerator) PopulateProviderStorage(storage *storage.ProviderStorage, count int) {
	// Generate distributed providers
	distributedProviders := fdg.GenerateDistributedProviders(count / 2)
	for _, provider := range distributedProviders {
		storage.AddProvider(provider)
	}

	// Generate high stake providers
	highStakeProviders := fdg.GenerateHighStakeProviders(count / 4)
	for _, provider := range highStakeProviders {
		storage.AddProvider(provider)
	}

	// Generate feature-rich providers
	featureRichProviders := fdg.GenerateFeatureRichProviders(count / 4)
	for _, provider := range featureRichProviders {
		storage.AddProvider(provider)
	}
}

// GenerateTestScenario generates a complete test scenario
func (fdg *FakeDataGenerator) GenerateTestScenario() (*storage.ProviderStorage, []*models.ConsumerPolicy) {
	storage := storage.NewProviderStorage()

	// Populate with 100 providers
	fdg.PopulateProviderStorage(storage, 100)

	// Generate various policies
	policies := make([]*models.ConsumerPolicy, 0)

	// Add strict policies
	for i := 0; i < 5; i++ {
		policies = append(policies, fdg.GenerateStrictPolicy())
	}

	// Add relaxed policies
	for i := 0; i < 5; i++ {
		policies = append(policies, fdg.GenerateRelaxedPolicy())
	}

	// Add random policies
	randomPolicies := fdg.GenerateRandomPolicies(10)
	policies = append(policies, randomPolicies...)

	return storage, policies
}

// GetAvailableLocations returns all available locations
func (fdg *FakeDataGenerator) GetAvailableLocations() []string {
	return locations
}

// GetAvailableFeatures returns all available features
func (fdg *FakeDataGenerator) GetAvailableFeatures() []string {
	return features
}

// GenerateRealisticWorkload generates a realistic workload for testing
func (fdg *FakeDataGenerator) GenerateRealisticWorkload(providerCount, policyCount int) (*storage.ProviderStorage, []*models.ConsumerPolicy) {
	storage := storage.NewProviderStorage()

	// Generate realistic provider distribution
	// 40% distributed across locations
	distributedCount := int(float64(providerCount) * 0.4)
	distributedProviders := fdg.GenerateDistributedProviders(distributedCount)
	for _, provider := range distributedProviders {
		storage.AddProvider(provider)
	}

	// 30% high stake providers
	highStakeCount := int(float64(providerCount) * 0.3)
	highStakeProviders := fdg.GenerateHighStakeProviders(highStakeCount)
	for _, provider := range highStakeProviders {
		storage.AddProvider(provider)
	}

	// 20% feature-rich providers
	featureRichCount := int(float64(providerCount) * 0.2)
	featureRichProviders := fdg.GenerateFeatureRichProviders(featureRichCount)
	for _, provider := range featureRichProviders {
		storage.AddProvider(provider)
	}

	// 10% random providers
	randomCount := providerCount - distributedCount - highStakeCount - featureRichCount
	randomProviders := fdg.GenerateRandomProviders(randomCount)
	for _, provider := range randomProviders {
		storage.AddProvider(provider)
	}

	// Generate realistic policy distribution
	policies := make([]*models.ConsumerPolicy, 0, policyCount)

	// 30% strict policies
	strictCount := int(float64(policyCount) * 0.3)
	for i := 0; i < strictCount; i++ {
		policies = append(policies, fdg.GenerateStrictPolicy())
	}

	// 40% relaxed policies
	relaxedCount := int(float64(policyCount) * 0.4)
	for i := 0; i < relaxedCount; i++ {
		policies = append(policies, fdg.GenerateRelaxedPolicy())
	}

	// 30% random policies
	randomPolicyCount := policyCount - strictCount - relaxedCount
	randomPolicies := fdg.GenerateRandomPolicies(randomPolicyCount)
	policies = append(policies, randomPolicies...)

	return storage, policies
}
