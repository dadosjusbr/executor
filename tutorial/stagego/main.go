package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

func main() {
	url := os.Getenv("URL")
	if url == "" {
		log.Fatal("URL env var can not be empty")
	}
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	err = ioutil.WriteFile("/output/result.json", data, 0644)
	if err != nil {
		log.Fatal(err)
	}
}
