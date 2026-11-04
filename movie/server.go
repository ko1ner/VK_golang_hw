package movie

import (
	"os"
)

const (
	DefMovieBaseURL = "https://homeworksite.site"
	DefHTTPAddr     = ":8080"
	DefGRPCAddr     = ":9090"
)

type Config struct {
	MovieBaseURL string
	HTTPAddr     string
	GRPCAddr     string
}

type Movie struct {
	ID       int64  `json:"id"`
	Title    string `json:"title"`
	Year     int32  `json:"year"`
	Director string `json:"director"`
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func LoadConfig() Config {
	return Config{
		MovieBaseURL: getEnv("MOVIE_BASE_URL", DefMovieBaseURL),
		HTTPAddr:     getEnv("HTTP_ADDR", DefHTTPAddr),
		GRPCAddr:     getEnv("GRPC_ADDR", DefGRPCAddr),
	}
}
