package client

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/sebasthuis/lsm-database/database"
)

func Run() {
	printInstructions()

	scanner := bufio.NewScanner(os.Stdin)
	db, err := database.Create("data")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing database: %v\n", err)
		return
	}

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		query := parseQuery(scanner.Text())
		if query == nil {
			continue
		}

		switch query.Command {
		case "PUT":
			handlePut(db, query.Args)
		case "GET":
			handleGet(db, query.Args)
		case "DELETE":
			handleDelete(db, query.Args)
		case "EXIT", "QUIT":
			handleExit()
			return
		default:
			handleInvalidCommand()
		}
	}

	if err := scanner.Err(); err != nil {
		handleError(err)
	}
}

func printInstructions() {
	fmt.Println("LSM Database CLI")
	fmt.Println("Commands: PUT <key> <value>, GET <key>, DELETE <key>, EXIT")
	fmt.Println()
}

func handleError(err error) {
	fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
}

func handlePut(db *database.Database, args []string) {
	if len(args) != 2 {
		fmt.Println("Usage: PUT <key> <value>")
		return
	}
	if err := db.Put(args[0], args[1]); err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Println("OK")
	}
}

func handleGet(db *database.Database, args []string) {
	if len(args) != 1 {
		fmt.Println("Usage: GET <key>")
		return
	}
	if value, err := db.Get(args[0]); err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(value)
	}
}

func handleDelete(db *database.Database, args []string) {
	if len(args) != 1 {
		fmt.Println("Usage: DELETE <key>")
		return
	}
	if err := db.Delete(args[0]); err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Println("OK")
	}
}

func handleExit() {
	fmt.Println("quitting")
}

func handleInvalidCommand() {
	fmt.Println("Invalid command. Available commands: PUT, GET, DELETE, EXIT")
}

type Query struct {
	Command string
	Args    []string
}

func parseQuery(input string) *Query {
	parts := strings.Fields(strings.TrimSpace(input))
	if len(parts) == 0 {
		return nil
	}
	return &Query{
		Command: strings.ToUpper(parts[0]),
		Args:    parts[1:],
	}
}
