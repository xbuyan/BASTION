-- seed_exercises.sql — All 75 exercises
-- Run after migrations: psql $DATABASE_URL -f seed_exercises.sql

-- ─── PHASE I: Foundation ─────────────────────────────────────────────────────
INSERT INTO exercises (phase,num,title,domain,difficulty,tags,description,starter_code,time_complexity,space_complexity) VALUES
(1,'I.1','Theorem Prover','math',2,'["logic","propositional","proof-trees"]',
'## Set Theory & Logic

Implement a theorem prover for propositional logic. Your prover should support resolution-based reasoning, build proof trees, and determine satisfiability of logical formulas.

### Learning Objectives
- Understand propositional logic semantics
- Implement CNF conversion
- Understand resolution as a proof method

### Theory
A formula is satisfiable if there is an assignment of truth values to variables that makes it true. The Davis-Putnam algorithm (and its DPLL extension) form the basis of modern SAT solvers.

**CNF Conversion:** Every propositional formula can be converted to Conjunctive Normal Form (a conjunction of disjunctions).

### Requirements
1. Parse propositional formulas: `(A AND B) OR (NOT C)`
2. Convert to Conjunctive Normal Form (CNF)
3. Implement the resolution algorithm
4. Build and display proof trees
5. Determine if a formula is satisfiable (SAT) or unsatisfiable (UNSAT)

### Edge Cases
- Tautologies: formulas that are always true
- Contradictions: formulas that are always false
- Formulas with 10+ variables (performance test)',
'package main

import "fmt"

type Formula interface {
    Evaluate(assignment map[string]bool) bool
    String() string
}

// TODO: Implement Atom, And, Or, Not, Implies types
// TODO: Implement ParseFormula(s string) Formula
// TODO: Implement ToCNF(f Formula) [][]string (list of clauses)
// TODO: Implement Resolve(clauses [][]string) bool

func main() {
    // Test: (A OR NOT A) should be TAUTOLOGY
    // Test: (A AND NOT A) should be UNSAT
    fmt.Println("Implement the theorem prover")
}',
'O(2^n) worst case, O(n*c) with good heuristics','O(n)'),

(1,'I.2','Number Theory Suite','math',1,'["euclidean","CRT","number-theory","primality"]',
'## Number Theory

Implement core number theory algorithms essential for cryptography. These are the mathematical building blocks of RSA, elliptic curve cryptography, and zero-knowledge proofs.

### Algorithms to Implement

**1. Extended Euclidean Algorithm**
Returns gcd(a,b) and coefficients x,y such that ax + by = gcd(a,b). Used for modular inverse computation.

**2. Miller-Rabin Primality Test**
Probabilistic primality test used in RSA key generation. Probability of false positive < 4^(-k) for k rounds.

**3. Chinese Remainder Theorem**
Given x ≡ a₁ (mod n₁), x ≡ a₂ (mod n₂), find x. Essential for efficient RSA implementations.

**4. Modular Exponentiation**
Compute a^b mod n efficiently using square-and-multiply. The basis of RSA encryption/decryption.',
'package main

import "fmt"

// GCD returns the greatest common divisor of a and b
func GCD(a, b int64) int64 {
    // TODO: Implement Euclidean algorithm
    return 0
}

// ExtendedGCD returns (gcd, x, y) where a*x + b*y = gcd
func ExtendedGCD(a, b int64) (int64, int64, int64) {
    // TODO
    return 0, 0, 0
}

// ModInverse returns a^(-1) mod m, or error if it does not exist
func ModInverse(a, m int64) (int64, error) {
    // TODO: Use ExtendedGCD
    return 0, nil
}

// ModExp computes base^exp mod m efficiently
func ModExp(base, exp, m int64) int64 {
    // TODO: Square-and-multiply
    return 0
}

// MillerRabin tests primality with k rounds
func MillerRabin(n int64, k int) bool {
    // TODO
    return false
}

// CRT solves the system x ≡ a[i] (mod n[i])
func CRT(remainders, moduli []int64) (int64, error) {
    // TODO
    return 0, nil
}

func main() {
    fmt.Println(GCD(48, 18)) // 6
    fmt.Println(ModExp(2, 10, 1000)) // 24
    fmt.Println(MillerRabin(17, 5))  // true
}',
'O(log n) for GCD, O(log^2 n) for ModExp','O(log n)'),

