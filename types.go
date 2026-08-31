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
