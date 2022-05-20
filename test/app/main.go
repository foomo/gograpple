package main

import (
	"flag"
	"log"
	"net/http"
)

func main() {
	var flagAddress string
	var flagGreeting string
	flag.StringVar(&flagAddress, "address", ":80", "address to listen to ")
	flag.StringVar(&flagGreeting, "greeting", "HELLO", "sets the greeting message")
	flag.Parse()

	greeting := []byte(flagGreeting)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(greeting)
	})
	log.Printf("listening on %v", flagAddress)
	log.Fatal(http.ListenAndServe(flagAddress, handler))
}
