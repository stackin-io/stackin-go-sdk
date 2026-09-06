package br

// Icms00 is ICMS00: ICMS fully taxed.
type Icms00 struct {
	Orig  string  `json:"orig"`
	CST   string  `json:"cst"`
	ModBC string  `json:"mod_bc"`
	VBC   string  `json:"v_bc"`
	PICMS string  `json:"p_icms"`
	VICMS string  `json:"v_icms"`
	PFCP  *string `json:"p_fcp,omitempty"`
	VFCP  *string `json:"v_fcp,omitempty"`
}

func (Icms00) icmsTag() string { return "ICMS00" }

// NewIcms00 fills the fields the schema requires.
func NewIcms00(orig, modBC, vBC, pICMS, vICMS string) Icms00 {
	return Icms00{CST: "00", Orig: orig, ModBC: modBC, VBC: vBC, PICMS: pICMS, VICMS: vICMS}
}

// Icms02 is ICMS02: ICMS monofasico, taxed by unit.
type Icms02 struct {
	Orig      string  `json:"orig"`
	CST       string  `json:"cst"`
	QBCMono   *string `json:"q_bc_mono,omitempty"`
	AdRemICMS string  `json:"ad_rem_icms"`
	VICMSMono string  `json:"v_icms_mono"`
}

func (Icms02) icmsTag() string { return "ICMS02" }

// NewIcms02 fills the fields the schema requires.
func NewIcms02(orig, adRemICMS, vICMSMono string) Icms02 {
	return Icms02{CST: "02", Orig: orig, AdRemICMS: adRemICMS, VICMSMono: vICMSMono}
}

// Icms10 is ICMS10: ICMS taxed with substitution.
type Icms10 struct {
	Orig         string  `json:"orig"`
	CST          string  `json:"cst"`
	ModBC        string  `json:"mod_bc"`
	VBC          string  `json:"v_bc"`
	PICMS        string  `json:"p_icms"`
	VICMS        string  `json:"v_icms"`
	VBCFCP       *string `json:"v_bc_fcp,omitempty"`
	PFCP         *string `json:"p_fcp,omitempty"`
	VFCP         *string `json:"v_fcp,omitempty"`
	ModBCST      string  `json:"mod_bc_st"`
	PMVAST       *string `json:"p_mva_st,omitempty"`
	PRedBCST     *string `json:"p_red_bc_st,omitempty"`
	VBCST        string  `json:"v_bc_st"`
	PICMSST      string  `json:"p_icms_st"`
	VICMSST      string  `json:"v_icms_st"`
	VBCFCPST     *string `json:"v_bc_fcp_st,omitempty"`
	PFCPST       *string `json:"p_fcp_st,omitempty"`
	VFCPST       *string `json:"v_fcp_st,omitempty"`
	VICMSSTDeson *string `json:"v_icms_st_deson,omitempty"`
	MotDesICMSST *string `json:"mot_des_icms_st,omitempty"`
}

func (Icms10) icmsTag() string { return "ICMS10" }

// NewIcms10 fills the fields the schema requires.
func NewIcms10(orig, modBC, vBC, pICMS, vICMS, modBCST, vBCST, pICMSST, vICMSST string) Icms10 {
	return Icms10{CST: "10", Orig: orig, ModBC: modBC, VBC: vBC, PICMS: pICMS, VICMS: vICMS, ModBCST: modBCST, VBCST: vBCST, PICMSST: pICMSST, VICMSST: vICMSST}
}

// Icms15 is ICMS15: ICMS monofasico with retention.
type Icms15 struct {
	Orig           string  `json:"orig"`
	CST            string  `json:"cst"`
	QBCMono        *string `json:"q_bc_mono,omitempty"`
	AdRemICMS      string  `json:"ad_rem_icms"`
	VICMSMono      string  `json:"v_icms_mono"`
	QBCMonoReten   *string `json:"q_bc_mono_reten,omitempty"`
	AdRemICMSReten string  `json:"ad_rem_icms_reten"`
	VICMSMonoReten string  `json:"v_icms_mono_reten"`
	PRedAdRem      *string `json:"p_red_ad_rem,omitempty"`
	MotRedAdRem    *string `json:"mot_red_ad_rem,omitempty"`
}

func (Icms15) icmsTag() string { return "ICMS15" }

// NewIcms15 fills the fields the schema requires.
func NewIcms15(orig, adRemICMS, vICMSMono, adRemICMSReten, vICMSMonoReten string) Icms15 {
	return Icms15{CST: "15", Orig: orig, AdRemICMS: adRemICMS, VICMSMono: vICMSMono, AdRemICMSReten: adRemICMSReten, VICMSMonoReten: vICMSMonoReten}
}

