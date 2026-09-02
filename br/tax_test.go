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
	if data["vTotTrib"] != "10.00" {
		t.Errorf("vTotTrib = %v, want 10.00", data["vTotTrib"])
	}
}

func TestTaxToDictIcms00(t *testing.T) {
	data := Tax{Icms: Icms00{Orig: "0", CST: "00", ModBC: "3", VBC: "100.00", PICMS: "18.0000", VICMS: "18.00"}}.ToDict()
	icms := data["ICMS"].(map[string]any)
	inner, ok := icms["ICMS00"].(map[string]any)
	if !ok {
		t.Fatalf("ICMS00 missing or wrong type: %v", icms)
	}
	if inner["vICMS"] != "18.00" {
		t.Errorf("vICMS = %v, want 18.00", inner["vICMS"])
	}
}

func TestTaxToDictIcms40(t *testing.T) {
	data := Tax{Icms: Icms40{Orig: "0", CST: "40"}}.ToDict()
	icms := data["ICMS"].(map[string]any)
	if _, ok := icms["ICMS40"]; !ok {
		t.Errorf("ICMS = %v, want ICMS40 key", icms)
	}
}

func TestTaxToDictIcms60(t *testing.T) {
	data := Tax{Icms: NewIcms60("0")}.ToDict()
	icms := data["ICMS"].(map[string]any)
	if _, ok := icms["ICMS60"]; !ok {
		t.Errorf("ICMS = %v, want ICMS60 key", icms)
	}
}

func TestTaxToDictIcmsSn101(t *testing.T) {
	data := Tax{Icms: NewIcmsSn101("0", "1.5000", "0.10")}.ToDict()
	icms := data["ICMS"].(map[string]any)
	inner := icms["ICMSSN101"].(map[string]any)
	if inner["CSOSN"] != "101" {
		t.Errorf("CSOSN = %v, want 101", inner["CSOSN"])
	}
}

func TestTaxToDictIcmsSn102(t *testing.T) {
	data := Tax{Icms: IcmsSn102{Orig: ptr("0"), CSOSN: "102"}}.ToDict()
	icms := data["ICMS"].(map[string]any)
	if _, ok := icms["ICMSSN102"]; !ok {
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
	icms := data["ICMS"].(map[string]any)
	inner := icms["ICMSSN900"].(map[string]any)
	if inner["vICMS"] != "12.22" {
		t.Errorf("vICMS = %v, want 12.22", inner["vICMS"])
	}
}

func TestTaxToDictIcmsUfDest(t *testing.T) {
	data := Tax{
		IcmsUfDest: &IcmsUfDest{
			VBCUFDest: "101.84", PICMSUFDest: "17.0000", PICMSInter: "12.00",
			PICMSInterPart: "100.0000", VICMSUFDest: "5.09", VICMSUFRemet: "0.00",
		},
	}.ToDict()
	ufDest := data["ICMSUFDest"].(map[string]any)
	if ufDest["vBCUFDest"] != "101.84" {
		t.Errorf("vBCUFDest = %v, want 101.84", ufDest["vBCUFDest"])
	}
}

func TestTaxToDictIpiTrib(t *testing.T) {
	data := Tax{Ipi: &Ipi{CEnq: "999", Trib: IpiTrib{CST: "00", VIPI: "0.00"}}}.ToDict()
	ipi := data["IPI"].(map[string]any)
	if ipi["cEnq"] != "999" {
		t.Errorf("cEnq = %v, want 999", ipi["cEnq"])
	}
	trib := ipi["IPITrib"].(map[string]any)
	if trib["CST"] != "00" {
		t.Errorf("CST = %v, want 00", trib["CST"])
	}
}

func TestTaxToDictIpiNt(t *testing.T) {
	data := Tax{Ipi: &Ipi{CEnq: "999", Trib: IpiNt{CST: "53"}}}.ToDict()
	ipi := data["IPI"].(map[string]any)
	if _, ok := ipi["IPINT"]; !ok {
		t.Errorf("IPI = %v, want IPINT key", ipi)
	}
}

func TestTaxToDictIpiWithoutTrib(t *testing.T) {
	data := Tax{Ipi: &Ipi{CEnq: "999"}}.ToDict()
	ipi := data["IPI"].(map[string]any)
	if len(ipi) != 1 {
		t.Errorf("IPI = %v, want only cEnq", ipi)
	}
}

func TestTaxToDictPisAliq(t *testing.T) {
	data := Tax{Pis: PisAliq{CST: "01", VBC: "100.00", PPIS: "0.6500", VPIS: "0.65"}}.ToDict()
	pis := data["PIS"].(map[string]any)
	if _, ok := pis["PISAliq"]; !ok {
		t.Errorf("PIS = %v, want PISAliq key", pis)
	}
}

func TestTaxToDictPisNt(t *testing.T) {
	data := Tax{Pis: PisNt{CST: "07"}}.ToDict()
	pis := data["PIS"].(map[string]any)
	if _, ok := pis["PISNT"]; !ok {
		t.Errorf("PIS = %v, want PISNT key", pis)
	}
}

func TestTaxToDictPisOutr(t *testing.T) {
	data := Tax{Pis: PisOutr{CST: "99", VPIS: "0.00"}}.ToDict()
	pis := data["PIS"].(map[string]any)
	if _, ok := pis["PISOutr"]; !ok {
		t.Errorf("PIS = %v, want PISOutr key", pis)
	}
}

func TestTaxToDictCofinsAliq(t *testing.T) {
	data := Tax{Cofins: CofinsAliq{CST: "01", VBC: "100.00", PCofins: "3.0000", VCofins: "3.00"}}.ToDict()
	cofins := data["COFINS"].(map[string]any)
	if _, ok := cofins["COFINSAliq"]; !ok {
		t.Errorf("COFINS = %v, want COFINSAliq key", cofins)
	}
}

func TestTaxToDictCofinsNt(t *testing.T) {
	data := Tax{Cofins: CofinsNt{CST: "07"}}.ToDict()
	cofins := data["COFINS"].(map[string]any)
	if _, ok := cofins["COFINSNT"]; !ok {
		t.Errorf("COFINS = %v, want COFINSNT key", cofins)
	}
}

func TestTaxToDictCofinsOutr(t *testing.T) {
	data := Tax{Cofins: CofinsOutr{CST: "99", VCofins: "0.00"}}.ToDict()
	cofins := data["COFINS"].(map[string]any)
	if _, ok := cofins["COFINSOutr"]; !ok {
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

	for _, key := range []string{"vTotTrib", "ICMS", "ICMSUFDest", "IPI", "PIS", "COFINS"} {
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
