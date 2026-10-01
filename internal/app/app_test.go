package app_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yeabt/lewe/internal/app"
	"github.com/yeabt/lewe/internal/db"
	"github.com/yeabt/lewe/internal/mail"
	"github.com/yeabt/lewe/internal/telegram"
)

// These end-to-end tests exercise the full HTTP API against a real
// PostgreSQL database. They are skipped unless TEST_DATABASE_URL is set, e.g.
//
//	TEST_DATABASE_URL=postgres://postgres@localhost:5432/lewe_test?sslmode=disable go test ./...
//
// The database is wiped before each test.

const testSecret = "test-secret-that-is-at-least-32-characters"

type env struct {
	t    *testing.T
	srv  *httptest.Server
	app  *app.App
	pool *pgxpool.Pool
}

func setup(t *testing.T) *env {
	t.Helper()
	return setupWith(t, app.Config{})
}

// setupWith is setup with extra app configuration (JWT secret and match
// interval are filled in).
func setupWith(t *testing.T, cfg app.Config) *env {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping end-to-end test")
	}
	if err := db.RunMigrations(url); err != nil {
		t.Fatalf("migrations: %v", err)
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)

	_, err = pool.Exec(context.Background(),
		`TRUNCATE users, refresh_tokens, account_tokens, items, wants, matches, ratings, messages, notifications CASCADE`)
	if err != nil {
		t.Fatalf("truncate: %v", err)
	}

	cfg.JWTSecret, cfg.MatchInterval = testSecret, time.Hour
	if cfg.UploadDir == "" {
		cfg.UploadDir = t.TempDir()
	}
	a, err := app.New(pool, cfg)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(a.Handler)
	t.Cleanup(srv.Close)
	return &env{t: t, srv: srv, app: a, pool: pool}
}

// match runs one matching pass and returns how many matches it created.
func (e *env) match() int64 {
	e.t.Helper()
	n, err := e.app.Worker.RunOnce(context.Background())
	if err != nil {
		e.t.Fatalf("matching pass: %v", err)
	}
	return n
}

type obj = map[string]any

// do sends a request and decodes the JSON response (if any) into a map.
func (e *env) do(method, path, token string, body any) (int, obj) {
	e.t.Helper()
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, e.srv.URL+path, rd)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	out := obj{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			e.t.Fatalf("%s %s: decode %q: %v", method, path, raw, err)
		}
	}
	return resp.StatusCode, out
}

// must asserts the status code and returns the body.
func (e *env) must(want int, method, path, token string, body any) obj {
	e.t.Helper()
	got, out := e.do(method, path, token, body)
	if got != want {
		e.t.Fatalf("%s %s: status %d, want %d; body %v", method, path, got, want, out)
	}
	return out
}

type user struct {
	id, token, refresh string
}

func (e *env) register(name string) user {
	e.t.Helper()
	out := e.must(201, "POST", "/api/v1/auth/register", "", obj{
		"email": strings.ToLower(name) + "@example.com", "password": "password123", "full_name": name,
	})
	return user{
		id:      out["user"].(obj)["id"].(string),
		token:   out["access_token"].(string),
		refresh: out["refresh_token"].(string),
	}
}

func (e *env) item(u user, title, category, condition string, value int) string {
	e.t.Helper()
	out := e.must(201, "POST", "/api/v1/items", u.token, obj{
		"title": title, "category": category, "condition": condition, "estimated_value": value,
	})
	return out["id"].(string)
}

func (e *env) want(u user, category string, keywords []string, minCondition any) string {
	e.t.Helper()
	out := e.must(201, "POST", "/api/v1/wants", u.token, obj{
		"category": category, "keywords": keywords, "min_condition": minCondition,
	})
	return out["id"].(string)
}

func (e *env) matches(u user, status string) []any {
	e.t.Helper()
	return e.must(200, "GET", "/api/v1/matches?status="+status, u.token, nil)["data"].([]any)
}

func (e *env) itemStatus(id string) string {
	e.t.Helper()
	var s string
	if err := e.pool.QueryRow(context.Background(), `SELECT status FROM items WHERE id = $1`, id).Scan(&s); err != nil {
		e.t.Fatalf("item status: %v", err)
	}
	return s
}

func TestAuthFlow(t *testing.T) {
	e := setup(t)

	alice := e.register("Alice")

	// Emails are case-insensitive.
	e.must(409, "POST", "/api/v1/auth/register", "", obj{
		"email": "ALICE@example.com", "password": "password123", "full_name": "Alice 2",
	})
	login := e.must(200, "POST", "/api/v1/auth/login", "", obj{"email": " Alice@Example.com ", "password": "password123"})
	e.must(401, "POST", "/api/v1/auth/login", "", obj{"email": "alice@example.com", "password": "wrong-password"})

	out := e.must(422, "POST", "/api/v1/auth/register", "", obj{"email": "nope", "password": "short"})
	fields := out["fields"].(obj)
	for _, f := range []string{"email", "password", "full_name"} {
		if _, ok := fields[f]; !ok {
			t.Errorf("expected validation error for %s, got %v", f, fields)
		}
	}
	e.must(400, "POST", "/api/v1/auth/login", "", "not an object")

	// Profile access requires a token.
	e.must(401, "GET", "/api/v1/users/me", "", nil)
	e.must(401, "GET", "/api/v1/users/me", "garbage", nil)
	me := e.must(200, "GET", "/api/v1/users/me", alice.token, nil)
	if me["email"] != "alice@example.com" {
		t.Errorf("email = %v", me["email"])
	}

	updated := e.must(200, "PUT", "/api/v1/users/me", alice.token, obj{"full_name": "Alice A.", "location": "Addis Ababa"})
	if updated["full_name"] != "Alice A." || updated["location"] != "Addis Ababa" {
		t.Errorf("update = %v", updated)
	}

	// Refresh rotates the token: the old one cannot be reused.
	refreshed := e.must(200, "POST", "/api/v1/auth/refresh", "", obj{"refresh_token": alice.refresh})
	e.must(401, "POST", "/api/v1/auth/refresh", "", obj{"refresh_token": alice.refresh})
	e.must(200, "POST", "/api/v1/auth/refresh", "", obj{"refresh_token": refreshed["refresh_token"]})

	// Logout revokes the refresh token and is idempotent.
	loginRefresh := login["refresh_token"].(string)
	e.must(204, "POST", "/api/v1/auth/logout", "", obj{"refresh_token": loginRefresh})
	e.must(204, "POST", "/api/v1/auth/logout", "", obj{"refresh_token": loginRefresh})
	e.must(401, "POST", "/api/v1/auth/refresh", "", obj{"refresh_token": loginRefresh})

	// Public profile hides the email.
	pub := e.must(200, "GET", "/api/v1/users/"+alice.id, "", nil)
	if _, ok := pub["email"]; ok {
		t.Error("public profile must not expose email")
	}
	e.must(404, "GET", "/api/v1/users/00000000-0000-0000-0000-000000000000", "", nil)
	e.must(404, "GET", "/api/v1/users/not-a-uuid", "", nil)
}

