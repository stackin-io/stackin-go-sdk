package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	stackin "github.com/stackin-io/stackin-go-sdk"
	"github.com/stackin-io/stackin-go-sdk/br"
)

func ptr[T any](value T) *T {
	return &value
}

func main() {
	godotenv.Load()
	client := stackin.NewInvoice(stackin.WithAPIKey(os.Getenv("STACKIN_API_KEY")))

	product := br.Product{
		Description:      "Produto completo - todos os campos",
		UnitPrice:        ptr(999.99),
		NCM:              ptr("84713012"),
		CFOP:             ptr("5102"),
		Unit:             "UN",
		Quantity:         2,
		Barcode:          ptr("7891000100103"),
		CEST:             ptr("0300700"),
		NVECodes:         []string{"NV0001", "NV0002"},
		IndEscala:        ptr("N"),
		ManufacturerCNPJ: ptr("12345678000195"),
		TaxBenefitCode:   ptr("PR820001"),
		PresumedCredits: []br.PresumedCredit{
			{Code: "PR820001", Percentage: 3.0, Amount: 30.00},
		},
		ExTipi:                     ptr("01"),
		Freight:                    ptr(20.00),
		Insurance:                  ptr(8.00),
		Discount:                   ptr(15.00),
		OtherExpenses:              ptr(5.00),
		UsedMovableAsset:           false,
		PurchaseOrder:              ptr("PC-2026-00042"),
		PurchaseOrderItem:          ptr("1"),
		ImportContentControlNumber: ptr("550E8400-E29B-41D4-A716-446655440000"),
		RecopiNumber:               ptr("00000000000012345678"),
		ExtraGroups:                map[string]any{},
	}

	result, err := client.Issue(stackin.IssueRequest{
		DocumentType: stackin.NFE,
		ClientName:   "Comprador Teste Ltda",
		TaxID:        "11222333000181",
		Items:        []br.Product{product},
		RecipientAddress: &stackin.Address{
			Street:       "Rua das Palmeiras",
			Number:       "100",
			Neighborhood: "Centro",
			City:         "Florianopolis",
			State:        "SC",
			ZipCode:      "88010000",
			CityCode:     "4205407",
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(result)
}
