package mysql

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/require"
)

func TestQueryResults(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"Get", testGetQueryResultRows},
		{"CountForQuery", testCountResultsForQuery},
		{"CountForQueryAndHost", testCountResultsForQueryAndHost},
		{"Overwrite", testOverwriteQueryResultRows},
		{"MaxRows", testQueryResultRowsDoNotExceedMaxRows},
		{"QueryResultRows", testQueryResultRows},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testGetQueryResultRows(t *testing.T, ds *Datastore) {
	user := test.NewUser(t, ds, "Test User", "test@example.com", true)
	query := test.NewQuery(t, ds, nil, "New Query", "SELECT 1", user.ID, true)
	host := test.NewHost(t, ds, "hostname123", "192.168.1.100", "1234", "UI8XB1223", time.Now())

	mockTime := time.Now().UTC().Truncate(time.Second)

	// Insert Result Rows for Query1
	query1Rows := []*fleet.ScheduledQueryResultRow{
		{
			QueryID:     query.ID,
			HostID:      host.ID,
			LastFetched: mockTime,
			Data:        nil,
		},
		{
			QueryID:     query.ID,
			HostID:      host.ID,
			LastFetched: mockTime,
			Data: ptr.RawMessage([]byte(`{
				"model": "USB Keyboard",
				"vendor": "Apple Inc."
			}`)),
		},
	}
	err := ds.SaveQueryResultRows(context.Background(), query1Rows)
	require.NoError(t, err)

	// Insert Result Row for different Scheduled Query
	query2 := test.NewQuery(t, ds, nil, "New Query 2", "SELECT 1", user.ID, true)
	query2Rows := []*fleet.ScheduledQueryResultRow{
		{
			QueryID:     query2.ID,
			HostID:      host.ID,
			LastFetched: mockTime,
			Data:        ptr.RawMessage([]byte(`{"model": "USB Hub","vendor": "Logitech"}`)),
		},
	}

	err = ds.SaveQueryResultRows(context.Background(), query2Rows)
	require.NoError(t, err)

	results, err := ds.QueryResultRows(context.Background(), query.ID)
	require.NoError(t, err)
	require.Len(t, results, 1) // Should not return rows with nil data
	require.Equal(t, query1Rows[1].QueryID, results[0].QueryID)
	require.Equal(t, query1Rows[1].HostID, results[0].HostID)
	require.Equal(t, query1Rows[1].LastFetched.Unix(), results[0].LastFetched.Unix())
	require.JSONEq(t, string(*query1Rows[1].Data), string(*results[0].Data))

	// Assert that Query2 returns 1 result
	results, err = ds.QueryResultRows(context.Background(), query2.ID)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, query2Rows[0].QueryID, results[0].QueryID)
	require.Equal(t, query2Rows[0].HostID, results[0].HostID)
	require.Equal(t, query2Rows[0].LastFetched.Unix(), results[0].LastFetched.Unix())
	require.JSONEq(t, string(*query2Rows[0].Data), string(*results[0].Data))

	// Assert that QueryResultRowsForHost returns empty slice when no results are found
	results, err = ds.QueryResultRowsForHost(context.Background(), 999, 999)
	require.NoError(t, err)
	require.Len(t, results, 0)
}

