package auth

import (
	"bytes"
	"compress/zlib"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"encoding/base64"
	"io"
	"os"
	"time"

	"github.com/pkg/errors"

	jwt "github.com/dgrijalva/jwt-go"
)

type JwtToken interface {
	GetToken(host string, data map[string]interface{}, exp uint8) (*string, error)
	GetTokenWithRefresh(host string, data map[string]interface{}, exp uint8) (*token, error)
	ParseToken(tokenStr string) (*jwt.Token, error)
	ParseTokenUnValidate(tokenStr string) (*jwt.Token, error)
	// 對特定資源存取金鑰
	GetAccessToken(host string, source string, id interface{}, db string, perm ApiPerm) (*string, error)
	RefreshAccessToken(refreshToken string) (*string, error)
}

type JwtDI interface {
	GetKid() string
	NewJwt() JwtToken
}

type JwtConf struct {
	PrivateKeyFile string `yaml:"privatekey"`
	PublicKeyFile  string `yaml:"publickey"`
	Header         struct {
		Kid string `yaml:"kid"`
	} `yaml:"header"`
	Claims struct {
		ExpDuration time.Duration `yaml:"exp"`
	} `yaml:"claims"`
	RefreshSecret string `yaml:"refresh_secret"`

	myHeader   map[string]interface{}
	publicKey  *rsa.PublicKey
	privateKey *rsa.PrivateKey
}

func (j *JwtConf) getHeader() map[string]interface{} {
	if j.myHeader != nil {
		delete(j.myHeader, "usa")
		return j.myHeader
	}
	j.myHeader = map[string]interface{}{
		"kid": j.Header.Kid,
	}
	return j.myHeader
}

func (j *JwtConf) getPublicKey() (*rsa.PublicKey, error) {
	if j.publicKey != nil {
		return j.publicKey, nil
	}
	publicData, err := os.ReadFile(j.PublicKeyFile)
	if err != nil {
		return nil, err
	}
	j.publicKey, err = jwt.ParseRSAPublicKeyFromPEM(publicData)
	return j.publicKey, err
}

func (j *JwtConf) getPrivateKey() (*rsa.PrivateKey, error) {
	if j.privateKey != nil {
		return j.privateKey, nil
	}
	privateData, err := os.ReadFile(j.PrivateKeyFile)
	if err != nil {
		return nil, err
	}
	j.privateKey, err = jwt.ParseRSAPrivateKeyFromPEM(privateData)
	return j.privateKey, err
}

func (j *JwtConf) GetKid() string {
	return j.Header.Kid
}

func (j *JwtConf) NewJwt() JwtToken {
	return j
}

func (j *JwtConf) ParseTokenUnValidate(tokenStr string) (*jwt.Token, error) {
	if j == nil {
		return nil, errors.New("jwtConf is nil")
	}
	parser := jwt.Parser{
		SkipClaimsValidation: true,
	}

	token, err := parser.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		pk, err := j.getPublicKey()
		return pk, err
	})
	if token == nil {
		return nil, errors.New("token is nil")
	}
	if token.Valid {
		return token, nil
	} else if ve, ok := err.(*jwt.ValidationError); ok {
		if ve.Errors&jwt.ValidationErrorMalformed != 0 {
			return nil, errors.New("That's not even a token")
		} else if ve.Errors&(jwt.ValidationErrorExpired|jwt.ValidationErrorNotValidYet) != 0 {
			// Token is either expired or not active yet
			return nil, errors.New("Timing is everything")
		} else {
			return nil, err
		}
	}
	return nil, err
}
func (j *JwtConf) ParseToken(tokenStr string) (*jwt.Token, error) {
	if j == nil {
		return nil, errors.New("jwtConf is nil")
	}
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		pk, err := j.getPublicKey()
		return pk, err
	})
	if token == nil {
		return nil, errors.New("token is nil")
	}
	if token.Valid {
		return token, nil
	} else if ve, ok := err.(*jwt.ValidationError); ok {
		if ve.Errors&jwt.ValidationErrorMalformed != 0 {
			return nil, errors.New("That's not even a token")
		} else if ve.Errors&(jwt.ValidationErrorExpired|jwt.ValidationErrorNotValidYet) != 0 {
			// Token is either expired or not active yet
			return nil, errors.New("Timing is everything")
		} else {
			return nil, err
		}
	}
	return nil, err
}

