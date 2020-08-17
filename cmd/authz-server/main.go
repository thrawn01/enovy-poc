package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"google.golang.org/grpc/reflection"
	"log"
	"net"
	"regexp"
	"strings"

	"github.com/davecgh/go-spew/spew"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	auth "github.com/envoyproxy/go-control-plane/envoy/service/auth/v2"
	v2 "github.com/envoyproxy/go-control-plane/envoy/type"
	"github.com/gogo/googleapis/google/rpc"
	"github.com/pkg/errors"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
)

type EndpointSpec struct {
	ServiceName string `json:"ServiceName"`
	AuthType    string `json:"AuthType"`
	Regex       string `json:"Regex"`
	regex       *regexp.Regexp
}

const (
	authTypeAccount = "account"
	authTypeDomain  = "domain"

	accountPrivateURL = "%v/v2/flagman/accounts/private"
	accountPublicURL  = "%v/v2/flagman/accounts/public"
	domainPrivateURL  = "%v/v2/flagman/domains/$1/private"
)

type AuthorizationServer struct {
	Specs []*EndpointSpec
}

func (a *AuthorizationServer) Check(ctx context.Context, req *auth.CheckRequest) (*auth.CheckResponse, error) {
	res, err := a.check(ctx, req)
	fmt.Printf("============================\n")
	spew.Dump(res)
	return res, err
}

func (a *AuthorizationServer) check(ctx context.Context, req *auth.CheckRequest) (*auth.CheckResponse, error) {
	header, ok := req.Attributes.Request.Http.Headers["authorization"]
	if !ok {
		return failAuth("missing 'Authorization' header")
	}

	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || parts[0] != "Basic" {
		return failAuth("malformed 'Authorization' header; missing 'Basic' prefix")
	}

	payload, _ := base64.StdEncoding.DecodeString(parts[1])
	pair := strings.SplitN(string(payload), ":", 2)

	if len(pair) != 2 {
		return failAuth("malformed 'Authorization' header; missing key/value pair after decode")
	}

	// This here only for our POC, either the spec should decide what sort
	// of auth should be done or we default to account level auth with flagman
	if pair[0] != "thrawn" && pair[1] != "password" {
		return failAuth("invalid username/password")
	}

	var headers []*core.HeaderValueOption
	// Match the spec against the request path
	if spec := a.matchSpec(req.Attributes.Request.Http.Path); spec != nil {
		// TODO: Preform auth using the spec and return valid headers for this spec
		headers = append(headers, &core.HeaderValueOption{
			Header: &core.HeaderValue{
				Key:   "x-mailgun-domain-id",
				Value: "domain-id-01",
			},
		}, &core.HeaderValueOption{
			Header: &core.HeaderValue{
				Key:   "x-mailgun-account-id",
				Value: "account-id-01",
			},
		}, &core.HeaderValueOption{
			Header: &core.HeaderValue{
				Key:   "x-spec-auth-type",
				Value: spec.AuthType,
			},
		})
	} else {
		// If no spec found, then preform account level authentication and add the account info header
		headers = append(headers, &core.HeaderValueOption{
			Header: &core.HeaderValue{
				Key:   "x-mailgun-account-id",
				Value: "account-id-01",
			},
		})
	}

	return &auth.CheckResponse{
		Status: &status.Status{
			Code: int32(rpc.OK),
		},
		HttpResponse: &auth.CheckResponse_OkResponse{
			OkResponse: &auth.OkHttpResponse{
				Headers: headers,
			},
		},
	}, nil
}

func (a *AuthorizationServer) matchSpec(path string) *EndpointSpec {
	for _, spec := range a.Specs {
		groups := spec.regex.FindStringSubmatch(path)
		if len(groups) == 0 {
			continue
		}
		return spec
	}
	return nil
}

func compileRegex(spec []*EndpointSpec) error {
	var err error
	for i, _ := range spec {
		spec[i].regex, err = regexp.Compile(spec[i].Regex)
		if err != nil {
			return errors.Wrapf(err, "while compiling regex '%s'", spec[i].Regex)
		}
	}
	return nil
}

func failAuth(msg string) (*auth.CheckResponse, error) {
	return &auth.CheckResponse{
		Status: &status.Status{
			Code: int32(rpc.UNAUTHENTICATED),
		},
		HttpResponse: &auth.CheckResponse_DeniedResponse{
			DeniedResponse: &auth.DeniedHttpResponse{
				Status: &v2.HttpStatus{
					Code: v2.StatusCode_Accepted,
				},
				Body: msg,
			},
		},
	}, nil
}

func main() {
	// TODO: Get the auth specs from vulcand config or consul/etcd

	lis, err := net.Listen("tcp", ":4000")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	log.Printf("listening on %s", lis.Addr())

	grpcServer := grpc.NewServer()
	authServer := &AuthorizationServer{}

	auth.RegisterAuthorizationServer(grpcServer, authServer)

	specs := []*EndpointSpec{
		{
			ServiceName: "api-server",
			AuthType:    authTypeDomain,
			Regex:       "/v[23]/domains/([^/]+)",
		},
	}
	authServer.Specs = specs

	if err := compileRegex(specs); err != nil {
		log.Fatalf("Failed to compile spec regexes: %s", err)
	}

	reflection.Register(grpcServer)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
