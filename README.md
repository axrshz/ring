# ring

`ring` is a distributed cache built in Go for learning purposes.

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