func TestItemsAndWants(t *testing.T) {
	e := setup(t)
	alice, bob := e.register("Alice"), e.register("Bob")

	cats := e.must(200, "GET", "/api/v1/categories", "", nil)
	if len(cats["categories"].([]any)) == 0 {
		t.Fatal("expected categories")
	}

	e.must(401, "POST", "/api/v1/items", "", obj{"title": "x"})
	out := e.must(422, "POST", "/api/v1/items", alice.token, obj{
		"title": "", "category": "spaceships", "condition": "mint", "image_urls": []string{"ftp://x"},
	})
	for _, f := range []string{"title", "category", "condition", "image_urls"} {
		if _, ok := out["fields"].(obj)[f]; !ok {
			t.Errorf("expected validation error for %s, got %v", f, out["fields"])
		}
	}

	camera := e.must(201, "POST", "/api/v1/items", alice.token, obj{
		"title": "Canon Camera", "description": "DSLR with 50mm lens", "category": "Electronics",
		"condition": "good", "estimated_value": 300, "image_urls": []string{"https://img.example.com/1.jpg"},
	})
	if camera["category"] != "electronics" || camera["status"] != "available" || camera["owner_id"] != alice.id {
		t.Errorf("created item = %v", camera)
	}
	cameraID := camera["id"].(string)
	e.item(alice, "Old paperback", "books", "fair", 5)
	e.item(bob, "Acoustic guitar", "music", "like_new", 250)

	// Public browsing with filters and pagination.
	if got := len(e.must(200, "GET", "/api/v1/items", "", nil)["data"].([]any)); got != 3 {
		t.Errorf("browse all = %d, want 3", got)
	}
	if got := len(e.must(200, "GET", "/api/v1/items?category=music", "", nil)["data"].([]any)); got != 1 {
		t.Errorf("browse music = %d, want 1", got)
	}
	if got := len(e.must(200, "GET", "/api/v1/items?q=dslr", "", nil)["data"].([]any)); got != 1 {
		t.Errorf("search dslr = %d, want 1", got)
	}
	if got := len(e.must(200, "GET", "/api/v1/items?q=%25", "", nil)["data"].([]any)); got != 0 {
		t.Errorf("wildcard search should be literal, got %d results", got)
	}
	if got := len(e.must(200, "GET", "/api/v1/items?owner_id="+alice.id+"&limit=1", "", nil)["data"].([]any)); got != 1 {
		t.Errorf("limit=1 returned %d", got)
	}

	// Only the owner can edit or withdraw.
	edit := obj{"title": "Canon EOS Camera", "category": "electronics", "condition": "good", "estimated_value": 280}
	e.must(403, "PUT", "/api/v1/items/"+cameraID, bob.token, edit)
	e.must(403, "DELETE", "/api/v1/items/"+cameraID, bob.token, nil)
	if got := e.must(200, "PUT", "/api/v1/items/"+cameraID, alice.token, edit); got["title"] != "Canon EOS Camera" {
		t.Errorf("edited title = %v", got["title"])
	}
	e.must(404, "PUT", "/api/v1/items/00000000-0000-0000-0000-000000000000", alice.token, edit)

	// Withdrawn items disappear from browsing and are only visible to the owner.
	e.must(204, "DELETE", "/api/v1/items/"+cameraID, alice.token, nil)
	e.must(404, "GET", "/api/v1/items/"+cameraID, "", nil)
	e.must(404, "GET", "/api/v1/items/"+cameraID, bob.token, nil)
	e.must(200, "GET", "/api/v1/items/"+cameraID, alice.token, nil)
	e.must(409, "DELETE", "/api/v1/items/"+cameraID, alice.token, nil)
	if got := len(e.must(200, "GET", "/api/v1/users/me/items?status=withdrawn", alice.token, nil)["data"].([]any)); got != 1 {
		t.Errorf("my withdrawn items = %d, want 1", got)
	}
	e.must(422, "GET", "/api/v1/users/me/items?status=bogus", alice.token, nil)

	// Wants are private to their owner.
	e.must(422, "POST", "/api/v1/wants", alice.token, obj{"category": "nope", "min_condition": "mint"})
	w := e.must(201, "POST", "/api/v1/wants", alice.token, obj{
		"category": "music", "keywords": []string{" Guitar ", "guitar", "50%_off"},
	})
	if kw := w["keywords"].([]any); len(kw) != 2 || kw[0] != "guitar" || kw[1] != "50off" {
		t.Errorf("keywords not normalised: %v", kw)
	}
	wantID := w["id"].(string)
	e.must(404, "GET", "/api/v1/wants/"+wantID, bob.token, nil)
	e.must(200, "PUT", "/api/v1/wants/"+wantID, alice.token, obj{"category": "music", "min_condition": "good"})
	if got := len(e.must(200, "GET", "/api/v1/wants", alice.token, nil)["data"].([]any)); got != 1 {
		t.Errorf("wants = %d, want 1", got)
	}
	if got := len(e.must(200, "GET", "/api/v1/wants", bob.token, nil)["data"].([]any)); got != 0 {
		t.Errorf("bob sees %d wants, want 0", got)
	}
	e.must(204, "DELETE", "/api/v1/wants/"+wantID, alice.token, nil)
	e.must(409, "PUT", "/api/v1/wants/"+wantID, alice.token, obj{"category": "music"})
	e.must(409, "DELETE", "/api/v1/wants/"+wantID, alice.token, nil)
}

