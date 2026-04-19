package gapi

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

const (
	gatewayURL = "http://localhost:8080"
	dbSource   = "postgresql://root:secret@localhost:5433/escrow_db?sslmode=disable"
)

// httpGet performs a GET request with optional query params and Bearer token.
func httpGet(path string, params map[string]string, token string) (*http.Response, string, error) {
	reqURL := gatewayURL + path
	if len(params) > 0 {
		q := url.Values{}
		for k, v := range params {
			q.Add(k, v)
		}
		reqURL += "?" + q.Encode()
	}
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, "", err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return resp, "", err
	}
	body, _ := io.ReadAll(resp.Body)
	return resp, string(body), nil
}

// httpPost performs a POST request with JSON body and Bearer token.
func httpPost(path string, body interface{}, token string) (*http.Response, string, error) {
	var bodyBytes []byte
	if body != nil {
		bodyBytes, _ = json.Marshal(body)
	}
	req, err := http.NewRequest(http.MethodPost, gatewayURL+path, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return resp, "", err
	}
	respBody, _ := io.ReadAll(resp.Body)
	return resp, string(respBody), nil
}

// login obtains a JWT for the given username/password.
func login(t *testing.T, username, password string) string {
	body := map[string]interface{}{"username": username, "password": password}
	resp, respBody, err := httpPost("/api/v1/auth/login", body, "")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode, "login failed: %s", respBody)

	var result map[string]interface{}
	err = json.Unmarshal([]byte(respBody), &result)
	require.NoError(t, err, "response body: %s", respBody)

	// SimpleBank returns "access_token", not "token"
	token, ok := result["access_token"].(string)
	require.True(t, ok, "no access_token in login response: %s", respBody)
	return token
}

// getAccountID fetches the account ID for a given username via the API.
func getAccountID(t *testing.T, token string) int64 {
	resp, body, err := httpGet("/api/v1/account", nil, token)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode, "get account failed: %s", body)

	var result map[string]interface{}
	err = json.Unmarshal([]byte(body), &result)
	require.NoError(t, err, body)

	accounts, ok := result["accounts"].([]interface{})
	require.True(t, ok, "no accounts in response: %s", body)
	require.NotEmpty(t, accounts, "no accounts found for user")

	account := accounts[0].(map[string]interface{})
	// ID is returned as string from gRPC-JSON gateway
	idStr, ok := account["id"].(string)
	require.True(t, ok, "account id is not a string: %v", account["id"])
	id, err := strconv.ParseInt(idStr, 10, 64)
	require.NoError(t, err)
	return id
}

