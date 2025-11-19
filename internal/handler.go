// Package internal provides HTTP handlers and supporting logic for the Movies API.
//
// @title           Robin Camp Movies API
// @version         1.0.0
// @description     Movies catalogue and rating API with box office enrichment.
//
// @BasePath        /
//
// @securityDefinitions.apikey BearerAuth
// @in              header
// @name            Authorization
// @description     Provide token as: Bearer <token>
//
// @securityDefinitions.apikey RaterId
// @in              header
// @name            X-Rater-Id
package internal

import (
	"Robin-Camp/internal/boxoffice"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/route"
)

// Handler bundles dependencies for HTTP handlers.
type Handler struct {
	db        *sql.DB
	boxClient BoxOfficeClient
	authToken string

	mu sync.RWMutex
	// pendingMovies  []*Movie
	// pendingRatings []RatingResult
}

// BoxOfficeClient captures the upstream client contract.
type BoxOfficeClient interface {
	GetMovieBoxOffice(ctx context.Context, title string) (*boxoffice.BoxOffice, error)
}

func NewHandler(db *sql.DB, boxClient BoxOfficeClient, authToken string) *Handler {
	h := &Handler{db: db, boxClient: boxClient, authToken: authToken}
	return h
}

// func (h *Handler) backgroundFlusher() {
// 	ticker := time.NewTicker(2 * time.Second)
// 	defer ticker.Stop()
// 	for range ticker.C {
// 		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 		h.flushPending(ctx)
// 		cancel()
// 	}
// }

// func (h *Handler) flushPending(ctx context.Context) {
// 	h.mu.Lock()
// 	movies := h.pendingMovies
// 	ratings := h.pendingRatings
// 	h.pendingMovies = nil
// 	h.pendingRatings = nil
// 	h.mu.Unlock()

// 	if len(movies) == 0 && len(ratings) == 0 {
// 		return
// 	}

// 	_ = WithTx(ctx, h.db, func(ctx context.Context, tx *sql.Tx) error {
// 		for _, m := range movies {
// 			if err := upsertMovie(ctx, tx, m); err != nil {
// 				return err
// 			}
// 		}
// 		for _, r := range ratings {
// 			if err := upsertRating(ctx, tx, r); err != nil {
// 				return err
// 			}
// 		}
// 		return nil
// 	})
// }

func upsertMovie(ctx context.Context, tx *sql.Tx, m *Movie) error {
	_, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO movies (id, title, release_date, genre, distributor, budget, mpa_rating, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, (STRFTIME('%Y-%m-%dT%H:%M:%fZ', 'now')), (STRFTIME('%Y-%m-%dT%H:%M:%fZ', 'now')))`,
		m.ID, m.Title, m.ReleaseDate, m.Genre, m.Distributor, m.Budget, m.MpaRating,
	)
	if err != nil {
		return fmt.Errorf("insert movie: %w", err)
	}
	if m.BoxOffice != nil {
		var worldwidePtr *int64
		if m.BoxOffice.Revenue.Worldwide != 0 {
			v := m.BoxOffice.Revenue.Worldwide
			worldwidePtr = &v
		}
		_, err = tx.ExecContext(ctx, `INSERT OR REPLACE INTO box_office (movie_id, currency, source, last_updated, revenue_worldwide, revenue_opening_weekend_usa) VALUES (?, ?, ?, ?, ?, ?)`,
			m.ID, m.BoxOffice.Currency, m.BoxOffice.Source, m.BoxOffice.LastUpdated,
			valueOrZero(worldwidePtr), valueOrZero(m.BoxOffice.Revenue.OpeningWeekendUsa),
		)
		if err != nil {
			return fmt.Errorf("upsert box_office: %w", err)
		}
	}
	return nil
}

func valueOrZero(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}

func upsertRating(ctx context.Context, tx *sql.Tx, r RatingResult) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO ratings (movie_id, rater_id, rating, updated_at) VALUES ((SELECT id FROM movies WHERE title = ?), ?, ?, (STRFTIME('%Y-%m-%dT%H:%M:%fZ', 'now'))) ON CONFLICT(movie_id, rater_id) DO UPDATE SET rating = excluded.rating, updated_at = excluded.updated_at`,
		r.MovieTitle, r.RaterID, r.Rating,
	)
	if err != nil {
		return fmt.Errorf("upsert rating: %w", err)
	}
	return nil
}

