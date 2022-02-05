package internal

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gbrlsnchs/jwt/v3"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Auth represents an authenticated user.
type Auth struct {
	AccountID string
	UserID    string
	Email     string
	Role      int
	Token     string
}

func (auth Auth) ReconstructToken() string {
	if strings.HasPrefix(auth.Token, "__tmp__experimental_public") {
		return auth.Token
	}
	return fmt.Sprintf("%s|%s", auth.UserID, auth.Token)
}

// JWTPayload contains the current user token
type JWTPayload struct {
	jwt.Payload
	Token string `json:"token,omitempty"`
}

var (
	ctx = context.Background()
)

type Account struct {
	ID    primitive.ObjectID `bson:"_id" json:"id"`
	Email string             `bson:"email" json:"email"`
}

type Token struct {
	ID        string    `json:"id"`
	AccountID string    `json:"accountId"`
	Token     string    `json:"token"`
	Email     string    `json:"email"`
	Password  string    `json:"-"`
	Role      int       `json:"role"`
	ResetCode string    `json:"-"`
	Created   time.Time `json:"created"`
}

type Login struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type Customer struct {
	ID               string    `bson:"_id" json:"id"`
	Email            string    `bson:"email" json:"email"`
	StripeID         string    `bson:"stripeId" json:"stripeId"`
	SubscriptionID   string    `bson:"subId" json:"subId"`
	IsActive         bool      `bson:"active" json:"-"`
	MonthlyEmailSent int       `bson:"mes" json:"-"`
	Created          time.Time `bson:"created" json:"created"`
}