// Icms20 is ICMS20: ICMS with a reduced base.
type Icms20 struct {
	Orig          string  `json:"orig"`
	CST           string  `json:"cst"`
	ModBC         string  `json:"mod_bc"`
	PRedBC        string  `json:"p_red_bc"`
	VBC           string  `json:"v_bc"`
	PICMS         string  `json:"p_icms"`
	VICMS         string  `json:"v_icms"`
	VBCFCP        *string `json:"v_bc_fcp,omitempty"`
	PFCP          *string `json:"p_fcp,omitempty"`
	VFCP          *string `json:"v_fcp,omitempty"`
	VICMSDeson    *string `json:"v_icms_deson,omitempty"`
	MotDesICMS    *string `json:"mot_des_icms,omitempty"`
	IndDeduzDeson *string `json:"ind_deduz_deson,omitempty"`
}

func (Icms20) icmsTag() string { return "ICMS20" }

// NewIcms20 fills the fields the schema requires.
func NewIcms20(orig, modBC, pRedBC, vBC, pICMS, vICMS string) Icms20 {
	return Icms20{CST: "20", Orig: orig, ModBC: modBC, PRedBC: pRedBC, VBC: vBC, PICMS: pICMS, VICMS: vICMS}
}

// Icms30 is ICMS30: ICMS exempt with substitution.
type Icms30 struct {
	Orig          string  `json:"orig"`
	CST           string  `json:"cst"`
	ModBCST       string  `json:"mod_bc_st"`
	PMVAST        *string `json:"p_mva_st,omitempty"`
	PRedBCST      *string `json:"p_red_bc_st,omitempty"`
	VBCST         string  `json:"v_bc_st"`
	PICMSST       string  `json:"p_icms_st"`
	VICMSST       string  `json:"v_icms_st"`
	VBCFCPST      *string `json:"v_bc_fcp_st,omitempty"`
	PFCPST        *string `json:"p_fcp_st,omitempty"`
	VFCPST        *string `json:"v_fcp_st,omitempty"`
	VICMSDeson    *string `json:"v_icms_deson,omitempty"`
	MotDesICMS    *string `json:"mot_des_icms,omitempty"`
	IndDeduzDeson *string `json:"ind_deduz_deson,omitempty"`
}

func (Icms30) icmsTag() string { return "ICMS30" }

// NewIcms30 fills the fields the schema requires.
func NewIcms30(orig, modBCST, vBCST, pICMSST, vICMSST string) Icms30 {
	return Icms30{CST: "30", Orig: orig, ModBCST: modBCST, VBCST: vBCST, PICMSST: pICMSST, VICMSST: vICMSST}
}

// Icms40 is ICMS40: ICMS exempt or not taxed.
type Icms40 struct {
	Orig          string  `json:"orig"`
	CST           string  `json:"cst"`
	VICMSDeson    *string `json:"v_icms_deson,omitempty"`
	MotDesICMS    *string `json:"mot_des_icms,omitempty"`
	IndDeduzDeson *string `json:"ind_deduz_deson,omitempty"`
}

func (Icms40) icmsTag() string { return "ICMS40" }

// NewIcms40 fills the fields the schema requires.
func NewIcms40(orig, cST string) Icms40 {
	return Icms40{Orig: orig, CST: cST}
}

// Icms51 is ICMS51: ICMS deferred.
type Icms51 struct {
	Orig      string  `json:"orig"`
	CST       string  `json:"cst"`
	ModBC     *string `json:"mod_bc,omitempty"`
	PRedBC    *string `json:"p_red_bc,omitempty"`
	CBenefRBC *string `json:"c_benef_rbc,omitempty"`
	VBC       *string `json:"v_bc,omitempty"`
	PICMS     *string `json:"p_icms,omitempty"`
	VICMSOp   *string `json:"v_icms_op,omitempty"`
	PDif      *string `json:"p_dif,omitempty"`
	VICMSDif  *string `json:"v_icms_dif,omitempty"`
	VICMS     *string `json:"v_icms,omitempty"`
	VBCFCP    *string `json:"v_bc_fcp,omitempty"`
	PFCP      *string `json:"p_fcp,omitempty"`
	VFCP      *string `json:"v_fcp,omitempty"`
	PFCPDif   *string `json:"p_fcp_dif,omitempty"`
	VFCPDif   *string `json:"v_fcp_dif,omitempty"`
	VFCPEfet  *string `json:"v_fcp_efet,omitempty"`
}

func (Icms51) icmsTag() string { return "ICMS51" }

