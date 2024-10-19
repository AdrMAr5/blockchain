package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
)

type Node struct {
	Address string
	Chain   *Chain
	Peers   []string
	mu      sync.Mutex
}

func NewNode(address string) *Node {
	return &Node{
		Address: address,
		Chain:   NewChain(),
		Peers:   make([]string, 0),
	}
}

func (n *Node) AddPeer(address string) {
	fmt.Printf("Adding peer: %s\n", address)
	n.mu.Lock()
	defer n.mu.Unlock()
	n.Peers = append(n.Peers, address)
}

func (n *Node) BroadcastNewBlock(block *Block) {
	for _, peer := range n.Peers {
		go n.sendBlock(peer, block)
	}
}

func (n *Node) sendBlock(peer string, block *Block) {
	writer := new(bytes.Buffer)
	err := block.ToJson(writer)
	if err != nil {
		fmt.Printf("Error marshaling block: %v\n", err)
		return
	}

	resp, err := http.Post(fmt.Sprintf("http://%s/receiveBlock", peer), "application/json", bytes.NewBuffer(writer.Bytes()))
	if err != nil {
		fmt.Printf("Error sending block to peer %s: %v\n", peer, err)
		return
	}
	defer resp.Body.Close()
}

func (n *Node) ReceiveBlock(block *Block) {
	if n.Chain.IsValidNewBlock(block, n.Chain.Blocks[len(n.Chain.Blocks)-1]) {
		n.Chain.AddBlockFromPeer(block)
		fmt.Printf("Received and added new block: %s\n", block)
	} else {
		fmt.Printf("Received invalid block: %s\n", block)
	}
}

func (n *Node) RequestChain(peer string) {
	resp, err := http.Get(fmt.Sprintf("http://%s/chain/%s", peer, n.Address))
	if err != nil {
		fmt.Printf("Error requesting chain from peer %s: %v\n", peer, err)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response body: %v\n", err)
		return
	}

	var newChain Chain
	err = json.Unmarshal(body, &newChain)
	if err != nil {
		fmt.Printf("Error unmarshaling chain: %v\n", err)
		return
	}

	err = n.Chain.ReplaceChain(&newChain)
	if err != nil {
		fmt.Printf("Error replacing chain: %v\n", err)
	} else {
		fmt.Println("Chain replaced with new chain from peer")
	}

}
