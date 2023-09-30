package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// struct to store the details of a node
type Nodes struct {
	Conn []net.Conn
	Mux  *sync.Mutex
}

type MemPool struct {
	Messages map[string]time.Time
	Mux      *sync.Mutex
}

var disconnect chan net.Conn
var mempool MemPool
var nodes Nodes

func addMessage(message string) bool {
	mempool.Mux.Lock()
	//search for message in mempool
	for i := 0; i < len(mempool.Messages); i++ {
		_, ok := mempool.Messages[message]
		if ok {
			mempool.Mux.Unlock()
			return false
		}
	}
	// delete old messages if mempool is full
	if len(mempool.Messages) == 100 {
		// find the oldest message
		var oldestMessage string
		var oldestTime time.Time
		for message, time := range mempool.Messages {
			if oldestTime.IsZero() || time.Before(oldestTime) {
				oldestTime = time
				oldestMessage = message
			}
		}
		// delete the oldest message
		delete(mempool.Messages, oldestMessage)
	}
	mempool.Messages[message] = time.Now()
	mempool.Mux.Unlock()
	return true
}

// peer discovery
func peerDiscovery() {
	// get the address from seeds.txt
	seed, err := ioutil.ReadFile("seeds.txt")
	if err != nil {
		fmt.Println("Error reading seeds.txt")
	}
	// get first seed node
	seedNode := strings.Split(string(seed), "\n")[0]
	fmt.Println("Seed node: ", seedNode)
	// request for peer list
	// Conn, err := net.Dial("udp", seedNode)
	laddr, err := net.ResolveUDPAddr("udp", ":4041")
	if err != nil {
		fmt.Println("Error resolving UDP address laddr", err)
		return
	}
	ip, port, err := net.SplitHostPort(seedNode)
	portInt, err := strconv.Atoi(port[:len(port)-1])
	if err != nil {
		fmt.Println("Error converting port to int", err)
		return
	}
	if err != nil {
		fmt.Println("Error splitting host and port", err)
		return
	}
	raddr := net.UDPAddr{IP: net.ParseIP(ip), Port: portInt}

	Conn, err := net.DialUDP("udp", laddr, &raddr)

	if err != nil {
		fmt.Println("Error Connecting to seed node", err)
		return
	}

	// send init message
	_, err = Conn.Write([]byte("init\n"))
	if err != nil {
		fmt.Println("Error sending init message")
		return
	}
	// read the list of peers
	buffer := make([]byte, 1024)
	n, err := Conn.Read(buffer)
	if err != nil {
		fmt.Println("Error reading from UDP port")
	}
	peerList := string(buffer[0:n])
	fmt.Println("Peer list: ", peerList)
	peers := strings.Split(peerList, " ")
	// Connect to the peers
	for i := 0; i < len(peers); i++ {
		connectToPeer(peers[i])
	}
}

// connect to a peer
func connectToPeer(peer string) {
	// conn, err := net.Dial("tcp", peer)
	conn, err := net.Dial("tcp", peer)
	if err != nil {
		fmt.Println("Error connecting to peer")
	}
	defer conn.Close()
	// send a message to the peer
	_, err = conn.Write([]byte("Hello from client"))
	if err != nil {
		fmt.Println("Error sending message to peer")
	}
	// listen to the peer
	go handleConnection(conn)
}

// handle the connection
func handleConnection(conn net.Conn) {
	buffer := make([]byte, 1024)
	for {
		// read from the connection
		n, err := conn.Read(buffer)
		if err != nil {
			fmt.Println("Error reading from connection", err)
			break
		}
		message := string(buffer[0:n])
		if !addMessage(message) {
			continue
		}
		// print the message
		fmt.Println("Message: ", message)
		broadcastMessage(message, conn)
	}
	disconnect <- conn
}

func handleDisconnections() {
	for {
		conn := <-disconnect
		fmt.Println("Disconnected: ", conn)
		//delete from nodes
		nodes.Mux.Lock()
		for i := 0; i < len(nodes.Conn); i++ {
			if nodes.Conn[i] == conn {
				nodes.Conn = append(nodes.Conn[:i], nodes.Conn[i+1:]...)
				break
			}
		}
		nodes.Mux.Unlock()
	}

}

// create a tcp server to listen to incoming connections
func tcpServer() {
	// listen to TCP port 4040
	server, err := net.Listen("tcp", ":4041")
	if err != nil {
		fmt.Println("Error listening to TCP port")
	}
	defer server.Close()

	// accept incoming connections
	for {
		conn, err := server.Accept()
		if err != nil {
			fmt.Println("Error accepting connection")
		}
		defer conn.Close()
		nodes.Mux.Lock()
		nodes.Conn = append(nodes.Conn, conn)
		nodes.Mux.Unlock()
		// handle the connection
		go handleConnection(conn)
	}
}

func broadcastMessage(message string, conn net.Conn) {
	nodes.Mux.Lock()
	for i := 0; i < len(nodes.Conn); i++ {
		if nodes.Conn[i] == conn {
			continue
		}
		_, err := nodes.Conn[i].Write([]byte(message))
		if err != nil {
			fmt.Println("Error writing to connection")
		}
	}
	nodes.Mux.Unlock()
}

func main() {
	go peerDiscovery()
	go handleDisconnections()
	tcpServer()
}
