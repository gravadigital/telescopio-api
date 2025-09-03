package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"gorm.io/gorm"

	"github.com/gravadigital/telescopio-api/internal/logger"
)

// QueryOptimizer provides database query optimization utilities
type QueryOptimizer struct {
	db  *gorm.DB
	log *log.Logger
}

// NewQueryOptimizer creates a new query optimizer
func NewQueryOptimizer(db *gorm.DB) *QueryOptimizer {
	return &QueryOptimizer{
		db:  db,
		log: logger.Repository("query_optimizer"),
	}
}

// OptimizationHint represents a database optimization hint
type OptimizationHint struct {
	Query      string    `json:"query"`
	Table      string    `json:"table"`
	Operation  string    `json:"operation"`
	Suggestion string    `json:"suggestion"`
	Impact     string    `json:"impact"`
	Priority   string    `json:"priority"`
	CreatedAt  time.Time `json:"created_at"`
}

// PerformanceMetrics holds query performance metrics
type PerformanceMetrics struct {
	SlowQueries     []SlowQuery        `json:"slow_queries"`
	IndexUsage      []IndexUsage       `json:"index_usage"`
	TableStats      []TableStats       `json:"table_stats"`
	ConnectionStats ConnectionStats    `json:"connection_stats"`
	Suggestions     []OptimizationHint `json:"suggestions"`
}

// SlowQuery represents a slow-performing query
type SlowQuery struct {
	Query        string        `json:"query"`
	Duration     time.Duration `json:"duration"`
	Calls        int           `json:"calls"`
	MeanTime     time.Duration `json:"mean_time"`
	LastExecuted time.Time     `json:"last_executed"`
}

// IndexUsage represents index usage statistics
type IndexUsage struct {
	SchemaName string  `json:"schema_name"`
	TableName  string  `json:"table_name"`
	IndexName  string  `json:"index_name"`
	IndexUsed  int64   `json:"index_used"`
	TableScans int64   `json:"table_scans"`
	Efficiency float64 `json:"efficiency"`
}

// TableStats represents table statistics
type TableStats struct {
	TableName    string     `json:"table_name"`
	RowCount     int64      `json:"row_count"`
	TableSize    string     `json:"table_size"`
	IndexSize    string     `json:"index_size"`
	LastAnalyzed *time.Time `json:"last_analyzed"`
	VacuumCount  int64      `json:"vacuum_count"`
	AutoVacuum   bool       `json:"auto_vacuum"`
}

// ConnectionStats represents connection statistics
type ConnectionStats struct {
	TotalConnections   int     `json:"total_connections"`
	ActiveConnections  int     `json:"active_connections"`
	IdleConnections    int     `json:"idle_connections"`
	MaxConnections     int     `json:"max_connections"`
	ConnectionsPercent float64 `json:"connections_percent"`
}

// AnalyzePerformance analyzes database performance and returns metrics
func (o *QueryOptimizer) AnalyzePerformance(ctx context.Context) (*PerformanceMetrics, error) {
	o.log.Debug("Analyzing database performance...")

	metrics := &PerformanceMetrics{}

	// Get slow queries (PostgreSQL specific)
	slowQueries, err := o.getSlowQueries(ctx)
	if err != nil {
		o.log.Warn("Failed to get slow queries", "error", err)
	} else {
		metrics.SlowQueries = slowQueries
	}

	// Get index usage statistics
	indexUsage, err := o.getIndexUsage(ctx)
	if err != nil {
		o.log.Warn("Failed to get index usage", "error", err)
	} else {
		metrics.IndexUsage = indexUsage
	}

	// Get table statistics
	tableStats, err := o.getTableStats(ctx)
	if err != nil {
		o.log.Warn("Failed to get table stats", "error", err)
	} else {
		metrics.TableStats = tableStats
	}

	// Get connection statistics
	connStats, err := o.getConnectionStats(ctx)
	if err != nil {
		o.log.Warn("Failed to get connection stats", "error", err)
	} else {
		metrics.ConnectionStats = *connStats
	}

	// Generate optimization suggestions
	suggestions := o.generateOptimizationHints(metrics)
	metrics.Suggestions = suggestions

	o.log.Info("Database performance analysis completed",
		"slow_queries", len(metrics.SlowQueries),
		"indexes_analyzed", len(metrics.IndexUsage),
		"tables_analyzed", len(metrics.TableStats),
		"suggestions", len(metrics.Suggestions))

	return metrics, nil
}

