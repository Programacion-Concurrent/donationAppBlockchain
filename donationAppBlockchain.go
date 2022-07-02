package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

type MessageType int32

const (
	NEWHOST   MessageType = 0
	ADDHOST   MessageType = 1
	ADDBLOCK  MessageType = 2
	NEWBLOCK  MessageType = 3
	SETBLOCKS MessageType = 4
	PROTOCOL              = "tcp"
	NEWMR                 = 1
	LISTMR                = 2
	LISTHOSTS             = 3
)

/******************BCIP**********************/
var HOSTS []string
var LOCALHOST string

type RequestBody struct {
	Message     string
	MessageType MessageType
}

func GetMessage(conn net.Conn) string {
	reader := bufio.NewReader(conn)
	data, _ := reader.ReadString('\n')
	return strings.TrimSpace(data)
}

func SendMessage(toHost string, message string) {
	conn, _ := net.Dial(PROTOCOL, toHost)
	defer conn.Close()
	fmt.Fprintln(conn, message)
}

func SendMessageWithReply(toHost string, message string) string {
	conn, _ := net.Dial(PROTOCOL, toHost)
	defer conn.Close()
	fmt.Fprintln(conn, message)
	return GetMessage(conn)
}

func RemoveHost(index int, hosts []string) []string {
	n := len(hosts)
	hosts[index] = hosts[n-1]
	hosts[n-1] = ""
	return hosts[:n-1]
}

func RemoveHostByValue(ip string, hosts []string) []string {
	for index, host := range hosts {
		if host == ip {
			return RemoveHost(index, hosts)
		}
	}
	return hosts
}

func Broadcast(newHost string) {
	for _, host := range HOSTS {
		data := append(HOSTS, newHost, LOCALHOST)
		data = RemoveHostByValue(host, data)
		requestBroadcast := RequestBody{
			Message:     strings.Join(data, ","),
			MessageType: ADDHOST,
		}
		broadcastMessage, _ := json.Marshal(requestBroadcast)
		SendMessage(host, string(broadcastMessage))
	}
}

func BroadcastBlock(newBlock Block) {
	for _, host := range HOSTS {
		data, _ := json.Marshal(newBlock)
		requestBroadcast := RequestBody{
			Message:     string(data),
			MessageType: ADDBLOCK,
		}
		broadcastMessage, _ := json.Marshal(requestBroadcast)
		SendMessage(host, string(broadcastMessage))
	}
}

func BCIPServer(end chan<- int, updatedBlocks chan<- int) {
	ln, _ := net.Listen(PROTOCOL, LOCALHOST)
	defer ln.Close()
	for {
		conn, _ := ln.Accept()
		defer conn.Close()
		request := RequestBody{}
		data := GetMessage(conn)
		_ = json.Unmarshal([]byte(data), &request)
		if request.MessageType == NEWHOST {
			message := strings.Join(append(HOSTS, LOCALHOST), ",")
			requestClient := RequestBody{
				Message:     message,
				MessageType: ADDHOST,
			}
			clientMessage, _ := json.Marshal(requestClient)
			SendMessage(request.Message, string(clientMessage))
			Broadcast(request.Message)
			HOSTS = append(HOSTS, request.Message)
		} else if request.MessageType == ADDHOST {
			HOSTS = strings.Split(request.Message, ",")
		} else if request.MessageType == NEWBLOCK {
			blocksMessage, _ := json.Marshal(localBlockChain.Chain)
			setBlocksRequest := RequestBody{
				Message:     string(blocksMessage),
				MessageType: SETBLOCKS,
			}
			setBlocksMessage, _ := json.Marshal(setBlocksRequest)
			SendMessage(request.Message, string(setBlocksMessage))
		} else if request.MessageType == SETBLOCKS {
			_ = json.Unmarshal([]byte(request.Message), &localBlockChain.Chain)
			updatedBlocks <- 0
		} else if request.MessageType == ADDBLOCK {
			block := Block{}
			src := []byte(request.Message)
			json.Unmarshal(src, &block)
			localBlockChain.Chain = append(localBlockChain.Chain, block)
		}
	}
	end <- 0
}

/******************BLOCKCHAIN**********************/

type DonationRecord struct {
	Name       string
	Ong       string
	Amount   string
	Description string
}

type Block struct {
	Index        int
	Timestamp    time.Time
	Data         DonationRecord
	PreviousHash string
	Hash         string
}

func (block *Block) CalculateHash() string {
	src:=fmt.Sprintf("%d-%s-%s",block.Index,block.Timestamp.String(),block.Data)
	h := sha256.New()
	h.Write([]byte(src))
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}

type BlockChain struct {
	Chain []Block
}

func (blockChain *BlockChain) CreateGenesisBlock() Block {
	block := Block{
		Index:        0,
		Timestamp:    time.Now(),
		Data:         DonationRecord{},
		PreviousHash: "0",
	}
	block.Hash = block.CalculateHash()
	return block
}

func (blockChain *BlockChain) GetLatesBlock() Block {
	n := len(blockChain.Chain)
	return blockChain.Chain[n-1]
}

func getMostCommonHash(hashes []Block) string {
	// ENTRADA: HASHES DE LOS BLOQUES VECINOS
	m := make(map[string]int)
  	compare := 0
  	var mostFrequent string

	for _, h := range hashes {
		word := h.Hash
		m[word] = m[word] + 1 
		if m[word] > compare { 
			 compare = m[word]  
			 mostFrequent = h.Hash
		}
	}
	
	return mostFrequent
}

