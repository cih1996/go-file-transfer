package main

import (
	"encoding/json"
	"fmt"
	"go-file-transfer/pkg/protocol"
	"io"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"
)

// ReceiverSession holds the connection and synchronization primitives for a receiver
type ReceiverSession struct {
	Conn      net.Conn
	Code      string
	OpLock    sync.Mutex      // Ensures only one sender can transfer at a time
	ReadyChan chan bool       // Notifies sender handler that receiver is ready for data
	CloseChan chan struct{}   // Notifies cleanup
}

var (
	receivers = make(map[string]*ReceiverSession)
	mu        sync.Mutex
)

func main() {
	rand.Seed(time.Now().UnixNano())
	listener, err := net.Listen("tcp", ":9999")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Server started on :9999")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Accept error:", err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	// Read initial handshake
	msg, err := protocol.ReadMessage(conn)
	if err != nil {
		log.Println("Read handshake error:", err)
		conn.Close()
		return
	}

	switch msg.Type {
	case protocol.MsgRegisterReceiver:
		handleReceiver(conn)
	case protocol.MsgConnectSender:
		var payload protocol.ConnectSenderPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			protocol.SendMessage(conn, protocol.MsgError, protocol.ErrorPayload{Message: "Invalid payload"})
			conn.Close()
			return
		}
		handleSender(conn, payload.Code)
	default:
		protocol.SendMessage(conn, protocol.MsgError, protocol.ErrorPayload{Message: "Unknown message type"})
		conn.Close()
	}
}

func handleReceiver(conn net.Conn) {
	code := generateCode()
	session := &ReceiverSession{
		Conn:      conn,
		Code:      code,
		ReadyChan: make(chan bool),
		CloseChan: make(chan struct{}),
	}

	mu.Lock()
	receivers[code] = session
	mu.Unlock()

	log.Printf("Receiver registered with code: %s", code)
	err := protocol.SendMessage(conn, protocol.MsgReceiverRegistered, protocol.ReceiverRegisteredPayload{Code: code})
	if err != nil {
		log.Println("Error sending code to receiver:", err)
		removeReceiver(code)
		conn.Close()
		return
	}

	// Keep connection open and listen for control messages from receiver
	defer func() {
		removeReceiver(code)
		conn.Close()
		close(session.CloseChan)
	}()

	for {
		msg, err := protocol.ReadMessage(conn)
		if err != nil {
			if err != io.EOF {
				log.Printf("Receiver %s connection error: %v", code, err)
			}
			return
		}

		switch msg.Type {
		case protocol.MsgReadyToReceive:
			// Signal that receiver is ready for data
			select {
			case session.ReadyChan <- true:
			case <-time.After(5 * time.Second):
				log.Printf("Receiver %s timed out sending ready signal", code)
			}
		default:
			log.Printf("Unexpected message from receiver %s: %s", code, msg.Type)
		}
	}
}

func removeReceiver(code string) {
	mu.Lock()
	delete(receivers, code)
	mu.Unlock()
}

func handleSender(senderConn net.Conn, code string) {
	defer senderConn.Close()

	mu.Lock()
	session, ok := receivers[code]
	mu.Unlock()

	if !ok {
		log.Printf("Sender tried invalid code: %s", code)
		protocol.SendMessage(senderConn, protocol.MsgError, protocol.ErrorPayload{Message: "Invalid or expired code"})
		return
	}

	// Lock the session to ensure only one sender at a time
	session.OpLock.Lock()
	defer session.OpLock.Unlock()

	// Check if session is still alive
	select {
	case <-session.CloseChan:
		protocol.SendMessage(senderConn, protocol.MsgError, protocol.ErrorPayload{Message: "Receiver disconnected"})
		return
	default:
	}

	log.Printf("Sender connected to receiver %s", code)
	err := protocol.SendMessage(senderConn, protocol.MsgSenderConnected, nil)
	if err != nil {
		log.Println("Error sending ack to sender:", err)
		return
	}

	// Read Metadata from Sender
	msg, err := protocol.ReadMessage(senderConn)
	if err != nil {
		log.Println("Error reading metadata from sender:", err)
		return
	}
	if msg.Type != protocol.MsgFileMetadata {
		log.Println("Expected metadata from sender, got:", msg.Type)
		return
	}

	// Forward Metadata to Receiver
	err = protocol.SendMessage(session.Conn, protocol.MsgFileMetadata, msg.Payload) // Forward raw payload
	if err != nil {
		log.Println("Error sending metadata to receiver:", err)
		return
	}

	// Wait for Receiver to be ready
	log.Println("Waiting for receiver to be ready...")
	select {
	case <-session.ReadyChan:
		log.Println("Receiver is ready")
	case <-time.After(10 * time.Second):
		log.Println("Timeout waiting for receiver ready")
		protocol.SendMessage(senderConn, protocol.MsgError, protocol.ErrorPayload{Message: "Receiver timeout"})
		return
	case <-session.CloseChan:
		log.Println("Receiver disconnected during handshake")
		return
	}

	// Tell Sender to start sending data
	err = protocol.SendMessage(senderConn, protocol.MsgReadyToReceive, nil)
	if err != nil {
		log.Println("Error sending ready signal to sender:", err)
		return
	}

	// Proxy Data
	log.Printf("Transferring data for code %s...", code)
	_, err = io.Copy(session.Conn, senderConn)
	if err != nil {
		log.Println("Error during data transfer:", err)
	}
	
	log.Printf("Transfer finished for code %s", code)
	// Note: We do NOT close session.Conn here.
}

func generateCode() string {
	// Simple 4 digit code
	return fmt.Sprintf("%04d", rand.Intn(10000))
}
