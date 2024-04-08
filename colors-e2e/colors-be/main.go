package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"
)

func main() {
	color := "red" // default color
	overrideColor := os.Getenv("OverrideColor")
	hostname := os.Getenv("HOSTNAME")
	faultPercentString := os.Getenv("FaultPercent")
	faultPercent := 0
	if faultPercentString != "" {
		parsedPercent, err := strconv.ParseInt(faultPercentString, 10, 32)
		if err != nil {
			log.Fatalf("cannot parse fault perecent %v: %v", faultPercent, err)
		}
		faultPercent = int(parsedPercent)
	}
	if overrideColor != "" {
		color = overrideColor
	}
	scvData, err := GetSerivceMetadata()
	if err != nil {
		log.Fatalf("cannot get service metadata: %v", err)
	}

	requestLogger, err := NewRequestLogger(context.Background(), scvData)
	if err != nil {
		log.Fatalf("cannot setup request logger")
	}

	createConstantLoad(context.Background(), "http://colors-be-scv:8080/color", 1)
	http.HandleFunc("/color", func(w http.ResponseWriter, r *http.Request) {
		var responseStatusGood bool = true
		result := struct {
			Color string `json:"color"`
			Name  string `json:"name"`
		}{
			Color: color,
			Name:  hostname,
		}
		if rand.Intn(101) < faultPercent {
			responseStatusGood = false
			w.WriteHeader(500)
		} else {
			err := json.NewEncoder(w).Encode(result)
			if err != nil {
				log.Printf("error encoding response: %v\n", err)
				w.WriteHeader(500)
				responseStatusGood = false
			}
		}

		requestLogger.LogRequest(r.Context(), responseStatusGood)
	})

	// Listen on port 8080.
	http.ListenAndServe(":8080", nil)
}

// createConstantLoad creates constant load against the endpoint forever
func createConstantLoad(ctx context.Context, url string, qps int) {
	log.Printf("creating constant load against %v with QPS %v", url, qps)
	delay := 1000 / qps
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				sendRequest(url)
				time.Sleep(time.Duration(delay) * time.Millisecond)
			}
		}
	}()
}

// sendRequest sends a get request to the endpoint and ignores the response
func sendRequest(endpoint string) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		log.Printf("error creating request: %v", err)
		return
	}
	req.Close = true
	response, err := client.Do(req)
	if err != nil {
		return
	}
	defer response.Body.Close()
	io.Copy(io.Discard, response.Body)
}