func (blockChain *BlockChain) AddBlock(block Block) {
	block.Timestamp = time.Now()
	block.Index = blockChain.GetLatesBlock().Index + 1
	block.PreviousHash = blockChain.GetLatesBlock().Hash
	block.Hash = block.CalculateHash()

	var hashesInfo []Block
	blocks := localBlockChain.Chain[1:]

	fmt.Println(block.Hash) // HASH QUE SE GENERA AL CREAR EL BLOQUE
	if len(blocks) == 0 {
		
	}else{
		for _, block1 := range blocks {
			hashesInfo = append(hashesInfo, block1)
		}
		
	}
	fmt.Println(getMostCommonHash(hashesInfo))
	blockChain.Chain = append(blockChain.Chain, block)
}

func (blockChain *BlockChain) IsChainValid() bool {
	n := len(blockChain.Chain)
	for i := 1; i < n; i++ {
		currentBlock := blockChain.Chain[i]
		previousBlock := blockChain.Chain[i-1]
		if currentBlock.Hash != currentBlock.CalculateHash() {
			return false
		}
		if currentBlock.PreviousHash != previousBlock.Hash {
			return false
		}
	}
	return true
}

func CreateBlockChain() BlockChain {
	bc := BlockChain{}
	genesisBlock := bc.CreateGenesisBlock()
	bc.Chain = append(bc.Chain, genesisBlock)
	return bc
}

var localBlockChain BlockChain

/******************MAIN**********************/

func PrintDonationRecords() {
	blocks := localBlockChain.Chain[1:]
	for index, block := range blocks {
		donationRecord := block.Data
		fmt.Printf("- - - Donation Record No. %d - - - \n", index+1)
		fmt.Printf("\tName: %s\n", donationRecord.Name)
		fmt.Printf("\tOng: %s\n", donationRecord.Ong)
		fmt.Printf("\tAmount: %s\n", donationRecord.Amount)
		fmt.Printf("\tDescription: %s\n", donationRecord.Description)
	}
}

func PrintMyDonations(donations []DonationRecord) {
	for index, donation := range donations {
		fmt.Printf("- - - My Donation Records No. %d - - - \n", index+1)
		fmt.Printf("\tName: %s\n", donation.Name)
		fmt.Printf("\tOng: %s\n", donation.Ong)
		fmt.Printf("\tAmount: %s\n", donation.Amount)
		fmt.Printf("\tDescription: %s\n", donation.Description)
	}
}

func PrintHosts() {
	fmt.Println("- - - HOSTS - - -")
	const first = 0
	fmt.Printf("\t%s (Your host)\n", LOCALHOST)
	for _, host := range HOSTS {
		fmt.Printf("\t%s\n", host)
	}
}

func main() {
	var dest string
	var donationsHost []DonationRecord

	end := make(chan int)
	updatedBlocks := make(chan int)
	fmt.Print("Enter your host: ")
	fmt.Scanf("%s\n", &LOCALHOST)
	fmt.Print("Enter destination host(Empty to be the first node): ")
	fmt.Scanf("%s\n", &dest)
	go BCIPServer(end, updatedBlocks)
	localBlockChain = CreateBlockChain()
	if dest != "" {
		requestBody := &RequestBody{
			Message:     LOCALHOST,
			MessageType: NEWHOST,
		}
		requestMessage, _ := json.Marshal(requestBody)
		SendMessage(dest, string(requestMessage))
		requestBody.MessageType = NEWBLOCK
		requestMessage, _ = json.Marshal(requestBody)
		SendMessage(dest, string(requestMessage))
		<-updatedBlocks
	}
	var action int
	fmt.Println("Welcome to DonationRecordApp! :)")
	in := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("1. New Donation Record\n2. List Donation Records\n3. List Hosts\n4. List My Donation Records\n")
		fmt.Print("Enter action(1|2|3):")
		fmt.Scanf("%d\n", &action)
		if action == NEWMR {
			donationRecord := DonationRecord{}
			fmt.Println("- - - Register - - -")
			fmt.Print("Enter user name: ")
			donationRecord.Name, _ = in.ReadString('\n')
			fmt.Print("Enter ONG: ")
			donationRecord.Ong, _ = in.ReadString('\n')
			fmt.Print("Enter amount: ")
			donationRecord.Amount, _ = in.ReadString('\n')
			fmt.Print("Enter a short message for the ONG: ")
			donationRecord.Description, _ = in.ReadString('\n')			
			newBlock := Block{
				Data: donationRecord,
			}
			localBlockChain.AddBlock(newBlock)
			BroadcastBlock(newBlock)
			donationsHost = append(donationsHost, donationRecord)
			
			fmt.Println("We are processing your transaction, wait ...")
			time.Sleep(2 * time.Second)
			fmt.Println("************************************")
			fmt.Println("You have registered successfully! :)")
			fmt.Println("************************************")
			time.Sleep(1 * time.Second)
			PrintDonationRecords()
		} else if action == LISTMR {
			PrintDonationRecords()
		} else if action == LISTHOSTS {
			PrintHosts()
		} else if action == 4 {
			PrintMyDonations(donationsHost)
		}
	}
	<-end
}
