package auction

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/m4rcelotoledo/Auction-in-Go/internal/entity/auction_entity"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// setupTestDatabase creates a test connection to MongoDB
func setupTestDatabase(t *testing.T) (*mongo.Database, func()) {
	ctx := context.Background()

	// Connects to the test MongoDB (using the same credentials as .env)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://admin:admin@localhost:27017/auction_test?authSource=admin"))
	if err != nil {
		t.Fatalf("Erro ao conectar ao MongoDB: %v", err)
	}

	// Pings to verify connection
	err = client.Ping(ctx, nil)
	if err != nil {
		t.Fatalf("Erro ao fazer ping no MongoDB: %v", err)
	}

	// Uses test database
	database := client.Database("auction_test")

	// Cleanup function
	cleanup := func() {
		// Clears test collection (only deletes documents, not the collection)
		_, err := database.Collection("auctions").DeleteMany(ctx, bson.M{})
		if err != nil {
			t.Logf("Warning: could not clear test collection: %v", err)
		}

		// Closes connection
		err = client.Disconnect(ctx)
		if err != nil {
			t.Logf("Warning: could not disconnect from test database: %v", err)
		}
	}

	return database, cleanup
}

// TestCreateAuctionSimple tests the basic creation of an auction
func TestCreateAuctionSimple(t *testing.T) {
	// Setup test database
	database, cleanup := setupTestDatabase(t)
	defer cleanup()

	// Configures duration for the test
	os.Setenv("AUCTION_DURATION", "5m")

	// Creates auction repository
	auctionRepo := NewAuctionRepository(database)

	// Creates a test auction
	ctx := context.Background()
	auction, err := auction_entity.CreateAuction(
		"Produto Teste",
		"Eletrônicos",
		"Descrição de teste para validação básica",
		auction_entity.New,
	)
	if err != nil {
		t.Fatalf("Erro inesperado ao criar auction: %v", err)
	}
	if auction == nil {
		t.Fatal("Auction não deveria ser nil")
	}

	// Verifies that the auction was created with status Active
	if auction.Status != auction_entity.Active {
		t.Errorf("Status esperado: %v, recebido: %v", auction_entity.Active, auction.Status)
	}
	if !auction.EndTime.After(time.Now()) {
		t.Error("EndTime deveria ser maior que o tempo atual")
	}

	// Saves the auction in the database
	err = auctionRepo.CreateAuction(ctx, auction)
	if err != nil {
		t.Fatalf("Erro ao salvar auction no banco: %v", err)
	}

	// Searches for the auction in the database to confirm that it was saved
	savedAuction, err := auctionRepo.FindAuctionById(ctx, auction.Id)
	if err != nil {
		t.Fatalf("Erro ao buscar auction no banco: %v", err)
	}
	if savedAuction == nil {
		t.Fatal("Auction salvo não deveria ser nil")
	}
	if savedAuction.Status != auction_entity.Active {
		t.Errorf("Status do auction salvo esperado: %v, recebido: %v", auction_entity.Active, savedAuction.Status)
	}

	t.Logf("Teste básico concluído com sucesso: leilão %s foi criado e salvo", auction.Id)
}

