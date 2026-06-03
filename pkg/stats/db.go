package stats

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

type DB struct {
	db *sql.DB
}

func OpenDB(path string) (*DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("stats db open: %w", err)
	}
	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		return nil, fmt.Errorf("stats db wal: %w", err)
	}
	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("stats db tables: %w", err)
	}
	return &DB{db: db}, nil
}

func (d *DB) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

func createTables(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_file TEXT NOT NULL,
			entry_id TEXT NOT NULL,
			folder TEXT NOT NULL,
			model TEXT NOT NULL,
			provider TEXT NOT NULL,
			api TEXT NOT NULL,
			timestamp INTEGER NOT NULL,
			duration INTEGER,
			ttft INTEGER,
			stop_reason TEXT NOT NULL,
			error_message TEXT,
			input_tokens INTEGER NOT NULL DEFAULT 0,
			output_tokens INTEGER NOT NULL DEFAULT 0,
			cache_read_tokens INTEGER NOT NULL DEFAULT 0,
			cache_write_tokens INTEGER NOT NULL DEFAULT 0,
			total_tokens INTEGER NOT NULL DEFAULT 0,
			cost_input REAL NOT NULL DEFAULT 0,
			cost_output REAL NOT NULL DEFAULT 0,
			cost_cache_read REAL NOT NULL DEFAULT 0,
			cost_cache_write REAL NOT NULL DEFAULT 0,
			cost_total REAL NOT NULL DEFAULT 0,
			UNIQUE(session_file, entry_id)
		);
		CREATE INDEX IF NOT EXISTS idx_messages_timestamp ON messages(timestamp);
		CREATE INDEX IF NOT EXISTS idx_messages_model ON messages(model);
		CREATE INDEX IF NOT EXISTS idx_messages_folder ON messages(folder);
		CREATE INDEX IF NOT EXISTS idx_messages_session ON messages(session_file);
		CREATE TABLE IF NOT EXISTS file_offsets (
			session_file TEXT PRIMARY KEY,
			offset_val INTEGER NOT NULL,
			last_modified INTEGER NOT NULL
		);
	`)
	return err
}

func (d *DB) InsertMessages(stats []MessageStats) (int, error) {
	if len(stats) == 0 {
		return 0, nil
	}
	tx, err := d.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT OR IGNORE INTO messages (
			session_file, entry_id, folder, model, provider, api, timestamp,
			duration, ttft, stop_reason, error_message,
			input_tokens, output_tokens, cache_read_tokens, cache_write_tokens, total_tokens,
			cost_input, cost_output, cost_cache_read, cost_cache_write, cost_total
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	inserted := 0
	for _, s := range stats {
		res, err := stmt.Exec(
			s.SessionFile, s.EntryID, s.Folder, s.Model, s.Provider, s.API, s.Timestamp,
			s.Duration, s.TTFT, s.StopReason, s.ErrorMessage,
			s.InputTokens, s.OutputTokens, s.CacheRead, s.CacheWrite, s.TotalTokens,
			s.CostInput, s.CostOutput, s.CostCacheRd, s.CostCacheWr, s.CostTotal,
		)
		if err != nil {
			return 0, err
		}
		n, _ := res.RowsAffected()
		if n > 0 {
			inserted++
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return inserted, nil
}

func buildAggregated(statsTotal, failed, input, output, cacheR, cacheW, premiumCost float64) AggregatedStats {
	total := int(statsTotal)
	fail := int(failed)
	rate := 0.0
	if total > 0 {
		rate = float64(fail) / float64(total)
	}
	cacheRate := 0.0
	if input+cacheR > 0 {
		cacheRate = cacheR / (input + cacheR)
	}
	return AggregatedStats{
		TotalRequests:      total,
		SuccessfulRequests: total - fail,
		FailedRequests:     fail,
		ErrorRate:          rate,
		TotalInputTokens:   int(input),
		TotalOutputTokens:  int(output),
		TotalCacheRead:     int(cacheR),
		TotalCacheWrite:    int(cacheW),
		CacheRate:          cacheRate,
	}
}

func (d *DB) OverallStats(cutoff int64) AggregatedStats {
	query := `SELECT 
		COALESCE(COUNT(*), 0) as total_requests,
		COALESCE(SUM(CASE WHEN stop_reason = 'error' THEN 1 ELSE 0 END), 0) as failed_requests,
		COALESCE(SUM(input_tokens), 0) as total_input,
		COALESCE(SUM(output_tokens), 0) as total_output,
		COALESCE(SUM(cache_read_tokens), 0) as total_cache_read,
		COALESCE(SUM(cache_write_tokens), 0) as total_cache_write,
		COALESCE(SUM(cost_total), 0) as total_cost,
		AVG(duration) as avg_dur,
		AVG(ttft) as avg_ttft,
		AVG(CASE WHEN COALESCE(duration, 0) > 0 THEN output_tokens * 1000.0 / duration ELSE NULL END) as avg_tps,
		COALESCE(MIN(timestamp), 0) as first_ts,
		COALESCE(MAX(timestamp), 0) as last_ts
	FROM messages`
	if cutoff > 0 {
		query += fmt.Sprintf(" WHERE timestamp >= %d", cutoff)
	}

	row := d.db.QueryRow(query)
	s := AggregatedStats{}
	var avgDur, avgTTFT, avgTPS sql.NullFloat64
	err := row.Scan(&s.TotalRequests, &s.FailedRequests,
		&s.TotalInputTokens, &s.TotalOutputTokens, &s.TotalCacheRead, &s.TotalCacheWrite,
		&s.TotalCost, &avgDur, &avgTTFT, &avgTPS, &s.FirstTimestamp, &s.LastTimestamp)
	if err != nil {
		return s
	}
	s.SuccessfulRequests = s.TotalRequests - s.FailedRequests
	if s.TotalRequests > 0 {
		s.ErrorRate = float64(s.FailedRequests) / float64(s.TotalRequests)
	}
	totalIn := float64(s.TotalInputTokens)
	if totalIn+float64(s.TotalCacheRead) > 0 {
		s.CacheRate = float64(s.TotalCacheRead) / (totalIn + float64(s.TotalCacheRead))
	}
	if avgDur.Valid {
		s.AvgDuration = &avgDur.Float64
	}
	if avgTTFT.Valid {
		s.AvgTTFT = &avgTTFT.Float64
	}
	if avgTPS.Valid {
		s.AvgTokensPerSec = &avgTPS.Float64
	}
	return s
}

func (d *DB) StatsByModel(cutoff int64) []ModelStats {
	query := `SELECT model, provider,
		COUNT(*) as total_requests,
		SUM(CASE WHEN stop_reason = 'error' THEN 1 ELSE 0 END) as failed,
		SUM(input_tokens) as inp, SUM(output_tokens) as out,
		SUM(cache_read_tokens) as cr, SUM(cache_write_tokens) as cw,
		SUM(cost_total) as cost,
		AVG(duration) as dur, AVG(ttft) as ttft,
		AVG(CASE WHEN COALESCE(duration, 0) > 0 THEN output_tokens * 1000.0 / duration ELSE NULL END) as tps,
		MIN(timestamp) as first, MAX(timestamp) as last
	FROM messages`
	if cutoff > 0 {
		query += fmt.Sprintf(" WHERE timestamp >= %d", cutoff)
	}
	query += " GROUP BY model, provider ORDER BY total_requests DESC"

	rows, err := d.db.Query(query)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var result []ModelStats
	for rows.Next() {
		var ms ModelStats
		var dur, ttft, tps sql.NullFloat64
		err := rows.Scan(&ms.Model, &ms.Provider,
			&ms.TotalRequests, &ms.FailedRequests,
			&ms.TotalInputTokens, &ms.TotalOutputTokens, &ms.TotalCacheRead, &ms.TotalCacheWrite,
			&ms.TotalCost, &dur, &ttft, &tps, &ms.FirstTimestamp, &ms.LastTimestamp)
		if err != nil {
			continue
		}
		ms.SuccessfulRequests = ms.TotalRequests - ms.FailedRequests
		if ms.TotalRequests > 0 {
			ms.ErrorRate = float64(ms.FailedRequests) / float64(ms.TotalRequests)
		}
		totalIn := float64(ms.TotalInputTokens)
		if totalIn+float64(ms.TotalCacheRead) > 0 {
			ms.CacheRate = float64(ms.TotalCacheRead) / (totalIn + float64(ms.TotalCacheRead))
		}
		if dur.Valid {
			ms.AvgDuration = &dur.Float64
		}
		if ttft.Valid {
			ms.AvgTTFT = &ttft.Float64
		}
		if tps.Valid {
			ms.AvgTokensPerSec = &tps.Float64
		}
		result = append(result, ms)
	}
	return result
}

func (d *DB) StatsByFolder(cutoff int64) []FolderStats {
	query := `SELECT folder,
		COUNT(*) as total,
		SUM(CASE WHEN stop_reason = 'error' THEN 1 ELSE 0 END) as failed,
		SUM(input_tokens) as inp, SUM(output_tokens) as out,
		SUM(cache_read_tokens) as cr, SUM(cache_write_tokens) as cw,
		SUM(cost_total) as cost,
		AVG(duration) as dur, AVG(ttft) as ttft,
		AVG(CASE WHEN COALESCE(duration, 0) > 0 THEN output_tokens * 1000.0 / duration ELSE NULL END) as tps,
		MIN(timestamp) as first, MAX(timestamp) as last
	FROM messages`
	if cutoff > 0 {
		query += fmt.Sprintf(" WHERE timestamp >= %d", cutoff)
	}
	query += " GROUP BY folder ORDER BY total DESC"

	rows, err := d.db.Query(query)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var result []FolderStats
	for rows.Next() {
		var fs FolderStats
		var dur, ttft, tps sql.NullFloat64
		err := rows.Scan(&fs.Folder,
			&fs.TotalRequests, &fs.FailedRequests,
			&fs.TotalInputTokens, &fs.TotalOutputTokens, &fs.TotalCacheRead, &fs.TotalCacheWrite,
			&fs.TotalCost, &dur, &ttft, &tps, &fs.FirstTimestamp, &fs.LastTimestamp)
		if err != nil {
			continue
		}
		fs.SuccessfulRequests = fs.TotalRequests - fs.FailedRequests
		if fs.TotalRequests > 0 {
			fs.ErrorRate = float64(fs.FailedRequests) / float64(fs.TotalRequests)
		}
		totalIn := float64(fs.TotalInputTokens)
		if totalIn+float64(fs.TotalCacheRead) > 0 {
			fs.CacheRate = float64(fs.TotalCacheRead) / (totalIn + float64(fs.TotalCacheRead))
		}
		if dur.Valid {
			fs.AvgDuration = &dur.Float64
		}
		if ttft.Valid {
			fs.AvgTTFT = &ttft.Float64
		}
		if tps.Valid {
			fs.AvgTokensPerSec = &tps.Float64
		}
		result = append(result, fs)
	}
	return result
}

func (d *DB) TimeSeries(cutoff int64, hours *int, bucketMs int64) []TimeSeriesPoint {
	where := ""
	args := []any{bucketMs, bucketMs}
	if cutoff > 0 {
		where = " WHERE timestamp >= ?"
		args = append(args, cutoff)
	} else if hours != nil {
		where = " WHERE timestamp >= ?"
		args = append(args, int64(*hours)*3600000)
	}

	query := fmt.Sprintf(`SELECT (timestamp / ?) * ? as bucket,
		COUNT(*) as requests,
		SUM(CASE WHEN stop_reason = 'error' THEN 1 ELSE 0 END) as errors,
		SUM(total_tokens) as tokens, SUM(cost_total) as cost
	FROM messages%s GROUP BY bucket ORDER BY bucket ASC`, where)

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var result []TimeSeriesPoint
	for rows.Next() {
		var p TimeSeriesPoint
		rows.Scan(&p.Timestamp, &p.Requests, &p.Errors, &p.Tokens, &p.Cost)
		result = append(result, p)
	}
	return result
}

func (d *DB) GetFileOffset(sessionFile string) (int, int, error) {
	row := d.db.QueryRow("SELECT COALESCE(offset_val, 0), COALESCE(last_modified, 0) FROM file_offsets WHERE session_file = ?", sessionFile)
	var offset, lmod int
	err := row.Scan(&offset, &lmod)
	if err == sql.ErrNoRows {
		return 0, 0, nil
	}
	return offset, lmod, err
}

func (d *DB) SetFileOffset(sessionFile string, offset, lastModified int) error {
	_, err := d.db.Exec("INSERT OR REPLACE INTO file_offsets (session_file, offset_val, last_modified) VALUES (?, ?, ?)",
		sessionFile, offset, lastModified)
	return err
}

func (d *DB) RecentRequests(limit int) []MessageStats {
	rows, err := d.db.Query(`SELECT session_file, entry_id, folder, model, provider, api, timestamp,
		duration, ttft, stop_reason, error_message,
		input_tokens, output_tokens, cache_read_tokens, cache_write_tokens, total_tokens,
		cost_input, cost_output, cost_cache_read, cost_cache_write, cost_total
	FROM messages ORDER BY timestamp DESC LIMIT ?`, limit)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var result []MessageStats
	for rows.Next() {
		var s MessageStats
		rows.Scan(&s.SessionFile, &s.EntryID, &s.Folder, &s.Model, &s.Provider, &s.API, &s.Timestamp,
			&s.Duration, &s.TTFT, &s.StopReason, &s.ErrorMessage,
			&s.InputTokens, &s.OutputTokens, &s.CacheRead, &s.CacheWrite, &s.TotalTokens,
			&s.CostInput, &s.CostOutput, &s.CostCacheRd, &s.CostCacheWr, &s.CostTotal)
		result = append(result, s)
	}
	return result
}

func (d *DB) MessageCount() int {
	row := d.db.QueryRow("SELECT COUNT(*) FROM messages")
	var n int
	row.Scan(&n)
	return n
}
