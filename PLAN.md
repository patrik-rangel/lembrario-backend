📋 Plano de Implementação: Autenticação JWT
O objetivo é trancar todas as rotas de gerenciamento de conteúdo (/contents, /search, /events), deixando apenas o /health e o novo /login abertos.

─── Passo 1: Infraestrutura e Configuração ────────────────────────────────

Variáveis de Ambiente: Adicionar no .env:

JWT_SECRET: Uma string longa e aleatória (a "chave mestra").

ADMIN_USER: Seu usuário para o login.

ADMIN_PASSWORD: Sua senha (no início, pode ser simples, depois evoluímos para hash).

─── Passo 2: O Coração do Auth (internal/api/auth.go) ──────────────────
Criar um serviço responsável por:

GenerateToken: Criar o token com claims (sub: username, exp: 24h).

ValidateToken: Verificar a assinatura e a expiração do token recebido.

─── Passo 3: O Segurança da Porta (internal/api/middleware.go) ──────────
Implementar o JWTMiddleware:

Extrair o token do Header Authorization: Bearer <TOKEN>.

Se o token for inválido ou ausente, retornar 401 Unauthorized imediatamente.

Se for válido, salvar o usuário no contexto do Gin (c.Set("user", username)) e seguir para a rota.

─── Passo 4: O Endpoint de Login (internal/api/login_handler.go) ───────
Criar o handler para POST /login:

Receber JSON com username e password.

Validar contra as variáveis de ambiente.

Retornar o JWT e o tempo de expiração.

─── Passo 5: Proteção das Rotas (internal/api/router.go) ────────────────
Reorganizar o roteador:

Rotas Públicas: /health e /login.

Rotas Protegidas: Criar um router.Group("/") que utiliza o JWTMiddleware e registrar os handlers de conteúdo dentro dele.