/*
OnePay PayOut — Account check + Funds transfer test script.

Based on the official OnePay sample code and your Python test script.

Run:
    go run main.go
*/

package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Credentials (MTF sandbox)
// ---------------------------------------------------------------------------
const (
	PartnerID  = "TESTVFICPO"
	PartnerKey = "C2B5DA903DAE19E215454211C66A59DD"
	AccountID  = "666894931888"
	BaseURL    = "https://mtf.onepay.vn"

	// Happy-case test recipient from the OnePay docs
	TestSwiftCode     = "ICBVVNVX"      // VietinBank (from official sample)
	TestAccountNumber = "103000614434"  // from official sample
	TestHolderName    = "Nguyen Danh Hoang"
	TestAmount         = "10000"         // min 10,000 VND
)

// ---------------------------------------------------------------------------
// Signature constants — matching official Authorization specifications
// ---------------------------------------------------------------------------
const (
	Scheme      = "OWS1"
	Algorithm   = "OWS1-HMAC-SHA256"
	Terminator  = "ows1_request"
	OwsRegion   = "onepay"
	OwsService  = "onepayout"
	DateLayout  = "20060102T150405Z"
	EmptyBodySha = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func hmacSHA256(key []byte, msg string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(msg))
	return h.Sum(nil)
}

