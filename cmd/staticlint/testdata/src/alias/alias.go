package main

import process "os"

func main() {
	process.Exit(1) // want "direct call to os.Exit in main is prohibited"
}
