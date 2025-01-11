package pg

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// Struct to hold pg_stat_activity data
type PgStatActivity struct {
	PID             int    `json:"pid"`
	Username        string `json:"username"`
	ApplicationName string `json:"application_name"`
	State           string `json:"state"`
	Query           string `json:"query"`
	QueryStart      string `json:"query_start"`
}

// Struct to hold additional PostgreSQL information
type DbInfo struct {
	PgVersion      string `json:"pg_version"`
	PostgisVersion string `json:"postgis_version"`
	DbSize         string `json:"db_size"`
}

// Struct to return all the data
type PgDiagnoseData struct {
	DbInfo           DbInfo           `json:"db_info"`
	PgStatActivities []PgStatActivity `json:"pg_stat_activities"`
}

func GetPgDiagnose() (PgDiagnoseData, error) {
	// Get the DATABASE_URL environment variable
	var diagnostic PgDiagnoseData

	databaseURL := os.Getenv("DATABASE_URL")
	// Open connection to the database
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Println(databaseURL)
		return diagnostic, fmt.Errorf("Can't connect to DATABASE_URL")
	}
	defer db.Close()

	// Execute query to fetch pg_stat_activity information
	query := `
		SELECT pid, usename, application_name, state, query, query_start
		FROM pg_stat_activity		
		ORDER BY query_start DESC;
	`
	rows, err := db.Query(query)
	if err != nil {
		return diagnostic, fmt.Errorf("Can't query to DATABASE_URL")
	}
	defer rows.Close()

	// Create a slice to store the results
	var activities []PgStatActivity

	// Iterate over rows and append to the results slice
	for rows.Next() {
		var activity PgStatActivity
		err := rows.Scan(&activity.PID, &activity.Username, &activity.ApplicationName, &activity.State, &activity.Query, &activity.QueryStart)
		if err == nil {
			activities = append(activities, activity)
		}
	}

	var pgVersion, postgisVersion, dbSize string
	db.QueryRow("SELECT version();").Scan(&pgVersion)
	db.QueryRow("SELECT PostGIS_Version();").Scan(&postgisVersion)
	db.QueryRow("SELECT pg_size_pretty(pg_database_size(current_database()));").Scan(&dbSize)

	diagnostic = PgDiagnoseData{
		DbInfo: DbInfo{
			PgVersion:      pgVersion,
			PostgisVersion: postgisVersion,
			DbSize:         dbSize,
		},
		PgStatActivities: activities,
	}
	return diagnostic, nil
}
