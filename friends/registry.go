package friends

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"log"
	"sync"
	"time"
)

// Registry to store active servers
type Registry struct {
	mu sync.Mutex
	// keyed by REQUEST uuid
	requests map[uuid.UUID]FriendRequest
	nats     *nats.Conn
	// First is sender, second is recipient
}

// NewRegistry creates a new Registry instance
func NewRegistry(nc *nats.Conn) *Registry {
	return &Registry{requests: make(map[uuid.UUID]FriendRequest), nats: nc}
}

func (r *Registry) AddOrUpdate(req FriendRequest) (bool, bool) {
	defer r.mu.Unlock()
	r.mu.Lock()

	if r.contains(req) {
		log.Printf("Friend request registry already contains %v", req)
		return false, false
	}

	if r.containsInverse(req) {
		log.Printf("Friend request registry contains an inverted request! %v", req)
		flipped := FriendRequest{
			Sender:    req.Recipient,
			Recipient: req.Sender,
			Expiry:    req.Expiry,
		}
		SendAcceptance(r.nats, flipped)
		return true, true // successful
	}

	reqUUID := uuid.New()

	log.Printf("Friend request registry adding %v", reqUUID)
	go func() {
		time.AfterFunc(req.Expiry.Sub(time.Now()), func() {
			log.Printf("Friend request registry expired %v", req)
			r.expireRequest(reqUUID)
		})
	}()

	r.requests[reqUUID] = req

	return true, false
}

func (r *Registry) AcceptByID(id uuid.UUID) (bool, FriendRequest) {
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

func (r *Registry) Accept(sender uuid.UUID, recipient uuid.UUID) (bool, FriendRequest) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var req FriendRequest
	var id uuid.UUID
	var found bool

	for u, request := range r.requests {
		if request.Sender == sender && request.Recipient == recipient {
			// remove this one :)
			req = request
			id = u
			found = true
		}
	}

	if found {
		delete(r.requests, id)
		log.Printf("Accepted request: %+v", id)
		return true, req
	}

	log.Printf("Attempted to accept invalid request: %+v", id)
	return false, FriendRequest{}
}

// DeclineByID  Functionally the same as AcceptByID, but it sends a slightly different message. :)
func (r *Registry) DeclineByID(id uuid.UUID) (bool, FriendRequest) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.containsKey(id) {
		req := r.requests[id]
		delete(r.requests, id)
		log.Printf("DeclineByID request: %+v", id)
		return true, req
	}
	log.Printf("Attempted to decline invalid request: %+v", id)
	return false, FriendRequest{}
}

func (r *Registry) Decline(sender uuid.UUID, recipient uuid.UUID) (bool, FriendRequest) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var req FriendRequest
	var id uuid.UUID
	var found bool

	for u, request := range r.requests {
		if request.Sender == sender && request.Recipient == recipient {
			// remove this one :)
			req = request
			id = u
			found = true
		}
	}

	if found {
		delete(r.requests, id)
		log.Printf("Declined request: %+v", id)
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
	request := r.requests[requestUUID]
	delete(r.requests, requestUUID)

	const subject = "friends.expire.notify"
	data, err := json.Marshal(request)
	if err != nil {
		log.Printf("Error marshalling request: %v", err)
		return
	}
	err1 := r.nats.Publish(subject, data)
	if err1 != nil {
		log.Printf("Error publishing request expiry: %v", err1)
		return
	}
}

func (r *Registry) contains(request FriendRequest) bool {
	for _, friendRequest := range r.requests {
		log.Printf("Friend request registry contains %v", friendRequest)
		if request.Sender == friendRequest.Sender && request.Recipient == friendRequest.Recipient {
			return true
		}
	}
	return false
}

func (r *Registry) containsInverse(request FriendRequest) bool {
	for _, friendRequest := range r.requests {
		if request.Sender == friendRequest.Recipient && request.Recipient == friendRequest.Sender {
			return true
		}
	}
	return false
}

func (r *Registry) containsKey(id uuid.UUID) bool {
	for u := range r.requests {
		if id == u {
			return true
		}
	}
	return false
}

func SendAcceptance(nc *nats.Conn, req FriendRequest) {

	const subject = "friends.accept.notify"
	marshal, errr := json.Marshal(req)
	if errr != nil {
		log.Fatalf("Error marshalling friend request: %v", errr)
		return
	}
	err := nc.Publish(subject, marshal)
	if err != nil {
		log.Fatalf("Error publishing friends acceptance message: %v", err)
		return
	}
}
