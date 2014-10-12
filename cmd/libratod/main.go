package main

import (
	"fmt"
	"log"

	"github.com/gorsuch/sqs"
)

func main() {
	for {
		log.Print("polling")
		messages, err := sqs.Get("https://sqs.us-east-1.amazonaws.com/854436987475/to-librato", "10")
		if err != nil {
			panic(err)
		}

		for _, m := range messages {
			log.Printf("received=%s\n", m.MessageId)
			log.Printf("deleting=%s\n", m.MessageId)
			err := sqs.Delete("https://sqs.us-east-1.amazonaws.com/854436987475/to-librato", m.ReceiptHandle)
			if err != nil {
				panic(err)
			}
			fmt.Printf("deleted=%s\n", m.MessageId)
		}
	}
}
