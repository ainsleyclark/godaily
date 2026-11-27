package http

import (
	"database/sql/driver"
	nethttp "net/http"

	"github.com/tursodatabase/libsql-client-go/libsql/internal/http/hranaV2"
)

func Connect(url, jwt, host string, schemaDb bool, remoteEncryptionKey string, requestHeaders map[string]string, httpClient *nethttp.Client) driver.Conn {
	return hranaV2.Connect(url, jwt, host, schemaDb, remoteEncryptionKey, requestHeaders, httpClient)
}
