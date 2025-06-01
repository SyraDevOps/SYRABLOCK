package main

import "time"

type Transaction struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Amount    int       `json:"amount"`
	Timestamp time.Time `json:"timestamp"`
	Contract  string    `json:"contract,omitempty"`
	PublicKey string    `json:"public_key,omitempty"`
	Nonce     int       `json:"nonce,omitempty"`
	Hash      string    `json:"hash,omitempty"`
	Signature string    `json:"signature,omitempty"`
}
