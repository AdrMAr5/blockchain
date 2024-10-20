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
)

const Difficulty = 20

var node *Node
var cancelMining = make(chan struct{})
var canMine bool

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

	node = NewNode(address)
	canMine = true

	go startServer()
	if address != "localhost:3000" {
		node.AddPeer("localhost:3000")
		node.RequestChain("localhost:3000")
	}

	for {
		if canMine {
			prevBlock := node.Chain.Blocks[len(node.Chain.Blocks)-1]

			blockData := strconv.Itoa(rand.Int())
			block := NewBlock(prevBlock.Index+1, blockData, prevBlock.Hash)
			node.Chain.Candidates = append(node.Chain.Candidates, block)
			if block != nil {
				canMine = false
				err := node.BroadcastAndSetNewBlock(block)
				if err != nil {
					fmt.Printf("Error broadcasting block: %v\n", err)
				} else {
					node.Chain.AddBlock(block)
					canMine = true
					fmt.Println("new block added to candidates and broadcasted to peers")
				}
			}
			node.Chain.Print()
		}
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

// cheks if block can be added and return appropirate error
func handleReceiveBlock(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Received block from peer")
	canMine = false
	cancelMining <- struct{}{}
	var block Block
	err := block.FromJson(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	node.Chain.Candidates = append(node.Chain.Candidates, &block)
	fmt.Printf("received block added to candidates, %v\n", block)
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

// this endpoint is used force to add this block to blockchain
func handleSetBlock(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Received setBlock from peer\n")
	var block Block
	err := block.FromJson(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	fromCandidates := node.Chain.getCandidateByHash(block.Hash)
	if fromCandidates == nil {
		http.Error(w, "block not found in candidates", http.StatusNotFound)
		return
	}
	node.Chain.AddBlock(fromCandidates)
	canMine = true
	fmt.Println("block added to blockchain")
	w.WriteHeader(http.StatusOK)
}

func printBlockchain() {
	for _, block := range node.Chain.Blocks {
		fmt.Println(strings.Repeat("=", 50))
		fmt.Println(block)
	}
}
