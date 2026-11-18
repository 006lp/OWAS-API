package upstream

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"owas-api/internal/domain"
	"owas-api/internal/protocol/crypto"
)

func TestHealthClientQuerySwitchStatus(t *testing.T) {
	httpClient := newInMemoryHTTPClient(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/c2s/switch/querySwitchStatus" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("signature") == "" {
			t.Error("expected HMAC signature header")
		}
		if _, exists := r.Header["Token"]; exists {
			t.Error("empty token must not be sent")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"errorCode":0,"message":"成功","body":{"switchStatus":1,"config":"{\"showPowerDialog\":true}"}}`))
	})

	client := NewHealthClient("https://mock.health.local", 5*time.Second, httpClient)
	got, err := client.QuerySwitchStatus(context.Background(), domain.DefaultDeviceProfile(), 116)
	if err != nil {
		t.Fatalf("QuerySwitchStatus failed: %v", err)
	}
	if got.ErrorCode != 0 || got.Body == nil || got.Body.SwitchStatus != 1 {
		t.Fatalf("unexpected switch response: %+v", got)
	}
	if got.Body.ParsedConfig["showPowerDialog"] != true {
		t.Fatalf("expected parsedConfig.showPowerDialog=true, got %+v", got.Body.ParsedConfig)
	}
}

func TestHealthClientCheckPayAuthBoundaryAndCrypto(t *testing.T) {
	t.Run("403 maps to ErrUnauthorized", func(t *testing.T) {
		httpClient := newInMemoryHTTPClient(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"errorCode":403}`))
		})

		client := NewHealthClient("https://mock.health.local", 5*time.Second, httpClient)
		_, err := client.CheckPay(context.Background(), domain.DefaultDeviceProfile(), domain.CheckPayInput{AppID: 31617863})
		if !errors.Is(err, domain.ErrUnauthorized) {
			t.Fatalf("expected ErrUnauthorized, got %v", err)
		}
	})

	t.Run("RSA encrypt and AES-GCM decrypt with publicKey", func(t *testing.T) {
		priv, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatalf("generate rsa key: %v", err)
		}
		pubDER, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
		if err != nil {
			t.Fatalf("marshal pub key: %v", err)
		}

		httpClient := newInMemoryHTTPClient(func(w http.ResponseWriter, r *http.Request) {
			raw, _ := io.ReadAll(r.Body)
			var req struct {
				AppID  int64  `json:"appid"`
				Cipher string `json:"cipher"`
			}
			_ = json.Unmarshal(raw, &req)
			encBytes, _ := base64.StdEncoding.DecodeString(req.Cipher)
			aesKeyBytes, err := rsa.DecryptPKCS1v15(rand.Reader, priv, encBytes)
			if err != nil {
				t.Errorf("server failed to decrypt client cipher: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			encResp, err := crypto.AESGCMEncryptText(string(aesKeyBytes), `[{"appid":37627586,"payStatus":1}]`)
			if err != nil {
				t.Errorf("server failed to encrypt response: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"errorCode": 0,
				"message":   "ok",
				"body":      encResp,
			})
		})

		profile := domain.DefaultDeviceProfile()
		profile.Token = "valid-token"
		client := NewHealthClient("https://mock.health.local", 5*time.Second, httpClient)
		got, err := client.CheckPay(context.Background(), profile, domain.CheckPayInput{
			AppID:     37627586,
			PublicKey: base64.StdEncoding.EncodeToString(pubDER),
		})
		if err != nil {
			t.Fatalf("CheckPay failed: %v", err)
		}
		if len(got.DecryptedBody) != 1 || got.DecryptedBody[0].AppID != 37627586 || got.DecryptedBody[0].PayStatus != 1 {
			t.Fatalf("unexpected decrypted checkPay body: %+v", got)
		}
	})
}

func TestHealthClientExchangeSymmetricKeyHeaders(t *testing.T) {
	httpClient := newInMemoryHTTPClient(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sporthealth/v1/c2s/encrypt/exchangeSymmetricKey" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		for _, hdr := range []string{"appid", "device-id", "device-model", "nonce", "timestamp", "signature", "app-key-version", "mobile-unique-id"} {
			if r.Header.Get(hdr) == "" {
				t.Errorf("missing required header %s", hdr)
			}
		}
		raw, _ := io.ReadAll(r.Body)
		if _, err := base64.StdEncoding.DecodeString(string(raw)); err != nil {
			t.Errorf("expected base64 RSA ciphertext body, got error: %v", err)
		}
		// Return envelope with empty body to validate transport & header wiring without server RSA private key.
		_ = json.NewEncoder(w).Encode(map[string]any{
			"errorCode": 0,
			"message":   "ok",
			"body":      "",
		})
	})

	client := NewHealthClient("https://mock.health.local", 5*time.Second, httpClient)
	got, err := client.ExchangeSymmetricKey(context.Background(), domain.DefaultDeviceProfile(), 4, 0)
	if err != nil {
		t.Fatalf("ExchangeSymmetricKey failed: %v", err)
	}
	if got.ErrorCode != 0 {
		t.Fatalf("unexpected errorCode: %d", got.ErrorCode)
	}
}

func TestApplyHeadersPreservesExactLowercaseKeys(t *testing.T) {
	var seenRawKeys map[string][]string
	httpClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			seenRawKeys = req.Header.Clone()
			rec := newInMemoryHTTPClient(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"errorCode":0,"message":"ok","body":{"switchStatus":1,"config":"{}"}}`))
			})
			return rec.Transport.RoundTrip(req)
		}),
	}

	client := NewHealthClient("https://mock.health.local", 5*time.Second, httpClient)
	if _, err := client.QuerySwitchStatus(context.Background(), domain.DefaultDeviceProfile(), 116); err != nil {
		t.Fatalf("QuerySwitchStatus failed: %v", err)
	}
	for _, key := range []string{"appid", "device-id", "device-model", "nonce", "timestamp", "signature"} {
		if len(seenRawKeys[key]) == 0 {
			t.Fatalf("expected exact lowercase wire header key %q, got keys=%v", key, seenRawKeys)
		}
	}
}
