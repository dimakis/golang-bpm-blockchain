package main
// Written by Dimitri Saridakis
// @dimakis, this is a program I am writing following a tutorial @https://mycoralhealth.medium.com/code-your-own-blockchain-in-less-than-200-lines-of-go-e296282bcffc,
// to build a blockchain that keeps a record of heart BPM, I am using this project to learn more about both GoLang and Blockchain
import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/davecgh/go-spew/spew"
	"github.com/joho/godotenv"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
)

// this is the structure for each block which will form the basis of the chain
type Block struct {
	Index int
	Timestamp string
	BPM int
	Hash string
	PrevHash string
}

var Blockchain []Block

// takes in a 'Block', calculates the hash, returns a SHA256 hashed string representation of the block
func calculateHash(block Block) string {
	record := string(block.Index) + block.Timestamp + string(block.BPM) + block.PrevHash
	h := sha256.New()
	h.Write([]byte(record))
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}

// takes in previous or 'oldBlock' and new BPM, uses these to generate new block, returns newBlock
func generateBlock( oldBlock Block, BPM int ) ( Block, error ) {

	var newBlock Block

	t := time.Now()

	newBlock.Index = oldBlock.Index + 1
	newBlock.Timestamp = t.String()
	newBlock.BPM = BPM
	newBlock.PrevHash = oldBlock.Hash
	newBlock.Hash = calculateHash(newBlock)

	return newBlock, nil
}

// this func checks to determine if the block is valid:
// 1: it checks that we've incremented index as expected
// 2: checks that 'PrevHash' is the same as the 'Hash' value that came before it
// 3: double checks the Hash again by rerunning the calculate hash func
// returns: bool, indicating block is valid or not
func isBlockValid(newBlock, oldBlock Block) bool  {
	if oldBlock.Index+1 != newBlock.Index {
		return false
	}

	if oldBlock.Hash != newBlock.PrevHash	{
		return false
	}

	if calculateHash( newBlock ) != newBlock.Hash {
		return false
	}

	return true
}

// for scenarios where two nodes both add to the chain simultaneously, we accept the longer chain as it will
// have more data
func replaceChain( newBlocks []Block)	{
	if len( newBlocks ) > len( Blockchain ) {
		Blockchain = newBlocks
	}
}

// creation of the webserver for use with read and write of the blockvchain
func run() error {
	mux := makeMuxRouter()
	httpAddr := os.Getenv("PORT")
	log.Println( "Listening on ", os.Getenv( "PORT" ))
	s := &http.Server {
		Addr: ":" + httpAddr,
		Handler: mux,
		ReadTimeout: 10 * time.Second,
		WriteTimeout: 10 * time.Second,
		MaxHeaderBytes:  1 << 20,
	}

	if err := s.ListenAndServe();err != nil {
		return err
	}

	return nil
}

// defining handlers to use in our creating of the server func,
// GET request handles read of the Blockchain
// POST handles write
func makeMuxRouter() http.Handler	{
	muxRouter := mux.NewRouter()
	muxRouter.HandleFunc("/", handleGetBlockchain).Methods("GET")
	muxRouter.HandleFunc("/", handleWriteBlock).Methods("POST")
	return muxRouter
}

// GET handler
func handleGetBlockchain( w http.ResponseWriter, r *http.Request) {
	bytes, err := json.MarshalIndent(Blockchain, "", " ")
	if err != nil {
		http.Error( w, err.Error(), http.StatusInternalServerError)
		return
	}
	io.WriteString(w , string(bytes))
}

// this takes in the request body of the of the JSON POST request used to write new blocks
type Message struct {
	BPM int
}

// create a new message and decode the request body into it.
// A new block is created with previous block and the new BPM, then validated.
// useful notes: spew.Dump 'pretty prints' structs to console
func handleWriteBlock( w http.ResponseWriter, r * http.Request)	{
	var m Message

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode( &m); err != nil {
		respondWithJSON( w , r, http.StatusBadRequest, r.Body )
		return
	}

	defer r.Body.Close()

	newBlock, err := generateBlock(Blockchain[len(Blockchain)-1], m.BPM)
	if err != nil {
		respondWithJSON( w, r, http.StatusInternalServerError, m )
		return
	}
	if isBlockValid(newBlock, Blockchain[len(Blockchain)-1]) {
		newBlockchain := append(Blockchain, newBlock)
		replaceChain(newBlockchain)
		spew.Dump(Blockchain)
	}

	respondWithJSON(w, r, http.StatusCreated, newBlock)
}

// alerted with json on the status of the POST request
func respondWithJSON( w http.ResponseWriter, r *http.Request, code int, payload interface{}) {
	response, err := json.MarshalIndent(payload, "", " ")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("HTTP 500: Internal Server Error"))
	}
	w.WriteHeader(code)
	w.Write(response)
}

// here the genesis block is created as the initial block in the chain.
// the genesis block is in its own goRoutine, so the webserver logic and the blockchain logic are seperate
func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		t := time.Now()
		genesisBlock := Block { 0, t.String(),0,  "",""}
		spew.Dump(genesisBlock)
		Blockchain = append(Blockchain, genesisBlock)
	}()
	log.Fatal(run())
}

