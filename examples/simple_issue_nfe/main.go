package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	stackin "github.com/stackin-io/stackin-go-sdk"
	"github.com/stackin-io/stackin-go-sdk/br"
)

func ptr[T any](v T) *T {
	return &v
}

func basic() br.Product {
	return br.Product{
		Description: "Produto basico",
		Amount:      50.00,
		NCM:         ptr("84713012"),
		CFOP:        ptr("5102"),
	}
}

func withQuantity() br.Product {
	return br.Product{
		Description: "Caixa de parafusos",
		Amount:      12.50,
		NCM:         ptr("73181500"),
		CFOP:        ptr("5102"),
		Unit:        "CX",
		Quantity:    20,
	}
}

func withBarcode() br.Product {
	return br.Product{
		Description: "Produto com codigo de barras",
		Amount:      29.90,
		NCM:         ptr("21069090"),
		CFOP:        ptr("5102"),
		Barcode:     ptr("7891000100103"),
	}
}

func withTaxBenefit() br.Product {
	return br.Product{
		Description:    "Produto com beneficio fiscal",
		Amount:         80.00,
		NCM:            ptr("22021000"),
		CFOP:           ptr("5102"),
		CEST:           ptr("0300700"),
		TaxBenefitCode: ptr("PR820001"),
		PresumedCredits: []br.PresumedCredit{
			{Code: "PR820001", Percentage: 3.0, Amount: 2.40},
		},
	}
}

func scaleManufactured() br.Product {
	return br.Product{
		Description:      "Produto de fabricacao em escala",
		Amount:           150.00,
		NCM:              ptr("87141000"),
		CFOP:             ptr("5102"),
		CEST:             ptr("0100100"),
		IndEscala:        ptr("N"),
		ManufacturerCNPJ: ptr("12345678000199"),
	}
}

func withExtraCharges() br.Product {
	return br.Product{
		Description:   "Produto com encargos adicionais",
		Amount:        200.00,
		NCM:           ptr("94036000"),
		CFOP:          ptr("5102"),
		Freight:       ptr(15.00),
		Insurance:     ptr(5.00),
		Discount:      ptr(10.00),
		OtherExpenses: ptr(3.50),
	}
}

func usedAsset() br.Product {
	return br.Product{
		Description:      "Bem movel usado",
		Amount:           500.00,
		NCM:              ptr("87032310"),
		CFOP:             ptr("5102"),
		UsedMovableAsset: true,
	}
}

func withPurchaseOrder() br.Product {
	return br.Product{
		Description:       "Produto vinculado a pedido de compra",
		Amount:            75.00,
		NCM:               ptr("84433210"),
		CFOP:              ptr("5102"),
		PurchaseOrder:     ptr("PC-2026-00042"),
		PurchaseOrderItem: ptr("1"),
	}
}

func imported() br.Product {
	return br.Product{
		Description:                "Produto importado",
		Amount:                     320.00,
		NCM:                        ptr("85171231"),
		CFOP:                       ptr("5102"),
		ExTipi:                     ptr("01"),
		ImportContentControlNumber: ptr("550E8400-E29B-41D4-A716-446655440000"),
	}
}

func taxedIcms() br.Product {
	return br.Product{
		Description: "Plastico celofane 50x50",
		Amount:      0.27,
		NCM:         ptr("39202019"),
		CFOP:        ptr("6108"),
		Freight:     ptr(0.03),
		Tax: &br.Tax{
			Icms: br.Icms00{
				Orig: "0", CST: "00", ModBC: "3", VBC: "0.30",
				PICMS: "12.0000", VICMS: "0.04",
			},
			Pis: br.PisAliq{CST: "01", VBC: "0.30", PPIS: "0.6500", VPIS: "0.00"},
			Cofins: br.CofinsAliq{
				CST: "01", VBC: "0.30", PCofins: "3.0000", VCofins: "0.01",
			},
		},
	}
}

func icmsIsento() br.Product {
	return br.Product{
		Description: "Rosa Holambra Vermelha",
		Amount:      112.44,
		NCM:         ptr("06031100"),
		CFOP:        ptr("6108"),
		Quantity:    6,
		Freight:     ptr(11.05),
		Tax:         &br.Tax{Icms: br.Icms40{Orig: "0", CST: "40"}},
	}
}

func interstateWithIcmsDest() br.Product {
	return br.Product{
		Description: "Urso de Pelucia Dudu",
		Amount:      92.72,
		NCM:         ptr("95030031"),
		CFOP:        ptr("6108"),
		Freight:     ptr(9.12),
		Tax: &br.Tax{
			Icms: br.Icms00{
				Orig: "0", CST: "00", ModBC: "3", VBC: "101.84",
				PICMS: "12.0000", VICMS: "12.22",
			},
			IcmsUfDest: &br.IcmsUfDest{
				VBCUFDest: "101.84", PICMSUFDest: "17.0000",
				PICMSInter: "12.00", PICMSInterPart: "100.0000",
				VICMSUFDest: "5.09", VICMSUFRemet: "0.00",
			},
		},
	}
}

func full() br.Product {
	return br.Product{
		Description:      "Produto completo - todos os campos",
		Amount:           999.99,
		NCM:              ptr("84713012"),
		CFOP:             ptr("5102"),
		Unit:             "UN",
		Quantity:         2,
		Barcode:          ptr("7891000100103"),
		CEST:             ptr("0300700"),
		NVECodes:         []string{"NV0001", "NV0002"},
		IndEscala:        ptr("S"),
		ManufacturerCNPJ: ptr("12345678000199"),
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
}

func productCatalog() []br.Product {
	return []br.Product{
		basic(), withQuantity(), withBarcode(), withTaxBenefit(),
		scaleManufactured(), withExtraCharges(), usedAsset(),
		withPurchaseOrder(), imported(), taxedIcms(), icmsIsento(),
		interstateWithIcmsDest(), full(),
	}
}

func main() {
	godotenv.Load()
	client := stackin.NewInvoice(stackin.WithAPIKey(os.Getenv("NFE_TEST_API_KEY")))

	result, err := client.Issue(stackin.IssueRequest{
		DocumentType:     stackin.NFE,
		ClientName:       "Comprador Teste Ltda",
		TaxID:            "11222333000181",
		Items:            productCatalog(),
		RecipientAddress: &stackin.Address{
			State:        "SC",
			CityCode:     "4205407",
			Street:       "Rua das Palmeiras",
			Number:       "100",
			Neighborhood: "Centro",
			City:         "Florianopolis",
			ZipCode:      "88010000",
		},
	})

	switch e := err.(type) {
	case nil:
		fmt.Println("Issued:", result)
	case *stackin.ConnectionFailedError:
		fmt.Println("Could not reach the platform")
	case *stackin.APIError:
		fmt.Printf("Request rejected (%d): %s\n", e.StatusCode, e.Detail)
	default:
		fmt.Println("Error:", err)
	}
}
