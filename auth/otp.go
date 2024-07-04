package auth

import (
	"fmt"
	"image/png"
	"io"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

type Totp interface {
	GenerateCode() (string, error)
	ValidateCode(code string) (valid bool, err error)
	WriteQRCode(w io.Writer) error
	ShowInfo() error
}

func NewTotp(host, account, secret string, PeriodSecs uint) Totp {
	return &totpConf{
		Host:    host,
		Account: account,
		Secret:  secret,
		Period:  PeriodSecs,
	}
}

type totpConf struct {
	Host    string
	Account string
	Secret  string
	Period  uint
}

func (tc *totpConf) generateKey() (*otp.Key, error) {
	return totp.Generate(totp.GenerateOpts{
		Issuer:      tc.Host,
		AccountName: tc.Account,
		Secret:      []byte(tc.Secret),
		Period:      tc.Period,
	})
}

func (tc *totpConf) GenerateCode() (code string, err error) {
	key, err := tc.generateKey()
	if key == nil {
		return
	}
	code, err = totp.GenerateCodeCustom(key.Secret(), time.Now().UTC(), totp.ValidateOpts{
		Period:    tc.Period,
		Skew:      1,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	return
}

func (tc *totpConf) ValidateCode(code string) (valid bool, err error) {
	key, err := tc.generateKey()
	if key == nil {
		return
	}
	valid, err = totp.ValidateCustom(
		code,
		key.Secret(),
		time.Now().UTC(),
		totp.ValidateOpts{
			Period:    tc.Period,
			Skew:      1,
			Digits:    otp.DigitsSix,
			Algorithm: otp.AlgorithmSHA1,
		},
	)
	return
}

func (tc *totpConf) WriteQRCode(w io.Writer) error {
	key, err := tc.generateKey()
	if key == nil {
		return err
	}
	img, err := key.Image(200, 200)
	if err != nil {
		return err
	}
	return png.Encode(w, img)
}

func (tc *totpConf) ShowInfo() error {
	key, err := tc.generateKey()
	if key == nil {
		return err
	}
	fmt.Printf("Issuer:       %s\n", key.Issuer())
	fmt.Printf("Account Name: %s\n", key.AccountName())
	fmt.Printf("Secret:       %s\n", key.Secret())
	return nil
}
