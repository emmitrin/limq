package api

import (
	"limq/internal/set"
	"sync"
)

type exclusiveAccess struct {
	access *set.Set[string]
	mu     *sync.Mutex
}

func (ea *exclusiveAccess) start(key string) bool {
	ea.mu.Lock()
	defer ea.mu.Unlock()

	return ea.access.Add(key)
}

func (ea *exclusiveAccess) stop(key string) {
	ea.mu.Lock()
	defer ea.mu.Unlock()

	ea.access.Delete(key)
}

func newEA() *exclusiveAccess {
	return &exclusiveAccess{access: set.NewSet[string](nil), mu: &sync.Mutex{}}
}