// TestAuctionAutomaticClosing tests the automatic closing of auctions
func TestAuctionAutomaticClosing(t *testing.T) {
	// Setup test database
	database, cleanup := setupTestDatabase(t)
	defer cleanup()

	// Configures duration for the test (2 seconds)
	os.Setenv("AUCTION_DURATION", "2s")
	// Configures interval for the worker (500ms)
	os.Setenv("WORKER_CHECK_INTERVAL", "500ms")

	// Creates auction repository
	auctionRepo := NewAuctionRepository(database)

	// Creates a test auction
	ctx := context.Background()
	auction, err := auction_entity.CreateAuction(
		"Produto Teste",
		"Eletrônicos",
		"Descrição de teste para validação do fechamento automático",
		auction_entity.New,
	)
	if err != nil {
		t.Fatalf("Erro inesperado ao criar auction: %v", err)
	}
	if auction == nil {
		t.Fatal("Auction não deveria ser nil")
	}

	// Verifies that the auction was created with status Active
	if auction.Status != auction_entity.Active {
		t.Errorf("Status esperado: %v, recebido: %v", auction_entity.Active, auction.Status)
	}
	if !auction.EndTime.After(time.Now()) {
		t.Error("EndTime deveria ser maior que o tempo atual")
	}

	// Saves the auction in the database
	err = auctionRepo.CreateAuction(ctx, auction)
	if err != nil {
		t.Fatalf("Erro ao salvar auction no banco: %v", err)
	}

	// Searches for the auction in the database to confirm that it was saved
	savedAuction, err := auctionRepo.FindAuctionById(ctx, auction.Id)
	if err != nil {
		t.Fatalf("Erro ao buscar auction no banco: %v", err)
	}
	if savedAuction == nil {
		t.Fatal("Auction salvo não deveria ser nil")
	}
	if savedAuction.Status != auction_entity.Active {
		t.Errorf("Status do auction salvo esperado: %v, recebido: %v", auction_entity.Active, savedAuction.Status)
	}

	// Starts the automatic closing worker in a goroutine
	workerCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go auctionRepo.StartAuctionClosingWorker(workerCtx)

	// Waits for a time slightly longer than the defined duration (3 seconds)
	// to give the worker time to process the expired auction
	time.Sleep(3 * time.Second)

	// Waits for a little more to ensure that the worker processed
	time.Sleep(1 * time.Second)

	// Stops the worker
	cancel()

	// Waits for a little more to ensure that the worker finished
	time.Sleep(500 * time.Millisecond)

	// Searches for the same auction in the database again
	closedAuction, err := auctionRepo.FindAuctionById(ctx, auction.Id)
	if err != nil {
		t.Fatalf("Erro ao buscar auction fechado no banco: %v", err)
	}
	if closedAuction == nil {
		t.Fatal("Auction fechado não deveria ser nil")
	}

	// Logs for debug
	t.Logf("Status do leilão após worker: %v", closedAuction.Status)
	t.Logf("EndTime do leilão: %v", closedAuction.EndTime)
	t.Logf("Tempo atual: %v", time.Now())

	// Verifies that the auction status is now Completed (closed)
	if closedAuction.Status != auction_entity.Completed {
		t.Errorf("Status esperado: %v, recebido: %v. O leilão deveria ter sido fechado automaticamente após expirar",
			auction_entity.Completed, closedAuction.Status)
	}

	// Verifies that the EndTime is less than the current time (auction expired)
	if !closedAuction.EndTime.Before(time.Now()) {
		t.Error("O leilão deveria ter expirado")
	}

	t.Logf("Teste concluído com sucesso: leilão %s foi fechado automaticamente", auction.Id)
}