// getSlowQueries retrieves slow query information
func (o *QueryOptimizer) getSlowQueries(ctx context.Context) ([]SlowQuery, error) {
	var queries []SlowQuery

	// PostgreSQL pg_stat_statements query (if extension is available)
	rows, err := o.db.WithContext(ctx).Raw(`
		SELECT 
			query,
			total_exec_time / 1000 as total_duration_ms,
			calls,
			mean_exec_time / 1000 as mean_duration_ms
		FROM pg_stat_statements 
		WHERE calls > 10 
		AND mean_exec_time > 100
		ORDER BY mean_exec_time DESC 
		LIMIT 20
	`).Rows()

	if err != nil {
		// pg_stat_statements might not be available
		o.log.Debug("pg_stat_statements not available, skipping slow query analysis")
		return queries, nil
	}
	defer rows.Close()

	for rows.Next() {
		var q SlowQuery
		var totalDurationMs, meanDurationMs float64

		if err := rows.Scan(&q.Query, &totalDurationMs, &q.Calls, &meanDurationMs); err != nil {
			continue
		}

		q.Duration = time.Duration(totalDurationMs) * time.Millisecond
		q.MeanTime = time.Duration(meanDurationMs) * time.Millisecond
		queries = append(queries, q)
	}

	return queries, nil
}

// getIndexUsage retrieves index usage statistics
func (o *QueryOptimizer) getIndexUsage(ctx context.Context) ([]IndexUsage, error) {
	var usage []IndexUsage

	rows, err := o.db.WithContext(ctx).Raw(`
		SELECT 
			schemaname,
			tablename,
			indexname,
			idx_scan as index_used,
			seq_scan as table_scans,
			CASE 
				WHEN idx_scan + seq_scan = 0 THEN 0 
				ELSE ROUND((idx_scan::float / (idx_scan + seq_scan)::float) * 100, 2)
			END as efficiency
		FROM pg_stat_user_indexes pui
		JOIN pg_stat_user_tables put ON pui.relid = put.relid
		ORDER BY efficiency ASC, index_used DESC
	`).Rows()

	if err != nil {
		return usage, err
	}
	defer rows.Close()

	for rows.Next() {
		var u IndexUsage
		if err := rows.Scan(&u.SchemaName, &u.TableName, &u.IndexName, &u.IndexUsed, &u.TableScans, &u.Efficiency); err != nil {
			continue
		}
		usage = append(usage, u)
	}

	return usage, nil
}

// getTableStats retrieves table statistics
func (o *QueryOptimizer) getTableStats(ctx context.Context) ([]TableStats, error) {
	var stats []TableStats

	rows, err := o.db.WithContext(ctx).Raw(`
		SELECT 
			relname as table_name,
			n_tup_ins + n_tup_upd + n_tup_del as row_count,
			pg_size_pretty(pg_total_relation_size(relid)) as table_size,
			pg_size_pretty(pg_indexes_size(relid)) as index_size,
			last_analyze,
			n_tup_ins + n_tup_upd + n_tup_del as vacuum_count,
			CASE WHEN autovacuum_count > 0 THEN true ELSE false END as auto_vacuum
		FROM pg_stat_user_tables
		ORDER BY pg_total_relation_size(relid) DESC
	`).Rows()

	if err != nil {
		return stats, err
	}
	defer rows.Close()

	for rows.Next() {
		var s TableStats
		if err := rows.Scan(&s.TableName, &s.RowCount, &s.TableSize, &s.IndexSize, &s.LastAnalyzed, &s.VacuumCount, &s.AutoVacuum); err != nil {
			continue
		}
		stats = append(stats, s)
	}

	return stats, nil
}

