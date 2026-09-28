package app_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yeabt/lewe/internal/app"
	"github.com/yeabt/lewe/internal/db"
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
		`TRUNCATE users, refresh_tokens, items, wants, matches, ratings CASCADE`)
	if err != nil {
		t.Fatalf("truncate: %v", err)
	}

	a := app.New(pool, testSecret, time.Hour)
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
