/**
 * Kirox Web - Frontend API Layer
 * Replaces Wails bindings with standard fetch API calls.
 * All requests include Authorization: Bearer *** from localStorage.
 * All functions are global (exposed on window) so other scripts can call them directly.
 */

var API_BASE = (typeof window !== 'undefined' && window.__KIROX_API_BASE) || '';

function _apiGetToken() {
  return localStorage.getItem('kirox_token') || '';
}

function _apiAuthHeaders(extra) {
  var h = { 'Authorization': 'Bearer ' + _apiGetToken() };
  if (extra) {
    for (var k in extra) { if (extra.hasOwnProperty(k)) h[k] = extra[k]; }
  }
  return h;
}

async function _apiRequest(method, path, opts) {
  opts = opts || {};
  var url = API_BASE + path;
  if (opts.params) {
    var entries = [];
    for (var k in opts.params) {
      if (opts.params.hasOwnProperty(k) && opts.params[k] != null) {
        entries.push(encodeURIComponent(k) + '=' + encodeURIComponent(opts.params[k]));
      }
    }
    if (entries.length) url += '?' + entries.join('&');
  }

  var fetchOpts = {
    method: method,
    headers: _apiAuthHeaders(opts.headers)
  };

  if (opts.body !== undefined) {
    if (opts.body instanceof FormData) {
      fetchOpts.body = opts.body;
      // Let browser set Content-Type with boundary
    } else {
      fetchOpts.headers['Content-Type'] = 'application/json';
      fetchOpts.body = JSON.stringify(opts.body);
    }
  }

  var res = await fetch(url, fetchOpts);

  // Handle 401 - token expired, redirect to login
  if (res.status === 401) {
    localStorage.removeItem('kirox_token');
    if (typeof showLoginPage === 'function') {
      showLoginPage();
    }
    throw new Error('认证已过期，请重新登录');
  }

  if (opts.raw) return res;

  var text = await res.text();
  var data;
  try {
    data = text ? JSON.parse(text) : null;
  } catch (e) {
    data = text;
  }

  if (!res.ok) {
    var msg = (data && data.error) || (data && data.message) || res.statusText;
    throw new Error(msg);
  }
  return data;
}

function _apiGet(path, opts) { return _apiRequest('GET', path, opts); }
function _apiPost(path, opts) { return _apiRequest('POST', path, opts); }
function _apiDel(path, opts) { return _apiRequest('DELETE', path, opts); }

// ──────────────────────────────────────────────
// API Functions (global, replacing Wails bindings)
// ──────────────────────────────────────────────

/** Login(password) -> POST /api/login */
async function Login(password) {
  return _apiPost('/api/login', { body: { password: password } });
}

/** GetOverview -> GET /api/overview */
async function GetOverview() {
  return _apiGet('/api/overview');
}

/** GetStatus -> GET /api/status */
async function GetStatus() {
  var result = await _apiGet('/api/status');
  // 后端返回 {"kiro": {"taskRunning": bool, ...}}，转换为前端期望的格式
  if (result && result.kiro) {
    return {
      running: result.kiro.taskRunning || false,
      success: result.kiro.taskSuccess || 0,
      failed: result.kiro.taskFailed || 0,
      completed: result.kiro.taskCompleted || 0,
      total: result.kiro.taskTotal || 0
    };
  }
  return result;
}

/** GetLogs -> GET /api/logs */
async function GetLogs() {
  var result = await _apiGet('/api/logs');
  // 后端返回 {"logs": [...]}，提取 logs 数组
  return result && result.logs ? result.logs : [];
}

/** StartTask(cfg) -> POST /api/task/start */
async function StartTask(cfg) {
  return _apiPost('/api/task/start', { body: cfg });
}

/** StopTask -> POST /api/task/stop */
async function StopTask() {
  return _apiPost('/api/task/stop');
}

/** GetTaskStatus -> GET /api/status */
async function GetTaskStatus() {
  return _apiGet('/api/status');
}

/** GetOutlookAccounts -> GET /api/outlook */
async function GetOutlookAccounts() {
  return _apiGet('/api/outlook');
}

/** AddOutlookAccounts(data) -> POST /api/outlook */
async function AddOutlookAccounts(data) {
  // data 是原始文本，直接发送，不要 JSON.stringify
  var url = API_BASE + '/api/outlook';
  var res = await fetch(url, {
    method: 'POST',
    headers: _apiAuthHeaders({ 'Content-Type': 'text/plain' }),
    body: data
  });
  if (res.status === 401) {
    localStorage.removeItem('kirox_token');
    if (typeof showLoginPage === 'function') showLoginPage();
    throw new Error('认证已过期，请重新登录');
  }
  var text = await res.text();
  var result;
  try { result = text ? JSON.parse(text) : null; } catch(e) { result = text; }
  if (!res.ok) throw new Error((result && result.error) || res.statusText);
  return result;
}

/** DeleteOutlookAccount(email) -> DELETE /api/outlook?email=xxx */
async function DeleteOutlookAccount(email) {
  return _apiDel('/api/outlook', { params: { email: email } });
}

/** ClearOutlookAccounts -> POST /api/outlook/clear */
async function ClearOutlookAccounts() {
  return _apiPost('/api/outlook/clear');
}

