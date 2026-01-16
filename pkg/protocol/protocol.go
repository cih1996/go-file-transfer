package protocol

import (
	"encoding/json"
	"fmt"
	"net"
)

// MessageType defines the type of message
type MessageType string

const (
	MsgRegisterReceiver   MessageType = "register_receiver"
	MsgReceiverRegistered MessageType = "receiver_registered"
	MsgConnectSender      MessageType = "connect_sender"
	MsgSenderConnected    MessageType = "sender_connected"
	MsgFileMetadata       MessageType = "file_metadata"
	MsgReadyToReceive     MessageType = "ready_to_receive"
	MsgError              MessageType = "error"
)

// Message is the generic container for control messages
type Message struct {
	Type    MessageType     `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type RegisterReceiverPayload struct{}

type ReceiverRegisteredPayload struct {
	Code string `json:"code"`
}

type ConnectSenderPayload struct {
	Code string `json:"code"`
}

type FileMetadataPayload struct {
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
}

type ErrorPayload struct {
	Message string `json:"message"`
}

// SendMessage sends a JSON message over the connection
func SendMessage(conn net.Conn, msgType MessageType, payload interface{}) error {
	var payloadBytes []byte
	var err error
	if payload != nil {
		payloadBytes, err = json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal payload error: %w", err)
		}
	}

	msg := Message{
		Type:    msgType,
		Payload: payloadBytes,
	}

	return json.NewEncoder(conn).Encode(msg)
}

// ReadMessage reads a JSON message from the connection
func ReadMessage(conn net.Conn) (*Message, error) {
	var msg Message
	decoder := json.NewDecoder(conn)
	if err := decoder.Decode(&msg); err != nil {
		return nil, err
	}
	return &msg, nil
}
