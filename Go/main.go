package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
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

	node = NewNode(address)
	canMine = true
	mineCondition = sync.NewCond(&mineLock)

	go startServer()
	if address != "localhost:3000" {
		node.AddPeer("localhost:3000")
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
		node.Chain.Candidates = append(node.Chain.Candidates, block)
		if block != nil {
			setCanMine(false)
			err := node.BroadcastAndSetNewBlock(block)
			if err != nil {
				fmt.Printf("Error broadcasting block: %v\n", err)
			} else {
				node.Chain.AddBlock(block, AddedMined)
				setCanMine(true)
				//fmt.Println("new block added to candidates and broadcasted to peers")
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

	fmt.Printf("Starting server on %s\n", node.Address)
	err := http.ListenAndServe(node.Address, nil)
	if err != nil {
		fmt.Printf("Error starting server: %v\n", err)
	}
}

func handleReceiveBlock(w http.ResponseWriter, r *http.Request) {
	//fmt.Printf("Received block from peer %s", r.PathValue("host"))
	setCanMine(false)
	cancelMining <- struct{}{}
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
	}
}

func handleGetChain(w http.ResponseWriter, r *http.Request) {
	srcHost := r.PathValue("host")

	node.AddPeer(srcHost)
	err := json.NewEncoder(w).Encode(node.Chain)
	if err != nil {
		return
	}
}

func handleSetBlock(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Received setBlock from peer\n")
	var block Block
	err := block.FromJson(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	node.Chain.AddBlock(&block, AddedFromPeer)
	setCanMine(true)
	//fmt.Println("block added to blockchain")
	w.WriteHeader(http.StatusOK)
}

func printBlockchain() {
	for _, block := range node.Chain.Blocks {
		fmt.Println(strings.Repeat("=", 50))
		fmt.Println(block)
	}
}
