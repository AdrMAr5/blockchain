package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
)

type lastOperation string

const (
	AddedFromPeer lastOperation = "AddedFromPeer"
	AddedMined    lastOperation = "AddedMined"
)

type Chain struct {
	Blocks        []*Block
	Candidates    []*Block
	mu            sync.Mutex
	lastOperation lastOperation
}

func NewChain() *Chain {
	chain := &Chain{Candidates: make([]*Block, 0)}
	chain.createGenesisBlock()
	return chain
}

// function to get block from Candidates by hash
func (c *Chain) getCandidateByHash(hash [32]byte) *Block {
	for _, block := range c.Candidates {
		if block.Hash == hash {
			return block
		}
	}
	return nil

}

func (c *Chain) Print() {
	for i := 0; i < len(c.Blocks); i++ {
		fmt.Printf("%s\n", c.Blocks[i].String())
	}
	fmt.Printf("last operation: %s\n\n", c.lastOperation)
}

func (c *Chain) createGenesisBlock() {
	genesisBlock := NewBlock(0, "Genesis Block", [32]byte{})
	c.Blocks = append(c.Blocks, genesisBlock)
}

func (c *Chain) AddBlock(block *Block, operation lastOperation) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Blocks = append(c.Blocks, block)
	c.lastOperation = operation
	c.Print()
}

func (c *Chain) IsValidNewBlock(newBlock, prevBlock *Block) error {
	if prevBlock.Index+1 != newBlock.Index {
		if prevBlock.Index+1 < newBlock.Index {
			fmt.Printf("invalid indexes: %d and %d\n", prevBlock.Index, newBlock.Index)
			return errors.New("longer chain exists on peer")
		} else {
			fmt.Printf("invalid indexes: %d and %d\n", prevBlock.Index, newBlock.Index)
			return errors.New("my chain is longer")
		}

	}
	if prevBlock.Hash != newBlock.PreviousHash {
		fmt.Println("invalid previous hash")
		return errors.New("invalid previous hash")
	}
	return nil
}

func (c *Chain) ReplaceChain(newChain *Chain) error {
	c.mu.Lock()
	c.Blocks = newChain.Blocks
	c.mu.Unlock()
	return nil
}

func (c *Chain) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Blocks []*Block `json:"blocks"`
	}{
		Blocks: c.Blocks,
	})
}

func (c *Chain) UnmarshalJSON(data []byte) error {
	var v struct {
		Blocks []*Block `json:"blocks"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	c.Blocks = v.Blocks
	return nil
}
