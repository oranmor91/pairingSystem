package queue

import (
	"context"
	"fmt"
	"sync"
	"time"

	"pairingSystem/internal/models"
	"pairingSystem/internal/services"
	"pairingSystem/internal/storage"
)

// ConcurrentPairingSystem manages the complete concurrent pairing system
type ConcurrentPairingSystem struct {
	// Core components
	providerStorage *storage.ProviderStorage
	pairingSystem   *services.DefaultPairingSystem
	queueManager    *QueueManager

	// Worker management - simplified
	maxWorkers int
	workers    []*PairingWorker

	// System state
	mu      sync.RWMutex
	running bool
	ctx     context.Context
	cancel  context.CancelFunc

	// Statistics
	totalProcessed int64
	totalErrors    int64
	startTime      time.Time

	// Results channel
	resultsChan     chan *services.ProcessorResult
	resultsBuffer   []*services.ProcessorResult
	resultsBufferMu sync.Mutex
}

// PairingWorker represents a worker that processes policies
type PairingWorker struct {
	id            int
	pairingSystem *services.DefaultPairingSystem
	processed     int64
	errors        int64
	mu            sync.RWMutex

	// Worker control
	ctx         context.Context
	cancel      context.CancelFunc
	queue       *PolicyQueue
	resultsChan chan *services.ProcessorResult
	running     bool
}

