package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	stackin "github.com/stackin-io/stackin-go-sdk"
)

func main() {
	godotenv.Load()
	accessKey := "42250611222333000181550010000000011000000017"
	documentType := stackin.NFE

	client := stackin.NewInvoice(stackin.WithAPIKey(os.Getenv("STACKIN_API_KEY")))
	result, err := client.Consult(accessKey, documentType)

	switch e := err.(type) {
	case nil:
		fmt.Println("Status:", result)
	case *stackin.ConnectionFailedError:
		fmt.Println("Could not reach the platform")
	case *stackin.APIError:
		fmt.Printf("Request rejected (%d): %s\n", e.StatusCode, e.Detail)
	default:
		fmt.Println("Error:", err)
	}
}
