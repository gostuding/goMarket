package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type table map[string]any

func tablesMaps() *map[string]table {

	users := make(table)
	users["login"] = "50,unique"
	users["pwd"] = "32"
	users["useragent"] = "255"
	users["ip"] = "15"
	users["session"] = "32"
	users["registered"] = time.Now()
	users["expared"] = time.Now()

	result := make(map[string]table)
	result["users"] = users
	return &result
}

func checkTable(ctx context.Context, name string, values map[string]any, con *sql.DB) error {
	items := make([]string, 0)
	for key, val := range values {
		switch val.(type) {
		case int, int16, int32, int64, uint16, uint32, uint64:
			items = append(items, fmt.Sprintf("%s bigserial", key))
		case string:
			value := fmt.Sprintf("%s", val)
			lenth := strings.Split(value, ",")[0]
			unique := ""
			if strings.Contains(value, "unique") {
				unique = " UNIQUE"
			}
			items = append(items, fmt.Sprintf("%s varchar(%s)%s", key, lenth, unique))
		case bool:
			items = append(items, fmt.Sprintf("%s boolean", key))
		case time.Time:
			items = append(items, fmt.Sprintf("%s timestamp", key))
		case float32, float64:
			items = append(items, fmt.Sprintf("%s double precision", key))
		}
	}

	query := fmt.Sprintf("CREATE TABLE  IF NOT EXISTS %s  (id SERIAL PRIMARY KEY, %s);", name, strings.Join(items, ","))
	_, err := con.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("create new table ('%s') error: %w ", name, err)
	}
	return err
}

func structCheck(con *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := con.PingContext(ctx); err != nil {
		return fmt.Errorf("check database structure ping error: %w", err)
	}
	for key, table := range *tablesMaps() {
		err := checkTable(ctx, key, table, con)
		if err != nil {
			return err
		}
	}
	return nil
}
