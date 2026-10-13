package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// MarketplaceAction represents actions sent by GitHub Marketplace webhook events.
type MarketplaceAction string

const (
	ActionPurchased              MarketplaceAction = "purchased"
	ActionCancelled              MarketplaceAction = "cancelled"
	ActionChanged                MarketplaceAction = "changed"
	ActionPendingChange          MarketplaceAction = "pending_change"
	ActionPendingChangeCancelled MarketplaceAction = "pending_change_cancelled"
)

// MarketplaceAccount represents the purchasing organization or user.
type MarketplaceAccount struct {
	Type  string `json:"type"`  // "User" or "Organization"
	ID    int64  `json:"id"`
	Login string `json:"login"` // GitHub username or org handle
	Email string `json:"email,omitempty"`
}

// MarketplacePlan describes the tier selected in GitHub Marketplace.
type MarketplacePlan struct {
	ID                  int64  `json:"id"`
	Name                string `json:"name"`
	Description         string `json:"description"`
	MonthlyPriceInCents int    `json:"monthly_price_in_cents"`
	PriceModel          string `json:"price_model"` // "FREE", "FLAT_RATE", or "PER_UNIT"
}

// MarketplacePurchaseDetails contains the active or previous subscription info.
type MarketplacePurchaseDetails struct {
	Account          MarketplaceAccount `json:"account"`
	BillingCycle     string             `json:"billing_cycle"` // "monthly" or "yearly"
	UnitCount        int                `json:"unit_count"`
	OnFreeTrial      bool               `json:"on_free_trial"`
	FreeTrialEndsOn  *time.Time         `json:"free_trial_ends_on,omitempty"`
	Plan             MarketplacePlan    `json:"plan"`
}

// MarketplaceEvent is the root payload sent to webhook endpoints on plan changes.
type MarketplaceEvent struct {
	Action                     MarketplaceAction           `json:"action"`
	EffectiveDate              time.Time                   `json:"effective_date"`
	MarketplacePurchase        MarketplacePurchaseDetails  `json:"marketplace_purchase"`
	PreviousMarketplacePurchase *MarketplacePurchaseDetails `json:"previous_marketplace_purchase,omitempty"`
	Sender                     MarketplaceAccount          `json:"sender"`
}

// VerifySignature validates GitHub's X-Hub-Signature-256 HMAC header.
func VerifySignature(payload []byte, signature, secret string) bool {
	if secret == "" {
		// CHANGE HERE: Return false if you require strict secret enforcement even in dev
		return true
	}
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}
	expectedMAC := hmac.New(sha256.New, []byte(secret))
	expectedMAC.Write(payload)
	expectedHex := "sha256=" + hex.EncodeToString(expectedMAC.Sum(nil))
	return hmac.Equal([]byte(signature), []byte(expectedHex))
}

// ParseMarketplaceWebhook decodes the marketplace_purchase payload and verifies signature.
func ParseMarketplaceWebhook(payload []byte, signature, secret string) (*MarketplaceEvent, error) {
	if !VerifySignature(payload, signature, secret) {
		return nil, errors.New("invalid webhook signature")
	}

	var event MarketplaceEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, fmt.Errorf("failed to parse marketplace event: %w", err)
	}
	return &event, nil
}

// NewMarketplaceHandler returns an HTTP handler for receiving GitHub Marketplace webhooks.
// CHANGE HERE: Pass your custom webhook secret and event callback function
func NewMarketplaceHandler(secret string, onEvent func(*MarketplaceEvent) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Read raw body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Cannot read body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		sig := r.Header.Get("X-Hub-Signature-256")
		event, err := ParseMarketplaceWebhook(body, sig, secret)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		if onEvent != nil {
			if err := onEvent(event); err != nil {
				// CHANGE HERE: Adjust error response code if failure handling requires retry
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		// 200 OK acknowledges receipt of the event to GitHub
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}
}
