package util

import (
	"log"
	"os"
	"strconv"
	"time"
)

func MustGetPostgresURL() string {
	url := os.Getenv("POSTGRES_URL")
	if url == "" {
		log.Fatal("POSTGRES_URL is not set")
	}
	return url
}

func MustGetApiBaseUrl() string {
	v := os.Getenv("API_BASE_URL")
	if v == "" {
		log.Fatalf("API_BASE_URL is required")
	}

	return v
}

func MustGetPublicApiBaseUrl() string {
	v := os.Getenv("API_PUBLIC_BASE_URL")
	if v == "" {
		log.Fatalf("API_PUBLIC_BASE_URL is required")
	}

	return v
}

func MustGetPlatformId() AccountId {
	v := os.Getenv("ROOT_ACCOUNT_ID")
	if v == "" {
		log.Fatalf("ROOT_ACCOUNT_ID is required")
	}

	return AccountId(v)
}

func MustGetRootTokenIssuer() string {
	v := os.Getenv("ROOT_JWT_TOKEN_ISSUER")
	if v == "" {
		log.Fatalf("ROOT_JWT_TOKEN_ISSUER is required")
	}

	return v
}

func MustGetRootTokenSecret() string {
	v := os.Getenv("ROOT_JWT_TOKEN_SECRET")
	if v == "" {
		log.Fatalf("ROOT_JWT_TOKEN_SECRET is required")
	}

	return v
}

func MustGetRootTokenMaxAge() time.Duration {
	if v := os.Getenv("ROOT_JWT_TOKEN_MAX_AGE"); v != "" {
		if v, err := strconv.Atoi(v); err == nil {
			dur := time.Duration(v) * time.Second
			return dur
		}
	}

	log.Fatalf("ROOT_JWT_TOKEN_MAX_AGE is required")
	return time.Duration(0)
}

func MustGetRootTokenAudience() string {
	v := os.Getenv("ROOT_JWT_TOKEN_AUDIENCE")
	if v == "" {
		log.Fatalf("ROOT_JWT_TOKEN_AUDIENCE is required")
	}

	return v
}
