package cloudns

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func (p *Provider) setAuthQuery(query url.Values) error {
	if p.AuthId == "" {
		return errors.New("missing auth id")
	}
	if p.AuthPassword == "" {
		return errors.New("missing auth password")
	}
	query.Set("auth-password", p.AuthPassword)
	if strings.ToLower(p.Sub) == "true" {
		query.Set("sub-auth-id", p.AuthId)
	} else {
		query.Set("auth-id", p.AuthId)
	}
	return nil
}
func (p *Provider) getResponse(ctx context.Context, api string, query url.Values) (*http.Response, error) {
	fmt.Println(fmt.Sprintf("%s%s?%s", DnsApi, api, query.Encode()))
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s%s?%s", DnsApi, api, query.Encode()),
		nil,
	)
	if err != nil {
		return nil, err
	}
	httpClient := http.Client{}
	return httpClient.Do(req)

}
func Unmarshal(rc io.ReadCloser, v interface{}) error {
	data, err := ioutil.ReadAll(rc)
	_ = rc.Close()
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, v)
	if err != nil {
		status, err2 := checkStatus(data)
		if err2 != nil {
			return err2
		}
		if status.IsError() {
			return status
		}

	}
	return err
}

type Status struct {
	Status      string          `json:"status"`
	Description string          `json:"statusDescription"`
	Data        json.RawMessage `json:"data,omitempty"`
}
type Id struct {
	Id string `json:"id,int"`
}

func (s Status) Error() string {
	return s.Description
}
func (s Status) IsError() bool {
	return s.Status != "Success"
}
func checkStatus(data []byte) (*Status, error) {
	var s Status
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	if s.Status == "" {
		return nil, errors.New("not status data")
	}
	return &s, nil
}

type Record struct {
	Id       string `json:"id"`
	Type     string `json:"type"`
	Host     string `json:"host"`
	Record   string `json:"record"`
	Failover string `json:"failover"`
	Ttl      int    `json:"ttl,string"`
	Status   int    `json:"status"`
}

func selectTTl(t time.Duration) int {
	s := int(t.Seconds())
	for _, v := range AvailableTTL {
		if s >= v {
			return v
		}
	}
	return 60
}
