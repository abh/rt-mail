package ses

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"go.askask.com/rt-mail/rt"
)

// certCacheEntry holds a cached certificate with expiration.
type certCacheEntry struct {
	cert      *x509.Certificate
	expiresAt time.Time
}

// certCache caches SNS signing certificates by URL.
var (
	certCache    = make(map[string]certCacheEntry)
	certCacheMu  sync.RWMutex
	certCacheTTL = 1 * time.Hour
)

// snsHostPattern validates SNS signing certificate URLs.
// Must be sns.<region>.amazonaws.com
var snsHostPattern = regexp.MustCompile(`^sns\.[a-z0-9-]+\.amazonaws\.com$`)

// maxEmailSize is the maximum email size to fetch from S3 (50MB).
const maxEmailSize = 50 * 1024 * 1024

// SNSMessage represents an AWS SNS message envelope.
type SNSMessage struct {
	Type             string `json:"Type"`
	MessageID        string `json:"MessageId"`
	TopicArn         string `json:"TopicArn"`
	Subject          string `json:"Subject,omitempty"`
	Message          string `json:"Message"`
	Timestamp        string `json:"Timestamp"`
	SignatureVersion string `json:"SignatureVersion"`
	Signature        string `json:"Signature"`
	SigningCertURL   string `json:"SigningCertURL"`
	SubscribeURL     string `json:"SubscribeURL,omitempty"`
	Token            string `json:"Token,omitempty"`
	UnsubscribeURL   string `json:"UnsubscribeURL,omitempty"`
}

// SESNotification represents the SES notification payload inside the SNS Message.
type SESNotification struct {
	NotificationType string `json:"notificationType"`
	Receipt          struct {
		Action struct {
			Type       string `json:"type"`
			BucketName string `json:"bucketName"`
			ObjectKey  string `json:"objectKey"`
		} `json:"action"`
		Recipients []string `json:"recipients"`
	} `json:"receipt"`
	Mail struct {
		MessageID   string   `json:"messageId"`
		Source      string   `json:"source"`
		Destination []string `json:"destination"`
	} `json:"mail"`
}

// SES handles AWS SES webhook requests via SNS.
type SES struct {
	RT         *rt.RT
	TopicARN   string
	S3Client   *s3.Client
	httpClient *http.Client
}

// New creates a new SES webhook handler.
func New(rtClient *rt.RT, topicARN string) (*SES, error) {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}

	return &SES{
		RT:       rtClient,
		TopicARN: topicARN,
		S3Client: s3.NewFromConfig(cfg),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}, nil
}

// GetRoutes returns the REST routes for the SES handler.
func (s *SES) GetRoutes() []*rest.Route {
	return []*rest.Route{
		rest.Post("/ses", s.Handler),
	}
}

