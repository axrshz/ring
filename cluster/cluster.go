// cluster/cluster.go
package cluster

import (
	"sync"
)

type Cluster struct {
    mu    sync.RWMutex
    nodes map[string]bool
    hash  *ConsistentHash
}

func NewCluster() *Cluster {
    return &Cluster{
        nodes: make(map[string]bool),
        hash:  NewConsistentHash(150), // 150 virtual nodes per physical node
    }
}

func (c *Cluster) AddNode(addr string) {
    c.mu.Lock()
    defer c.mu.Unlock()

    if !c.nodes[addr] {
        c.nodes[addr] = true
        c.hash.AddNode(addr)
    }
}

func (c *Cluster) RemoveNode(addr string) {
    c.mu.Lock()
    defer c.mu.Unlock()

    if c.nodes[addr] {
        delete(c.nodes, addr)
        c.hash.RemoveNode(addr)
    }
}

func (c *Cluster) GetNodeForKey(key string) string {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.hash.GetNode(key)
}

func (c *Cluster) GetAllNodes() []string {
    c.mu.RLock()
    defer c.mu.RUnlock()

    nodes := make([]string, 0, len(c.nodes))
    for node := range c.nodes {
        nodes = append(nodes, node)
    }
    return nodes
}