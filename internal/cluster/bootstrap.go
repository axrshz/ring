package cluster

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"ringg/internal/membership"
)

type joinRequest struct {
	NodeID string `json:"node_id"`
	Addr   string `json:"addr"`
}

func ParseNodeList(raw string) (map[string]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]string{}, nil
	}

	nodes := make(map[string]string)
	entries := strings.Split(raw, ",")

	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid node entry %q: expected node-id=http://host:port", entry)
		}

		nodeID := strings.TrimSpace(parts[0])
		addr := strings.TrimSpace(parts[1])
		if nodeID == "" || addr == "" {
			return nil, fmt.Errorf("invalid node entry %q: node id and address are required", entry)
		}

		nodes[nodeID] = addr
	}

	return nodes, nil
}

func JoinRemote(ctx context.Context, client *http.Client, targetAddr string, local membership.Member) (membership.Snapshot, error) {
	if client == nil {
		client = &http.Client{Timeout: 2 * time.Second}
	}

	payload, err := json.Marshal(joinRequest{
		NodeID: local.NodeID,
		Addr:   local.Addr,
	})
	if err != nil {
		return membership.Snapshot{}, fmt.Errorf("marshal join request: %w", err)
	}

	targetURL := strings.TrimRight(targetAddr, "/") + "/members/join"
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(payload))
	if err != nil {
		return membership.Snapshot{}, fmt.Errorf("create join request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := client.Do(request)
	if err != nil {
		return membership.Snapshot{}, fmt.Errorf("send join request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return membership.Snapshot{}, fmt.Errorf("join request failed with status %s", response.Status)
	}

	var snapshot membership.Snapshot
	if err := json.NewDecoder(response.Body).Decode(&snapshot); err != nil {
		return membership.Snapshot{}, fmt.Errorf("decode join response: %w", err)
	}

	return snapshot, nil
}
