package stats

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

type Server struct {
	db     *DB
	server *http.Server
}

func NewServer(db *DB, addr string) *Server {
	s := &Server{db: db}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/stats", s.cors(s.handleStats))
	mux.HandleFunc("/api/stats/overview", s.cors(s.handleOverview))
	mux.HandleFunc("/api/stats/model-dashboard", s.cors(s.handleStats))
	mux.HandleFunc("/api/stats/costs", s.cors(s.handleCosts))
	mux.HandleFunc("/api/stats/behavior", s.cors(s.handleBehavior))
	mux.HandleFunc("/api/stats/recent", s.cors(s.handleRecent))
	mux.HandleFunc("/api/stats/errors", s.cors(s.handleErrors))
	mux.HandleFunc("/api/stats/models", s.cors(s.handleModels))
	mux.HandleFunc("/api/stats/folders", s.cors(s.handleFolders))
	mux.HandleFunc("/api/stats/timeseries", s.cors(s.handleTimeSeries))
	mux.HandleFunc("/api/request/", s.cors(s.handleRequest))
	mux.HandleFunc("/api/sync", s.cors(s.handleSync))
	s.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	return s
}

func (s *Server) Start() error {
	log.Printf("stats server listening on %s", s.server.Addr)
	return s.server.ListenAndServe()
}

func (s *Server) Close() error {
	return s.server.Close()
}

func (s *Server) cors(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

func parseRange(r *http.Request) int64 {
	rangeStr := r.URL.Query().Get("range")
	if rangeStr == "" {
		return 0
	}
	hours, err := strconv.ParseFloat(rangeStr, 64)
	if err != nil || hours <= 0 {
		return 0
	}
	return time.Now().UnixMilli() - int64(hours*3600*1000)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	cutoff := parseRange(r)
	dash := DashboardStats{
		Overall:    s.db.OverallStats(cutoff),
		ByModel:    s.db.StatsByModel(cutoff),
		ByFolder:   s.db.StatsByFolder(cutoff),
		TimeSeries: s.db.TimeSeries(cutoff, nil, 3600000),
	}
	writeJSON(w, dash)
}

func (s *Server) handleOverview(w http.ResponseWriter, r *http.Request) {
	cutoff := parseRange(r)
	writeJSON(w, s.db.OverallStats(cutoff))
}

func (s *Server) handleCosts(w http.ResponseWriter, r *http.Request) {
	cutoff := parseRange(r)
	_ = cutoff
	writeJSON(w, []any{})
}

func (s *Server) handleBehavior(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, BehaviorDashboardStats{})
}

func (s *Server) handleRecent(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}
	writeJSON(w, s.db.RecentRequests(limit))
}

func (s *Server) handleErrors(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}
	recent := s.db.RecentRequests(limit)
	var errors []MessageStats
	for _, m := range recent {
		if m.StopReason == "error" {
			errors = append(errors, m)
		}
	}
	writeJSON(w, errors)
}

func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	cutoff := parseRange(r)
	writeJSON(w, s.db.StatsByModel(cutoff))
}

func (s *Server) handleFolders(w http.ResponseWriter, r *http.Request) {
	cutoff := parseRange(r)
	writeJSON(w, s.db.StatsByFolder(cutoff))
}

func (s *Server) handleTimeSeries(w http.ResponseWriter, r *http.Request) {
	cutoff := parseRange(r)
	writeJSON(w, s.db.TimeSeries(cutoff, nil, 3600000))
}

func (s *Server) handleRequest(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/api/request/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	recent := s.db.RecentRequests(1000)
	for _, m := range recent {
		if m.ID == id {
			writeJSON(w, m)
			return
		}
	}
	http.Error(w, "Not Found", http.StatusNotFound)
}

func (s *Server) handleSync(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{
		"synced":       0,
		"totalMessages": s.db.MessageCount(),
	})
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.server.Handler.ServeHTTP(w, r)
}

func StartStatsServer(dbPath, addr string) error {
	db, err := OpenDB(dbPath)
	if err != nil {
		return fmt.Errorf("stats db: %w", err)
	}
	defer db.Close()

	srv := NewServer(db, addr)
	return srv.Start()
}
