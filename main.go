package main

import (
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
	portName := "/dev/ttyUSB0"
	baudRate := 19200
	mqttBroker := "tcp://10.0.89.54:1883"
	mqttTopic := "hdmi-switch/input"
	mqttClientId := "hdmi-switcher"

	// Configure & open the serial port
	mode := &serial.Mode{
		BaudRate: baudRate,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
		DataBits: 8,
	}

	localPort, err := serial.Open(portName, mode)
	port = localPort
	if err != nil {
		log.Fatalf("Failed to open serial port: %v", err)
	}
	defer port.Close()
	log.Printf("Connected to serial port %s successfully", portName)

	// Setup a HTTP listener for HTTP based control
	http.HandleFunc("/input/{id}", input)
	go func() {
		log.Println("Listening on port 8080...")
		err := http.ListenAndServe(":8080", nil)
		if err != nil {
			panic("ListenAndServe: " + err.Error())
		}
	}()

	// MQTT client options
	opts := mqtt.NewClientOptions().AddBroker(mqttBroker).SetConnectionLostHandler(func(client mqtt.Client, err error) {
		// Custom handler for connection lost - just exit and assume systemd will restart the service
		log.Fatalf("Connection to MQTT broker lost: %v", err)
	})
	opts.SetClientID(mqttClientId)

	// Create and start the MQTT client for MQTT based control
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to connect to MQTT broker: %v", token.Error())
	}
	defer client.Disconnect(250)
	log.Println("Connected to MQTT broker")

	// Subscribe to the HDMI switching topic
	client.Subscribe(mqttTopic, 0, func(client mqtt.Client, msg mqtt.Message) {
		id := string(msg.Payload())
		log.Printf("Received MQTT input: %s", id)
		sendSerialCommand(inputIdToCommand(id))
	})

	// Wait for interrupt signal to gracefully shutdown the application
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	fmt.Println("Shutting down.")
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
