package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"owas-api/internal/domain"
)

// Constants extracted from p177z2.C3146a (AppStoreService).
const (
	AppStoreOAK        = "7e5b11cfed1fd2c3"
	AppStoreSignPrefix = "7e5b11cfed1fd2c3ac04b72d4c12c19c2a29b7450c3a90f1"
	AppStoreSignSuffix = "efceac0bfb3a0ea7"
)

// Constants extracted from p177z2.C3148c & p077k3.C2060e (HealthCloudService).
const (
	HealthCloudAppID = "H8zNrzmPjDDFRIvAVcyDDE"
	xxteaKeyA        = "fb6ec1bc-bfbf-4368-8fba-ac86797de144"
	xxteaKeyB        = "d4g5248v-ma24-mk87-146hj21378a1"
	healthSignKeyEnc = "B0sFdYWS6vTiZFHIoZ62yhiIPoh2NjFm/RKH1g=="
)

// Constants extracted from MsgReceive & p077k3.C2057b (exchangeSymmetricKey).
const (
	exchangeSignKeyEnc    = "EqoMTrqCJxycw07YfouQ3KI49y2SAnX676OCFA=="
	exchangeAppIDEnc      = "4r03uY5zeRfvNCu17i2p0EOkZV0hQr1V8xnh8Q=="
	exchangeKeyVersionEnc = "pSxtgI8ZjNMCRbfz9zwRGJg/iAA="
	exchangeRSAPubKeyEnc  = "NI3uKO7yL4WWTFHM1Vi8GlBEut/sfHeMov+6VqaehP6FErE3HbiZaWMe7mq+TzZuiVQhYgM8UeoSGoO85gGJNASpDVkZwQSFsff/" +
		"uGTyXX2RDJM9uWw0FzkgHp8V9VB5+kwms5HrghkYp8d6LaKJS+287l4PnAmqVb34DAZcsbAHI97VFFEf1Jop/VJef9Sx9wDmellc1" +
		"P9e/3Vaz6gBqGmOA1HMk0YTK5a4GQpbaUSkWB3UYUbphJvbt5o4XtvErd+3XEVO/7BAEsnnu60NvJduMzF0YbBTwwoj9GQyM8q5he" +
		"Z+QKW6O4NlrA4oMg/01xlCDPcUjnN3IJ5E863DL3OPuMUmfWBo6+r8TZnGsc7d4Lp/gZuvF3M8p8CXqtqc4LqnfNW4INZZb/ak3wb" +
		"o9/XZh7lSYWa4tI4DC6VzwS40IUUQEEq4PU5Ufcq0u5S6OSeTRz3GgM6d0+DDqjnnS2hAd/IxLapp0MYcGS3Nz7Fb56llMseCoI4G5" +
		"xKHx47ZciDHWN2m+spJwXPo"
)

const aesGCMNonceSize = 16

// ComputeAppStoreSign computes the 32-char lowercase MD5 signature matching p177z2.C3146a.mo5403a().
func ComputeAppStoreSign(encodedPath, encodedQuery, userAgent, timestampMs, deviceID string) string {
	if deviceID == "" {
		deviceID = domain.AppStoreDeviceID
	}
	raw := AppStoreSignPrefix + userAgent + timestampMs + deviceID + encodedPath + encodedQuery
	payload := fmt.Sprintf("%s%d%s", raw, utf8.RuneCountInString(raw), AppStoreSignSuffix)
	sum := md5.Sum([]byte(payload))
	return hex.EncodeToString(sum[:])
}