(1,'I.3','Entropy Calculator','math',1,'["combinatorics","entropy","probability","shannon"]',
'## Information Theory

Implement entropy calculations and combinatorial generators essential for understanding cryptographic strength.

### Theory

**Shannon Entropy:** H(X) = -Σ p(x) log₂ p(x)

Measures the average information content (surprise) of a random variable. Higher entropy = more unpredictable = stronger cryptographic keys.

**Birthday Problem:** The probability that 2 values collide in a set of n values when choosing k samples ≈ 1 - e^(-k²/2n). This is why 128-bit security requires 2^64 operations to find a collision.

### Requirements
1. Shannon entropy for a frequency distribution
2. Min-entropy: H∞(X) = -log₂(max p(x))
3. Birthday attack probability calculator
4. Password entropy estimator (character set × length)
5. Combination and permutation generators',
'package main

import (
    "fmt"
    "math"
)

// ShannonEntropy returns H(X) given a probability distribution
func ShannonEntropy(probs []float64) float64 {
    // TODO: H(X) = -Σ p log₂(p)
    return 0
}

// MinEntropy returns H∞(X) = -log₂(max_prob)
func MinEntropy(probs []float64) float64 {
    // TODO
    return 0
}

// BirthdayProbability estimates collision probability
// for k samples from a space of n values
func BirthdayProbability(n, k float64) float64 {
    // TODO: 1 - e^(-k^2 / 2n)
    return 0
}

// PasswordEntropy estimates bits of entropy
func PasswordEntropy(charsetSize, length int) float64 {
    return math.Log2(math.Pow(float64(charsetSize), float64(length)))
}

func main() {
    // Uniform distribution over 8 outcomes = 3 bits
    probs := []float64{0.125, 0.125, 0.125, 0.125, 0.125, 0.125, 0.125, 0.125}
    fmt.Printf("Entropy: %.2f bits\n", ShannonEntropy(probs)) // 3.0
}',
'O(n)','O(1)'),

(1,'I.11','Merkle Tree','data-structures',2,'["merkle","inclusion-proofs","blockchain","hashing"]',
'## Merkle Tree with Inclusion Proofs

Implement a Merkle tree — the data structure at the heart of Bitcoin, Ethereum, and virtually every blockchain. Used to create tamper-evident summaries of large datasets.

### Theory

A Merkle tree is a binary hash tree where:
- Leaf nodes contain H(data)
- Internal nodes contain H(left_child || right_child)
- The root summarizes ALL data with a single hash

**Inclusion Proof:** To prove element e is in a tree of n elements, you need only O(log n) hashes (the "sibling path" from leaf to root), not all n elements.

### Requirements
1. Build tree from [][]byte data
2. Get root hash
3. Generate inclusion proof for index i
4. Verify inclusion proof (without the full tree)
5. Batch verification (multiple proofs at once)
6. Handle odd numbers of leaves (duplicate last leaf)

### Applications
- Bitcoin: Transaction Merkle tree in each block header
- Ethereum: State trie, receipt trie, transaction trie
- Certificate Transparency: Merkle tree of TLS certificates',
'package main

import (
    "crypto/sha256"
    "fmt"
)

type MerkleTree struct {
    Leaves [][]byte
    Layers [][][]byte
    Root   []byte
}

type InclusionProof struct {
    Index  int
    Path   [][]byte  // sibling hashes
    Sides  []bool    // true = sibling is on right
}

func Hash(data []byte) []byte {
    h := sha256.Sum256(data)
    return h[:]
}

// Build constructs a Merkle tree from data
func Build(data [][]byte) *MerkleTree {
    // TODO
    return nil
}

// GenerateProof returns an inclusion proof for index i
func (t *MerkleTree) GenerateProof(index int) *InclusionProof {
    // TODO
    return nil
}

// VerifyProof verifies an inclusion proof against root
func VerifyProof(root []byte, data []byte, proof *InclusionProof) bool {
    // TODO: Reconstruct root from proof path
    return false
}

func main() {
    data := [][]byte{[]byte("alice"), []byte("bob"), []byte("carol"), []byte("dave")}
    tree := Build(data)
    fmt.Printf("Root: %x\n", tree.Root)

    proof := tree.GenerateProof(1) // Prove "bob" is in tree
    fmt.Println(VerifyProof(tree.Root, []byte("bob"), proof)) // true
    fmt.Println(VerifyProof(tree.Root, []byte("eve"), proof))  // false
}',
'O(n) build, O(log n) proof','O(n)');

