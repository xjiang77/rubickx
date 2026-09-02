package singleton

import (
	"sort"
	"sync"
)

type PatternError struct{ code string }

func (e *PatternError) Error() string { return e.code }
func (e *PatternError) Code() string  { return e.code }

type ProviderCatalog struct {
	mu      sync.Mutex
	entries map[string]string
}

var instance *ProviderCatalog
var instanceMu sync.Mutex

func Instance() *ProviderCatalog {
	instanceMu.Lock()
	defer instanceMu.Unlock()
	if instance == nil {
		instance = &ProviderCatalog{entries: map[string]string{}}
	}
	return instance
}
func resetForTest() { instanceMu.Lock(); instance = nil; instanceMu.Unlock() }
func (c *ProviderCatalog) Register(name, endpoint string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if current, ok := c.entries[name]; ok && current != endpoint {
		return &PatternError{code: "registration_conflict"}
	}
	c.entries[name] = endpoint
	return nil
}
func (c *ProviderCatalog) Entries() map[string]string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := map[string]string{}
	keys := make([]string, 0, len(c.entries))
	for k := range c.entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		out[k] = c.entries[k]
	}
	return out
}
func Evaluate(input map[string]any) (any, error) {
	resetForTest()
	first, second := Instance(), Instance()
	for _, raw := range input["registrations"].([]any) {
		v := raw.(map[string]any)
		if err := first.Register(v["name"].(string), v["endpoint"].(string)); err != nil {
			return nil, err
		}
	}
	entries := second.Entries()
	return map[string]any{"same_instance": first == second, "entries": entries, "size": len(entries)}, nil
}
