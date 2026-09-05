package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	stackin "github.com/stackin-io/stackin-go-sdk"
)

func main() {
	godotenv.Load()
	invoiceID := "00000000-0000-0000-0000-000000000000"

	client := stackin.NewInvoice(stackin.WithAPIKey(os.Getenv("STACKIN_API_KEY")))
	result, err := client.Reissue(invoiceID)

	switch e := err.(type) {
	case nil:
		fmt.Println("Reissued:", result)
	case *stackin.ConnectionFailedError:
		fmt.Println("Could not reach the platform")
	case *stackin.APIError:
		fmt.Printf("Request rejected (%d): %s\n", e.StatusCode, e.Detail)
	default:
		fmt.Println("Error:", err)
	}
}
