package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"slices"
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
	if slices.Contains(n.Peers, address) {
		return
	}
	fmt.Printf("Adding peer: %s\n", address)
	n.mu.Lock()
	defer n.mu.Unlock()

	n.Peers = append(n.Peers, address)
}

func (n *Node) BroadcastAndSetNewBlock(block *Block) error {
	var errs []error
	for _, peer := range n.Peers {
		err := n.sendBlock(peer, block)
		if err != nil {
			if err.Error() == "invalid previous hash" {
				fmt.Printf("Peer %s rejected block, requesting chain from peer\n", peer)
				n.RequestChain(peer)
			}
			if err.Error() == "invalid block index" {
				fmt.Printf("Peer %s rejected block, requesting chain from peer\n", peer)
				n.RequestChain(peer)
			} else {
				fmt.Printf("Error sending block to peer %s: %v\n", peer, err)
			}
		}
		errs = append(errs, err)
	}

	for _, err := range errs {
		if err != nil && (err.Error() == "invalid previous hash" || err.Error() == "invalid block index") {
			return errors.New("chain replaced with longer chain from peer")
		}
	}
	err := n.PostSetBlock(block)
	if err != nil {
		fmt.Printf("Error sending block to peers %v\n", err)
		return err
	}

	return nil
}

func (n *Node) sendBlock(peer string, block *Block) error {
	writer := new(bytes.Buffer)
	err := block.ToJson(writer)
	if err != nil {
		fmt.Printf("Error marshaling block: %v\n", err)
		return err
	}

	resp, err := http.Post(fmt.Sprintf("http://%s/receiveBlock/%s", peer, n.Address), "application/json", bytes.NewBuffer(writer.Bytes()))
	if err != nil {
		fmt.Printf("Error sending block to peer %s: %v\n", peer, err)
		return err
	}
	if resp.StatusCode == http.StatusOK {
		return nil
	}
	if resp.StatusCode == http.StatusBadRequest {
		return errors.New("could not unmarshal block")
	}
	if resp.StatusCode == http.StatusConflict {
		return errors.New("invalid block index")
	}
	if resp.StatusCode == http.StatusNotAcceptable {
		return errors.New("invalid previous hash")

	}
	defer resp.Body.Close()
	return nil
}
func (n *Node) PostSetBlock(block *Block) error {
	writer := new(bytes.Buffer)
	err := block.ToJson(writer)
	if err != nil {
		fmt.Printf("Error marshaling block: %v\n", err)
	}
	for _, peer := range n.Peers {
		resp, err := http.Post(fmt.Sprintf("http://%s/setBlock", peer), "application/json", bytes.NewBuffer(writer.Bytes()))
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return errors.New(fmt.Sprintf("could not set block code: %d", resp.StatusCode))
		}
		resp.Body.Close()
	}
	return nil
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
		fmt.Println("Chain replaced with new chain from peer\n")
	}
}