// BuildAppStoreHeaders constructs all 16 request headers injected by p177z2.C3146a.
func BuildAppStoreHeaders(encodedPath, encodedQuery string, profile domain.DeviceProfile, timestampMs string) map[string]string {
	dev := profile.WithDefaults()
	ts := timestampMs
	if ts == "" {
		ts = strconv.FormatInt(time.Now().UnixMilli(), 10)
	}
	ua := dev.UserAgent()
	sign := ComputeAppStoreSign(encodedPath, encodedQuery, ua, ts, domain.AppStoreDeviceID)
	return map[string]string{
		"token":          dev.Token,
		"id":             domain.AppStoreDeviceID,
		"oak":            AppStoreOAK,
		"sign":           sign,
		"t":              ts,
		"ch":             domain.AppStoreChannel,
		"accept":         "application/x2-protostuff",
		"support":        dev.Support,
		"User-Agent":     ua,
		"locale":         dev.Locale,
		"color-os-ver":   dev.ColorOSVer,
		"sys-ver":        dev.SysVer,
		"linkage":        dev.Linkage,
		"platCode":       strconv.Itoa(dev.PlatCode),
		"instantSwClose": dev.InstantSwClose,
		"watch-type":     dev.WatchType,
	}
}

func xxteaKey() []byte {
	var b strings.Builder
	limit := min(len(xxteaKeyA), len(xxteaKeyB))
	b.Grow(limit * 2)
	for i := range limit {
		b.WriteByte(xxteaKeyA[i])
		b.WriteByte(xxteaKeyB[i])
	}
	raw := []byte(b.String())
	key := make([]byte, 16)
	copy(key, raw)
	return key
}

// XXTEADecryptString decrypts an APK obfuscated Base64 string matching androidx.activity.C0096l.m123s().
func XXTEADecryptString(b64Ciphertext string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(b64Ciphertext)
	if err != nil {
		return "", fmt.Errorf("base64 decode failed: %w", err)
	}
	if len(data) == 0 {
		return "", nil
	}

	v := bytesToUint32LE(data)
	k := bytesToUint32LE(xxteaKey())
	n := len(v) - 1
	if n >= 1 {
		const delta uint32 = 0x9E3779B9
		rounds := uint32(6 + 52/(n+1))
		total := rounds * delta
		y := v[0]
		for total != 0 {
			e := (total >> 2) & 3
			for p := n; p >= 1; p-- {
				z := v[p-1]
				mx := (((z >> 5) ^ (y << 2)) + ((y >> 3) ^ (z << 4))) ^ ((total ^ y) + (k[(uint32(p)&3)^e] ^ z))
				v[p] -= mx
				y = v[p]
			}
			z := v[n]
			mx := (((z >> 5) ^ (y << 2)) + ((y >> 3) ^ (z << 4))) ^ ((total ^ y) + (k[e] ^ z))
			v[0] -= mx
			y = v[0]
			total -= delta
		}
	}

	outLen := v[len(v)-1]
	raw := uint32LEToBytes(v)
	if uint64(outLen) > uint64(len(raw)) {
		return "", fmt.Errorf("invalid xxtea output length %d (buffer %d)", outLen, len(raw))
	}
	return string(raw[:outLen]), nil
}

func bytesToUint32LE(buf []byte) []uint32 {
	rem := len(buf) % 4
	paddedLen := len(buf)
	if rem != 0 {
		paddedLen += 4 - rem
	}
	padded := make([]byte, paddedLen)
	copy(padded, buf)
	out := make([]uint32, paddedLen/4)
	for i := range out {
		out[i] = binary.LittleEndian.Uint32(padded[i*4 : i*4+4])
	}
	return out
}

func uint32LEToBytes(vals []uint32) []byte {
	out := make([]byte, len(vals)*4)
	for i, v := range vals {
		binary.LittleEndian.PutUint32(out[i*4:i*4+4], v)
	}
	return out
}

// BuildHealthCloudHeaders constructs signed headers for HealthCloudService matching p177z2.C3148c.m6473a().
func BuildHealthCloudHeaders(rawURL, bodyText string, profile domain.DeviceProfile) (map[string]string, error) {
	nonce, err := GenerateUUIDHex()
	if err != nil {
		return nil, err
	}
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	return buildHealthCloudHeadersWithNonce(rawURL, bodyText, profile, nonce, ts)
}

