package billing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const mollieAPIBase = "https://api.mollie.com/v2"

// Mollie is a thin client for the Mollie payments and subscriptions APIs.
type Mollie struct {
	APIKey     string
	HTTPClient *http.Client
	BaseURL    string
}

func (m *Mollie) client() *http.Client {
	if m.HTTPClient != nil {
		return m.HTTPClient
	}
	return &http.Client{Timeout: 20 * time.Second}
}

func (m *Mollie) base() string {
	if strings.TrimSpace(m.BaseURL) != "" {
		return strings.TrimRight(m.BaseURL, "/")
	}
	return mollieAPIBase
}

type mollieAmount struct {
	Currency string `json:"currency"`
	Value    string `json:"value"`
}

type mollieCustomerReq struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
}

type mollieCustomerRes struct {
	ID string `json:"id"`
}

type molliePaymentReq struct {
	Amount       mollieAmount      `json:"amount"`
	Description  string            `json:"description"`
	RedirectURL  string            `json:"redirectUrl"`
	WebhookURL   string            `json:"webhookUrl,omitempty"`
	CustomerID   string            `json:"customerId"`
	SequenceType string            `json:"sequenceType"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

type molliePaymentRes struct {
	ID             string            `json:"id"`
	Status         string            `json:"status"`
	CustomerID     string            `json:"customerId"`
	MandateID      string            `json:"mandateId"`
	SubscriptionID string            `json:"subscriptionId"`
	SequenceType   string            `json:"sequenceType"`
	Metadata       map[string]string `json:"metadata"`
	Links          struct {
		Checkout struct {
			Href string `json:"href"`
		} `json:"checkout"`
	} `json:"_links"`
}

type mollieSubscriptionReq struct {
	Amount      mollieAmount      `json:"amount"`
	Interval    string            `json:"interval"`
	StartDate   string            `json:"startDate,omitempty"`
	Description string            `json:"description"`
	WebhookURL  string            `json:"webhookUrl,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type mollieSubscriptionRes struct {
	ID       string       `json:"id"`
	Status   string       `json:"status"`
	Amount   mollieAmount `json:"amount"`
	Interval string       `json:"interval"`
}

type mollieAmountPatch struct {
	Amount mollieAmount `json:"amount"`
}

func (m *Mollie) CreateCustomer(ctx context.Context, email, name string) (Customer, error) {
	var res mollieCustomerRes
	if err := m.do(ctx, http.MethodPost, "/customers", mollieCustomerReq{Name: name, Email: email}, &res); err != nil {
		return Customer{}, err
	}
	return Customer{ID: res.ID, Email: email, Name: name}, nil
}

func (m *Mollie) CreateFirstPayment(ctx context.Context, in CreatePaymentInput) (Payment, error) {
	var res molliePaymentRes
	err := m.do(ctx, http.MethodPost, "/payments", molliePaymentReq{
		Amount:       mollieAmount{Currency: in.Currency, Value: FormatEUR(in.AmountCents)},
		Description:  in.Description,
		RedirectURL:  in.RedirectURL,
		WebhookURL:   in.WebhookURL,
		CustomerID:   in.CustomerID,
		SequenceType: "first",
		Metadata:     in.Metadata,
	}, &res)
	if err != nil {
		return Payment{}, err
	}
	return paymentFromMollie(res), nil
}

func (m *Mollie) GetPayment(ctx context.Context, paymentID string) (Payment, error) {
	var res molliePaymentRes
	if err := m.do(ctx, http.MethodGet, "/payments/"+paymentID, nil, &res); err != nil {
		return Payment{}, err
	}
	return paymentFromMollie(res), nil
}

func (m *Mollie) CreateSubscription(ctx context.Context, in CreateSubscriptionInput) (Subscription, error) {
	var res mollieSubscriptionRes
	err := m.do(ctx, http.MethodPost, "/customers/"+in.CustomerID+"/subscriptions", mollieSubscriptionReq{
		Amount:      mollieAmount{Currency: in.Currency, Value: FormatEUR(in.AmountCents)},
		Interval:    in.Interval,
		StartDate:   in.StartDate,
		Description: in.Description,
		WebhookURL:  in.WebhookURL,
		Metadata:    in.Metadata,
	}, &res)
	if err != nil {
		return Subscription{}, err
	}
	return Subscription{ID: res.ID, Status: res.Status, Amount: res.Amount.Value, Interval: res.Interval}, nil
}

func (m *Mollie) UpdateSubscriptionAmount(
	ctx context.Context, customerID, subscriptionID string, amountCents int, currency string,
) (Subscription, error) {
	var res mollieSubscriptionRes
	err := m.do(ctx, http.MethodPatch, "/customers/"+customerID+"/subscriptions/"+subscriptionID, mollieAmountPatch{
		Amount: mollieAmount{Currency: currency, Value: FormatEUR(amountCents)},
	}, &res)
	if err != nil {
		return Subscription{}, err
	}
	return Subscription{ID: res.ID, Status: res.Status, Amount: res.Amount.Value, Interval: res.Interval}, nil
}

func (m *Mollie) CancelSubscription(ctx context.Context, customerID, subscriptionID string) error {
	return m.do(ctx, http.MethodDelete, "/customers/"+customerID+"/subscriptions/"+subscriptionID, nil, nil)
}

func paymentFromMollie(res molliePaymentRes) Payment {
	return Payment{
		ID:             res.ID,
		Status:         res.Status,
		CheckoutURL:    res.Links.Checkout.Href,
		CustomerID:     res.CustomerID,
		MandateID:      res.MandateID,
		SubscriptionID: res.SubscriptionID,
		SequenceType:   res.SequenceType,
		Metadata:       res.Metadata,
	}
}

func (m *Mollie) do(ctx context.Context, method, path string, body any, out any) error {
	if strings.TrimSpace(m.APIKey) == "" {
		return fmt.Errorf("mollie api key is not configured")
	}
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, m.base()+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+m.APIKey)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := m.client().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("mollie %s %s: %s", method, path, strings.TrimSpace(string(payload)))
	}
	if out == nil || len(payload) == 0 {
		return nil
	}
	return json.Unmarshal(payload, out)
}
