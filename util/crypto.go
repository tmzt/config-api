package util

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"log"
	"os"
)

func MustGetRootTokenPrivateKey() *ecdsa.PrivateKey {
	v := os.Getenv("ROOT_JWT_TOKEN_ECDSA_PRIVATE_KEY_BASE64")
	if v == "" {
		log.Fatalf("ROOT_JWT_TOKEN_ECDSA_PRIVATE_KEY_BASE64 is required")
	}

	log.Printf("ROOT_JWT_TOKEN_ECDSA_PRIVATE_KEY_BASE64: %s", v)

	// Decode outer base64
	decoded, err := base64.StdEncoding.DecodeString(v)
	if err != nil {
		log.Fatalf("Failed to decode ROOT_JWT_TOKEN_PRIVATE_KEY_BASE64: %v", err)
	}

	log.Printf("ROOT_JWT_TOKEN_ECDSA_PRIVATE_KEY_BASE64 decoded: %s", decoded)

	// Decode pem
	block, _ := pem.Decode(decoded)
	if block == nil {
		log.Fatalf("Failed to decode ROOT_JWT_TOKEN_PRIVATE_KEY_BASE64 as PEM: %v", err)
	}

	// Decode x509 from block
	key, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		log.Fatalf("Failed to parse ROOT_JWT_TOKEN_PRIVATE_KEY_BASE64: %v", err)
	}

	// // Assert that this is an ecdsa key
	// ecdsaKey, ok := key.(*ecdsa.PrivateKey)
	// if !ok {
	// 	log.Fatalf("Failed to assert ROOT_JWT_TOKEN_PRIVATE_KEY is an ECDSA key")
	// }

	// return ecdsaKey
	return key
}

func MustGetRootTokenPublicKey() *ecdsa.PublicKey {
	v := os.Getenv("ROOT_JWT_TOKEN_ECDSA_PUBLIC_KEY_BASE64")
	if v == "" {
		log.Fatalf("ROOT_JWT_TOKEN_ECDSA_PUBLIC_KEY_BASE64 is required")
	}

	log.Printf("ROOT_JWT_TOKEN_ECDSA_PUBLIC_KEY_BASE64: %s", v)

	// Decode outer base64
	decoded, err := base64.StdEncoding.DecodeString(v)
	if err != nil {
		log.Fatalf("Failed to decode ROOT_JWT_TOKEN_PUBLIC_KEY_BASE64: %v", err)
	}

	log.Printf("ROOT_JWT_TOKEN_ECDSA_PUBLIC_KEY_BASE64 decoded: %s", decoded)

	// Decode pem
	block, _ := pem.Decode(decoded)
	if block == nil {
		log.Fatalf("Failed to decode ROOT_JWT_TOKEN_PUBLIC_KEY_BASE64 as PEM: %v", err)
	}

	// Decode EC public key
	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		log.Fatalf("Failed to parse ROOT_JWT_TOKEN_PUBLIC_KEY_BASE64: %v", err)
	}

	// Assert that this is an ecdsa key
	ecdsaKey, ok := key.(*ecdsa.PublicKey)
	if !ok {
		log.Fatalf("Failed to assert ROOT_JWT_TOKEN_PUBLIC_KEY is an ECDSA key")
	}

	return ecdsaKey
}