// listMovies godoc
// @Summary      List and search movies
// @Description  Returns a paginated list of movies, optionally filtered by query, year, genre and cursor.
// @Tags         Movies
// @Accept       json
// @Produce      json
// @Param        q           query     string  false  "Search query for movie title substring"
// @Param        year        query     string  false  "Release year (YYYY) derived from releaseDate"
// @Param        genre       query     string  false  "Exact genre filter"
// @Param        limit       query     int     false  "Maximum number of items to return (default 20)"
// @Param        cursor      query     string  false  "Pagination cursor from previous page's nextCursor"
// @Success      200         {object}  MoviePage
// @Failure      500         {object}  Error  "Internal server error"
// @Router       /movies [get]
func (h *Handler) listMovies(ctx context.Context, c *app.RequestContext) {
	q := c.Query("q")
	year := c.Query("year")
	genre := c.Query("genre")
	limitStr := c.Query("limit")
	cursor := c.Query("cursor")

	limit := 20
	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 {
			limit = v
		}
	}

	movies, nextCursor, err := listMoviesFromDB(ctx, h.db, q, year, genre, limit, cursor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Error{Code: "INTERNAL", Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, MoviePage{Items: movies, NextCursor: nextCursor})
}

func listMoviesFromDB(ctx context.Context, db *sql.DB, q, year, genre string, limit int, cursor string) ([]Movie, *string, error) {
	var args []any
	var where []string

	if q != "" {
		where = append(where, "title LIKE ?")
		args = append(args, "%"+q+"%")
	}
	if year != "" {
		where = append(where, "substr(release_date,1,4) = ?")
		args = append(args, year)
	}
	if genre != "" {
		where = append(where, "genre = ?")
		args = append(args, genre)
	}
	if cursor != "" {
		where = append(where, "id > ?")
		args = append(args, cursor)
	}

	query := `SELECT m.id, m.title, m.release_date, m.genre, m.distributor, m.budget, m.mpa_rating, b.currency, b.source, b.last_updated, b.revenue_worldwide, b.revenue_opening_weekend_usa FROM movies m LEFT JOIN box_office b ON m.id = b.movie_id`
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY m.id LIMIT ?"
	args = append(args, limit+1)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var res []Movie
	var lastID string
	for rows.Next() {
		var m Movie
		var currency, source, lastUpdated sql.NullString
		var revenueWorldwide, revenueOpeningWeekend sql.NullInt64

		if err := rows.Scan(&m.ID, &m.Title, &m.ReleaseDate, &m.Genre, &m.Distributor, &m.Budget, &m.MpaRating, &currency, &source, &lastUpdated, &revenueWorldwide, &revenueOpeningWeekend); err != nil {
			return nil, nil, err
		}
		lastID = m.ID

		// Populate BoxOffice if we have any box office data; otherwise leave as nil (serializes as null).
		if currency.Valid || source.Valid || lastUpdated.Valid || revenueWorldwide.Valid || revenueOpeningWeekend.Valid {
			bo := &BoxOffice{}
			if revenueWorldwide.Valid {
				bo.Revenue.Worldwide = revenueWorldwide.Int64
			}
			if revenueOpeningWeekend.Valid {
				val := revenueOpeningWeekend.Int64
				bo.Revenue.OpeningWeekendUsa = &val
			}
			if currency.Valid {
				bo.Currency = currency.String
			}
			if source.Valid {
				bo.Source = source.String
			}
			if lastUpdated.Valid {
				bo.LastUpdated = lastUpdated.String
			}
			m.BoxOffice = bo
		}

		res = append(res, m)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	var nextCursor *string
	if len(res) > limit {
		res = res[:limit]
		nextCursor = &lastID
	}
	return res, nextCursor, nil
}

// createMovie godoc
// @Summary      Create a new movie
// @Description  Creates a new movie and synchronously enriches it with box office data when available.
// @Tags         Movies
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        movie       body      MovieCreate  true   "Movie to create"
// @Success      201         {object}  Movie
// @Header       201         {string}  Location     "Location of the newly created movie resource"
// @Failure      400         {object}  Error        "Bad request"
// @Failure      401         {object}  Error        "Unauthorized"
// @Failure      422         {object}  Error        "Unprocessable entity (validation or invalid JSON)"
// @Failure      500         {object}  Error        "Internal server error"
// @Router       /movies [post]
func (h *Handler) createMovie(ctx context.Context, c *app.RequestContext) {
	var payload MovieCreate
	if err := c.Bind(&payload); err != nil {
		// Invalid JSON/body format -> 422 Unprocessable Entity (per assignment tests)
		c.JSON(http.StatusUnprocessableEntity, Error{Code: "BAD_REQUEST", Message: "Invalid request body"})
		return
	}
	// Validate required fields -> 422 Unprocessable Entity
	if strings.TrimSpace(payload.Title) == "" || strings.TrimSpace(payload.Genre) == "" || strings.TrimSpace(payload.ReleaseDate) == "" {
		c.JSON(http.StatusUnprocessableEntity, Error{Code: "BAD_REQUEST", Message: "title, genre and releaseDate are required"})
		return
	}
	// Basic releaseDate format validation (YYYY-MM-DD) -> 422 on semantic error
	if len(payload.ReleaseDate) != 10 || payload.ReleaseDate[4] != '-' || payload.ReleaseDate[7] != '-' {
		c.JSON(http.StatusUnprocessableEntity, Error{Code: "BAD_REQUEST", Message: "invalid releaseDate format"})
		return
	}

	id := Now()
	var box *BoxOffice

	if h.boxClient != nil {
		bo, err := h.boxClient.GetMovieBoxOffice(ctx, payload.Title)
		if err != nil && !errors.Is(err, boxoffice.ErrNotFound) {
			// ignore upstream errors except 404
			bo = nil
		}
		if bo != nil {
			var worldwide int64
			if bo.Revenue.Worldwide != nil {
				worldwide = *bo.Revenue.Worldwide
			}
			box = &BoxOffice{
				Revenue: Revenue{
					Worldwide:         worldwide,
					OpeningWeekendUsa: bo.Revenue.OpeningWeekendUSA,
				},
				Currency:    bo.Currency,
				Source:      bo.Source,
				LastUpdated: bo.LastUpdated,
			}
		}
	}

	movie := &Movie{
		ID:          id,
		Title:       payload.Title,
		ReleaseDate: payload.ReleaseDate,
		Genre:       payload.Genre,
		Distributor: payload.Distributor,
		Budget:      payload.Budget,
		MpaRating:   payload.MpaRating,
		BoxOffice:   box,
	}

	// Write-through to DB so that subsequent GET /movies sees the new movie immediately.
	if err := WithTx(ctx, h.db, func(ctx context.Context, tx *sql.Tx) error {
		return upsertMovie(ctx, tx, movie)
	}); err != nil {
		c.JSON(http.StatusInternalServerError, Error{Code: "INTERNAL", Message: err.Error()})
		return
	}

	c.Header("Location", "/movies/"+payload.Title)
	c.JSON(http.StatusCreated, movie)
}

// submitRating godoc
// @Summary      Submit or update a rating for a movie
// @Description  Submits a rating for the given movie title. If the rater has already rated this movie, the rating is updated.
// @Tags         Ratings
// @Accept       json
// @Produce      json
// @Security     RaterId
// @Param        title       path      string        true   "Movie title"
// @Param        X-Rater-Id  header    string        true   "Unique rater identifier"
// @Param        rating      body      RatingSubmit  true   "Rating payload (0.5-5.0 in 0.5 steps)"
// @Success      201         {object}  RatingResult  "Rating created"
// @Header       201         {string}  Location      "Location of the rating resource when created"
// @Success      200         {object}  RatingResult  "Rating updated"
// @Failure      400         {object}  Error         "Bad request (invalid body or rating)"
// @Failure      401         {object}  Error         "Unauthorized (missing or invalid X-Rater-Id)"
// @Failure      404         {object}  Error         "Movie not found"
// @Failure      500         {object}  Error         "Internal server error"
// @Router       /movies/{title}/ratings [post]
func (h *Handler) submitRating(ctx context.Context, c *app.RequestContext) {
	title := c.Param("title")
	if len(title) == 0 {
		c.JSON(http.StatusBadRequest, Error{Code: "BAD_REQUEST", Message: "missing title"})
		return
	}

	// Normalize title to avoid subtle mismatches and reuse consistently
	normalizedTitle := strings.TrimSpace(title)

	raterID := string(c.GetHeader("X-Rater-Id"))
	if raterID == "" {
		c.JSON(http.StatusUnauthorized, Error{Code: "UNAUTHORIZED", Message: "Missing or invalid authentication信息"})
		return
	}

	var payload RatingSubmit
	if err := c.Bind(&payload); err != nil {
		c.JSON(http.StatusBadRequest, Error{Code: "BAD_REQUEST", Message: "Invalid request body"})
		return
	}
	// Enforce allowed rating set: 0.5,1.0,...,5.0 (step 0.5) -> 422 on semantic validation failure
	if payload.Rating < 0.5 || payload.Rating > 5.0 || math.Mod(payload.Rating*2, 1) != 0 {
		c.JSON(http.StatusUnprocessableEntity, Error{Code: "BAD_REQUEST", Message: "rating out of range"})
		return
	}

	// Ensure movie exists and get its id so we can distinguish 404 movie-not-found separately.
	var movieID string
	if err := h.db.QueryRowContext(ctx, `SELECT id FROM movies WHERE title = ?`, normalizedTitle).Scan(&movieID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, Error{Code: "NOT_FOUND", Message: "movie not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, Error{Code: "INTERNAL", Message: err.Error()})
		return
	}

	// Check if this rater already has a rating for the movie to decide between 201 (create) and 200 (update).
	var existingCount int
	if err := h.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM ratings WHERE movie_id = ? AND rater_id = ?`, movieID, raterID).Scan(&existingCount); err != nil {
		c.JSON(http.StatusInternalServerError, Error{Code: "INTERNAL", Message: err.Error()})
		return
	}

	statusCode := http.StatusCreated
	if existingCount > 0 {
		statusCode = http.StatusOK
	}

	// Directly persist rating so reads see it immediately.
	if err := WithTx(ctx, h.db, func(ctx context.Context, tx *sql.Tx) error {
		return upsertRating(ctx, tx, RatingResult{
			MovieTitle: normalizedTitle,
			RaterID:    raterID,
			Rating:     payload.Rating,
		})
	}); err != nil {
		c.JSON(http.StatusInternalServerError, Error{Code: "INTERNAL", Message: err.Error()})
		return
	}

	// Prepare response
	res := RatingResult{
		MovieTitle: normalizedTitle,
		RaterID:    raterID,
		Rating:     payload.Rating,
	}

	// Set Location header only when a new rating is created
	if statusCode == http.StatusCreated {
		c.Header("Location", "/movies/"+normalizedTitle+"/ratings/"+raterID)
	}

	c.JSON(statusCode, res)
}

// getRatingAggregate godoc
// @Summary      Get rating aggregate for a movie
// @Description  Returns the average rating (rounded to one decimal) and count of ratings for the given movie.
// @Tags         Ratings
// @Accept       json
// @Produce      json
// @Param        title   path      string  true  "Movie title"
// @Success      200     {object}  RatingAggregate
// @Failure      404     {object}  Error  "Movie not found or no ratings yet"
// @Failure      500     {object}  Error  "Internal server error"
// @Router       /movies/{title}/rating [get]
func (h *Handler) getRatingAggregate(ctx context.Context, c *app.RequestContext) {
	title := c.Param("title")
	if len(title) == 0 {
		c.JSON(http.StatusBadRequest, Error{Code: "BAD_REQUEST", Message: "missing title"})
		return
	}

	var movieID string
	if err := h.db.QueryRowContext(ctx, `SELECT id FROM movies WHERE title = ?`, string(title)).Scan(&movieID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, Error{Code: "NOT_FOUND", Message: "movie not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, Error{Code: "INTERNAL", Message: err.Error()})
		return
	}

	var avg sql.NullFloat64
	var count sql.NullInt64
	if err := h.db.QueryRowContext(ctx, `SELECT AVG(rating), COUNT(*) FROM ratings WHERE movie_id = ?`, movieID).Scan(&avg, &count); err != nil {
		c.JSON(http.StatusInternalServerError, Error{Code: "INTERNAL", Message: err.Error()})
		return
	}
	if !count.Valid || count.Int64 == 0 {
		c.JSON(http.StatusNotFound, Error{Code: "NOT_FOUND", Message: "no ratings"})
		return
	}

	avgRounded := math.Round(avg.Float64*10) / 10
	c.JSON(http.StatusOK, RatingAggregate{Average: avgRounded, Count: count.Int64})
}

// requireBearer wraps handlers that need Bearer auth for writes.
func (h *Handler) requireBearer(next app.HandlerFunc) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		raw := string(c.GetHeader("Authorization"))
		if h.authToken == "" {
			// If no auth token configured, treat as auth disabled (useful for local/dev),
			// but the e2e tests require a token, so only skip strict checks when authToken is empty.
			next(ctx, c)
			return
		}

		if strings.TrimSpace(raw) == "" {
			c.JSON(http.StatusUnauthorized, Error{Code: "UNAUTHORIZED", Message: "Missing or invalid authentication信息"})
			return
		}

		const prefix = "Bearer "
		if !strings.HasPrefix(raw, prefix) {
			c.JSON(http.StatusUnauthorized, Error{Code: "UNAUTHORIZED", Message: "Missing or invalid authentication信息"})
			return
		}

		token := strings.TrimSpace(strings.TrimPrefix(raw, prefix))
		if token != h.authToken {
			c.JSON(http.StatusUnauthorized, Error{Code: "UNAUTHORIZED", Message: "Missing or invalid authentication信息"})
			return
		}

		next(ctx, c)
	}
}

// requireRater enforces X-Rater-Id header for rating endpoints.
func (h *Handler) requireRater(next app.HandlerFunc) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		raterID := string(c.GetHeader("X-Rater-Id"))
		if raterID == "" {
			c.JSON(http.StatusUnauthorized, Error{Code: "UNAUTHORIZED", Message: "Missing or invalid authentication信息"})
			return
		}
		next(ctx, c)
	}
}

func (h *Handler) RegisterRoutes(rg *route.RouterGroup) {
	movies := rg.Group("/movies")
	{
		movies.GET("", h.listMovies)
		movies.POST("", h.requireBearer(h.createMovie))
		movies.GET("/:title/rating", h.getRatingAggregate)
		movies.POST("/:title/ratings", h.requireRater(h.submitRating))
	}

	// healthz godoc
	// @Summary   Health check
	// @Tags      System
	// @Produce   json
	// @Success   200  {object}  map[string]string  "Service is healthy"
	// @Router    /healthz [get]
	rg.GET("/healthz", func(ctx context.Context, c *app.RequestContext) {
		c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})
}