// NewIcms51 fills the fields the schema requires.
func NewIcms51(orig string) Icms51 {
	return Icms51{CST: "51", Orig: orig}
}

// Icms53 is ICMS53: ICMS monofasico deferred.
type Icms53 struct {
	Orig         string  `json:"orig"`
	CST          string  `json:"cst"`
	QBCMono      *string `json:"q_bc_mono,omitempty"`
	AdRemICMS    *string `json:"ad_rem_icms,omitempty"`
	VICMSMonoOp  *string `json:"v_icms_mono_op,omitempty"`
	PDif         *string `json:"p_dif,omitempty"`
	VICMSMonoDif *string `json:"v_icms_mono_dif,omitempty"`
	VICMSMono    *string `json:"v_icms_mono,omitempty"`
	QBCMonoDif   *string `json:"q_bc_mono_dif,omitempty"`
	AdRemICMSDif *string `json:"ad_rem_icms_dif,omitempty"`
}

func (Icms53) icmsTag() string { return "ICMS53" }

// NewIcms53 fills the fields the schema requires.
func NewIcms53(orig string) Icms53 {
	return Icms53{CST: "53", Orig: orig}
}

// Icms60 is ICMS60: ICMS already charged by an earlier substitution.
type Icms60 struct {
	Orig            string  `json:"orig"`
	CST             string  `json:"cst"`
	VBCSTRet        *string `json:"v_bc_st_ret,omitempty"`
	PST             *string `json:"p_st,omitempty"`
	VICMSSubstituto *string `json:"v_icms_substituto,omitempty"`
	VICMSSTRet      *string `json:"v_icms_st_ret,omitempty"`
	VBCFCPSTRet     *string `json:"v_bc_fcp_st_ret,omitempty"`
	PFCPSTRet       *string `json:"p_fcp_st_ret,omitempty"`
	VFCPSTRet       *string `json:"v_fcp_st_ret,omitempty"`
	PRedBCEfet      *string `json:"p_red_bc_efet,omitempty"`
	VBCEfet         *string `json:"v_bc_efet,omitempty"`
	PICMSEfet       *string `json:"p_icms_efet,omitempty"`
	VICMSEfet       *string `json:"v_icms_efet,omitempty"`
}

func (Icms60) icmsTag() string { return "ICMS60" }

// NewIcms60 fills the fields the schema requires.
func NewIcms60(orig string) Icms60 {
	return Icms60{CST: "60", Orig: orig}
}

// Icms61 is ICMS61: ICMS monofasico already charged earlier.
type Icms61 struct {
	Orig         string  `json:"orig"`
	CST          string  `json:"cst"`
	QBCMonoRet   *string `json:"q_bc_mono_ret,omitempty"`
	AdRemICMSRet string  `json:"ad_rem_icms_ret"`
	VICMSMonoRet string  `json:"v_icms_mono_ret"`
}

func (Icms61) icmsTag() string { return "ICMS61" }

// NewIcms61 fills the fields the schema requires.
func NewIcms61(orig, adRemICMSRet, vICMSMonoRet string) Icms61 {
	return Icms61{CST: "61", Orig: orig, AdRemICMSRet: adRemICMSRet, VICMSMonoRet: vICMSMonoRet}
}

// Icms70 is ICMS70: ICMS with a reduced base and substitution.
type Icms70 struct {
	Orig          string  `json:"orig"`
	CST           string  `json:"cst"`
	ModBC         string  `json:"mod_bc"`
	PRedBC        string  `json:"p_red_bc"`
	VBC           string  `json:"v_bc"`
	PICMS         string  `json:"p_icms"`
	VICMS         string  `json:"v_icms"`
	VBCFCP        *string `json:"v_bc_fcp,omitempty"`
	PFCP          *string `json:"p_fcp,omitempty"`
	VFCP          *string `json:"v_fcp,omitempty"`
	ModBCST       string  `json:"mod_bc_st"`
	PMVAST        *string `json:"p_mva_st,omitempty"`
	PRedBCST      *string `json:"p_red_bc_st,omitempty"`
	VBCST         string  `json:"v_bc_st"`
	PICMSST       string  `json:"p_icms_st"`
	VICMSST       string  `json:"v_icms_st"`
	VBCFCPST      *string `json:"v_bc_fcp_st,omitempty"`
	PFCPST        *string `json:"p_fcp_st,omitempty"`
	VFCPST        *string `json:"v_fcp_st,omitempty"`
	VICMSDeson    *string `json:"v_icms_deson,omitempty"`
	MotDesICMS    *string `json:"mot_des_icms,omitempty"`
	IndDeduzDeson *string `json:"ind_deduz_deson,omitempty"`
	VICMSSTDeson  *string `json:"v_icms_st_deson,omitempty"`
	MotDesICMSST  *string `json:"mot_des_icms_st,omitempty"`
}

