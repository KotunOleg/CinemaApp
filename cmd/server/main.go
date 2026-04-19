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
		return
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

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func pathInt(r *http.Request, key string) (int32, error) {
	n, err := strconv.Atoi(r.PathValue(key))
	return int32(n), err
}

func queryInt(r *http.Request, key string, def int32) int32 {
	v, err := strconv.Atoi(r.URL.Query().Get(key))
	if err != nil || v <= 0 {
		return def
	}
	return int32(v)
}

func main() {
	loadEnv(".env")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := database.ConfigFromEnv()
	pool, err := database.New(ctx, cfg)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer pool.Close()
	log.Println("Connected to PostgreSQL")

	q := db.New(pool)
	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, database.Health(pool))
	})

	// =========================================================================
	// Cinemas
	// =========================================================================

	mux.HandleFunc("GET /api/cinemas", func(w http.ResponseWriter, r *http.Request) {
		rows, err := q.ListCinemas(r.Context(), db.ListCinemasParams{
			Limit:  queryInt(r, "limit", 20),
			Offset: queryInt(r, "offset", 0),
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, rows)
	})

	mux.HandleFunc("GET /api/cinemas/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := pathInt(r, "id")
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		row, err := q.GetCinema(r.Context(), id)
		if err == pgx.ErrNoRows {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, row)
	})

	mux.HandleFunc("POST /api/cinemas", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name                string `json:"name"`
			Address             string `json:"address"`
			LocationCoordinates string `json:"location_coordinates"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		row, err := q.CreateCinema(r.Context(), db.CreateCinemaParams{
			Name:                req.Name,
			Address:             req.Address,
			LocationCoordinates: pgtype.Text{String: req.LocationCoordinates, Valid: req.LocationCoordinates != ""},
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusCreated, row)
	})

	mux.HandleFunc("PUT /api/cinemas/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := pathInt(r, "id")
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		var req struct {
			Name                string `json:"name"`
			Address             string `json:"address"`
			LocationCoordinates string `json:"location_coordinates"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		row, err := q.UpdateCinema(r.Context(), db.UpdateCinemaParams{
			CinemaID:            id,
			Name:                req.Name,
			Address:             req.Address,
			LocationCoordinates: pgtype.Text{String: req.LocationCoordinates, Valid: req.LocationCoordinates != ""},
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, row)
	})

	mux.HandleFunc("DELETE /api/cinemas/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := pathInt(r, "id")
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		if err := q.DeleteCinema(r.Context(), id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	// =========================================================================
	// Movies
	// =========================================================================

	mux.HandleFunc("GET /api/movies", func(w http.ResponseWriter, r *http.Request) {
		rows, err := q.ListMovies(r.Context(), db.ListMoviesParams{
			Limit:  queryInt(r, "limit", 20),
			Offset: queryInt(r, "offset", 0),
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, rows)
	})

	mux.HandleFunc("GET /api/movies/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := pathInt(r, "id")
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		row, err := q.GetMovie(r.Context(), id)
		if err == pgx.ErrNoRows {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, row)
	})

	mux.HandleFunc("POST /api/movies", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Title       string   `json:"title"`
			Description string   `json:"description"`
			Genre       []string `json:"genre"`
			TrailerURL  string   `json:"trailer_url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		row, err := q.CreateMovie(r.Context(), db.CreateMovieParams{
			Title:       req.Title,
			Description: pgtype.Text{String: req.Description, Valid: req.Description != ""},
			Genre:       req.Genre,
			TrailerUrl:  pgtype.Text{String: req.TrailerURL, Valid: req.TrailerURL != ""},
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusCreated, row)
	})

	mux.HandleFunc("PUT /api/movies/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := pathInt(r, "id")
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		var req struct {
			Title       string   `json:"title"`
			Description string   `json:"description"`
			Genre       []string `json:"genre"`
			TrailerURL  string   `json:"trailer_url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		row, err := q.UpdateMovie(r.Context(), db.UpdateMovieParams{
			MovieID:     id,
			Title:       req.Title,
			Description: pgtype.Text{String: req.Description, Valid: req.Description != ""},
			Genre:       req.Genre,
			TrailerUrl:  pgtype.Text{String: req.TrailerURL, Valid: req.TrailerURL != ""},
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, row)
	})

	mux.HandleFunc("DELETE /api/movies/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := pathInt(r, "id")
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		if err := q.DeleteMovie(r.Context(), id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	// =========================================================================
	// Showtimes
	// =========================================================================

	mux.HandleFunc("GET /api/showtimes", func(w http.ResponseWriter, r *http.Request) {
		rows, err := q.ListShowtimes(r.Context(), db.ListShowtimesParams{
			Limit:  queryInt(r, "limit", 20),
			Offset: queryInt(r, "offset", 0),
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, rows)
	})

	mux.HandleFunc("GET /api/showtimes/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := pathInt(r, "id")
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		row, err := q.GetShowtime(r.Context(), id)
		if err == pgx.ErrNoRows {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, row)
	})

	mux.HandleFunc("POST /api/showtimes", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			MovieID   int32   `json:"movie_id"`
			CinemaID  int32   `json:"cinema_id"`
			StartTime string  `json:"start_time"`
			Price     float64 `json:"price"`
			Duration  int32   `json:"duration"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		t, err := time.Parse(time.RFC3339, req.StartTime)
		if err != nil {
			http.Error(w, "invalid start_time, use RFC3339", http.StatusBadRequest)
			return
		}
		var price pgtype.Numeric
		if err := price.Scan(strconv.FormatFloat(req.Price, 'f', 2, 64)); err != nil {
			http.Error(w, "invalid price", http.StatusBadRequest)
			return
		}
		row, err := q.CreateShowtime(r.Context(), db.CreateShowtimeParams{
			MovieID:   req.MovieID,
			CinemaID:  req.CinemaID,
			StartTime: pgtype.Timestamptz{Time: t, Valid: true},
			Price:     price,
			Duration:  req.Duration,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusCreated, row)
	})

	mux.HandleFunc("PUT /api/showtimes/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := pathInt(r, "id")
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		var req struct {
			MovieID   int32   `json:"movie_id"`
			CinemaID  int32   `json:"cinema_id"`
			StartTime string  `json:"start_time"`
			Price     float64 `json:"price"`
			Duration  int32   `json:"duration"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		t, err := time.Parse(time.RFC3339, req.StartTime)
		if err != nil {
			http.Error(w, "invalid start_time, use RFC3339", http.StatusBadRequest)
			return
		}
		var price pgtype.Numeric
		if err := price.Scan(strconv.FormatFloat(req.Price, 'f', 2, 64)); err != nil {
			http.Error(w, "invalid price", http.StatusBadRequest)
			return
		}
		row, err := q.UpdateShowtime(r.Context(), db.UpdateShowtimeParams{
			ShowtimeID: id,
			MovieID:    req.MovieID,
			CinemaID:   req.CinemaID,
			StartTime:  pgtype.Timestamptz{Time: t, Valid: true},
			Price:      price,
			Duration:   req.Duration,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, row)
	})

	mux.HandleFunc("DELETE /api/showtimes/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := pathInt(r, "id")
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		if err := q.DeleteShowtime(r.Context(), id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	// =========================================================================
	// Tickets
	// =========================================================================

	mux.HandleFunc("GET /api/tickets", func(w http.ResponseWriter, r *http.Request) {
		rows, err := q.ListTickets(r.Context(), db.ListTicketsParams{
			Limit:  queryInt(r, "limit", 20),
			Offset: queryInt(r, "offset", 0),
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, rows)
	})

	mux.HandleFunc("GET /api/tickets/{id}", func(w http.ResponseWriter, r *http.Request) {
		var ticketID pgtype.UUID
		if err := ticketID.Scan(r.PathValue("id")); err != nil {
			http.Error(w, "invalid uuid", http.StatusBadRequest)
			return
		}
		row, err := q.GetTicket(r.Context(), ticketID)
		if err == pgx.ErrNoRows {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, row)
	})

	mux.HandleFunc("POST /api/tickets", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ShowtimeID    int32  `json:"showtime_id"`
			UserID        int32  `json:"user_id"`
			SeatNumber    string `json:"seat_number"`
			PaymentStatus string `json:"payment_status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if req.PaymentStatus == "" {
			req.PaymentStatus = "pending"
		}
		row, err := q.CreateTicket(r.Context(), db.CreateTicketParams{
			ShowtimeID:    req.ShowtimeID,
			UserID:        req.UserID,
			SeatNumber:    req.SeatNumber,
			PaymentStatus: pgtype.Text{String: req.PaymentStatus, Valid: true},
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusCreated, row)
	})

	mux.HandleFunc("PATCH /api/tickets/{id}", func(w http.ResponseWriter, r *http.Request) {
		var ticketID pgtype.UUID
		if err := ticketID.Scan(r.PathValue("id")); err != nil {
			http.Error(w, "invalid uuid", http.StatusBadRequest)
			return
		}
		var req struct {
			PaymentStatus string `json:"payment_status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		row, err := q.UpdateTicketStatus(r.Context(), db.UpdateTicketStatusParams{
			TicketID:      ticketID,
			PaymentStatus: pgtype.Text{String: req.PaymentStatus, Valid: true},
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, row)
	})

	mux.HandleFunc("DELETE /api/tickets/{id}", func(w http.ResponseWriter, r *http.Request) {
		var ticketID pgtype.UUID
		if err := ticketID.Scan(r.PathValue("id")); err != nil {
			http.Error(w, "invalid uuid", http.StatusBadRequest)
			return
		}
		if err := q.DeleteTicket(r.Context(), ticketID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	// =========================================================================
	// Users
	// =========================================================================

	mux.HandleFunc("GET /api/users", func(w http.ResponseWriter, r *http.Request) {
		rows, err := q.ListUsers(r.Context(), db.ListUsersParams{
			Limit:  queryInt(r, "limit", 20),
			Offset: queryInt(r, "offset", 0),
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, rows)
	})

	mux.HandleFunc("GET /api/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := pathInt(r, "id")
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		row, err := q.GetUser(r.Context(), id)
		if err == pgx.ErrNoRows {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, row)
	})

	mux.HandleFunc("POST /api/users", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Email           string `json:"email"`
			Password        string `json:"password"`
			Phone           string `json:"phone"`
			FullName        string `json:"full_name"`
			PermissionLevel int32  `json:"permission_level"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if req.PermissionLevel == 0 {
			req.PermissionLevel = 2
		}
		row, err := q.CreateUser(r.Context(), db.CreateUserParams{
			Email:           req.Email,
			PasswordHash:    hashPassword(req.Password),
			Phone:           pgtype.Text{String: req.Phone, Valid: req.Phone != ""},
			FullName:        req.FullName,
			PermissionLevel: pgtype.Int4{Int32: req.PermissionLevel, Valid: true},
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusCreated, row)
	})

	mux.HandleFunc("PUT /api/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := pathInt(r, "id")
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		var req struct {
			Email           string `json:"email"`
			Phone           string `json:"phone"`
			FullName        string `json:"full_name"`
			IsBlocked       bool   `json:"is_blocked"`
			PermissionLevel int32  `json:"permission_level"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		row, err := q.UpdateUser(r.Context(), db.UpdateUserParams{
			UserID:          id,
			Email:           req.Email,
			Phone:           pgtype.Text{String: req.Phone, Valid: req.Phone != ""},
			FullName:        req.FullName,
			IsBlocked:       pgtype.Bool{Bool: req.IsBlocked, Valid: true},
			PermissionLevel: pgtype.Int4{Int32: req.PermissionLevel, Valid: true},
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, row)
	})

	mux.HandleFunc("DELETE /api/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := pathInt(r, "id")
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		if err := q.DeleteUser(r.Context(), id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	// =========================================================================
	// Reviews
	// =========================================================================

	mux.HandleFunc("GET /api/movies/{id}/reviews", func(w http.ResponseWriter, r *http.Request) {
		id, err := pathInt(r, "id")
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		rows, err := q.ListReviewsByMovie(r.Context(), id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, rows)
	})

	mux.HandleFunc("GET /api/reviews", func(w http.ResponseWriter, r *http.Request) {
		rows, err := q.ListReviews(r.Context(), db.ListReviewsParams{
			Limit:  queryInt(r, "limit", 20),
			Offset: queryInt(r, "offset", 0),
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, rows)
	})

	mux.HandleFunc("GET /api/reviews/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := pathInt(r, "id")
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		row, err := q.GetReview(r.Context(), id)
		if err == pgx.ErrNoRows {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, row)
	})

	mux.HandleFunc("POST /api/reviews", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			UserID  int32  `json:"user_id"`
			MovieID int32  `json:"movie_id"`
			Rating  int32  `json:"rating"`
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		row, err := q.CreateReview(r.Context(), db.CreateReviewParams{
			UserID:  req.UserID,
			MovieID: req.MovieID,
			Rating:  pgtype.Int4{Int32: req.Rating, Valid: req.Rating > 0},
			Content: pgtype.Text{String: req.Content, Valid: req.Content != ""},
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusCreated, row)
	})

	mux.HandleFunc("PUT /api/reviews/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := pathInt(r, "id")
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		var req struct {
			Rating  int32  `json:"rating"`
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		row, err := q.UpdateReview(r.Context(), db.UpdateReviewParams{
			ReviewID: id,
			Rating:   pgtype.Int4{Int32: req.Rating, Valid: req.Rating > 0},
			Content:  pgtype.Text{String: req.Content, Valid: req.Content != ""},
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, row)
	})

	mux.HandleFunc("DELETE /api/reviews/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := pathInt(r, "id")
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		if err := q.DeleteReview(r.Context(), id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	// =========================================================================
	// Permissions
	// =========================================================================

	mux.HandleFunc("GET /api/permissions", func(w http.ResponseWriter, r *http.Request) {
		rows, err := q.ListPermissions(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, rows)
	})

	// Static files (React build)
	mux.Handle("/", http.FileServer(http.Dir("./static")))

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
		log.Printf("Server running on http://localhost:%s", port)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()
	server.Shutdown(shutCtx)
	log.Println("Server stopped")
}

func hashPassword(password string) string {
	return "hashed_" + password
}