/** ClearRegisteredOutlookAccounts -> POST /api/outlook/clear-registered */
async function ClearRegisteredOutlookAccounts() {
  return _apiPost('/api/outlook/clear-registered');
}

/**
 * ImportOutlookFile(file) -> POST /api/outlook/import
 * In web mode, file should be a File object (from <input type="file">).
 */
async function ImportOutlookFile(file) {
  var fd = new FormData();
  fd.append('file', file);
  return _apiPost('/api/outlook/import', { body: fd });
}

/** GetMoeMailConfigs -> GET /api/moemail */
async function GetMoeMailConfigs() {
  return _apiGet('/api/moemail');
}

/** SaveMoeMailConfigs(json) -> POST /api/moemail */
async function SaveMoeMailConfigs(json) {
  // json 已经是 JSON 字符串，直接发送
  var url = API_BASE + '/api/moemail';
  var res = await fetch(url, {
    method: 'POST',
    headers: _apiAuthHeaders({ 'Content-Type': 'application/json' }),
    body: json
  });
  if (res.status === 401) {
    localStorage.removeItem('kirox_token');
    if (typeof showLoginPage === 'function') showLoginPage();
    throw new Error('认证已过期，请重新登录');
  }
  var text = await res.text();
  var result;
  try { result = text ? JSON.parse(text) : null; } catch(e) { result = text; }
  if (!res.ok) throw new Error((result && result.error) || res.statusText);
  return result;
}

/** TestMoeMailConnection(json) -> POST /api/moemail/test */
async function TestMoeMailConnection(json) {
  // json 已经是 JSON 字符串，直接发送
  var url = API_BASE + '/api/moemail/test';
  var res = await fetch(url, {
    method: 'POST',
    headers: _apiAuthHeaders({ 'Content-Type': 'application/json' }),
    body: json
  });
  if (res.status === 401) {
    localStorage.removeItem('kirox_token');
    if (typeof showLoginPage === 'function') showLoginPage();
    throw new Error('认证已过期，请重新登录');
  }
  var text = await res.text();
  var result;
  try { result = text ? JSON.parse(text) : null; } catch(e) { result = text; }
  if (!res.ok) throw new Error((result && result.error) || res.statusText);
  return result;
}

/** GetProxy -> GET /api/proxy */
async function GetProxy() {
  return _apiGet('/api/proxy');
}

/** SetProxy(proxy) -> POST /api/proxy */
async function SetProxy(proxy) {
  return _apiPost('/api/proxy', { body: { proxy: proxy } });
}

/** ResetProxy -> POST /api/proxy/reset */
async function ResetProxy() {
  return _apiPost('/api/proxy/reset');
}

/** GetDataDir -> GET /api/datadir */
async function GetDataDir() {
  return _apiGet('/api/datadir');
}

/** SetDataDir(dir) -> POST /api/datadir */
async function SetDataDir(dir) {
  return _apiPost('/api/datadir', { body: { dir: dir } });
}

/** ResetDataDir -> POST /api/datadir/reset */
async function ResetDataDir() {
  return _apiPost('/api/datadir/reset');
}

/** GetResultOutputDir -> GET /api/outputdir */
async function GetResultOutputDir() {
  return _apiGet('/api/outputdir');
}

/** SetResultOutputDir(dir) -> POST /api/outputdir */
async function SetResultOutputDir(dir) {
  return _apiPost('/api/outputdir', { body: { dir: dir } });
}

/** ResetResultOutputDir -> POST /api/outputdir/reset */
async function ResetResultOutputDir() {
  return _apiPost('/api/outputdir/reset');
}

/** LoadOutputAccounts -> GET /api/accounts */
async function LoadOutputAccounts() {
  return _apiGet('/api/accounts');
}

/** GetSubscriptionPlans(email) -> GET /api/subscription/plans?email=xxx */
async function GetSubscriptionPlans(email) {
  return _apiGet('/api/subscription/plans', { params: { email: email } });
}

/** GetSubscriptionLink(email, planType) -> POST /api/subscription/link */
async function GetSubscriptionLink(email, planType) {
  return _apiPost('/api/subscription/link', { body: { email: email, planType: planType } });
}

/** CheckUpdate -> POST /api/check-update */
async function CheckUpdate() {
  return _apiPost('/api/check-update');
}

/**
 * SelectDirectory -> removed (web does not support native file dialogs)
 * Kept as a no-op stub so existing code doesn't break.
 */
async function SelectDirectory() {
  console.warn('SelectDirectory is not available in web mode.');
  return '';
}

/**
 * SelectOutlookFile -> returns a File via <input type="file">.
 * Returns a Promise that resolves with the chosen File or null if cancelled.
 */
function SelectOutlookFile() {
  return new Promise(function(resolve) {
    var input = document.createElement('input');
    input.type = 'file';
    input.accept = '.txt,.csv,.json';
    input.onchange = function() { resolve(input.files[0] || null); };
    input.click();
  });
}

/** OpenURL(url) -> open in new tab */
function OpenURL(url) {
  window.open(url, '_blank', 'noopener,noreferrer');
}
