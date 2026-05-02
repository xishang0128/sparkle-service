package auth

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/UruhaLushia/sparkle-service/route/httphelper"
)

const (
	authVersionV2       = "2"
	maxTimestampDriftV2 = 30 * time.Second
)

type nonceStore struct {
	mu    sync.Mutex
	ttl   time.Duration
	seen  map[string]time.Time
	order []nonceEntry
}

type nonceEntry struct {
	key       string
	expiresAt time.Time
}

func newNonceStore(ttl time.Duration) *nonceStore {
	return &nonceStore{
		ttl:   ttl,
		seen:  make(map[string]time.Time),
		order: make([]nonceEntry, 0),
	}
}

func (s *nonceStore) Remember(key string, now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.evictExpiredLocked(now)

	if expiresAt, exists := s.seen[key]; exists && expiresAt.After(now) {
		return false
	}

	expiresAt := now.Add(s.ttl)
	s.seen[key] = expiresAt
	s.order = append(s.order, nonceEntry{
		key:       key,
		expiresAt: expiresAt,
	})
	return true
}

func (s *nonceStore) evictExpiredLocked(now time.Time) {
	evicted := 0
	for evicted < len(s.order) {
		entry := s.order[evicted]
		if entry.expiresAt.After(now) {
			break
		}

		if currentExpiresAt, exists := s.seen[entry.key]; exists && currentExpiresAt.Equal(entry.expiresAt) {
			delete(s.seen, entry.key)
		}
		evicted++
	}

	if evicted > 0 {
		s.order = append([]nonceEntry(nil), s.order[evicted:]...)
	}
}

var requestNonceStore = newNonceStore(2 * maxTimestampDriftV2)

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		km := GetKeyManager()
		if !km.IsInitialized() || !km.HasAuthorizedPrincipal() {
			httphelper.SendError(w, httphelper.ServiceUnavailable("服务未初始化"))
			return
		}

		if err := km.VerifyRequestPrincipal(r); err != nil {
			httphelper.SendError(w, httphelper.Forbidden(fmt.Sprintf("请求方未授权: %v", err)))
			return
		}

		if r.Header.Get("X-Auth-Version") != authVersionV2 {
			httphelper.SendError(w, httphelper.Unauthorized("仅支持 Auth V2"))
			return
		}
		err := authenticateV2(r, km)
		if err != nil {
			httphelper.SendError(w, err)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func RequireAuth(next http.Handler) http.Handler {
	return AuthMiddleware(next)
}

func authenticateV2(r *http.Request, km *KeyManager) error {
	timestamp := r.Header.Get("X-Timestamp")
	keyID := r.Header.Get("X-Key-Id")
	nonce := r.Header.Get("X-Nonce")
	contentHash := strings.ToLower(r.Header.Get("X-Content-SHA256"))
	signature := r.Header.Get("X-Signature")

	if timestamp == "" || keyID == "" || nonce == "" || contentHash == "" || signature == "" {
		return httphelper.Unauthorized("缺少 V2 认证信息")
	}

	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return httphelper.BadRequest("无效的时间戳格式")
	}

	requestTime := time.UnixMilli(ts)
	now := time.Now()
	timeDiff := now.Sub(requestTime)

	if timeDiff < -maxTimestampDriftV2 || timeDiff > maxTimestampDriftV2 {
		return httphelper.Unauthorized("请求已过期或时间戳无效")
	}

	bodyHash, err := hashRequestBody(r)
	if err != nil {
		return err
	}
	if bodyHash != contentHash {
		return httphelper.Unauthorized("请求体摘要不匹配")
	}

	canonical, err := buildCanonicalRequestV2(r, timestamp, nonce, keyID, bodyHash)
	if err != nil {
		return httphelper.BadRequest(err.Error())
	}

	if err := km.VerifySignature(keyID, canonical, signature); err != nil {
		return httphelper.Unauthorized(err.Error())
	}

	nonceKey := keyID + ":" + timestamp + ":" + nonce
	if !requestNonceStore.Remember(nonceKey, now) {
		return httphelper.Conflict("请求已重放")
	}

	return nil
}

func hashRequestBody(r *http.Request) (string, error) {
	var body []byte

	if r.Body != nil {
		rawBody, err := io.ReadAll(r.Body)
		if err != nil {
			return "", fmt.Errorf("读取请求体失败： %w", err)
		}
		body = rawBody
		r.Body = io.NopCloser(bytes.NewReader(rawBody))
	}

	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:]), nil
}

func buildCanonicalRequestV2(r *http.Request, timestamp string, nonce string, keyID string, bodyHash string) (string, error) {
	query, err := canonicalizeQuery(r.URL.RawQuery)
	if err != nil {
		return "", fmt.Errorf("规范化请求参数失败： %w", err)
	}

	path := r.URL.EscapedPath()
	if path == "" {
		path = "/"
	}

	return strings.Join([]string{
		"SPARKLE-AUTH-V2",
		timestamp,
		nonce,
		keyID,
		strings.ToUpper(r.Method),
		path,
		query,
		bodyHash,
	}, "\n"), nil
}

func canonicalizeQuery(rawQuery string) (string, error) {
	if rawQuery == "" {
		return "", nil
	}

	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return "", err
	}

	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0)
	for _, key := range keys {
		vals := append([]string(nil), values[key]...)
		sort.Strings(vals)
		for _, value := range vals {
			parts = append(parts, url.QueryEscape(key)+"="+url.QueryEscape(value))
		}
	}

	return strings.Join(parts, "&"), nil
}