func sha256Hex(data []byte) string {
	h := sha256.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// uriEncode mimics Python's urllib.parse.quote(..., safe="~" or "~/")
func uriEncode(data string, encodeSlash bool) string {
	var result strings.Builder
	for i := 0; i < len(data); i++ {
		c := data[i]
		// Safe characters: alphanumeric, '-', '_', '.', '~'
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
			c == '-' || c == '_' || c == '.' || c == '~' {
			result.WriteByte(c)
		} else if c == '/' {
			if encodeSlash {
				result.WriteString("%2F")
			} else {
				result.WriteByte('/')
			}
		} else {
			result.WriteString(fmt.Sprintf("%%%02X", c))
		}
	}
	return result.String()
}

// ---------------------------------------------------------------------------
// Signing Core Logic
// ---------------------------------------------------------------------------

func signRequest(
	accessKeyID string,
	secretKey string,
	region string,
	service string,
	httpMethod string,
	uri string,
	queryParameters map[string]string,
	signedHeaders map[string]string,
	payload []byte,
	timestamp time.Time,
) string {
	// 1. Canonical URI
	canonicalURI := uriEncode(uri, false)

	// 2. Canonical Query String — sort by raw key, then encode key=value
	keys := make([]string, 0, len(queryParameters))
	for k := range queryParameters {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var canonicalQueryParts []string
	for _, k := range keys {
		part := uriEncode(k, true) + "=" + uriEncode(queryParameters[k], true)
		canonicalQueryParts = append(canonicalQueryParts, part)
	}
	canonicalQueryString := strings.Join(canonicalQueryParts, "&")

	// 3. Canonical Headers + SignedHeaders
	headerKeys := make([]string, 0, len(signedHeaders))
	for k := range signedHeaders {
		headerKeys = append(headerKeys, k)
	}
	// Sort header names in lowercase
	sort.Slice(headerKeys, func(i, j int) bool {
		return strings.ToLower(headerKeys[i]) < strings.ToLower(headerKeys[j])
	})

	var canonicalHeaders strings.Builder
	var signedHeaderNames []string
	for _, k := range headerKeys {
		lowerKey := strings.ToLower(k)
		val := strings.TrimSpace(signedHeaders[k])
		canonicalHeaders.WriteString(lowerKey + ":" + val + "\n")
		signedHeaderNames = append(signedHeaderNames, lowerKey)
	}
	joinedSignedHeaderNames := strings.Join(signedHeaderNames, ";")

	// 4. Hashed Payload
	var hashedPayload string
	if payload != nil {
		if len(payload) > 0 {
			hashedPayload = sha256Hex(payload)
		} else {
			hashedPayload = EmptyBodySha
		}
	} else {
		hashedPayload = "UNSIGNED-PAYLOAD"
	}

	// 5. Canonical Request
	canonicalRequest := strings.Join([]string{
		httpMethod,
		canonicalURI,
		canonicalQueryString,
		canonicalHeaders.String(),
		joinedSignedHeaderNames,
		hashedPayload,
	}, "\n")

	// 6. StringToSign
	tsISO := timestamp.UTC().Format(DateLayout)
	tsDate := timestamp.UTC().Format("20060102")
	scope := tsDate + "/" + region + "/" + service + "/" + Terminator
	stringToSign := strings.Join([]string{
		Algorithm,
		tsISO,
		scope,
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")

	// 7. Signing Key — 4 layers: date → region → service → terminator
	dateKey := hmacSHA256([]byte(Scheme+secretKey), tsDate)
	dateRegionKey := hmacSHA256(dateKey, region)
	dateRegionServiceKey := hmacSHA256(dateRegionKey, service)
	signingKey := hmacSHA256(dateRegionServiceKey, Terminator)

	// 8. Signature
	signature := hex.EncodeToString(hmacSHA256(signingKey, stringToSign))

	credential := accessKeyID + "/" + scope

	fmt.Println("===== CanonicalRequest =====")
	fmt.Printf("%q\n", canonicalRequest)
	fmt.Println("===== StringToSign =====")
	fmt.Printf("%q\n", stringToSign)
	fmt.Println("===== Signing Key =====")
	fmt.Println(hex.EncodeToString(signingKey))
	fmt.Println("===== Signature =====")
	fmt.Println(signature)

	return fmt.Sprintf("%s Credential=%s,SignedHeaders=%s,Signature=%s",
		Algorithm, credential, joinedSignedHeaderNames, signature)
}

// sendRequest prints the RAW request metadata and body, executes it, and handles response
func sendRequest(method, requestURL string, headers map[string]string, body []byte) (*http.Response, []byte, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, requestURL, bodyReader)
	if err != nil {
		return nil, nil, err
	}

	// Set headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	fmt.Println()
	fmt.Println("========== RAW HTTP REQUEST ==========")
	fmt.Printf("%s %s %s\n", req.Method, req.URL.RequestURI(), req.Proto)
	for k, vs := range req.Header {
		for _, v := range vs {
			fmt.Printf("%s: %s\n", k, v)
		}
	}
	if len(body) > 0 {
		fmt.Println()
		fmt.Println(string(body))
	}
	fmt.Println("=======================================")
	fmt.Println()

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	fmt.Printf("Status: %d\n", resp.StatusCode)

	// Format response nicely if JSON
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, respBody, "", "  "); err == nil {
		fmt.Println(prettyJSON.String())
	} else {
		fmt.Println(string(respBody))
	}

	return resp, respBody, nil
}

// ---------------------------------------------------------------------------
// 1. Account check (GET /customers)
// ---------------------------------------------------------------------------
func testCheckAccount() (*http.Response, []byte, error) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("TEST 1: Account Check (GET /customers)")
	fmt.Println(strings.Repeat("=", 60))

	ts := time.Now().UTC()
	tsISO := ts.Format(DateLayout)
	requestID := fmt.Sprintf("REQ%d", time.Now().Unix())

	headerSign := map[string]string{
		"Accept":       "application/json",
		"Content-Type": "application/json",
		"X-OP-Date":    tsISO,
		"X-OP-Expires": "3600",
	}

	queryParamMap := map[string]string{
		"request_id":     requestID,
		"swift_code":     TestSwiftCode,
		"account_number": TestAccountNumber,
		"amount":         TestAmount,
		"account_id":     AccountID,
	}

	// Sign signatures using custom canonical-order map values
	queryAfter := "?"
	keys := make([]string, 0, len(queryParamMap))
	for k := range queryParamMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		queryAfter += k + "=" + url.QueryEscape(queryParamMap[k]) + "&"
	}
	queryAfter = strings.TrimSuffix(queryAfter, "&")

	uri := "/onepayout/api/v1/customers"

	authString := signRequest(
		PartnerID,
		PartnerKey,
		OwsRegion,
		OwsService,
		"GET",
		uri,
		queryParamMap,
		headerSign,
		[]byte(""),
		ts,
	)

	urlRequest := BaseURL + uri + queryAfter
	headerRequest := map[string]string{
		"Accept":             "application/json",
		"Content-Type":       "application/json",
		"X-OP-Date":          tsISO,
		"X-OP-Authorization": authString,
		"X-OP-Expires":       "3600",
	}

	return sendRequest("GET", urlRequest, headerRequest, nil)
}

