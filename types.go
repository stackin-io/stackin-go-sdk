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
