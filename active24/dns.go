/*
Copyright 2023 Richard Kosegi

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package active24

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type DnsRecordType string

//goland:noinspection GoUnusedConst
const (
	DnsRecordTypeA     = DnsRecordType("A")
	DnsRecordTypeAAAA  = DnsRecordType("AAAA")
	DnsRecordTypeANAME = DnsRecordType("ANAME")
	DnsRecordTypeCAA   = DnsRecordType("CAA")
	DnsRecordTypeCNAME = DnsRecordType("CNAME")
	DnsRecordTypeMX    = DnsRecordType("MX")
	DnsRecordTypeSRV   = DnsRecordType("SRV")
	DnsRecordTypeTXT   = DnsRecordType("TXT")
	DnsRecordTypeCERT  = DnsRecordType("CERT")
	DnsRecordTypeLOC   = DnsRecordType("LOC")
	DnsRecordTypeSSHFP = DnsRecordType("SSHFP")
	DnsRecordTypeTLSA  = DnsRecordType("TLSA")
	DnsRecordTypeDS    = DnsRecordType("DS")
	DnsRecordTypeNS    = DnsRecordType("NS")
)

// Dns provides a way to interact with DNS domains
type Dns interface {
	// With returns interface to interact with DNS records in given service ID (domain)
	With(serviceID int) DnsRecordActions
}

type DnsRecord struct {
	Type     *string `json:"type,omitempty"`
	ID       *int    `json:"id,omitempty"`
	Name     string  `json:"name"`
	Content  *string `json:"content,omitempty"`
	Ttl      int     `json:"ttl"`
	Priority *int    `json:"priority,omitempty"`
	Port     *int    `json:"port,omitempty"`
	Weight   *int    `json:"weight,omitempty"`
}

type DnsRecordPaginatedCollection struct {
	CurrentPage  *int        `json:"currentPage,omitempty"`
	TotalPages   *int        `json:"totalPages,omitempty"`
	TotalRecords *int        `json:"totalRecords,omitempty"`
	RowsPerPage  *int        `json:"rowsPerPage,omitempty"`
	NextPageUrl  *string     `json:"nextPageUrl,omitempty"`
	Data         []DnsRecord `json:"data,omitempty"`
}

// DnsRecordActions allows interaction with DNS records
type DnsRecordActions interface {
	// Create creates a new DNS record
	Create(*DnsRecord) ApiError
	// List lists all DNS records in this domain.
	ListAll() ([]DnsRecord, ApiError)
	// List lists DNS records of specified type or name in this domain.
	List(DnsRecordType, string) ([]DnsRecord, ApiError)
	// ListPage lists 1 page of DNS records of specified type or name in this domain.
	ListPage(DnsRecordType, string, string, int) ([]DnsRecord, string, int, ApiError)
	// Update updates an existing DNS record
	Update(int, *DnsRecord) ApiError
	// Delete removes single DNS record based on its ID
	Delete(int) ApiError
}

type dns struct {
	h helper
}

func (d *dns) With(serviceID int) DnsRecordActions {
	return &domainAction{
		h:     d.h,
		svcID: serviceID,
	}
}

type domainAction struct {
	h     helper
	svcID int
}

func (d *domainAction) ListAll() ([]DnsRecord, ApiError) {
	return d.List("", "")
}

func (d *domainAction) List(recType DnsRecordType, recName string) ([]DnsRecord, ApiError) {
	var allRecords []DnsRecord
	var nextPageUrl string
	var nextPage int
	var err ApiError

	pageCount := 1
	for (pageCount == 1 || nextPageUrl != "" || nextPage > 0) && pageCount <= d.h.maxPages {
		var pageRecords []DnsRecord
		pageRecords, nextPageUrl, nextPage, err = d.ListPage(recType, recName, nextPageUrl, nextPage)
		if err != nil {
			return nil, err
		}
		allRecords = append(allRecords, pageRecords...)
		pageCount++
	}
	if pageCount > d.h.maxPages && (nextPageUrl != "" || nextPage > d.h.maxPages) {
		return allRecords, apiErr(nil, fmt.Errorf("maximum page limit reached in List, partial result returned, maxPages: %d, increase the limit in the configuration", d.h.maxPages))
	}
	return allRecords, nil
}

func (d *domainAction) ListPage(recType DnsRecordType, recName string, recPageUrl string, recPage int) ([]DnsRecord, string, int, ApiError) {
	ret, err := d.ListPaginated(recType, recName, recPageUrl, recPage)
	if err != nil {
		return nil, "", 0, err
	}

	var nextPageUrl string
	var nextPage int

	switch {
	// Get the next page request paramater (query string) from the field nextPageUrl of the last response - preferred
	case ret.NextPageUrl != nil && *ret.NextPageUrl != "":
		// Typical response field nextPageUrl: "/?page=2", strip all leading "/?" characters from it
		nextPageUrl = strings.TrimLeft(*ret.NextPageUrl, "/?")
	// Calculate the next page from the currentPage and totalPages fields of the last response - only if not set via nextPageUrl
	case ret.CurrentPage != nil && ret.TotalPages != nil && *ret.CurrentPage < *ret.TotalPages:
		nextPage = *ret.CurrentPage + 1
	}
	return ret.Data, nextPageUrl, nextPage, err
}

func (d *domainAction) ListPaginated(recType DnsRecordType, recName string, recPageUrl string, recPage int) (DnsRecordPaginatedCollection, ApiError) {
	// HTTP request params
	reqParams := url.Values{}
	reqParams.Add("descending", "false")
	reqParams.Add("sortBy", "name")
	if recType != "" {
		reqParams.Add("filters[type]", fmt.Sprintf("[\"%s\"]", recType))
	}
	if recName != "" {
		reqParams.Add("filters[name]", recName)
	}
	var ret DnsRecordPaginatedCollection
	// Requested page obtained from the nextPageUrl field of the last response - preferred
	hasRecPageUrl := false
	if recPageUrl != "" {
		recPageUrlValues, err := url.ParseQuery(recPageUrl)
		if err != nil {
			return ret, apiErr(nil, err)
		}
		for k, v := range recPageUrlValues {
			if k != "" && len(v) > 0 && v[0] != "" {
				reqParams.Add(k, v[0])
				hasRecPageUrl = true
			}
			break
		}
	}
	// Requested page calculated from the currentPage and totalPages fields of the last response - only if not set via nextPageUrl
	if !hasRecPageUrl && recPage > 1 {
		reqParams.Add("page", strconv.Itoa(recPage))
	}

	resp, err := d.h.doWithParams(http.MethodGet, fmt.Sprintf("v2/service/%d/dns/record", d.svcID), reqParams, nil)
	if err != nil {
		return ret, apiErr(nil, err)
	}
	defer func(b io.ReadCloser) {
		_ = b.Close()
	}(resp.Body)
	if resp.StatusCode > 399 && resp.StatusCode < 600 {
		return ret, apiErr(resp, fmt.Errorf("invalid response from API: %d", resp.StatusCode))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ret, apiErr(resp, err)
	}
	//ret := make([]DnsRecord, 0)
	err = json.Unmarshal(body, &ret)
	if err != nil {
		return ret, apiErr(resp, err)
	}
	return ret, apiErr(resp, nil)
}

func (d *domainAction) Create(r *DnsRecord) ApiError {
	data, err := json.Marshal(r)
	if err != nil {
		return apiErr(nil, err)
	}
	return apiErr(d.h.do(http.MethodPost, fmt.Sprintf("v2/service/%d/dns/record", d.svcID), bytes.NewBuffer(data)))
}

func (d *domainAction) change(method string, recordID int, r *DnsRecord) (*http.Response, error) {
	data, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	return d.h.do(method, fmt.Sprintf("v2/service/%d/dns/record/%d", d.svcID, recordID), bytes.NewBuffer(data))
}

func (d *domainAction) Update(ID int, r *DnsRecord) ApiError {
	return apiErr(d.change(http.MethodPut, ID, r))
}

func (d *domainAction) Delete(ID int) ApiError {
	return apiErr(d.h.do(http.MethodDelete, fmt.Sprintf("v2/service/%d/dns/record/%d", d.svcID, ID), nil))
}