func buildHealthCloudHeadersWithNonce(rawURL, bodyText string, profile domain.DeviceProfile, nonce, timestampMs string) (map[string]string, error) {
	dev := profile.WithDefaults()
	if strings.Contains(rawURL, "v1/c2s/appStore/checkPay") {
		return map[string]string{
			"token":        dev.Token,
			"ch":           domain.AppStoreChannel,
			"accept":       "application/json",
			"User-Agent":   dev.UserAgent(),
			"color-os-ver": dev.ColorOSVer,
			"sys-ver":      dev.SysVer,
		}, nil
	}

	signMap := map[string]string{
		"appid":        HealthCloudAppID,
		"device-id":    domain.AppStoreDeviceID,
		"device-model": dev.Model,
		"nonce":        nonce,
		"timestamp":    timestampMs,
	}
	if dev.Token != "" {
		signMap["token"] = dev.Token
	}

	keys := make([]string, 0, len(signMap))
	for k, v := range signMap {
		if v != "" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	pairs := make([]string, 0, len(keys))
	for _, k := range keys {
		pairs = append(pairs, k+"="+signMap[k])
	}
	canonical := strings.Join(pairs, "&")
	if bodyText != "" {
		canonical += stripAllWhitespace(bodyText)
	}

	secret, err := XXTEADecryptString(healthSignKeyEnc)
	if err != nil {
		return nil, fmt.Errorf("decrypt health sign key: %w", err)
	}
	signature := hmacSHA256Base64([]byte(secret), []byte(canonical))

	headers := make(map[string]string, len(signMap)+4)
	for k, v := range signMap {
		headers[k] = v
	}
	headers["signature"] = signature
	headers["accept"] = "application/json"
	headers["color-os-ver"] = dev.ColorOSVer
	headers["sys-ver"] = dev.SysVer
	return headers, nil
}

// BuildExchangeKeyHeaders builds signed headers for /sporthealth/v1/c2s/encrypt/exchangeSymmetricKey (p077k3.C2057b).
func BuildExchangeKeyHeaders(encryptedBody string, profile domain.DeviceProfile) (map[string]string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(0x7FFFFFFF))
	if err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}
	nonce := strconv.FormatInt(n.Int64()+1, 10)
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	return buildExchangeKeyHeadersWithNonce(encryptedBody, profile, nonce, ts)
}

func buildExchangeKeyHeadersWithNonce(encryptedBody string, profile domain.DeviceProfile, nonce, timestampMs string) (map[string]string, error) {
	dev := profile.WithDefaults()
	appID, err := XXTEADecryptString(exchangeAppIDEnc)
	if err != nil {
		return nil, fmt.Errorf("decrypt exchange appId: %w", err)
	}
	keyVersion, err := XXTEADecryptString(exchangeKeyVersionEnc)
	if err != nil {
		return nil, fmt.Errorf("decrypt exchange keyVersion: %w", err)
	}
	secret, err := XXTEADecryptString(exchangeSignKeyEnc)
	if err != nil {
		return nil, fmt.Errorf("decrypt exchange sign key: %w", err)
	}

	canonical := fmt.Sprintf(
		"appid=%s&device-id=%s&device-model=%s&nonce=%s&timestamp=%s%s",
		appID,
		domain.AppStoreDeviceID,
		dev.Model,
		nonce,
		timestampMs,
		encryptedBody,
	)
	signature := hmacSHA256Base64([]byte(secret), []byte(canonical))

	return map[string]string{
		"appid":            appID,
		"device-id":        domain.AppStoreDeviceID,
		"device-model":     dev.Model,
		"nonce":            nonce,
		"timestamp":        timestampMs,
		"signature":        signature,
		"app-key-version":  keyVersion,
		"mobile-unique-id": domain.AppStoreDeviceID,
	}, nil
}

