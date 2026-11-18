package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"testing"

	"owas-api/internal/domain"
)

func TestComputeAppStoreSignGolden(t *testing.T) {
	profile := domain.DefaultDeviceProfile()
	ua := profile.UserAgent()
	if ua != "OPPO/OWW251/30/11/4.0/20192/2230/8000023" {
		t.Fatalf("unexpected User-Agent: %s", ua)
	}

	got := ComputeAppStoreSign(
		"/watch/search/list",
		"word=%E5%BE%AE%E4%BF%A1&type=0",
		ua,
		"1794988800000",
		domain.AppStoreDeviceID,
	)
	const want = "778273067a9172816918c67423f791ab"
	if got != want {
		t.Fatalf("sign mismatch: got %s, want %s", got, want)
	}
}

func TestBuildAppStoreHeaders(t *testing.T) {
	profile := domain.DefaultDeviceProfile()
	headers := BuildAppStoreHeaders("/watch/search/list", "word=%E5%BE%AE%E4%BF%A1&type=0", profile, "1794988800000")
	if len(headers) != 16 {
		t.Fatalf("expected 16 headers, got %d", len(headers))
	}
	if headers["sign"] != "778273067a9172816918c67423f791ab" {
		t.Fatalf("unexpected sign header: %s", headers["sign"])
	}
	if headers["accept"] != "application/x2-protostuff" {
		t.Fatalf("unexpected accept header: %s", headers["accept"])
	}
}

func TestXXTEADecryptConstants(t *testing.T) {
	appID, err := XXTEADecryptString(exchangeAppIDEnc)
	if err != nil {
		t.Fatalf("decrypt exchangeAppIDEnc failed: %v", err)
	}
	if appID != "YsQkfC2m8nMs2jgtmRRwr0" {
		t.Fatalf("got appID %q, want YsQkfC2m8nMs2jgtmRRwr0", appID)
	}

	ver, err := XXTEADecryptString(exchangeKeyVersionEnc)
	if err != nil {
		t.Fatalf("decrypt exchangeKeyVersionEnc failed: %v", err)
	}
	if ver != "1683169790789" {
		t.Fatalf("got keyVersion %q, want 1683169790789", ver)
	}

	healthKey, err := XXTEADecryptString(healthSignKeyEnc)
	if err != nil || len(healthKey) != 24 {
		t.Fatalf("unexpected health sign key: len=%d err=%v", len(healthKey), err)
	}

	exchKey, err := XXTEADecryptString(exchangeSignKeyEnc)
	if err != nil || len(exchKey) != 24 {
		t.Fatalf("unexpected exchange sign key: len=%d err=%v", len(exchKey), err)
	}

	rsaPub, err := XXTEADecryptString(exchangeRSAPubKeyEnc)
	if err != nil || len(rsaPub) != 392 {
		t.Fatalf("unexpected rsa pubkey: len=%d err=%v", len(rsaPub), err)
	}
}

func TestBuildHealthCloudHeadersTokenOmission(t *testing.T) {
	profile := domain.DefaultDeviceProfile()
	headers, err := BuildHealthCloudHeaders("https://sport.health.heytapmobi.com/v1/c2s/switch/querySwitchStatus", `{"switchType":116}`, profile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, exists := headers["token"]; exists {
		t.Fatal("empty token header must be omitted to avoid upstream 10100 signature error")
	}
	if headers["signature"] == "" || headers["appid"] != HealthCloudAppID {
		t.Fatalf("missing signature or appid: %+v", headers)
	}
}

func TestBuildHealthCloudHeadersCheckPaySimplified(t *testing.T) {
	profile := domain.DefaultDeviceProfile()
	profile.Token = "user-token"
	headers, err := BuildHealthCloudHeaders("https://sport.health.heytapmobi.com/v1/c2s/appStore/checkPay", `{"appid":1}`, profile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, exists := headers["signature"]; exists {
		t.Fatal("checkPay should use simplified headers without HMAC signature")
	}
	if headers["token"] != "user-token" || headers["ch"] != domain.AppStoreChannel {
		t.Fatalf("unexpected checkPay headers: %+v", headers)
	}
}

func TestRSAEncryptAndAESGCMRoundTrip(t *testing.T) {
	// Test built-in RSA public key encryption.
	cipherText, err := RSAEncryptPKCS1(`{"test":1}`, "")
	if err != nil {
		t.Fatalf("built-in RSA encrypt failed: %v", err)
	}
	if cipherText == "" {
		t.Fatal("expected non-empty RSA ciphertext")
	}

	// Test custom key pair round-trip.
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}
	der, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatalf("marshal public key: %v", err)
	}
	enc, err := RSAEncryptPKCS1("hello-owas", base64.StdEncoding.EncodeToString(der))
	if err != nil {
		t.Fatalf("RSAEncryptPKCS1 failed: %v", err)
	}
	rawEnc, _ := base64.StdEncoding.DecodeString(enc)
	plain, err := rsa.DecryptPKCS1v15(rand.Reader, priv, rawEnc)
	if err != nil || string(plain) != "hello-owas" {
		t.Fatalf("RSA decrypt mismatch: %q err=%v", string(plain), err)
	}

	// Test AES-256-GCM with 16-byte IV (matching exchangeSymmetricKey 32-char UUID key).
	key, err := GenerateUUIDHex()
	if err != nil || len(key) != 32 {
		t.Fatalf("GenerateUUIDHex failed: %q err=%v", key, err)
	}
	sealed, err := AESGCMEncryptText(key, `{"encryptionAlgorithm":4,"symmetricKey":"abc"}`)
	if err != nil {
		t.Fatalf("AESGCMEncryptText failed: %v", err)
	}
	decrypted, err := AESGCMDecryptText(key, sealed)
	if err != nil {
		t.Fatalf("AESGCMDecryptText failed: %v", err)
	}
	if decrypted != `{"encryptionAlgorithm":4,"symmetricKey":"abc"}` {
		t.Fatalf("unexpected decrypted payload: %s", decrypted)
	}
}
