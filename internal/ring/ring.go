package ring

import (
	"errors"
	"hash/crc32"
	"sort"
	"sync"
)

var ErrEmptyRing = errors.New("hash ring has no nodes")

type Ring struct {
	mu        sync.RWMutex
	positions []uint32
	nodes     map[uint32]string
}

func New() *Ring {
	return &Ring{
		nodes: make(map[uint32]string),
	}
}

func (r *Ring) AddNode(nodeID string) {
	position := hash(nodeID)

	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, ok := r.nodes[position]; ok && existing == nodeID {
		return
	}

	r.nodes[position] = nodeID
	r.rebuildPositions()
}

func (r *Ring) RemoveNode(nodeID string) {
	position := hash(nodeID)

	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.nodes[position]
	if !ok || existing != nodeID {
		return
	}

	delete(r.nodes, position)
	r.rebuildPositions()
}

func (r *Ring) GetOwner(key string) (string, error) {
	keyPosition := hash(key)

	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.positions) == 0 {
		return "", ErrEmptyRing
	}

	index := sort.Search(len(r.positions), func(i int) bool {
		return r.positions[i] >= keyPosition
	})
	if index == len(r.positions) {
		index = 0
	}

	return r.nodes[r.positions[index]], nil
}

func (r *Ring) Nodes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	nodeIDs := make([]string, 0, len(r.positions))
	for _, position := range r.positions {
		nodeIDs = append(nodeIDs, r.nodes[position])
	}

	return nodeIDs
}

func (r *Ring) rebuildPositions() {
	r.positions = r.positions[:0]
	for position := range r.nodes {
		r.positions = append(r.positions, position)
	}
	sort.Slice(r.positions, func(i, j int) bool {
		return r.positions[i] < r.positions[j]
	})
}

func hash(value string) uint32 {
	return crc32.ChecksumIEEE([]byte(value))
}
