package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/klauspost/compress/zstd"
	"github.com/segmentio/kafka-go"
)

func main() {
	topic := "bid.requests"
	if len(os.Args) > 1 {
		topic = os.Args[1]
	}

	fmt.Printf("Listening to topic: %s (Broker: localhost:9092)\n", topic)

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{"localhost:9092"},
		Topic:     topic,
		Partition: 0,
		MaxBytes:  10e6, // 10MB
	})
	defer r.Close()

	// ZSTD Decoder
	decoder, _ := zstd.NewReader(nil)
	defer decoder.Close()

	var filter string
	if len(os.Args) > 2 {
		filter = os.Args[2]
		fmt.Printf("Filtering for: '%s'\n", filter)
	}

	for {
		m, err := r.ReadMessage(context.Background())
		if err != nil {
			break
		}

		// Decompress
		decompressed, err := decoder.DecodeAll(m.Value, nil)
		if err != nil {
			// Try raw if decompression fails (maybe uncompressed?)
			decompressed = m.Value
		}

		// Apply filter if set
		if filter != "" {
			if !strings.Contains(string(decompressed), filter) && !strings.Contains(string(m.Key), filter) {
				continue
			}
		}

		// Pretty Print JSON
		var obj interface{}
		if err := json.Unmarshal(decompressed, &obj); err != nil {
			fmt.Printf("[%s] Raw: %s\n", string(m.Key), string(decompressed))
		} else {
			pretty, _ := json.MarshalIndent(obj, "", "  ")
			fmt.Printf("[%s] JSON:\n%s\n", string(m.Key), string(pretty))
		}
	}
}
