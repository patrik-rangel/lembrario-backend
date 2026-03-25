Plano de Refatoração: Inversão de Dependência e Configuração por .env
Objetivo: Centralizar a fiação do projeto no main.go (Composition Root), carregar variáveis de ambiente via .env e injetar dependências para facilitar testes e manutenção.

🛠️ Passo 1: Configuração de Ambiente
[ ] Instalar github.com/joho/godotenv.

[ ] Criar arquivo .env.example com as chaves: DATABASE_URL, DATABASE_PORT, DATABASE_USERNAME, DATABASE_PASSWORD, REDIS_ADDR, PORT.

[ ] No main.go, carregar o .env logo no início da execução.

🏗️ Passo 2: Inversão de Dependência (DI)
[ ] Service: Garantir que o ContentService recebe o pool de DB e o cliente Redis no construtor.

[ ] Handler: Refatorar o ContentHandler para receber uma interface do ContentService (permitindo mocks futuros).

[ ] Router: Alterar SetupRouter para receber os handlers injetados como argumentos.

🚀 Passo 3: O Novo main.go
[ ] Instanciar o "grafo de dependências" em ordem:
Config -> DB Pool -> Redis -> Queries -> Services -> Handlers -> Router.

[ ] Iniciar o servidor na porta definida pela env PORT.