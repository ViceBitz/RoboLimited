package tools

/**
Stores mean and standard deviation of items' past sales data for quick querying
*/
import (
	"encoding/csv"
	"fmt"
	"os"
	"robolimited/config"
	"strconv"
)

// Represents the mean and standard deviation for an item
type StatsData struct {
	ID     string
	Mean   float64
	StdDev float64
}

// Represents mean and standard deviation values
type Stats struct {
	Mean   float64
	StdDev float64
}

// Global map to store sales data
var SalesData = make(map[string]Stats)

// Store sales statistics data to a CSV file
func StoreSales(data []StatsData) {
	file, err := os.Create(config.SalesDataFile)
	if err != nil {
		fmt.Printf("Failed to create CSV file")
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{"ID", "Mean", "SD"}
	if err := writer.Write(header); err != nil {
		fmt.Println("Failed to write header")
		return
	}

	// Write data rows
	for _, stat := range data {
		record := []string{
			stat.ID,
			strconv.FormatFloat(stat.Mean, 'f', -1, 64),
			strconv.FormatFloat(stat.StdDev, 'f', -1, 64),
		}
		if err := writer.Write(record); err != nil {
			fmt.Println("Failed to write record")
			return
		}
	}
}

// Retrieves statistics data for a specific ID from CSV file
func RetrieveSales() map[string]Stats {
	file, err := os.Open(config.SalesDataFile)
	if err != nil {
		fmt.Println("failed to open CSV file")
		return nil
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		fmt.Println("failed to read CSV file")
		return nil
	}

	if len(records) == 0 {
		fmt.Println("CSV file is empty")
		return nil
	}

	// Add all ids and data to rows
	result := make(map[string]Stats)
	for i := 1; i < len(records); i++ {
		record := records[i]
		if len(record) != 3 {
			continue // Skip malformed rows
		}
		id := record[0]
		mean, err := strconv.ParseFloat(record[1], 64)
		if err != nil {
			fmt.Println("Failed to parse mean for ID", id)
			return nil
		}

		stdDev, err := strconv.ParseFloat(record[2], 64)
		if err != nil {
			fmt.Println("failed to parse standard deviation for ID", id)
			return nil
		}

		result[id] = Stats{
			Mean:   mean,
			StdDev: stdDev,
		}
	}
	return result
}