// getConnectionStats retrieves connection statistics
func (o *QueryOptimizer) getConnectionStats(ctx context.Context) (*ConnectionStats, error) {
	var stats ConnectionStats

	// Get current connection counts
	row := o.db.WithContext(ctx).Raw(`
		SELECT 
			count(*) as total,
			count(*) FILTER (WHERE state = 'active') as active,
			count(*) FILTER (WHERE state = 'idle') as idle,
			(SELECT setting::int FROM pg_settings WHERE name = 'max_connections') as max_conn
		FROM pg_stat_activity
		WHERE datname = current_database()
	`).Row()

	if err := row.Scan(&stats.TotalConnections, &stats.ActiveConnections, &stats.IdleConnections, &stats.MaxConnections); err != nil {
		return nil, err
	}

	if stats.MaxConnections > 0 {
		stats.ConnectionsPercent = (float64(stats.TotalConnections) / float64(stats.MaxConnections)) * 100
	}

	return &stats, nil
}

// generateOptimizationHints generates optimization suggestions based on metrics
func (o *QueryOptimizer) generateOptimizationHints(metrics *PerformanceMetrics) []OptimizationHint {
	var hints []OptimizationHint
	now := time.Now()

	// Check for unused indexes
	for _, index := range metrics.IndexUsage {
		if index.IndexUsed == 0 && index.TableScans > 1000 {
			hints = append(hints, OptimizationHint{
				Table:      index.TableName,
				Operation:  "DROP_INDEX",
				Suggestion: fmt.Sprintf("Consider dropping unused index '%s' on table '%s'", index.IndexName, index.TableName),
				Impact:     "Medium",
				Priority:   "Low",
				CreatedAt:  now,
			})
		}

		if index.Efficiency < 20 && index.TableScans > index.IndexUsed {
			hints = append(hints, OptimizationHint{
				Table:      index.TableName,
				Operation:  "OPTIMIZE_INDEX",
				Suggestion: fmt.Sprintf("Index '%s' on table '%s' has low efficiency (%.2f%%). Consider redesigning or dropping.", index.IndexName, index.TableName, index.Efficiency),
				Impact:     "High",
				Priority:   "Medium",
				CreatedAt:  now,
			})
		}
	}

	// Check for tables needing analysis
	for _, table := range metrics.TableStats {
		if table.LastAnalyzed == nil || time.Since(*table.LastAnalyzed) > 7*24*time.Hour {
			hints = append(hints, OptimizationHint{
				Table:      table.TableName,
				Operation:  "ANALYZE",
				Suggestion: fmt.Sprintf("Table '%s' hasn't been analyzed recently. Run ANALYZE to update statistics.", table.TableName),
				Impact:     "Medium",
				Priority:   "Medium",
				CreatedAt:  now,
			})
		}

		if !table.AutoVacuum && table.VacuumCount < 10 {
			hints = append(hints, OptimizationHint{
				Table:      table.TableName,
				Operation:  "VACUUM",
				Suggestion: fmt.Sprintf("Table '%s' may need manual vacuum. Consider enabling autovacuum.", table.TableName),
				Impact:     "Medium",
				Priority:   "Low",
				CreatedAt:  now,
			})
		}
	}

	// Check connection usage
	if metrics.ConnectionStats.ConnectionsPercent > 80 {
		hints = append(hints, OptimizationHint{
			Operation:  "CONNECTION_POOL",
			Suggestion: fmt.Sprintf("Connection usage is high (%.2f%%). Consider optimizing connection pool settings.", metrics.ConnectionStats.ConnectionsPercent),
			Impact:     "High",
			Priority:   "High",
			CreatedAt:  now,
		})
	}

	// Check for slow queries
	if len(metrics.SlowQueries) > 0 {
		hints = append(hints, OptimizationHint{
			Operation:  "QUERY_OPTIMIZATION",
			Suggestion: fmt.Sprintf("Found %d slow queries. Consider adding indexes or optimizing query patterns.", len(metrics.SlowQueries)),
			Impact:     "High",
			Priority:   "High",
			CreatedAt:  now,
		})
	}

	return hints
}

