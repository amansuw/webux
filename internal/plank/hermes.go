// Package plank manages a local Hermes Agent serve process for the P.L.A.N.K. chat UI.
package plank

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	DefaultHost = "127.0.0.1"
	DefaultPort = 9119
	SourceTag   = "webux"
)

var tokenRe = regexp.MustCompile(`window\.__HERMES_SESSION_TOKEN__\s*=\s*"([^"]+)"`)

// Manager ensures hermes serve is reachable and provides the session token for WS auth.
type Manager struct {
	Host string
	Port int

	mu        sync.Mutex
	token     string
	cmd       *exec.Cmd
	startedBy bool // true if we spawned the process
	logFile   *os.File
}

func NewManager() *Manager {
	return &Manager{
		Host: DefaultHost,
		Port: DefaultPort,
	}
}

func (m *Manager) baseURL() string {
	return fmt.Sprintf("http://%s:%d", m.Host, m.Port)
}

func (m *Manager) wsURL(token string) string {
	return fmt.Sprintf("ws://%s:%d/api/ws?token=%s", m.Host, m.Port, token)
}

// Status is returned by GET /api/plank/status.
type Status struct {
	Online      bool   `json:"online"`
	Serving     bool   `json:"serving"`
	URL         string `json:"url"`
	Model       string `json:"model"`
	Provider    string `json:"provider"`
	Skin        string `json:"skin"`
	Personality string `json:"personality"`
	AgentName   string `json:"agent_name"`
	Tagline     string `json:"tagline"`
	Error       string `json:"error,omitempty"`
	StartedByUs bool   `json:"started_by_us"`
}

// Ensure brings hermes serve up if needed and refreshes the session token.
func (m *Manager) Ensure(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.probeLocked(ctx); err == nil {
		if err := m.refreshTokenLocked(ctx); err == nil && m.token != "" {
			return nil
		}
		// Healthy but token unknown (no SPA inject) — take ownership.
		slog.Info("plank: hermes healthy but token unknown; restarting serve under WebUX token")
		m.stopHermesCLI()
		time.Sleep(500 * time.Millisecond)
	}

	if err := m.startLocked(ctx); err != nil {
		return err
	}

	deadline := time.Now().Add(25 * time.Second)
	for time.Now().Before(deadline) {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := m.probeLocked(ctx); err == nil {
			if m.token != "" {
				return nil
			}
			return m.refreshTokenLocked(ctx)
		}
		time.Sleep(400 * time.Millisecond)
	}
	return fmt.Errorf("hermes serve did not become healthy on %s", m.baseURL())
}

// Token returns the current Hermes dashboard session token (call Ensure first).
func (m *Manager) Token() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.token
}

// WSURL returns the authenticated Hermes JSON-RPC WebSocket URL.
func (m *Manager) WSURL() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.wsURL(m.token)
}

// GetStatus probes Hermes and reads lightweight config for the UI.
func (m *Manager) GetStatus(ctx context.Context) Status {
	st := Status{
		URL:         m.baseURL(),
		AgentName:   "P.L.A.N.K.",
		Tagline:     "Possibly Lethal Automaton for Neural Knowledge",
		Personality: "Honesty 90% · Humor 75% · Lethality 0%",
		StartedByUs: m.startedByUs(),
	}

	if err := m.Ensure(ctx); err != nil {
		st.Error = err.Error()
		st.Online = false
		return st
	}

	st.Online = true
	st.Serving = true
	st.StartedByUs = m.startedByUs()

	cfg := readHermesConfig()
	st.Model = cfg.Model
	st.Provider = cfg.Provider
	st.Skin = cfg.Skin
	if cfg.Skin == "" {
		st.Skin = "plank"
	}
	return st
}

func (m *Manager) startedByUs() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.startedBy
}

func (m *Manager) probeLocked(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.baseURL()+"/api/health", nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 2 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 500 {
		return fmt.Errorf("health status %d", res.StatusCode)
	}
	return nil
}

func (m *Manager) refreshTokenLocked(ctx context.Context) error {
	if m.token != "" && m.startedBy {
		// We own the process and set the env token — keep using it.
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.baseURL()+"/", nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 3 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return err
	}
	match := tokenRe.FindSubmatch(body)
	if len(match) < 2 {
		if m.token != "" {
			return nil
		}
		return fmt.Errorf("could not discover Hermes session token")
	}
	m.token = string(match[1])
	return nil
}

