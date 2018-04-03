package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/boltdb/bolt"
)

func IntTohex(in int64) []byte {
	return []byte(fmt.Sprintf("%x", in))
}

type Block struct {
	Timestamp     int64
	Data          []byte
	PrevBlockHash []byte
	Hash          []byte
	Nonce         int
}

func NewBlock(data string, prevBlockHash []byte) *Block {
	block := &Block{time.Now().Unix(), []byte(data), prevBlockHash, []byte{}, 0}
	pow := NewProofOfWork(block)
	nonce, hash := pow.Run()

	block.Hash = hash[:]
	block.Nonce = nonce
	return block
}
func NewGenesisBlock() *Block {
	return NewBlock("Genesis Block", []byte{})
}
func DeserializeBlock(d []byte, block *Block) {
	decoder := gob.NewDecoder(bytes.NewReader(d))
	decoder.Decode(block)
}
func (b *Block) Serialize() []byte {
	var resutl bytes.Buffer
	encoder := gob.NewEncoder(&resutl)
	encoder.Encode(b)
	return resutl.Bytes()
}
func (b *Block) Clear() {
	b.Data = nil
	b.Hash = nil
	b.Nonce = 0
	b.PrevBlockHash = nil
	b.Timestamp = 0
}

///
type Blockchain struct {
	tip []byte
	db  *bolt.DB
}

const dbFile = "./bc.bolt"

var blocksBucket = []byte{'b', 'l', 'k'}

func NewBlockchain() *Blockchain {
	var tip []byte
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		panic(err.Error())
	}
	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(blocksBucket)
		if b == nil {
			genesis := NewGenesisBlock()
			b, _ := tx.CreateBucket(blocksBucket)
			b.Put(genesis.Hash, genesis.Serialize())
			b.Put([]byte{'l'}, genesis.Hash)
			tip = genesis.Hash
		} else {
			tip = b.Get([]byte{'l'})
		}
		return nil
	})
	bc := Blockchain{tip, db}
	return &bc
}
func (bc *Blockchain) AddBlock(data string) {
	var lastHash []byte
	bc.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(blocksBucket)
		lastHash = b.Get([]byte{'l'})
		return nil
	})
	newBlock := NewBlock(data, lastHash)
	bc.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(blocksBucket)
		b.Put(newBlock.Hash, newBlock.Serialize())
		b.Put([]byte{'l'}, newBlock.Hash)
		bc.tip = newBlock.Hash
		return nil
	})
}
func (bc *Blockchain) Iterator() *BlockchainIterator {
	bci := &BlockchainIterator{
		bc.tip,
		bc.db,
	}
	return bci
}

type BlockchainIterator struct {
	currentHash []byte
	db          *bolt.DB
}

func (iter *BlockchainIterator) Next(block *Block) bool {
	block.Clear()
	if len(iter.currentHash) == 0 {
		return false
	}
	err := iter.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(blocksBucket)
		encodedBlock := b.Get(iter.currentHash)
		DeserializeBlock(encodedBlock, block)
		return nil
	})
	if err != nil {
		return false
	}
	iter.currentHash = block.PrevBlockHash
	return true
}

///////////////
type ProofOfWork struct {
	block  *Block
	target *big.Int
}

const targetBits = 4

func NewProofOfWork(b *Block) *ProofOfWork {
	target := big.NewInt(1)
	target.Lsh(target, uint(256-targetBits))
	pow := &ProofOfWork{b, target}
	return pow
}
func (pow *ProofOfWork) prepareData(nonce int) []byte {
	data := bytes.Join(
		[][]byte{
			pow.block.PrevBlockHash,
			pow.block.Data,
			IntTohex(pow.block.Timestamp),
			IntTohex(int64(targetBits)),
			IntTohex(int64(nonce)),
		},
		[]byte{},
	)
	return data
}
func (pow *ProofOfWork) Run() (int, []byte) {
	var hashInt big.Int
	var hash [32]byte
	nonce := 0
	maxNonce := math.MaxInt64
	for nonce < maxNonce {
		data := pow.prepareData(nonce)
		hash = sha256.Sum256(data)
		hashInt.SetBytes(hash[:])
		if hashInt.Cmp(pow.target) == -1 {
			fmt.Printf("[%x]-->[ok]\n", hash)
			break
		} else {
			fmt.Printf("[%x]--[xx]\n", hash)
			nonce++
		}
	}
	fmt.Println()
	fmt.Println()
	return nonce, hash[:]
}
func (pow *ProofOfWork) Validate() bool {
	var hashInt big.Int
	data := pow.prepareData(pow.block.Nonce)
	hash := sha256.Sum256(data)
	hashInt.SetBytes(hash[:])
	isValid := hashInt.Cmp(pow.target) == -1
	return isValid
}

///
// func main() {
// 	bc := NewBlockchain()
// 	bc.AddBlock("send 1 btc to ivan")
// 	bc.AddBlock("send 2 more btc to ivan")
// 	iter := bc.Iterator()
// 	block := new(Block)
// 	var idx int
// 	for iter.Next(block) {
// 		fmt.Println("---:")
// 		fmt.Printf("%x %x\n", block.Hash, block.PrevBlockHash)
// 		idx++
// 	}
// }
