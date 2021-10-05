# Comando para execução de pipelines poliglotas

Este comando visa facilitar a execução de pipelines poliglotas - que podem ter [estágios](https://github.com/dadosjusbr/executor/blob/45cacc0878707a7cbc9ed0d38299959e67c72f68/pipeline.go#L24) escritos em qualquer linguagem de programação - usando a biblioteca [executor](github.com/dadosjusbr/executor). Estágios são contêineres Docker conectados via entrada e saída padrão. Se um estágio retorna um código de execução diferente de zero a execução do pipeline é abortada. Apesar de ser criada para suprir uma demanda do projeto dadosjusbr, o executor acaba é um comando que serve de propósito geral.

Essa linha de comando recebe via entrada padrão a descrição de um [Pipeline](https://github.com/dadosjusbr/executor/blob/45cacc0878707a7cbc9ed0d38299959e67c72f68/pipeline.go#L33). Um exemplo de entrada por de ser encontrado em [exemplo.json]. Para executá-lo:

```sh
$ go build
$ cat exemplo.json | sudo ./executor
```