func TestExchangeLifecycle(t *testing.T) {
	e := setup(t)
	alice, bob, carol := e.register("Alice"), e.register("Bob"), e.register("Carol")

	camera := e.item(alice, "Canon Camera", "electronics", "good", 300)
	guitar := e.item(bob, "Acoustic Guitar", "music", "like_new", 250)
	amp := e.item(carol, "Guitar amplifier", "music", "fair", 100)
	e.item(carol, "Broken camera", "electronics", "poor", 10) // too worn for Bob's want

	aliceWant := e.want(alice, "music", []string{"guitar"}, nil)
	bobWant := e.want(bob, "electronics", []string{"camera"}, "good")
	e.want(carol, "electronics", []string{"camera"}, nil)

	// Alice ↔ Bob (camera ↔ guitar) and Alice ↔ Carol (camera ↔ amp).
	// Bob ↔ Carol does not match: Carol's camera is below Bob's minimum
	// condition, and Carol wants electronics, which Bob doesn't have.
	if n := e.match(); n != 2 {
		t.Fatalf("first pass created %d matches, want 2", n)
	}
	if n := e.match(); n != 0 {
		t.Fatalf("second pass created %d matches, want 0 (no duplicates)", n)
	}

	if got := len(e.matches(alice, "pending")); got != 2 {
		t.Fatalf("alice pending = %d, want 2", got)
	}
	bobMatches := e.matches(bob, "")
	if len(bobMatches) != 1 {
		t.Fatalf("bob matches = %d, want 1", len(bobMatches))
	}
	m := bobMatches[0].(obj)
	matchID := m["id"].(string)
	if m["your_item"].(obj)["id"] != guitar || m["their_item"].(obj)["id"] != camera ||
		m["other_user"].(obj)["full_name"] != "Alice" {
		t.Errorf("bob's view of match is wrong: %v", m)
	}
	if score := m["score"].(float64); score < 0.8 || score > 0.9 {
		t.Errorf("score = %v, want ≈ 0.83 (250 vs 300)", score)
	}

	// Outsiders can't see or act on the match.
	e.must(404, "GET", "/api/v1/matches/"+matchID, carol.token, nil)
	e.must(404, "POST", "/api/v1/matches/"+matchID+"/accept", carol.token, nil)
	e.must(401, "GET", "/api/v1/matches", "", nil)

	// Nothing but accept/decline is allowed while pending.
	path := "/api/v1/matches/" + matchID
	e.must(409, "POST", path+"/complete", alice.token, nil)
	e.must(409, "PUT", path+"/exchange", alice.token, obj{"method": "meetup"})
	e.must(409, "POST", path+"/cancel", alice.token, nil)
	e.must(409, "POST", path+"/rating", alice.token, obj{"score": 5})

	// Both must accept.
	got := e.must(200, "POST", path+"/accept", alice.token, nil)
	if got["status"] != "pending" || got["you_accepted"] != true || got["they_accepted"] != false {
		t.Errorf("after alice accepts: %v", got)
	}
	e.must(200, "POST", path+"/accept", alice.token, nil) // idempotent
	got = e.must(200, "POST", path+"/accept", bob.token, nil)
	if got["status"] != "accepted" {
		t.Fatalf("after both accept, status = %v", got["status"])
	}

	// Items are reserved, and the competing match for the camera is cancelled.
	if s := e.itemStatus(camera); s != "reserved" {
		t.Errorf("camera status = %s", s)
	}
	if s := e.itemStatus(amp); s != "available" {
		t.Errorf("amp status = %s", s)
	}
	if got := e.matches(carol, ""); len(got) != 1 || got[0].(obj)["status"] != "cancelled" {
		t.Errorf("carol's match should be cancelled: %v", got)
	}
	e.must(409, "PUT", "/api/v1/items/"+camera, alice.token, obj{"title": "x", "category": "electronics", "condition": "good"})
	e.must(409, "DELETE", "/api/v1/items/"+camera, alice.token, nil)
	e.must(409, "POST", path+"/decline", bob.token, nil)

	// Reserved items don't show up in the marketplace or create new matches.
	for _, it := range e.must(200, "GET", "/api/v1/items", "", nil)["data"].([]any) {
		if id := it.(obj)["id"]; id == camera || id == guitar {
			t.Errorf("reserved item %v is listed", id)
		}
	}
	if n := e.match(); n != 0 {
		t.Errorf("matching reserved items created %d matches", n)
	}

	// An exchange method is needed before completing.
	e.must(409, "POST", path+"/complete", alice.token, nil)
	e.must(422, "PUT", path+"/exchange", alice.token, obj{"method": "teleport"})
	got = e.must(200, "PUT", path+"/exchange", bob.token, obj{"method": "meetup", "details": "Cafe, Saturday 10am"})
	if ex := got["exchange"].(obj); ex["method"] != "meetup" || ex["proposed_by_you"] != true {
		t.Errorf("exchange = %v", ex)
	}

	got = e.must(200, "POST", path+"/complete", alice.token, nil)
	if got["status"] != "accepted" || got["you_completed"] != true {
		t.Errorf("after alice completes: %v", got)
	}
	// Changing the plan resets confirmations.
	got = e.must(200, "PUT", path+"/exchange", bob.token, obj{"method": "dropoff"})
	if got["they_completed"] != false {
		t.Errorf("completion should reset when the exchange changes: %v", got)
	}
	e.must(200, "POST", path+"/complete", alice.token, nil)
	got = e.must(200, "POST", path+"/complete", bob.token, nil)
	if got["status"] != "completed" || got["completed_at"] == nil {
		t.Fatalf("after both complete: %v", got)
	}
	if e.itemStatus(camera) != "exchanged" || e.itemStatus(guitar) != "exchanged" {
		t.Error("items should be exchanged")
	}
	for _, w := range []struct {
		u  user
		id string
	}{{alice, aliceWant}, {bob, bobWant}} {
		if s := e.must(200, "GET", "/api/v1/wants/"+w.id, w.u.token, nil)["status"]; s != "fulfilled" {
			t.Errorf("want %s status = %v, want fulfilled", w.id, s)
		}
	}
	e.must(409, "POST", path+"/cancel", alice.token, nil)

	// Ratings: once per participant, only participants.
	e.must(422, "POST", path+"/rating", alice.token, obj{"score": 6})
	e.must(404, "POST", path+"/rating", carol.token, obj{"score": 1})
	r := e.must(201, "POST", path+"/rating", alice.token, obj{"score": 5, "comment": "Great swap!"})
	if r["rater"].(obj)["id"] != alice.id {
		t.Errorf("rating = %v", r)
	}
	e.must(409, "POST", path+"/rating", alice.token, obj{"score": 4})
	e.must(201, "POST", path+"/rating", bob.token, obj{"score": 4})

	if got := e.must(200, "GET", path, alice.token, nil); got["you_rated"] != true {
		t.Error("you_rated should be true")
	}
	profile := e.must(200, "GET", "/api/v1/users/"+bob.id, "", nil)
	if rating := profile["rating"].(obj); rating["average"] != 5.0 || rating["count"] != 1.0 {
		t.Errorf("bob's rating = %v", rating)
	}
	ratings := e.must(200, "GET", "/api/v1/users/"+alice.id+"/ratings", "", nil)["data"].([]any)
	if len(ratings) != 1 || ratings[0].(obj)["score"] != 4.0 {
		t.Errorf("alice's ratings = %v", ratings)
	}
}