// OptimizeIndexes analyzes and suggests index optimizations for the telescopio schema
func (o *QueryOptimizer) OptimizeIndexes(ctx context.Context) ([]OptimizationHint, error) {
	o.log.Debug("Analyzing index optimization opportunities...")

	var hints []OptimizationHint
	now := time.Now()

	// Suggested indexes based on common query patterns in the telescopio system
	suggestedIndexes := []struct {
		table    string
		columns  string
		reason   string
		priority string
	}{
		{
			table:    "votes",
			columns:  "(event_id, voter_id)",
			reason:   "Optimize vote retrieval by event and voter",
			priority: "High",
		},
		{
			table:    "votes",
			columns:  "(attachment_id, rank_position)",
			reason:   "Optimize ranking queries for attachments",
			priority: "High",
		},
		{
			table:    "assignments",
			columns:  "(event_id, participant_id)",
			reason:   "Optimize assignment lookups",
			priority: "High",
		},
		{
			table:    "assignments",
			columns:  "(event_id, is_completed)",
			reason:   "Optimize completion status queries",
			priority: "Medium",
		},
		{
			table:    "attachments",
			columns:  "(event_id, participant_id)",
			reason:   "Optimize attachment queries by event and participant",
			priority: "Medium",
		},
		{
			table:    "event_participants",
			columns:  "(event_id, user_id)",
			reason:   "Optimize participant lookups (should already exist as PK)",
			priority: "Low",
		},
		{
			table:    "votes",
			columns:  "(voted_at)",
			reason:   "Optimize time-based vote queries",
			priority: "Low",
		},
		{
			table:    "events",
			columns:  "(stage, start_date)",
			reason:   "Optimize event filtering by stage and date",
			priority: "Medium",
		},
	}

	// Check if suggested indexes already exist
	for _, suggestion := range suggestedIndexes {
		exists, err := o.indexExists(ctx, suggestion.table, suggestion.columns)
		if err != nil {
			o.log.Warn("Failed to check index existence", "table", suggestion.table, "columns", suggestion.columns, "error", err)
			continue
		}

		if !exists {
			hints = append(hints, OptimizationHint{
				Table:     suggestion.table,
				Operation: "CREATE_INDEX",
				Suggestion: fmt.Sprintf("CREATE INDEX CONCURRENTLY idx_%s_%s ON %s %s -- %s",
					suggestion.table,
					sanitizeIndexName(suggestion.columns),
					suggestion.table,
					suggestion.columns,
					suggestion.reason),
				Impact:    "High",
				Priority:  suggestion.priority,
				CreatedAt: now,
			})
		}
	}

	o.log.Info("Index optimization analysis completed", "suggestions", len(hints))
	return hints, nil
}

// indexExists checks if an index with similar columns exists on a table
func (o *QueryOptimizer) indexExists(ctx context.Context, tableName, columns string) (bool, error) {
	var count int64

	// Simple check - this could be enhanced to parse column lists more intelligently
	err := o.db.WithContext(ctx).Raw(`
		SELECT COUNT(*)
		FROM pg_indexes 
		WHERE tablename = ? 
		AND indexdef ILIKE ?
	`, tableName, "%"+columns+"%").Scan(&count).Error

	return count > 0, err
}

// sanitizeIndexName creates a safe index name from column specification
func sanitizeIndexName(columns string) string {
	// Simple sanitization - remove special characters
	name := strings.ReplaceAll(columns, "(", "")
	name = strings.ReplaceAll(name, ")", "")
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, ",", "_")
	return name
}
