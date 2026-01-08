package main

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	passwords := map[string]string{
		"secadmin":    "@AdamLEO123311",
		"secobserver": "@ADAMobserver3211",
		"secanalyst":  "@JEXPERT_3211",
	}

	for user, pass := range passwords {
		hash, err := bcrypt.GenerateFromPassword([]byte(pass), 10)
		if err != nil {
			fmt.Println("Error:", err)
			continue
		}
		fmt.Printf("User: %s\nPassword: %s\nHash: %s\n\n", user, pass, string(hash))
	}
}
