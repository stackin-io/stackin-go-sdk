package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	stackin "github.com/stackin-io/stackin-go-sdk"
)

func main() {
	godotenv.Load()

	client := stackin.NewFiscalReference(stackin.WithAPIKey(os.Getenv("STACKIN_API_KEY")))

	ncm, err := client.NCM.Get("84716052")
	if report(err) {
		return
	}
	fmt.Println("NCM:", ncm["description"])
	fmt.Println("Extras:", ncm["metadata"])

	found, err := client.NCM.Search(stackin.SearchQuery{Term: "teclado", Limit: 5})
	if report(err) {
		return
	}
	fmt.Printf("\n%v match 'teclado'; first page:\n", found["total"])
	if rows, ok := found["data"].([]any); ok {
		for _, row := range rows {
			if item, ok := row.(map[string]any); ok {
				fmt.Println(" ", item["code"], item["description"])
			}
		}
	}

	kinds, err := client.Kinds()
	if report(err) {
		return
	}
	fmt.Println("\nClassifications available:", kinds)

	cfop, err := client.Kind("cfop").Get("5102")
	if report(err) {
		return
	}
	fmt.Println("Any kind by name:", cfop["description"])
}

func report(err error) bool {
	switch e := err.(type) {
	case nil:
		return false
	case *stackin.ConnectionFailedError:
		fmt.Println("Could not reach the platform")
	case *stackin.APIError:
		fmt.Printf("Request rejected (%d): %s\n", e.StatusCode, e.Detail)
	default:
		fmt.Println("Error:", err)
	}
	return true
}
