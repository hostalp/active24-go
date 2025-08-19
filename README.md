# Active24.cz client in Go

This is client library to interact with [Active24 APIv2](https://www.active24.cz/centrum-napovedy/api-rozhrani).
Currently, only subset of API is implemented, but contributions are always welcome.

## Usage

```go
package main

import "github.com/hostalp/active24-go/active24"

func main() {
	client := active24.New("my-secret-api-key", "my-secret-api-secret")

	recordType := string(active24.DnsRecordTypeA)
	hostName := "host1" // Short hostname (excl. the domain)
	ipAddress := "1.2.3.4"
	ttl := 600

	_, err := client.Dns().With(12345678).Create(&active24.DnsRecord{
		Type:    &recordType,
		Name:    hostName;
		Content: &ipAddress,
		Ttl:     ttl,
	})
	if err != nil {
		panic(err)
	}
}
```
