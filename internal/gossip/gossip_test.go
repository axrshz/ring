package gossip

import (
	"context"
	"net/http/httptest"
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
