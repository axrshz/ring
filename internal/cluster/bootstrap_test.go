package cluster

import "testing"

func TestParseNodeList(t *testing.T) {
	nodes, err := ParseNodeList("node-a=http://localhost:8080,node-b=http://localhost:8081")
	if err != nil {
		t.Fatalf("expected parsing to succeed, got %v", err)
	}

	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}

	if nodes["node-a"] != "http://localhost:8080" {
		t.Fatalf("expected node-a address to be parsed, got %q", nodes["node-a"])
	}
}
