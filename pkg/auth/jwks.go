package auth

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWKS struct {
	Keys []JSONWebKey `json:"keys"`
}

type JSONWebKey struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type Provider struct {
	mu        sync.RWMutex
	keys      map[string]*JSONWebKey
	url       string
	refreshed time.Time
}

func NewProvider(jwksURL string) *Provider {
	return &Provider{
		url:  jwksURL,
		keys: make(map[string]*JSONWebKey),
	}
}

func (p *Provider) KeyFunc(token *jwt.Token) (interface{}, error) {
	// Verify signing method
	if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
		return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
	}

	kid, ok := token.Header["kid"].(string)
	if !ok {
		return nil, fmt.Errorf("kid header not found")
	}

	key, err := p.GetKey(kid)
	if err != nil {
		return nil, err
	}

	return key.GetPublicKey()
}

func (p *Provider) GetKey(kid string) (*JSONWebKey, error) {
	p.mu.RLock()
	key, exists := p.keys[kid]
	p.mu.RUnlock()

	if exists {
		return key, nil
	}

	// Refresh keys
	if err := p.fetchKeys(); err != nil {
		return nil, err
	}

	p.mu.RLock()
	key, exists = p.keys[kid]
	p.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("key not found")
	}
	return key, nil
}

func (p *Provider) fetchKeys() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Rate limit refresh (1 min)
	if time.Since(p.refreshed) < time.Minute && len(p.keys) > 0 {
		return nil
	}

	resp, err := http.Get(p.url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var jwks JWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return err
	}

	p.keys = make(map[string]*JSONWebKey)
	for _, k := range jwks.Keys {
		k := k // Capture loop variable
		p.keys[k.Kid] = &k
	}
	p.refreshed = time.Now()
	return nil
}

func (k *JSONWebKey) GetPublicKey() (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		return nil, err
	}

	n := new(big.Int).SetBytes(nBytes)

	var e int
	for _, b := range eBytes {
		e = e<<8 | int(b)
	}

	return &rsa.PublicKey{N: n, E: e}, nil
}
