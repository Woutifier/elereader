package main

import (
	"context"
	"github.com/Woutifier/elereader/parser"
	"flag"
	"log"
	"strings"
	"time"

	"github.com/jacobsa/go-serial/serial"

	"cloud.google.com/go/pubsub"
	"github.com/golang/protobuf/proto"
)

var (
	serialPort    = flag.String("serialport", "/dev/tty0", "Serial port to listen for telegrams on")
	baudRate      = flag.Int("baudrate", 115200, "Baudrate")
	googleProject = flag.String("googleProject", "", "Google pub/sub project to publish to")
	googleTopic   = flag.String("googleTopic", "", "Google topic to publish to")
)

func main() {
	// Parse commandline options
	flag.Parse()

	log.Printf("Starting elereader")

	// Setup serial connection
	options := serial.OpenOptions{
		PortName:        *serialPort,
		BaudRate:        uint(*baudRate),
		DataBits:        8,
		StopBits:        1,
		MinimumReadSize: 4,
	}

	// Open the port.
	port, err := serial.Open(options)
	if err != nil {
		log.Fatalf("serial.Open: %v", err)
	}

	// Make sure to close it later.
	defer port.Close()

	// Create google pub/sub client.
	client, err := pubsub.NewClient(context.Background(), *googleProject)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	topic := client.Topic(*googleTopic)

	bufferChannel := make(chan string)
	// 1 telegram per 10 seconds, buffer for 1 hour
	telegramChannel := make(chan parser.Telegram, 360)
	go func() {
		stringBuffer := ""
		buffer := make([]byte, 2048)
		for {
			_, err = port.Read(buffer)
			if err != nil {
				log.Fatalf("Failed to read from serial port: %s", err)
			}
			bufferStr := string(buffer)
			stringBuffer = stringBuffer + bufferStr
			if strings.ContainsRune(bufferStr, '!') {
				bufferChannel <- bufferStr
				stringBuffer = ""
			}
		}
	}()

	go func() {
		for {
			telegram, err := parser.ParseTelegram(<-bufferChannel)
			if err != nil {
				log.Printf("Failed to parse telegram: %s", err)
				continue
			}

			telegramChannel <- *telegram
		}
	}()

	go func() {
		for {
			telegram := <-telegramChannel
			log.Printf("Got telegram: %v", telegram)

			// Convert telegram to reading and marshal to bytes
			reading := telegram.GetReading()
			readingBytes, err := proto.Marshal(&reading)
			if err != nil {
				log.Fatalf("Could not marshal Reading to bytes: %s", err)
			}

			// Send to pub/sub channel
			for {
				serverId, err := topic.Publish(context.Background(), &pubsub.Message{Data: readingBytes}).Get(context.Background())
				if err != nil {
					log.Printf("Could not publish telegram, retrying in 5 seconds: %s", err)
					time.Sleep(time.Second * 5)
					continue
				}
				log.Printf("Telegram published to server %s.", serverId)
				break
			}
		}
	}()

	select {}
}
