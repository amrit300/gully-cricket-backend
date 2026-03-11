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
Verify Telegram WebApp initData

Telegram sends a signed payload to the Mini App.
This function verifies that signature.
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

	secret := sha256.Sum256([]byte(botToken))

	h := hmac.New(sha256.New, secret[:])

	h.Write([]byte(dataCheckString))

	calculatedHash := hex.EncodeToString(h.Sum(nil))

	return calculatedHash == hash
}