-- Phase II exercises (abbreviated — full set loaded in Go seed)
INSERT INTO exercises (phase,num,title,domain,difficulty,tags,description,starter_code,time_complexity,space_complexity) VALUES
(2,'II.1','Finite Field Arithmetic','crypto',3,'["GF","finite-field","algebra","cryptography"]',
'## Finite Field Arithmetic GF(2⁸) and GF(p)

Implement arithmetic in finite fields — the mathematical foundation of AES, elliptic curves, and all modern cryptography.

### Theory

A **finite field** GF(q) has exactly q elements and supports +, -, ×, ÷ with the usual algebraic properties. For cryptography we use:
- **GF(p)**: integers mod prime p. Used in RSA, DH, elliptic curves
- **GF(2⁸)**: polynomials over GF(2) mod irreducible poly. Used in AES S-box

**GF(2⁸) multiplication** uses the irreducible polynomial x⁸ + x⁴ + x³ + x + 1 (0x11b in AES).

### Requirements
1. GF(p): +, -, ×, ÷ (mod p), pow, inverse
2. GF(2⁸): +, ×, inverse using xtime algorithm
3. Find primitive roots of GF(p)
4. Verify Fermat''s little theorem: a^(p-1) ≡ 1 (mod p)',
'package main

import "fmt"

// GFp implements arithmetic in GF(p)
type GFp struct{ p int64 }

func (g GFp) Add(a, b int64) int64      { return (a + b) % g.p }
func (g GFp) Sub(a, b int64) int64      { return ((a - b) % g.p + g.p) % g.p }
func (g GFp) Mul(a, b int64) int64      { /* TODO */ return 0 }
func (g GFp) Inv(a int64) int64         { /* TODO: Fermat or ExtGCD */ return 0 }
func (g GFp) Div(a, b int64) int64      { return g.Mul(a, g.Inv(b)) }

// GF256 implements arithmetic in GF(2^8) with AES polynomial
type GF256 struct{}

const aespoly = 0x11b

func (g GF256) Add(a, b byte) byte    { return a ^ b } // XOR
func (g GF256) Xtime(a byte) byte     { /* TODO: shift + conditional XOR */ return 0 }
func (g GF256) Mul(a, b byte) byte    { /* TODO: Russian peasant algorithm */ return 0 }
func (g GF256) Inv(a byte) byte       { /* TODO: Extended Euclidean in GF(2^8) */ return 0 }

func main() {
    gf := GFp{p: 17}
    fmt.Println(gf.Add(10, 9))  // 2 (10+9=19, 19 mod 17 = 2)
    fmt.Println(gf.Mul(3, 6))   // 1 (18 mod 17 = 1)
    fmt.Println(gf.Inv(3))      // 6 (3*6 = 18 ≡ 1 mod 17)
}',
'O(log p) for inverse','O(1)'),