func testGetQueryResultRowsForHost(t *testing.T, ds *Datastore) {
	user := test.NewUser(t, ds, "Test User", "test@example.com", true)
	query := test.NewQuery(t, ds, nil, "New Query", "SELECT 1", user.ID, true)
	host1 := test.NewHost(t, ds, "hostname1", "192.168.1.100", "1111", "UI8XB1223", time.Now())
	host2 := test.NewHost(t, ds, "hostname2", "192.168.1.100", "2222", "UI8XB1223", time.Now())

	mockTime := time.Now().UTC().Truncate(time.Second)

	// Insert 2 Result Rows for Query1 Host1
	host1ResultRows := []*fleet.ScheduledQueryResultRow{
		{
			QueryID:     query.ID,
			HostID:      host1.ID,
			LastFetched: mockTime,
			Data:        nil,
		},
		{
			QueryID:     query.ID,
			HostID:      host1.ID,
			LastFetched: mockTime,
			Data:        ptr.RawMessage([]byte(`{"model": "USB Mouse", "vendor": "Logitech"}`)),
		},
	}
	err := ds.OverwriteQueryResultRows(context.Background(), host1ResultRows)
	require.NoError(t, err)

	// Insert 1 Result Row for Query1 Host2
	host2ResultRows := []*fleet.ScheduledQueryResultRow{
		{
			QueryID:     query.ID,
			HostID:      host2.ID,
			LastFetched: mockTime,
			Data:        ptr.RawMessage([]byte(`{"model": "USB Mouse", "vendor": "Logitech"}`)),
		},
	}
	err = ds.OverwriteQueryResultRows(context.Background(), host2ResultRows)
	require.NoError(t, err)

	// Assert that Query1 returns 2 results for Host1
	results, err := ds.QueryResultRowsForHost(context.Background(), query.ID, host1.ID)
	require.NoError(t, err)
	require.Len(t, results, 2) // should return rows with nil data
	require.Equal(t, host1ResultRows[0].QueryID, results[0].QueryID)
	require.Equal(t, host1ResultRows[0].HostID, results[0].HostID)
	require.Equal(t, host1ResultRows[0].LastFetched.Unix(), results[0].LastFetched.Unix())
	require.Nil(t, results[0].Data)
	require.Equal(t, host1ResultRows[1].QueryID, results[1].QueryID)
	require.Equal(t, host1ResultRows[1].HostID, results[1].HostID)
	require.Equal(t, host1ResultRows[1].LastFetched.Unix(), results[1].LastFetched.Unix())
	require.JSONEq(t, string(*host1ResultRows[1].Data), string(*results[1].Data))

	// Assert that Query1 returns 1 result for Host2
	results, err = ds.QueryResultRowsForHost(context.Background(), query.ID, host2.ID)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, host2ResultRows[0].QueryID, results[0].QueryID)
	require.Equal(t, host2ResultRows[0].HostID, results[0].HostID)
	require.Equal(t, host2ResultRows[0].LastFetched.Unix(), results[0].LastFetched.Unix())
	require.JSONEq(t, string(*host2ResultRows[0].Data), string(*results[0].Data))
}

