package main

import (
	"fmt"
	"os"

	"github.com/sojebsikder/tunnel/cmd"
)

var version = "0.0.1"
var appName = "tunnel"

func showUsage() {
	fmt.Printf("Usage:\n")
	fmt.Printf("  %s start-server\n\n", appName)
	fmt.Printf("  %s tunnel [--url] [--host]\n\n", appName)
	fmt.Printf("  %s help\n", appName)
	fmt.Printf("  %s version\n", appName)
}

func main() {
	if len(os.Args) < 2 {
		showUsage()
		return
	}

	cmdName := os.Args[1]

	switch cmdName {
	case "start-server":
		cmd.StartServer(os.Args[2:])

	case "tunnel":
		cmd.StartClient(os.Args[2:])

	case "help":
		showUsage()
	case "version":
		fmt.Printf("%s %s \n", version, appName)
	default:
		fmt.Println("Unknown command:", cmdName)
		fmt.Printf("Use '%s help' to see available commands.\n", version)
		os.Exit(1)
	}
}
