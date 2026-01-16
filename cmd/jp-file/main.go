package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go-file-transfer/pkg/protocol"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"
)

var (
	serverAddr      = flag.String("server", "", "Server address")
	serverAddrShort = flag.String("s", "", "Server address (short)")
	outputDir       = flag.String("d", ".", "Output directory for receiver")
)

type Config struct {
	Server string `json:"server"`
	Code   string `json:"code,omitempty"`
}

func getConfigPath() string {
	// Try to use home directory first
	home, err := os.UserHomeDir()
	if err == nil {
		path := filepath.Join(home, ".jp-file-config.json")
		// Check if we can write to it (or create it)
		f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
		if err == nil {
			f.Close()
			return path
		}
	}
	// Fallback to current directory
	return ".jp-file-config.json"
}

func loadConfig() Config {
	var cfg Config
	data, err := os.ReadFile(getConfigPath())
	if err == nil {
		json.Unmarshal(data, &cfg)
	}
	return cfg
}

func saveConfig(cfg Config) {
	path := getConfigPath()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err == nil {
		os.WriteFile(path, data, 0644)
	}
}

func main() {
	flag.Parse()

	cfg := loadConfig()

	// Determine server address: flag > config > default
	finalServerAddr := "127.0.0.1:9999"
	if *serverAddr != "" {
		finalServerAddr = *serverAddr
	} else if *serverAddrShort != "" {
		finalServerAddr = *serverAddrShort
	} else if cfg.Server != "" {
		finalServerAddr = cfg.Server
	}

	// Update config if changed (and save it)
	if finalServerAddr != cfg.Server {
		cfg.Server = finalServerAddr
		saveConfig(cfg)
	}

	*serverAddr = finalServerAddr

	args := flag.Args()
	if len(args) == 0 {
		fmt.Printf("Using Server: %s\n", *serverAddr)
		runReceiver()
	} else if len(args) == 1 {
		// Try to use stored code if available
		if cfg.Code == "" {
			fmt.Println("Error: No code provided and no stored code found in config.")
			fmt.Println("Usage: jp-file [-server addr] <code> <filepath>")
			os.Exit(1)
		}
		fmt.Printf("Using Server: %s\n", *serverAddr)
		fmt.Printf("Using Stored Code: %s\n", cfg.Code)
		runSender(cfg.Code, args[0], cfg)
	} else if len(args) == 2 {
		fmt.Printf("Using Server: %s\n", *serverAddr)
		runSender(args[0], args[1], cfg)
	} else {
		fmt.Println("Usage:")
		fmt.Println("  Receiver: jp-file [-server addr] [-d output_dir]")
		fmt.Println("  Sender:   jp-file [-server addr] [code] <filepath>")
		os.Exit(1)
	}
}

func runReceiver() {
	// Ensure output directory exists
	if _, err := os.Stat(*outputDir); os.IsNotExist(err) {
		err := os.MkdirAll(*outputDir, 0755)
		if err != nil {
			log.Fatalf("Failed to create output directory: %v", err)
		}
	}

	conn, err := net.Dial("tcp", *serverAddr)
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	// Register
	err = protocol.SendMessage(conn, protocol.MsgRegisterReceiver, protocol.RegisterReceiverPayload{})
	if err != nil {
		log.Fatalf("Failed to send register message: %v", err)
	}

	// Wait for code
	msg, err := protocol.ReadMessage(conn)
	if err != nil {
		log.Fatalf("Failed to read server response: %v", err)
	}

	if msg.Type == protocol.MsgReceiverRegistered {
		var payload protocol.ReceiverRegisteredPayload
		json.Unmarshal(msg.Payload, &payload)
		fmt.Printf("Connected. Your code is: %s\n", payload.Code)
		fmt.Println("Waiting for files... (Press Ctrl+C to exit)")
	} else {
		log.Fatalf("Unexpected message: %s", msg.Type)
	}

	// Loop to receive multiple files
	for {
		// Wait for file metadata
		msg, err = protocol.ReadMessage(conn)
		if err != nil {
			if err == io.EOF {
				fmt.Println("Server closed connection.")
				return
			}
			log.Printf("Failed to read file metadata: %v. Reconnecting...", err)
			return
		}

		if msg.Type != protocol.MsgFileMetadata {
			log.Printf("Expected file metadata, got: %s", msg.Type)
			continue
		}

		var metadata protocol.FileMetadataPayload
		json.Unmarshal(msg.Payload, &metadata)

		fmt.Printf("\nReceiving file: %s (%d bytes)\n", metadata.Filename, metadata.Size)

		// Prepare file
		outputPath := filepath.Join(*outputDir, metadata.Filename)
		file, err := os.Create(outputPath)
		if err != nil {
			log.Printf("Failed to create file: %v", err)
			continue
		}

		// Send Ready
		protocol.SendMessage(conn, protocol.MsgReadyToReceive, nil)

		// Receive data (Exactly metadata.Size bytes)
		pb := &ProgressBar{Total: metadata.Size}

		// Use io.LimitReader to read exactly the file size
		limitReader := io.LimitReader(conn, metadata.Size)

		_, err = io.Copy(io.MultiWriter(file, pb), limitReader)
		if err != nil {
			log.Printf("Error during transfer: %v", err)
		}

		file.Close()
		fmt.Println("\nFile received successfully! Sending confirmation...")

		// Send TransferComplete
		err = protocol.SendMessage(conn, protocol.MsgTransferComplete, nil)
		if err != nil {
			log.Printf("Failed to send confirmation: %v", err)
		}

		fmt.Println("Waiting for next file...")
	}
}