(2,'II.6','Raft Consensus','consensus',3,'["raft","distributed","leader-election","log-replication"]',
'## Raft Consensus Algorithm

Implement the complete Raft consensus protocol — the algorithm that powers etcd (Kubernetes'' brain), CockroachDB, and TiKV.

### Theory

Raft decomposes consensus into three sub-problems:
1. **Leader Election**: One server is elected leader per term
2. **Log Replication**: Leader accepts log entries, replicates to followers
3. **Safety**: If two logs have same index/term, they are identical

**Key Properties:**
- Election Safety: at most one leader per term
- Log Matching: if logs agree at index i, they agree at all j < i
- Leader Completeness: committed entries appear in all future leaders

### State Machine
```
Follower → (timeout, no heartbeat) → Candidate → (majority votes) → Leader
Leader → (higher term seen) → Follower
Candidate → (higher term seen) → Follower
```

### Requirements
1. Leader election with randomized timeouts (150-300ms)
2. Log replication with AppendEntries RPC
3. RequestVote RPC with log completeness check
4. Log commitment when majority acknowledges
5. Log compaction and snapshotting
6. Network partition recovery',
'package main

// Implement full Raft in this file
// Focus on correctness over performance

type State int
const (
    Follower State = iota
    Candidate
    Leader
)

type LogEntry struct {
    Term    int
    Index   int
    Command interface{}
}

type RaftNode struct {
    id          int
    state       State
    currentTerm int
    votedFor    int
    log         []LogEntry
    commitIndex int
    lastApplied int
    // Leader state
    nextIndex   map[int]int
    matchIndex  map[int]int
    // TODO: Add timer fields, peer connections
}

// TODO: Implement StartElection, RequestVote, AppendEntries
// TODO: Implement heartbeat loop, election timeout
// TODO: Implement log compaction

func main() {
    // Create a 3-node cluster and demonstrate leader election
}',
'O(log n) per operation in steady state','O(n) for log');

