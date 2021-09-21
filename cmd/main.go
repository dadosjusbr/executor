package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/dadosjusbr/executor"
)

func main() {
	in, err := io.ReadAll(os.Stdin)
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
