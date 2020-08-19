# executor

A partir da necessidade da configuração de um Pipeline no  [DadosJusBR](https://dadosjusbr.org/), criamos um pacote em Go chamado Executor. Ele será utilizado dentro da orquestradora Alba e sua principal função é ser capaz de definir, configurar e executar  um Pipeline.

Consideramos um Pipeline como uma sequência de estágios que visam realizar uma tarefa macro, onde essa tarefa foi dividida em uma série de programas "dockerizados".

## O que é um Pipeline DadosjusBR?

É uma sequência de etapas que visa a libertação e padronização de dados do sistema de justiça brasileiro.

As etapas são: 
- **Coleta**: Responsável por extrair informações, consolidar e fazer a tradução necessária para um formato único do DadosJusBr. Também realiza o download de todo artefato que for necessário para validação dos dados extraídos. Mais informações [nesse link](https://github.com/dadosjusbr/coletores).
- **Validação**: Responsável por fazer validações nos dados de acordo a cada contexto;
- **Empacotamento**: Responsável por padronizar os dados no formato de datapackages;
- **Armazenamento**: Responsável por armazenar os dados extraídos, além de versionar também os artefatos baixados e gerados durante a coleta;