func TestDeclineCancelAndWithdraw(t *testing.T) {
	e := setup(t)
	alice, bob := e.register("Alice"), e.register("Bob")

	book := e.item(alice, "Go Programming book", "books", "good", 0)
	lamp := e.item(bob, "Desk lamp", "home", "new", 0)
	e.want(alice, "home", nil, nil)
	e.want(bob, "books", nil, nil)

	if n := e.match(); n != 1 {
		t.Fatalf("created %d matches, want 1", n)
	}
	matchID := e.matches(alice, "")[0].(obj)["id"].(string)
	path := "/api/v1/matches/" + matchID
	if score := e.matches(alice, "")[0].(obj)["score"]; score != 1.0 {
		t.Errorf("equal values should score 1, got %v", score)
	}

	// Accept, then cancel: items return to the marketplace.
	e.must(200, "POST", path+"/accept", alice.token, nil)
	e.must(200, "POST", path+"/accept", bob.token, nil)
	if got := e.must(200, "POST", path+"/cancel", bob.token, nil); got["status"] != "cancelled" {
		t.Errorf("status = %v", got["status"])
	}
	if e.itemStatus(book) != "available" || e.itemStatus(lamp) != "available" {
		t.Error("items should be available after cancel")
	}
	// A pair is only ever matched once.
	if n := e.match(); n != 0 {
		t.Errorf("rematched a cancelled pair: %d", n)
	}

	// Decline a fresh match.
	e.item(bob, "Floor lamp", "home", "good", 0)
	if n := e.match(); n != 1 {
		t.Fatalf("created %d matches, want 1", n)
	}
	pending := e.matches(bob, "pending")
	if len(pending) != 1 {
		t.Fatalf("pending = %d", len(pending))
	}
	path2 := "/api/v1/matches/" + pending[0].(obj)["id"].(string)
	if got := e.must(200, "POST", path2+"/decline", bob.token, nil); got["status"] != "declined" {
		t.Errorf("status = %v", got["status"])
	}
	e.must(409, "POST", path2+"/accept", alice.token, nil)

	// Withdrawing an item cancels its pending matches.
	e.item(bob, "Table lamp", "home", "good", 0)
	if n := e.match(); n != 1 {
		t.Fatalf("created %d matches, want 1", n)
	}
	e.must(204, "DELETE", "/api/v1/items/"+book, alice.token, nil)
	if got := e.matches(alice, "pending"); len(got) != 0 {
		t.Errorf("pending after withdraw = %d, want 0", len(got))
	}

	e.must(422, "GET", "/api/v1/matches?status=weird", alice.token, nil)
	e.must(404, "GET", "/api/v1/matches/not-a-uuid", alice.token, nil)
}

func TestAcceptWhenItemGone(t *testing.T) {
	e := setup(t)
	alice, bob := e.register("Alice"), e.register("Bob")

	e.item(alice, "Bike", "sports", "good", 100)
	bobItem := e.item(bob, "Skateboard", "sports", "good", 80)
	e.want(alice, "sports", []string{"skate"}, nil)
	e.want(bob, "sports", []string{"bike"}, nil)
	if n := e.match(); n != 1 {
		t.Fatalf("created %d matches, want 1", n)
	}
	path := "/api/v1/matches/" + e.matches(alice, "")[0].(obj)["id"].(string)

	e.must(200, "POST", path+"/accept", alice.token, nil)
	// Simulate the item leaving the marketplace outside the match flow.
	if _, err := e.pool.Exec(context.Background(), `UPDATE items SET status = 'exchanged' WHERE id = $1`, bobItem); err != nil {
		t.Fatal(err)
	}
	e.must(409, "POST", path+"/accept", bob.token, nil)
	if got := e.must(200, "GET", path, bob.token, nil); got["status"] != "cancelled" {
		t.Errorf("match should be cancelled, got %v", got["status"])
	}
}

func TestHealth(t *testing.T) {
	e := setup(t)
	if out := e.must(200, "GET", "/health", "", nil); out["status"] != "ok" {
		t.Errorf("health = %v", out)
	}
}