func testCountResultsForQuery(t *testing.T, ds *Datastore) {
	user := test.NewUser(t, ds, "Test User", "test@example.com", true)
	query1 := test.NewQuery(t, ds, nil, "New Query", "SELECT 1", user.ID, true)
	query2 := test.NewQuery(t, ds, nil, "New Query 2", "SELECT 1", user.ID, true)
	host := test.NewHost(t, ds, "hostname123", "192.168.1.100", "1234", "UI8XB1223", time.Now())

	mockTime := time.Now().UTC().Truncate(time.Second)

	// Insert 1 Result Row for Query1
	resultRow := []*fleet.ScheduledQueryResultRow{
		{
			QueryID:     query1.ID,
			HostID:      host.ID,
			LastFetched: mockTime,
			Data: ptr.RawMessage([]byte(`{
				"model": "USB Keyboard",
				"vendor": "Apple Inc."
			}`)),
		},
	}
	err := ds.SaveQueryResultRows(context.Background(), resultRow)
	require.NoError(t, err)

	// Insert 1 Result Row with nil Data for Query1
	// This should not be counted
	resultRowNilData := []*fleet.ScheduledQueryResultRow{
		{
			QueryID:     query1.ID,
			HostID:      host.ID,
			LastFetched: mockTime,
			Data:        nil,
		},
	}
	err = ds.SaveQueryResultRows(context.Background(), resultRowNilData)
	require.NoError(t, err)

	// Insert 5 Result Rows for Query2
	resultRow2 := []*fleet.ScheduledQueryResultRow{
		{
			QueryID:     query2.ID,
			HostID:      host.ID,
			LastFetched: mockTime,
			Data: ptr.RawMessage([]byte(`{
				"model": "USB Mouse",
				"vendor": "Apple Inc."
			}`)),
		},
	}
	for i := 0; i < 5; i++ {
		err = ds.SaveQueryResultRows(context.Background(), resultRow2)
		require.NoError(t, err)
	}

	// Assert that ResultCountForQuery returns 1
	count, err := ds.ResultCountForQuery(context.Background(), query1.ID)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	// Assert that ResultCountForQuery returns 5
	count, err = ds.ResultCountForQuery(context.Background(), query2.ID)
	require.NoError(t, err)
	require.Equal(t, 5, count)

	// Returns 0 when no results are found
	count, err = ds.ResultCountForQuery(context.Background(), 999)
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func testCountResultsForQueryAndHost(t *testing.T, ds *Datastore) {
	user := test.NewUser(t, ds, "Test User", "test@example.com", true)
	query1 := test.NewQuery(t, ds, nil, "New Query", "SELECT 1", user.ID, true)
	query2 := test.NewQuery(t, ds, nil, "New Query 2", "SELECT 1", user.ID, true)
	host := test.NewHost(t, ds, "host1", "192.168.1.100", "1234", "UI8XB1223", time.Now())
	host2 := test.NewHost(t, ds, "host2", "192.168.1.101", "4567", "UI8XB1224", time.Now())

	mockTime := time.Now().UTC().Truncate(time.Second)

	resultRows := []*fleet.ScheduledQueryResultRow{
		{
			QueryID:     query1.ID,
			HostID:      host.ID,
			LastFetched: mockTime,
			Data: ptr.RawMessage([]byte(`{
				"model": "USB Keyboard",
				"vendor": "Apple Inc."
			}`)),
		},
		{
			QueryID:     query1.ID,
			HostID:      host.ID,
			LastFetched: mockTime,
			Data: ptr.RawMessage([]byte(`{
				"model": "USB Mouse",
				"vendor": "Logitech"
			}`)),
		},
		{
			QueryID:     query1.ID,
			HostID:      host2.ID,
			LastFetched: mockTime,
			Data: ptr.RawMessage([]byte(`{
				"model": "USB Mouse",
				"vendor": "Logitech"
			}`)),
		},
		{
			QueryID:     query2.ID,
			HostID:      host.ID,
			LastFetched: mockTime,
			Data: ptr.RawMessage([]byte(`{
				"model": "USB Mouse",
				"vendor": "Logitech"
			}`)),
		},
		{
			QueryID:     query2.ID, // This row should not be counted
			HostID:      host.ID,
			LastFetched: mockTime,
			Data:        nil,
		},
	}

	err := ds.SaveQueryResultRows(context.Background(), resultRows)
	require.NoError(t, err)

	// Assert that Query1 returns 2
	count, err := ds.ResultCountForQueryAndHost(context.Background(), query1.ID, host.ID)
	require.NoError(t, err)
	require.Equal(t, 2, count)

	// Assert that ResultCountForQuery returns 1
	count, err = ds.ResultCountForQueryAndHost(context.Background(), query2.ID, host.ID)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	// Returns empty result when no results are found
	count, err = ds.ResultCountForQueryAndHost(context.Background(), 999, host.ID)
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func testOverwriteQueryResultRows(t *testing.T, ds *Datastore) {
	user := test.NewUser(t, ds, "Test User", "test@example.com", true)
	query := test.NewQuery(t, ds, nil, "Overwrite Test Query", "SELECT 1", user.ID, true)
	host := test.NewHost(t, ds, "hostname1234", "192.168.1.101", "12345", "UI8XB1224", time.Now())

	mockTime := time.Now().UTC().Truncate(time.Second)

	// Insert initial Result Rows
	initialRows := []*fleet.ScheduledQueryResultRow{
		{
			QueryID:     query.ID,
			HostID:      host.ID,
			LastFetched: mockTime,
			Data:        ptr.RawMessage([]byte(`{"model": "USB Keyboard", "vendor": "Apple Inc."}`)),
		},
	}

	err := ds.SaveQueryResultRows(context.Background(), initialRows)
	require.NoError(t, err)

	// Overwrite Result Rows with new data
	newMockTime := mockTime.Add(2 * time.Minute)
	overwriteRows := []*fleet.ScheduledQueryResultRow{
		{
			QueryID:     query.ID,
			HostID:      host.ID,
			LastFetched: newMockTime,
			Data:        ptr.RawMessage([]byte(`{"model": "USB Mouse", "vendor": "Logitech"}`)),
		},
	}

	err = ds.OverwriteQueryResultRows(context.Background(), overwriteRows)
	require.NoError(t, err)

	// Assert that we get the overwritten data (1 result with USB Mouse data)
	results, err := ds.QueryResultRowsForHost(context.Background(), overwriteRows[0].QueryID, overwriteRows[0].HostID)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, overwriteRows[0].QueryID, results[0].QueryID)
	require.Equal(t, overwriteRows[0].HostID, results[0].HostID)
	require.Equal(t, overwriteRows[0].LastFetched.Unix(), results[0].LastFetched.Unix())
	require.JSONEq(t, string(*overwriteRows[0].Data), string(*results[0].Data))

	// Test calling OverwriteQueryResultRows with a query that doesn't exist (e.g. a deleted query).
	overwriteRows = []*fleet.ScheduledQueryResultRow{
		{
			QueryID:     9999,
			HostID:      host.ID,
			LastFetched: newMockTime,
			Data:        ptr.RawMessage([]byte(`{"model": "USB Mouse", "vendor": "Logitech"}`)),
		},
	}
	err = ds.OverwriteQueryResultRows(context.Background(), overwriteRows)
	require.NoError(t, err)

	// Assert that the data has not changed
	results, err = ds.QueryResultRowsForHost(context.Background(), overwriteRows[0].QueryID, overwriteRows[0].HostID)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, overwriteRows[0].QueryID, results[0].QueryID)
	require.Equal(t, overwriteRows[0].HostID, results[0].HostID)
	require.Equal(t, overwriteRows[0].LastFetched.Unix(), results[0].LastFetched.Unix())
	require.JSONEq(t, string(*overwriteRows[0].Data), string(*results[0].Data))
}

