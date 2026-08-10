package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/xuri/excelize/v2"
)

func main() {
	// Open the Excel file
	f, err := excelize.OpenFile("/Users/dev/Downloads/EPE.xlsx")
	if err != nil {
		log.Fatalf("Error opening file: %v", err)
	}
	defer f.Close()

	// List all sheets
	sheets := f.GetSheetList()
	fmt.Printf("Found %d sheets: %v\n\n", len(sheets), sheets)

	for _, sheet := range sheets {
		fmt.Printf("=== Sheet: %q ===\n", sheet)

		rows, err := f.GetRows(sheet)
		if err != nil {
			log.Printf("Error reading rows: %v", err)
			continue
		}

		// Show first 20 rows
		maxRows := len(rows)
		if maxRows > 20 {
			maxRows = 20
		}

		fmt.Printf("Total rows: %d\n", len(rows))
		fmt.Println("\nFirst 20 rows:")

		for i, row := range rows {
			if i >= maxRows {
				break
			}
			fmt.Printf("Row %2d: ", i+1)
			// Show first 15 columns
			maxCols := len(row)
			if maxCols > 15 {
				maxCols = 15
			}
			for j := 0; j < maxCols; j++ {
				val := strings.TrimSpace(row[j])
				if val == "" {
					val = "None"
				}
				fmt.Printf("[%s] ", val)
			}
			if len(row) > 15 {
				fmt.Printf("...(+%d cols)", len(row)-15)
			}
			fmt.Println()
		}
		fmt.Println()
	}
}
