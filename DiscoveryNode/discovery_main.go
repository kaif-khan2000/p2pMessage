package main

import (
	"fmt"
	"math/rand"
	"net"
	"time"
)

// struct to store the details of a node
type Node struct {
	Addr         string
	Availability bool
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
					list = list + NodeList[i].String() + " "
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

func main() {
	NodeList := make([]Node, 0)
	fmt.Println("Starting discovery node...")
	udpServer(NodeList)
}
