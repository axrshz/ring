package ring

import (
	"errors"
	"fmt"
	"testing"
)

func TestGetOwnerOnEmptyRing(t *testing.T) {
	ring := New()

	_, err := ring.GetOwner("language")
	if !errors.Is(err, ErrEmptyRing) {
		t.Fatalf("expected ErrEmptyRing, got %v", err)
	}
}

func TestGetOwnerReturnsOneOfTheAddedNodes(t *testing.T) {
	ring := New()
	ring.AddNode("node-a")
	ring.AddNode("node-b")

	owner, err := ring.GetOwner("language")
	if err != nil {
		t.Fatalf("expected owner lookup to succeed, got %v", err)
	}

	if owner != "node-a" && owner != "node-b" {
		t.Fatalf("expected owner to be one of the ring nodes, got %q", owner)
	}
}

func TestRemovingNodeChangesOwnershipWhenNeeded(t *testing.T) {
	ring := New()
	ring.AddNode("node-a")
	ring.AddNode("node-b")

	keys := []string{"alpha", "beta", "gamma", "delta", "epsilon", "zeta"}

	var keyOwnedByNodeA string
	for _, key := range keys {
		owner, err := ring.GetOwner(key)
		if err != nil {
			t.Fatalf("expected owner lookup to succeed, got %v", err)
		}
		if owner == "node-a" {
			keyOwnedByNodeA = key
			break
		}
	}

	if keyOwnedByNodeA == "" {
		t.Fatal("expected at least one sample key to be owned by node-a")
	}

	ring.RemoveNode("node-a")

	owner, err := ring.GetOwner(keyOwnedByNodeA)
	if err != nil {
		t.Fatalf("expected owner lookup after removal to succeed, got %v", err)
	}

	if owner != "node-b" {
		t.Fatalf("expected key %q to move to node-b after removing node-a, got %q", keyOwnedByNodeA, owner)
	}
}

func TestAddingNodeOnlyMovesSomeKeys(t *testing.T) {
	before := New()
	before.AddNode("node-a")
	before.AddNode("node-b")

	after := New()
	after.AddNode("node-a")
	after.AddNode("node-b")
	after.AddNode("node-c")

	totalKeys := 500
	movedKeys := 0

	for i := 0; i < totalKeys; i++ {
		key := fmt.Sprintf("key-%d", i)

		beforeOwner, err := before.GetOwner(key)
		if err != nil {
			t.Fatalf("expected owner lookup before adding node to succeed, got %v", err)
		}

		afterOwner, err := after.GetOwner(key)
		if err != nil {
			t.Fatalf("expected owner lookup after adding node to succeed, got %v", err)
		}

		if beforeOwner != afterOwner {
			movedKeys++
		}
	}

	if movedKeys == 0 {
		t.Fatal("expected some keys to move after adding a node")
	}

	if movedKeys == totalKeys {
		t.Fatal("expected some keys to stay on the same node after adding a node")
	}
}