func testQueryResultRowsDoNotExceedMaxRows(t *testing.T, ds *Datastore) {
	user := test.NewUser(t, ds, "Test User", "test@example.com", true)
	query := test.NewQuery(t, ds, nil, "Overwrite Test Query", "SELECT 1", user.ID, true)
	query2 := test.NewQuery(t, ds, nil, "Overwrite Test Query 2", "SELECT 1", user.ID, true)
	host1 := test.NewHost(t, ds, "hostname1", "192.168.1.101", "11111", "UI8XB1221", time.Now())
	host2 := test.NewHost(t, ds, "hostname2", "192.168.1.101", "22222", "UI8XB1222", time.Now())
	host3 := test.NewHost(t, ds, "hostname3", "192.168.1.101", "33333", "UI8XB1223", time.Now())
	host4 := test.NewHost(t, ds, "hostname4", "192.168.1.101", "44444", "UI8XB1224", time.Now())

	mockTime := time.Now().UTC().Truncate(time.Second)

	// Generate max rows -1
	maxRows := fleet.MaxQueryReportRows - 1
	maxMinusOneRows := make([]*fleet.ScheduledQueryResultRow, maxRows)
	for i := 0; i < maxRows; i++ {
		maxMinusOneRows[i] = &fleet.ScheduledQueryResultRow{
			QueryID:     query.ID,
			HostID:      host1.ID,
			LastFetched: mockTime,
			Data:        ptr.RawMessage([]byte(`{"model": "USB Mouse", "vendor": "Logitech"}`)),
		}
	}
	err := ds.OverwriteQueryResultRows(context.Background(), maxMinusOneRows)
	require.NoError(t, err)

	// Add an empty data rows which do not count towards the max
	err = ds.OverwriteQueryResultRows(context.Background(), []*fleet.ScheduledQueryResultRow{
		{
			QueryID:     query.ID,
			HostID:      host2.ID,
			LastFetched: mockTime,
			Data:        nil,
		},
	})
	require.NoError(t, err)

	// Confirm that we can still add a row
	err = ds.OverwriteQueryResultRows(context.Background(), []*fleet.ScheduledQueryResultRow{
		{
			QueryID:     query.ID,
			HostID:      host3.ID,
			LastFetched: mockTime,
			Data:        ptr.RawMessage([]byte(`{"model": "USB Mouse", "vendor": "Logitech"}`)),
		},
	})
	require.NoError(t, err)

	// Assert that we now have max rows
	count, err := ds.ResultCountForQuery(context.Background(), query.ID)
	require.NoError(t, err)
	require.Equal(t, fleet.MaxQueryReportRows, count)

	// Attempt to add another row
	err = ds.OverwriteQueryResultRows(context.Background(), []*fleet.ScheduledQueryResultRow{
		{
			QueryID:     query.ID,
			HostID:      host4.ID,
			LastFetched: mockTime,
			Data:        ptr.RawMessage([]byte(`{"model": "USB Mouse", "vendor": "Logitech"}`)),
		},
	})
	require.NoError(t, err)

	// Assert that the last row was not added
	host4result, err := ds.QueryResultRowsForHost(context.Background(), query.ID, host4.ID)
	require.NoError(t, err)
	require.Len(t, host4result, 0)

	// Generate more than max rows in Query 2
	rows := fleet.MaxQueryReportRows + 50
	largeBatchRows := make([]*fleet.ScheduledQueryResultRow, rows)
	for i := 0; i < rows; i++ {
		largeBatchRows[i] = &fleet.ScheduledQueryResultRow{
			QueryID:     query2.ID,
			HostID:      host1.ID,
			LastFetched: mockTime,
			Data:        ptr.RawMessage([]byte(`{"model": "USB Mouse", "vendor": "Logitech"}`)),
		}
	}
	err = ds.OverwriteQueryResultRows(context.Background(), largeBatchRows)
	require.NoError(t, err)

	// Confirm only max rows are stored for the queryID
	allResults, err := ds.QueryResultRowsForHost(context.Background(), query2.ID, host1.ID)
	require.NoError(t, err)
	require.Len(t, allResults, fleet.MaxQueryReportRows)

	// Confirm that new rows are not added when the max is reached
	newMockTime := mockTime.Add(2 * time.Minute)
	overwriteRows := []*fleet.ScheduledQueryResultRow{
		{
			QueryID:     query2.ID,
			HostID:      host2.ID,
			LastFetched: newMockTime,
			Data:        ptr.RawMessage([]byte(`{"model": "USB Mouse", "vendor": "Logitech"}`)),
		},
	}

	err = ds.OverwriteQueryResultRows(context.Background(), overwriteRows)
	require.NoError(t, err)

	host2Results, err := ds.QueryResultRowsForHost(context.Background(), query2.ID, host2.ID)
	require.NoError(t, err)
	require.Len(t, host2Results, 0)
}

