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
	"time"
)

const Difficulty = 23

var node *Node
var cancelMining = make(chan struct{})

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

	go startServer()
	if address != "localhost:3000" {
		node.AddPeer("localhost:3000")
		node.RequestChain("localhost:3000")
	}
	for {
		blockData := strconv.Itoa(rand.Int())
		block := node.Chain.AddBlock(blockData)
		if block != nil {
			node.BroadcastNewBlock(block)
			fmt.Println("New block added and broadcasted to peers\n")

		}
		node.Chain.Print()
		time.Sleep(time.Second * 5) // Wait 5 seconds between mining attempts

	}

}

func startServer() {
	http.HandleFunc("POST /receiveBlock", handleReceiveBlock)
	http.HandleFunc("GET /chain/{host}", handleGetChain)

	fmt.Printf("Starting server on %s\n", node.Address)
	err := http.ListenAndServe(node.Address, nil)
	if err != nil {
		fmt.Printf("Error starting server: %v\n", err)
	}
}

func handleReceiveBlock(w http.ResponseWriter, r *http.Request) {

	var block Block
	err := block.FromJson(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	node.ReceiveBlock(&block)
	//fmt.Printf("received block: %s\n", block.String())
	w.WriteHeader(http.StatusOK)

}

func handleGetChain(w http.ResponseWriter, r *http.Request) {
	srcHost := r.PathValue("host")

	node.AddPeer(srcHost)
	err := json.NewEncoder(w).Encode(node.Chain)
	if err != nil {
		return
	}
}

func printBlockchain() {
	for _, block := range node.Chain.Blocks {
		fmt.Println(strings.Repeat("=", 50))
		fmt.Println(block)
	}
}
