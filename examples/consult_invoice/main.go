package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	stackin "github.com/stackin-io/stackin-go-sdk"
)

func main() {
	godotenv.Load()
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run ./examples/consult_invoice <access_key> <nfe|nfse>")
		return
	}
	accessKey, documentType := os.Args[1], stackin.DocumentType(os.Args[2])

	client := stackin.NewInvoice(stackin.WithAPIKey(os.Getenv("NFE_TEST_API_KEY")))
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
