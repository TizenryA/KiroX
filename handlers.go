package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"reg_go/internal/data"
	"reg_go/internal/email"
	"reg_go/internal/proxy"
	"reg_go/internal/storage"
	"reg_go/internal/subscription"
	"reg_go/internal/task"
	"reg_go/internal/updater"
)

// ---- 通用工具 ----

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func readBody(r *http.Request) string {
	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()
	return strings.TrimSpace(string(body))
}

func readJSON(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}



// ---- 登录接口 ----

func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"error": "Method not allowed"})
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "请求格式错误"})
		return
	}

	if req.Password != getPassword() {
		writeJSON(w, http.StatusUnauthorized, map[string]interface{}{"error": "密码错误"})
		return
	}

	token := GenerateToken()
	SaveToken(token)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"token":   token,
	})
}

// ---- 概览 ----

func handleOverview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"error": "Method not allowed"})
		return
	}

	outlookTotal, outlookRegistered, outlookSuccess, outlookPending := countOutlookAccounts()
	taskStatus := task.Manager.GetStatus()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"version": updater.GetCurrentVersion(),
		"kiro": map[string]interface{}{
			"taskRunning":   taskStatus["running"],
			"taskSuccess":   taskStatus["success"],
			"taskFailed":    taskStatus["failed"],
			"taskCompleted": taskStatus["completed"],
			"taskTotal":     taskStatus["total"],
		},
		"outlook": map[string]interface{}{
			"total":      outlookTotal,
			"registered": outlookRegistered,
			"success":    outlookSuccess,
			"pending":    outlookPending,
		},
	})
}

func countOutlookAccounts() (total, registered, success, pending int) {
	accounts := storage.GetAccountsCached()
	if len(accounts) == 0 {
		return
	}
	total = len(accounts)
	for _, acc := range accounts {
		reg, _ := acc["registered"].(bool)
		suc, _ := acc["success"].(bool)
		if reg {
			registered++
			if suc {
				success++
			}
		} else {
			pending++
		}
	}
	return
}

// ---- 任务状态/日志 ----

func handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"error": "Method not allowed"})
		return
	}
	taskStatus := task.Manager.GetStatus()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"kiro": map[string]interface{}{
			"taskRunning":   taskStatus["running"],
			"taskSuccess":   taskStatus["success"],
			"taskFailed":    taskStatus["failed"],
			"taskCompleted": taskStatus["completed"],
			"taskTotal":     taskStatus["total"],
		},
	})
}

func handleLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"error": "Method not allowed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"logs": task.Manager.GetLogs(),
	})
}

// ---- 任务启动/停止 ----

func handleStartTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"error": "Method not allowed"})
		return
	}

	var req task.StartTaskRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "请求格式错误"})
		return
	}

	result := task.StartTask(req)
	writeJSON(w, http.StatusOK, result)
}

func handleStopTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"error": "Method not allowed"})
		return
	}
	result := task.StopTask(true)
	writeJSON(w, http.StatusOK, result)
}

// ---- Outlook 账号 ----

func handleOutlookGet(w http.ResponseWriter, r *http.Request) {
	accounts := email.GetOutlookAccounts()
	writeJSON(w, http.StatusOK, map[string]interface{}{"accounts": accounts})
}

func handleOutlookPost(w http.ResponseWriter, r *http.Request) {
	body := readBody(r)
	if body == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "请求体为空"})
		return
	}
	result := email.AddOutlookAccounts(body)
	writeJSON(w, http.StatusOK, result)
}

func handleOutlookDelete(w http.ResponseWriter, r *http.Request) {
	emailAddr := r.PathValue("email")
	if emailAddr == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "缺少邮箱地址"})
		return
	}
	result := email.DeleteOutlookAccount(emailAddr)
	writeJSON(w, http.StatusOK, result)
}

func handleOutlookClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"error": "Method not allowed"})
		return
	}
	result := email.ClearOutlookAccounts()
	writeJSON(w, http.StatusOK, result)
}

