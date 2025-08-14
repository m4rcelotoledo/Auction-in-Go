package auction

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/m4rcelotoledo/Auction-in-Go/configuration/logger"
	"github.com/m4rcelotoledo/Auction-in-Go/internal/entity/auction_entity"
	"github.com/m4rcelotoledo/Auction-in-Go/internal/internal_error"
	"go.uber.org/zap"

	"go.mongodb.org/mongo-driver/bson"
)

// findExpiredAuctions queries the database for all auctions with status = Active
// and where end_time is less than or equal to the current time
func (ar *AuctionRepository) findExpiredAuctions(ctx context.Context) ([]auction_entity.Auction, *internal_error.InternalError) {
	now := time.Now().Unix()

	filter := bson.M{
		"status":   auction_entity.Active,
		"end_time": bson.M{"$lte": now},
	}

	cursor, err := ar.Collection.Find(ctx, filter)
	if err != nil {
		logger.Error("Error finding expired auctions", err)
		return nil, internal_error.NewInternalServerError("Error finding expired auctions")
	}
	defer cursor.Close(ctx)

	var auctionsMongo []AuctionEntityMongo
	if err := cursor.All(ctx, &auctionsMongo); err != nil {
		logger.Error("Error decoding expired auctions", err)
		return nil, internal_error.NewInternalServerError("Error decoding expired auctions")
	}

	var auctionsEntity []auction_entity.Auction
	for _, auction := range auctionsMongo {
		auctionsEntity = append(auctionsEntity, auction_entity.Auction{
			Id:          auction.Id,
			ProductName: auction.ProductName,
			Category:    auction.Category,
			Description: auction.Description,
			Condition:   auction.Condition,
			Status:      auction.Status,
			Timestamp:   time.Unix(auction.Timestamp, 0),
			EndTime:     time.Unix(auction.EndTime, 0),
		})
	}

	return auctionsEntity, nil
}

// closeAuction receives an auction ID and executes an UPDATE in the database to change its status to Completed
func (ar *AuctionRepository) closeAuction(ctx context.Context, auctionId string) *internal_error.InternalError {
	filter := bson.M{"_id": auctionId}
	update := bson.M{
		"$set": bson.M{
			"status": auction_entity.Completed,
		},
	}

	result, err := ar.Collection.UpdateOne(ctx, filter, update)
	if err != nil {
		logger.Error("Error trying to close auction", err)
		return internal_error.NewInternalServerError("Error trying to close auction")
	}

	if result.MatchedCount == 0 {
		logger.Error("Auction not found for closing", nil)
		return internal_error.NewNotFoundError("Auction not found for closing")
	}

	if result.ModifiedCount == 0 {
		logger.Error("Auction was not modified during closing", nil)
		return internal_error.NewInternalServerError("Auction was not modified during closing")
	}

	logger.Info("Auction closed successfully", zap.String("auctionId", auctionId))

	return nil
}

// StartAuctionClosingWorker starts the worker that checks and closes expired auctions
func (ar *AuctionRepository) StartAuctionClosingWorker(ctx context.Context) {
	logger.Info("Starting auction closing worker")

	// Check interval (1 minute) - can be overridden for tests
	checkInterval := 1 * time.Minute

	// For tests, use a smaller interval if the environment variable is defined
	if testInterval := os.Getenv("WORKER_CHECK_INTERVAL"); testInterval != "" {
		if parsedInterval, err := time.ParseDuration(testInterval); err == nil {
			checkInterval = parsedInterval
			logger.Info("Worker using test interval", zap.String("interval", testInterval))
		}
	}

	for {
		select {
		case <-ctx.Done():
			logger.Info("Auction closing worker stopped")
			return
		default:
			// Find expired auctions
			expiredAuctions, err := ar.findExpiredAuctions(ctx)
			if err != nil {
				logger.Error("Error finding expired auctions in worker", err)
				time.Sleep(checkInterval)
				continue
			}

			// Logs for debug
			logger.Info("Worker check", zap.String("expiredAuctionsFound", fmt.Sprintf("%d", len(expiredAuctions))))

			// Close expired auctions
			for _, auction := range expiredAuctions {
				if err := ar.closeAuction(ctx, auction.Id); err != nil {
					logger.Error("Error closing expired auction", err)
				}
			}

			// Log the number of closed auctions
			if len(expiredAuctions) > 0 {
				logger.Info("Closed expired auctions", zap.Int("count", len(expiredAuctions)))
			}

			// Wait before the next check
			time.Sleep(checkInterval)
		}
	}
}
