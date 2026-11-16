package stackin

type DocumentType string

const (
	NFE  DocumentType = "nfe"
	NFSE DocumentType = "nfse"
)

type Environment string

const (
	Local      Environment = "local"
	Test       Environment = "test"
	Production Environment = "production"
)

// Manifestation is the recipient's answer to a document issued against it.
type Manifestation string

const (
	Confirmacao          Manifestation = "210200"
	Ciencia              Manifestation = "210210"
	Desconhecimento      Manifestation = "210220"
	OperacaoNaoRealizada Manifestation = "210240"
)

// Purpose is why an NF-e is issued (finNFe). The API defaults to
// PurposeNormal when it is left out.
type Purpose string

const (
	PurposeNormal        Purpose = "normal"
	PurposeComplementary Purpose = "complementary"
	PurposeAdjustment    Purpose = "adjustment"
	PurposeReturn        Purpose = "return"
)

// Direction is whether an NF-e is outgoing or an incoming one the company
// issues itself (tpNF). The API defaults to DirectionOut when it is left out.
type Direction string

const (
	DirectionOut Direction = "out"
	DirectionIn  Direction = "in"
)
