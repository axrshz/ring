// cluster/consistent_hash.go
package cluster

import (
	"hash/crc32"
	"sort"
	"sync"
)

type ConsistentHash struct {
    mu           sync.RWMutex
    hashRing     []uint32
    nodeMap      map[uint32]string
    virtualNodes int
}

func NewConsistentHash(virtualNodes int) *ConsistentHash {
    return &ConsistentHash{
        nodeMap:      make(map[uint32]string),
        virtualNodes: virtualNodes,
    }
}

func (ch *ConsistentHash) AddNode(node string) {
    ch.mu.Lock()
    defer ch.mu.Unlock()

    for i := 0; i < ch.virtualNodes; i++ {
        hash := ch.hash(node + string(rune(i)))
        ch.hashRing = append(ch.hashRing, hash)
        ch.nodeMap[hash] = node
    }
    sort.Slice(ch.hashRing, func(i, j int) bool {
        return ch.hashRing[i] < ch.hashRing[j]
    })
}

func (ch *ConsistentHash) RemoveNode(node string) {
    ch.mu.Lock()
    defer ch.mu.Unlock()

    for i := 0; i < ch.virtualNodes; i++ {
        hash := ch.hash(node + string(rune(i)))
        idx := sort.Search(len(ch.hashRing), func(i int) bool {
            return ch.hashRing[i] >= hash
        })
        if idx < len(ch.hashRing) && ch.hashRing[idx] == hash {
            ch.hashRing = append(ch.hashRing[:idx], ch.hashRing[idx+1:]...)
        }
        delete(ch.nodeMap, hash)
    }
}

func (ch *ConsistentHash) GetNode(key string) string {
    ch.mu.RLock()
    defer ch.mu.RUnlock()

    if len(ch.hashRing) == 0 {
        return ""
    }

    hash := ch.hash(key)
    idx := sort.Search(len(ch.hashRing), func(i int) bool {
        return ch.hashRing[i] >= hash
    })

    if idx == len(ch.hashRing) {
        idx = 0
    }

    return ch.nodeMap[ch.hashRing[idx]]
}

func (ch *ConsistentHash) hash(key string) uint32 {
    return crc32.ChecksumIEEE([]byte(key))
}