func (j *JwtConf) GetToken(host string, data map[string]interface{}, exp uint8) (*string, error) {
	if j == nil {
		return nil, errors.New("jwtConf not set")
	}
	if data == nil {
		return nil, errors.New("no data")
	}
	if exp <= 0 {
		exp = 60
	} else if exp > 180 {
		exp = 180
	}

	now := time.Now()
	data["iss"] = host
	data["iat"] = now.Unix()
	if exp > 0 {
		data["exp"] = now.Add(time.Duration(exp) * time.Minute).Unix()
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims(data))

	for k, v := range j.getHeader() {
		token.Header[k] = v
	}

	pk, err := j.getPrivateKey()
	if err != nil {
		return nil, err
	}
	ss, err := token.SignedString(pk)
	if err != nil {
		return nil, err
	}
	return &ss, nil
}

func (j *JwtConf) GetTokenWithRefresh(host string, data map[string]interface{}, exp uint8) (*token, error) {
	if j.RefreshSecret == "" {
		return nil, errors.New("refresh secret not set")
	}

	t, err := j.GetToken(host, data, exp)
	if err != nil {
		return nil, err
	}

	refreshToken, err := j.createRefreshToken(host, data)
	if err != nil {
		return nil, err
	}
	return &token{AccessToken: *t, RefreshToken: refreshToken}, nil
}

func (j *JwtConf) RefreshAccessToken(refreshToken string) (*string, error) {
	if j == nil {
		return nil, errors.New("jwtConf not set")
	}
	host, data, err := j.pareserRefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}
	return j.GetToken(host, data, 60)
}

func (j *JwtConf) GetAccessToken(host string, source string, id interface{}, db string, perm ApiPerm) (*string, error) {
	if j == nil {
		return nil, errors.New("jwtConf not set")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims(map[string]interface{}{
		"iss":      host,
		"source":   source,
		"sourceId": id,
		"db":       db,
		"per":      perm,
	}))

	token.Header = j.getHeader()
	token.Header["usa"] = "access"

	pk, err := j.getPrivateKey()
	if err != nil {
		return nil, err
	}
	ss, err := token.SignedString(pk)
	if err != nil {
		return nil, err
	}
	return &ss, nil
}

type token struct {
	AccessToken  string
	RefreshToken string
}

func (j *JwtConf) pareserRefreshToken(refreshToken string) (host string, data map[string]any, err error) {
	sha1 := sha1.New()
	io.WriteString(sha1, j.RefreshSecret)

	salt := string(sha1.Sum(nil))[0:16]
	block, err := aes.NewCipher([]byte(salt))
	if err != nil {
		return "", nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", nil, err
	}

	decodeData, err := base64.URLEncoding.DecodeString(refreshToken)
	if err != nil {
		return "", nil, err
	}

	nonceSize := gcm.NonceSize()
	nonce, ciphertext := decodeData[:nonceSize], decodeData[nonceSize:]

	compressData, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", nil, err
	}

	b := bytes.NewReader(compressData)
	var out bytes.Buffer
	r, _ := zlib.NewReader(b)
	io.Copy(&out, r)
	refreshStr := out.String()
	jwtToken, err := jwt.Parse(refreshStr, func(token *jwt.Token) (interface{}, error) {
		return []byte(j.RefreshSecret), nil
	})
	if err != nil {
		return "", nil, err
	}
	data = jwtToken.Claims.(jwt.MapClaims)
	host = data["iss"].(string)
	delete(data, "iss")
	delete(data, "iat")
	delete(data, "exp")
	return

}

func (j *JwtConf) createRefreshToken(host string, data map[string]any) (string, error) {
	now := time.Now()
	data["iss"] = host
	data["iat"] = now.Unix()
	data["exp"] = now.AddDate(0, 0, 1).Unix()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims(data))
	for k, v := range j.getHeader() {
		token.Header[k] = v
	}
	refreshToken, err := token.SignedString([]byte(j.RefreshSecret))
	if err != nil {
		return "", err
	}

	sha1 := sha1.New()
	io.WriteString(sha1, j.RefreshSecret)

	salt := string(sha1.Sum(nil))[0:16]
	block, err := aes.NewCipher([]byte(salt))
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		return "", err
	}
	var in bytes.Buffer
	w := zlib.NewWriter(&in)
	w.Write([]byte(refreshToken))
	w.Close()

	return base64.URLEncoding.EncodeToString(gcm.Seal(nonce, nonce, in.Bytes(), nil)), nil
}
