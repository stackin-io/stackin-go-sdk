package br

import "testing"

func TestTaxToDictEmpty(t *testing.T) {
	data := Tax{}.ToDict()
	if len(data) != 0 {
		t.Errorf("ToDict() = %v, want empty map", data)
	}
}

func TestTaxToDictVTotTrib(t *testing.T) {
	data := Tax{VTotTrib: ptr("10.00")}.ToDict()
	if data["v_tot_trib"] != "10.00" {
		t.Errorf("vTotTrib = %v, want 10.00", data["v_tot_trib"])
	}
}

func TestTaxToDictIcms00(t *testing.T) {
	data := Tax{Icms: Icms00{Orig: "0", CST: "00", ModBC: "3", VBC: "100.00", PICMS: "18.0000", VICMS: "18.00"}}.ToDict()
	icms := data["icms"].(map[string]any)
	inner, ok := icms, true
	if !ok {
		t.Fatalf("ICMS00 missing or wrong type: %v", icms)
	}
	if inner["v_icms"] != "18.00" {
		t.Errorf("vICMS = %v, want 18.00", inner["v_icms"])
	}
}

func TestTaxToDictIcms40(t *testing.T) {
	data := Tax{Icms: Icms40{Orig: "0", CST: "40"}}.ToDict()
	icms := data["icms"].(map[string]any)
	if icms["cst"] == nil {
		t.Errorf("ICMS = %v, want ICMS40 key", icms)
	}
}

func TestTaxToDictIcms60(t *testing.T) {
	data := Tax{Icms: NewIcms60("0")}.ToDict()
	icms := data["icms"].(map[string]any)
	if icms["cst"] == nil {
		t.Errorf("ICMS = %v, want ICMS60 key", icms)
	}
}

func TestTaxToDictIcmsSn101(t *testing.T) {
	data := Tax{Icms: NewIcmsSn101("0", "1.5000", "0.10")}.ToDict()
	icms := data["icms"].(map[string]any)
	inner := icms
	if inner["csosn"] != "101" {
		t.Errorf("CSOSN = %v, want 101", inner["csosn"])
	}
}

func TestTaxToDictIcmsSn102(t *testing.T) {
	data := Tax{Icms: IcmsSn102{Orig: ptr("0"), CSOSN: "102"}}.ToDict()
	icms := data["icms"].(map[string]any)
	if icms["csosn"] == nil {
		t.Errorf("ICMS = %v, want ICMSSN102 key", icms)
	}
}

func TestTaxToDictIcmsSn900(t *testing.T) {
	data := Tax{
		Icms: IcmsSn900{
			Orig: ptr("0"), CSOSN: "900", ModBC: ptr("3"),
			VBC: ptr("101.84"), PICMS: ptr("12.0000"), VICMS: ptr("12.22"),
		},
	}.ToDict()
	icms := data["icms"].(map[string]any)
	inner := icms
	if inner["v_icms"] != "12.22" {
		t.Errorf("vICMS = %v, want 12.22", inner["v_icms"])
	}
}

func TestTaxToDictIcmsUfDest(t *testing.T) {
	data := Tax{
		IcmsUfDest: &IcmsUfDest{
			VBCUFDest: "101.84", PICMSUFDest: "17.0000", PICMSInter: "12.00",
			PICMSInterPart: "100.0000", VICMSUFDest: "5.09", VICMSUFRemet: "0.00",
		},
	}.ToDict()
	ufDest := data["icms_uf_dest"].(map[string]any)
	if ufDest["v_bc_uf_dest"] != "101.84" {
		t.Errorf("vBCUFDest = %v, want 101.84", ufDest["v_bc_uf_dest"])
	}
}

func TestTaxToDictIpiTrib(t *testing.T) {
	data := Tax{Ipi: &Ipi{CEnq: "999", Trib: IpiTrib{CST: "00", VIPI: "0.00"}}}.ToDict()
	ipi := data["ipi"].(map[string]any)
	if ipi["c_enq"] != "999" {
		t.Errorf("cEnq = %v, want 999", ipi["c_enq"])
	}
	trib := ipi["trib"].(map[string]any)
	if trib["cst"] != "00" {
		t.Errorf("CST = %v, want 00", trib["cst"])
	}
}

