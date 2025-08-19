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

package main

import (
	"fmt"

	"github.com/hostalp/active24-go/active24"
)

func main() {
	client := active24.New("ak48l3h7-ak5d-qn4t-p8gc-b6fs8c3l", "ajvkeo3y82ndsu2smvxy3o36496dcascksldncsq", active24.ApiEndpoint("https://rest.active24.cz"))

	dns := client.Dns()

	//list DNS records in domain with ServiceID 12345678
	recs, err := dns.With(12345678).ListAll()
	if err != nil {
		panic(err)
	}
	for _, rec := range recs {
		fmt.Printf("rec[type:%s, name:%s, ttl:%d]\n", *rec.Type, rec.Name, rec.Ttl)
	}

	//create CNAME record
	recordType := string(active24.DnsRecordTypeCNAME)
	hostName := "host1"
	alias := "host.example.com"
	ttl := 600

	err = dns.With(12345678).Create(&active24.DnsRecord{
		Type:    &recordType,
		Name:    hostName,
		Content: &alias,
		Ttl:     ttl,
	})
	if err != nil {
		panic(err)
	}

}
