// Package crypto defines extension points for optional end-to-end encryption of DMs.
//
// The MVP ships plaintext messages. Implement EnvelopeEncryptor and wire it into
// the message service when E2EE is enabled per conversation.
package crypto

import "context"

// Envelope holds ciphertext and associated metadata for an E2EE payload.
type Envelope struct {
	Ciphertext  []byte
	ContentType string
}

// EnvelopeEncryptor encrypts and decrypts message payloads for E2EE DMs.
type EnvelopeEncryptor interface {
	Encrypt(ctx context.Context, plaintext []byte) (Envelope, error)
	Decrypt(ctx context.Context, env Envelope) ([]byte, error)
}

// Passthrough leaves payloads unencrypted. Used for the plaintext MVP.
type Passthrough struct{}

func (Passthrough) Encrypt(_ context.Context, plaintext []byte) (Envelope, error) {
	return Envelope{
		Ciphertext:  plaintext,
		ContentType: "text/plain",
	}, nil
}

func (Passthrough) Decrypt(_ context.Context, env Envelope) ([]byte, error) {
	return env.Ciphertext, nil
}
