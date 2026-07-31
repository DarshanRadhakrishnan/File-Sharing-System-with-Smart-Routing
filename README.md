# 🚀 P2P File Sharing System with Smart Routing

## Day 1: Peer Discovery & Overlay Network

A **production-grade P2P file sharing system** built with Go, gRPC, and graph algorithms.

### Features Implemented (Day 1)

- ✅ **Bootstrap Server** — Centralized peer registry on port 5000
- ✅ **Peer Discovery** — Automatic registration and discovery via bootstrap
- ✅ **Overlay Network** — In-memory graph of all connected peers
- ✅ **Latency Measurement** — Ping/pong RTT measurement between peers
- ✅ **gRPC Services** — Protocol Buffer-based RPC communication
- ✅ **Concurrent Connections** — Goroutine-based parallel peer connections

### Tech Stack

| Component | Technology |
|-----------|-----------|
| Language | Go 1.19+ |
| RPC Framework | gRPC + Protocol Buffers |
| Code Generation | Buf CLI |
| Concurrency | Goroutines & Channels |
| Storage | In-Memory (Maps) |
| Graph Algorithms | Dijkstra's Algorithm (Day 2) |

---

## 📁 Project Structure

```
File-Sharing-System-with-Smart-Routing/
├── go.mod                          # Go module dependencies
├── go.sum                          # Dependency checksums
├── buf.yaml                        # Buf configuration
├── buf.gen.yaml                    # Buf code generation config
├── cmd/
│   ├── bootstrap/
│   │   └── main.go                # Bootstrap server entry point
│   └── peer/
│       └── main.go                # Peer node entry point
├── internal/
│   ├── peer/
│   │   └── peer.go                # Peer node core logic & network graph
│   └── network/
│       └── grpc_server.go         # gRPC service implementation
├── proto/
│   ├── p2p.proto                  # Protocol buffer definitions
│   ├── p2p.pb.go                  # Generated: message types
│   └── p2p_grpc.pb.go            # Generated: gRPC service stubs
├── bin/                           # Compiled binaries
├── test_day1.ps1                  # PowerShell integration test
└── test_day1.sh                   # Bash integration test
```

---

## 🚀 Quick Start

### Prerequisites

```bash
go version  # Go 1.19+
```

### Build

```bash
# Build both binaries
go build -o bin/bootstrap.exe ./cmd/bootstrap
go build -o bin/peer.exe ./cmd/peer
```

### Run

```bash
# Terminal 1: Start bootstrap server
./bin/bootstrap.exe

# Terminal 2-6: Start peers
./bin/peer.exe --id=1 --port=9001
./bin/peer.exe --id=2 --port=9002
./bin/peer.exe --id=3 --port=9003
./bin/peer.exe --id=4 --port=9004
./bin/peer.exe --id=5 --port=9005
```

### Expected Output

**Bootstrap:**
```
[Bootstrap] Server listening on :5000
[Bootstrap] Registered peer 1 at localhost:9001
[Bootstrap] Registered peer 2 at localhost:9002
...
```

**Peer (example):**
```
Peer-4: gRPC server listening on :9004
Peer-4: Registered with bootstrap. Got 3 peers
Peer-4: Connected to peer-1 (latency: 9ms)
Peer-4: Connected to peer-2 (latency: 9ms)
Peer-4: Connected to peer-3 (latency: 7ms)

=== PEER-4 NETWORK GRAPH ===
Nodes (4):
  1: localhost:9001
  2: localhost:9002
  3: localhost:9003
  4: localhost
Edges (3):
  4 → 1 (latency: 9ms)
  4 → 2 (latency: 9ms)
  4 → 3 (latency: 7ms)
```

---

## 🔧 Architecture

```
┌─────────────────────────────────────┐
│      BOOTSTRAP SERVER (:5000)       │
│  ├─ Stores peer registry            │
│  └─ Returns random peers on join    │
└─────────────────────────────────────┘
          ↑         ↑         ↑
       Register  Register  Register
          ↓         ↓         ↓
┌──────────────────────────────────────┐
│     PEER OVERLAY NETWORK (Mesh)      │
│                                      │
│  Peer-1 ←→ Peer-2 ←→ Peer-3         │
│    ↑         ↓         ↑            │
│    └─→ Peer-4 ←→ Peer-5             │
│                                      │
│  Each peer maintains:                │
│  • gRPC server (listen)              │
│  • gRPC clients (to neighbors)       │
│  • In-memory network graph           │
│  • Latency measurements              │
└──────────────────────────────────────┘
```

---

## 📊 gRPC Services

### BootstrapService
| RPC | Request | Response | Description |
|-----|---------|----------|-------------|
| `RegisterPeer` | peer_id, address | success, peers[] | Register and get peer list |
| `GetPeers` | limit | peers[] | Get active peers |

### PeerService
| RPC | Request | Response | Description |
|-----|---------|----------|-------------|
| `Ping` | sender_id, timestamp | responder_id, timestamp | Latency measurement |
| `GetNetworkGraph` | requester_id | nodes[], edges[] | Get network topology |
| `NotifyNewPeer` | peer_id, address | acknowledged | New peer notification |

---

## 🔄 Regenerate Protobuf Code

```bash
# Install buf plugins (one-time)
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Generate Go code from proto definitions
go run github.com/bufbuild/buf/cmd/buf@latest generate
```

---

## 📈 Roadmap

- [x] **Day 1:** Peer Discovery & Overlay Network
- [ ] **Day 2:** Dijkstra Routing & Path Finding
- [ ] **Day 3:** Bloom Filters & Hash Verification
- [ ] **Day 4:** Concurrent Downloads & Backpressure
- [ ] **Day 5:** Docker Deployment & Benchmarking

---

## 📄 License

Educational project for learning P2P systems, distributed algorithms, and Go.