func TestTaxToDictIpiNt(t *testing.T) {
	data := Tax{Ipi: &Ipi{CEnq: "999", Trib: IpiNt{CST: "53"}}}.ToDict()
	ipi := data["ipi"].(map[string]any)
	trib, ok := ipi["trib"].(map[string]any)
	if !ok || trib["cst"] != "53" {
		t.Errorf("ipi = %v, want trib.cst 53", ipi)
	}
}

func TestTaxToDictIpiWithoutTrib(t *testing.T) {
	data := Tax{Ipi: &Ipi{CEnq: "999"}}.ToDict()
	ipi := data["ipi"].(map[string]any)
	if len(ipi) != 1 {
		t.Errorf("IPI = %v, want only cEnq", ipi)
	}
}

func TestTaxToDictPisAliq(t *testing.T) {
	data := Tax{Pis: PisAliq{CST: "01", VBC: "100.00", PPIS: "0.6500", VPIS: "0.65"}}.ToDict()
	pis := data["pis"].(map[string]any)
	if pis["cst"] == nil {
		t.Errorf("PIS = %v, want PISAliq key", pis)
	}
}

func TestTaxToDictPisNt(t *testing.T) {
	data := Tax{Pis: PisNt{CST: "07"}}.ToDict()
	pis := data["pis"].(map[string]any)
	if pis["cst"] == nil {
		t.Errorf("PIS = %v, want PISNT key", pis)
	}
}

func TestTaxToDictPisOutr(t *testing.T) {
	data := Tax{Pis: PisOutr{CST: "99", VPIS: "0.00"}}.ToDict()
	pis := data["pis"].(map[string]any)
	if pis["cst"] == nil {
		t.Errorf("PIS = %v, want PISOutr key", pis)
	}
}

func TestTaxToDictCofinsAliq(t *testing.T) {
	data := Tax{Cofins: CofinsAliq{CST: "01", VBC: "100.00", PCofins: "3.0000", VCofins: "3.00"}}.ToDict()
	cofins := data["cofins"].(map[string]any)
	if cofins["cst"] == nil {
		t.Errorf("COFINS = %v, want COFINSAliq key", cofins)
	}
}

func TestTaxToDictCofinsNt(t *testing.T) {
	data := Tax{Cofins: CofinsNt{CST: "07"}}.ToDict()
	cofins := data["cofins"].(map[string]any)
	if cofins["cst"] == nil {
		t.Errorf("COFINS = %v, want COFINSNT key", cofins)
	}
}

func TestTaxToDictCofinsOutr(t *testing.T) {
	data := Tax{Cofins: CofinsOutr{CST: "99", VCofins: "0.00"}}.ToDict()
	cofins := data["cofins"].(map[string]any)
	if cofins["cst"] == nil {
		t.Errorf("COFINS = %v, want COFINSOutr key", cofins)
	}
}

func TestTaxToDictFullCombination(t *testing.T) {
	data := Tax{
		VTotTrib: ptr("1.00"),
		Icms:     IcmsSn102{Orig: ptr("0"), CSOSN: "102"},
		IcmsUfDest: &IcmsUfDest{
			VBCUFDest: "1.00", PICMSUFDest: "1.00", PICMSInter: "4.00",
			PICMSInterPart: "1.00", VICMSUFDest: "1.00", VICMSUFRemet: "1.00",
		},
		Ipi:    &Ipi{CEnq: "999", Trib: IpiNt{CST: "53"}},
		Pis:    PisNt{CST: "07"},
		Cofins: CofinsNt{CST: "07"},
	}.ToDict()

	for _, key := range []string{"v_tot_trib", "icms", "icms_uf_dest", "ipi", "pis", "cofins"} {
		if _, ok := data[key]; !ok {
			t.Errorf("missing key %q in %v", key, data)
		}
	}
}

func TestNewIcms00Helper(t *testing.T) {
	icms := NewIcms00("0", "3", "100.00", "18.0000", "18.00")
	if icms.CST != "00" {
		t.Errorf("CST = %q, want 00", icms.CST)
	}
}

