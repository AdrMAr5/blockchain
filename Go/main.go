package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

const Difficulty = 23

var node *Node
var cancelMining = make(chan struct{})
var canMine bool
var mineLock sync.Mutex
var mineCondition *sync.Cond

func main() {
	fmt.Print("Enter the address for this node (e.g., localhost:3000): ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	address := scanner.Text()
	if address == "" {
		address = "localhost:3000"
	}
	if address == "1" {
		address = "localhost:3001"
	}
	if address == "2" {
		address = "localhost:3002"
	}
	if address == "3" {
		address = "localhost:3003"
	}

	node = NewNode(address)
	canMine = true
	mineCondition = sync.NewCond(&mineLock)

	go startServer()
	if address != "localhost:3000" {
		node.JoinNetwork("localhost:3000")
		node.RequestChain("localhost:3000")
	}

	go miningLoop()

	select {}
}

func miningLoop() {
	for {
		mineLock.Lock()
		for !canMine {
			mineCondition.Wait()
		}
		mineLock.Unlock()

		prevBlock := node.Chain.Blocks[len(node.Chain.Blocks)-1]

		blockData := strconv.Itoa(rand.Int())
		block := NewBlock(prevBlock.Index+1, blockData, prevBlock.Hash)
		if block != nil {
			node.Chain.Candidates = append(node.Chain.Candidates, block)
			setCanMine(false)
			err := node.BroadcastAndSetNewBlock(block)
			if err != nil {
				fmt.Printf("Error broadcasting block: %v\n", err)
			} else {
				node.Chain.AddBlock(block, AddedMined)
				setCanMine(true)
			}
		}
	}
}

func setCanMine(value bool) {
	mineLock.Lock()
	canMine = value
	mineLock.Unlock()
	if value {
		mineCondition.Signal()
	}
}

func startServer() {
	http.HandleFunc("POST /receiveBlock/{host}", handleReceiveBlock)
	http.HandleFunc("POST /setBlock", handleSetBlock)
	http.HandleFunc("GET /chain/{host}", handleGetChain)
	http.HandleFunc("POST /join/{host}", handleJoinNetwork)
	http.HandleFunc("POST /addPeer", handleAddPeer)

	fmt.Printf("Starting server on %s\n", node.Address)
	err := http.ListenAndServe(node.Address, nil)
	if err != nil {
		fmt.Printf("Error starting server: %v\n", err)
	}
}

// handleReceiveBlock handles reception of a new block from a peer
func handleReceiveBlock(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("%s: Received receiveBlock from peer\n", time.Now().String())
	cancelMining <- struct{}{}
	setCanMine(false)
	var block Block
	err := block.FromJson(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	node.Chain.Candidates = append(node.Chain.Candidates, &block)
	fmt.Printf("received block added to candidates, %v\n", block.String())
	err = node.Chain.IsValidNewBlock(&block, node.Chain.Blocks[len(node.Chain.Blocks)-1])
	if err != nil {
		if err.Error() == "my chain is longer" {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		if err.Error() == "longer chain exists on peer" {
			http.Error(w, err.Error(), http.StatusOK)
			node.RequestChain(r.PathValue("host"))
			return
		}
		if err.Error() == "invalid previous hash" {
			http.Error(w, err.Error(), http.StatusNotAcceptable)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.WriteHeader(http.StatusOK)
}

// handleSetBlock handles a set block request from a peer
func handleSetBlock(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("%s: Received setBlock from peer\n", time.Now().String())
	var block Block
	err := block.FromJson(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	node.Chain.AddBlock(&block, AddedFromPeer)
	setCanMine(true)
	w.WriteHeader(http.StatusOK)
}

// handleGetChain handles a get chain request from a peer
func handleGetChain(w http.ResponseWriter, r *http.Request) {
	srcHost := r.PathValue("host")

	node.AddPeer(srcHost)
	err := json.NewEncoder(w).Encode(node.Chain)
	if err != nil {
		return
	}
}

// handleJoinNetwork handles a join network request from a new peer
func handleJoinNetwork(w http.ResponseWriter, r *http.Request) {
	srcHost := r.PathValue("host")
	fmt.Printf("Received join request from %s\n", srcHost)
	data := map[string][]string{
		"peers": node.Peers,
	}
	data["peers"] = append(data["peers"], node.Address)
	node.NotifyNetwork(srcHost)
	node.AddPeer(srcHost)
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		fmt.Printf("Error encoding peers: %v\n", err)
	}
}

// handleAddPeer handles a request from current network member to add a new peer
func handleAddPeer(w http.ResponseWriter, r *http.Request) {
	var data map[string]string
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	newPeer := data["peer"]
	fmt.Printf("Received addPeer request, new peer: %s\n", newPeer)
	node.AddPeer(newPeer)
	fmt.Printf("current peers: %v\n", node.Peers)
	w.WriteHeader(http.StatusOK)
}