func (Icms70) icmsTag() string { return "ICMS70" }

// NewIcms70 fills the fields the schema requires.
func NewIcms70(orig, modBC, pRedBC, vBC, pICMS, vICMS, modBCST, vBCST, pICMSST, vICMSST string) Icms70 {
	return Icms70{CST: "70", Orig: orig, ModBC: modBC, PRedBC: pRedBC, VBC: vBC, PICMS: pICMS, VICMS: vICMS, ModBCST: modBCST, VBCST: vBCST, PICMSST: pICMSST, VICMSST: vICMSST}
}

// Icms90 is ICMS90: ICMS, other cases.
type Icms90 struct {
	Orig          string  `json:"orig"`
	CST           string  `json:"cst"`
	ModBC         *string `json:"mod_bc,omitempty"`
	VBC           *string `json:"v_bc,omitempty"`
	PRedBC        *string `json:"p_red_bc,omitempty"`
	CBenefRBC     *string `json:"c_benef_rbc,omitempty"`
	PICMS         *string `json:"p_icms,omitempty"`
	VICMSOp       *string `json:"v_icms_op,omitempty"`
	PDif          *string `json:"p_dif,omitempty"`
	VICMSDif      *string `json:"v_icms_dif,omitempty"`
	VICMS         *string `json:"v_icms,omitempty"`
	VBCFCP        *string `json:"v_bc_fcp,omitempty"`
	PFCP          *string `json:"p_fcp,omitempty"`
	VFCP          *string `json:"v_fcp,omitempty"`
	PFCPDif       *string `json:"p_fcp_dif,omitempty"`
	VFCPDif       *string `json:"v_fcp_dif,omitempty"`
	VFCPEfet      *string `json:"v_fcp_efet,omitempty"`
	ModBCST       *string `json:"mod_bc_st,omitempty"`
	PMVAST        *string `json:"p_mva_st,omitempty"`
	PRedBCST      *string `json:"p_red_bc_st,omitempty"`
	VBCST         *string `json:"v_bc_st,omitempty"`
	PICMSST       *string `json:"p_icms_st,omitempty"`
	VICMSST       *string `json:"v_icms_st,omitempty"`
	VBCFCPST      *string `json:"v_bc_fcp_st,omitempty"`
	PFCPST        *string `json:"p_fcp_st,omitempty"`
	VFCPST        *string `json:"v_fcp_st,omitempty"`
	VICMSDeson    *string `json:"v_icms_deson,omitempty"`
	MotDesICMS    *string `json:"mot_des_icms,omitempty"`
	IndDeduzDeson *string `json:"ind_deduz_deson,omitempty"`
	VICMSSTDeson  *string `json:"v_icms_st_deson,omitempty"`
	MotDesICMSST  *string `json:"mot_des_icms_st,omitempty"`
}

func (Icms90) icmsTag() string { return "ICMS90" }

// NewIcms90 fills the fields the schema requires.
func NewIcms90(orig string) Icms90 {
	return Icms90{CST: "90", Orig: orig}
}

// IcmsPart is ICMSPart: ICMS split between the origin and destination states.
type IcmsPart struct {
	Orig          string  `json:"orig"`
	CST           string  `json:"cst"`
	ModBC         string  `json:"mod_bc"`
	VBC           string  `json:"v_bc"`
	PRedBC        *string `json:"p_red_bc,omitempty"`
	PICMS         string  `json:"p_icms"`
	VICMS         string  `json:"v_icms"`
	ModBCST       string  `json:"mod_bc_st"`
	PMVAST        *string `json:"p_mva_st,omitempty"`
	PRedBCST      *string `json:"p_red_bc_st,omitempty"`
	VBCST         string  `json:"v_bc_st"`
	PICMSST       string  `json:"p_icms_st"`
	VICMSST       string  `json:"v_icms_st"`
	VBCFCPST      *string `json:"v_bc_fcp_st,omitempty"`
	PFCPST        *string `json:"p_fcp_st,omitempty"`
	VFCPST        *string `json:"v_fcp_st,omitempty"`
	PBCOp         string  `json:"p_bc_op"`
	UFST          string  `json:"uf_st"`
	VICMSDeson    *string `json:"v_icms_deson,omitempty"`
	MotDesICMS    *string `json:"mot_des_icms,omitempty"`
	IndDeduzDeson *string `json:"ind_deduz_deson,omitempty"`
}

func (IcmsPart) icmsTag() string { return "ICMSPart" }

