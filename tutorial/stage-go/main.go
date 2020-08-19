package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/dadosjusbr/executor/status"
)

func main() {
	url := os.Getenv("URL")
	if url == "" {
		status.ExitFromError(status.NewError(status.InvalidParameters, fmt.Errorf("URL env var can not be empty")))
	}
	output := os.Getenv("OUTPUT_FOLDER")
	if output == "" {
		status.ExitFromError(status.NewError(status.InvalidParameters, fmt.Errorf("OUTPUT_FOLDER env var can not be empty")))
	}

	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(status.DataUnavailable)
		status.ExitFromError(status.NewError(status.DataUnavailable, fmt.Errorf("error requesting url: %s", err)))
	}
	if resp.StatusCode != 200 {
		log.Fatalf("http status is not 200 OK. request returned the %d status", resp.StatusCode)
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		status.ExitFromError(status.NewError(status.DataUnavailable, fmt.Errorf("error reading data: %s", err)))
	}

	pathFile := fmt.Sprintf("%s/result.json", output)
	err = ioutil.WriteFile(pathFile, data, 0666)
	if err != nil {
		status.ExitFromError(status.NewError(status.SystemError, fmt.Errorf("error writing file: %s", err)))
	}

	fmt.Println(string(data))
}
