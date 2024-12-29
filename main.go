package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"go.bug.st/serial"
)

func main() {
	portName := "/dev/ttyUSB0"
	baudRate := 19200
	mqttBroker := "tcp://10.0.89.54:1883"
	mqttTopic := "hdmi-switch/input"

	// Configure & open the serial port
	mode := &serial.Mode{
		BaudRate: baudRate,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
		DataBits: 8,
	}

	port, err := serial.Open(portName, mode)
	if err != nil {
		log.Fatalf("Failed to open serial port: %v", err)
	}
	defer port.Close()
	log.Printf("Connected to serial port %s successfully.", portName)

	// MQTT client options
	opts := mqtt.NewClientOptions().AddBroker(mqttBroker)
	opts.SetClientID("go-hdmi-client")

	// Create and start the MQTT client
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to connect to MQTT broker: %v", token.Error())
	}
	defer client.Disconnect(250)
	log.Println("Connected to MQTT broker.")

	// Subscribe to the HDMI switching topic
	client.Subscribe(mqttTopic, 0, func(client mqtt.Client, msg mqtt.Message) {
		input := string(msg.Payload())
		var command string

		log.Printf("Received input: %s", input)

		switch input {
		case "1":
			command = "sw i01\r\n"
		case "2":
			command = "sw i02\r\n"
		case "3":
			command = "sw i03\r\n"
		case "4":
			command = "sw i04\r\n"
		default:
			log.Printf("Invalid input received: %s", input)
			return
		}

		// Send switch command to the HDMI switcher over serial
		_, err := port.Write([]byte(command))
		if err != nil {
			log.Printf("Failed to send command: %v", err)
		} else {
			log.Printf("Command %q sent successfully to %s.\n", command, portName)
		}
	})

	// Wait for interrupt signal to gracefully shutdown the application
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	fmt.Println("Shutting down...")
}
