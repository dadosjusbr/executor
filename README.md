# executor

A partir da necessidade da configuração de um Pipeline no [DadosJusBR](https://dadosjusbr.org/), criamos um pacote em Go chamado ***executor***. Ele será utilizado dentro da orquestradora [Alba](https://github.com/dadosjusbr/alba) e sua principal função é ser capaz de [definir](https://medium.com/r/?url=https%3A%2F%2Fgithub.com%2Fdadosjusbr%2Fexecutor%2Fblob%2Facd99e1da1cf1e0298e5fecb1ce6790ca43c40e8%2Fpipeline.go%23L32), configurar e executar um [Pipeline](https://medium.com/r/?url=https%3A%2F%2Fgithub.com%2Fdadosjusbr%2Fexecutor%2Fblob%2Facd99e1da1cf1e0298e5fecb1ce6790ca43c40e8%2Fpipeline.go%23L32).

Consideramos um Pipeline como uma sequência de [estágios](https://github.com/dadosjusbr/executor/blob/45cacc0878707a7cbc9ed0d38299959e67c72f68/pipeline.go#L24) que visam realizar uma tarefa macro, onde essa tarefa foi dividida em uma série de programas "dockerizados" que são executados de forma sequencial. De forma simplificada, a saída padrão de um estágio vira a entrada padrão do estágio seguinte.


## Entendendo um Pipeline DadosJusBR

Considerando o contexto do DadosJusBR, o pipeline capaz de atingir a tarefa de libertação de dados do sistema judiciário brasileiro tem os seguintes estágios:

- Coleta: Etapa responsável por encontrar, fazer o download dos arquivos e consolidar/traduzir as informações para um formato único do DadosJusBr;
- Validação: Responsável por fazer validações nos dados de acordo a cada contexto;
- Empacotamento: Responsável por padronizar os dados no formato de [datapackages](https://medium.com/r/?url=https%3A%2F%2Ffrictionlessdata.io%2Fdata-package%2F);
- Armazenamento: Responsável por armazenar os dados extraídos, além de versionar também os artefatos baixados e gerados durante a coleta;

Cada programa capaz de cumprir um estágio é dockerizado, ou seja, escrito de forma capaz de ser executado pela ferramenta docker, com os comandos "docker build"(que constrói uma imagem a partir das especificações definidas) e "docker run"(que executa essa imagem em um *container*).

No Pipeline, cada estágio, exceto o primeiro, recebe a saída padrão do estágio anterior, essa é uma forma de compartilharem informações. Também podemos definir um estágio chamado [ErrorHandler](https://github.com/dadosjusbr/executor/blob/45cacc0878707a7cbc9ed0d38299959e67c72f68/pipeline.go#L39), que será construído e executado quando ocorrer um erro no fluxo padrão. Consideramos como fluxo padrão a sequência de estágios descrita na definição do Pipeline.

Uma vez que sua aplicação seja dockerizada, você pode utilizar o executor para configurar e executar um Pipeline de acordo com suas necessidades.

## Configurações, compartilhamento de informações e tratamento de erros

### Variáveis de ambiente

É possível configurar variáveis de ambiente tanto para o `docker build` quanto para o `docker run` do seu estágio. E, caso elas se repitam para todos os estágios, você também pode defini-las como variáveis padrão. Para que essas customizações pudessem ser realizadas, adicionamos na sintaxe do Pipeline estruturas especiais que devem ser configuradas a partir da sua necessidade: [BuildEnv, RunEnv](https://github.com/dadosjusbr/executor/blob/45cacc0878707a7cbc9ed0d38299959e67c72f68/pipeline.go#L28), [DefaultBuildEnv e DefaultRunEnv](https://github.com/dadosjusbr/executor/blob/45cacc0878707a7cbc9ed0d38299959e67c72f68/pipeline.go#L36).

*Exemplo*
``` go
buildEnv := map[string]string{
    "COMMIT":           "1a2b3c4d5e",
}

runEnv := map[string]string{
    "URL":           "https://dadosjusbr.org",
    "OUTPUT_FOLDER": "/output",
}
```

### Volume dadosjusbr

Antes de iniciar a execução do primeiro estágio de um Pipeline, nós criamos um volume chamado dadosjusbr. Esse volume é  do tipo bind e será montado em uma pasta local(chamada output) criada a partir do diretório base que você nos informa na definição do Pipeline [(Veja linhas 27 e 35 da estrutura do Pipeline)](https://github.com/dadosjusbr/executor/blob/45cacc0878707a7cbc9ed0d38299959e67c72f68/pipeline.go#L27). A cada "docker run" de um estágio, esse mesmo volume é utilizado para espelhar o conteúdo da pasta /output **de dentro do container em execução** para a sua pasta local.

`$ docker run -i -v dadosjusbr:/output stage-coleta`

Por isso, nós recomendamos fortemente que quando o seu programa precisar persistir arquivos ele utilize a pasta `/output` dentro do container. E assim o seu diretório base local tera todos os conteúdos persistidos pelos estágios. 

### Tratamento de erros

Caso você deseje descrever um comportamento padrão para quando houver erro na execução do pipeline, você vai definir um estágio especial para isso: o  que nós chamamos de ErrorHandler. [Ele será construído e executado como os demais](https://github.com/dadosjusbr/executor/blob/45cacc0878707a7cbc9ed0d38299959e67c72f68/pipeline.go#L213), porém, se ocorrer outro erro, interrompemos a execução e retornamos todos os detalhes da execução do Pipeline até aquele ponto.

Aqui, consideramos erro quando a construção ou execução de uma imagem [levanta um erro durante seu processamento](https://github.com/dadosjusbr/executor/blob/45cacc0878707a7cbc9ed0d38299959e67c72f68/pipeline.go#L151) ou [quando não levanta erro mas retorna um status diferente de 0(OK)](https://github.com/dadosjusbr/executor/blob/45cacc0878707a7cbc9ed0d38299959e67c72f68/pipeline.go#L155).

---

## Como usar o pacote *executor*?

O tutorial de utilização pode ser encontrado [nesse link](https://medium.com/dadosjusbr/dadosjusbr-executando-um-pipeline-cfd26a50165e). E o código completo do tutorial [aqui](https://github.com/dadosjusbr/executor/tree/master/tutorial).

Para mais detalhes, dúvidas ou sugestões você pode entrar em contato conosco pelo email dadosjusbr@gmail.com.

***Esse pacote foi desenvolvido a partir de todo o trabalho e experiência dos integrantes do time [DadosJusBR](https://dadosjusbr.org/equipe)***.