func handleOutlookClearRegistered(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"error": "Method not allowed"})
		return
	}
	result := email.ClearRegisteredOutlookAccounts()
	writeJSON(w, http.StatusOK, result)
}

func handleOutlookImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"error": "Method not allowed"})
		return
	}

	// 解析 multipart 文件
	r.ParseMultipartForm(10 << 20) // 10MB
	file, _, err := r.FormFile("file")
	if err != nil {
		// 如果没有文件，尝试从 body 读取纯文本
		body := readBody(r)
		if body == "" {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "未提供文件"})
			return
		}
		result := email.AddOutlookAccounts(body)
		writeJSON(w, http.StatusOK, result)
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"error": "读取文件失败"})
		return
	}

	result := email.AddOutlookAccounts(string(content))
	writeJSON(w, http.StatusOK, result)
}

// ---- MoeMail ----

func handleMoeMail(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		configs := email.GetMoeMailConfigs()
		writeJSON(w, http.StatusOK, map[string]interface{}{"configs": configs})

	case http.MethodPost:
		body := readBody(r)
		if body == "" {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "请求体为空"})
			return
		}
		result := email.SaveMoeMailConfigs(body)
		writeJSON(w, http.StatusOK, result)

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"error": "Method not allowed"})
	}
}

func handleMoeMailTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"error": "Method not allowed"})
		return
	}
	body := readBody(r)
	if body == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "请求体为空"})
		return
	}
	result := email.TestMoeMailConnection(body)
	writeJSON(w, http.StatusOK, result)
}

// ---- 代理 ----

func handleProxy(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"proxy": storage.GetProxy(),
		})

	case http.MethodPost:
		var req struct {
			Raw string `json:"proxy"`
		}
		if err := readJSON(r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "请求格式错误"})
			return
		}
		normalized, err := storage.SetProxy(req.Raw)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": err.Error()})
			return
		}
		resp := map[string]interface{}{"success": true, "proxy": normalized}
		if normalized != "" {
			resp["detect"] = proxy.Detect(normalized)
		}
		writeJSON(w, http.StatusOK, resp)

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"error": "Method not allowed"})
	}
}

func handleProxyReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"error": "Method not allowed"})
		return
	}
	storage.ResetProxy()
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}

// ---- 数据目录 ----

func handleDataDir(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"path": storage.GetDataDir(),
		})

	case http.MethodPost:
		var req struct {
			Path string `json:"path"`
		}
		if err := readJSON(r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "请求格式错误"})
			return
		}
		path, err := storage.SetDataDirPath(req.Path)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "path": path})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"error": "Method not allowed"})
	}
}

func handleDataDirReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"error": "Method not allowed"})
		return
	}
	path := storage.ResetDataDirPath()
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "path": path})
}

// ---- 输出目录 ----

func handleOutputDir(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"path": storage.GetResultOutputDir(),
		})

	case http.MethodPost:
		var req struct {
			Path string `json:"path"`
		}
		if err := readJSON(r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "请求格式错误"})
			return
		}
		path, err := storage.SetResultOutputDir(req.Path)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "path": path})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"error": "Method not allowed"})
	}
}

func handleOutputDirReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"error": "Method not allowed"})
		return
	}
	path := storage.ResetResultOutputDir()
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "path": path})
}

// ---- 已注册账号 ----

func handleAccounts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"error": "Method not allowed"})
		return
	}
	items, err := data.LoadAccounts(storage.GetResultOutputDir())
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	cache := subscription.LoadCache(storage.GetDataDir())
	for _, m := range items {
		if em, _ := m["email"].(string); em != "" {
			if entry, ok := cache[em]; ok {
				m["cachedUrl"] = entry.URL
				m["cachedPlanType"] = entry.PlanType
				m["cachedFetchedAt"] = entry.FetchedAt
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":   true,
		"accounts":  items,
		"outputDir": storage.GetResultOutputDir(),
	})
}

