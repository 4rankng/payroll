package tenantqueue

import (
	"sync"
)

// Manager keeps per-tenant queues
type Manager struct {
	mu      sync.Mutex
	queues  map[string]*TenantQueue
	workers int
	buffer  int
}

func NewManager(workers, buffer int) *Manager {
	return &Manager{queues: make(map[string]*TenantQueue), workers: workers, buffer: buffer}
}

func (m *Manager) GetOrCreate(tenant string) *TenantQueue {
	m.mu.Lock()
	defer m.mu.Unlock()
	if q, ok := m.queues[tenant]; ok {
		return q
	}
	q := NewTenantQueue(m.workers)
	m.queues[tenant] = q
	return q
}

func (m *Manager) ShutdownAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, q := range m.queues {
		q.Shutdown()
	}
}
