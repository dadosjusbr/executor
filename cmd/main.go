package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/dadosjusbr/executor"
	"github.com/dadosjusbr/executor/status"
)

var (
	input = flag.String("in", "", "Path for the descriptor file.")
)

func main() {
	flag.Parse()

	if *input == "" {
		log.Fatal("Path to the input file not found. Forgot --in?")
	}

	in, err := ioutil.ReadFile(*input)
	if err != nil {
		log.Fatalf("Erro lendo dados da entrada padrão: %q", err)
	}

	var p executor.Pipeline
	if err := json.Unmarshal(in, &p); err != nil {
		log.Fatalf("Erro convertendo pipeline da entrada padrão: %q\n\"%s\"", err, string(in))
	}

	log.Printf("Pipeline: %+v\n\n", p)

	log.Printf("Executando pipeline %s", p.Name)
	result := p.Run()
	if result.Status != status.OK {
		log.Printf("Erro executando pipeline: %s. Imprimindo resultado:\n\n", p.Name)
		log.Printf("%+v", result)
		return
	}
	log.Printf("Pipeline %s executado com sucesso! Imprimindo resultado:\n\n", p.Name)
	fmt.Printf("%+v", result)
}