func (m *Manager) startLocked(ctx context.Context) error {
	bin, hermesHome, err := lookHermes()
	if err != nil {
		return err
	}

	if m.token == "" {
		tok, err := randomToken()
		if err != nil {
			return err
		}
		m.token = tok
	}

	// Prefer HERMES_HOME/logs — /tmp is often sticky + fs.protected_regular,
	// so a user-owned 0600 log there is unwritable even to root in some setups.
	logPath, lf := openServeLog(hermesHome)
	if lf != nil {
		m.logFile = lf
	}

	cmd := exec.CommandContext(context.Background(), bin, "serve",
		"--host", m.Host,
		"--port", fmt.Sprintf("%d", m.Port),
		"--skip-build",
	)
	env := append(os.Environ(),
		"HERMES_DASHBOARD_SESSION_TOKEN="+m.token,
	)
	if hermesHome != "" {
		env = append(env, "HERMES_HOME="+hermesHome, "HOME="+filepath.Dir(hermesHome))
	}
	cmd.Env = env
	if lf != nil {
		cmd.Stdout = lf
		cmd.Stderr = lf
	} else {
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
	}
	cmd.Dir = hermesHome
	if cred := credentialForHome(hermesHome); cred != nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{Credential: cred}
	}
	// Detach from WebUX lifecycle somewhat — if WebUX dies, OS reaps; Ensure restarts later.
	if err := cmd.Start(); err != nil {
		if lf != nil {
			lf.Close()
		}
		return fmt.Errorf("start hermes serve: %w", err)
	}
	m.cmd = cmd
	m.startedBy = true
	slog.Info("plank: started hermes serve", "pid", cmd.Process.Pid, "url", m.baseURL(), "log", logPath, "hermes_home", hermesHome)

	go func() {
		err := cmd.Wait()
		m.mu.Lock()
		defer m.mu.Unlock()
		if m.cmd == cmd {
			m.cmd = nil
			m.startedBy = false
		}
		if err != nil {
			slog.Warn("plank: hermes serve exited", "err", err)
		}
	}()

	_ = ctx
	return nil
}

func (m *Manager) stopHermesCLI() {
	bin, hermesHome, err := lookHermes()
	if err != nil {
		return
	}
	cmd := exec.Command(bin, "serve", "--stop")
	env := os.Environ()
	if hermesHome != "" {
		env = append(env, "HERMES_HOME="+hermesHome, "HOME="+filepath.Dir(hermesHome))
	}
	cmd.Env = env
	_ = cmd.Run()
	if m.cmd != nil && m.cmd.Process != nil {
		_ = m.cmd.Process.Kill()
		m.cmd = nil
		m.startedBy = false
	}
}

// lookHermes returns hermes binary path and HERMES_HOME (may be empty).
func lookHermes() (bin string, hermesHome string, err error) {
	homes := uniqueNonEmpty(
		os.Getenv("HERMES_HOME"),
		filepath.Join(os.Getenv("HOME"), ".hermes"),
	)
	if entries, e := os.ReadDir("/home"); e == nil {
		for _, ent := range entries {
			if ent.IsDir() {
				homes = append(homes, filepath.Join("/home", ent.Name(), ".hermes"))
			}
		}
		homes = uniqueNonEmpty(homes...)
	}
	// Prefer an existing Hermes home that has config.yaml.
	for _, h := range homes {
		if st, e := os.Stat(filepath.Join(h, "config.yaml")); e == nil && !st.IsDir() {
			hermesHome = h
			break
		}
	}
	if hermesHome == "" && len(homes) > 0 {
		hermesHome = homes[0]
	}

	candidates := []string{}
	if hermesHome != "" {
		candidates = append(candidates,
			filepath.Join(hermesHome, "hermes-agent/venv/bin/hermes"),
			filepath.Join(filepath.Dir(hermesHome), ".local/bin/hermes"),
		)
	}
	candidates = append(candidates,
		"hermes",
		filepath.Join(os.Getenv("HOME"), ".local/bin/hermes"),
		"/usr/local/bin/hermes",
	)
	if entries, e := os.ReadDir("/home"); e == nil {
		for _, ent := range entries {
			if ent.IsDir() {
				candidates = append(candidates, filepath.Join("/home", ent.Name(), ".local/bin/hermes"))
			}
		}
	}

	for _, c := range candidates {
		if c == "hermes" {
			if p, e := exec.LookPath("hermes"); e == nil {
				return p, hermesHome, nil
			}
			continue
		}
		if st, e := os.Stat(c); e == nil && !st.IsDir() {
			return c, hermesHome, nil
		}
	}
	return "", "", fmt.Errorf("hermes binary not found — install Hermes Agent or put hermes on PATH")
}

