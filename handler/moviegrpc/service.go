package moviegrpc

import (
	"context"
	"errors"
	"strconv"

	"VK_golang_hw/movie"
	pb "VK_golang_hw/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Service struct {
	pb.UnimplementedMovieServiceServer
	client *movie.Client
}

func NewService(c *movie.Client) *Service {
	return &Service{client: c}
}

func (s *Service) LoadMovies(ctx context.Context, req *pb.LoadMoviesRequest) (*pb.LoadMoviesResponse, error) {
	if req.GetFrom() < 1 || req.GetTo() < req.GetFrom() {
		return nil, status.Error(codes.InvalidArgument, "invalid from/to")
	}

	ctx, cancel := context.WithTimeout(ctx, movie.RangeTimeout)
	defer cancel()

	var movies []*pb.Movie
	var partial []string

	for id := req.GetFrom(); id <= req.GetTo(); id++ {
		m, err := s.client.GetByID(ctx, int(id))
		if errors.Is(err, movie.ErrNotFound) {
			partial = append(partial, "movie "+strconv.FormatInt(id, 10)+" not found")
			continue
		}
		if errors.Is(err, movie.ErrUpstreamTimeout) {
			return nil, status.Error(codes.DeadlineExceeded, "movie stand timeout")
		}
		if err != nil {
			return nil, status.Errorf(codes.Unavailable, "movie stand error: %v", err)
		}
		movies = append(movies, toProto(m))
	}

	return &pb.LoadMoviesResponse{Movies: movies, PartialErrors: partial}, nil
}

func (s *Service) GetLastMovie(ctx context.Context, _ *pb.GetLastMovieRequest) (*pb.Movie, error) {
	ctx, cancel := context.WithTimeout(ctx, movie.RangeTimeout)
	defer cancel()

	m, err := s.client.FindLast(ctx)
	if errors.Is(err, movie.ErrNotFound) {
		return nil, status.Error(codes.NotFound, "no movies")
	}
	if errors.Is(err, movie.ErrUpstreamTimeout) {
		return nil, status.Error(codes.DeadlineExceeded, "movie stand timeout")
	}
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "movie stand error: %v", err)
	}
	return toProto(m), nil
}

func toProto(m *movie.Movie) *pb.Movie {
	return &pb.Movie{
		Id:       m.ID,
		Title:    m.Title,
		Year:     m.Year,
		Director: m.Director,
	}
}
