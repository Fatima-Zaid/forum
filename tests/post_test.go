package tests

import (
	"database/sql"
	"testing"
	"time"

	"forum/database"
	"forum/models"
)

// newTestDB spins up a fresh in-memory database with the schema applied,
// isolated per test (nothing touches your real forum.db).
func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := database.InitDB(":memory:")
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestCreateAndGetUser(t *testing.T) {
	db := newTestDB(t)

	id, err := database.CreateUser(db, "alice", "alice@example.com", "hashedpw")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero user id")
	}

	u, err := database.GetUserByEmail(db, "alice@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail failed: %v", err)
	}
	if u.Username != "alice" {
		t.Errorf("expected username alice, got %s", u.Username)
	}

	// duplicate email must fail with ErrEmailTaken
	_, err = database.CreateUser(db, "alice2", "alice@example.com", "otherhash")
	if err != database.ErrEmailTaken {
		t.Errorf("expected ErrEmailTaken, got %v", err)
	}

	exists, err := database.EmailExists(db, "nobody@example.com")
	if err != nil {
		t.Fatalf("EmailExists failed: %v", err)
	}
	if exists {
		t.Error("expected EmailExists to be false for unknown email")
	}
}

func TestCreatePostWithCategories(t *testing.T) {
	db := newTestDB(t)

	userID, err := database.CreateUser(db, "bob", "bob@example.com", "hash")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	catID, err := database.GetOrCreateCategory(db, "RPG")
	if err != nil {
		t.Fatalf("GetOrCreateCategory failed: %v", err)
	}
	// second call with different case must reuse the same row, not duplicate
	catID2, err := database.GetOrCreateCategory(db, "rpg")
	if err != nil {
		t.Fatalf("GetOrCreateCategory (case-insensitive) failed: %v", err)
	}
	if catID != catID2 {
		t.Errorf("expected same category id for 'RPG' and 'rpg', got %d vs %d", catID, catID2)
	}

	postID, err := database.CreatePost(db, userID, "Great game", "Elden Ring", "Loved it.", []int{catID})
	if err != nil {
		t.Fatalf("CreatePost failed: %v", err)
	}

	post, err := database.GetPostByID(db, postID)
	if err != nil {
		t.Fatalf("GetPostByID failed: %v", err)
	}
	if post.Title != "Great game" {
		t.Errorf("expected title 'Great game', got %s", post.Title)
	}
	if len(post.Categories) != 1 || post.Categories[0].Name != "RPG" {
		t.Errorf("expected 1 category 'RPG', got %+v", post.Categories)
	}
	if post.Username != "bob" {
		t.Errorf("expected author bob, got %s", post.Username)
	}
}

func TestPostFilters(t *testing.T) {
	db := newTestDB(t)

	userA, _ := database.CreateUser(db, "userA", "a@example.com", "hash")
	userB, _ := database.CreateUser(db, "userB", "b@example.com", "hash")
	catID, _ := database.GetOrCreateCategory(db, "Shooter")

	postA, err := database.CreatePost(db, userA, "A's post", "", "content", []int{catID})
	if err != nil {
		t.Fatalf("CreatePost failed: %v", err)
	}
	_, err = database.CreatePost(db, userB, "B's post", "", "content", []int{catID})
	if err != nil {
		t.Fatalf("CreatePost failed: %v", err)
	}

	// filter: created posts (userA should only see their own)
	mine, err := database.GetPostsByUser(db, userA)
	if err != nil {
		t.Fatalf("GetPostsByUser failed: %v", err)
	}
	if len(mine) != 1 || mine[0].ID != postA {
		t.Errorf("expected exactly post A for userA's created filter, got %+v", mine)
	}

	// filter: category
	byCat, err := database.GetPostsByCategory(db, catID)
	if err != nil {
		t.Fatalf("GetPostsByCategory failed: %v", err)
	}
	if len(byCat) != 2 {
		t.Errorf("expected 2 posts in category, got %d", len(byCat))
	}

	// filter: liked posts
	if err := database.SetPostReaction(db, postA, userB, models.Like); err != nil {
		t.Fatalf("SetPostReaction failed: %v", err)
	}
	liked, err := database.GetPostsLikedByUser(db, userB)
	if err != nil {
		t.Fatalf("GetPostsLikedByUser failed: %v", err)
	}
	if len(liked) != 1 || liked[0].ID != postA {
		t.Errorf("expected post A in userB's liked filter, got %+v", liked)
	}
}

func TestReactionToggle(t *testing.T) {
	db := newTestDB(t)

	userID, _ := database.CreateUser(db, "carl", "carl@example.com", "hash")
	catID, _ := database.GetOrCreateCategory(db, "Indie")
	postID, err := database.CreatePost(db, userID, "Post", "", "content", []int{catID})
	if err != nil {
		t.Fatalf("CreatePost failed: %v", err)
	}

	// like it
	if err := database.SetPostReaction(db, postID, userID, models.Like); err != nil {
		t.Fatalf("SetPostReaction (like) failed: %v", err)
	}
	post, _ := database.GetPostByID(db, postID)
	if post.LikeCount != 1 || post.DislikeCount != 0 {
		t.Errorf("expected 1 like 0 dislikes, got %d/%d", post.LikeCount, post.DislikeCount)
	}

	// like again -> should toggle off
	if err := database.SetPostReaction(db, postID, userID, models.Like); err != nil {
		t.Fatalf("SetPostReaction (untoggle) failed: %v", err)
	}
	post, _ = database.GetPostByID(db, postID)
	if post.LikeCount != 0 {
		t.Errorf("expected like removed after second click, got %d", post.LikeCount)
	}

	// dislike -> should set dislike
	if err := database.SetPostReaction(db, postID, userID, models.Dislike); err != nil {
		t.Fatalf("SetPostReaction (dislike) failed: %v", err)
	}
	post, _ = database.GetPostByID(db, postID)
	if post.DislikeCount != 1 || post.LikeCount != 0 {
		t.Errorf("expected 0 likes 1 dislike, got %d/%d", post.LikeCount, post.DislikeCount)
	}
}

func TestSessionExpiration(t *testing.T) {
	db := newTestDB(t)

	userID, _ := database.CreateUser(db, "dana", "dana@example.com", "hash")

	// expired session (1 hour in the past)
	past := time.Now().Add(-time.Hour)
	if err := database.CreateSession(db, "expired-session", userID, past); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	_, err := database.GetValidSession(db, "expired-session")
	if err != database.ErrNotFound {
		t.Errorf("expected ErrNotFound for expired session, got %v", err)
	}

	// valid session (1 hour in the future)
	future := time.Now().Add(time.Hour)
	if err := database.CreateSession(db, "valid-session", userID, future); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	session, err := database.GetValidSession(db, "valid-session")
	if err != nil {
		t.Fatalf("GetValidSession failed for valid session: %v", err)
	}
	if session.UserID != userID {
		t.Errorf("expected session for user %d, got %d", userID, session.UserID)
	}
}