// ---------------------------------------------------------------------------
// 2. Funds Transfer (PUT /accounts/{id}/funds_transfers/{id})
// ---------------------------------------------------------------------------
func testFundsTransfer() (*http.Response, []byte, error) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("TEST 2: Funds Transfer (PUT /accounts/.../funds_transfers/...)")
	fmt.Println(strings.Repeat("=", 60))

	ts := time.Now().UTC()
	tsISO := ts.Format(DateLayout)
	fundsTransferID := fmt.Sprintf("TF%d", time.Now().Unix())

	uri := fmt.Sprintf("/onepayout/api/v1/accounts/%s/funds_transfers/%s", AccountID, fundsTransferID)

	type TransferPayload struct {
		SwiftCode         string `json:"swift_code"`
		AccountNumber     string `json:"account_number"`
		HolderName        string `json:"holder_name"`
		Amount            string `json:"amount"`
		Currency          string `json:"currency"`
		FundsTransferInfo string `json:"funds_transfer_info"`
		Remark            string `json:"remark"`
	}

	payload := TransferPayload{
		SwiftCode:         TestSwiftCode,
		AccountNumber:     TestAccountNumber,
		HolderName:        TestHolderName,
		Amount:            TestAmount,
		Currency:          "VND",
		FundsTransferInfo: fundsTransferID,
		Remark:            "test transfer from script",
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, err
	}

	headerSign := map[string]string{
		"Accept":       "application/json",
		"Content-Type": "application/json",
		"X-OP-Date":    tsISO,
		"X-OP-Expires": "3600",
	}

	authString := signRequest(
		PartnerID,
		PartnerKey,
		OwsRegion,
		OwsService,
		"PUT",
		uri,
		nil, // empty query parameters
		headerSign,
		bodyBytes,
		ts,
	)

	urlRequest := BaseURL + uri
	headerRequest := map[string]string{
		"Accept":             "application/json",
		"Content-Type":       "application/json",
		"X-OP-Date":          tsISO,
		"X-OP-Authorization": authString,
		"X-OP-Expires":       "3600",
	}

	return sendRequest("PUT", urlRequest, headerRequest, bodyBytes)
}

// ---------------------------------------------------------------------------
// Main Flow
// ---------------------------------------------------------------------------
func main() {
	resp, respBody, err := testCheckAccount()
	if err != nil {
		fmt.Printf("\n❌ Account check failed with application error: %v\n", err)
		return
	}

	if resp.StatusCode == 200 {
		var data map[string]interface{}
		if err := json.Unmarshal(respBody, &data); err != nil {
			fmt.Printf("\n❌ Failed to parse JSON response: %v\n", err)
			return
		}

		responseCode, _ := data["response_code"].(string)
		message, _ := data["message"].(string)

		if responseCode == "00" {
			fmt.Println("\n✅ Account check passed, proceeding to transfer...")
			_, _, err := testFundsTransfer()
			if err != nil {
				fmt.Printf("\n❌ Funds transfer failed with application error: %v\n", err)
			}
		} else {
			fmt.Printf("\n❌ Account check failed: %s %s\n", responseCode, message)
		}
	} else {
		fmt.Printf("\n❌ Account check HTTP error: %d\n", resp.StatusCode)
	}
}
