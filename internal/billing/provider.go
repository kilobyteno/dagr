package billing

import "context"

// Customer is a Mollie customer bound to a workspace.
type Customer struct {
	ID    string
	Email string
	Name  string
}

// Payment is a Mollie payment (first checkout or recurring).
type Payment struct {
	ID             string
	Status         string
	CheckoutURL    string
	CustomerID     string
	MandateID      string
	SubscriptionID string
	SequenceType   string
	Metadata       map[string]string
}

// Subscription is a Mollie recurring subscription.
type Subscription struct {
	ID       string
	Status   string
	Amount   string
	Interval string
}

// CreatePaymentInput starts hosted checkout for a first recurring payment.
type CreatePaymentInput struct {
	CustomerID  string
	AmountCents int
	Currency    string
	Description string
	RedirectURL string
	WebhookURL  string
	Metadata    map[string]string
}

// CreateSubscriptionInput schedules recurring charges after a mandate exists.
type CreateSubscriptionInput struct {
	CustomerID  string
	AmountCents int
	Currency    string
	Interval    string
	StartDate   string
	Description string
	WebhookURL  string
	Metadata    map[string]string
}

// Provider is the payment service used on Cloud. Tests use a fake.
type Provider interface {
	CreateCustomer(ctx context.Context, email, name string) (Customer, error)
	CreateFirstPayment(ctx context.Context, in CreatePaymentInput) (Payment, error)
	GetPayment(ctx context.Context, paymentID string) (Payment, error)
	CreateSubscription(ctx context.Context, in CreateSubscriptionInput) (Subscription, error)
	UpdateSubscriptionAmount(ctx context.Context, customerID, subscriptionID string, amountCents int, currency string) (Subscription, error)
	CancelSubscription(ctx context.Context, customerID, subscriptionID string) error
}
