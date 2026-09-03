package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"go.bug.st/serial"
)

const responseTimeout = 500 * time.Millisecond

var (
	port      serial.Port
	portMutex sync.Mutex
)

func main() {

	// Parse command line arguments
	serialPortPath := flag.String("serial-path", "/dev/ttyUSB0", "Path to the serial port")
	mqttBroker := flag.String("mqtt-broker", "tcp://10.0.89.54:1883", "Connection string for the MQTT broker, including protocol and port")
	mqttTopic := flag.String("mqtt-topic", "hdmi-switch/input", "MQTT topic to subscribe to for HDMI input switching")
	mqttClientId := flag.String("mqtt-client-id", "hdmi-switcher", "MQTT client id")
	httpPort := flag.String("http-port", "8080", "Port for the HTTP server to listen on")
	flag.Parse()

	// Configure & open the serial port
	mode := &serial.Mode{
		BaudRate: 19200,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
		DataBits: 8,
	}

	var err error
	port, err = serial.Open(*serialPortPath, mode)
	if err != nil {
		log.Fatalf("Failed to open serial port: %v", err)
	}
	defer port.Close()
	if err := port.SetReadTimeout(responseTimeout); err != nil {
		log.Fatalf("Failed to set serial read timeout: %v", err)
	}
	log.Printf("Connected to serial port %s successfully", *serialPortPath)

	// Setup a HTTP listener for HTTP based control
	http.HandleFunc("/input/{id}", input)
	go func() {
		log.Printf("Starting HTTP server on port %s\n", *httpPort)
		err := http.ListenAndServe(fmt.Sprintf(":%s", *httpPort), nil)
		if err != nil {
			panic("ListenAndServe: " + err.Error())
		}
	}()

	// MQTT client options
	opts := mqtt.NewClientOptions().AddBroker(*mqttBroker).SetConnectionLostHandler(func(client mqtt.Client, err error) {
		// Custom handler for connection lost - just exit and assume systemd will restart the service
		log.Fatalf("Connection to MQTT broker lost: %v", err)
	})
	opts.SetClientID(*mqttClientId)

	// Create and start the MQTT client for MQTT based control
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to connect to MQTT broker: %v", token.Error())
	}
	defer client.Disconnect(250)
	log.Println("Connected to MQTT broker")

	// Subscribe to the HDMI switching topic
	client.Subscribe(*mqttTopic, 0, func(client mqtt.Client, msg mqtt.Message) {
		id := string(msg.Payload())
		log.Printf("Received MQTT input: %s", id)
		command, ok := inputIdToCommand(id)
		if !ok {
			return
		}
		sendSerialCommand(command)
	})

	// Wait for interrupt signal to gracefully shutdown the application
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Println("Shutting down")
}

// Send a command to the HDMI switcher over serial and wait for its acknowledgement
func sendSerialCommand(command string) error {
	portMutex.Lock()
	defer portMutex.Unlock()

	if err := port.ResetInputBuffer(); err != nil {
		log.Printf("Failed to reset serial input buffer: %v\n", err)
	}

	if _, err := port.Write([]byte(command)); err != nil {
		log.Printf("Failed to send command %q: %v\n", command, err)
		return err
	}

	response, err := readResponseLine()
	if err != nil {
		log.Printf("Command %q sent but %v\n", command, err)
		return err
	}
	if !strings.HasSuffix(response, "Command OK") {
		err := fmt.Errorf("switch rejected command: %q", response)
		log.Printf("Command %q sent but %v\n", command, err)
		return err
	}

	log.Printf("Command %q acknowledged: %q\n", command, response)
	return nil
}

// Read from the serial port until a line terminator arrives or the read timeout elapses
func readResponseLine() (string, error) {
	var response []byte
	buffer := make([]byte, 64)
	for {
		n, err := port.Read(buffer)
		if err != nil {
			return "", fmt.Errorf("failed reading response: %w", err)
		}
		if n == 0 {
			if len(response) == 0 {
				return "", errors.New("no response from switch within " + responseTimeout.String())
			}
			return "", fmt.Errorf("incomplete response from switch: %q", response)
		}
		response = append(response, buffer[:n]...)
		if bytes.HasSuffix(response, []byte("\r\n")) {
			return strings.TrimSpace(string(response)), nil
		}
	}
}

// HTTP handler for input switching
func input(w http.ResponseWriter, req *http.Request) {
	id := req.PathValue("id")
	log.Printf("Received input via http: %s\n", id)

	command, ok := inputIdToCommand(id)
	if !ok {
		http.Error(w, fmt.Sprintf("Invalid input: %s", id), http.StatusBadRequest)
		return
	}

	if err := sendSerialCommand(command); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	fmt.Fprintf(w, "OK\n")
}

// Map input ID to HDMI switcher command
func inputIdToCommand(input string) (string, bool) {
	switch input {
	case "1":
		return "sw i01\r\n", true
	case "2":
		return "sw i02\r\n", true
	case "3":
		return "sw i03\r\n", true
	case "4":
		return "sw i04\r\n", true
	default:
		log.Printf("Invalid input received: %s", input)
		return "", false
	}
}
