package services

import (
	"bytes"
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"linkmeqr/backend/internal/models"
)

var (
	ErrGoogleWalletNotConfigured = errors.New("google wallet is not configured")
	ErrGoogleWalletNoLogo        = errors.New("business has no logo configured")
)

// GoogleWalletConfig carries the credentials of a single Google Wallet API
// Issuer account — LinkMeQR itself is the issuer, and every business using
// the platform gets its own loyalty "class" under that one account (the
// same pattern used by every loyalty-as-a-service competitor: the platform
// owns one developer account, not each of its customers).
type GoogleWalletConfig struct {
	IssuerID            string
	ServiceAccountEmail string
	PrivateKeyPEM       string
	// ReviewStatus must be "UNDER_REVIEW" until Google approves the issuer
	// account for production, then flipped to "APPROVED" — see
	// https://developers.google.com/wallet/retail/loyalty-cards/getting-started/issuer-onboarding
	ReviewStatus string
}

// GoogleWalletService builds "Save to Google Wallet" links for loyalty cards
// and pushes stamp-count updates to already-saved passes. It degrades to a
// harmless no-op (Enabled() == false) when no credentials are configured,
// so the rest of the loyalty flow never has to special-case its absence.
type GoogleWalletService struct {
	cfg        GoogleWalletConfig
	privateKey *rsa.PrivateKey
	publicBase string
	httpClient *http.Client
}

func NewGoogleWalletService(cfg GoogleWalletConfig, publicBaseURL string) (*GoogleWalletService, error) {
	s := &GoogleWalletService{cfg: cfg, publicBase: publicBaseURL, httpClient: &http.Client{Timeout: 10 * time.Second}}
	if cfg.IssuerID == "" || cfg.ServiceAccountEmail == "" || cfg.PrivateKeyPEM == "" {
		return s, nil
	}
	// Service account keys downloaded as JSON store the PEM with literal
	// "\n" sequences once flattened into a single env var line.
	pem := strings.ReplaceAll(cfg.PrivateKeyPEM, `\n`, "\n")
	key, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(pem))
	if err != nil {
		return nil, fmt.Errorf("google wallet: invalid private key: %w", err)
	}
	s.privateKey = key
	return s, nil
}

func (s *GoogleWalletService) Enabled() bool {
	return s.privateKey != nil
}

func (s *GoogleWalletService) loyaltyClassID(programID string) string {
	return fmt.Sprintf("%s.loyalty_%s", s.cfg.IssuerID, programID)
}

func (s *GoogleWalletService) loyaltyObjectID(customerID string) string {
	return fmt.Sprintf("%s.customer_%s", s.cfg.IssuerID, customerID)
}

// LoyaltyWalletInput is everything BuildSaveURL needs to describe one
// customer's card — assembled by the caller from Profile/Theme/LoyaltyProgram
// so this service stays free of repository dependencies.
type LoyaltyWalletInput struct {
	Program      *models.LoyaltyProgram
	BusinessName string
	LogoURL      string // absolute HTTPS URL — Google's servers fetch it directly
	ProfileURL   string
	LoyaltyURL   string
	PrimaryColor string // hex, e.g. "#4f46e5"
	Customer     *models.LoyaltyCustomer
}