// notificationTypes returns the user's notification types, newest first.
func (e *env) notificationTypes(u user, query string) []string {
	e.t.Helper()
	out := []string{}
	for _, n := range e.must(200, "GET", "/api/v1/notifications"+query, u.token, nil)["data"].([]any) {
		out = append(out, n.(obj)["type"].(string))
	}
	return out
}

func (e *env) unread(u user) float64 {
	e.t.Helper()
	return e.must(200, "GET", "/api/v1/notifications/unread-count", u.token, nil)["unread"].(float64)
}

func TestMessagesAndNotifications(t *testing.T) {
	e := setup(t)
	alice, bob, carol := e.register("Alice"), e.register("Bob"), e.register("Carol")

	e.item(alice, "Road bike", "sports", "good", 200)
	e.item(bob, "Camping tent", "sports", "good", 180)
	e.item(carol, "Tent pegs", "sports", "good", 5)
	e.want(alice, "sports", []string{"tent"}, nil)
	e.want(bob, "sports", []string{"bike"}, nil)
	e.want(carol, "sports", []string{"bike"}, nil)
	if n := e.match(); n != 2 {
		t.Fatalf("created %d matches, want 2", n)
	}

	// Everyone involved hears about their new matches.
	if got := e.notificationTypes(alice, ""); len(got) != 2 || got[0] != "match_found" {
		t.Errorf("alice notifications = %v", got)
	}
	if got := e.unread(bob); got != 1 {
		t.Errorf("bob unread = %v, want 1", got)
	}

	var matchID string
	for _, m := range e.matches(bob, "") {
		matchID = m.(obj)["id"].(string)
	}
	path := "/api/v1/matches/" + matchID

	// Chat between the two participants only.
	e.must(422, "POST", path+"/messages", alice.token, obj{"body": "   "})
	e.must(422, "POST", path+"/messages", alice.token, obj{"body": strings.Repeat("x", 2001)})
	e.must(404, "POST", path+"/messages", carol.token, obj{"body": "hi"})
	e.must(404, "GET", path+"/messages", carol.token, nil)
	msg := e.must(201, "POST", path+"/messages", alice.token, obj{"body": " Is the tent waterproof? "})
	if msg["body"] != "Is the tent waterproof?" || msg["from_you"] != true {
		t.Errorf("message = %v", msg)
	}
	e.must(201, "POST", path+"/messages", bob.token, obj{"body": "Yes, 3000mm."})

	msgs := e.must(200, "GET", path+"/messages", bob.token, nil)["data"].([]any)
	if len(msgs) != 2 || msgs[0].(obj)["from_you"] != false || msgs[1].(obj)["body"] != "Yes, 3000mm." {
		t.Errorf("bob's view of messages = %v", msgs)
	}
	if got := e.notificationTypes(bob, "?unread=true"); len(got) != 2 || got[0] != "message_received" {
		t.Errorf("bob unread notifications = %v", got)
	}

	// Each step of the lifecycle notifies the other participant.
	e.must(200, "POST", path+"/accept", alice.token, nil)
	if got := e.notificationTypes(bob, "")[0]; got != "match_accepted" {
		t.Errorf("bob latest = %s, want match_accepted", got)
	}
	e.must(200, "POST", path+"/accept", bob.token, nil)
	if got := e.notificationTypes(alice, "")[0]; got != "match_confirmed" {
		t.Errorf("alice latest = %s, want match_confirmed", got)
	}
	// Carol's competing match for Alice's bike was cancelled; she is told.
	if got := e.notificationTypes(carol, "")[0]; got != "match_cancelled" {
		t.Errorf("carol latest = %s, want match_cancelled", got)
	}
	e.must(200, "PUT", path+"/exchange", alice.token, obj{"method": "meetup"})
	if got := e.notificationTypes(bob, "")[0]; got != "exchange_updated" {
		t.Errorf("bob latest = %s, want exchange_updated", got)
	}
	e.must(200, "POST", path+"/complete", alice.token, nil)
	e.must(200, "POST", path+"/complete", bob.token, nil)
	if got := e.notificationTypes(alice, "")[0]; got != "match_completed" {
		t.Errorf("alice latest = %s, want match_completed", got)
	}
	e.must(201, "POST", path+"/rating", alice.token, obj{"score": 5})
	bobLatest := e.must(200, "GET", "/api/v1/notifications?limit=1", bob.token, nil)["data"].([]any)[0].(obj)
	if bobLatest["type"] != "rating_received" || bobLatest["match_id"] != matchID || bobLatest["message"] == "" {
		t.Errorf("bob latest = %v", bobLatest)
	}

	// Chat stays open after completion but closes on cancelled matches.
	e.must(201, "POST", path+"/messages", bob.token, obj{"body": "Thanks!"})
	var carolMatch string
	for _, m := range e.matches(carol, "") {
		carolMatch = m.(obj)["id"].(string)
	}
	e.must(409, "POST", "/api/v1/matches/"+carolMatch+"/messages", carol.token, obj{"body": "hello?"})
	e.must(200, "GET", "/api/v1/matches/"+carolMatch+"/messages", carol.token, nil)

	// Marking notifications read.
	n := e.must(200, "GET", "/api/v1/notifications?unread=true", alice.token, nil)["data"].([]any)
	before := e.unread(alice)
	e.must(204, "POST", "/api/v1/notifications/"+n[0].(obj)["id"].(string)+"/read", alice.token, nil)
	if got := e.unread(alice); got != before-1 {
		t.Errorf("unread after marking one = %v, want %v", got, before-1)
	}
	e.must(404, "POST", "/api/v1/notifications/"+n[0].(obj)["id"].(string)+"/read", bob.token, nil)
	e.must(404, "POST", "/api/v1/notifications/nope/read", alice.token, nil)
	e.must(204, "POST", "/api/v1/notifications/read-all", alice.token, nil)
	if got := e.unread(alice); got != 0 {
		t.Errorf("unread after read-all = %v", got)
	}
	if got := len(e.notificationTypes(alice, "?unread=true")); got != 0 {
		t.Errorf("unread list = %d", got)
	}
	e.must(401, "GET", "/api/v1/notifications", "", nil)
}

