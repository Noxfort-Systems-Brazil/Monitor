// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.
//
// File: internal/transport/http/redirect.go
// Author: Gabriel Moraes
// Date: 2026-09-04

package http

import (
	"fmt"
	"html"
	"net/http"
)

// Redirect replies to the request with a redirect to url.
// It sets the HTTP Location header and status code, and outputs an HTML document containing
// both a <meta http-equiv="refresh"> tag and a JavaScript window.location.replace() call.
// This guarantees automatic, seamless navigation even in embedded WebViews (such as Wails / WebKitGTK
// custom URI scheme handlers) that do not automatically follow 3xx redirect status codes.
func Redirect(w http.ResponseWriter, r *http.Request, targetURL string, code int) {
	w.Header().Set("Location", targetURL)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(code)

	escapedURL := html.EscapeString(targetURL)
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<meta http-equiv="refresh" content="0; url=%s">
<script>window.location.replace(%q);</script>
</head>
<body>
<p>Redirecionando... <a href="%s">Clique aqui caso não seja redirecionado</a>.</p>
</body>
</html>`, escapedURL, targetURL, escapedURL)
}
