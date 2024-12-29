# ATEN HDMI switcher "smart" bridge

A small application for controlling ATEN HDMI switches via HTTP and MQTT.

So you bought a ATEN HDMI switcher which has RS232 control because you want to turn it into something "smart" as part of your home automation... Congrats! I did the same!

So here's what I built, let me know if it's useful to you!

## Dependencies
* Linux (Tested on Debian, other distributions will very likely work, good luck with any other OS)
* Go >= 1.22 ([This application uses the new path params routing functions](https://www.willem.dev/articles/url-path-parameters-in-routes/))

## Hardware Required
* [ATEN VS481C](https://www.aten.com/gb/en/products/professional-audiovideo/video-switches/vs481c/) ([Manual](https://assets.aten.com/product/manual/vs481c_um_w_2021-06-10.pdf)) - Other ATEN switches seem to have the same interface, but YMMV
* A generic Linux box to run this application on - I use it on a [Raspberry Pi Zero 2 W](https://www.raspberrypi.com/products/raspberry-pi-zero-2-w/) without issues
* A USB to Serial convertor - [I'm using this generic one from UGREEN](https://www.amazon.co.uk/dp/B00QUZY4UG) without issues

## Use
1) Work out what [configuration](#Configuration) you need to work in your setup, specifically you will want to customise the broker address, topic, and serial port path for your specific setup
2) Run it... `go run main.go`
3) If it works for you, compile it (`go build main.go`), and keep the output binary running in whatever toolchain you use (I use a `systemd` unit, there's an [example unit file here](hdmi-switcher.service))

## Configuration

The application uses the golang `flag` library for parsing configuration, so you can [a variety of formats](https://pkg.go.dev/flag#hdr-Command_line_flag_syntax), for example to customise the serial path and http port:

```
go run main.go --serial-path /dev/ttyUSB0 --http-port 8989
```

### Full configuration options

| Configuration | Description | Default Value |
|-----|-----|-----|
| `serial-path` | Path to the serial port | `/dev/ttyUSB0` |
| `mqtt-broker` | Address of the MQTT broker | `tcp://localhost:1883` |
| `mqtt-topic` | MQTT topic to listen for input changes | `hdmi-switch/input` |
| `mqtt-client-id` | Client ID for the MQTT connection | `hdmi-switcher` |
| `http-port` | Port for the HTTP server to listen on | `8080` |

The default configuration is unlikely to work for your personal setup, you'll very likely want to use something custom.

## API

### HTTP

The service exposes a simple HTTP API for controlling the input:

```
/input/{id}
```

For example, the following `curl` command will switch to input 1: 
```
curl localhost:8080/input/1
```

The HTTP verb used does not matter.

### MQTT

The service listens for messages on the specified topic, simply containing the requested input ID.

For example, the following `mosquitto_pub` command sent to the same MQTT broker that the application is connected to, will switch to input 1:

```
mosquitto_pub -h 10.0.89.54 -t "hdmi-switch/input" -m "1"
```

## Use in Home Assistant

Unsurprisingly, I built this so I could control the HDMI switcher from [Home Assistant](https://www.home-assistant.io/). To integrate to Home Assistant, you can use the MQTT broker, and configure it something like this:

```
mqtt:
  - number:
      unique_id: "hdmi-switcher"
      name: "HDMI Input"
      mode: "slider"
      command_topic: hdmi-switch/input
      min: 1
      max: 4
```

Then you can add that entity to a dashboard, and it'll look something like this:

![Screenshot of HDMI input in Home Assistant](ha-screenshot.png)

## Resilience model
* The application deliberately crashes if the MQTT broker goes down - my deployment relies on `systemd` to restart it appropriately

## TODO: 
* Make configuration command line arguments