func TestTheGroupsTheLeiauteAddedLater(t *testing.T) {
	t.Run("monofasico carries its own fields", func(t *testing.T) {
		tax := Tax{Icms: Icms02{Orig: "0", CST: "02",
			AdRemICMS: "0.1234", VICMSMono: "1.23"}}

		group := tax.ToDict()["icms"].(map[string]any)
		if group["ad_rem_icms"] != "0.1234" || group["v_icms_mono"] != "1.23" {
			t.Fatalf("unexpected ICMS02: %v", group)
		}
	})

	t.Run("partilha names the destination state", func(t *testing.T) {
		tax := Tax{Icms: IcmsPart{Orig: "0", CST: "10", ModBC: "3",
			VBC: "100.00", PICMS: "18.00", VICMS: "18.00", ModBCST: "4",
			VBCST: "120.00", PICMSST: "18.00", VICMSST: "21.60",
			PBCOp: "100.0000", UFST: "RJ"}}

		group := tax.ToDict()["icms"].(map[string]any)
		if group["uf_st"] != "RJ" || group["p_bc_op"] != "100.0000" {
			t.Fatalf("unexpected ICMSPart: %v", group)
		}
	})

	t.Run("the substituted taxpayer group is its own variant", func(t *testing.T) {
		tax := Tax{Icms: IcmsSt{Orig: "0", CST: "60", VBCSTRet: "100.00",
			VICMSSTRet: "18.00", VBCSTDest: "120.00", VICMSSTDest: "21.60"}}

		if _, ok := tax.ToDict()["icms"]; !ok {
			t.Fatal("ICMSST missing")
		}
	})

	t.Run("the remaining Simples variants are available", func(t *testing.T) {
		cases := map[string]IcmsGroup{
			"201": IcmsSn201{Orig: "0", CSOSN: "201", ModBCST: "4",
				VBCST: "120.00", PICMSST: "18.00", VICMSST: "21.60",
				PCredSN: "2.50", VCredICMSSN: "2.50"},
			"202": IcmsSn202{Orig: "0", CSOSN: "202", ModBCST: "4",
				VBCST: "120.00", PICMSST: "18.00", VICMSST: "21.60"},
			"500": IcmsSn500{Orig: "0", CSOSN: "500"},
		}

		for csosn, group := range cases {
			tax := Tax{Icms: group}
			got := tax.ToDict()["icms"].(map[string]any)["csosn"]
			if got != csosn {
				t.Fatalf("csosn = %v, want %s", got, csosn)
			}
		}
	})

	t.Run("PIS and COFINS can be taxed by quantity", func(t *testing.T) {
		tax := Tax{
			Pis:    PisQtde{CST: "03", QBCProd: "10.0000", VAliqProd: "0.1000", VPIS: "1.00"},
			Cofins: CofinsQtde{CST: "03", QBCProd: "10.0000", VAliqProd: "0.1000", VCofins: "1.00"},
		}

		data := tax.ToDict()
		if _, ok := data["pis"]; !ok {
			t.Fatal("PISQtde missing")
		}
		if _, ok := data["cofins"]; !ok {
			t.Fatal("COFINSQtde missing")
		}
	})

	t.Run("the withheld groups sit under their own keys", func(t *testing.T) {
		tax := Tax{
			PisSt:    &PisSt{VPIS: "1.65", VBC: ptr("100.00"), PPIS: ptr("1.65")},
			CofinsSt: &CofinsSt{VCofins: "7.60", VBC: ptr("100.00"), PCofins: ptr("7.60")},
		}

		data := tax.ToDict()
		if data["pis_st"].(map[string]any)["v_pis"] != "1.65" {
			t.Fatalf("unexpected PISST: %v", data["pis_st"])
		}
		if data["cofins_st"].(map[string]any)["v_cofins"] != "7.60" {
			t.Fatalf("unexpected COFINSST: %v", data["cofins_st"])
		}
	})

	t.Run("IPI carries the stamp fields the schema allows", func(t *testing.T) {
		tax := Tax{Ipi: &Ipi{CNPJProd: ptr("11222333000181"), CSelo: ptr("001"),
			QSelo: ptr("10"), CEnq: "999", Trib: IpiNt{CST: "53"}}}

		data := tax.ToDict()["ipi"].(map[string]any)
		if data["cnpj_prod"] != "11222333000181" || data["q_selo"] != "10" {
			t.Fatalf("unexpected IPI: %v", data)
		}
		if _, ok := data["trib"]; !ok {
			t.Fatal("IPINT missing")
		}
	})
}
