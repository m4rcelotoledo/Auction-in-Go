package auction_entity

import (
	"context"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/m4rcelotoledo/Auction-in-Go/internal/internal_error"
)

// calculateEndTime calculates the expiration time of the auction based on the AUCTION_DURATION environment variable
func calculateEndTime() time.Time {
	durationStr := os.Getenv("AUCTION_DURATION")
	if durationStr == "" {
		// Default to 5 minutes if the environment variable is not defined
		durationStr = "5m"
	}

	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		// If there is an error parsing, use 5 minutes as fallback
		duration = 5 * time.Minute
	}

	return time.Now().Add(duration)
}

func CreateAuction(
	productName, category, description string,
	condition ProductCondition) (*Auction, *internal_error.InternalError) {
	auction := &Auction{
		Id:          uuid.New().String(),
		ProductName: productName,
		Category:    category,
		Description: description,
		Condition:   condition,
		Status:      Active,
		Timestamp:   time.Now(),
		EndTime:     calculateEndTime(),
	}

	if err := auction.Validate(); err != nil {
		return nil, err
	}

	return auction, nil
}

func (au *Auction) Validate() *internal_error.InternalError {
	if len(au.ProductName) <= 1 {
		return internal_error.NewBadRequestError("product name must be longer than 1 character")
	}

	if len(au.Category) <= 2 {
		return internal_error.NewBadRequestError("category must be longer than 2 characters")
	}

	if len(au.Description) <= 10 {
		return internal_error.NewBadRequestError("description must be longer than 10 characters")
	}

	if au.Condition != New && au.Condition != Refurbished && au.Condition != Used {
		return internal_error.NewBadRequestError("invalid product condition")
	}

	return nil
}

type Auction struct {
	Id          string
	ProductName string
	Category    string
	Description string
	Condition   ProductCondition
	Status      AuctionStatus
	Timestamp   time.Time
	EndTime     time.Time
}

type ProductCondition int
type AuctionStatus int

const (
	Active AuctionStatus = iota
	Completed
)

const (
	New ProductCondition = iota + 1
	Used
	Refurbished
)

type AuctionRepositoryInterface interface {
	CreateAuction(
		ctx context.Context,
		auctionEntity *Auction) *internal_error.InternalError

	FindAuctions(
		ctx context.Context,
		status AuctionStatus,
		category, productName string) ([]Auction, *internal_error.InternalError)

	FindAuctionById(
		ctx context.Context, id string) (*Auction, *internal_error.InternalError)
}
