package credential

import (
	"errors"
	"strings"
	"testing"

	keyring "github.com/zalando/go-keyring"
)

type fakeBackend struct {
	values    map[string]string
	getErr    error
	setErr    error
	deleteErr error
}

func (f *fakeBackend) key(service, account string) string { return service + "|" + account }

func (f *fakeBackend) Get(service, account string) (string, error) {
	if f.getErr != nil {
		return "", f.getErr
	}
	value, ok := f.values[f.key(service, account)]
	if !ok {
		return "", keyring.ErrNotFound
	}
	return value, nil
}

func (f *fakeBackend) Set(service, account, value string) error {
	if f.setErr != nil {
		return f.setErr
	}
	f.values[f.key(service, account)] = value
	return nil
}

func (f *fakeBackend) Delete(service, account string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	key := f.key(service, account)
	if _, ok := f.values[key]; !ok {
		return keyring.ErrNotFound
	}
	delete(f.values, key)
	return nil
}

func TestKeyringStoreRoundTrip(t *testing.T) {
	backend := &fakeBackend{values: map[string]string{}}
	store := newKeyringStore(backend)
	account := Account("local", "https://example.test/api/v1/")
	want := Tokens{AuthToken: "auth", RefreshToken: "refresh"}
	if err := store.Set(account, want); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(account)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("tokens = %#v, want %#v", got, want)
	}
	if err := store.Delete(account); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(account); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get() error = %v, want ErrNotFound", err)
	}
}

func TestKeyringStoreDeleteMissingIsIdempotent(t *testing.T) {
	store := newKeyringStore(&fakeBackend{values: map[string]string{}})
	if err := store.Delete("missing"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
}

func TestKeyringStoreRejectsMalformedEntry(t *testing.T) {
	backend := &fakeBackend{values: map[string]string{serviceName + "|broken": "not-json"}}
	_, err := newKeyringStore(backend).Get("broken")
	if err == nil || !strings.Contains(err.Error(), "decode OS keyring entry") {
		t.Fatalf("Get() error = %v", err)
	}
}

func TestKeyringStoreWrapsBackendErrors(t *testing.T) {
	backendErr := errors.New("backend unavailable")
	tests := []struct {
		name string
		call func(*KeyringStore) error
		want string
		fake *fakeBackend
	}{
		{name: "get", fake: &fakeBackend{values: map[string]string{}, getErr: backendErr}, call: func(store *KeyringStore) error { _, err := store.Get("account"); return err }, want: "read OS keyring"},
		{name: "set", fake: &fakeBackend{values: map[string]string{}, setErr: backendErr}, call: func(store *KeyringStore) error { return store.Set("account", Tokens{AuthToken: "auth"}) }, want: "write OS keyring"},
		{name: "delete", fake: &fakeBackend{values: map[string]string{}, deleteErr: backendErr}, call: func(store *KeyringStore) error { return store.Delete("account") }, want: "delete OS keyring"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.call(newKeyringStore(test.fake))
			if err == nil || !strings.Contains(err.Error(), test.want) || !errors.Is(err, backendErr) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

// The rename must not silently log everyone out, so a credential stored under
// the previous keyring service is adopted on first read.
func TestGetAdoptsACredentialFromTheLegacyService(t *testing.T) {
	backend := &fakeBackend{values: map[string]string{
		legacyServiceName + "|profile|https://example.test/api/v1/": `{"auth_token":"kept","refresh_token":"also-kept"}`,
	}}
	store := newKeyringStore(backend)
	tokens, err := store.Get("profile|https://example.test/api/v1/")
	if err != nil {
		t.Fatalf("Get() = %v, want the legacy credential", err)
	}
	if tokens.AuthToken != "kept" || tokens.RefreshToken != "also-kept" {
		t.Fatalf("tokens = %#v", tokens)
	}
	if _, ok := backend.values[serviceName+"|profile|https://example.test/api/v1/"]; !ok {
		t.Fatal("the credential was not copied to the current service, so every run would pay the lookup")
	}
}

func TestGetPrefersTheCurrentServiceOverTheLegacyOne(t *testing.T) {
	backend := &fakeBackend{values: map[string]string{
		serviceName + "|account":       `{"auth_token":"current"}`,
		legacyServiceName + "|account": `{"auth_token":"stale"}`,
	}}
	if tokens, err := newKeyringStore(backend).Get("account"); err != nil || tokens.AuthToken != "current" {
		t.Fatalf("Get() = %#v, %v", tokens, err)
	}
}

func TestGetReportsNotFoundWhenNeitherServiceHasIt(t *testing.T) {
	if _, err := newKeyringStore(&fakeBackend{values: map[string]string{}}).Get("account"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get() = %v, want ErrNotFound", err)
	}
}

func TestDeleteRemovesBothServices(t *testing.T) {
	backend := &fakeBackend{values: map[string]string{
		serviceName + "|account":       `{"auth_token":"current"}`,
		legacyServiceName + "|account": `{"auth_token":"stale"}`,
	}}
	if err := newKeyringStore(backend).Delete("account"); err != nil {
		t.Fatal(err)
	}
	if len(backend.values) != 0 {
		t.Fatalf("logout left credentials behind: %#v", backend.values)
	}
}
