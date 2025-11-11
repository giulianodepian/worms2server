package main

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"net"
	"ws2server/serverhandler"
)

func handleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	for {
		clientData := make([]byte, 1024)
		bytesReaded, err := reader.Read(clientData)
		if err != nil {
			fmt.Println("Error reading data:", err)
			serverhandler.HandleDisconnection(conn.RemoteAddr().String())
			break
		}
		fmt.Println("Bytes Readed:", bytesReaded)
		fmt.Printf("Received message: %s", hex.EncodeToString(clientData))
		serverhandler.HandleClientData(conn, clientData, conn.RemoteAddr().String())
	}
}

func main() {
	listener, err := net.Listen("tcp", "127.0.0.1:17001")
	if err != nil {
		fmt.Println("Failed to start server:", err)
		return
	}
	defer listener.Close()
	fmt.Println("Listening on port 17001")
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Failed Connection:", err)
			continue
		}
		go handleConnection(conn)
	}

}
