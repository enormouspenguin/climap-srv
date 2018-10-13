package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {
	var listenSocket string
	if len(os.Args) > 1 {
		listenSocket = os.Args[1]
	} else {
		listenSocket = "localhost:9999"
	}

	fmt.Println("listening to ", listenSocket, "...")
	fmt.Println(http.ListenAndServe(listenSocket, rtr))
}
