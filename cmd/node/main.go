package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"

	"ringg/internal/cache"
	"ringg/internal/cluster"
	"ringg/internal/membership"
	"ringg/internal/server"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	nodeID := flag.String("node-id", "node-a", "unique node identifier used on the hash ring")
	advertiseAddr := flag.String("advertise-addr", "http://localhost:8080", "address other nodes should use to reach this node")
	nodesFlag := flag.String("nodes", "", "comma-separated known nodes in the form node-a=http://localhost:8080,node-b=http://localhost:8081")
	joinAddr := flag.String("join", "", "optional existing node address to join, for example http://localhost:8080")
	flag.Parse()

	state, err := membership.NewState(membership.Member{
		NodeID: *nodeID,
		Addr:   *advertiseAddr,
	})
	if err != nil {
		log.Fatalf("invalid local member: %v", err)
	}

	nodes, err := cluster.ParseNodeList(*nodesFlag)
	if err != nil {
		log.Fatalf("invalid node list: %v", err)
	}
	for knownNodeID, knownAddr := range nodes {
		if knownNodeID == *nodeID {
			continue
		}
		if err := state.Upsert(membership.Member{
			NodeID: knownNodeID,
			Addr:   knownAddr,
		}); err != nil {
			log.Fatalf("failed to add known node %s: %v", knownNodeID, err)
		}
	}

	store := cache.NewStore()
	client := &http.Client{}
	handler := server.NewHandler(store, server.Options{
		Membership: state,
		Client:     client,
	})

	if strings.TrimSpace(*joinAddr) != "" {
		snapshot, err := cluster.JoinRemote(context.Background(), client, *joinAddr, membership.Member{
			NodeID: *nodeID,
			Addr:   *advertiseAddr,
		})
		if err != nil {
			log.Fatalf("failed to join cluster via %s: %v", *joinAddr, err)
		}
		if err := state.MergeSnapshot(snapshot); err != nil {
			log.Fatalf("failed to merge membership snapshot: %v", err)
		}
	}

	log.Printf(
		"phase 4 cache node listening on %s as %s with members %s",
		*addr,
		*nodeID,
		formatMembers(state.Snapshot().Members),
	)
	if err := http.ListenAndServe(*addr, handler); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}

func formatMembers(members []membership.Member) string {
	if len(members) == 0 {
		return "(empty)"
	}

	ordered := make([]string, 0, len(members))
	for _, member := range members {
		ordered = append(ordered, fmt.Sprintf("%s=%s(%s)", member.NodeID, member.Addr, member.Status))
	}
	sort.Strings(ordered)

	return strings.Join(ordered, ", ")
}
