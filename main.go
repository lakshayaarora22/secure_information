package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	port       = "8080" // Set the port number here
	difficulty = 1
	reward     = 1 // Reward for successful mining
)

// Transaction represents a single transaction in the blockchain
type Transaction struct {
	Sender   string
	Receiver string
	Amount   int
}

// Block represents each 'item' in the blockchain
type Block struct {
	Index        int
	Timestamp    string
	Transactions []Transaction
	PrevHash     string
	Hash         string
	Nonce        string
	Difficulty   int
}

// Blockchain is a series of validated Blocks
var Blockchain []Block

// Mutex to ensure thread-safe access to the blockchain
var mutex = &sync.Mutex{}

func main() {
	// Initialize the blockchain with the genesis block
	initializeBlockchain()

	// Set up HTTP server to handle API requests
	http.HandleFunc("/", handleAPIRequests)
	log.Println("HTTP Server Listening on port :", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

// Initialize the blockchain with the genesis block
func initializeBlockchain() {
	genesisBlock := Block{}
	genesisBlock = Block{0, time.Now().String(), []Transaction{}, "", calculateHash(genesisBlock), "", difficulty}
	Blockchain = append(Blockchain, genesisBlock)
}

// Handle incoming API requests
func handleAPIRequests(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		handleGetBlockchain(w, r)
	case "POST":
		handleWriteTransaction(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Handle GET requests to retrieve the blockchain
func handleGetBlockchain(w http.ResponseWriter, r *http.Request) {
	bytes, err := json.MarshalIndent(Blockchain, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	io.WriteString(w, string(bytes))
}

// Handle POST requests to write a new transaction to the blockchain
func handleWriteTransaction(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var transaction Transaction
	if err := decoder.Decode(&transaction); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Validate the transaction
	if !isTransactionValid(transaction) {
		http.Error(w, "Invalid transaction", http.StatusBadRequest)
		return
	}

	// Create a new block with the transaction
	newBlock := generateBlock(Blockchain[len(Blockchain)-1], []Transaction{transaction})

	// Verify the new block
	if isBlockValid(newBlock, Blockchain[len(Blockchain)-1]) {
		mutex.Lock()
		Blockchain = append(Blockchain, newBlock)
		mutex.Unlock()
		fmt.Fprintf(w, "Transaction added to Block %d\n", newBlock.Index)
	} else {
		http.Error(w, "Invalid block", http.StatusInternalServerError)
	}
}

// Generate a new block with the provided transactions
func generateBlock(oldBlock Block, transactions []Transaction) Block {
	var newBlock Block

	t := time.Now()

	newBlock.Index = oldBlock.Index + 1
	newBlock.Timestamp = t.String()
	newBlock.Transactions = transactions
	newBlock.PrevHash = oldBlock.Hash
	newBlock.Difficulty = difficulty

	// Mining: Find nonce that satisfies the difficulty
	for i := 0; ; i++ {
		newBlock.Nonce = strconv.Itoa(i)
		hash := calculateHash(newBlock)
		if isHashValid(hash, newBlock.Difficulty) {
			newBlock.Hash = hash
			break
		}
	}

	return newBlock
}

// Calculate the hash of a block
func calculateHash(block Block) string {
	record := strconv.Itoa(block.Index) + block.Timestamp + block.PrevHash + fmt.Sprint(block.Transactions) + block.Nonce
	h := sha256.New()
	h.Write([]byte(record))
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}

// Check if a hash meets the required difficulty level
func isHashValid(hash string, difficulty int) bool {
	prefix := ""
	for i := 0; i < difficulty; i++ {
		prefix += "0"
	}
	return hash[:difficulty] == prefix
}

// Check if a block is valid
func isBlockValid(newBlock, oldBlock Block) bool {
	if oldBlock.Index+1 != newBlock.Index {
		return false
	}

	if oldBlock.Hash != newBlock.PrevHash {
		return false
	}

	if calculateHash(newBlock) != newBlock.Hash {
		return false
	}

	return true
}

// Check if a transaction is valid
func isTransactionValid(transaction Transaction) bool {
	// Check if sender, receiver, and amount are non-empty and valid
	if transaction.Sender == "" || transaction.Receiver == "" || transaction.Amount <= 0 {
		return false
	}
	return true
}
