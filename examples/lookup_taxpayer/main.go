package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	stackin "github.com/stackin-io/stackin-go-sdk"
)

func main() {
	godotenv.Load()

	client := stackin.NewTaxpayer(stackin.WithAPIKey(os.Getenv("STACKIN_API_KEY")))

	found, err := client.Get("00000000000191")

	switch e := err.(type) {
	case nil:
		fmt.Println("Name:", found["name"])
		fmt.Println("Trade name:", found["trade_name"])
		fmt.Println("City code:", found["city_code"], "State:", found["state"])
	case *stackin.ConnectionFailedError:
		fmt.Println("Could not reach the platform")
	case *stackin.APIError:
		if e.StatusCode == http.StatusNotFound {
			fmt.Println("The registry has no record of this tax id yet. It")
			fmt.Println("reloads monthly, so a recently registered company is")
			fmt.Println("simply not in it — this is not proof the company does")
			fmt.Println("not exist, and it is not a validation rule.")
			return
		}
		fmt.Printf("Request rejected (%d): %s\n", e.StatusCode, e.Detail)
	default:
		fmt.Println("Error:", err)
	}
}
