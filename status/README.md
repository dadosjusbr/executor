# Status Package

Esse pacote tem o objetivo de padronizar os status de execução dos coletores e de possíveis erros em estágios de um pipeline. O Pacote foi originalmente desenvolvido no [dadosjusbr/coletores](https://github.com/dadosjusbr/coletores), e foi copiado para cá por conta da transição da funcionalidade de execução dos coletores. Em um futuro próximo, o pacote em [dadosjusbr/coletores/status](https://github.com/dadosjusbr/coletores/tree/master/status) será descontinuado, passando a valer apenas o **dadosjusbr/executor/status**.

## Status disponíveis

Abaixo segue uma tabela com os status disponíveis:

| Status code | Significado |
--------------|----------
|OK| O processo ocorreu sem erros.|
|InvalidParameters|Deve ser utilizado em caso de parâmetros inválidos, como dados que não sejam condizentes com ano e mês, ou parametros obrigatórios que não foram fornecidos.|
|SystemError|Deve ser usado em casos como falha ao criar o diretório dos arquivos ou na leitura de arquivos.|
|ConnectionError|Deve ser usado em problemas de conexão, como timeout ou serviço fora do ar.|
|DataUnavailable|A informação solicitada não foi encontrada, provavelmente o órgão não disponibilizou ainda.|
|InvalidFile| Deve ser usado para cenários onde o arquivo não é o esperado ou em caso de falhas na extração de dados.|
|Unknown|Deve ser usando quando um erro inesperado ocorrer.|
|SetupError|Deve ser usado quando um erro acontecer na configuração do ambiente para execução.|
|BuildError|Deve ser usado para relatar erros que ocorreram durante a construção de uma imagem.|
|RunError|Deve ser usado para relatar erros que ocorreram durante a execução de uma imagem.|
|ErrorHandlerError|Deve ser usado para relatar erros que ocorreram durante a construção ou execução no estágio de manipulação de erros.|
______________


## Exemplo de uso

```
import (
	"fmt"
    "https://github.com/dadosjusbr/coletores/status"
)

func myFunc() *StatusError {
  // code
  return status.NewStatusError(status.DataUnavailable, err.Error())
}

func main() {
  err := myFunc()
  status.ExitFromError(err)
}
```
## Exemplo de uso no Tutorial para execução de Pipeline

[main.go](https://github.com/dadosjusbr/executor/blob/master/tutorial/stage-go/main.go)