func TestAuthRateLimit(t *testing.T) {
	e := setupWith(t, app.Config{AuthRateLimit: 3})
	body := obj{"email": "nobody@example.com", "password": "password123"}
	for i := 0; i < 3; i++ {
		e.must(401, "POST", "/api/v1/auth/login", "", body)
	}
	e.must(429, "POST", "/api/v1/auth/login", "", body)
	e.must(429, "POST", "/api/v1/auth/register", "", body)
	// Other routes are not limited.
	e.must(200, "GET", "/api/v1/items", "", nil)
}

func TestCORS(t *testing.T) {
	e := setupWith(t, app.Config{CORSOrigins: []string{"https://app.example.com"}})
	req, _ := http.NewRequest("OPTIONS", e.srv.URL+"/api/v1/items", nil)
	req.Header.Set("Origin", "https://app.example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 204 || resp.Header.Get("Access-Control-Allow-Origin") != "https://app.example.com" {
		t.Errorf("preflight: %d %v", resp.StatusCode, resp.Header)
	}
}

var tokenInLink = regexp.MustCompile(`https://app\.test/([a-z-]+)\?token=([0-9a-f]{64})`)

// lastLink returns the page and token of the newest email sent to addr.
func lastLink(t *testing.T, m *mail.MemoryMailer, addr string) (page, token string) {
	t.Helper()
	sent := m.Sent()
	for i := len(sent) - 1; i >= 0; i-- {
		if sent[i].To == addr {
			match := tokenInLink.FindStringSubmatch(sent[i].Body)
			if match == nil {
				t.Fatalf("no link in email: %q", sent[i].Body)
			}
			return match[1], match[2]
		}
	}
	t.Fatalf("no email sent to %s", addr)
	return "", ""
}

func TestEmailVerificationAndPasswordReset(t *testing.T) {
	mailer := &mail.MemoryMailer{}
	e := setupWith(t, app.Config{Mailer: mailer, AppURL: "https://app.test/"})

	// Signing up sends a verification link.
	out := e.must(201, "POST", "/api/v1/auth/register", "", obj{
		"email": "alice@example.com", "password": "password123", "full_name": "Alice",
	})
	alice := user{id: out["user"].(obj)["id"].(string), token: out["access_token"].(string), refresh: out["refresh_token"].(string)}
	if out["user"].(obj)["email_verified"] != false {
		t.Errorf("new user should be unverified: %v", out["user"])
	}
	if sent := mailer.Sent(); len(sent) != 1 || sent[0].Subject != "Confirm your email for Lewe" {
		t.Fatalf("sent = %+v", sent)
	}
	page, verifyToken := lastLink(t, mailer, "alice@example.com")
	if page != "verify-email" {
		t.Errorf("link page = %s", page)
	}

	e.must(400, "POST", "/api/v1/auth/verify-email", "", obj{"token": strings.Repeat("0", 64)})
	e.must(400, "POST", "/api/v1/auth/verify-email", "", obj{"token": ""})
	e.must(204, "POST", "/api/v1/auth/verify-email", "", obj{"token": verifyToken})
	e.must(400, "POST", "/api/v1/auth/verify-email", "", obj{"token": verifyToken}) // single use
	if me := e.must(200, "GET", "/api/v1/users/me", alice.token, nil); me["email_verified"] != true {
		t.Errorf("after verifying: %v", me)
	}
	e.must(409, "POST", "/api/v1/users/me/verify-email", alice.token, nil)

	// Resending replaces the earlier link.
	bob := e.register("Bob")
	_, firstBob := lastLink(t, mailer, "bob@example.com")
	e.must(202, "POST", "/api/v1/users/me/verify-email", bob.token, nil)
	_, secondBob := lastLink(t, mailer, "bob@example.com")
	if firstBob == secondBob {
		t.Fatal("resend should issue a new token")
	}
	e.must(400, "POST", "/api/v1/auth/verify-email", "", obj{"token": firstBob})
	e.must(204, "POST", "/api/v1/auth/verify-email", "", obj{"token": secondBob})

	// Forgot password never reveals whether an account exists.
	before := len(mailer.Sent())
	unknown := e.must(202, "POST", "/api/v1/auth/forgot-password", "", obj{"email": "nobody@example.com"})
	known := e.must(202, "POST", "/api/v1/auth/forgot-password", "", obj{"email": " ALICE@example.com "})
	if unknown["message"] != known["message"] {
		t.Errorf("responses differ: %v vs %v", unknown, known)
	}
	if got := len(mailer.Sent()) - before; got != 1 {
		t.Fatalf("expected exactly one reset email, got %d", got)
	}
	e.must(422, "POST", "/api/v1/auth/forgot-password", "", obj{"email": "not-an-email"})

	page, resetToken := lastLink(t, mailer, "alice@example.com")
	if page != "reset-password" {
		t.Errorf("link page = %s", page)
	}
	e.must(422, "POST", "/api/v1/auth/reset-password", "", obj{"token": resetToken, "password": "short"})
	e.must(204, "POST", "/api/v1/auth/reset-password", "", obj{"token": resetToken, "password": "new-password-1"})
	e.must(400, "POST", "/api/v1/auth/reset-password", "", obj{"token": resetToken, "password": "new-password-2"})

	// The reset signed Alice out everywhere and replaced the password.
	e.must(401, "POST", "/api/v1/auth/refresh", "", obj{"refresh_token": alice.refresh})
	e.must(401, "POST", "/api/v1/auth/login", "", obj{"email": "alice@example.com", "password": "password123"})
	login := e.must(200, "POST", "/api/v1/auth/login", "", obj{"email": "alice@example.com", "password": "new-password-1"})
	token := login["access_token"].(string)

	// Changing a known password: wrong current password is a field error.
	out = e.must(422, "PUT", "/api/v1/users/me/password", token, obj{"current_password": "nope", "new_password": "another-pass-3"})
	if out["fields"].(obj)["current_password"] != "is incorrect" {
		t.Errorf("fields = %v", out["fields"])
	}
	e.must(401, "PUT", "/api/v1/users/me/password", "", obj{"current_password": "x", "new_password": "y"})
	changed := e.must(200, "PUT", "/api/v1/users/me/password", token, obj{"current_password": "new-password-1", "new_password": "another-pass-3"})
	e.must(401, "POST", "/api/v1/auth/refresh", "", obj{"refresh_token": login["refresh_token"]})
	e.must(200, "POST", "/api/v1/auth/refresh", "", obj{"refresh_token": changed["refresh_token"]})
	e.must(200, "POST", "/api/v1/auth/login", "", obj{"email": "alice@example.com", "password": "another-pass-3"})

	// Expired links are rejected.
	e.must(202, "POST", "/api/v1/auth/forgot-password", "", obj{"email": "bob@example.com"})
	_, bobReset := lastLink(t, mailer, "bob@example.com")
	if _, err := e.pool.Exec(context.Background(), `UPDATE account_tokens SET expires_at = now() - interval '1 minute'`); err != nil {
		t.Fatal(err)
	}
	e.must(400, "POST", "/api/v1/auth/reset-password", "", obj{"token": bobReset, "password": "whatever-123"})
}

