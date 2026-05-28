package secrets

import (
	"errors"
	"os"

	"github.com/zalando/go-keyring"
)

const service = "previously-on"

var ErrNotFound = errors.New("secret not found")

type Store interface {
	Get(provider string) (string, error)
	Set(provider, value string) error
	Delete(provider string) error
}

type KeyringStore struct{}

func (KeyringStore) Get(provider string) (string, error) {
	value, err := keyring.Get(service, provider)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", ErrNotFound
	}
	return value, err
}

func (KeyringStore) Set(provider, value string) error {
	return keyring.Set(service, provider, value)
}

func (KeyringStore) Delete(provider string) error {
	if err := keyring.Delete(service, provider); errors.Is(err, keyring.ErrNotFound) {
		return nil
	} else {
		return err
	}
}

func Resolve(store Store, provider, envVar string) (string, string, error) {
	if envVar != "" {
		if value := os.Getenv(envVar); value != "" {
			return value, "env", nil
		}
	}
	value, err := store.Get(provider)
	if err != nil {
		return "", "", err
	}
	return value, "keychain", nil
}
