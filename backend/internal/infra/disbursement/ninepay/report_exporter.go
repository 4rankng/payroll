package ninepay

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"
)

var ErrNoTransactions = errors.New("ninepay: no transactions in the selected date range")

// ExportReconciliation satisfies infrastructure.ReportExporter on
// *Provider. Queries the 9pay index endpoint to discover the total
// page count, then downloads each page's CSV and concatenates them
// into a single file. Returns the combined CSV bytes and the
// provider-issued file name from the first page.
func (p *Provider) ExportReconciliation(ctx context.Context, dateFrom, dateTo time.Time) ([]byte, string, error) {
	if p.client == nil {
		return nil, "", fmt.Errorf("ninepay: report exporter not configured")
	}

	lastPage, err := p.client.GetExportPageCount(ctx, dateFrom, dateTo)
	if err != nil {
		return nil, "", fmt.Errorf("ninepay: get page count: %w", err)
	}
	if lastPage == 0 {
		return nil, "", ErrNoTransactions
	}

	p.logger.Info("ninepay: export reconciliation",
		"last_page", lastPage, "date_from", dateFrom.Format("02/01/2006"), "date_to", dateTo.Format("02/01/2006"))

	var combined bytes.Buffer
	var firstFileName string

	for page := 1; page <= lastPage; page++ {
		fileName, err := p.client.RequestExport(ctx, dateFrom, dateTo, page)
		if err != nil {
			return nil, firstFileName, fmt.Errorf("ninepay: request export page %d: %w", page, err)
		}
		if firstFileName == "" {
			firstFileName = fileName
		}

		csvBytes, err := p.client.DownloadExport(ctx, fileName)
		if err != nil {
			return nil, firstFileName, fmt.Errorf("ninepay: download export page %d: %w", page, err)
		}
		if len(csvBytes) == 0 {
			continue
		}

		if combined.Len() == 0 {
			combined.Write(csvBytes)
			continue
		}

		// Strip the header line from pages 2..N to avoid duplicated headers.
		// The CSV starts with a UTF-8 BOM (3 bytes), then the header row.
		if _, after, ok := bytes.Cut(csvBytes, []byte("\n")); ok {
			combined.Write(after)
		}
	}

	if combined.Len() == 0 {
		return nil, firstFileName, fmt.Errorf("ninepay: export returned empty body")
	}
	return combined.Bytes(), firstFileName, nil
}

// ExportReconciliation on the queued wrapper delegates to the inner
// Provider — exports are off the throttled disbursement path.
func (q *queuedProvider) ExportReconciliation(ctx context.Context, dateFrom, dateTo time.Time) ([]byte, string, error) {
	return q.inner.ExportReconciliation(ctx, dateFrom, dateTo)
}