const testBotToken = "123456:TEST-bot-token"

// telegramInitData builds launch data signed the way Telegram signs it.
func telegramInitData(id int64, first, username string) string {
	v := url.Values{}
	v.Set("auth_date", strconv.FormatInt(time.Now().Unix(), 10))
	v.Set("user", fmt.Sprintf(`{"id":%d,"first_name":%q,"username":%q}`, id, first, username))
	v.Set("hash", telegram.Sign(v, testBotToken))
	return v.Encode()
}

type sentMessage struct {
	chatID int64
	text   string
	button *telegram.WebAppButton
}

// fakeSender records Telegram messages; err makes every send fail.
type fakeSender struct {
	mu   sync.Mutex
	sent []sentMessage
	err  error
}

func (f *fakeSender) SendMessage(_ context.Context, chatID int64, text string, b *telegram.WebAppButton) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	f.sent = append(f.sent, sentMessage{chatID, text, b})
	return nil
}

func (f *fakeSender) messages() []sentMessage {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]sentMessage(nil), f.sent...)
}

func (e *env) telegramLogin(id int64, first, username string) user {
	e.t.Helper()
	out := e.must(200, "POST", "/api/v1/auth/telegram", "", obj{"init_data": telegramInitData(id, first, username)})
	return user{id: out["user"].(obj)["id"].(string), token: out["access_token"].(string), refresh: out["refresh_token"].(string)}
}

func TestTelegramLogin(t *testing.T) {
	// Disabled without a bot token.
	plain := setup(t)
	plain.must(404, "POST", "/api/v1/auth/telegram", "", obj{"init_data": telegramInitData(1, "A", "a")})

	e := setupWith(t, app.Config{TelegramBotToken: testBotToken})
	out := e.must(200, "POST", "/api/v1/auth/telegram", "", obj{"init_data": telegramInitData(777, "Selam", "selamt")})
	u := out["user"].(obj)
	if u["full_name"] != "Selam" || u["telegram_username"] != "selamt" {
		t.Errorf("user = %v", u)
	}
	if _, hasEmail := u["email"]; hasEmail {
		t.Errorf("telegram user should have no email: %v", u)
	}

	// Signing in again finds the same account and refreshes the username.
	again := e.must(200, "POST", "/api/v1/auth/telegram", "", obj{"init_data": telegramInitData(777, "Selam T", "selam_new")})
	if again["user"].(obj)["id"] != u["id"] || again["user"].(obj)["telegram_username"] != "selam_new" {
		t.Errorf("second sign-in = %v", again["user"])
	}
	token := again["access_token"].(string)
	e.must(200, "GET", "/api/v1/users/me", token, nil)

	// Forged or stale data is rejected.
	forged := strings.Replace(telegramInitData(777, "Selam", "selamt"), "777", "778", 1)
	e.must(401, "POST", "/api/v1/auth/telegram", "", obj{"init_data": forged})
	e.must(401, "POST", "/api/v1/auth/telegram", "", obj{"init_data": ""})
	stale := url.Values{}
	stale.Set("auth_date", strconv.FormatInt(time.Now().Add(-48*time.Hour).Unix(), 10))
	stale.Set("user", `{"id":777,"first_name":"Selam"}`)
	stale.Set("hash", telegram.Sign(stale, testBotToken))
	e.must(401, "POST", "/api/v1/auth/telegram", "", obj{"init_data": stale.Encode()})

	// Password and email features don't apply to Telegram-only accounts.
	e.must(409, "PUT", "/api/v1/users/me/password", token, obj{"current_password": "whatever1", "new_password": "whatever2"})
	e.must(409, "POST", "/api/v1/users/me/verify-email", token, nil)
}

