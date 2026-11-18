package domain

// SwitchBody represents the body returned by POST /v1/c2s/switch/querySwitchStatus.
type SwitchBody struct {
	SwitchStatus   int            `json:"switchStatus"`
	Config         string         `json:"config"`
	ParsedConfig   map[string]any `json:"parsedConfig,omitempty"`
	ValidTimestamp *int64         `json:"validTimestamp,omitempty"`
	EndTimestamp   *int64         `json:"endTimestamp,omitempty"`
}

// SwitchStatusResponse represents the full envelope from querySwitchStatus.
type SwitchStatusResponse struct {
	ErrorCode int         `json:"errorCode"`
	Message   string      `json:"message"`
	Body      *SwitchBody `json:"body,omitempty"`
}

// CheckPayInput represents request parameters for POST /v1/c2s/appStore/checkPay.
// Callers may supply a pre-encrypted Cipher, or provide a Base64 X.509 RSA PublicKey
// to let the service generate an AES key, RSA-encrypt it, and AES-GCM decrypt the response.
type CheckPayInput struct {
	AppID     int64  `json:"appId" binding:"required,gt=0"`
	Cipher    string `json:"cipher,omitempty"`
	PublicKey string `json:"publicKey,omitempty"`
}

// CheckPayItem represents a decrypted purchase status item from checkPay.
type CheckPayItem struct {
	AppID     int64 `json:"appid"`
	PayStatus int32 `json:"payStatus"`
}

// CheckPayResponse represents the response from POST /v1/c2s/appStore/checkPay.
type CheckPayResponse struct {
	ErrorCode     int            `json:"errorCode"`
	Message       string         `json:"message"`
	Body          string         `json:"body,omitempty"`
	DecryptedBody []CheckPayItem `json:"decryptedBody,omitempty"`
}

// ExchangeKeyInput represents request options for POST /sporthealth/v1/c2s/encrypt/exchangeSymmetricKey.
type ExchangeKeyInput struct {
	EncryptionAlgorithm *int `json:"encryptionAlgorithm,omitempty"`
	FirstExchangeKey    *int `json:"firstExchangeKey,omitempty"`
}

// ExchangeKeyDecryptedBody represents the AES-256-GCM decrypted response payload.
type ExchangeKeyDecryptedBody struct {
	EncryptionAlgorithm int     `json:"encryptionAlgorithm"`
	SymmetricKey        string  `json:"symmetricKey"`
	KeyLength           int     `json:"keyLength"`
	PublicKey           *string `json:"publicKey"`
	PublicKeyVersion    *int    `json:"publicKeyVersion"`
}

// ExchangeKeyResponse represents the full response from exchangeSymmetricKey.
type ExchangeKeyResponse struct {
	ErrorCode     int                       `json:"errorCode"`
	Message       string                    `json:"message"`
	Body          string                    `json:"body,omitempty"`
	DecryptedBody *ExchangeKeyDecryptedBody `json:"decryptedBody,omitempty"`
}
