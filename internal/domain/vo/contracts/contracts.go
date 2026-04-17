package contracts

type ExtractContractsInput struct {
	TargetFile string
}

type ExtractContractsOutput struct {
	Contracts []ContractFragment
}

type ContractFragment struct {
	Name string
	Type string
	Path string
	Body string
}
