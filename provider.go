// Package libdnstemplate implements a DNS record management client compatible
// with the libdns interfaces for <PROVIDER NAME>. TODO: This package is a
// template only. Customize all godocs for actual implementation.
package cloudns

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"sync"

	"github.com/libdns/libdns"
)

// TODO: Providers must not require additional provisioning steps by the callers; it
// should work simply by populating a struct and calling methods on it. If your DNS
// service requires long-lived state or some extra provisioning step, do it implicitly
// when methods are called; sync.Once can help with this, and/or you can use a
// sync.(RW)Mutex in your Provider struct to synchronize implicit provisioning.

// Provider facilitates DNS record manipulation with <TODO: PROVIDER NAME>.
type Provider struct {
	// TODO: put config fields here (with snake_case json
	// struct tags on exported fields), for example:
	AuthId       string `json:"auth_id"`
	Sub          string `json:"sub,omitempty"`
	AuthPassword string `json:"auth_password"`
	lock         sync.Mutex
}

// GetRecords lists all the records in the zone.
func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	p.lock.Lock()
	defer p.lock.Unlock()
	param := url.Values{}
	param.Set("domain-name", zone)
	err := p.setAuthQuery(param)
	if err != nil {
		return nil, err
	}
	var response *http.Response
	response, err = p.getResponse(ctx, "dns/records.json", param)
	if err != nil {
		return nil, err
	}
	m := make(map[string]Record)
	if err = Unmarshal(response.Body, &m); err != nil {
		return nil, err
	}
	var records []libdns.Record
	for _, v := range m {
		records = append(records, libdns.Record{
			ID:    v.Id,
			Type:  v.Type,
			Name:  v.Host,
			Value: v.Record,
			TTL:   v.Ttl,
		})
	}
	return records, nil
}

// AppendRecords adds records to the zone. It returns the records that were added.
func (p *Provider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	p.lock.Lock()
	defer p.lock.Unlock()
	param := url.Values{}
	param.Set("domain-name", zone)
	err := p.setAuthQuery(param)
	if err != nil {
		return nil, err
	}
	var rls []libdns.Record
	for _, r := range records {
		param.Set("record-type", r.Type)
		param.Set("host", r.Name)
		param.Set("record", r.Value)
		param.Set("ttl", strconv.FormatInt(int64(r.TTL), 10))
		res, err := p.getResponse(ctx, "dns/add-record.json", param)
		if err != nil {
			return nil, err
		}
		data, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}
		status,err := checkStatus(data)
		if err!=nil {
			return nil, err
		}
		if status.IsError() {
			return nil,status
		}
		var id Id
		if err = json.Unmarshal(status.Data, &id); err != nil {
			return nil, err
		}
		r.ID = id.Id
		rls = append(rls, r)
	}
	return rls, nil
}

// SetRecords sets the records in the zone, either by updating existing records or creating new ones.
// It returns the updated records.
func (p *Provider) SetRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	p.lock.Lock()
	defer p.lock.Unlock()
	param := url.Values{}
	param.Set("domain-name", zone)
	err := p.setAuthQuery(param)
	if err != nil {
		return nil, err
	}
	for _, r := range records {
		param.Set("record-id", r.ID)
		param.Set("host", r.Name)
		param.Set("record", r.Value)
		param.Set("ttl", strconv.FormatInt(int64(r.TTL), 10))
		res, err := p.getResponse(ctx, "dns/mod-record.json", param)
		if err != nil {
			return nil, err
		}
		data, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}
		status,err := checkStatus(data)
		if err!=nil {
			return nil, err
		}
		if status.IsError() {
			return nil,status
		}
	}
	return records, nil
}

// DeleteRecords deletes the records from the zone. It returns the records that were deleted.
func (p *Provider) DeleteRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	p.lock.Lock()
	defer p.lock.Unlock()
	param := url.Values{}
	param.Set("domain-name", zone)
	err := p.setAuthQuery(param)
	if err != nil {
		return nil, err
	}
	for _, r := range records {
		param.Set("record-id", r.ID)
		res, err := p.getResponse(ctx, "dns/delete-record.json", param)
		if err != nil {
			return nil, err
		}
		data, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}
		status,err := checkStatus(data)
		if err!=nil {
			return nil, err
		}
		if status.IsError() {
			return nil,status
		}
	}
	return records, nil
}

// Interface guards
var (
	_ libdns.RecordGetter   = (*Provider)(nil)
	_ libdns.RecordAppender = (*Provider)(nil)
	_ libdns.RecordSetter   = (*Provider)(nil)
	_ libdns.RecordDeleter  = (*Provider)(nil)
)