// TestAuctionNotExpiredYet tests that auctions that have not expired are not closed
func TestAuctionNotExpiredYet(t *testing.T) {
	// Setup test database
	database, cleanup := setupTestDatabase(t)
	defer cleanup()

	// Configures duration for the test (10 seconds)
	os.Setenv("AUCTION_DURATION", "10s")

	// Creates auction repository
	auctionRepo := NewAuctionRepository(database)

	// Creates a test auction
	ctx := context.Background()
	auction, err := auction_entity.CreateAuction(
		"Produto Teste Não Expirado",
		"Eletrônicos",
		"Descrição de teste para validação de leilão não expirado",
		auction_entity.Used,
	)
	if err != nil {
		t.Fatalf("Erro inesperado ao criar auction: %v", err)
	}
	if auction == nil {
		t.Fatal("Auction não deveria ser nil")
	}

	// Saves the auction in the database
	err = auctionRepo.CreateAuction(ctx, auction)
	if err != nil {
		t.Fatalf("Erro ao salvar auction no banco: %v", err)
	}

	// Starts the automatic closing worker in a goroutine
	workerCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go auctionRepo.StartAuctionClosingWorker(workerCtx)

	// Waits for only 2 seconds (less than the duration of 10s)
	time.Sleep(2 * time.Second)

	// Stops the worker
	cancel()

	// Searches for the auction in the database
	savedAuction, err := auctionRepo.FindAuctionById(ctx, auction.Id)
	if err != nil {
		t.Fatalf("Erro ao buscar auction no banco: %v", err)
	}
	if savedAuction == nil {
		t.Fatal("Auction salvo não deveria ser nil")
	}

	// Verifies that the auction status is still Active (not closed)
	if savedAuction.Status != auction_entity.Active {
		t.Errorf("Status esperado: %v, recebido: %v. O leilão não deveria ter sido fechado ainda, pois não expirou",
			auction_entity.Active, savedAuction.Status)
	}

	// Verifies that the EndTime is still greater than the current time
	if !savedAuction.EndTime.After(time.Now()) {
		t.Error("O leilão não deveria ter expirado ainda")
	}

	t.Logf("Teste concluído com sucesso: leilão %s não foi fechado prematuramente", auction.Id)
}

// TestCreateAuctionEntity tests only the creation of the Auction entity
func TestCreateAuctionEntity(t *testing.T) {
	// Configures duration for the test
	os.Setenv("AUCTION_DURATION", "5m")

	// Creates a test auction
	auction, err := auction_entity.CreateAuction(
		"Produto Teste",
		"Eletrônicos",
		"Descrição de teste para validação da entidade",
		auction_entity.New,
	)

	// Logs for debug
	t.Logf("Auction criada: %+v", auction)
	t.Logf("Erro retornado: %v", err)

	// Verifies that there was no error
	if err != nil {
		t.Errorf("Erro inesperado ao criar auction: %v", err)
	}

	if auction == nil {
		t.Error("Auction não deveria ser nil")
		return
	}

	// Verifies that the auction was created with status Active
	if auction.Status != auction_entity.Active {
		t.Errorf("Status esperado: %v, recebido: %v", auction_entity.Active, auction.Status)
	}

	if !auction.EndTime.After(time.Now()) {
		t.Error("EndTime deveria ser maior que o tempo atual")
	}

	t.Logf("Teste da entidade concluído com sucesso: leilão %s foi criado", auction.Id)
}

// TestBasicFunctionality tests basic functionality
func TestBasicFunctionality(t *testing.T) {
	// Configures duration for the test
	os.Setenv("AUCTION_DURATION", "5m")

	// Creates a test auction
	auction, err := auction_entity.CreateAuction(
		"Produto Teste",
		"Eletrônicos",
		"Descrição de teste para validação básica",
		auction_entity.New,
	)

	// Logs for debug
	t.Logf("Auction criada: %+v", auction)
	t.Logf("Erro retornado: %v", err)

	// Verifies that there was no error
	if err != nil {
		t.Errorf("Erro inesperado ao criar auction: %v", err)
	}

	if auction == nil {
		t.Error("Auction não deveria ser nil")
		return
	}

	// Verifies that the auction was created with status Active
	if auction.Status != auction_entity.Active {
		t.Errorf("Status esperado: %v, recebido: %v", auction_entity.Active, auction.Status)
	}

	if !auction.EndTime.After(time.Now()) {
		t.Error("EndTime deveria ser maior que o tempo atual")
	}

	t.Logf("Teste básico concluído com sucesso: leilão %s foi criado", auction.Id)
}

