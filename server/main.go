package main

import (
	"log"
	"net"
	"net/http"

	"VK_golang_hw/docs"
	moviehttp "VK_golang_hw/handler/http"
	moviegrpc "VK_golang_hw/handler/moviegrpc"
	"VK_golang_hw/movie"
	moviepb "VK_golang_hw/proto"

	"github.com/swaggest/swgui/v5emb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	cfg := movie.LoadConfig()
	client := movie.NewClient(cfg.MovieBaseURL)

	// HTTP
	h := moviehttp.NewHandler(client)
	mux := http.NewServeMux()
	h.Register(mux)
	registerSwagger(mux)

	errCh := make(chan error, 2)

	go func() {
		log.Println("HTTP on", cfg.HTTPAddr)
		errCh <- http.ListenAndServe(cfg.HTTPAddr, mux)
	}()

	// gRPC
	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		// Не роняем сервис: HTTP-часть должна продолжать работать.
		log.Printf("gRPC disabled: listen %s: %v", cfg.GRPCAddr, err)
	} else {
		s := grpc.NewServer()
		moviepb.RegisterMovieServiceServer(s, moviegrpc.NewService(client))
		reflection.Register(s)

		log.Println("gRPC on", cfg.GRPCAddr)
		go func() { errCh <- s.Serve(lis) }()
	}

	log.Fatal(<-errCh)
}

func registerSwagger(mux *http.ServeMux) {
	ui := v5emb.New(
		"Movie Service",
		"/swagger/doc.json",
		"/swagger/",
	)

	mux.Handle("/swagger/", ui)
	mux.HandleFunc("GET /swagger/doc.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(docs.SwaggerJSON())
	})
	// swgui отдаёт UI только на своём base path, поэтому /swagger/index.html
	// (адрес из требований) подменяем на /swagger/.
	mux.HandleFunc("GET /swagger/index.html", func(w http.ResponseWriter, r *http.Request) {
		r = r.Clone(r.Context())
		r.URL.Path = "/swagger/"
		ui.ServeHTTP(w, r)
	})
}
