# ringg

`ringg` is a beginner-friendly distributed cache project in Go.

## Phase 1

Phase 1 is a single-node cache with:

- an in-memory key/value store
- HTTP endpoints for `GET`, `PUT`, and `DELETE`
- tests for the store and handler

## Phase 2

Phase 2 adds a minimal consistent hash ring:

- add and remove nodes by ID
- look up which node owns a key
- tests that show only some keys move when a node joins

This phase is intentionally standalone. The hash ring is not wired into HTTP yet, because that belongs in the next step when we start routing requests between nodes.

## Phase 3

Phase 3 wires the hash ring into the HTTP server:

- every node gets a `node-id`
- every node starts with the same static cluster map
- cache requests are forwarded to the owner node when needed
- `GET /cluster` shows the local node and ring view for debugging

## Phase 4

Phase 4 replaces the fixed cluster config with tracked membership:

- nodes keep a membership table with `alive` and `left` status
- the ring is rebuilt from the currently alive members
- `POST /members/join` lets a node announce itself to an existing node
- `GET /members` shows the current membership snapshot

This phase still does not spread updates automatically. If node A learns about node C, node B will not learn that until we add gossip in the next phase.

## Phase 5

Phase 5 adds a tiny gossip loop:

- nodes periodically send their membership snapshot to a random alive peer
- peers merge the received snapshot and reply with their own snapshot
- membership changes now spread automatically without manual joins on every node

This is still a very small version of gossip. It spreads information, but it does not yet do richer failure detection like `suspect` and `dead`.

## Run

```bash
go run ./cmd/node -addr :8080 -node-id node-a -advertise-addr http://localhost:8080
```

## Try It

```bash
curl -X PUT http://localhost:8080/cache/name --data "ringg"
curl http://localhost:8080/cache/name
curl -X DELETE http://localhost:8080/cache/name
```

## Try A 2-Node Cluster

Start node A:

```bash
go run ./cmd/node -addr :8080 -node-id node-a -advertise-addr http://localhost:8080
```

Start node B in another terminal:

```bash
go run ./cmd/node -addr :8081 -node-id node-b -advertise-addr http://localhost:8081 -join http://localhost:8080
```

Then send requests to either node:

```bash
curl -X PUT http://localhost:8080/cache/name --data "ringg"
curl http://localhost:8081/cache/name
curl http://localhost:8080/members
curl http://localhost:8081/cluster
```

If the ring says `name` belongs on the other node, the request will be proxied there automatically.

To slow gossip down for debugging, you can override the interval:

```bash
go run ./cmd/node -addr :8081 -node-id node-b -advertise-addr http://localhost:8081 -join http://localhost:8080 -gossip-interval 2s
```

## Planned Next Phases

1. Rebalance keys when nodes join or leave
