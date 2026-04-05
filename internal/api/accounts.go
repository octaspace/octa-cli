package api

import (
	"fmt"
	"math/big"
)

// BigInt is a *big.Int that marshals/unmarshals as a JSON number.
type BigInt struct {
	*big.Int
}

func (b *BigInt) UnmarshalJSON(data []byte) error {
	n, ok := new(big.Int).SetString(string(data), 10)
	if !ok {
		return fmt.Errorf("invalid big integer: %s", data)
	}
	b.Int = n
	return nil
}

func (b BigInt) MarshalJSON() ([]byte, error) {
	if b.Int == nil {
		return []byte("0"), nil
	}
	return []byte(b.Int.String()), nil
}

// Account holds account information.
type Account struct {
	AccountUUID string `json:"account_uuid"`
	Avatar      string `json:"avatar"`
	Balance     BigInt `json:"balance"`
	Email       string `json:"email"`
	RefCode     string `json:"ref_code"`
	UID         int64  `json:"uid"`
	Wallet      string `json:"wallet"`
}

// GetAccount fetches account information from GET /accounts.
func (c *Client) GetAccount() (*Account, error) {
	var account Account
	if err := c.get("/accounts", &account); err != nil {
		return nil, err
	}
	return &account, nil
}

// GetBalance fetches the current account balance in Wei from GET /accounts/balance.
func (c *Client) GetBalance() (*big.Int, error) {
	var resp struct {
		Balance BigInt `json:"balance"`
	}
	if err := c.get("/accounts/balance", &resp); err != nil {
		return nil, err
	}
	return resp.Balance.Int, nil
}

// ValidateToken checks that the API key is valid by calling GET /accounts.
func (c *Client) ValidateToken() error {
	var account Account
	return c.get("/accounts", &account)
}
