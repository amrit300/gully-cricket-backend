package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net/url"
	"sort"
	"strings"
)

func verifyTelegram(initData string, botToken string) bool {

	log.Println("INIT DATA RECEIVED:", initData)

	values, err := url.ParseQuery(initData)
	if err != nil {
		log.Println("Parse error:", err)
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

	log.Println("DATA STRING:", dataCheckString)

	secretMac := hmac.New(sha256.New, []byte("WebAppData"))
	secretMac.Write([]byte(botToken))
	secretKey := secretMac.Sum(nil)

	mac := hmac.New(sha256.New, secretKey)
	mac.Write([]byte(dataCheckString))

	calculatedHash := hex.EncodeToString(mac.Sum(nil))

	log.Println("TELEGRAM HASH:", hash)
	log.Println("CALCULATED HASH:", calculatedHash)

	return calculatedHash == hash
}