func testQueryResultRows(t *testing.T, ds *Datastore) {
	user := test.NewUser(t, ds, "Test User", "test@example.com", true)
	query := test.NewQuery(t, ds, nil, "Overwrite Test Query", "SELECT 1", user.ID, true)

	mockTime := time.Now().UTC().Truncate(time.Second)

	overwriteRows := []*fleet.ScheduledQueryResultRow{
		{
			QueryID:     query.ID,
			HostID:      9999,
			LastFetched: mockTime,
			Data:        ptr.RawMessage([]byte(`{"model": "USB Mouse", "vendor": "Logitech"}`)),
		},
	}
	err := ds.OverwriteQueryResultRows(context.Background(), overwriteRows)
	require.NoError(t, err)

	// Test calling QueryResultRows with a query that has an entry with a host that doesn't exist anymore.
	results, err := ds.QueryResultRows(context.Background(), query.ID)
	require.NoError(t, err)
	require.Len(t, results, 1)
}

func (ds *Datastore) SaveQueryResultRows(ctx context.Context, rows []*fleet.ScheduledQueryResultRow) error {
	if len(rows) == 0 {
		return nil // Nothing to insert
	}

	valueStrings := make([]string, 0, len(rows))
	valueArgs := make([]interface{}, 0, len(rows)*4)

	for _, row := range rows {
		valueStrings = append(valueStrings, "(?, ?, ?, ?)")
		valueArgs = append(valueArgs, row.QueryID, row.HostID, row.LastFetched, row.Data)
	}

	insertStmt := fmt.Sprintf(`
        INSERT INTO query_results (query_id, host_id, last_fetched, data)
            VALUES %s
    `, strings.Join(valueStrings, ","))

	_, err := ds.writer(ctx).ExecContext(ctx, insertStmt, valueArgs...)
	if err != nil {
		return err
	}

	return nil
}
