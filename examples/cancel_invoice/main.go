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
	reason := "Emitida com dados incorretos do destinatario"

	client := stackin.NewInvoice(stackin.WithAPIKey(os.Getenv("STACKIN_API_KEY")))
	result, err := client.Cancel(accessKey, documentType, reason)

	switch e := err.(type) {
	case nil:
		fmt.Println("Cancelled:", result)
	case *stackin.ConnectionFailedError:
		fmt.Println("Could not reach the platform")
	case *stackin.APIError:
		fmt.Printf("Request rejected (%d): %s\n", e.StatusCode, e.Detail)
	default:
		fmt.Println("Error:", err)
	}
}