-- ─── Concept Nodes ─────────────────────────────────────────────────────────
INSERT INTO concepts (id,label,domain,phase,description) VALUES
('set-theory','Set Theory','math',1,'Foundations of mathematical reasoning: sets, relations, functions'),
('number-theory','Number Theory','math',1,'Divisibility, primes, modular arithmetic — foundation of cryptography'),
('combinatorics','Combinatorics','math',1,'Counting principles, permutations, combinations, probability'),
('information-theory','Information Theory','math',1,'Shannon entropy, channel capacity, data compression'),
('complexity','Computational Complexity','math',1,'P vs NP, reductions, hardness assumptions'),
('abstract-algebra','Abstract Algebra','math',2,'Groups, rings, fields — the structure of cryptographic objects'),
('finite-fields','Finite Fields','math',2,'GF(p) and GF(2^n): arithmetic over finite sets'),
('game-theory','Game Theory','math',3,'Nash equilibria, incentive compatibility, mechanism design'),
('markov-chains','Markov Chains','math',3,'Stochastic processes, steady states, blockchain finality'),
('formal-methods','Formal Methods','math',2,'TLA+, Alloy, Coq: rigorous system specification'),
('memory-mgmt','Memory Management','systems',1,'Heap, stack, garbage collection, allocators'),
('filesystems','Filesystems','systems',1,'POSIX, VFS, FUSE, content-addressed storage'),
('concurrency','Concurrency','systems',1,'Goroutines, channels, locks, lock-free data structures'),
('networking','Networking','systems',1,'TCP, UDP, sockets, network programming primitives'),
('profiling','Profiling & Performance','systems',1,'pprof, flame graphs, benchmarking, optimization'),
('wasm-vm','WASM Runtime','systems',4,'WebAssembly: bytecode, execution model, sandboxing'),
('ebpf','eBPF/XDP','systems',4,'Linux extended BPF: kernel programmability, observability'),
('ecc','Elliptic Curves','crypto',2,'Weierstrass form, point arithmetic, discrete log'),
('ecdsa','ECDSA','crypto',2,'Elliptic curve digital signatures, nonce security'),
('aes','AES','crypto',2,'Advanced Encryption Standard: SPN cipher, key schedule'),
('sha256','SHA-256','crypto',2,'Merkle-Damgård hash, compression function, Merkle trees'),
('schnorr','Schnorr Protocol','crypto',3,'Interactive/non-interactive proofs of knowledge'),
('bls-sigs','BLS Signatures','crypto',3,'Pairing-based signatures, aggregation, threshold schemes'),
('threshold-crypto','Threshold Cryptography','crypto',3,'Shamir secret sharing, distributed key generation'),
('pq-crypto','Post-Quantum Crypto','crypto',4,'Lattice-based cryptography, NIST PQC standards'),
('fhe','Homomorphic Encryption','crypto',4,'BFV/BGV: computation on encrypted data'),
('tee','TEE/SGX','crypto',4,'Trusted execution environments, remote attestation'),
('raft','Raft Consensus','consensus',2,'Understandable consensus: leader election, log replication'),
('pbft','PBFT','consensus',2,'Byzantine fault tolerance, 3f+1 nodes, view change'),
('nakamoto','Nakamoto Consensus','consensus',2,'Proof of work, longest chain, selfish mining'),
('pos','Proof of Stake','consensus',2,'Validator selection, slashing, finality gadgets'),
('dag-consensus','DAG Consensus','consensus',2,'Hashgraph, IOTA, DAG-based ordering'),
('mev','MEV','consensus',3,'Maximal extractable value, front-running, PBS'),
('merkle','Merkle Tree','systems',1,'Binary hash tree, inclusion proofs, authenticated data'),
('bloom','Bloom Filter','systems',1,'Probabilistic membership, false positive rate'),
('lsm-tree','LSM Tree','systems',1,'Log-structured merge tree, RocksDB, LevelDB'),
('bulletproofs','Bulletproofs','zkp',3,'Range proofs, inner product argument, no trusted setup'),
('snarks','ZK-SNARKs','zkp',3,'Succinct non-interactive arguments of knowledge'),
('starks','ZK-STARKs','zkp',3,'Scalable transparent arguments, FRI, post-quantum'),
('pairings','Elliptic Pairings','zkp',4,'Tate/Weil pairing, BLS12-381, SNARK verification'),
('docker','Docker','devops',3,'Containerization, Dockerfile, image layers, registries'),
('kubernetes','Kubernetes','devops',3,'Container orchestration, pods, services, CRDs, operators'),
('prometheus','Prometheus','devops',3,'Pull-based metrics, PromQL, alertmanager, Grafana'),
('otel','OpenTelemetry','devops',3,'Distributed tracing, spans, context propagation'),
('gitops','GitOps','devops',3,'ArgoCD, declarative deployment, reconciliation loops'),
('service-mesh','Service Mesh','devops',3,'Istio/Linkerd, mTLS, traffic management, observability'),
('libp2p','libp2p','networking',2,'Modular P2P networking, DHT, pubsub, muxing'),
('grpc','gRPC','networking',2,'Protocol Buffers, HTTP/2, streaming, service definition'),
('quic','QUIC','networking',2,'0-RTT, multiplexed streams, connection migration'),
('shamir-ss','Shamir Secret Sharing','crypto',3,'(k,n)-threshold scheme over GF(2^8)'),
('dkg','Distributed Key Gen','crypto',3,'Pedersen DKG, VSS, joint key generation'),
('vdf','Verifiable Delay Functions','crypto',4,'Sequential computation, randomness beacons, Wesolowski'),
('zkrollup','ZK-Rollup','zkp',4,'L2 scaling, circuit validity proofs, sequencer'),
('cross-chain','Cross-Chain Bridges','systems',5,'Light client proofs, optimistic vs ZK bridges');

