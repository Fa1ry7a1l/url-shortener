package main

import "os"

func main() {
	os.Exit(1) // want "direct call to os.Exit in main is prohibited"
}

func helper() {
	os.Exit(1)
}
