package membership

import (
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"ringg/internal/ring"
)

const (
	StatusAlive = "alive"
	StatusLeft  = "left"
)

var (
	ErrBlankNodeID    = errors.New("node id is required")
	ErrBlankNodeAddr  = errors.New("node address is required")
	ErrMemberNotFound = errors.New("member not found")
	ErrNoAliveMembers = errors.New("membership has no alive members")
)

type Member struct {
	NodeID    string    `json:"node_id"`
	Addr      string    `json:"addr"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Snapshot struct {
	LocalNodeID string   `json:"local_node_id"`
	Members     []Member `json:"members"`
	RingNodes   []string `json:"ring_nodes"`
}

type State struct {
	mu          sync.RWMutex
	localNodeID string
	members     map[string]Member
	ring        *ring.Ring
}

func NewState(local Member) (*State, error) {
	if err := validateMember(local); err != nil {
		return nil, err
	}

	if strings.TrimSpace(local.Status) == "" {
		local.Status = StatusAlive
	}
	if local.UpdatedAt.IsZero() {
		local.UpdatedAt = time.Now().UTC()
	}

	state := &State{
		localNodeID: local.NodeID,
		members:     map[string]Member{local.NodeID: local},
		ring:        ring.New(),
	}
	state.rebuildRingLocked()

	return state, nil
}

func (s *State) LocalNodeID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.localNodeID
}

func (s *State) Upsert(member Member) error {
	if err := validateMember(member); err != nil {
		return err
	}

	member.Status = normalizeStatus(member.Status)
	if member.UpdatedAt.IsZero() {
		member.UpdatedAt = time.Now().UTC()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	current, ok := s.members[member.NodeID]
	if ok && current.UpdatedAt.After(member.UpdatedAt) {
		return nil
	}

	s.members[member.NodeID] = member
	s.rebuildRingLocked()

	return nil
}

func (s *State) MergeSnapshot(snapshot Snapshot) error {
	for _, member := range snapshot.Members {
		if err := s.Upsert(member); err != nil {
			return err
		}
	}
	return nil
}

func (s *State) MarkLeft(nodeID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	member, ok := s.members[nodeID]
	if !ok {
		return ErrMemberNotFound
	}

	member.Status = StatusLeft
	member.UpdatedAt = time.Now().UTC()
	s.members[nodeID] = member
	s.rebuildRingLocked()

	return nil
}

func (s *State) Get(nodeID string) (Member, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	member, ok := s.members[nodeID]
	return member, ok
}

func (s *State) GetOwner(key string) (Member, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ownerNodeID, err := s.ring.GetOwner(key)
	if err != nil {
		if errors.Is(err, ring.ErrEmptyRing) {
			return Member{}, ErrNoAliveMembers
		}
		return Member{}, err
	}

	member, ok := s.members[ownerNodeID]
	if !ok {
		return Member{}, ErrMemberNotFound
	}

	return member, nil
}

func (s *State) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	members := make([]Member, 0, len(s.members))
	for _, member := range s.members {
		members = append(members, member)
	}
	sort.Slice(members, func(i, j int) bool {
		return members[i].NodeID < members[j].NodeID
	})

	return Snapshot{
		LocalNodeID: s.localNodeID,
		Members:     members,
		RingNodes:   s.ring.Nodes(),
	}
}

func (s *State) rebuildRingLocked() {
	newRing := ring.New()
	for _, member := range s.members {
		if member.Status == StatusAlive {
			newRing.AddNode(member.NodeID)
		}
	}
	s.ring = newRing
}

func normalizeStatus(status string) string {
	status = strings.TrimSpace(status)
	if status == "" {
		return StatusAlive
	}
	return status
}

func validateMember(member Member) error {
	member.NodeID = strings.TrimSpace(member.NodeID)
	member.Addr = strings.TrimSpace(member.Addr)

	if member.NodeID == "" {
		return ErrBlankNodeID
	}
	if member.Addr == "" {
		return ErrBlankNodeAddr
	}

	return nil
}
