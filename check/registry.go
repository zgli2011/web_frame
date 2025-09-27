package check

import (
	"sync"
)

// APIInfo represents the information for a registered API endpoint
type APIInfo struct {
	Path        string `json:"path"`        // API路径
	Method      string `json:"method"`      // HTTP方法
	Tag         string `json:"tag"`         // API标签/分组
	Description string `json:"description"` // API描述
	Service     string `json:"service"`     // 服务名称
	Handler     string `json:"handler"`     // 处理器名称
	Summary     string `json:"summary"`     // API摘要
	Deprecated  bool   `json:"deprecated"`  // 是否已废弃
}

// APIRegistry manages registered API endpoints
type APIRegistry struct {
	mu   sync.RWMutex
	apis map[string]*APIInfo // key: method:path
}

var (
	globalRegistry = &APIRegistry{
		apis: make(map[string]*APIInfo),
	}
)

// RegisterAPI registers a new API endpoint
func RegisterAPI(info APIInfo) {
	globalRegistry.RegisterAPI(info)
}

// GetAllAPIs returns all registered API endpoints
func GetAllAPIs() []*APIInfo {
	return globalRegistry.GetAllAPIs()
}

// GetAPIsByTag returns APIs filtered by tag
func GetAPIsByTag(tag string) []*APIInfo {
	return globalRegistry.GetAPIsByTag(tag)
}

// GetAPIsByService returns APIs filtered by service
func GetAPIsByService(service string) []*APIInfo {
	return globalRegistry.GetAPIsByService(service)
}

// ClearAPIs clears all registered APIs (mainly for testing)
func ClearAPIs() {
	globalRegistry.ClearAPIs()
}

// RegisterAPI registers a new API endpoint
func (r *APIRegistry) RegisterAPI(info APIInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := info.Method + ":" + info.Path
	r.apis[key] = &info
}

// GetAllAPIs returns all registered API endpoints
func (r *APIRegistry) GetAllAPIs() []*APIInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*APIInfo, 0, len(r.apis))
	for _, api := range r.apis {
		result = append(result, api)
	}
	return result
}

// GetAPIsByTag returns APIs filtered by tag
func (r *APIRegistry) GetAPIsByTag(tag string) []*APIInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*APIInfo
	for _, api := range r.apis {
		if api.Tag == tag {
			result = append(result, api)
		}
	}
	return result
}

// GetAPIsByService returns APIs filtered by service
func (r *APIRegistry) GetAPIsByService(service string) []*APIInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*APIInfo
	for _, api := range r.apis {
		if api.Service == service {
			result = append(result, api)
		}
	}
	return result
}

// ClearAPIs clears all registered APIs
func (r *APIRegistry) ClearAPIs() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.apis = make(map[string]*APIInfo)
}

// GetAPICount returns the number of registered APIs
func (r *APIRegistry) GetAPICount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.apis)
}

// HasAPI checks if an API is registered
func (r *APIRegistry) HasAPI(method, path string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	key := method + ":" + path
	_, exists := r.apis[key]
	return exists
}