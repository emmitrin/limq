package api

import (
	"github.com/emmitrin/util"
	"sync"
)

type exclusiveAccess struct {
	access *util.Set[string]
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
	return &exclusiveAccess{access: util.NewSet[string](), mu: &sync.Mutex{}}
}
