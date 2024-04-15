package filters

import (
	"log"
	"regexp"
	"strings"

	restful "github.com/emicklei/go-restful/v3"
)

type BypassAuthFilter struct {
	// checkoutTokenRegex   *regexp.Regexp
	purchaseRequestRegex *regexp.Regexp
	accountTokenRegex    *regexp.Regexp
	logger               *log.Logger
}

func NewBypassAuthFilter() *BypassAuthFilter {
	logger := log.New(log.Writer(), "BypassAuthFilter: ", log.LstdFlags|log.Lshortfile)

	purchaseRequestRegex := regexp.MustCompile(`^/accounts/([a-f0-9-]+)/purchase_requests/?$`)
	accountTokenRegex := regexp.MustCompile(`^/accounts/(([a-f0-9-]+)|platform)/auth/tokens/?$`)

	return &BypassAuthFilter{
		// checkoutTokenRegex,
		purchaseRequestRegex,
		accountTokenRegex,
		logger,
	}
}

func (f BypassAuthFilter) checkBypass(req *restful.Request, resp *restful.Response) bool {
	trimmedPath := strings.TrimSuffix(req.Request.URL.Path, "/")

	if trimmedPath == "/health" && req.Request.Method == "GET" {
		return true
	}

	if trimmedPath == "/auth/tokens" && req.Request.Method == "POST" {
		return true
	}

	if trimmedPath == "/platform/auth/tokens" && req.Request.Method == "POST" {
		return true
	}

	if f.accountTokenRegex.MatchString(trimmedPath) && req.Request.Method == "POST" {
		return true
	}

	if trimmedPath == "/signups" && req.Request.Method == "POST" {
		return true
	}

	if (trimmedPath == "/marketing/contacts" || trimmedPath == "/marketing/contacts_from_form") && req.Request.Method == "POST" {
		return true
	}

	if req.Request.Method == "GET" && strings.HasPrefix(trimmedPath, "/subdomain_checks/") {
		return true
	}

	// Allow creating a checkout flow for any account anonymously
	if req.Request.Method == "POST" && f.purchaseRequestRegex.MatchString(trimmedPath) {
		f.logger.Printf("payment request allowed: %s %s", req.Request.Method, req.Request.URL.Path)
		return true
	}

	// if req.Request.Method == "POST" && f.checkoutTokenRegex.MatchString(trimmedPath) {
	// 	f.logger.Printf("checkout token request allowed: %s %s", req.Request.Method, req.Request.URL.Path)
	// 	return true
	// }

	// if req.Request.Method == "GET" && checkoutTokenIdRegex.MatchString(trimmedPath) {
	// 	f.logger.Printf("checkout token request allowed: %s %s", req.Request.Method, req.Request.URL.Path)
	// 	return true
	// }

	// if req.Request.Method == "POST" && f.purchaseRequestRegex.MatchString(trimmedPath) {
	// 	f.logger.Printf("purchase request allowed: %s %s", req.Request.Method, req.Request.URL.Path)
	// 	return true
	// }

	if req.Request.Method == "POST" && trimmedPath == "/checkout_transactions" {
		// TODO: Validate the checkout token
		f.logger.Printf("checkout transaction request allowed: %s %s", req.Request.Method, req.Request.URL.Path)
		return true
	}

	return false
}

func (f BypassAuthFilter) Filter(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	f.logger.Printf("url: %s", req.Request.URL)

	f.logger.Printf("Method: %s", req.Request.Method)
	f.logger.Printf("Path: %s", req.Request.URL.Path)

	if f.checkBypass(req, resp) {
		req.SetAttribute("bypassAuth", true)
	}

	chain.ProcessFilter(req, resp)
}