// NewConcurrentPairingSystem creates a new concurrent pairing system
func NewConcurrentPairingSystem(providerStorage *storage.ProviderStorage, maxWorkers int, queueCapacity int) *ConcurrentPairingSystem {
	if maxWorkers <= 0 {
		maxWorkers = 10 // Default number of workers
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &ConcurrentPairingSystem{
		providerStorage: providerStorage,
		pairingSystem:   services.NewDefaultPairingSystem(providerStorage, 100),
		queueManager:    NewQueueManager(queueCapacity),
		maxWorkers:      maxWorkers,
		workers:         make([]*PairingWorker, 0, maxWorkers),
		ctx:             ctx,
		cancel:          cancel,
		resultsChan:     make(chan *services.ProcessorResult, 1000),
		resultsBuffer:   make([]*services.ProcessorResult, 0),
	}
}

// NewPairingWorker creates a new pairing worker
func NewPairingWorker(id int, pairingSystem *services.DefaultPairingSystem, queue *PolicyQueue, resultsChan chan *services.ProcessorResult) *PairingWorker {
	ctx, cancel := context.WithCancel(context.Background())
	return &PairingWorker{
		id:            id,
		pairingSystem: pairingSystem,
		ctx:           ctx,
		cancel:        cancel,
		queue:         queue,
		resultsChan:   resultsChan,
	}
}

// Start starts the worker's processing loop
func (pw *PairingWorker) Start() {
	pw.mu.Lock()
	defer pw.mu.Unlock()

	if pw.running {
		return
	}

	pw.running = true
	go pw.workLoop()
}

// Stop stops the worker
func (pw *PairingWorker) Stop() {
	pw.mu.Lock()
	defer pw.mu.Unlock()

	if !pw.running {
		return
	}

	pw.running = false
	pw.cancel()
}

// workLoop is the main worker loop - this is the efficient pattern!
func (pw *PairingWorker) workLoop() {
	for {
		select {
		case <-pw.ctx.Done():
			return
		default:
			// Worker directly pulls from queue when ready
			policy, err := pw.queue.DequeueWithContext(pw.ctx)
			if err != nil {
				// If queue is empty or context cancelled, continue
				if err == context.Canceled {
					return
				}
				// Add small delay to prevent tight error loops
				select {
				case <-pw.ctx.Done():
					return
				case <-time.After(10 * time.Millisecond):
					continue
				}
			}

			// Process the policy immediately
			result := pw.ProcessPolicy(policy)

			// Send result to results channel
			select {
			case pw.resultsChan <- result:
			case <-pw.ctx.Done():
				return
			}
		}
	}
}

// ProcessPolicy processes a single policy
func (pw *PairingWorker) ProcessPolicy(policy *models.ConsumerPolicy) *services.ProcessorResult {
	pw.mu.Lock()
	defer pw.mu.Unlock()

	// Validate policy before processing
	if policy == nil {
		pw.errors++
		return &services.ProcessorResult{
			Policy: nil,
			Error:  fmt.Errorf("policy cannot be nil"),
		}
	}

	result, err := pw.pairingSystem.GetPairingListWithDetails(nil, policy)
	if err != nil {
		pw.errors++
		return &services.ProcessorResult{
			Policy: policy,
			Error:  fmt.Errorf("failed to process policy: %w", err),
		}
	}

	pw.processed++
	return result
}

// IsRunning returns whether the worker is running
func (pw *PairingWorker) IsRunning() bool {
	pw.mu.RLock()
	defer pw.mu.RUnlock()
	return pw.running
}

// GetStats returns worker statistics
func (pw *PairingWorker) GetStats() map[string]interface{} {
	pw.mu.RLock()
	defer pw.mu.RUnlock()

	return map[string]interface{}{
		"id":        pw.id,
		"processed": pw.processed,
		"errors":    pw.errors,
		"running":   pw.running,
	}
}

// Start starts the concurrent pairing system with worker-driven processing
func (cps *ConcurrentPairingSystem) Start() error {
	cps.mu.Lock()
	defer cps.mu.Unlock()

	if cps.running {
		return fmt.Errorf("system is already running")
	}

	// Create and start workers - they will directly pull from queue
	for i := 0; i < cps.maxWorkers; i++ {
		worker := NewPairingWorker(i, cps.pairingSystem, cps.queueManager.GetQueue(), cps.resultsChan)
		cps.workers = append(cps.workers, worker)
		worker.Start() // Each worker starts its own processing loop
	}

	// Start results collector
	go cps.collectResults()

	// Start queue system (for producers)
	cps.queueManager.StartAll()

	cps.running = true
	cps.startTime = time.Now()

	return nil
}

// Stop stops the concurrent pairing system
func (cps *ConcurrentPairingSystem) Stop() {
	cps.mu.Lock()
	defer cps.mu.Unlock()

	if !cps.running {
		return
	}

	// Stop all workers
	for _, worker := range cps.workers {
		worker.Stop()
	}

	// Stop queue system
	cps.queueManager.StopAll()

	// Cancel context to stop all goroutines
	cps.cancel()

	// Close results channel
	close(cps.resultsChan)

	cps.running = false
}

// collectResults collects processing results
func (cps *ConcurrentPairingSystem) collectResults() {
	for {
		select {
		case <-cps.ctx.Done():
			return
		case result, ok := <-cps.resultsChan:
			if !ok {
				return
			}

			// Add result to buffer
			cps.resultsBufferMu.Lock()
			cps.resultsBuffer = append(cps.resultsBuffer, result)
			cps.resultsBufferMu.Unlock()

			// Update statistics
			cps.mu.Lock()
			cps.totalProcessed++
			if result.Error != nil {
				cps.totalErrors++
			}
			cps.mu.Unlock()
		}
	}
}

// EnqueuePolicy adds a policy to the processing queue
func (cps *ConcurrentPairingSystem) EnqueuePolicy(policy *models.ConsumerPolicy) error {
	if !cps.IsRunning() {
		return fmt.Errorf("system is not running")
	}

	producer := cps.queueManager.AddProducer()
	producer.Start()
	defer producer.Stop()

	return producer.ProducePolicy(policy)
}

// EnqueuePolicies adds multiple policies to the processing queue
func (cps *ConcurrentPairingSystem) EnqueuePolicies(policies []*models.ConsumerPolicy) error {
	if !cps.IsRunning() {
		return fmt.Errorf("system is not running")
	}

	producer := cps.queueManager.AddProducer()
	producer.Start()
	defer producer.Stop()

	return producer.ProducePoliciesBatch(policies)
}

// GetResults returns and clears the results buffer
func (cps *ConcurrentPairingSystem) GetResults() []*services.ProcessorResult {
	cps.resultsBufferMu.Lock()
	defer cps.resultsBufferMu.Unlock()

	results := cps.resultsBuffer
	cps.resultsBuffer = make([]*services.ProcessorResult, 0)
	return results
}

// GetResultsCount returns the current number of results in the buffer
func (cps *ConcurrentPairingSystem) GetResultsCount() int {
	cps.resultsBufferMu.Lock()
	defer cps.resultsBufferMu.Unlock()

	return len(cps.resultsBuffer)
}

// IsRunning returns whether the system is running
func (cps *ConcurrentPairingSystem) IsRunning() bool {
	cps.mu.RLock()
	defer cps.mu.RUnlock()

	return cps.running
}

// GetStats returns comprehensive system statistics
func (cps *ConcurrentPairingSystem) GetStats() map[string]interface{} {
	cps.mu.RLock()
	defer cps.mu.RUnlock()

	var uptime time.Duration
	if cps.running {
		uptime = time.Since(cps.startTime)
	}

	// Get worker statistics
	workerStats := make([]map[string]interface{}, len(cps.workers))
	for i, worker := range cps.workers {
		workerStats[i] = worker.GetStats()
	}

	return map[string]interface{}{
		"running":             cps.running,
		"uptime":              uptime,
		"total_processed":     cps.totalProcessed,
		"total_errors":        cps.totalErrors,
		"max_workers":         cps.maxWorkers,
		"worker_stats":        workerStats,
		"results_buffer_size": cps.GetResultsCount(),
		"queue_stats":         cps.queueManager.GetStats(),
	}
}

// GetQueueStats returns queue system statistics
func (cps *ConcurrentPairingSystem) GetQueueStats() map[string]interface{} {
	return cps.queueManager.GetStats()
}

// GetWorkerStats returns detailed worker statistics
func (cps *ConcurrentPairingSystem) GetWorkerStats() []map[string]interface{} {
	cps.mu.RLock()
	defer cps.mu.RUnlock()

	stats := make([]map[string]interface{}, len(cps.workers))
	for i, worker := range cps.workers {
		stats[i] = worker.GetStats()
	}
	return stats
}

// GetPairingSystemStats returns pairing system statistics
func (cps *ConcurrentPairingSystem) GetPairingSystemStats() map[string]interface{} {
	return cps.pairingSystem.GetStats()
}

// GetProviderCount returns the number of providers in storage
func (cps *ConcurrentPairingSystem) GetProviderCount() int {
	return cps.providerStorage.GetProviderCount()
}

// AddProvider adds a provider to the storage
func (cps *ConcurrentPairingSystem) AddProvider(provider *models.Provider) {
	cps.providerStorage.AddProvider(provider)
}

// GetProviderStorage returns the provider storage (for testing/debugging)
func (cps *ConcurrentPairingSystem) GetProviderStorage() *storage.ProviderStorage {
	return cps.providerStorage
}

// GetPairingSystem returns the pairing system (for testing/debugging)
func (cps *ConcurrentPairingSystem) GetPairingSystem() *services.DefaultPairingSystem {
	return cps.pairingSystem
}

// GetQueueManager returns the queue manager (for testing/debugging)
func (cps *ConcurrentPairingSystem) GetQueueManager() *QueueManager {
	return cps.queueManager
}

// ProcessPolicyDirectly processes a policy directly without using the queue
func (cps *ConcurrentPairingSystem) ProcessPolicyDirectly(policy *models.ConsumerPolicy) (*services.ProcessorResult, error) {
	if !cps.IsRunning() {
		return nil, fmt.Errorf("system is not running")
	}

	return cps.pairingSystem.GetPairingListWithDetails(nil, policy)
}

// ProcessPoliciesDirectly processes multiple policies directly without using the queue
func (cps *ConcurrentPairingSystem) ProcessPoliciesDirectly(policies []*models.ConsumerPolicy) ([]*services.ProcessorResult, error) {
	if !cps.IsRunning() {
		return nil, fmt.Errorf("system is not running")
	}

	return cps.pairingSystem.GetPairingListConcurrently(nil, policies, cps.maxWorkers)
}

// ResetStats resets all statistics
func (cps *ConcurrentPairingSystem) ResetStats() {
	cps.mu.Lock()
	defer cps.mu.Unlock()

	cps.totalProcessed = 0
	cps.totalErrors = 0
	cps.startTime = time.Now()

	// Reset worker stats
	for _, worker := range cps.workers {
		worker.mu.Lock()
		worker.processed = 0
		worker.errors = 0
		worker.mu.Unlock()
	}

	// Reset pairing system stats
	cps.pairingSystem.ResetStats()

	// Clear results buffer
	cps.resultsBufferMu.Lock()
	cps.resultsBuffer = make([]*services.ProcessorResult, 0)
	cps.resultsBufferMu.Unlock()
}
