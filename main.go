package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"go.bug.st/serial"
)

var port serial.Port

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

	localPort, err := serial.Open(*serialPortPath, mode)
	port = localPort
	if err != nil {
		log.Fatalf("Failed to open serial port: %v", err)
	}
	defer port.Close()
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
		sendSerialCommand(inputIdToCommand(id))
	})

	// Wait for interrupt signal to gracefully shutdown the application
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Println("Shutting down")
}

// Send switch command to the HDMI switcher over serial
func sendSerialCommand(command string) {
	if command == "" {
		log.Printf("Skipping empty command\n")
		return
	}
	_, err := port.Write([]byte(command))
	if err != nil {
		log.Printf("Failed to send command: %v\n", err)
	} else {
		log.Printf("Command %q sent successfully\n", command)
	}
}

// HTTP handler for input switching
func input(w http.ResponseWriter, req *http.Request) {
	id := req.PathValue("id")
	log.Printf("Received input via http: %s\n", id)

	// Send the switch command
	sendSerialCommand(inputIdToCommand(id))
	fmt.Fprintf(w, "OK\n")
}

// Map input ID to HDMI switcher command
func inputIdToCommand(input string) string {
	switch input {
	case "1":
		return "sw i01\r\n"
	case "2":
		return "sw i02\r\n"
	case "3":
		return "sw i03\r\n"
	case "4":
		return "sw i04\r\n"
	default:
		log.Printf("Invalid input received: %s", input)
		return ""
	}
}
