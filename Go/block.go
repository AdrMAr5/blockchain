package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"time"
)

type Block struct {
	Index        int      `json:"index"`
	Timestamp    int64    `json:"timestamp"`
	Data         string   `json:"data"`
	PreviousHash [32]byte `json:"previousHash"`
	Hash         [32]byte `json:"hash"`
	Nonce        int      `json:"nonce"`
}

func NewBlock(index int, data string, previousHash [32]byte) *Block {
	block := &Block{
		Index:        index,
		Timestamp:    time.Now().Unix(),
		Data:         data,
		PreviousHash: previousHash,
	}
	block.Mine(Difficulty)
	return block
}

func (b *Block) Mine(difficulty int) {
	target := new(big.Int).Lsh(big.NewInt(1), uint(256-difficulty))

	for {
		b.Hash = b.calculateHash()
		hashInt := new(big.Int).SetBytes(b.Hash[:])

		if hashInt.Cmp(target) == -1 {
			return
		}
		b.Nonce++
	}
}

func (b *Block) calculateHash() [32]byte {
	record := fmt.Sprintf("%d%d%s%x%d", b.Index, b.Timestamp, b.Data, b.PreviousHash, b.Nonce)
	return sha256.Sum256([]byte(record))
}

func (b *Block) String() string {
	return fmt.Sprintf("Index: %d, Timestamp: %d, Data: %s, Hash: %x, PrevHash: %x, Nonce: %d",
		b.Index, b.Timestamp, b.Data, b.Hash, b.PreviousHash, b.Nonce)
}

func (b *Block) ToJson(writer io.Writer) error {
	encoder := json.NewEncoder(writer)
	return encoder.Encode(b)
}

func (b *Block) FromJson(reader io.Reader) error {
	decoder := json.NewDecoder(reader)
	return decoder.Decode(b)
}
