package acme

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"

	"github.com/go-acme/lego/v3/lego"
	"github.com/go-acme/lego/v3/registration"
	"github.com/hashicorp/vault/sdk/logical"
)

type account struct {
	Email                string
	Registration         *registration.Resource
	Key                  *ecdsa.PrivateKey
	ServerURL            string
	Provider             string
	EnableHTTP01         bool
	EnableTLSALPN01      bool
	TermsOfServiceAgreed bool
}

// GetEmail returns the Email of the user
func (a *account) GetEmail() string {
	return a.Email
}

// GetRegistration returns the Email of the user
func (a *account) GetRegistration() *registration.Resource {
	return a.Registration
}

// GetPrivateKey returns the private key of the user
func (a *account) GetPrivateKey() crypto.PrivateKey {
	return a.Key
}

func (a *account) getClient() (*lego.Client, error) {
	config := lego.NewConfig(a)
	config.CADirURL = a.ServerURL

	return lego.NewClient(config)
}

func getAccount(ctx context.Context, storage logical.Storage, path string) (*account, error) {
	storageEntry, err := storage.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	if storageEntry == nil {
		return nil, nil
	}
	var d map[string]interface{}
	if err = storageEntry.DecodeJSON(&d); err != nil {
		return nil, err
	}

	block, _ := pem.Decode([]byte(d["private_key"].(string)))
	privateKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	return &account{
		Email: d["contact"].(string),
		Key:   privateKey,
		Registration: &registration.Resource{
			URI: d["registration_uri"].(string),
		},
		ServerURL:            d["server_url"].(string),
		Provider:             d["provider"].(string),
		TermsOfServiceAgreed: d["terms_of_service_agreed"].(bool),
		EnableHTTP01:         d["enable_http_01"].(bool),
		EnableTLSALPN01:      d["enable_tls_alpn_01"].(bool),
	}, nil
}

func (a *account) save(ctx context.Context, storage logical.Storage, path string, serverURL string) error {
	x509Encoded, err := x509.MarshalECPrivateKey(a.Key)
	if err != nil {
		return err
	}
	pemEncoded := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: x509Encoded})

	storageEntry, err := logical.StorageEntryJSON(path, map[string]interface{}{
		"server_url":              serverURL,
		"registration_uri":        a.Registration.URI,
		"contact":                 a.GetEmail(),
		"terms_of_service_agreed": a.TermsOfServiceAgreed,
		"private_key":             string(pemEncoded),
		"provider":                a.Provider,
		"enable_http_01":          a.EnableHTTP01,
		"enable_tls_alpn_01":      a.EnableTLSALPN01,
	})
	if err != nil {
		return err
	}

	return storage.Put(ctx, storageEntry)
}
