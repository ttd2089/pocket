package main

import "log"

func main() {
	err := WatchDir(".", func(_ WatcherEvent) {})
	if err != nil {
		log.Fatal(err)
	}
}
