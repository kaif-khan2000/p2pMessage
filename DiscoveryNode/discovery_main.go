package main

import (
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"
)

// struct to store the details of a node
type Node struct {
	Addr         string
	Availability bool
}

// struct to store the details of a node
type Nodes struct {
	Conn []net.Conn
	Mux  *sync.Mutex
}

var NodeMap map[string]bool

// function to convert a node to a string
func (n Node) String() string {
	return n.Addr
}

// a program to work as a discovery node
func udpServer(NodeList []Node) {
	noOfNodes := 5
	// listen to UDP port 4040
	server, err := net.ListenPacket("udp", ":4040")
	if err != nil {
		fmt.Println("Error listening to UDP port")
	}
	defer server.Close()

	// read from UDP port continuously
	for {
		buffer := make([]byte, 1024)
		n, addr, err := server.ReadFrom(buffer)
		if err != nil {
			fmt.Println("Error reading from UDP port")
		}
		if string(buffer[0:n-1]) == "init" {
			// add the node to the list if its not already present
			present := false
			for i := 0; i < len(NodeList); i++ {
				if NodeList[i].Addr == addr.String() {
					present = true
					break
				}
			}
			if !present {
				fmt.Println("Adding node: ", addr)
				NodeList = append(NodeList, Node{addr.String(), true})
			}

			list := ""
			// randomly select 5 available nodes
			rand := rand.New(rand.NewSource(time.Now().Unix()))
			checkList := make(map[int]bool)
			if len(NodeList) < noOfNodes {
				for i := 0; i < len(NodeList); i++ {
					if NodeList[i].String() != addr.String() {
						list = list + NodeList[i].String() + " "
					}
				}
				list = list + "13.49.75.242:4041"
			} else {
				for i := 0; i < noOfNodes; i++ {
					index := rand.Intn(len(NodeList))
					fmt.Println("Index: ", index)
					if checkList[index] {
						i = i - 1
						continue
					}
					if NodeList[index].Availability && NodeList[index].Addr != addr.String() {
						list = list + NodeList[index].String() + " "
						checkList[index] = true
					} else {
						i -= 1
					}
				}
			}
			// send the list of nodes to the client
			_, err = server.WriteTo([]byte(list), addr)
			if err != nil {
				fmt.Println("Error writing to UDP port")
			}
		}
		fmt.Println("Received ", string(buffer[0:n-1]), " from: ", addr)
		fmt.Println("Node List: ", NodeList)
	}

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
			break
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
	NodeList := make([]Node, 0)
	fmt.Println("Starting discovery node...")
	go udpServer(NodeList)
	go handleDisconnections()
	tcpServer()
}