// BuildSaveURL returns a https://pay.google.com/gp/v/save/<jwt> link that
// hands the browser a "Save to Google Wallet" flow for this one customer.
// The loyalty class (the business's card design) and object (this
// customer's current stamp count) are embedded directly in the JWT — Google
// creates both server-side the first time the customer taps save, so no
// REST provisioning call is needed up front.
func (s *GoogleWalletService) BuildSaveURL(in LoyaltyWalletInput) (string, error) {
	if !s.Enabled() {
		return "", ErrGoogleWalletNotConfigured
	}
	if in.LogoURL == "" {
		return "", ErrGoogleWalletNoLogo
	}

	classID := s.loyaltyClassID(in.Program.ID)
	objectID := s.loyaltyObjectID(in.Customer.ID)

	rewardLabel := "Premio"
	rewardValue := "Completa tu tarjeta"
	if in.Program.RewardDescription != nil && *in.Program.RewardDescription != "" {
		rewardValue = *in.Program.RewardDescription
	}

	loyaltyClass := map[string]any{
		"id":          classID,
		"issuerName":  in.BusinessName,
		"programName": "Tarjeta de sellos",
		"programLogo": map[string]any{
			"sourceUri": map[string]any{"uri": in.LogoURL},
			"contentDescription": map[string]any{
				"defaultValue": map[string]any{"language": "es", "value": in.BusinessName},
			},
		},
		"hexBackgroundColor": in.PrimaryColor,
		"reviewStatus":       s.cfg.ReviewStatus,
		"homepageUri":        map[string]any{"uri": in.ProfileURL},
	}

	loyaltyObject := map[string]any{
		"id":          objectID,
		"classId":     classID,
		"state":       "ACTIVE",
		"accountId":   in.Customer.ID,
		"accountName": in.Customer.FullName,
		"loyaltyPoints": map[string]any{
			"label": "Sellos",
			"balance": map[string]any{
				"int": in.Customer.StampsCount,
			},
		},
		"textModulesData": []any{
			map[string]any{"id": "reward", "header": rewardLabel, "body": rewardValue},
		},
		"barcode": map[string]any{
			"type":  "QR_CODE",
			"value": in.LoyaltyURL,
		},
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"iss":     s.cfg.ServiceAccountEmail,
		"aud":     "google",
		"typ":     "savetowallet",
		"iat":     now.Unix(),
		"origins": []string{s.publicBase},
		"payload": map[string]any{
			"loyaltyClasses": []any{loyaltyClass},
			"loyaltyObjects": []any{loyaltyObject},
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(s.privateKey)
	if err != nil {
		return "", err
	}
	return "https://pay.google.com/gp/v/save/" + signed, nil
}

// SyncStamps pushes an already-saved pass's stamp count to Google so the
// wallet updates without the customer having to re-save it. It is
// deliberately best-effort: a customer who never tapped "save" simply
// doesn't have an object yet, and that 404 is not an error worth surfacing —
// stamping must always succeed locally regardless of Wallet sync.
func (s *GoogleWalletService) SyncStamps(ctx context.Context, customerID string, stamps int) error {
	if !s.Enabled() {
		return ErrGoogleWalletNotConfigured
	}

	token, err := s.accessToken(ctx)
	if err != nil {
		return err
	}

	objectID := s.loyaltyObjectID(customerID)
	body, err := json.Marshal(map[string]any{
		"loyaltyPoints": map[string]any{
			"balance": map[string]any{"int": stamps},
		},
	})
	if err != nil {
		return err
	}

	endpoint := "https://walletobjects.googleapis.com/walletobjects/v1/loyaltyObject/" + url.PathEscape(objectID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("google wallet sync failed: %s", resp.Status)
	}
	return nil
}

// accessToken exchanges a self-signed JWT assertion for a short-lived OAuth2
// bearer token, following the standard service-account "JWT bearer grant"
// flow — done by hand here (rather than pulling in golang.org/x/oauth2) since
// it's a dozen lines on top of the RS256 signer BuildSaveURL already needs.
func (s *GoogleWalletService) accessToken(ctx context.Context) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"iss":   s.cfg.ServiceAccountEmail,
		"scope": "https://www.googleapis.com/auth/wallet_object.issuer",
		"aud":   "https://oauth2.googleapis.com/token",
		"iat":   now.Unix(),
		"exp":   now.Add(time.Hour).Unix(),
	}
	assertion, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(s.privateKey)
	if err != nil {
		return "", err
	}

	form := url.Values{}
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:jwt-bearer")
	form.Set("assertion", assertion)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://oauth2.googleapis.com/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("google oauth token exchange failed: %s", resp.Status)
	}

	var out struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	return out.AccessToken, nil
}