-- ─── Concept Edges (prerequisites) ──────────────────────────────────────────
INSERT INTO concept_edges (concept_id, prerequisite_id) VALUES
('abstract-algebra','set-theory'),
('finite-fields','number-theory'),
('finite-fields','abstract-algebra'),
('ecc','finite-fields'),
('ecdsa','ecc'),
('schnorr','ecc'),
('bls-sigs','ecc'),
('pairings','bls-sigs'),
('aes','finite-fields'),
('sha256','information-theory'),
('merkle','sha256'),
('lsm-tree','filesystems'),
('raft','formal-methods'),
('raft','concurrency'),
('raft','networking'),
('pbft','raft'),
('nakamoto','sha256'),
('nakamoto','ecdsa'),
('pos','nakamoto'),
('dag-consensus','pos'),
('mev','game-theory'),
('mev','pos'),
('markov-chains','pos'),
('schnorr','bulletproofs'),
('bulletproofs','threshold-crypto'),
('pairings','snarks'),
('schnorr','snarks'),
('starks','snarks'),
('shamir-ss','threshold-crypto'),
('bls-sigs','threshold-crypto'),
('threshold-crypto','dkg'),
('dkg','tee'),
('pq-crypto','tee'),
('fhe','pq-crypto'),
('docker','kubernetes'),
('kubernetes','service-mesh'),
('prometheus','otel'),
('gitops','kubernetes'),
('wasm-vm','snarks'),
('ebpf','kubernetes'),
('memory-mgmt','wasm-vm'),
('filesystems','raft'),
('information-theory','sha256'),
('combinatorics','information-theory'),
('snarks','zkrollup'),
('cross-chain','merkle'),
('libp2p','networking'),
('quic','networking'),
('grpc','networking');

-- ─── Default Papers ──────────────────────────────────────────────────────────
INSERT INTO papers (id,title,authors,year,venue,tags,abstract,url) VALUES
('bitcoin2008','Bitcoin: A Peer-to-Peer Electronic Cash System','Satoshi Nakamoto',2008,'','["bitcoin","blockchain","consensus","pow"]',
'A purely peer-to-peer version of electronic cash would allow online payments to be sent directly from one party to another without going through a financial institution.',
'https://bitcoin.org/bitcoin.pdf'),
('raft2014','In Search of an Understandable Consensus Algorithm','Diego Ongaro, John Ousterhout',2014,'USENIX ATC','["raft","consensus","distributed-systems"]',
'Raft is a consensus algorithm designed to be more understandable than Paxos while providing equivalent fault-tolerance guarantees.',
'https://raft.github.io/raft.pdf'),
('pbft1999','Practical Byzantine Fault Tolerance','Miguel Castro, Barbara Liskov',1999,'OSDI','["PBFT","byzantine","consensus","distributed"]',
'This paper describes a new replication algorithm that is able to tolerate Byzantine faults. The algorithm works in asynchronous environments.',
'https://pmg.csail.mit.edu/papers/osdi99.pdf'),
('ethereum2014','Ethereum: A Next-Generation Smart Contract and Decentralized Application Platform','Vitalik Buterin',2014,'','["ethereum","smart-contracts","EVM","blockchain"]',
'A blockchain-based platform that enables Turing-complete smart contracts through the Ethereum Virtual Machine.',
'https://ethereum.org/whitepaper'),
('stark2018','Scalable, transparent, and post-quantum secure computational integrity','Ben-Sasson et al.',2018,'IACR','["stark","zkp","post-quantum","polynomial"]',
'STARKs achieve transparency (no trusted setup), post-quantum security, and efficient verification via FRI.',
'https://eprint.iacr.org/2018/046.pdf'),
('ouroboros2017','Ouroboros: A Provably Secure Proof-of-Stake Blockchain Protocol','Kiayias et al.',2017,'CRYPTO','["proof-of-stake","cardano","security-proof","formal"]',
'The first provably secure PoS blockchain protocol with formal security analysis in the universal composability framework.',
'https://eprint.iacr.org/2016/889.pdf');

SELECT 'Seed complete: ' || COUNT(*) || ' exercises' FROM exercises;
SELECT 'Concepts: ' || COUNT(*) FROM concepts;
SELECT 'Papers: ' || COUNT(*) FROM papers;