// TestFindExpiredAuctions tests if the findExpiredAuctions function is working
func TestFindExpiredAuctions(t *testing.T) {
	// Setup test database
	database, cleanup := setupTestDatabase(t)
	defer cleanup()

	// Configures duration for the test (1 second)
	os.Setenv("AUCTION_DURATION", "1s")

	// Creates auction repository
	auctionRepo := NewAuctionRepository(database)

	// Creates a test auction
	ctx := context.Background()
	auction, err := auction_entity.CreateAuction(
		"Produto Teste Expirado",
		"Eletrônicos",
		"Descrição de teste para validação de expiração",
		auction_entity.New,
	)
	if err != nil {
		t.Fatalf("Erro inesperado ao criar auction: %v", err)
	}
	if auction == nil {
		t.Fatal("Auction não deveria ser nil")
	}

	// Saves the auction in the database
	err = auctionRepo.CreateAuction(ctx, auction)
	if err != nil {
		t.Fatalf("Erro ao salvar auction no banco: %v", err)
	}

	// Waits for 2 seconds for the auction to expire
	time.Sleep(2 * time.Second)

	// Verifies that the auction expired
	now := time.Now()
	if auction.EndTime.After(now) {
		t.Logf("Leilão ainda não expirou. EndTime: %v, Now: %v", auction.EndTime, now)
	} else {
		t.Logf("Leilão expirou. EndTime: %v, Now: %v", auction.EndTime, now)
	}

	// Searches for expired auctions
	expiredAuctions, err := auctionRepo.findExpiredAuctions(ctx)
	if err != nil {
		t.Fatalf("Erro ao buscar leilões expirados: %v", err)
	}

	t.Logf("Leilões expirados encontrados: %d", len(expiredAuctions))
	for i, expired := range expiredAuctions {
		t.Logf("Leilão %d: ID=%s, Status=%v, EndTime=%v", i, expired.Id, expired.Status, expired.EndTime)
	}

	// Verifies that the expired auction was found
	if len(expiredAuctions) == 0 {
		t.Error("Deveria ter encontrado pelo menos um leilão expirado")
	} else {
		t.Logf("Teste concluído com sucesso: encontrou %d leilão(ões) expirado(s)", len(expiredAuctions))
	}
}

// TestCloseAuction tests if the closeAuction function is working
func TestCloseAuction(t *testing.T) {
	// Setup test database
	database, cleanup := setupTestDatabase(t)
	defer cleanup()

	// Configures duration for the test
	os.Setenv("AUCTION_DURATION", "5m")

	// Creates auction repository
	auctionRepo := NewAuctionRepository(database)

	// Creates a test auction
	ctx := context.Background()
	auction, err := auction_entity.CreateAuction(
		"Produto Teste para Fechamento",
		"Eletrônicos",
		"Descrição de teste para validação de fechamento",
		auction_entity.New,
	)
	if err != nil {
		t.Fatalf("Erro inesperado ao criar auction: %v", err)
	}
	if auction == nil {
		t.Fatal("Auction não deveria ser nil")
	}

	// Saves the auction in the database
	err = auctionRepo.CreateAuction(ctx, auction)
	if err != nil {
		t.Fatalf("Erro ao salvar auction no banco: %v", err)
	}

	// Verifies that the auction was created with status Active
	savedAuction, err := auctionRepo.FindAuctionById(ctx, auction.Id)
	if err != nil {
		t.Fatalf("Erro ao buscar auction no banco: %v", err)
	}
	if savedAuction.Status != auction_entity.Active {
		t.Errorf("Status esperado: %v, recebido: %v", auction_entity.Active, savedAuction.Status)
	}

	// Closes the auction
	err = auctionRepo.closeAuction(ctx, auction.Id)
	if err != nil {
		t.Fatalf("Erro ao fechar auction: %v", err)
	}

	// Verifies that the auction was closed
	closedAuction, err := auctionRepo.FindAuctionById(ctx, auction.Id)
	if err != nil {
		t.Fatalf("Erro ao buscar auction fechado no banco: %v", err)
	}
	if closedAuction.Status != auction_entity.Completed {
		t.Errorf("Status esperado após fechamento: %v, recebido: %v", auction_entity.Completed, closedAuction.Status)
	}

	t.Logf("Teste concluído com sucesso: leilão %s foi fechado", auction.Id)
}
