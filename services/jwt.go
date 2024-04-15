package services

import (
	"crypto/ecdsa"
	"log"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/tmzt/config-api/models"
	"github.com/tmzt/config-api/util"
)

type JwtService struct {
	logger util.LoggerInterface

	publicKey  *ecdsa.PublicKey
	privateKey *ecdsa.PrivateKey

	maxAge time.Duration
	issuer string
}

func NewJwtService() *JwtService {
	logger := util.NewLogger("JwtService", 0)

	publicKey := util.MustGetRootTokenPublicKey()
	privateKey := util.MustGetRootTokenPrivateKey()
	maxAge := util.MustGetRootTokenMaxAge()
	issuer := util.MustGetRootTokenIssuer()

	return &JwtService{
		logger:     logger,
		publicKey:  publicKey,
		privateKey: privateKey,
		maxAge:     maxAge,
		issuer:     issuer,
	}
}

func (s *JwtService) ParseWithClaims(tokenString string, claims jwt.Claims) (*jwt.Token, error) {
	return jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return s.publicKey, nil
	})
}

func (s *JwtService) ParseValidToken(tokenString string) (*jwt.Token, jwt.Claims, error) {
	claims := &jwt.MapClaims{}

	tokenObj, err := s.ParseWithClaims(tokenString, claims)

	if err != nil {
		log.Printf("JwtService: Error parsing token: %v", err)
		return nil, nil, err
	}

	if !tokenObj.Valid {
		log.Printf("JwtService: Invalid token")
		return nil, nil, jwt.ErrSignatureInvalid
	}

	return tokenObj, claims, nil
}

func (s *JwtService) ParseValidTokenWithCommonClaims(tokenString string) (*jwt.Token, jwt.Claims, *models.CommonTokenClaims, error) {
	tokenObj, claims, err := s.ParseValidToken(tokenString)
	if err != nil {
		return nil, nil, nil, err
	} else if !tokenObj.Valid {
		return nil, nil, nil, jwt.ErrSignatureInvalid
	}

	s.logger.Printf("Claims: %v\n", claims)

	common := models.GetCommonClaims(tokenObj)
	if common == nil {
		s.logger.Printf("Invalid token, missing common claims")
		return nil, nil, nil, jwt.NewValidationError("Invalid token, missing common claims", jwt.ValidationErrorClaimsInvalid)
	}

	return tokenObj, claims, common, nil
}

func (s *JwtService) GetDefaultIssuer() string {
	return s.issuer
}

func (s *JwtService) GetDefaultMaxAge() time.Duration {
	return s.maxAge
}

func (s *JwtService) CreateStandardClaims(ts time.Time, jti string, maxAge *time.Duration, audience string, subject string) jwt.StandardClaims {
	// if maxAge != nil {
	// 	maxAge = util.DurationPtr(*maxAge)
	// }

	expiresIn := s.maxAge
	if maxAge != nil {
		expiresIn = *maxAge
	}

	iat := ts.Unix()
	exp := ts.Add(expiresIn).Unix()

	return jwt.StandardClaims{
		Id:        jti,
		Issuer:    s.issuer,
		Subject:   subject,
		IssuedAt:  iat,
		ExpiresAt: exp,
		Audience:  audience,
		NotBefore: iat,
	}
}

func (s *JwtService) CreateSignedToken(claims jwt.Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	tokenString, err := token.SignedString(s.privateKey)
	if err != nil {
		log.Printf("JwtService: Error creating token: %v", err)
		return "", err
	}

	return tokenString, nil
}