func openServeLog(hermesHome string) (string, *os.File) {
	candidates := []string{}
	if hermesHome != "" {
		candidates = append(candidates, filepath.Join(hermesHome, "logs", "webux-serve.log"))
	}
	if dataDir := os.Getenv("WEBUX_DATA_DIR"); dataDir != "" {
		candidates = append(candidates, filepath.Join(dataDir, "hermes-serve.log"))
	}
	candidates = append(candidates, filepath.Join(os.TempDir(), "webux-hermes-serve.log"))

	for _, path := range candidates {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			continue
		}
		// Replace a foreign-owned sticky-dir file we cannot append to.
		if st, err := os.Stat(path); err == nil {
			if st.Mode().Perm()&0o200 == 0 {
				_ = os.Remove(path)
			}
		}
		lf, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			slog.Warn("plank: hermes log unavailable", "path", path, "err", err)
			continue
		}
		return path, lf
	}
	slog.Warn("plank: discarding hermes serve stdout/stderr (no writable log path)")
	return "", nil
}

func credentialForHome(hermesHome string) *syscall.Credential {
	if hermesHome == "" || os.Geteuid() != 0 {
		return nil
	}
	info, err := os.Stat(hermesHome)
	if err != nil {
		return nil
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return nil
	}
	return &syscall.Credential{Uid: st.Uid, Gid: st.Gid}
}

func uniqueNonEmpty(vals ...string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(vals))
	for _, v := range vals {
		v = strings.TrimSpace(v)
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

type hermesCfg struct {
	Model    string
	Provider string
	Skin     string
}

func readHermesConfig() hermesCfg {
	_, hermesHome, _ := lookHermes()
	candidates := []string{}
	if hermesHome != "" {
		candidates = append(candidates, filepath.Join(hermesHome, "config.yaml"))
	}
	candidates = append(candidates, filepath.Join(os.Getenv("HOME"), ".hermes", "config.yaml"))
	if entries, e := os.ReadDir("/home"); e == nil {
		for _, ent := range entries {
			if ent.IsDir() {
				candidates = append(candidates, filepath.Join("/home", ent.Name(), ".hermes", "config.yaml"))
			}
		}
	}
	var data []byte
	var err error
	for _, path := range candidates {
		data, err = os.ReadFile(path)
		if err == nil {
			break
		}
	}
	if err != nil {
		return hermesCfg{}
	}
	// Minimal parse without pulling a YAML dep into this package's hot path —
	// go.mod already has yaml.v3; use simple line scan for robustness.
	out := hermesCfg{}
	lines := strings.Split(string(data), "\n")
	section := ""
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))
		if indent == 0 && strings.HasSuffix(trimmed, ":") {
			section = strings.TrimSuffix(trimmed, ":")
			continue
		}
		key, val, ok := strings.Cut(trimmed, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.Trim(strings.TrimSpace(val), `"'`)
		switch {
		case section == "model" && key == "default":
			out.Model = val
		case section == "model" && key == "provider":
			out.Provider = val
		case section == "display" && key == "skin":
			out.Skin = val
		}
	}
	return out
}

// InjectWebuxSource rewrites a JSON-RPC request so session.create carries source=webux.
func InjectWebuxSource(raw []byte) []byte {
	var msg map[string]any
	if err := json.Unmarshal(raw, &msg); err != nil {
		return raw
	}
	method, _ := msg["method"].(string)
	if method != "session.create" {
		return raw
	}
	params, _ := msg["params"].(map[string]any)
	if params == nil {
		params = map[string]any{}
	}
	if src, _ := params["source"].(string); strings.TrimSpace(src) == "" {
		params["source"] = SourceTag
		msg["params"] = params
		if out, err := json.Marshal(msg); err == nil {
			return out
		}
	}
	return raw
}