// RSAEncryptPKCS1 encrypts UTF-8 plaintext with RSA/ECB/PKCS1Padding matching p077k3.C2056a.m4687a().
// If base64X509PubKey is empty, the built-in APK exchange public key is used.
func RSAEncryptPKCS1(plaintext, base64X509PubKey string) (string, error) {
	pubB64 := strings.TrimSpace(base64X509PubKey)
	if pubB64 == "" {
		decrypted, err := XXTEADecryptString(exchangeRSAPubKeyEnc)
		if err != nil {
			return "", fmt.Errorf("decrypt built-in RSA public key: %w", err)
		}
		pubB64 = decrypted
	}

	derBytes, err := base64.StdEncoding.DecodeString(pubB64)
	if err != nil {
		return "", fmt.Errorf("decode RSA public key base64: %w", err)
	}
	parsed, err := x509.ParsePKIXPublicKey(derBytes)
	if err != nil {
		return "", fmt.Errorf("parse X.509 RSA public key: %w", err)
	}
	rsaPub, ok := parsed.(*rsa.PublicKey)
	if !ok {
		return "", errors.New("public key is not an RSA key")
	}
	encrypted, err := rsa.EncryptPKCS1v15(rand.Reader, rsaPub, []byte(plaintext))
	if err != nil {
		return "", fmt.Errorf("RSA PKCS1v15 encrypt: %w", err)
	}
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// AESGCMDecryptText decrypts Base64(16-byte IV || ciphertext+tag) matching kotlinx.coroutines.C2279c0.m4966v().
func AESGCMDecryptText(password, b64Payload string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(b64Payload)
	if err != nil {
		return "", fmt.Errorf("base64 decode AES-GCM payload: %w", err)
	}
	if len(raw) <= aesGCMNonceSize {
		return "", fmt.Errorf("AES-GCM payload too short: %d bytes", len(raw))
	}
	block, err := aes.NewCipher([]byte(password))
	if err != nil {
		return "", fmt.Errorf("create AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCMWithNonceSize(block, aesGCMNonceSize)
	if err != nil {
		return "", fmt.Errorf("create AES-GCM with 16-byte nonce: %w", err)
	}
	iv := raw[:aesGCMNonceSize]
	ctAndTag := raw[aesGCMNonceSize:]
	plaintext, err := gcm.Open(nil, iv, ctAndTag, nil)
	if err != nil {
		return "", fmt.Errorf("AES-GCM decrypt failed: %w", err)
	}
	return string(plaintext), nil
}

// AESGCMEncryptText encrypts plaintext into Base64(16-byte IV || ciphertext+tag).
func AESGCMEncryptText(password, plaintext string) (string, error) {
	block, err := aes.NewCipher([]byte(password))
	if err != nil {
		return "", fmt.Errorf("create AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCMWithNonceSize(block, aesGCMNonceSize)
	if err != nil {
		return "", fmt.Errorf("create AES-GCM with 16-byte nonce: %w", err)
	}
	iv := make([]byte, aesGCMNonceSize)
	if _, err := rand.Read(iv); err != nil {
		return "", fmt.Errorf("generate random IV: %w", err)
	}
	sealed := gcm.Seal(iv, iv, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// GenerateUUIDHex returns a 32-char lowercase hex RFC 4122 UUIDv4 string (matching Python uuid.uuid4().hex).
func GenerateUUIDHex() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate uuid bytes: %w", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return hex.EncodeToString(b[:]), nil
}

// GenerateAES128KeyBase64 generates a 128-bit random key encoded as standard Base64 (no line wraps).
func GenerateAES128KeyBase64() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate AES-128 key: %w", err)
	}
	return base64.StdEncoding.EncodeToString(b[:]), nil
}

func hmacSHA256Base64(key, msg []byte) string {
	mac := hmac.New(sha256.New, key)
	mac.Write(msg)
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func stripAllWhitespace(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if !unicode.IsSpace(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}