func TestTelegramNotificationDelivery(t *testing.T) {
	sender := &fakeSender{}
	e := setupWith(t, app.Config{TelegramBotToken: testBotToken, TelegramSender: sender, AppURL: "https://app.test/"})
	if e.app.Dispatcher == nil {
		t.Fatal("dispatcher should be configured")
	}

	selam := e.telegramLogin(101, "Selam", "selam")
	dawit := e.telegramLogin(202, "Dawit", "dawit")
	e.register("Mail") // an email-only user: nothing to send them

	e.item(selam, "Canon camera", "electronics", "good", 300)
	e.item(dawit, "Krar", "music", "good", 250)
	e.want(selam, "music", nil, nil)
	e.want(dawit, "electronics", nil, nil)
	if n := e.match(); n != 1 {
		t.Fatalf("matches = %d", n)
	}
	matchID := e.matches(selam, "")[0].(obj)["id"].(string)

	n, err := e.app.Dispatcher.RunOnce(context.Background())
	if err != nil || n != 2 {
		t.Fatalf("RunOnce = %d, %v; want 2 sent", n, err)
	}
	got := sender.messages()
	chats := map[int64]bool{}
	for _, m := range got {
		chats[m.chatID] = true
		if !strings.Contains(m.text, "found a new swap") || m.button == nil ||
			m.button.URL != "https://app.test/#/matches/"+matchID {
			t.Errorf("message = %+v", m)
		}
	}
	if !chats[101] || !chats[202] {
		t.Errorf("chats = %v", chats)
	}
	if n, _ := e.app.Dispatcher.RunOnce(context.Background()); n != 0 {
		t.Errorf("second pass sent %d, want 0", n)
	}

	// A failing Telegram keeps notifications pending until it recovers.
	sender.err = fmt.Errorf("telegram is down")
	e.must(201, "POST", "/api/v1/matches/"+matchID+"/messages", selam.token, obj{"body": "Hi Dawit"})
	if n, _ := e.app.Dispatcher.RunOnce(context.Background()); n != 0 {
		t.Errorf("sent %d while failing", n)
	}
	sender.err = nil
	if n, _ := e.app.Dispatcher.RunOnce(context.Background()); n != 1 {
		t.Errorf("after recovery sent %d, want 1", n)
	}
	if last := sender.messages()[len(sender.messages())-1]; last.chatID != 202 || !strings.Contains(last.text, "new message") {
		t.Errorf("last = %+v", last)
	}

	// Users who blocked the bot don't block the queue.
	sender.err = telegram.ErrBlocked
	e.must(201, "POST", "/api/v1/matches/"+matchID+"/messages", dawit.token, obj{"body": "Hello"})
	e.app.Dispatcher.RunOnce(context.Background())
	var pending int
	e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM notifications WHERE delivered_at IS NULL`).Scan(&pending)
	if pending != 0 {
		t.Errorf("pending = %d, want 0", pending)
	}
}

func pngImage() []byte {
	// A valid 1x1 PNG.
	return []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15\xc4\x89\x00\x00\x00\rIDATx\x9cc\xf8\xcf\xc0\xf0\x1f\x00\x05\x00\x01\xff\x89\x99=\x1d\x00\x00\x00\x00IEND\xaeB`\x82")
}

func TestUploadsAndCityMatching(t *testing.T) {
	e := setup(t)
	alice, bob := e.register("Alice"), e.register("Bob")

	// Upload a photo and use it on a listing.
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("file", "photo.png")
	fw.Write(pngImage())
	mw.Close()
	req, _ := http.NewRequest("POST", e.srv.URL+"/api/v1/uploads", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+alice.token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	var up obj
	json.NewDecoder(resp.Body).Decode(&up)
	resp.Body.Close()
	if resp.StatusCode != 201 {
		t.Fatalf("upload: %d %v", resp.StatusCode, up)
	}
	photo := up["url"].(string)

	item := e.must(201, "POST", "/api/v1/items", alice.token, obj{
		"title": "Bike", "category": "sports", "condition": "good", "image_urls": []string{photo},
	})
	if item["image_urls"].([]any)[0] != photo {
		t.Errorf("item = %v", item)
	}
	img, err := http.Get(e.srv.URL + photo)
	if err != nil || img.StatusCode != 200 || img.Header.Get("Content-Type") != "image/png" {
		t.Errorf("GET photo: %v %v", err, img)
	}
	img.Body.Close()
	e.must(422, "POST", "/api/v1/items", alice.token, obj{
		"title": "x", "category": "sports", "condition": "good", "image_urls": []string{"/uploads/../../etc/passwd"},
	})
	e.must(401, "POST", "/api/v1/uploads", "", nil)

	// Profiles take a city from the fixed list.
	e.must(422, "PUT", "/api/v1/users/me", alice.token, obj{"full_name": "Alice", "city": "Atlantis"})
	me := e.must(200, "PUT", "/api/v1/users/me", alice.token, obj{"full_name": "Alice", "city": "Addis Ababa", "avatar_url": photo})
	if me["city"] != "Addis Ababa" || me["avatar_url"] != photo {
		t.Errorf("profile = %v", me)
	}
	e.must(200, "PUT", "/api/v1/users/me", bob.token, obj{"full_name": "Bob", "city": "Hawassa"})
	if cities := e.must(200, "GET", "/api/v1/categories", "", nil)["cities"].([]any); len(cities) == 0 {
		t.Error("expected cities")
	}

	// Different cities: no match unless both wants allow any city.
	e.item(bob, "Tent", "sports", "good", 0)
	aliceWant := e.want(alice, "sports", []string{"tent"}, nil)
	bobWant := e.want(bob, "sports", []string{"bike"}, nil)
	if n := e.match(); n != 0 {
		t.Fatalf("cross-city match created: %d", n)
	}
	e.must(200, "PUT", "/api/v1/wants/"+aliceWant, alice.token, obj{"category": "sports", "keywords": []string{"tent"}, "any_city": true})
	if n := e.match(); n != 0 {
		t.Fatalf("one-sided any_city should not match: %d", n)
	}
	got := e.must(200, "PUT", "/api/v1/wants/"+bobWant, bob.token, obj{"category": "sports", "keywords": []string{"bike"}, "any_city": true})
	if got["any_city"] != true {
		t.Errorf("want = %v", got)
	}
	if n := e.match(); n != 1 {
		t.Fatalf("both any_city: %d matches, want 1", n)
	}

	// Same city matches by default.
	carol, dan := e.register("Carol"), e.register("Dan")
	for _, u := range []user{carol, dan} {
		e.must(200, "PUT", "/api/v1/users/me", u.token, obj{"full_name": "X", "city": "Bahir Dar"})
	}
	e.item(carol, "Chess set", "toys", "good", 0)
	e.item(dan, "Lego box", "toys", "good", 0)
	e.want(carol, "toys", []string{"lego"}, nil)
	e.want(dan, "toys", []string{"chess"}, nil)
	if n := e.match(); n != 1 {
		t.Fatalf("same city: %d matches, want 1", n)
	}
}