// TestFulfillmentE2E_FullBountyFlow tests the complete bounty lifecycle
// including: create → accept → confirm → submit → approve → review → fulfillment_index update.
func TestFulfillmentE2E_FullBountyFlow(t *testing.T) {
	// ── Step 0: Verify services are running ──────────────────────────────
	resp, _, err := httpGet("/health", nil, "")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Skip("Gateway not running at localhost:8080, skipping e2e test")
	}

	// ── Step 1: Login both users ─────────────────────────────────────────
	employerToken := login(t, "zhangsan", "12345678")
	hunterToken := login(t, "lisi", "12345678")
	t.Logf("[Step1] Login OK: employer=%s..., hunter=%s...", employerToken[:20], hunterToken[:20])

	// Get employer account ID for creating bounty
	employerAccountID := getAccountID(t, employerToken)
	t.Logf("[Step1b] Employer account ID: %d", employerAccountID)

	// ── Step 2: Employer creates bounty ───────────────────────────────────
	// deadline_timestamp is proto field name, gateway gRPC-JSON uses camelCase by default
	createReq := map[string]interface{}{
		"title":               "E2E测试悬赏_" + strconv.FormatInt(time.Now().UnixNano(), 36),
		"description":         "用于履约系统全链路测试",
		"rewardAmount":        10000, // 100.00 元（zhangsan 余额约 134 元）
		"employerAccountId":  employerAccountID,
		"deadlineTimestamp":   time.Now().Add(72 * time.Hour).Unix(),
	}
	resp, body, err := httpPost("/api/v1/bounties", createReq, employerToken)
	require.NoError(t, err, "request error: %v", err)
	require.Equal(t, http.StatusOK, resp.StatusCode, "create bounty failed (status=%d): %s", resp.StatusCode, body)

	var bountyResp map[string]interface{}
	err = json.Unmarshal([]byte(body), &bountyResp)
	require.NoError(t, err, body)

	bountyData, ok := bountyResp["bounty"].(map[string]interface{})
	require.True(t, ok, "no bounty in response: %s", body)
	bountyID, err := strconv.ParseInt(bountyData["id"].(string), 10, 64)
	require.NoError(t, err)
	t.Logf("[Step2] Bounty created: id=%d, status=%v", bountyID, bountyData["status"])

	// ── Step 3: Hunter accepts bounty (POST /api/v1/bounties/{id}/accept) ─
	// Note: NOT /apply, the actual API is /accept
	acceptResp, acceptBody, err := httpPost(fmt.Sprintf("/api/v1/bounties/%d/accept", bountyID),
		map[string]interface{}{"bountyId": bountyID}, hunterToken)
	require.NoError(t, err, "accept request error: %v", err)
	require.Equal(t, http.StatusOK, acceptResp.StatusCode, "accept failed: %s", acceptBody)
	t.Logf("[Step3] Hunter accepted bounty: %s", acceptBody)

	// Extract application ID from accept response
	var acceptData map[string]interface{}
	json.Unmarshal([]byte(acceptBody), &acceptData)
	appData := acceptData["application"].(map[string]interface{})
	appIDStr := appData["id"].(string)
	applicationID, err := strconv.ParseInt(appIDStr, 10, 64)
	require.NoError(t, err)
	t.Logf("[Step3b] Application ID: %d", applicationID)

	// ── Step 4: Employer confirms hunter (POST /api/v1/bounties/{id}/confirm) ─
	// Proto field is application_id (snake_case)
	confirmResp, confirmBody, err := httpPost(fmt.Sprintf("/api/v1/bounties/%d/confirm", bountyID),
		map[string]interface{}{"applicationId": applicationID}, employerToken)
	require.NoError(t, err, "confirm request error: %v", err)
	require.Equal(t, http.StatusOK, confirmResp.StatusCode, "confirm failed: %s", confirmBody)
	t.Logf("[Step4] Employer confirmed hunter: %s", confirmBody)

	// ── Step 5: Hunter submits work (POST /api/v1/bounties/{id}/submit) ───
	// Proto field is submission_text
	submitResp, submitBody, err := httpPost(fmt.Sprintf("/api/v1/bounties/%d/submit", bountyID),
		map[string]interface{}{"bountyId": bountyID, "submissionText": "E2E 测试工作成果已完成"}, hunterToken)
	require.NoError(t, err, "submit request error: %v", err)
	require.Equal(t, http.StatusOK, submitResp.StatusCode, "submit failed: %s", submitBody)
	t.Logf("[Step5] Hunter submitted: %s", submitBody)

	// ── Step 6: Employer approves (POST /api/v1/bounties/{id}/approve) ─────
	// ApproveBounty: SUBMITTED → COMPLETED, writes task_records for both parties
	approveResp, approveBody, err := httpPost(fmt.Sprintf("/api/v1/bounties/%d/approve", bountyID),
		map[string]interface{}{"bountyId": bountyID}, employerToken)
	require.NoError(t, err, "approve request error: %v", err)
	require.Equal(t, http.StatusOK, approveResp.StatusCode, "approve failed: %s", approveBody)

	var approvedBounty map[string]interface{}
	json.Unmarshal([]byte(approveBody), &approvedBounty)
	approvedBountyData := approvedBounty["bounty"].(map[string]interface{})
	require.Equal(t, "COMPLETED", approvedBountyData["status"], "status should be COMPLETED: %s", approveBody)
	t.Logf("[Step6] Bounty COMPLETED: id=%d", bountyID)

	// ── Step 7: Hunter reviews employer ──────────────────────────────────
	review1Body, err := postReview(t, hunterToken, "zhangsan", bountyID, 5, "好雇主，合作愉快", "HUNTER_TO_EMPLOYER")
	require.NoError(t, err, "hunter review failed: %s", review1Body)
	t.Logf("[Step7] Hunter reviewed employer: %s", review1Body)

	// ── Step 8: Employer reviews hunter ──────────────────────────────────
	review2Body, err := postReview(t, employerToken, "lisi", bountyID, 5, "好猎人，质量不错", "EMPLOYER_TO_HUNTER")
	require.NoError(t, err, "employer review failed: %s", review2Body)
	t.Logf("[Step8] Employer reviewed hunter: %s", review2Body)

	// ── Step 9: Give async workers time to process ─────────────────────────
	time.Sleep(5 * time.Second)

	// ── Step 10: Verify database state ────────────────────────────────────
	db, err := sql.Open("postgres", dbSource)
	require.NoError(t, err)
	defer db.Close()

	// 10a: Check task_records exist (hunter + employer each gets one record)
	var taskCount int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM task_records
		WHERE bounty_id = $1`, bountyID).Scan(&taskCount)
	require.NoError(t, err)
	require.Equal(t, 2, taskCount, "should have 2 task_records (hunter + employer role)")
	t.Logf("[Step10a] task_records count: %d (expected 2)", taskCount)

	// 10b: Check fulfillment_outbox records exist
	var outboxCount int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM fulfillment_outbox
		WHERE bounty_id = $1`, bountyID).Scan(&outboxCount)
	require.NoError(t, err)
	t.Logf("[Step10b] fulfillment_outbox count: %d", outboxCount)

	// 10c: Check fulfillment indices updated correctly
	var hunterFI, employerFI int
	rows, err := db.Query(`
		SELECT username, hunter_fulfillment_index, employer_fulfillment_index
		FROM user_profiles WHERE username IN ('zhangsan','lisi')`)
	require.NoError(t, err)
	defer rows.Close()

	found := map[string]bool{"zhangsan": false, "lisi": false}
	for rows.Next() {
		var n, h, e string
		require.NoError(t, rows.Scan(&n, &h, &e))
		t.Logf("  user=%s hunter_fulfillment_index=%s employer_fulfillment_index=%s", n, h, e)
		if n == "lisi" {
			hunterFI = mustInt(h)
			found["lisi"] = true
		} else if n == "zhangsan" {
			employerFI = mustInt(e)
			found["zhangsan"] = true
		}
	}
	require.True(t, found["lisi"], "lisi profile not found")
	require.True(t, found["zhangsan"], "zhangsan profile not found")

	// Hunter (lisi): hunter_fulfillment_index should be > 50
	require.Greater(t, hunterFI, 50, "lisi hunter_fulfillment_index should be > 50, got %d", hunterFI)
	// Employer (zhangsan): employer_fulfillment_index should be > 50
	require.Greater(t, employerFI, 50, "zhangsan employer_fulfillment_index should be > 50, got %d", employerFI)

	t.Logf("[Step10c] PASS: lisi hunter_fulfillment_index=%d (>50), zhangsan employer_fulfillment_index=%d (>50)",
		hunterFI, employerFI)
}

// postReview submits a review via the HTTP API.
func postReview(t *testing.T, token, reviewedUsername string, bountyID int64, rating int, comment, reviewType string) (string, error) {
	body := map[string]interface{}{
		"reviewedUsername": reviewedUsername,
		"bountyId":        bountyID,
		"rating":          rating,
		"comment":          comment,
		"reviewType":      reviewType,
	}
	_, respBody, err := httpPost("/api/v1/reviews", body, token)
	return respBody, err
}

func mustInt(s string) int {
	v, _ := strconv.Atoi(s)
	return v
}
