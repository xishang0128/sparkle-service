package route

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type KeyManager struct {
	publicKey ed25519.PublicKey
	mu        sync.RWMutex
	keyPath   string
}

var globalKeyManager *KeyManager
var kmOnce sync.Once

func GetKeyManager() *KeyManager {
	kmOnce.Do(func() {
		globalKeyManager = &KeyManager{}
	})
	return globalKeyManager
}

func InitKeyManager(keyDir string) error {
	km := GetKeyManager()
	km.keyPath = filepath.Join(keyDir, "public_key.pem")

	if err := os.MkdirAll(keyDir, 0o755); err != nil {
		return fmt.Errorf("创建密钥目录失败： %w", err)
	}

	if err := km.loadPublicKey(); err != nil {
		return fmt.Errorf("加载公钥失败： %w", err)
	}

	return nil
}

func (km *KeyManager) SetPublicKey(pubKeyBase64 string) error {
	km.mu.Lock()
	defer km.mu.Unlock()

	pubKeyBytes, err := base64.StdEncoding.DecodeString(pubKeyBase64)
	if err != nil {
		return fmt.Errorf("公钥 base64 解码失败： %w", err)
	}

	pub, err := x509.ParsePKIXPublicKey(pubKeyBytes)
	if err != nil {
		return fmt.Errorf("解析公钥失败： %w", err)
	}

	edPub, ok := pub.(ed25519.PublicKey)
	if !ok {
		return fmt.Errorf("公钥不是 Ed25519 类型")
	}

	km.publicKey = edPub

	pubKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	})

	if err := os.WriteFile(km.keyPath, pubKeyPEM, 0o644); err != nil {
		return fmt.Errorf("保存公钥失败： %w", err)
	}

	return nil
}

func (km *KeyManager) loadPublicKey() error {
	pubKeyPEM, err := os.ReadFile(km.keyPath)
	if err != nil {
		return fmt.Errorf("读取公钥文件失败： %w", err)
	}

	block, _ := pem.Decode(pubKeyPEM)
	if block == nil {
		return fmt.Errorf("无效的 PEM 格式")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("解析公钥失败： %w", err)
	}

	edPub, ok := pub.(ed25519.PublicKey)
	if !ok {
		return fmt.Errorf("公钥不是 Ed25519 类型")
	}

	km.publicKey = edPub
	return nil
}

func (km *KeyManager) VerifySignature(message string, signature string) error {
	km.mu.RLock()
	defer km.mu.RUnlock()

	if km.publicKey == nil {
		return fmt.Errorf("公钥未初始化")
	}

	sig, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return fmt.Errorf("签名解码失败： %w", err)
	}

	if !ed25519.Verify(km.publicKey, []byte(message), sig) {
		return fmt.Errorf("签名验证失败")
	}

	return nil
}

func (km *KeyManager) IsInitialized() bool {
	km.mu.RLock()
	defer km.mu.RUnlock()
	return km.publicKey != nil
}
