// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package api defines the HTTP response envelope and the general OpenAPI
// metadata used by swaggo/swag to generate the GoDaily API contract.
//
//	@title			GoDaily API
//	@version		1.0
//	@description	The GoDaily news digest API — ranked, summarised Go news delivered daily.
//	@contact.name	ainsley.dev
//	@contact.url	https://ainsley.dev
//	@license.name	BSD-style
//	@BasePath		/api
//
//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				Bearer token supplied in the Authorization header.
package api
