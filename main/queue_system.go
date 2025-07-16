package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// PolicyQueue represents a thread-safe queue for consumer policies
type PolicyQueue struct {
	queue    chan *ConsumerPolicy
	closed   bool
	mu       sync.RWMutex
	capacity int
	enqueued int64
	dequeued int64
}

// NewPolicyQueue creates a new policy queue with specified capacity
func NewPolicyQueue(capacity int) *PolicyQueue {
	if capacity <= 0 {
		capacity = 1000 // Default capacity
	}

	return &PolicyQueue{
		queue:    make(chan *ConsumerPolicy, capacity),
		capacity: capacity,
	}
}

// Enqueue adds a policy to the queue (non-blocking)
func (pq *PolicyQueue) Enqueue(policy *ConsumerPolicy) error {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	if pq.closed {
		return fmt.Errorf("queue is closed")
	}

	select {
	case pq.queue <- policy:
		pq.enqueued++
		return nil
	default:
		return fmt.Errorf("queue is full")
	}
}

// EnqueueWithTimeout adds a policy to the queue with timeout
func (pq *PolicyQueue) EnqueueWithTimeout(policy *ConsumerPolicy, timeout time.Duration) error {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	if pq.closed {
		return fmt.Errorf("queue is closed")
	}

	select {
	case pq.queue <- policy:
		pq.enqueued++
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("timeout while enqueueing policy")
	}
}

// Dequeue removes and returns a policy from the queue (blocking)
func (pq *PolicyQueue) Dequeue() (*ConsumerPolicy, error) {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	if pq.closed {
		return nil, fmt.Errorf("queue is closed")
	}

	policy, ok := <-pq.queue
	if !ok {
		return nil, fmt.Errorf("queue is closed")
	}

	pq.dequeued++
	return policy, nil
}

// DequeueWithTimeout removes and returns a policy from the queue with timeout
func (pq *PolicyQueue) DequeueWithTimeout(timeout time.Duration) (*ConsumerPolicy, error) {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	if pq.closed {
		return nil, fmt.Errorf("queue is closed")
	}

	select {
	case policy, ok := <-pq.queue:
		if !ok {
			return nil, fmt.Errorf("queue is closed")
		}
		pq.dequeued++
		return policy, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout while dequeuing policy")
	}
}

