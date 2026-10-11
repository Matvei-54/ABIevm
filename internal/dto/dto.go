package dto

type MerklOpportunity struct {
	ChainID  int     `json:"chainId"`
	Protocol string  `json:"protocol"`
	APR      float64 `json:"apr"`
	Tokens   struct {
		Deposit string `json:"deposit"`
	} `json:"tokens"`
}

type MerklResponse struct {
	Opportunities []MerklOpportunity `json:"opportunities"`
}
