# ATEN HDMI switcher "smart" bridge

A small application for controlling ATEN HDMI switches via HTTP and MQTT

So you bought a ATEN HDMI switcher which has RS232 because you want to turn it into something "smart" as part of your home automation... Congrats! I did the same!

So here's what I built.

## Dependencies
* Linux (Tested on Debian, other distributions will very likely work, good luck with any other OS)
* Go >= 1.22 ([This application uses the new path params routing functions](https://www.willem.dev/articles/url-path-parameters-in-routes/))

## Use
1) Modify the configuration in `main.go`, specifically you will want to customise the broker address, topic, and serial port path for your specific setup
2) Run it... `go run main.go`
3) Keep it running in whatever toolchain you use (I use a `systemd` unit)

## Hardware Required
* [ATEN VS481C](https://www.aten.com/gb/en/products/professional-audiovideo/video-switches/vs481c/) ([Manual](https://assets.aten.com/product/manual/vs481c_um_w_2021-06-10.pdf)) - Other ATEN switches seem to have the same interface, but YMMV
* A generic Linux box to run this application on - I use it on a [Raspberry Pi Zero 2 W](https://www.raspberrypi.com/products/raspberry-pi-zero-2-w/) without issues
* A USB to Serial convertor - [I'm using this generic one from UGREEN](https://www.amazon.co.uk/dp/B00QUZY4UG) without issues

## Resilience model
* The application deliberately crashes if the MQTT broker goes down - it relies on systemd to restart it appropriately

## TODO: 
* Publish the `systemd` unit I use with this application
