package billing

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Fake is an in-memory Provider for tests.
type Fake struct {
	mu            sync.Mutex
	Customers     map[string]Customer
	Payments      map[string]Payment
	Subscriptions map[string]Subscription
	FailNext      error
	nextID        int
}

func NewFake() *Fake {
	return &Fake{
		Customers:     map[string]Customer{},
		Payments:      map[string]Payment{},
		Subscriptions: map[string]Subscription{},
	}
}

func (f *Fake) id(prefix string) string {
	f.nextID++
	return fmt.Sprintf("%s_%d", prefix, f.nextID)
}

func (f *Fake) CreateCustomer(_ context.Context, email, name string) (Customer, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.FailNext != nil {
		err := f.FailNext
		f.FailNext = nil
		return Customer{}, err
	}
	c := Customer{ID: f.id("cst"), Email: email, Name: name}
	f.Customers[c.ID] = c
	return c, nil
}

func (f *Fake) CreateFirstPayment(_ context.Context, in CreatePaymentInput) (Payment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.FailNext != nil {
		err := f.FailNext
		f.FailNext = nil
		return Payment{}, err
	}
	p := Payment{
		ID:           f.id("tr"),
		Status:       "open",
		CheckoutURL:  "https://www.mollie.com/checkout/test/" + time.Now().UTC().Format("150405"),
		CustomerID:   in.CustomerID,
		SequenceType: "first",
		Metadata:     in.Metadata,
	}
	f.Payments[p.ID] = p
	return p, nil
}

func (f *Fake) GetPayment(_ context.Context, paymentID string) (Payment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.Payments[paymentID]
	if !ok {
		return Payment{}, fmt.Errorf("payment not found")
	}
	return p, nil
}

func (f *Fake) MarkPaid(paymentID, mandateID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p := f.Payments[paymentID]
	p.Status = "paid"
	p.MandateID = mandateID
	if p.MandateID == "" {
		p.MandateID = f.id("mdt")
	}
	f.Payments[paymentID] = p
}

func (f *Fake) MarkFailed(paymentID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p := f.Payments[paymentID]
	p.Status = "failed"
	f.Payments[paymentID] = p
}

func (f *Fake) CreateSubscription(_ context.Context, in CreateSubscriptionInput) (Subscription, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.FailNext != nil {
		err := f.FailNext
		f.FailNext = nil
		return Subscription{}, err
	}
	sub := Subscription{
		ID:       f.id("sub"),
		Status:   "active",
		Amount:   FormatEUR(in.AmountCents),
		Interval: in.Interval,
	}
	f.Subscriptions[sub.ID] = sub
	return sub, nil
}

func (f *Fake) UpdateSubscriptionAmount(_ context.Context, _, subscriptionID string, amountCents int, _ string) (Subscription, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	sub, ok := f.Subscriptions[subscriptionID]
	if !ok {
		return Subscription{}, fmt.Errorf("subscription not found")
	}
	sub.Amount = FormatEUR(amountCents)
	f.Subscriptions[subscriptionID] = sub
	return sub, nil
}

func (f *Fake) CancelSubscription(_ context.Context, _, subscriptionID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	sub, ok := f.Subscriptions[subscriptionID]
	if !ok {
		return fmt.Errorf("subscription not found")
	}
	sub.Status = "canceled"
	f.Subscriptions[subscriptionID] = sub
	return nil
}
