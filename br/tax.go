package br

type Icms00 struct {
	Orig  string  `json:"orig"`
	CST   string  `json:"CST"`
	ModBC string  `json:"modBC"`
	VBC   string  `json:"vBC"`
	PICMS string  `json:"pICMS"`
	VICMS string  `json:"vICMS"`
	PFCP  *string `json:"pFCP,omitempty"`
	VFCP  *string `json:"vFCP,omitempty"`
}

func (Icms00) icmsTag() string { return "ICMS00" }

func NewIcms00(orig, modBC, vBC, pICMS, vICMS string) Icms00 {
	return Icms00{Orig: orig, CST: "00", ModBC: modBC, VBC: vBC, PICMS: pICMS, VICMS: vICMS}
}

type Icms40 struct {
	Orig       string  `json:"orig"`
	CST        string  `json:"CST"`
	VICMSDeson *string `json:"vICMSDeson,omitempty"`
	MotDesICMS *string `json:"motDesICMS,omitempty"`
}

func (Icms40) icmsTag() string { return "ICMS40" }

type Icms60 struct {
	Orig            string  `json:"orig"`
	CST             string  `json:"CST"`
	VBCSTRet        *string `json:"vBCSTRet,omitempty"`
	PST             *string `json:"pST,omitempty"`
	VICMSSubstituto *string `json:"vICMSSubstituto,omitempty"`
	VICMSSTRet      *string `json:"vICMSSTRet,omitempty"`
	VBCFCPSTRet     *string `json:"vBCFCPSTRet,omitempty"`
	PFCPSTRet       *string `json:"pFCPSTRet,omitempty"`
	VFCPSTRet       *string `json:"vFCPSTRet,omitempty"`
	PRedBCEfet      *string `json:"pRedBCEfet,omitempty"`
	VBCEfet         *string `json:"vBCEfet,omitempty"`
	PICMSEfet       *string `json:"pICMSEfet,omitempty"`
	VICMSEfet       *string `json:"vICMSEfet,omitempty"`
}

func (Icms60) icmsTag() string { return "ICMS60" }

func NewIcms60(orig string) Icms60 {
	return Icms60{Orig: orig, CST: "60"}
}

type IcmsSn101 struct {
	Orig        string `json:"orig"`
	CSOSN       string `json:"CSOSN"`
	PCredSN     string `json:"pCredSN"`
	VCredICMSSN string `json:"vCredICMSSN"`
}

func (IcmsSn101) icmsTag() string { return "ICMSSN101" }

func NewIcmsSn101(orig, pCredSN, vCredICMSSN string) IcmsSn101 {
	return IcmsSn101{Orig: orig, CSOSN: "101", PCredSN: pCredSN, VCredICMSSN: vCredICMSSN}
}

type IcmsSn102 struct {
	Orig  *string `json:"orig,omitempty"`
	CSOSN string  `json:"CSOSN"`
}

func (IcmsSn102) icmsTag() string { return "ICMSSN102" }

type IcmsGroup interface {
	icmsTag() string
}

type IcmsUfDest struct {
	VBCUFDest      string  `json:"vBCUFDest"`
	VBCFCPUFDest   *string `json:"vBCFCPUFDest,omitempty"`
	PFCPUFDest     *string `json:"pFCPUFDest,omitempty"`
	PICMSUFDest    string  `json:"pICMSUFDest"`
	PICMSInter     string  `json:"pICMSInter"`
	PICMSInterPart string  `json:"pICMSInterPart"`
	VFCPUFDest     *string `json:"vFCPUFDest,omitempty"`
	VICMSUFDest    string  `json:"vICMSUFDest"`
	VICMSUFRemet   string  `json:"vICMSUFRemet"`
}

type IpiTrib struct {
	CST   string  `json:"CST"`
	VBC   *string `json:"vBC,omitempty"`
	PIPI  *string `json:"pIPI,omitempty"`
	QUnid *string `json:"qUnid,omitempty"`
	VUnid *string `json:"vUnid,omitempty"`
	VIPI  string  `json:"vIPI"`
}

func (IpiTrib) ipiTag() string { return "IPITrib" }

type IpiNt struct {
	CST string `json:"CST"`
}

func (IpiNt) ipiTag() string { return "IPINT" }

type IpiVariant interface {
	ipiTag() string
}

type Ipi struct {
	CEnq string     `json:"cEnq"`
	Trib IpiVariant `json:"-"`
}

func (i Ipi) toDict() map[string]any {
	data := map[string]any{"cEnq": i.CEnq}
	if i.Trib != nil {
		data[i.Trib.ipiTag()] = toDict(i.Trib)
	}
	return data
}

type PisAliq struct {
	CST  string `json:"CST"`
	VBC  string `json:"vBC"`
	PPIS string `json:"pPIS"`
	VPIS string `json:"vPIS"`
}

func (PisAliq) pisTag() string { return "PISAliq" }

type PisNt struct {
	CST string `json:"CST"`
}

func (PisNt) pisTag() string { return "PISNT" }

type PisOutr struct {
	CST  string  `json:"CST"`
	VBC  *string `json:"vBC,omitempty"`
	PPIS *string `json:"pPIS,omitempty"`
	VPIS string  `json:"vPIS"`
}

func (PisOutr) pisTag() string { return "PISOutr" }

type PisGroup interface {
	pisTag() string
}

type CofinsAliq struct {
	CST     string `json:"CST"`
	VBC     string `json:"vBC"`
	PCofins string `json:"pCOFINS"`
	VCofins string `json:"vCOFINS"`
}

func (CofinsAliq) cofinsTag() string { return "COFINSAliq" }

type CofinsNt struct {
	CST string `json:"CST"`
}

func (CofinsNt) cofinsTag() string { return "COFINSNT" }

type CofinsOutr struct {
	CST     string  `json:"CST"`
	VBC     *string `json:"vBC,omitempty"`
	PCofins *string `json:"pCOFINS,omitempty"`
	VCofins string  `json:"vCOFINS"`
}

func (CofinsOutr) cofinsTag() string { return "COFINSOutr" }

type CofinsGroup interface {
	cofinsTag() string
}

type Tax struct {
	VTotTrib   *string
	Icms       IcmsGroup
	IcmsUfDest *IcmsUfDest
	Ipi        *Ipi
	Pis        PisGroup
	Cofins     CofinsGroup
}

func (t Tax) ToDict() map[string]any {
	data := map[string]any{}
	if t.VTotTrib != nil {
		data["vTotTrib"] = *t.VTotTrib
	}
	if t.Icms != nil {
		data["ICMS"] = map[string]any{t.Icms.icmsTag(): toDict(t.Icms)}
	}
	if t.IcmsUfDest != nil {
		data["ICMSUFDest"] = toDict(*t.IcmsUfDest)
	}
	if t.Ipi != nil {
		data["IPI"] = t.Ipi.toDict()
	}
	if t.Pis != nil {
		data["PIS"] = map[string]any{t.Pis.pisTag(): toDict(t.Pis)}
	}
	if t.Cofins != nil {
		data["COFINS"] = map[string]any{t.Cofins.cofinsTag(): toDict(t.Cofins)}
	}
	return data
}
