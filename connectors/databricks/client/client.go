package client

import (
	"fmt"

	"github.com/go-resty/resty/v2"
)

type DatabricksClient struct {
	cl *resty.Client
}

func New(url, token string) *DatabricksClient {
	return &DatabricksClient{
		cl: resty.New().SetBaseURL(url).SetHeader("Authorization", fmt.Sprintf("Bearer %s", token)),
	}
}