// Handler processes an SNS POST request containing SES events.
func (s *SES) Handler(w rest.ResponseWriter, r *rest.Request) {
	ctx := r.Context()

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("SES: failed to read request body: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse SNS envelope
	var msg SNSMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		log.Printf("SES: invalid JSON payload: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Verify SNS signature
	if err := s.verifySignature(ctx, &msg); err != nil {
		log.Printf("SES: signature verification failed: %s", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Validate TopicArn
	if msg.TopicArn != s.TopicARN {
		log.Printf("SES: TopicArn mismatch: got %q, expected %q", msg.TopicArn, s.TopicARN)
		w.WriteHeader(http.StatusForbidden)
		return
	}

	// Handle message type
	switch msg.Type {
	case "SubscriptionConfirmation":
		s.handleSubscriptionConfirmation(ctx, w, &msg)
	case "Notification":
		s.handleNotification(ctx, w, &msg)
	case "UnsubscribeConfirmation":
		log.Printf("SES: received unsubscribe confirmation for topic %s", msg.TopicArn)
		w.WriteHeader(http.StatusOK)
	default:
		log.Printf("SES: unknown message type: %s", msg.Type)
		w.WriteHeader(http.StatusBadRequest)
	}
}

// handleSubscriptionConfirmation auto-confirms SNS subscription.
func (s *SES) handleSubscriptionConfirmation(ctx context.Context, w rest.ResponseWriter, msg *SNSMessage) {
	if msg.SubscribeURL == "" {
		log.Printf("SES: subscription confirmation missing SubscribeURL")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Validate SubscribeURL is from AWS SNS (prevent SSRF)
	subscribeURL, err := url.Parse(msg.SubscribeURL)
	if err != nil {
		log.Printf("SES: invalid SubscribeURL: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if subscribeURL.Scheme != "https" {
		log.Printf("SES: SubscribeURL must use HTTPS: %s", msg.SubscribeURL)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if !snsHostPattern.MatchString(subscribeURL.Host) {
		log.Printf("SES: SubscribeURL host not valid SNS endpoint: %s", subscribeURL.Host)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Fetch the SubscribeURL to confirm subscription
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, msg.SubscribeURL, nil)
	if err != nil {
		log.Printf("SES: failed to create confirmation request: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("SES: failed to confirm subscription: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("SES: subscription confirmation failed with status %d", resp.StatusCode)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Printf("SES: subscription confirmed for topic %s", msg.TopicArn)
	w.WriteHeader(http.StatusOK)
}

// handleNotification processes an SES email notification.
func (s *SES) handleNotification(ctx context.Context, w rest.ResponseWriter, msg *SNSMessage) {
	// Parse the inner SES notification
	var sesNotif SESNotification
	if err := json.Unmarshal([]byte(msg.Message), &sesNotif); err != nil {
		log.Printf("SES: failed to parse SES notification: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Verify this is a received email notification with S3 action
	if sesNotif.NotificationType != "Received" {
		log.Printf("SES: ignoring notification type %q (expected Received)", sesNotif.NotificationType)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if sesNotif.Receipt.Action.Type != "S3" {
		log.Printf("SES: ignoring action type %q (expected S3)", sesNotif.Receipt.Action.Type)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Fetch email from S3
	bucket := sesNotif.Receipt.Action.BucketName
	key := sesNotif.Receipt.Action.ObjectKey
	if bucket == "" || key == "" {
		log.Printf("SES: missing S3 bucket or key in notification")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	rawEmail, err := s.fetchEmailFromS3(ctx, bucket, key)
	if err != nil {
		log.Printf("SES: failed to fetch email from S3 (bucket=%s, key=%s): %s", bucket, key, err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	// Post to RT for each recipient
	recipients := sesNotif.Receipt.Recipients
	if len(recipients) == 0 {
		log.Printf("SES: no recipients in notification")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var lastErr error
	var notFoundCount int
	for _, recipient := range recipients {
		err := s.RT.Postmail(recipient, string(rawEmail))
		if err != nil {
			log.Printf("SES: post error for recipient %s: %s", recipient, err)
			if rtErr, ok := err.(*rt.Error); ok && rtErr.NotFound {
				notFoundCount++
				continue
			}
			lastErr = err
		}
	}

	// If all recipients resulted in not found, return 404
	if notFoundCount == len(recipients) {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// If any non-404 error occurred, return 503 to trigger SNS retry
	if lastErr != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// fetchEmailFromS3 retrieves the raw email content from S3.
func (s *SES) fetchEmailFromS3(ctx context.Context, bucket, key string) ([]byte, error) {
	resp, err := s.S3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})
	if err != nil {
		return nil, fmt.Errorf("S3 GetObject: %w", err)
	}
	defer resp.Body.Close()

	// Limit read size to prevent memory exhaustion
	limitedReader := io.LimitReader(resp.Body, maxEmailSize+1)
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("read S3 object: %w", err)
	}

	if len(data) > maxEmailSize {
		return nil, fmt.Errorf("email exceeds size limit (%d bytes)", maxEmailSize)
	}

	return data, nil
}

// verifySignature validates the SNS message signature.
func (s *SES) verifySignature(ctx context.Context, msg *SNSMessage) error {
	// Validate SigningCertURL is from AWS SNS
	certURL, err := url.Parse(msg.SigningCertURL)
	if err != nil {
		return fmt.Errorf("invalid SigningCertURL: %w", err)
	}
	if certURL.Scheme != "https" {
		return fmt.Errorf("SigningCertURL must use HTTPS")
	}
	if !snsHostPattern.MatchString(certURL.Host) {
		return fmt.Errorf("SigningCertURL host not valid SNS endpoint: %s", certURL.Host)
	}

	// Get certificate (from cache or fetch)
	cert, err := s.getCertificate(ctx, msg.SigningCertURL)
	if err != nil {
		return err
	}

	// Build canonical string based on message type
	canonicalString := s.buildCanonicalString(msg)

	// Decode signature
	signature, err := base64.StdEncoding.DecodeString(msg.Signature)
	if err != nil {
		return fmt.Errorf("decode signature: %w", err)
	}

	// Verify signature based on SignatureVersion
	var algo x509.SignatureAlgorithm
	switch msg.SignatureVersion {
	case "1":
		algo = x509.SHA1WithRSA
	case "2":
		algo = x509.SHA256WithRSA
	default:
		return fmt.Errorf("unsupported SignatureVersion: %s", msg.SignatureVersion)
	}

	if err := cert.CheckSignature(algo, []byte(canonicalString), signature); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	return nil
}

// getCertificate retrieves a certificate from cache or fetches it from the URL.
func (s *SES) getCertificate(ctx context.Context, certURL string) (*x509.Certificate, error) {
	now := time.Now()

	// Check cache first (read lock)
	certCacheMu.RLock()
	entry, ok := certCache[certURL]
	certCacheMu.RUnlock()

	if ok && now.Before(entry.expiresAt) {
		return entry.cert, nil
	}

	// Acquire write lock and double-check to prevent thundering herd
	certCacheMu.Lock()
	defer certCacheMu.Unlock()

	// Re-check cache after acquiring write lock
	entry, ok = certCache[certURL]
	if ok && now.Before(entry.expiresAt) {
		return entry.cert, nil
	}

	// Fetch certificate
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, certURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create cert request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch certificate: %w", err)
	}
	defer resp.Body.Close()

	certPEM, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read certificate: %w", err)
	}

	// Parse certificate
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM certificate")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse certificate: %w", err)
	}

	// Validate certificate is currently valid
	if now.Before(cert.NotBefore) || now.After(cert.NotAfter) {
		return nil, fmt.Errorf("certificate not valid (NotBefore: %s, NotAfter: %s)",
			cert.NotBefore, cert.NotAfter)
	}

	// Verify certificate is from Amazon
	if !strings.Contains(cert.Subject.CommonName, "Amazon") &&
		!strings.Contains(cert.Issuer.CommonName, "Amazon") {
		return nil, fmt.Errorf("certificate not issued by Amazon")
	}

	// Cache the certificate
	// Use the shorter of: cache TTL or certificate expiry
	expiresAt := now.Add(certCacheTTL)
	if cert.NotAfter.Before(expiresAt) {
		expiresAt = cert.NotAfter
	}

	certCache[certURL] = certCacheEntry{
		cert:      cert,
		expiresAt: expiresAt,
	}

	return cert, nil
}

// buildCanonicalString creates the string to sign for SNS signature verification.
func (s *SES) buildCanonicalString(msg *SNSMessage) string {
	var sb strings.Builder

	// Fields must be in specific order
	sb.WriteString("Message\n")
	sb.WriteString(msg.Message)
	sb.WriteString("\n")
	sb.WriteString("MessageId\n")
	sb.WriteString(msg.MessageID)
	sb.WriteString("\n")

	// Subject only for Notification type with non-empty subject
	if msg.Type == "Notification" && msg.Subject != "" {
		sb.WriteString("Subject\n")
		sb.WriteString(msg.Subject)
		sb.WriteString("\n")
	}

	// SubscribeURL only for SubscriptionConfirmation and UnsubscribeConfirmation
	if msg.Type == "SubscriptionConfirmation" || msg.Type == "UnsubscribeConfirmation" {
		sb.WriteString("SubscribeURL\n")
		sb.WriteString(msg.SubscribeURL)
		sb.WriteString("\n")
	}

	sb.WriteString("Timestamp\n")
	sb.WriteString(msg.Timestamp)
	sb.WriteString("\n")

	// Token only for SubscriptionConfirmation and UnsubscribeConfirmation
	if msg.Type == "SubscriptionConfirmation" || msg.Type == "UnsubscribeConfirmation" {
		sb.WriteString("Token\n")
		sb.WriteString(msg.Token)
		sb.WriteString("\n")
	}

	sb.WriteString("TopicArn\n")
	sb.WriteString(msg.TopicArn)
	sb.WriteString("\n")
	sb.WriteString("Type\n")
	sb.WriteString(msg.Type)
	sb.WriteString("\n")

	return sb.String()
}
