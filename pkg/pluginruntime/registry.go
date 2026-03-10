package pluginruntime

import "sync"

// Registry is a threadsafe named registry for runtime extensions.
type Registry[T any] struct {
	mu    sync.RWMutex
	items map[string]T
}

// NewRegistry creates an empty registry.
func NewRegistry[T any]() *Registry[T] {
	return &Registry[T]{items: make(map[string]T)}
}

// Register stores an item under name.
func (r *Registry[T]) Register(name string, item T) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[name] = item
}

// Get returns the item registered under name.
func (r *Registry[T]) Get(name string) (T, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.items[name]
	return item, ok
}

// Delete removes an item from the registry.
func (r *Registry[T]) Delete(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.items, name)
}

// Names returns a snapshot of all registered names.
func (r *Registry[T]) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.items))
	for name := range r.items {
		names = append(names, name)
	}
	return names
}
