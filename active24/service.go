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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// https://rest.active24.cz/docs/index
// https://rest.active24.cz/docs/v1.service

// Service provides a way to interact with services
type Service interface {
	// List lists all services.
	ListAll() ([]ServiceItems, ApiError)
	// List lists service of specified name.
	//List(string) ([]ServiceItems, ApiError)
}

type ServiceItems struct {
	ID          *int     `json:"id,omitempty"`
	ServiceName *string  `json:"serviceName,omitempty"`
	Status      *string  `json:"status,omitempty"`
	Name        string   `json:"name"`
	CreateTime  *int64   `json:"createTime,omitempty"`
	ExpireTime  *int64   `json:"expireTime,omitempty"`
	Price       *float32 `json:"price,omitempty"`
	AutoExtend  *bool    `json:"autoExtend,omitempty"`
}

type ServicePager struct {
	Page     *int `json:"page,omitempty"`
	PageSize *int `json:"pagesize,omitempty"`
	Items    *int `json:"items,omitempty"`
}

type ServiceCollection struct {
	Items []ServiceItems `json:"items,omitempty"`
	Pager *ServicePager  `json:"pager,omitempty"`
}

type service struct {
	h helper
}

/*func (s *service) ListAll() ([]ServiceItems, ApiError) {
	return s.List("")
}*/

// func (s *service) List(svcName string) ([]ServiceItems, ApiError) {
func (s *service) ListAll() ([]ServiceItems, ApiError) {
	var ret ServiceCollection

	resp, err := s.h.do(http.MethodGet, "v1/user/self/service", nil)
	if err != nil {
		return nil, apiErr(nil, err)
	}
	defer func(b io.ReadCloser) {
		_ = b.Close()
	}(resp.Body)
	if resp.StatusCode > 399 && resp.StatusCode < 600 {
		return nil, apiErr(resp, fmt.Errorf("invalid response from API: %d", resp.StatusCode))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, apiErr(resp, err)
	}
	//ret := make([]Service, 0)
	err = json.Unmarshal(body, &ret)
	if err != nil {
		return nil, apiErr(resp, err)
	}

	/*if svcName != "" {
		for i := range ret.Items {
			if s.h.l.V(9).Enabled() {
				s.h.l.V(9).Info("item=", ret.Items[i], "name=", ret.Items[i].Name)
			}
			if ret.Items[i].Name == svcName {
				s.h.l.V(4).Info("Found service name:", svcName)
				return []ServiceItems{ret.Items[i]}, nil
			}
		}
	}*/

	return ret.Items, nil
}