// NewIcmsPart fills the fields the schema requires.
func NewIcmsPart(orig, cST, modBC, vBC, pICMS, vICMS, modBCST, vBCST, pICMSST, vICMSST, pBCOp, uFST string) IcmsPart {
	return IcmsPart{Orig: orig, CST: cST, ModBC: modBC, VBC: vBC, PICMS: pICMS, VICMS: vICMS, ModBCST: modBCST, VBCST: vBCST, PICMSST: pICMSST, VICMSST: vICMSST, PBCOp: pBCOp, UFST: uFST}
}

// IcmsSt is ICMSST: ICMS charged earlier, for the substituted taxpayer.
type IcmsSt struct {
	Orig            string  `json:"orig"`
	CST             string  `json:"cst"`
	VBCSTRet        string  `json:"v_bc_st_ret"`
	PST             *string `json:"p_st,omitempty"`
	VICMSSubstituto *string `json:"v_icms_substituto,omitempty"`
	VICMSSTRet      string  `json:"v_icms_st_ret"`
	VBCFCPSTRet     *string `json:"v_bc_fcp_st_ret,omitempty"`
	PFCPSTRet       *string `json:"p_fcp_st_ret,omitempty"`
	VFCPSTRet       *string `json:"v_fcp_st_ret,omitempty"`
	VBCSTDest       string  `json:"v_bc_st_dest"`
	VICMSSTDest     string  `json:"v_icms_st_dest"`
	PRedBCEfet      *string `json:"p_red_bc_efet,omitempty"`
	VBCEfet         *string `json:"v_bc_efet,omitempty"`
	PICMSEfet       *string `json:"p_icms_efet,omitempty"`
	VICMSEfet       *string `json:"v_icms_efet,omitempty"`
}

func (IcmsSt) icmsTag() string { return "ICMSST" }

// NewIcmsSt fills the fields the schema requires.
func NewIcmsSt(orig, cST, vBCSTRet, vICMSSTRet, vBCSTDest, vICMSSTDest string) IcmsSt {
	return IcmsSt{Orig: orig, CST: cST, VBCSTRet: vBCSTRet, VICMSSTRet: vICMSSTRet, VBCSTDest: vBCSTDest, VICMSSTDest: vICMSSTDest}
}

// IcmsSn101 is ICMSSN101: Simples Nacional ICMS with a credit.
type IcmsSn101 struct {
	Orig        string `json:"orig"`
	CSOSN       string `json:"csosn"`
	PCredSN     string `json:"p_cred_sn"`
	VCredICMSSN string `json:"v_cred_icms_sn"`
}

func (IcmsSn101) icmsTag() string { return "ICMSSN101" }

// NewIcmsSn101 fills the fields the schema requires.
func NewIcmsSn101(orig, pCredSN, vCredICMSSN string) IcmsSn101 {
	return IcmsSn101{CSOSN: "101", Orig: orig, PCredSN: pCredSN, VCredICMSSN: vCredICMSSN}
}

// IcmsSn102 is ICMSSN102: Simples Nacional ICMS without a credit.
type IcmsSn102 struct {
	Orig  *string `json:"orig,omitempty"`
	CSOSN string  `json:"csosn"`
}

func (IcmsSn102) icmsTag() string { return "ICMSSN102" }

// NewIcmsSn102 fills the fields the schema requires.
func NewIcmsSn102(cSOSN string) IcmsSn102 {
	return IcmsSn102{CSOSN: cSOSN}
}

// IcmsSn201 is ICMSSN201: Simples Nacional ICMS with a credit and substitution.
type IcmsSn201 struct {
	Orig        string  `json:"orig"`
	CSOSN       string  `json:"csosn"`
	ModBCST     string  `json:"mod_bc_st"`
	PMVAST      *string `json:"p_mva_st,omitempty"`
	PRedBCST    *string `json:"p_red_bc_st,omitempty"`
	VBCST       string  `json:"v_bc_st"`
	PICMSST     string  `json:"p_icms_st"`
	VICMSST     string  `json:"v_icms_st"`
	VBCFCPST    *string `json:"v_bc_fcp_st,omitempty"`
	PFCPST      *string `json:"p_fcp_st,omitempty"`
	VFCPST      *string `json:"v_fcp_st,omitempty"`
	PCredSN     string  `json:"p_cred_sn"`
	VCredICMSSN string  `json:"v_cred_icms_sn"`
}

func (IcmsSn201) icmsTag() string { return "ICMSSN201" }

