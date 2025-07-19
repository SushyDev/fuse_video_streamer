package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

type Node interface {
	ID() uint64
	Path() string
	Type() string
}

type Handle interface {
	ID() uint64
	NodeID() uint64
	Type() string
}

type Error struct {
	Err string
}

type debug struct {
	mu      sync.RWMutex
	nodes   map[uint64]Node
	handles map[uint64]Handle
	errors  map[uint64][]Error
}

var d = &debug{
	nodes:   make(map[uint64]Node),
	handles: make(map[uint64]Handle),
	errors:  make(map[uint64][]Error),
}

func RegisterNode(n Node) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.nodes[n.ID()] = n
}

func UnregisterNode(id uint64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.nodes, id)
	delete(d.errors, id)
}

func RegisterHandle(h Handle) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.handles[h.ID()] = h
}

func UnregisterHandle(id uint64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.handles, id)
	delete(d.errors, id)
}

func AddError(id uint64, err error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.errors[id] = append(d.errors[id], Error{Err: err.Error()})
}

func Start(port uint16) {
	mux := http.NewServeMux()
	mux.HandleFunc("/nodes", listNodes)
	mux.HandleFunc("/handles", listHandles)
	mux.HandleFunc("/errors/", getErrors)

	go http.ListenAndServe(fmt.Sprintf(":%d", port), mux)
}

func listNodes(w http.ResponseWriter, r *http.Request) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	json.NewEncoder(w).Encode(d.nodes)
}

func listHandles(w http.ResponseWriter, r *http.Request) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	json.NewEncoder(w).Encode(d.handles)
}

func getErrors(w http.ResponseWriter, r *http.Request) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	// a more robust way to get id from path is needed
	idStr := r.URL.Path[len("/errors/"):]
	var id uint64
	fmt.Sscanf(idStr, "%d", &id)
	json.NewEncoder(w).Encode(d.errors[id])
}
