package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"sort"
	"strings"
)

/*
Verify Telegram WebApp initData signature
*/

func verifyTelegram(initData string, botToken string) bool {

	values, err := url.ParseQuery(initData)
	if err != nil {
		return false
	}

	hash := values.Get("hash")
	values.Del("hash")

	var data []string

	for k, v := range values {
		data = append(data, k+"="+v[0])
	}

	sort.Strings(data)

	dataCheckString := strings.Join(data, "\n")

	/*
	Generate Telegram secret key
	secret = HMAC_SHA256("WebAppData", bot_token)
	*/

	secretKeyMac := hmac.New(sha256.New, []byte("WebAppData"))
	secretKeyMac.Write([]byte(botToken))
	secretKey := secretKeyMac.Sum(nil)

	/*
	Compute final hash
	*/

	mac := hmac.New(sha256.New, secretKey)
	mac.Write([]byte(dataCheckString))

	calculatedHash := hex.EncodeToString(mac.Sum(nil))

	return calculatedHash == hash
}