func runSender(code string, filePath string, cfg Config) {
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()

	// Update code in config if changed
	if code != cfg.Code {
		cfg.Code = code
		saveConfig(cfg)
	}

	info, err := file.Stat()
	if err != nil {
		log.Fatalf("Failed to stat file: %v", err)
	}

	conn, err := net.Dial("tcp", *serverAddr)
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	// Connect
	err = protocol.SendMessage(conn, protocol.MsgConnectSender, protocol.ConnectSenderPayload{Code: code})
	if err != nil {
		log.Fatalf("Failed to send connect message: %v", err)
	}

	// Wait for ACK
	msg, err := protocol.ReadMessage(conn)
	if err != nil {
		log.Fatalf("Failed to read server response: %v", err)
	}

	if msg.Type == protocol.MsgError {
		var payload protocol.ErrorPayload
		json.Unmarshal(msg.Payload, &payload)
		log.Fatalf("Server error: %s", payload.Message)
	}

	if msg.Type != protocol.MsgSenderConnected {
		log.Fatalf("Unexpected message: %s", msg.Type)
	}

	fmt.Println("Connected to receiver. Sending metadata...")

	// Send Metadata
	err = protocol.SendMessage(conn, protocol.MsgFileMetadata, protocol.FileMetadataPayload{
		Filename: filepath.Base(filePath),
		Size:     info.Size(),
	})
	if err != nil {
		log.Fatalf("Failed to send metadata: %v", err)
	}

	// Wait for Ready
	msg, err = protocol.ReadMessage(conn)
	if err != nil {
		log.Fatalf("Failed to read ready signal: %v", err)
	}
	if msg.Type != protocol.MsgReadyToReceive {
		log.Fatalf("Expected ready signal, got: %s", msg.Type)
	}

	fmt.Println("Receiver ready. Sending data...")
	pb := &ProgressBar{Total: info.Size()}

	// Send data
	_, err = io.Copy(io.MultiWriter(conn, pb), file)
	if err != nil {
		log.Fatalf("Failed to send data: %v", err)
	}

	fmt.Println("\nFile data sent. Waiting for receiver confirmation...")

	// Wait for TransferComplete
	msg, err = protocol.ReadMessage(conn)
	if err != nil {
		log.Fatalf("Failed to read confirmation: %v", err)
	}
	if msg.Type != protocol.MsgTransferComplete {
		if msg.Type == protocol.MsgError {
			var payload protocol.ErrorPayload
			json.Unmarshal(msg.Payload, &payload)
			log.Fatalf("Transfer failed: %s", payload.Message)
		}
		log.Fatalf("Unexpected confirmation message: %s", msg.Type)
	}

	fmt.Println("Transfer confirmed by receiver!")
}

// Simple Progress Bar
type ProgressBar struct {
	Total      int64
	Current    int64
	lastUpdate time.Time
}

func (pb *ProgressBar) Write(p []byte) (n int, err error) {
	n = len(p)
	pb.Current += int64(n)
	// Update every 100ms to avoid console spam
	if time.Since(pb.lastUpdate) > 100*time.Millisecond || pb.Current == pb.Total {
		pb.Print()
		pb.lastUpdate = time.Now()
	}
	return
}

func (pb *ProgressBar) Print() {
	percent := float64(pb.Current) / float64(pb.Total) * 100
	fmt.Printf("\rProgress: %.2f%% (%d/%d bytes)", percent, pb.Current, pb.Total)
}