// NewIcmsSn201 fills the fields the schema requires.
func NewIcmsSn201(orig, modBCST, vBCST, pICMSST, vICMSST, pCredSN, vCredICMSSN string) IcmsSn201 {
	return IcmsSn201{CSOSN: "201", Orig: orig, ModBCST: modBCST, VBCST: vBCST, PICMSST: pICMSST, VICMSST: vICMSST, PCredSN: pCredSN, VCredICMSSN: vCredICMSSN}
}

// IcmsSn202 is ICMSSN202: Simples Nacional ICMS without a credit, with substitution.
type IcmsSn202 struct {
	Orig     string  `json:"orig"`
	CSOSN    string  `json:"csosn"`
	ModBCST  string  `json:"mod_bc_st"`
	PMVAST   *string `json:"p_mva_st,omitempty"`
	PRedBCST *string `json:"p_red_bc_st,omitempty"`
	VBCST    string  `json:"v_bc_st"`
	PICMSST  string  `json:"p_icms_st"`
	VICMSST  string  `json:"v_icms_st"`
	VBCFCPST *string `json:"v_bc_fcp_st,omitempty"`
	PFCPST   *string `json:"p_fcp_st,omitempty"`
	VFCPST   *string `json:"v_fcp_st,omitempty"`
}

func (IcmsSn202) icmsTag() string { return "ICMSSN202" }

// NewIcmsSn202 fills the fields the schema requires.
func NewIcmsSn202(orig, cSOSN, modBCST, vBCST, pICMSST, vICMSST string) IcmsSn202 {
	return IcmsSn202{Orig: orig, CSOSN: cSOSN, ModBCST: modBCST, VBCST: vBCST, PICMSST: pICMSST, VICMSST: vICMSST}
}

// IcmsSn500 is ICMSSN500: Simples Nacional ICMS already charged by substitution.
type IcmsSn500 struct {
	Orig            string  `json:"orig"`
	CSOSN           string  `json:"csosn"`
	VBCSTRet        *string `json:"v_bc_st_ret,omitempty"`
	PST             *string `json:"p_st,omitempty"`
	VICMSSubstituto *string `json:"v_icms_substituto,omitempty"`
	VICMSSTRet      *string `json:"v_icms_st_ret,omitempty"`
	VBCFCPSTRet     *string `json:"v_bc_fcp_st_ret,omitempty"`
	PFCPSTRet       *string `json:"p_fcp_st_ret,omitempty"`
	VFCPSTRet       *string `json:"v_fcp_st_ret,omitempty"`
	PRedBCEfet      *string `json:"p_red_bc_efet,omitempty"`
	VBCEfet         *string `json:"v_bc_efet,omitempty"`
	PICMSEfet       *string `json:"p_icms_efet,omitempty"`
	VICMSEfet       *string `json:"v_icms_efet,omitempty"`
}

func (IcmsSn500) icmsTag() string { return "ICMSSN500" }

// NewIcmsSn500 fills the fields the schema requires.
func NewIcmsSn500(orig string) IcmsSn500 {
	return IcmsSn500{CSOSN: "500", Orig: orig}
}

// IcmsSn900 is ICMSSN900: Simples Nacional ICMS, other cases.
type IcmsSn900 struct {
	Orig        *string `json:"orig,omitempty"`
	CSOSN       string  `json:"csosn"`
	ModBC       *string `json:"mod_bc,omitempty"`
	VBC         *string `json:"v_bc,omitempty"`
	PRedBC      *string `json:"p_red_bc,omitempty"`
	PICMS       *string `json:"p_icms,omitempty"`
	VICMS       *string `json:"v_icms,omitempty"`
	ModBCST     *string `json:"mod_bc_st,omitempty"`
	PMVAST      *string `json:"p_mva_st,omitempty"`
	PRedBCST    *string `json:"p_red_bc_st,omitempty"`
	VBCST       *string `json:"v_bc_st,omitempty"`
	PICMSST     *string `json:"p_icms_st,omitempty"`
	VICMSST     *string `json:"v_icms_st,omitempty"`
	VBCFCPST    *string `json:"v_bc_fcp_st,omitempty"`
	PFCPST      *string `json:"p_fcp_st,omitempty"`
	VFCPST      *string `json:"v_fcp_st,omitempty"`
	PCredSN     *string `json:"p_cred_sn,omitempty"`
	VCredICMSSN *string `json:"v_cred_icms_sn,omitempty"`
}

func (IcmsSn900) icmsTag() string { return "ICMSSN900" }

// IcmsGroup is any of the ICMS variants.
type IcmsGroup interface {
	icmsTag() string
}