// ---- 订阅 ----

func accountFromMap(m map[string]interface{}) subscription.Account {
	get := func(k string) string { v, _ := m[k].(string); return v }
	return subscription.Account{
		Email:        get("email"),
		RefreshToken: get("refreshToken"),
		ClientID:     get("clientId"),
		ClientSecret: get("clientSecret"),
		Region:       get("region"),
		Provider:     get("provider"),
		Time:         get("time"),
		Subscription: get("subscription"),
	}
}

func handleSubscriptionPlans(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"error": "Method not allowed"})
		return
	}
	emailParam := r.URL.Query().Get("email")

	items, err := data.LoadAccounts(storage.GetResultOutputDir())
	if err != nil || len(items) == 0 {
		writeJSON(w, http.StatusOK, map[string]interface{}{"success": false, "error": "未找到任何账号"})
		return
	}

	if emailParam != "" {
		for _, m := range items {
			if e, _ := m["email"].(string); e == emailParam {
				acc := accountFromMap(m)
				token, err := subscription.RefreshAccessToken(acc)
				if err != nil {
					writeJSON(w, http.StatusOK, map[string]interface{}{"success": false, "error": err.Error()})
					return
				}
				plans, err := subscription.ListPlans(acc, token)
				if err != nil {
					writeJSON(w, http.StatusOK, map[string]interface{}{"success": false, "error": err.Error()})
					return
				}
				writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "plans": plans})
				return
			}
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"success": false, "error": "未找到账号: " + emailParam})
		return
	}

	var lastErr error
	for _, m := range items {
		acc := accountFromMap(m)
		if acc.RefreshToken == "" || acc.ClientID == "" {
			continue
		}
		token, err := subscription.RefreshAccessToken(acc)
		if err != nil {
			lastErr = err
			continue
		}
		plans, err := subscription.ListPlans(acc, token)
		if err != nil {
			lastErr = err
			continue
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "plans": plans})
		return
	}
	msg := "全部账号均无法获取计划列表"
	if lastErr != nil {
		msg = lastErr.Error()
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": false, "error": msg})
}

func handleSubscriptionLink(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"error": "Method not allowed"})
		return
	}

	var req struct {
		Email    string `json:"email"`
		PlanType string `json:"planType"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "请求格式错误"})
		return
	}

	items, err := data.LoadAccounts(storage.GetResultOutputDir())
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}

	var acc subscription.Account
	for _, m := range items {
		if e, _ := m["email"].(string); e == req.Email {
			acc = accountFromMap(m)
			break
		}
	}
	if acc.Email == "" {
		writeJSON(w, http.StatusOK, map[string]interface{}{"success": false, "error": "未找到账号: " + req.Email})
		return
	}

	token, err := subscription.RefreshAccessToken(acc)
	if err != nil {
		if subscription.IsSuspended(err) {
			removed, _ := data.DeleteAccount(storage.GetResultOutputDir(), req.Email)
			subscription.DeleteCache(storage.GetDataDir(), req.Email)
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"success": false, "error": err.Error(), "suspended": true, "removed": removed,
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}

	url, err := subscription.CreateSubscriptionLink(acc, token, req.PlanType)
	if err != nil {
		if subscription.IsSuspended(err) {
			removed, _ := data.DeleteAccount(storage.GetResultOutputDir(), req.Email)
			subscription.DeleteCache(storage.GetDataDir(), req.Email)
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"success": false, "error": err.Error(), "suspended": true, "removed": removed,
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}

	_ = subscription.PutCache(storage.GetDataDir(), req.Email, url, req.PlanType)
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "url": url})
}

// ---- 检查更新 ----

func handleCheckUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"error": "Method not allowed"})
		return
	}
	result := updater.CheckUpdate()
	writeJSON(w, http.StatusOK, result)
}


