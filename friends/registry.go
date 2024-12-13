package friends

import (
	"fmt"
	"github.com/google/uuid"
	"log"
	"sync"
	"time"
)

// Registry to store active servers
type Registry struct {
	mu sync.Mutex
	// keyed by REQUEST uuid
	requests map[uuid.UUID]FriendRequest
	// First is sender, second is recipient
}

// NewRegistry creates a new Registry instance
func NewRegistry() *Registry {
	return &Registry{requests: make(map[uuid.UUID]FriendRequest)}
}

func (r *Registry) AddOrUpdate(req FriendRequest) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.contains(req) {
		fmt.Printf("Friend request registry already contains %v", req)
		return
	}

	reqUUID := uuid.New()

	time.AfterFunc(req.Expiry.Sub(time.Now()), func() {
		r.expireRequest(reqUUID)
	})

	r.requests[reqUUID] = req

	log.Printf("Added: %+v", req)
	//todo send messages
}

func (r *Registry) Accept(id uuid.UUID) (bool, FriendRequest) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.containsKey(id) {
		req := r.requests[id]
		delete(r.requests, id)
		log.Printf("Accepted request: %+v", id)
		return true, req
	}
	log.Printf("Attempted to accept invalid request: %+v", id)
	return false, FriendRequest{}
}

// Decline  Functionally the same as Accept, but it sends a slightly different message. :)
func (r *Registry) Decline(id uuid.UUID) (bool, FriendRequest) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.containsKey(id) {
		req := r.requests[id]
		delete(r.requests, id)
		log.Printf("Decline request: %+v", id)
		return true, req
	}
	log.Printf("Attempted to decline invalid request: %+v", id)
	return false, FriendRequest{}
}

// GetAll returns all active servers
func (r *Registry) GetAll() []FriendRequest {
	r.mu.Lock()
	defer r.mu.Unlock()
	servers := make([]FriendRequest, 0, len(r.requests))
	for _, info := range r.requests {
		servers = append(servers, info)
	}
	return servers
}

func (r *Registry) Get(id uuid.UUID) FriendRequest {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.requests[id]
}

func (r *Registry) expireRequest(requestUUID uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.requests, requestUUID)
	//todo: broadcast expiry message
}

func (r *Registry) contains(request FriendRequest) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, friendRequest := range r.requests {
		if request == friendRequest {
			return true
		}
	}
	return false
}

func (r *Registry) containsKey(id uuid.UUID) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	for u := range r.requests {
		if id == u {
			return true
		}
	}
	return false
}
