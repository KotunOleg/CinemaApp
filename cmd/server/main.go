package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
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

	"crypto/rand"
	"encoding/hex"
	"io"
	"net/url"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/xuri/excelize/v2"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
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

	// Google OAuth2 config
	googleOAuth := &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  "http://localhost:8080/api/auth/google/callback",
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}

	// Seed default admin on first run
	go func() {
		ctx2 := context.Background()
		_, err := q.GetUserByEmail(ctx2, "admin@cinema.com")
		if err == pgx.ErrNoRows {
			q.CreateUser(ctx2, db.CreateUserParams{
				Email:           "admin@cinema.com",
				PasswordHash:    hashPassword("Admin_1234"),
				FullName:        "Administrator",
				PermissionLevel: pgtype.Int4{Int32: 1, Valid: true},
			})
			log.Println("Default admin created: admin@cinema.com / Admin_1234")
		}
	}()

	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, database.Health(pool))
	})

	// =========================================================================
	// Auth
	// =========================================================================

	mux.HandleFunc("POST /api/auth/register", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
			FullName string `json:"full_name"`
			Phone    string `json:"phone"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if req.Email == "" || req.Password == "" || req.FullName == "" {
			http.Error(w, "email, password, full_name required", http.StatusBadRequest)
			return
		}
		user, err := q.CreateUser(r.Context(), db.CreateUserParams{
			Email:           req.Email,
			PasswordHash:    hashPassword(req.Password),
			Phone:           pgtype.Text{String: req.Phone, Valid: req.Phone != ""},
			FullName:        req.FullName,
			PermissionLevel: pgtype.Int4{Int32: 2, Valid: true},
		})
		if err != nil {
			if strings.Contains(err.Error(), "23505") || strings.Contains(err.Error(), "unique") {
				http.Error(w, "This email is already registered", http.StatusConflict)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		token, err := generateToken(user.UserID, user.PermissionLevel.Int32)
		if err != nil {
			http.Error(w, "token error", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{
			"token": token,
			"user": map[string]any{
				"user_id":          user.UserID,
				"email":            user.Email,
				"full_name":        user.FullName,
				"permission_level": user.PermissionLevel.Int32,
			},
		})
	})

	mux.HandleFunc("POST /api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		user, err := q.GetUserByEmail(r.Context(), req.Email)
		if err == pgx.ErrNoRows {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !checkPassword(user.PasswordHash, req.Password) {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}
		if user.IsBlocked.Valid && user.IsBlocked.Bool {
			http.Error(w, "account is blocked", http.StatusForbidden)
			return
		}
		token, err := generateToken(user.UserID, user.PermissionLevel.Int32)
		if err != nil {
			http.Error(w, "token error", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"token": token,
			"user": map[string]any{
				"user_id":          user.UserID,
				"email":            user.Email,
				"full_name":        user.FullName,
				"permission_level": user.PermissionLevel.Int32,
			},
		})
	})

	mux.HandleFunc("GET /api/auth/me", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(ctxUserID).(int32)
		user, err := q.GetUser(r.Context(), userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"user_id":          user.UserID,
			"email":            user.Email,
			"full_name":        user.FullName,
			"permission_level": user.PermissionLevel.Int32,
		})
	}))

	// =========================================================================
	// Profile (self-service, requires auth only)
	// =========================================================================

	mux.HandleFunc("GET /api/profile/tickets", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(ctxUserID).(int32)
		rows, err := pool.Query(r.Context(), `
			SELECT t.ticket_id, t.showtime_id, t.user_id, t.seat_number, t.payment_status,
			       m.title AS movie_title, c.name AS cinema_name, s.start_time
			FROM tickets t
			JOIN showtimes s ON s.showtime_id = t.showtime_id
			JOIN movies m ON m.movie_id = s.movie_id
			JOIN cinemas c ON c.cinema_id = s.cinema_id
			WHERE t.user_id = $1
			ORDER BY s.start_time DESC
		`, userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		type ticketRow struct {
			TicketID      string `json:"ticket_id"`
			ShowtimeID    int32  `json:"showtime_id"`
			UserID        int32  `json:"user_id"`
			SeatNumber    string `json:"seat_number"`
			PaymentStatus string `json:"payment_status"`
			MovieTitle    string `json:"movie_title"`
			CinemaName    string `json:"cinema_name"`
			StartTime     string `json:"start_time"`
		}
		result := []ticketRow{}
		for rows.Next() {
			var row ticketRow
			var ticketID [16]byte
			var payStatus pgtype.Text
			var startTime pgtype.Timestamptz
			if err := rows.Scan(&ticketID, &row.ShowtimeID, &row.UserID, &row.SeatNumber,
				&payStatus, &row.MovieTitle, &row.CinemaName, &startTime); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			row.TicketID = fmt.Sprintf("%x-%x-%x-%x-%x", ticketID[0:4], ticketID[4:6], ticketID[6:8], ticketID[8:10], ticketID[10:])
			row.PaymentStatus = payStatus.String
			if startTime.Valid {
				row.StartTime = startTime.Time.Format(time.RFC3339)
			}
			result = append(result, row)
		}
		writeJSON(w, http.StatusOK, result)
	}))

	mux.HandleFunc("PUT /api/profile", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(ctxUserID).(int32)
		var req struct {
			FullName string `json:"full_name"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if req.FullName != "" {
			if _, err := pool.Exec(r.Context(),
				`UPDATE users SET full_name = $1, updated_at = NOW() WHERE user_id = $2`,
				req.FullName, userID); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		if req.Password != "" {
			if _, err := pool.Exec(r.Context(),
				`UPDATE users SET password_hash = $1, updated_at = NOW() WHERE user_id = $2`,
				hashPassword(req.Password), userID); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}))

	// Google OAuth — redirect to consent screen
	mux.HandleFunc("GET /api/auth/google", func(w http.ResponseWriter, r *http.Request) {
		state := randomState()
		http.SetCookie(w, &http.Cookie{
			Name:     "oauth_state",
			Value:    state,
			MaxAge:   300,
			HttpOnly: true,
			Path:     "/",
		})
		http.Redirect(w, r, googleOAuth.AuthCodeURL(state, oauth2.AccessTypeOnline), http.StatusTemporaryRedirect)
	})

	// Google OAuth — handle callback
	mux.HandleFunc("GET /api/auth/google/callback", func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("oauth_state")
		if err != nil || cookie.Value != r.URL.Query().Get("state") {
			http.Error(w, "invalid state", http.StatusBadRequest)
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "oauth_state", MaxAge: -1, Path: "/"})

		token, err := googleOAuth.Exchange(r.Context(), r.URL.Query().Get("code"))
		if err != nil {
			http.Error(w, "code exchange failed", http.StatusInternalServerError)
			return
		}

		client := googleOAuth.Client(r.Context(), token)
		resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
		if err != nil {
			http.Error(w, "failed to get user info", http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)

		var info struct {
			Email string `json:"email"`
			Name  string `json:"name"`
		}
		if err := json.Unmarshal(body, &info); err != nil || info.Email == "" {
			http.Error(w, "invalid user info", http.StatusInternalServerError)
			return
		}

		// Find or create user
		user, err := q.GetUserByEmail(r.Context(), info.Email)
		if err == pgx.ErrNoRows {
			user, err = q.CreateUser(r.Context(), db.CreateUserParams{
				Email:           info.Email,
				PasswordHash:    "",
				FullName:        info.Name,
				PermissionLevel: pgtype.Int4{Int32: 2, Valid: true},
			})
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		jwtToken, err := generateToken(user.UserID, user.PermissionLevel.Int32)
		if err != nil {
			http.Error(w, "token error", http.StatusInternalServerError)
			return
		}

		redirectURL := "http://localhost:5173/oauth?token=" + url.QueryEscape(jwtToken)
		if os.Getenv("PORT") != "" {
			redirectURL = "/oauth?token=" + url.QueryEscape(jwtToken)
		}
		http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
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

	mux.HandleFunc("POST /api/cinemas", requireAdmin(func(w http.ResponseWriter, r *http.Request) {
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
	}))

	mux.HandleFunc("PUT /api/cinemas/{id}", requireAdmin(func(w http.ResponseWriter, r *http.Request) {
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
	}))

	mux.HandleFunc("DELETE /api/cinemas/{id}", requireAdmin(func(w http.ResponseWriter, r *http.Request) {
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
	}))

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

	mux.HandleFunc("POST /api/movies", requireAdmin(func(w http.ResponseWriter, r *http.Request) {
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
	}))

	mux.HandleFunc("PUT /api/movies/{id}", requireAdmin(func(w http.ResponseWriter, r *http.Request) {
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
	}))

	mux.HandleFunc("DELETE /api/movies/{id}", requireAdmin(func(w http.ResponseWriter, r *http.Request) {
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
	}))

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

	mux.HandleFunc("GET /api/showtimes/{id}/seats", func(w http.ResponseWriter, r *http.Request) {
		id, err := pathInt(r, "id")
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		rows, err := pool.Query(r.Context(), `SELECT seat_number FROM tickets WHERE showtime_id = $1`, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		seats := []string{}
		for rows.Next() {
			var s string
			if err := rows.Scan(&s); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			seats = append(seats, s)
		}
		writeJSON(w, http.StatusOK, seats)
	})

	mux.HandleFunc("POST /api/showtimes", requireAdmin(func(w http.ResponseWriter, r *http.Request) {
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
		if req.Duration < 10 {
			http.Error(w, "duration must be at least 10 minutes", http.StatusBadRequest)
			return
		}
		t, err := time.Parse(time.RFC3339, req.StartTime)
		if err != nil {
			http.Error(w, "invalid start_time, use RFC3339", http.StatusBadRequest)
			return
		}
		todayUTC := time.Now().UTC().Truncate(24 * time.Hour)
		if !t.UTC().Truncate(24 * time.Hour).After(todayUTC) {
			http.Error(w, "showtime date must be tomorrow or later", http.StatusBadRequest)
			return
		}
		var count int
		err = pool.QueryRow(r.Context(), `
			SELECT COUNT(*) FROM showtimes
			WHERE cinema_id = $1
			  AND start_time < $2::timestamptz + ($3 * interval '1 minute')
			  AND start_time + (duration * interval '1 minute') > $2::timestamptz
		`, req.CinemaID, t, req.Duration).Scan(&count)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if count > 0 {
			http.Error(w, "showtime overlaps with an existing session in this cinema", http.StatusConflict)
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
	}))

	mux.HandleFunc("PUT /api/showtimes/{id}", requireAdmin(func(w http.ResponseWriter, r *http.Request) {
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
		if req.Duration < 10 {
			http.Error(w, "duration must be at least 10 minutes", http.StatusBadRequest)
			return
		}
		t, err := time.Parse(time.RFC3339, req.StartTime)
		if err != nil {
			http.Error(w, "invalid start_time, use RFC3339", http.StatusBadRequest)
			return
		}
		todayUTC := time.Now().UTC().Truncate(24 * time.Hour)
		if !t.UTC().Truncate(24 * time.Hour).After(todayUTC) {
			http.Error(w, "showtime date must be tomorrow or later", http.StatusBadRequest)
			return
		}
		var count int
		err = pool.QueryRow(r.Context(), `
			SELECT COUNT(*) FROM showtimes
			WHERE cinema_id = $1
			  AND showtime_id != $2
			  AND start_time < $3::timestamptz + ($4 * interval '1 minute')
			  AND start_time + (duration * interval '1 minute') > $3::timestamptz
		`, req.CinemaID, id, t, req.Duration).Scan(&count)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if count > 0 {
			http.Error(w, "showtime overlaps with an existing session in this cinema", http.StatusConflict)
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
	}))

	mux.HandleFunc("DELETE /api/showtimes/{id}", requireAdmin(func(w http.ResponseWriter, r *http.Request) {
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
	}))

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

	mux.HandleFunc("POST /api/tickets", requireAuth(func(w http.ResponseWriter, r *http.Request) {
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
		var taken bool
		if err := pool.QueryRow(r.Context(),
			`SELECT EXISTS(SELECT 1 FROM tickets WHERE showtime_id = $1 AND seat_number = $2)`,
			req.ShowtimeID, req.SeatNumber,
		).Scan(&taken); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if taken {
			http.Error(w, "seat is already booked", http.StatusConflict)
			return
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
	}))

	mux.HandleFunc("PATCH /api/tickets/{id}", requireAdmin(func(w http.ResponseWriter, r *http.Request) {
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
	}))

	mux.HandleFunc("DELETE /api/tickets/{id}", requireAdmin(func(w http.ResponseWriter, r *http.Request) {
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
	}))

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

	mux.HandleFunc("POST /api/users", requireAdmin(func(w http.ResponseWriter, r *http.Request) {
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
	}))

	mux.HandleFunc("PUT /api/users/{id}", requireAdmin(func(w http.ResponseWriter, r *http.Request) {
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
	}))

	mux.HandleFunc("DELETE /api/users/{id}", requireAdmin(func(w http.ResponseWriter, r *http.Request) {
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
	}))

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

	mux.HandleFunc("POST /api/reviews", requireAuth(func(w http.ResponseWriter, r *http.Request) {
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
	}))

	mux.HandleFunc("PUT /api/reviews/{id}", requireAdmin(func(w http.ResponseWriter, r *http.Request) {
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
	}))

	mux.HandleFunc("DELETE /api/reviews/{id}", requireAdmin(func(w http.ResponseWriter, r *http.Request) {
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
	}))

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

	// =========================================================================
	// Statistics
	// =========================================================================

	mux.HandleFunc("GET /api/stats/movies-by-genre", func(w http.ResponseWriter, r *http.Request) {
		rows, err := pool.Query(r.Context(), `
			SELECT g, COUNT(*) AS count
			FROM movies, unnest(genre) AS g
			GROUP BY g
			ORDER BY count DESC`)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		type row struct {
			Genre string `json:"genre"`
			Count int64  `json:"count"`
		}
		var result []row
		for rows.Next() {
			var item row
			if err := rows.Scan(&item.Genre, &item.Count); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			result = append(result, item)
		}
		if result == nil {
			result = []row{}
		}
		writeJSON(w, http.StatusOK, result)
	})

	mux.HandleFunc("GET /api/stats/avg-rating", func(w http.ResponseWriter, r *http.Request) {
		rows, err := pool.Query(r.Context(), `
			SELECT m.title, ROUND(AVG(r.rating)::numeric, 1)::float8 AS avg_rating, COUNT(r.review_id) AS review_count
			FROM movies m
			JOIN reviews r ON r.movie_id = m.movie_id
			GROUP BY m.movie_id, m.title
			ORDER BY avg_rating DESC
			LIMIT 10`)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		type row struct {
			Title       string  `json:"title"`
			AvgRating   float64 `json:"avg_rating"`
			ReviewCount int64   `json:"review_count"`
		}
		var result []row
		for rows.Next() {
			var item row
			if err := rows.Scan(&item.Title, &item.AvgRating, &item.ReviewCount); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			result = append(result, item)
		}
		if result == nil {
			result = []row{}
		}
		writeJSON(w, http.StatusOK, result)
	})

	// =========================================================================
	// Excel Import / Export
	// =========================================================================

	mux.HandleFunc("GET /api/tickets/export", requireAdmin(func(w http.ResponseWriter, r *http.Request) {
		tickets, err := q.ListTickets(r.Context(), db.ListTicketsParams{Limit: 10000, Offset: 0})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		f := excelize.NewFile()
		sheet := "Tickets"
		f.SetSheetName("Sheet1", sheet)
		for i, h := range []string{"ID", "Movie", "Cinema", "Start Time", "Seat", "User", "Email", "Status"} {
			cell, _ := excelize.CoordinatesToCellName(i+1, 1)
			f.SetCellValue(sheet, cell, h)
		}
		for i, t := range tickets {
			row := i + 2
			startTime := ""
			if t.StartTime.Valid {
				startTime = t.StartTime.Time.Format("02.01.2006 15:04")
			}
			f.SetCellValue(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("%x-%x-%x-%x-%x", t.TicketID.Bytes[0:4], t.TicketID.Bytes[4:6], t.TicketID.Bytes[6:8], t.TicketID.Bytes[8:10], t.TicketID.Bytes[10:]))
			f.SetCellValue(sheet, fmt.Sprintf("B%d", row), t.MovieTitle)
			f.SetCellValue(sheet, fmt.Sprintf("C%d", row), t.CinemaName)
			f.SetCellValue(sheet, fmt.Sprintf("D%d", row), startTime)
			f.SetCellValue(sheet, fmt.Sprintf("E%d", row), t.SeatNumber)
			f.SetCellValue(sheet, fmt.Sprintf("F%d", row), t.UserName)
			f.SetCellValue(sheet, fmt.Sprintf("G%d", row), t.UserEmail)
			f.SetCellValue(sheet, fmt.Sprintf("H%d", row), t.PaymentStatus.String)
		}
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", `attachment; filename="tickets.xlsx"`)
		f.Write(w)
	}))

	mux.HandleFunc("GET /api/profile/tickets/export", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(ctxUserID).(int32)
		rows, err := pool.Query(r.Context(), `
			SELECT t.ticket_id, t.seat_number, t.payment_status,
			       m.title AS movie_title, c.name AS cinema_name, s.start_time
			FROM tickets t
			JOIN showtimes s ON s.showtime_id = t.showtime_id
			JOIN movies m ON m.movie_id = s.movie_id
			JOIN cinemas c ON c.cinema_id = s.cinema_id
			WHERE t.user_id = $1
			ORDER BY s.start_time DESC
		`, userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		f := excelize.NewFile()
		sheet := "My Tickets"
		f.SetSheetName("Sheet1", sheet)
		for i, h := range []string{"Movie", "Cinema", "Start Time", "Seat", "Status"} {
			cell, _ := excelize.CoordinatesToCellName(i+1, 1)
			f.SetCellValue(sheet, cell, h)
		}
		rowNum := 2
		for rows.Next() {
			var ticketID [16]byte
			var seat, movie, cinema string
			var status pgtype.Text
			var startTime pgtype.Timestamptz
			if err := rows.Scan(&ticketID, &seat, &status, &movie, &cinema, &startTime); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			st := ""
			if startTime.Valid {
				st = startTime.Time.Format("02.01.2006 15:04")
			}
			f.SetCellValue(sheet, fmt.Sprintf("A%d", rowNum), movie)
			f.SetCellValue(sheet, fmt.Sprintf("B%d", rowNum), cinema)
			f.SetCellValue(sheet, fmt.Sprintf("C%d", rowNum), st)
			f.SetCellValue(sheet, fmt.Sprintf("D%d", rowNum), seat)
			f.SetCellValue(sheet, fmt.Sprintf("E%d", rowNum), status.String)
			rowNum++
		}
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", `attachment; filename="my_tickets.xlsx"`)
		f.Write(w)
	}))

	mux.HandleFunc("GET /api/movies/export", func(w http.ResponseWriter, r *http.Request) {
		movies, err := q.ListMovies(r.Context(), db.ListMoviesParams{Limit: 10000, Offset: 0})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		f := excelize.NewFile()
		sheet := "Movies"
		f.SetSheetName("Sheet1", sheet)
		for i, h := range []string{"ID", "Title", "Description", "Genre", "Trailer URL"} {
			cell, _ := excelize.CoordinatesToCellName(i+1, 1)
			f.SetCellValue(sheet, cell, h)
		}
		for i, m := range movies {
			rowNum := i + 2
			desc := ""
			if m.Description.Valid {
				desc = m.Description.String
			}
			trailer := ""
			if m.TrailerUrl.Valid {
				trailer = m.TrailerUrl.String
			}
			f.SetCellValue(sheet, fmt.Sprintf("A%d", rowNum), m.MovieID)
			f.SetCellValue(sheet, fmt.Sprintf("B%d", rowNum), m.Title)
			f.SetCellValue(sheet, fmt.Sprintf("C%d", rowNum), desc)
			f.SetCellValue(sheet, fmt.Sprintf("D%d", rowNum), strings.Join(m.Genre, ", "))
			f.SetCellValue(sheet, fmt.Sprintf("E%d", rowNum), trailer)
		}
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", `attachment; filename="movies.xlsx"`)
		f.Write(w)
	})

	mux.HandleFunc("POST /api/movies/import", requireAdmin(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			http.Error(w, "invalid multipart form", http.StatusBadRequest)
			return
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "missing file field", http.StatusBadRequest)
			return
		}
		defer file.Close()
		f, err := excelize.OpenReader(file)
		if err != nil {
			http.Error(w, "invalid xlsx file", http.StatusBadRequest)
			return
		}
		sheets := f.GetSheetList()
		if len(sheets) == 0 {
			http.Error(w, "empty workbook", http.StatusBadRequest)
			return
		}
		xlRows, err := f.GetRows(sheets[0])
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		imported := 0
		for i, xlRow := range xlRows {
			if i == 0 {
				continue // skip header
			}
			if len(xlRow) < 2 || xlRow[1] == "" {
				continue
			}
			title := xlRow[1]
			desc := ""
			if len(xlRow) > 2 {
				desc = xlRow[2]
			}
			var genres []string
			if len(xlRow) > 3 && xlRow[3] != "" {
				for _, g := range strings.Split(xlRow[3], ",") {
					if t := strings.TrimSpace(g); t != "" {
						genres = append(genres, t)
					}
				}
			}
			trailer := ""
			if len(xlRow) > 4 {
				trailer = xlRow[4]
			}
			_, err := q.CreateMovie(r.Context(), db.CreateMovieParams{
				Title:       title,
				Description: pgtype.Text{String: desc, Valid: desc != ""},
				Genre:       genres,
				TrailerUrl:  pgtype.Text{String: trailer, Valid: trailer != ""},
			})
			if err != nil {
				http.Error(w, fmt.Sprintf("row %d: %v", i+1, err), http.StatusInternalServerError)
				return
			}
			imported++
		}
		writeJSON(w, http.StatusOK, map[string]int{"imported": imported})
	}))

	// Static files (React SPA — fallback to index.html for client-side routes)
	staticFS := http.Dir("./static")
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		f, err := staticFS.Open(r.URL.Path)
		if err != nil {
			http.ServeFile(w, r, "./static/index.html")
			return
		}
		f.Close()
		http.FileServer(staticFS).ServeHTTP(w, r)
	})

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

var jwtSecret = []byte(func() string {
	if s := os.Getenv("JWT_SECRET"); s != "" {
		return s
	}
	return "cinema-secret-key"
}())

type ctxKey string

const (
	ctxUserID    ctxKey = "user_id"
	ctxPermLevel ctxKey = "perm_level"
)

type authClaims struct {
	UserID    int32 `json:"user_id"`
	PermLevel int32 `json:"perm_level"`
	jwt.RegisteredClaims
}

func generateToken(userID, permLevel int32) (string, error) {
	claims := authClaims{
		UserID:    userID,
		PermLevel: permLevel,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(jwtSecret)
}

func parseToken(tokenStr string) (*authClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &authClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*authClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}

func requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenStr := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if tokenStr == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		claims, err := parseToken(tokenStr)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), ctxUserID, claims.UserID)
		ctx = context.WithValue(ctx, ctxPermLevel, claims.PermLevel)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return requireAuth(func(w http.ResponseWriter, r *http.Request) {
		level, _ := r.Context().Value(ctxPermLevel).(int32)
		if level != 1 {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func randomState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func hashPassword(password string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return ""
	}
	return string(hash)
}

func checkPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
