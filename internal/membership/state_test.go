package membership

import "testing"

func TestNewStateStartsWithLocalAliveMember(t *testing.T) {
	state, err := NewState(Member{
		NodeID: "node-a",
		Addr:   "http://localhost:8080",
	})
	if err != nil {
		t.Fatalf("expected state creation to succeed, got %v", err)
	}

	snapshot := state.Snapshot()
	if len(snapshot.Members) != 1 {
		t.Fatalf("expected one member, got %d", len(snapshot.Members))
	}
	if snapshot.Members[0].Status != StatusAlive {
		t.Fatalf("expected local member to be alive, got %q", snapshot.Members[0].Status)
	}
}

func TestUpsertAddsAliveMemberToRing(t *testing.T) {
	state, err := NewState(Member{
		NodeID: "node-a",
		Addr:   "http://localhost:8080",
	})
	if err != nil {
		t.Fatalf("expected state creation to succeed, got %v", err)
	}

	if err := state.Upsert(Member{
		NodeID: "node-b",
		Addr:   "http://localhost:8081",
	}); err != nil {
		t.Fatalf("expected upsert to succeed, got %v", err)
	}

	snapshot := state.Snapshot()
	if len(snapshot.RingNodes) != 2 {
		t.Fatalf("expected 2 ring nodes, got %d", len(snapshot.RingNodes))
	}
}

func TestMarkLeftRemovesMemberFromRing(t *testing.T) {
	state, err := NewState(Member{
		NodeID: "node-a",
		Addr:   "http://localhost:8080",
	})
	if err != nil {
		t.Fatalf("expected state creation to succeed, got %v", err)
	}

	if err := state.Upsert(Member{
		NodeID: "node-b",
		Addr:   "http://localhost:8081",
	}); err != nil {
		t.Fatalf("expected upsert to succeed, got %v", err)
	}

	if err := state.MarkLeft("node-b"); err != nil {
		t.Fatalf("expected mark left to succeed, got %v", err)
	}

	member, ok := state.Get("node-b")
	if !ok {
		t.Fatal("expected node-b to still exist in membership list")
	}
	if member.Status != StatusLeft {
		t.Fatalf("expected node-b status to be left, got %q", member.Status)
	}

	snapshot := state.Snapshot()
	if len(snapshot.RingNodes) != 1 {
		t.Fatalf("expected only one alive ring node after leave, got %d", len(snapshot.RingNodes))
	}
}
