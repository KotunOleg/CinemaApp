package main

import (
	"bufio"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"webapp/internal/database"
	"webapp/internal/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func loadEnv(filename string) {
	f, err := os.Open(filename)
	if err != nil {
		return // no .env file is fine
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}
}

func main() {
	loadEnv(".env")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Connect to database
	cfg := database.ConfigFromEnv()
	pool, err := database.New(ctx, cfg)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer pool.Close()
	log.Println("✓ Connected to PostgreSQL")

	// Create sqlc queries instance
	queries := db.New(pool)

	// Setup routes
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(database.Health(pool))
	})

	// =========================================================================
	// User endpoints (demonstrating sqlc type safety)
	// =========================================================================

	// List users
	mux.HandleFunc("GET /api/users", func(w http.ResponseWriter, r *http.Request) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit <= 0 {
			limit = 20
		}
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

		// Type-safe! Compiler checks all fields
		users, err := queries.ListUsers(r.Context(), db.ListUsersParams{
			Active: pgtype.Bool{Valid: false}, // NULL = all users
			Limit:  int32(limit),
			Offset: int32(offset),
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(users)
	})

	// Get single user
	mux.HandleFunc("GET /api/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
			return
		}

		user, err := queries.GetUser(r.Context(), int32(id))
		if err == pgx.ErrNoRows {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
	})

	// Create user
	mux.HandleFunc("POST /api/users", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name     string `json:"name"`
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Type-safe parameters - compiler verifies this!
		user, err := queries.CreateUser(r.Context(), db.CreateUserParams{
			Name:         req.Name,
			Email:        req.Email,
			PasswordHash: hashPassword(req.Password), // You'd use bcrypt
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(user)
	})

	// Update user
	mux.HandleFunc("PATCH /api/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(r.PathValue("id"))

		var req struct {
			Name  *string `json:"name"`
			Email *string `json:"email"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Build update params with nullable fields
		params := db.UpdateUserParams{ID: int32(id)}
		if req.Name != nil {
			params.Name = db.NullString(*req.Name)
		}
		if req.Email != nil {
			params.Email = db.NullString(*req.Email)
		}

		user, err := queries.UpdateUser(r.Context(), params)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
	})

	// Delete user
	mux.HandleFunc("DELETE /api/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(r.PathValue("id"))
		if err := queries.DeleteUser(r.Context(), int32(id)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	// =========================================================================
	// Post endpoints with JOINs
	// =========================================================================

	// List posts with author info (demonstrates JOIN queries)
	mux.HandleFunc("GET /api/posts", func(w http.ResponseWriter, r *http.Request) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit <= 0 {
			limit = 20
		}
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

		// Returns ListPostsWithAuthorsRow - includes author_id, author_name
		posts, err := queries.ListPostsWithAuthors(r.Context(), int32(limit), int32(offset))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(posts)
	})

	// Get post with full author details
	mux.HandleFunc("GET /api/posts/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(r.PathValue("id"))

		// Returns GetPostWithAuthorRow - different struct than ListPosts!
		post, err := queries.GetPostWithAuthor(r.Context(), int32(id))
		if err == pgx.ErrNoRows {
			http.Error(w, "Post not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(post)
	})

	// Create post
	mux.HandleFunc("POST /api/posts", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			UserID    int32  `json:"user_id"`
			Title     string `json:"title"`
			Content   string `json:"content"`
			Published bool   `json:"published"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		post, err := queries.CreatePost(r.Context(), db.CreatePostParams{
			UserID:    req.UserID,
			Title:     req.Title,
			Content:   db.NullString(req.Content),
			Published: db.NullBool(req.Published),
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(post)
	})

	// =========================================================================
	// Transaction example
	// =========================================================================

	// Create user with initial post (transaction)
	mux.HandleFunc("POST /api/users/with-post", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name      string `json:"name"`
			Email     string `json:"email"`
			Password  string `json:"password"`
			PostTitle string `json:"post_title"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Start transaction
		tx, err := pool.Begin(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer tx.Rollback(r.Context())

		// Use queries with transaction
		qtx := queries.WithTx(tx)

		// Create user
		user, err := qtx.CreateUser(r.Context(), db.CreateUserParams{
			Name:         req.Name,
			Email:        req.Email,
			PasswordHash: hashPassword(req.Password),
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Create post
		post, err := qtx.CreatePost(r.Context(), db.CreatePostParams{
			UserID:    user.ID,
			Title:     req.PostTitle,
			Published: db.NullBool(true),
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Commit transaction
		if err := tx.Commit(r.Context()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"user": user,
			"post": post,
		})
	})

	// Serve static files
	mux.Handle("GET /", http.FileServer(http.Dir("./static")))

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		log.Printf("🚀 Server running on http://localhost:%s", port)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	server.Shutdown(ctx)
	log.Println("Server stopped")
}

// Placeholder - use bcrypt in production
func hashPassword(password string) string {
	return "hashed_" + password
}
