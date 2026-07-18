package gossip

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"ringg/internal/cache"
	"ringg/internal/membership"
	"ringg/internal/server"
)

func TestSpreadOnceSharesMembershipAndMergesResponse(t *testing.T) {
	peerState, err := membership.NewState(membership.Member{
		NodeID: "node-b",
		Addr:   "http://node-b.invalid",
	})
	if err != nil {
		t.Fatalf("expected peer membership creation to succeed, got %v", err)
	}
	if err := peerState.Upsert(membership.Member{
		NodeID: "node-c",
		Addr:   "http://localhost:8082",
	}); err != nil {
		t.Fatalf("expected peer membership upsert to succeed, got %v", err)
	}

	peerServer := httptest.NewServer(server.NewHandler(cache.NewStore(), server.Options{
		Membership: peerState,
	}))
	defer peerServer.Close()

	localState, err := membership.NewState(membership.Member{
		NodeID: "node-a",
		Addr:   "http://localhost:8080",
	})
	if err != nil {
		t.Fatalf("expected local membership creation to succeed, got %v", err)
	}
	if err := localState.Upsert(membership.Member{
		NodeID: "node-b",
		Addr:   peerServer.URL,
	}); err != nil {
		t.Fatalf("expected local membership upsert to succeed, got %v", err)
	}

	engine := New(Config{
		LocalNodeID: "node-a",
		Membership:  localState,
		Client:      peerServer.Client(),
	})

	if err := engine.SpreadOnce(context.Background()); err != nil {
		t.Fatalf("expected gossip spread to succeed, got %v", err)
	}

	if _, ok := peerState.Get("node-a"); !ok {
		t.Fatal("expected peer state to learn about node-a from gossip")
	}

	if _, ok := localState.Get("node-c"); !ok {
		t.Fatal("expected local state to learn about node-c from peer gossip response")
	}
}

func TestNewInfersLocalNodeIDFromMembership(t *testing.T) {
	var requests atomic.Int32
	peerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer peerServer.Close()

	state, err := membership.NewState(membership.Member{
		NodeID: "node-a",
		Addr:   peerServer.URL,
	})
	if err != nil {
		t.Fatalf("expected membership creation to succeed, got %v", err)
	}

	engine := New(Config{
		Membership: state,
		Client:     peerServer.Client(),
	})

	if err := engine.SpreadOnce(context.Background()); err != nil {
		t.Fatalf("expected single-node gossip to be a no-op, got %v", err)
	}
	if requests.Load() != 0 {
		t.Fatalf("expected gossip to skip the local member, got %d requests", requests.Load())
	}
}
