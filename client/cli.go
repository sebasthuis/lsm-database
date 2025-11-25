package client

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func Run() {
	fmt.Println("LSM Database CLI")
	fmt.Println("Commands: PUT <key> <value>, GET <key>, DELETE <key>, EXIT")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		command := strings.ToUpper(strings.TrimSpace(scanner.Text()))

		switch command {
		case "PUT":
			fmt.Println("put")
		case "GET":
			fmt.Println("get")
		case "DELETE":
			fmt.Println("delete")
		case "EXIT", "QUIT":
			fmt.Println("quitting")
			return
		default:
			fmt.Println("Unknown command. Available: PUT, GET, DELETE, EXIT")
		}
	}

	if error := scanner.Err(); error != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n,", error)
	}
}