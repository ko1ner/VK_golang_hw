package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"
)

const baseURL = "https://homeworksite.site/%d/info.0.json"

type Movie struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	Year     int    `json:"year"`
	Director string `json:"director"`
}

type Result struct {
	ID    int
	Movie Movie
	Err   error
}

func checkFlagsErr(from, to, workers int, timeout time.Duration) error {
	passed := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) {
		passed[f.Name] = true
	})
	var missing []string
	for _, name := range []string{"from", "to"} {
		if !passed[name] {
			missing = append(missing, "--"+name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required flags: %s", strings.Join(missing, ", "))
	}
	if from <= 0 || to <= 0 {
		return fmt.Errorf("invalid range: from and to must be positive integers")
	}
	if from > to {
		return fmt.Errorf("invalid range: from must be less than or equal to to")
	}
	if workers <= 0 {
		return fmt.Errorf("invalid workers: must be a positive integer")
	}
	if timeout <= 0 {
		return fmt.Errorf("invalid timeout: must be a positive duration")
	}
	if flag.NArg() > 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(flag.Args(), " "))
	}
	return nil
}

func getFilm(ctx context.Context, timeout time.Duration, id int, client *http.Client) (Movie, error) {
	url := fmt.Sprintf(baseURL, id)

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Movie{}, fmt.Errorf("failed to create request: %w", err)
	}

	response, err := client.Do(request)
	if err != nil {
		return Movie{}, fmt.Errorf("request failed: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return Movie{}, fmt.Errorf("request failed: %d", response.StatusCode)
	}

	var movie Movie
	if err := json.NewDecoder(response.Body).Decode(&movie); err != nil {
		return Movie{}, fmt.Errorf("failed to decode: %w", err)
	}

	return movie, nil
}

func worker(ctx context.Context, client *http.Client, timeout time.Duration, jobs <-chan int, results chan<- Result, wg *sync.WaitGroup) {
	defer wg.Done()

	for id := range jobs {
		if ctx.Err() != nil {
			return
		}
		movie, err := getFilm(ctx, timeout, id, client)
		result := Result{
			ID:    id,
			Movie: movie,
			Err:   err,
		}
		select {
		case <-ctx.Done():
			return
		case results <- result:
		}
	}

}
func main() {
	from := flag.Int("from", 0, "Starting ID of the range (inclusive)")
	to := flag.Int("to", 0, "Ending ID of the range (inclusive)")
	workers := flag.Int("workers", 10, "Number of concurrent workers")
	timeout := flag.Duration("timeout", 5*time.Second, "Timeout for HTTP request")
	flag.Parse()

	if err := checkFlagsErr(*from, *to, *workers, *timeout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		flag.Usage()
		os.Exit(1)
	}

	filmsAmount := *to - *from + 1
	jobs := make(chan int, filmsAmount)
	results := make(chan Result, filmsAmount)
	var wg sync.WaitGroup

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	client := &http.Client{}

	for range *workers {
		wg.Add(1)
		go worker(ctx, client, *timeout, jobs, results, &wg)
	}
	for ID := *from; ID <= *to; ID++ {
		jobs <- ID
	}
	close(jobs)
	wg.Wait()
	close(results)
	for res := range results {
		if res.Err != nil {
			if errors.Is(res.Err, context.Canceled) {
				continue
			}
			fmt.Fprintf(os.Stderr, "film %d: %v\n", res.ID, res.Err)
			continue
		}
		fmt.Printf("%d — %s — %d — %s\n", res.Movie.ID, res.Movie.Title, res.Movie.Year, res.Movie.Director)
	}
	if ctx.Err() != nil {
		fmt.Fprintln(os.Stderr, "interrupted: remaining requests canceled")
	}

}
