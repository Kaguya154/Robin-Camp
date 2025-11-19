package internal

type MovieCreate struct {
	Title       string  `json:"title"`
	Genre       string  `json:"genre"`
	ReleaseDate string  `json:"releaseDate"`
	Distributor *string `json:"distributor,omitempty"`
	Budget      *int64  `json:"budget,omitempty"`
	MpaRating   *string `json:"mpaRating,omitempty"`
}

type BoxOffice struct {
	Revenue     Revenue `json:"revenue"`
	Currency    string  `json:"currency"`
	Source      string  `json:"source"`
	LastUpdated string  `json:"lastUpdated"`
}

type Revenue struct {
	Worldwide         int64  `json:"worldwide"`
	OpeningWeekendUsa *int64 `json:"openingWeekendUSA,omitempty"`
}

type Movie struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	ReleaseDate string     `json:"releaseDate"`
	Genre       string     `json:"genre"`
	Distributor *string    `json:"distributor,omitempty"`
	Budget      *int64     `json:"budget,omitempty"`
	MpaRating   *string    `json:"mpaRating,omitempty"`
	BoxOffice   *BoxOffice `json:"boxOffice"`
}

type RatingSubmit struct {
	Rating float64 `json:"rating"`
}

type RatingResult struct {
	MovieTitle string  `json:"movieTitle"`
	RaterID    string  `json:"raterId"`
	Rating     float64 `json:"rating"`
}

type RatingAggregate struct {
	Average float64 `json:"average"`
	Count   int64   `json:"count"`
}

type MoviePage struct {
	Items      []Movie `json:"items"`
	NextCursor *string `json:"nextCursor,omitempty"`
}

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}
