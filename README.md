# Auction-in-Go 🏷️

Sistema de leilões desenvolvido em Go com arquitetura limpa, MongoDB como banco de dados e funcionalidade de **fechamento automático de leilões**.

## ✨ Funcionalidades

- **Criação de leilões** com tempo de expiração configurável
- **Sistema de lances** para usuários
- **Fechamento automático** de leilões expirados via worker em background
- **API REST** completa com validações
- **Arquitetura limpa** seguindo princípios SOLID
- **Testes automatizados** para todas as funcionalidades

## 🆕 Nova Funcionalidade: Fechamento Automático

O sistema agora inclui um **worker automático** que:

- ✅ Verifica periodicamente leilões expirados
- ✅ Fecha automaticamente leilões com tempo vencido
- ✅ Atualiza status de `Active` para `Completed`
- ✅ Executa em background sem interferir na performance
- ✅ Configurável via variáveis de ambiente

### Como Funciona

1. **Criação**: Leilão criado com `EndTime` baseado em `AUCTION_DURATION`
2. **Worker**: Executa em background verificando a cada intervalo configurado
3. **Detecção**: Encontra leilões com `status = Active` e `end_time <= now()`
4. **Fechamento**: Atualiza status para `Completed` automaticamente
5. **Logs**: Registra todas as operações para monitoramento

## 🚀 Pré-requisitos

- **Docker** (versão 20.10+)
- **Docker Compose** (versão 2.0+)
- **Go** (versão 1.21+) - apenas para desenvolvimento local

## 🛠️ Como Rodar

### 1. Clone o repositório
```bash
git clone https://github.com/m4rcelotoledo/Auction-in-Go.git
cd Auction-in-Go
```

### 2. Configure as variáveis de ambiente
```bash
# Copie o arquivo de exemplo
cp cmd/auction/env.sample cmd/auction/.env

# Edite o arquivo .env com suas configurações
vim cmd/auction/.env
```

### 3. Inicie os serviços
```bash
# Inicia MongoDB e aplicação
docker-compose up -d

# Para ver os logs
docker-compose logs -f
```

### 4. Acesse a aplicação
- **API**: http://localhost:8080
- **MongoDB**: localhost:27017

## ⚙️ Variáveis de Ambiente

### Configurações Principais

| Variável | Descrição | Padrão | Exemplo |
|----------|-----------|---------|---------|
| `AUCTION_DURATION` | **Duração padrão dos leilões** | `5m` | `2s`, `10m`, `1h` |
| `WORKER_CHECK_INTERVAL` | Intervalo de verificação do worker | `1m` | `500ms`, `30s` |

### Configurações do MongoDB

| Variável | Descrição | Padrão |
|----------|-----------|---------|
| `MONGODB_URL` | URL de conexão com MongoDB | `mongodb://admin:admin@mongodb:27017/auctions?authSource=admin` |
| `MONGODB_DB` | Nome do banco de dados | `auctions` |

### Exemplo de `.env`
```env
# Duração dos leilões (formato Go: 5m, 30s, 1h)
AUCTION_DURATION=5m

# Intervalo do worker (para testes: 500ms, produção: 1m)
WORKER_CHECK_INTERVAL=1m

# MongoDB
MONGODB_URL=mongodb://admin:admin@mongodb:27017/auctions?authSource=admin
MONGODB_DB=auctions
```

## 🧪 Executando Testes

### Testes Locais (com MongoDB rodando)
```bash
# Todos os testes
go test ./... -v

# Testes específicos
go test ./internal/infra/database/auction -v
go test ./internal/entity/auction_entity -v
```

### Testes com Docker
```bash
# Executa testes em container
docker-compose run --rm auction go test ./... -v
```

## 🏗️ Arquitetura

```
Auction-in-Go/
├── cmd/auction/          # Ponto de entrada da aplicação
├── internal/             # Código interno da aplicação
│   ├── entity/           # Entidades de domínio
│   ├── usecase/          # Casos de uso
│   └── infra/            # Infraestrutura (API, DB)
├── configuration/        # Configurações (logger, DB)
└── docker-compose.yml    # Orquestração dos serviços
```

### Componentes Principais

- **`auction_entity`**: Entidade Auction com lógica de expiração
- **`auction_worker`**: Worker de fechamento automático
- **`auction_repository`**: Persistência no MongoDB
- **`auction_controller`**: API REST para leilões

## 📊 Endpoints da API

### Leilões
- `POST /auctions` - Criar leilão
- `GET /auctions/:id` - Buscar leilão por ID
- `GET /auctions` - Listar leilões (com filtros)

### Lances
- `POST /bids` - Criar lance
- `GET /bids/:id` - Buscar lance por ID

### Usuários
- `GET /users/:id` - Buscar usuário por ID

## 🔍 Monitoramento

### Logs do Worker
```bash
# Ver logs do worker de fechamento automático
docker-compose logs auction | grep "auction closing worker"
```

### Status dos Leilões
```bash
# Ver leilões ativos
curl http://localhost:8080/auction?status=0

# Ver leilões fechados
curl http://localhost:8080/auction?status=1
```

## 🚨 Troubleshooting

### Problemas Comuns

1. **MongoDB não conecta**
   ```bash
   # Verifique se o container está rodando
   docker-compose ps

   # Reinicie os serviços
   docker-compose down && docker-compose up -d
   ```

2. **Worker não está fechando leilões**
   ```bash
   # Verifique as variáveis de ambiente
   docker-compose exec auction env | grep AUCTION

   # Verifique os logs
   docker-compose logs auction
   ```

3. **Testes falhando**
   ```bash
   # Limpe o banco de teste
   docker-compose exec mongodb mongo auction_test --eval "db.auctions.deleteMany({})"
   ```

## 🤝 Contribuindo

1. Fork o projeto
2. Crie uma branch para sua feature (`git checkout -b feature/AmazingFeature`)
3. Commit suas mudanças (`git commit -m 'Add some AmazingFeature'`)
4. Push para a branch (`git push origin feature/AmazingFeature`)
5. Abra um Pull Request

## 📝 Licença

Este projeto está sob a licença MIT. Veja o arquivo `LICENSE` para mais detalhes.

## 👨‍💻 Autor

**Marcelo Toledo**
- GitHub: [@m4rcelotoledo](https://github.com/m4rcelotoledo)

---

⭐ **Se este projeto te ajudou, considere dar uma estrela!**