// IcmsUfDest is the interstate ICMS share owed to the destination state.
type IcmsUfDest struct {
	VBCUFDest      string  `json:"v_bc_uf_dest"`
	VBCFCPUFDest   *string `json:"v_bc_fcp_uf_dest,omitempty"`
	PFCPUFDest     *string `json:"p_fcp_uf_dest,omitempty"`
	PICMSUFDest    string  `json:"p_icms_uf_dest"`
	PICMSInter     string  `json:"p_icms_inter"`
	PICMSInterPart string  `json:"p_icms_inter_part"`
	VFCPUFDest     *string `json:"v_fcp_uf_dest,omitempty"`
	VICMSUFDest    string  `json:"v_icms_uf_dest"`
	VICMSUFRemet   string  `json:"v_icms_uf_remet"`
}

// IpiTrib is IPITrib: IPI taxed by rate or by quantity.
type IpiTrib struct {
	CST   string  `json:"cst"`
	VBC   *string `json:"v_bc,omitempty"`
	PIPI  *string `json:"p_ipi,omitempty"`
	QUnid *string `json:"q_unid,omitempty"`
	VUnid *string `json:"v_unid,omitempty"`
	VIPI  string  `json:"v_ipi"`
}

func (IpiTrib) ipiTag() string { return "IPITrib" }

// NewIpiTrib fills the fields the schema requires.
func NewIpiTrib(cST, vIPI string) IpiTrib {
	return IpiTrib{CST: cST, VIPI: vIPI}
}

// IpiNt is IPINT: IPI not taxed.
type IpiNt struct {
	CST string `json:"cst"`
}

func (IpiNt) ipiTag() string { return "IPINT" }

// NewIpiNt fills the fields the schema requires.
func NewIpiNt(cST string) IpiNt {
	return IpiNt{CST: cST}
}

// IpiVariant is any of the IPI variants.
type IpiVariant interface {
	ipiTag() string
}

// Ipi is this item's IPI.
type Ipi struct {
	CNPJProd *string    `json:"cnpj_prod,omitempty"`
	CSelo    *string    `json:"c_selo,omitempty"`
	QSelo    *string    `json:"q_selo,omitempty"`
	CEnq     string     `json:"c_enq"`
	Trib     IpiVariant `json:"-"`
}

func (i Ipi) toDict() map[string]any {
	data := toDict(i)
	if i.Trib != nil {
		data["trib"] = toDict(i.Trib)
	}
	return data
}

// PisAliq is PISAliq: PIS taxed by rate.
type PisAliq struct {
	CST  string `json:"cst"`
	VBC  string `json:"v_bc"`
	PPIS string `json:"p_pis"`
	VPIS string `json:"v_pis"`
}

func (PisAliq) pisTag() string { return "PISAliq" }

// NewPisAliq fills the fields the schema requires.
func NewPisAliq(cST, vBC, pPIS, vPIS string) PisAliq {
	return PisAliq{CST: cST, VBC: vBC, PPIS: pPIS, VPIS: vPIS}
}

// PisQtde is PISQtde: PIS taxed by quantity.
type PisQtde struct {
	CST       string `json:"cst"`
	QBCProd   string `json:"q_bc_prod"`
	VAliqProd string `json:"v_aliq_prod"`
	VPIS      string `json:"v_pis"`
}

func (PisQtde) pisTag() string { return "PISQtde" }

// NewPisQtde fills the fields the schema requires.
func NewPisQtde(qBCProd, vAliqProd, vPIS string) PisQtde {
	return PisQtde{CST: "03", QBCProd: qBCProd, VAliqProd: vAliqProd, VPIS: vPIS}
}

// PisNt is PISNT: PIS not taxed.
type PisNt struct {
	CST string `json:"cst"`
}

func (PisNt) pisTag() string { return "PISNT" }

// NewPisNt fills the fields the schema requires.
func NewPisNt(cST string) PisNt {
	return PisNt{CST: cST}
}

// PisOutr is PISOutr: PIS taxed some other way.
type PisOutr struct {
	CST       string  `json:"cst"`
	VBC       *string `json:"v_bc,omitempty"`
	PPIS      *string `json:"p_pis,omitempty"`
	QBCProd   *string `json:"q_bc_prod,omitempty"`
	VAliqProd *string `json:"v_aliq_prod,omitempty"`
	VPIS      string  `json:"v_pis"`
}

func (PisOutr) pisTag() string { return "PISOutr" }

// NewPisOutr fills the fields the schema requires.
func NewPisOutr(cST, vPIS string) PisOutr {
	return PisOutr{CST: cST, VPIS: vPIS}
}

// PisGroup is any of the PIS variants.
type PisGroup interface {
	pisTag() string
}

