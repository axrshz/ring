package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ringg/internal/cache"
	"ringg/internal/membership"
)

func TestHandlerCRUD(t *testing.T) {
	memberState, err := membership.NewState(membership.Member{
		NodeID: "node-a",
		Addr:   "http://localhost:8080",
	})
	if err != nil {
		t.Fatalf("expected single-node membership to be valid, got %v", err)
	}

	handler := NewHandler(cache.NewStore(), Options{
		Membership: memberState,
	})

	putRequest := httptest.NewRequest(http.MethodPut, "/cache/name", strings.NewReader("ringg"))
	putResponse := httptest.NewRecorder()
	handler.ServeHTTP(putResponse, putRequest)

	if putResponse.Code != http.StatusNoContent {
		t.Fatalf("expected PUT status %d, got %d", http.StatusNoContent, putResponse.Code)
	}

	getRequest := httptest.NewRequest(http.MethodGet, "/cache/name", nil)
	getResponse := httptest.NewRecorder()
	handler.ServeHTTP(getResponse, getRequest)

	if getResponse.Code != http.StatusOK {
		t.Fatalf("expected GET status %d, got %d", http.StatusOK, getResponse.Code)
	}

	body, err := io.ReadAll(getResponse.Body)
	if err != nil {
		t.Fatalf("failed to read GET response body: %v", err)
	}

	if !strings.Contains(string(body), "\"value\":\"ringg\"") {
		t.Fatalf("expected response body to contain stored value, got %s", string(body))
	}
	if !strings.Contains(string(body), "\"node_id\":\"node-a\"") {
		t.Fatalf("expected response body to contain serving node id, got %s", string(body))
	}

	deleteRequest := httptest.NewRequest(http.MethodDelete, "/cache/name", nil)
	deleteResponse := httptest.NewRecorder()
	handler.ServeHTTP(deleteResponse, deleteRequest)

	if deleteResponse.Code != http.StatusNoContent {
		t.Fatalf("expected DELETE status %d, got %d", http.StatusNoContent, deleteResponse.Code)
	}

	missingRequest := httptest.NewRequest(http.MethodGet, "/cache/name", nil)
	missingResponse := httptest.NewRecorder()
	handler.ServeHTTP(missingResponse, missingRequest)

	if missingResponse.Code != http.StatusNotFound {
		t.Fatalf("expected missing GET status %d, got %d", http.StatusNotFound, missingResponse.Code)
	}
}

func TestHandlerJoinAddsMemberAndReturnsSnapshot(t *testing.T) {
	memberState, err := membership.NewState(membership.Member{
		NodeID: "node-a",
		Addr:   "http://localhost:8080",
	})
	if err != nil {
		t.Fatalf("expected single-node membership to be valid, got %v", err)
	}

	server := NewHandler(cache.NewStore(), Options{
		Membership: memberState,
	})

	body, err := json.Marshal(membership.Member{
		NodeID: "node-b",
		Addr:   "http://localhost:8081",
	})
	if err != nil {
		t.Fatalf("expected join body to marshal, got %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/members/join", bytes.NewReader(body))
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected join status %d, got %d", http.StatusOK, response.Code)
	}

	member, ok := memberState.Get("node-b")
	if !ok {
		t.Fatal("expected node-b to be present after join")
	}
	if member.Status != membership.StatusAlive {
		t.Fatalf("expected node-b to be alive after join, got %q", member.Status)
	}

	var snapshot membership.Snapshot
	if err := json.NewDecoder(response.Body).Decode(&snapshot); err != nil {
		t.Fatalf("expected join response to decode, got %v", err)
	}

	if len(snapshot.Members) != 2 {
		t.Fatalf("expected snapshot to contain 2 members, got %d", len(snapshot.Members))
	}
}

func TestHandlerForwardsRequestsToAliveOwnerNode(t *testing.T) {
	ownerStore := cache.NewStore()

	ownerState, err := membership.NewState(membership.Member{
		NodeID: "node-b",
		Addr:   "http://owner.invalid",
	})
	if err != nil {
		t.Fatalf("expected owner membership to be valid, got %v", err)
	}
	if err := ownerState.Upsert(membership.Member{
		NodeID: "node-a",
		Addr:   "http://node-a.invalid",
	}); err != nil {
		t.Fatalf("expected owner membership upsert to succeed, got %v", err)
	}

	ownerServer := httptest.NewServer(NewHandler(ownerStore, Options{
		Membership: ownerState,
	}))
	defer ownerServer.Close()

	routerState, err := membership.NewState(membership.Member{
		NodeID: "node-a",
		Addr:   "http://router.invalid",
	})
	if err != nil {
		t.Fatalf("expected router membership to be valid, got %v", err)
	}
	if err := routerState.Upsert(membership.Member{
		NodeID: "node-b",
		Addr:   ownerServer.URL,
	}); err != nil {
		t.Fatalf("expected router membership upsert to succeed, got %v", err)
	}

	routerServer := httptest.NewServer(NewHandler(cache.NewStore(), Options{
		Membership: routerState,
		Client:     ownerServer.Client(),
	}))
	defer routerServer.Close()

	key := findKeyOwnedBy(t, routerState, "node-b")

	putRequest, err := http.NewRequest(http.MethodPut, routerServer.URL+"/cache/"+key, strings.NewReader("forwarded"))
	if err != nil {
		t.Fatalf("expected PUT request creation to succeed, got %v", err)
	}

	putResponse, err := ownerServer.Client().Do(putRequest)
	if err != nil {
		t.Fatalf("expected forwarded PUT request to succeed, got %v", err)
	}
	defer putResponse.Body.Close()

	if putResponse.StatusCode != http.StatusNoContent {
		t.Fatalf("expected forwarded PUT status %d, got %d", http.StatusNoContent, putResponse.StatusCode)
	}

	value, err := ownerStore.Get(key)
	if err != nil {
		t.Fatalf("expected owner store to contain forwarded key, got %v", err)
	}
	if value != "forwarded" {
		t.Fatalf("expected forwarded value to be stored on owner, got %q", value)
	}

	getResponse, err := http.Get(routerServer.URL + "/cache/" + key)
	if err != nil {
		t.Fatalf("expected forwarded GET request to succeed, got %v", err)
	}
	defer getResponse.Body.Close()

	if getResponse.StatusCode != http.StatusOK {
		t.Fatalf("expected forwarded GET status %d, got %d", http.StatusOK, getResponse.StatusCode)
	}

	body, err := io.ReadAll(getResponse.Body)
	if err != nil {
		t.Fatalf("expected forwarded GET body to be readable, got %v", err)
	}

	if !strings.Contains(string(body), "\"node_id\":\"node-b\"") {
		t.Fatalf("expected GET response to come from node-b, got %s", string(body))
	}
	if !strings.Contains(string(body), "\"value\":\"forwarded\"") {
		t.Fatalf("expected GET response to contain forwarded value, got %s", string(body))
	}
}

func findKeyOwnedBy(t *testing.T, state *membership.State, ownerNodeID string) string {
	t.Helper()

	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key-%d", i)
		owner, err := state.GetOwner(key)
		if err != nil {
			t.Fatalf("expected ring lookup to succeed, got %v", err)
		}
		if owner.NodeID == ownerNodeID {
			return key
		}
	}

	t.Fatalf("expected to find a key owned by %s", ownerNodeID)
	return ""
}
