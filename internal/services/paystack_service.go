package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

type PaystackClient struct {
	SecretKey string
	BaseURL   string
	Client    *http.Client
}

func NewPaystackClient() (*PaystackClient, error) {
	secretKey := os.Getenv("PAYSTACK_SECRET_KEY")
	if secretKey == "" {
		return nil, fmt.Errorf("PAYSTACK_SECRET_KEY environment variable is not set")
	}
	return &PaystackClient{
		SecretKey: secretKey,
		BaseURL:   "https://api.paystack.co",
		Client:    &http.Client{Timeout: 30 * time.Second},
	}, nil
}

type PaystackResponse struct {
	Status  bool        `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func (p *PaystackClient) doRequest(method, endpoint string, body interface{}) (*PaystackResponse, error) {
	endpointURL := fmt.Sprintf("%s%s", p.BaseURL, endpoint)
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(data)
	}

	req, err := http.NewRequest(method, endpointURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Add("Authorization", "Bearer "+p.SecretKey)
	req.Header.Add("Content-Type", "application/json")

	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(respBody))
	}

	var res PaystackResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode response (status %d): %w", resp.StatusCode, err)
	}

	if !res.Status {
		return nil, fmt.Errorf("API error: %s", res.Message)
	}

	return &res, nil
}

func (p *PaystackClient) InitializePayment(form map[string]interface{}) (*PaystackResponse, error) {
	requiredFields := []string{"amount", "email"}
	for _, field := range requiredFields {
		if _, ok := form[field]; !ok {
			return nil, fmt.Errorf("missing required field: %s", field)
		}
	}
	return p.doRequest("POST", "/transaction/initialize", form)
}

func (p *PaystackClient) VerifyPayment(ref string) (*PaystackResponse, error) {
	if ref == "" {
		return nil, fmt.Errorf("reference cannot be empty")
	}
	escapedRef := url.PathEscape(ref)
	return p.doRequest("GET", fmt.Sprintf("/transaction/verify/%s", escapedRef), nil)
}

func (p *PaystackClient) CreateRecipient(form map[string]interface{}) (*PaystackResponse, error) {
	requiredFields := []string{"type", "name", "account_number", "bank_code"}
	for _, field := range requiredFields {
		if _, ok := form[field]; !ok {
			return nil, fmt.Errorf("missing required field: %s", field)
		}
	}
	return p.doRequest("POST", "/transferrecipient", form)
}

func (p *PaystackClient) InitiateTransfer(form map[string]interface{}) (*PaystackResponse, error) {
	requiredFields := []string{"source", "amount", "recipient"}
	for _, field := range requiredFields {
		if _, ok := form[field]; !ok {
			return nil, fmt.Errorf("missing required field: %s", field)
		}
	}
	return p.doRequest("POST", "/transfer", form)
}
