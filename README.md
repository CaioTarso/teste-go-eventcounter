## Contador de Eventos Concorrente em Go
Este projeto é uma solução para um desafio técnico que consiste em criar um microserviço para contar eventos por usuário, consumindo mensagens de uma fila do RabbitMQ.

A aplicação foi desenvolvida utilizando Go e explora conceitos avançados de concorrência, como goroutines, canais, e o pacote sync para garantir o processamento seguro e eficiente das mensagens.

## O Desafio (Escopo Original)
Com a implementação de um sistema orientado a eventos, houve a necessidade de criar um contador de eventos por usuário. Para resolver o problema a equipe de desenvolvimento bolou a seguinte solução:

Um micro serviço escrito em Go roda periodicamente buscando todas as mensagens disponíveis na fila. Para cada mensagem na fila é extraído o ID do usuário, o ID da mensagem e o tipo do evento. Se a mensagem não tiver sido processada ainda, a mesma deve ser enviada para um canal baseado no tipo e processada em concorrência. O consumidor do canal deve adicionar 1 ao contador do usuário baseado no tipo de evento por cada mensagem única recebida. Após 5 segundos do processamento da última mensagem o serviço deve desligar automaticamente ao finalizar todos os processamentos e escrever a contagem em arquivos json separados por tipo, identificando o usuário e quantas mensagens o mesmo recebeu.

### Arquitetura da Solução
A solução foi estruturada para ser robusta, escalável e de fácil manutenção, separando as responsabilidades em três camadas principais:

- cmd/consumer: O ponto de entrada da aplicação, responsável pela inicialização e orquestração dos componentes.

- internal/consumer: Camada de infraestrutura, responsável pela comunicação com o RabbitMQ, incluindo a lógica de consumo e o desligamento automático por inatividade.

- internal/service: Camada de negócio, onde a lógica de processamento de eventos, verificação de duplicatas e contagem concorrente é implementada.

### Pré-requisitos
Antes de começar, certifique-se de que você tem os seguintes softwares instalados:

- Go (versão 1.18 ou superior)

Docker e Docker Compose

### Como Executar
Siga os passos abaixo para configurar e rodar o projeto localmente.

####  1. Clone o Repositório
`git clone <https://github.com/CaioTarso/teste-go-eventcounter>`

`cd <teste-go-eventcounter>`

#### 2. Suba o Ambiente
O projeto utiliza make para simplificar a gestão do ambiente. O comando abaixo irá subir um container Docker com o RabbitMQ.

`make env-up`

Você pode verificar se o RabbitMQ está rodando acessando http://localhost:15672 (login: guest/guest).

#### 3. Publique as Mensagens de Teste
Este comando irá executar um gerador que publica 100 mensagens de teste na fila do RabbitMQ.

`make generator-publish`

#### 4. Execute o Consumidor
Finalmente, execute a aplicação principal que irá consumir e processar as mensagens.

`go run cmd/consumer/main.go`

#### O que Esperar
- Ao executar o consumidor, você verá os seguintes passos no seu terminal:

- O serviço se conectará ao RabbitMQ e começará a processar as mensagens em lote.

- Logs serão exibidos para cada evento processado.

- Após processar a última mensagem, o serviço aguardará 5 segundos.

- Se nenhuma nova mensagem chegar, ele iniciará um desligamento seguro (graceful shutdown), garantindo que todo o processamento seja finalizado.

- Ao final, serão gerados arquivos .json na raiz do projeto (ex: created.json, updated.json), contendo a contagem final de eventos por usuário.

#### 5. Desligue o Ambiente
Quando terminar, você pode parar e remover o container do RabbitMQ com o comando:

`make env-down`