// PisSt is PIS withheld by substitution.
type PisSt struct {
	VBC          *string `json:"v_bc,omitempty"`
	PPIS         *string `json:"p_pis,omitempty"`
	QBCProd      *string `json:"q_bc_prod,omitempty"`
	VAliqProd    *string `json:"v_aliq_prod,omitempty"`
	VPIS         string  `json:"v_pis"`
	IndSomaPISST *string `json:"ind_soma_pis_st,omitempty"`
}

// CofinsAliq is COFINSAliq: COFINS taxed by rate.
type CofinsAliq struct {
	CST     string `json:"cst"`
	VBC     string `json:"v_bc"`
	PCofins string `json:"p_cofins"`
	VCofins string `json:"v_cofins"`
}

func (CofinsAliq) cofinsTag() string { return "COFINSAliq" }

// NewCofinsAliq fills the fields the schema requires.
func NewCofinsAliq(cST, vBC, pCOFINS, vCOFINS string) CofinsAliq {
	return CofinsAliq{CST: cST, VBC: vBC, PCofins: pCOFINS, VCofins: vCOFINS}
}

// CofinsQtde is COFINSQtde: COFINS taxed by quantity.
type CofinsQtde struct {
	CST       string `json:"cst"`
	QBCProd   string `json:"q_bc_prod"`
	VAliqProd string `json:"v_aliq_prod"`
	VCofins   string `json:"v_cofins"`
}

func (CofinsQtde) cofinsTag() string { return "COFINSQtde" }

// NewCofinsQtde fills the fields the schema requires.
func NewCofinsQtde(qBCProd, vAliqProd, vCOFINS string) CofinsQtde {
	return CofinsQtde{CST: "03", QBCProd: qBCProd, VAliqProd: vAliqProd, VCofins: vCOFINS}
}

// CofinsNt is COFINSNT: COFINS not taxed.
type CofinsNt struct {
	CST string `json:"cst"`
}

func (CofinsNt) cofinsTag() string { return "COFINSNT" }

// NewCofinsNt fills the fields the schema requires.
func NewCofinsNt(cST string) CofinsNt {
	return CofinsNt{CST: cST}
}

// CofinsOutr is COFINSOutr: COFINS taxed some other way.
type CofinsOutr struct {
	CST       string  `json:"cst"`
	VBC       *string `json:"v_bc,omitempty"`
	PCofins   *string `json:"p_cofins,omitempty"`
	QBCProd   *string `json:"q_bc_prod,omitempty"`
	VAliqProd *string `json:"v_aliq_prod,omitempty"`
	VCofins   string  `json:"v_cofins"`
}

func (CofinsOutr) cofinsTag() string { return "COFINSOutr" }

// NewCofinsOutr fills the fields the schema requires.
func NewCofinsOutr(cST, vCOFINS string) CofinsOutr {
	return CofinsOutr{CST: cST, VCofins: vCOFINS}
}

// CofinsGroup is any of the COFINS variants.
type CofinsGroup interface {
	cofinsTag() string
}

// CofinsSt is COFINS withheld by substitution.
type CofinsSt struct {
	VBC             *string `json:"v_bc,omitempty"`
	PCofins         *string `json:"p_cofins,omitempty"`
	QBCProd         *string `json:"q_bc_prod,omitempty"`
	VAliqProd       *string `json:"v_aliq_prod,omitempty"`
	VCofins         string  `json:"v_cofins"`
	IndSomaCOFINSST *string `json:"ind_soma_cofins_st,omitempty"`
}

// Tax is this item's taxes, already computed by the caller.
type Tax struct {
	VTotTrib   *string
	Icms       IcmsGroup
	IcmsUfDest *IcmsUfDest
	Ipi        *Ipi
	Pis        PisGroup
	PisSt      *PisSt
	Cofins     CofinsGroup
	CofinsSt   *CofinsSt
}

// ToDict returns the taxes as a plain map.
func (t Tax) ToDict() map[string]any {
	data := map[string]any{}
	if t.VTotTrib != nil {
		data["v_tot_trib"] = *t.VTotTrib
	}
	if t.Icms != nil {
		data["icms"] = toDict(t.Icms)
	}
	if t.IcmsUfDest != nil {
		data["icms_uf_dest"] = toDict(t.IcmsUfDest)
	}
	if t.Ipi != nil {
		data["ipi"] = t.Ipi.toDict()
	}
	if t.Pis != nil {
		data["pis"] = toDict(t.Pis)
	}
	if t.PisSt != nil {
		data["pis_st"] = toDict(t.PisSt)
	}
	if t.Cofins != nil {
		data["cofins"] = toDict(t.Cofins)
	}
	if t.CofinsSt != nil {
		data["cofins_st"] = toDict(t.CofinsSt)
	}
	return data
}
