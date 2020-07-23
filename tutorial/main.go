package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/dadosjusbr/executor"
)

func main() {
	goPath := os.Getenv("GOPATH")
	repo := fmt.Sprintf("%s/src/github.com/dadosjusbr/executor/tutorial", goPath)

	collectRunEnv := map[string]string{
		"URL": "https://dadosjusbr.org/api/v1/orgao/trt13/2020/4",
	}

	p := executor.Pipeline{}
	p.Name = "Tutorial"
	p.DefaultRepo = repo
	p.Stages = []executor.Stage{
		{
			Name:   "Get data from API Dadosjusbr",
			Dir:    "stagego",
			RunEnv: collectRunEnv,
		},
	}

	result, err := p.Run()
	if err != nil {
		log.Fatal(err)
	}
	resultJSON, err := json.MarshalIndent(result, "", " ")
	if err != nil {
		log.Fatal(err)
	}
	err = ioutil.WriteFile("result_pipeline.json", resultJSON, 0644)
	if err != nil {
		log.Fatal(err)
	}
}
