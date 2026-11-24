package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	stackin "github.com/stackin-io/stackin-go-sdk"
)

func main() {
	godotenv.Load()
	client := stackin.NewFiscalReference(stackin.WithAPIKey(os.Getenv("STACKIN_API_KEY")))

	found, err := client.NBSSuggestions("1.07")
	if err != nil {
		log.Fatal(err)
	}

	for _, row := range found.Suggestions {
		fmt.Println(row.NBS)
	}
}
