# MDW
Este projeto serve de base para estudos de backend, com foco em balanceamento de carga, monitoramento de saúde e testes de falhas (chaos engineering).

## Configuração inicial
Primeiro, crie um arquivo .env no diretório principal com as seguintes variáveis de ambiente:

* **SERVER_01_PORT** e **SERVER_02_PORT**: Portas dos servidores backend. São duas por padrão, mas você pode adicionar mais servidores modificando o roteador no arquivo routes/router.go e o main.go.
* **CHAOS_MODE_ENABLED**: Define se o modo de falhas (chaos testing) estará ativado no ambiente.
* **CHAOS_FAILURE_RATE**: Taxa de falha para os testes de caos (porcentagem).
* **CHAOS_SHUTDOWN_RATE**: Taxa de quedas para os testes de caos (porcentagem).

## Rodando o projeto
Depois de configurar o arquivo .env, execute o projeto a partir do diretório principal:

`go run main.go`

## Testes de carga
Para realizar testes de carga enquanto os servidores estão em execução, abra outra janela de terminal e execute o seguinte comando:

`go run go run test/load_tester/load_tester.go -url=http://localhost:8000 -duracao=1m -rps=200 -concorrencia=10`

Você pode ajustar os valores das flags (`-duracao`, `-rps`, `-concorrencia`) de acordo com o seu teste.

## Endereços de acesso
Você pode acessar os servidores e o balanceador de carga pelos seguintes endereços:
- localhost:8080 - Servidor 1
- localhost:8081 - Servidor 2
- localhost:8000 - Balanceador de carga (que distribui as requisições entre os servidores usando a estratégia de Round-Robin, desde que o servidor esteja "vivo").

## Funcionalidades futuras
Mais páginas e funcionalidades serão adicionadas ao projeto no futuro.