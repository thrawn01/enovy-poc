## Envoy POC

# Envoy as an API Gateway
Consists of a envoy proxy with 1 cluster for the api-server and another for the authz-server

# Usage
```bash
$ docker-compose up

# Domain auth extracts the domain from the URL and injects the domain-id as a header in the upstream request
$ curl --user thrawn:password http://localhost:8001/v3/domains/thrawn01.org/info
{
 "Message": "Domain Handler",
 "Domain": "thrawn01.org",
 "Headers": {
  "Accept": [
   "*/*"
  ],
  "Authorization": [
   "Basic dGhyYXduOnBhc3N3b3Jk"
  ],
  "Content-Length": [
   "0"
  ],
  "User-Agent": [
   "curl/7.64.1"
  ],
  "X-Envoy-Expected-Rq-Timeout-Ms": [
   "15000"
  ],
  "X-Forwarded-Proto": [
   "http"
  ],
  "X-Mailgun-Account-Id": [
   "account-id-01"
  ],
  "X-Mailgun-Domain-Id": [
   "domain-id-01"
  ],
  "X-Request-Id": [
   "b0a89015-46bc-44de-a377-1801dc214dc1"
  ],
  "X-Spec-Auth-Type": [
   "domain"
  ]
 }
}

# Account auth only injects the account id in the upstream request
$ curl --user thrawn:password -v  http://localhost:8001/stats
{
 "Message": "Stats here",
 "Domain": "",
 "Headers": {
  "Accept": [
   "*/*"
  ],
  "Authorization": [
   "Basic dGhyYXduOnBhc3N3b3Jk"
  ],
  "Content-Length": [
   "0"
  ],
  "User-Agent": [
   "curl/7.64.1"
  ],
  "X-Envoy-Expected-Rq-Timeout-Ms": [
   "15000"
  ],
  "X-Forwarded-Proto": [
   "http"
  ],
  "X-Mailgun-Account-Id": [
   "account-id-01"
  ],
  "X-Request-Id": [
   "ffbebbf2-c38d-4dc8-b9e2-461e913c12ba"
  ]
 }
}
```