// DequeueWithContext removes and returns a policy from the queue with context
func (pq *PolicyQueue) DequeueWithContext(ctx context.Context) (*ConsumerPolicy, error) {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	if pq.closed {
		return nil, fmt.Errorf("queue is closed")
	}

	select {
	case policy, ok := <-pq.queue:
		if !ok {
			return nil, fmt.Errorf("queue is closed")
		}
		pq.dequeued++
		return policy, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Size returns the current number of policies in the queue
func (pq *PolicyQueue) Size() int {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	return len(pq.queue)
}

// Capacity returns the maximum capacity of the queue
func (pq *PolicyQueue) Capacity() int {
	return pq.capacity
}

// IsClosed returns whether the queue is closed
func (pq *PolicyQueue) IsClosed() bool {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	return pq.closed
}

// Close closes the queue
func (pq *PolicyQueue) Close() {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	if !pq.closed {
		pq.closed = true
		close(pq.queue)
	}
}

// GetStats returns queue statistics
func (pq *PolicyQueue) GetStats() map[string]interface{} {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	return map[string]interface{}{
		"size":     len(pq.queue),
		"capacity": pq.capacity,
		"enqueued": pq.enqueued,
		"dequeued": pq.dequeued,
		"pending":  pq.enqueued - pq.dequeued,
		"closed":   pq.closed,
	}
}

// PolicyProducer generates and enqueues policies
type PolicyProducer struct {
	queue    *PolicyQueue
	active   bool
	mu       sync.RWMutex
	produced int64
	errors   int64
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewPolicyProducer creates a new policy producer
func NewPolicyProducer(queue *PolicyQueue) *PolicyProducer {
	ctx, cancel := context.WithCancel(context.Background())
	return &PolicyProducer{
		queue:  queue,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start starts the policy producer
func (pp *PolicyProducer) Start() {
	pp.mu.Lock()
	defer pp.mu.Unlock()

	pp.active = true
}

// Stop stops the policy producer
func (pp *PolicyProducer) Stop() {
	pp.mu.Lock()
	defer pp.mu.Unlock()

	pp.active = false
	pp.cancel()
}

// IsActive returns whether the producer is active
func (pp *PolicyProducer) IsActive() bool {
	pp.mu.RLock()
	defer pp.mu.RUnlock()

	return pp.active
}

// ProducePolicy creates and enqueues a single policy
func (pp *PolicyProducer) ProducePolicy(policy *ConsumerPolicy) error {
	pp.mu.RLock()
	defer pp.mu.RUnlock()

	if !pp.active {
		return fmt.Errorf("producer is not active")
	}

	err := pp.queue.Enqueue(policy)
	if err != nil {
		pp.errors++
		return err
	}

	pp.produced++
	return nil
}

// ProducePoliciesBatch creates and enqueues multiple policies
func (pp *PolicyProducer) ProducePoliciesBatch(policies []*ConsumerPolicy) error {
	pp.mu.RLock()
	defer pp.mu.RUnlock()

	if !pp.active {
		return fmt.Errorf("producer is not active")
	}

	for _, policy := range policies {
		err := pp.queue.Enqueue(policy)
		if err != nil {
			pp.errors++
			return fmt.Errorf("failed to enqueue policy: %w", err)
		}
		pp.produced++
	}

	return nil
}

// GetStats returns producer statistics
func (pp *PolicyProducer) GetStats() map[string]interface{} {
	pp.mu.RLock()
	defer pp.mu.RUnlock()

	return map[string]interface{}{
		"active":   pp.active,
		"produced": pp.produced,
		"errors":   pp.errors,
	}
}

// PolicyConsumer consumes policies from the queue
type PolicyConsumer struct {
	queue     *PolicyQueue
	active    bool
	mu        sync.RWMutex
	consumed  int64
	errors    int64
	ctx       context.Context
	cancel    context.CancelFunc
	processFn func(*ConsumerPolicy) error
}

// NewPolicyConsumer creates a new policy consumer
func NewPolicyConsumer(queue *PolicyQueue, processFn func(*ConsumerPolicy) error) *PolicyConsumer {
	ctx, cancel := context.WithCancel(context.Background())
	return &PolicyConsumer{
		queue:     queue,
		processFn: processFn,
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Start starts the policy consumer
func (pc *PolicyConsumer) Start() {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	pc.active = true
}

// Stop stops the policy consumer
func (pc *PolicyConsumer) Stop() {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	pc.active = false
	pc.cancel()
}

// IsActive returns whether the consumer is active
func (pc *PolicyConsumer) IsActive() bool {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	return pc.active
}

// ConsumePolicy consumes and processes a single policy
func (pc *PolicyConsumer) ConsumePolicy() error {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	if !pc.active {
		return fmt.Errorf("consumer is not active")
	}

	policy, err := pc.queue.DequeueWithContext(pc.ctx)
	if err != nil {
		pc.errors++
		return err
	}

	if pc.processFn != nil {
		err = pc.processFn(policy)
		if err != nil {
			pc.errors++
			return fmt.Errorf("failed to process policy: %w", err)
		}
	}

	pc.consumed++
	return nil
}

// ConsumeLoop continuously consumes policies until stopped
func (pc *PolicyConsumer) ConsumeLoop() {
	for {
		select {
		case <-pc.ctx.Done():
			return
		default:
			err := pc.ConsumePolicy()
			if err != nil {
				// Log error or handle appropriately
				// Add a small delay to prevent tight error loops
				select {
				case <-pc.ctx.Done():
					return
				case <-time.After(10 * time.Millisecond):
					continue
				}
			}
		}
	}
}

// GetStats returns consumer statistics
func (pc *PolicyConsumer) GetStats() map[string]interface{} {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	return map[string]interface{}{
		"active":   pc.active,
		"consumed": pc.consumed,
		"errors":   pc.errors,
	}
}

// QueueManager manages the overall queue system
type QueueManager struct {
	queue     *PolicyQueue
	producers []*PolicyProducer
	mu        sync.RWMutex
}

// NewQueueManager creates a new queue manager
func NewQueueManager(queueCapacity int) *QueueManager {
	return &QueueManager{
		queue:     NewPolicyQueue(queueCapacity),
		producers: make([]*PolicyProducer, 0),
	}
}

// AddProducer adds a new producer to the manager
func (qm *QueueManager) AddProducer() *PolicyProducer {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	producer := NewPolicyProducer(qm.queue)
	qm.producers = append(qm.producers, producer)
	return producer
}

// GetQueue returns the managed queue
func (qm *QueueManager) GetQueue() *PolicyQueue {
	return qm.queue
}

// StartAll starts all producers
func (qm *QueueManager) StartAll() {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	for _, producer := range qm.producers {
		producer.Start()
	}
}

// StopAll stops all producers
func (qm *QueueManager) StopAll() {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	for _, producer := range qm.producers {
		producer.Stop()
	}
}

// GetStats returns comprehensive queue manager statistics
func (qm *QueueManager) GetStats() map[string]interface{} {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	producerStats := make([]map[string]interface{}, len(qm.producers))
	for i, producer := range qm.producers {
		producerStats[i] = producer.GetStats()
	}

	return map[string]interface{}{
		"queue":     qm.queue.GetStats(),
		"producers": producerStats,
	}
}
