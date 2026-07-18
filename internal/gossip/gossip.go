package gossip

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"ringg/internal/membership"
)

type Config struct {
	LocalNodeID string
	Membership  *membership.State
	Client      *http.Client
	Interval    time.Duration
	MaxPeers    int
}

type Engine struct {
	localNodeID string
	membership  *membership.State
	client      *http.Client
	interval    time.Duration
	maxPeers    int
	rngMu       sync.Mutex
	rng         *rand.Rand
}

func New(config Config) *Engine {
	membershipState := config.Membership
	localNodeID := strings.TrimSpace(config.LocalNodeID)
	if localNodeID == "" && membershipState != nil {
		localNodeID = membershipState.LocalNodeID()
	}

	client := config.Client
	if client == nil {
		client = &http.Client{Timeout: 2 * time.Second}
	}

	interval := config.Interval
	if interval <= 0 {
		interval = time.Second
	}

	maxPeers := config.MaxPeers
	if maxPeers <= 0 {
		maxPeers = 1
	}

	return &Engine{
		localNodeID: localNodeID,
		membership:  membershipState,
		client:      client,
		interval:    interval,
		maxPeers:    maxPeers,
		rng:         rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (e *Engine) Run(ctx context.Context) {
	ticker := time.NewTicker(e.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = e.SpreadOnce(ctx)
		}
	}
}

func (e *Engine) SpreadOnce(ctx context.Context) error {
	if e.membership == nil {
		return errors.New("membership is required")
	}

	snapshot := e.membership.Snapshot()
	peers := alivePeers(snapshot, e.localNodeID)
	if len(peers) == 0 {
		return nil
	}

	e.rngMu.Lock()
	e.rng.Shuffle(len(peers), func(i, j int) {
		peers[i], peers[j] = peers[j], peers[i]
	})
	e.rngMu.Unlock()

	limit := e.maxPeers
	if limit > len(peers) {
		limit = len(peers)
	}

	var firstErr error
	for _, peer := range peers[:limit] {
		peerSnapshot, err := e.sendSnapshot(ctx, peer.Addr, snapshot)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}

		if err := e.membership.MergeSnapshot(peerSnapshot); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

func (e *Engine) sendSnapshot(ctx context.Context, targetAddr string, snapshot membership.Snapshot) (membership.Snapshot, error) {
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return membership.Snapshot{}, fmt.Errorf("marshal gossip snapshot: %w", err)
	}

	targetURL := strings.TrimRight(targetAddr, "/") + "/members/gossip"
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(payload))
	if err != nil {
		return membership.Snapshot{}, fmt.Errorf("create gossip request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := e.client.Do(request)
	if err != nil {
		return membership.Snapshot{}, fmt.Errorf("send gossip request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return membership.Snapshot{}, fmt.Errorf("gossip request failed with status %s", response.Status)
	}

	var peerSnapshot membership.Snapshot
	decoder := json.NewDecoder(io.LimitReader(response.Body, 1<<20))
	if err := decoder.Decode(&peerSnapshot); err != nil {
		return membership.Snapshot{}, fmt.Errorf("decode gossip response: %w", err)
	}

	return peerSnapshot, nil
}

func alivePeers(snapshot membership.Snapshot, localNodeID string) []membership.Member {
	peers := make([]membership.Member, 0, len(snapshot.Members))
	for _, member := range snapshot.Members {
		if member.NodeID == localNodeID {
			continue
		}
		if member.Status != membership.StatusAlive {
			continue
		}
		peers = append(peers, member)
	}
	return peers
}
