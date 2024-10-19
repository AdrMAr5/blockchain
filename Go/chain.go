package main

import (
	"encoding/json"
	"fmt"
	"sync"
)

type Chain struct {
	Blocks []*Block
	mu     sync.Mutex
}

func NewChain() *Chain {
	chain := &Chain{}
	chain.createGenesisBlock()
	return chain
}
func (c *Chain) Print() {
	for i := 0; i < len(c.Blocks); i++ {
		fmt.Printf("%d: %s\n", c.Blocks[i].Index, c.Blocks[i].String())
	}
}

func (c *Chain) createGenesisBlock() {
	genesisBlock := NewBlock(0, "Genesis Block", [32]byte{})
	c.Blocks = append(c.Blocks, genesisBlock)
}

func (c *Chain) AddBlock(data string) *Block {
	c.mu.Lock()
	defer c.mu.Unlock()
	prevBlock := c.Blocks[len(c.Blocks)-1]
	newBlock := NewBlock(prevBlock.Index+1, data, prevBlock.Hash)
	if newBlock == nil {
		return nil
	}
	if c.IsValidNewBlock(newBlock, prevBlock) {
		c.Blocks = append(c.Blocks, newBlock)
		fmt.Printf("Adding my mined block %d\n", newBlock.Index)
		return newBlock
	}
	return nil
}
func (c *Chain) AddBlockFromPeer(block *Block) {
	fmt.Printf("Adding block from peer, %d\n", block.Index)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Blocks = append(c.Blocks, block)
}

func (c *Chain) IsValidNewBlock(newBlock, prevBlock *Block) bool {
	if prevBlock.Index+1 != newBlock.Index {
		fmt.Printf("invalid indexes: %d and %d\n", prevBlock.Index, newBlock.Index)
		return false
	}
	if prevBlock.Hash != newBlock.PreviousHash {
		fmt.Println("invalid previous hash")
		return false
	}
	return true
}

func (c *Chain) IsValidChain() bool {
	for i := 1; i < len(c.Blocks); i++ {
		if !c.IsValidNewBlock(c.Blocks[i], c.Blocks[i-1]) {
			return false
		}
	}
	return true
}

func (c *Chain) ReplaceChain(newChain *Chain) error {
	//if !newChain.IsValidChain() {
	//	return errors.New("invalid chain")
	//}
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
