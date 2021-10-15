package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/dadosjusbr/executor"
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

	log.Printf("Executando pipeline %s", p.Name)
	result, err := p.Run()
	if err != nil {
		log.Fatalf("Erro executando pipeline: %q\n\"%+v\"", err, p)
	}
	log.Printf("Pipeline %s executado com sucesso!", p.Name)
	fmt.Printf("%+v", result)
}
