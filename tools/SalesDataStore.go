package tools

/**
Stores mean and standard deviation of items' past sales data for quick querying
*/
import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"robolimited/config"
	"strconv"
	"time"
	"log"
)

//Mean and standard deviation for an asset's sales
type StatsPoint struct {
	ID     string
	Mean   float64
	StdDev float64
}
type Stats struct {
	Mean   float64
	StdDev float64
}

//Raw time-series sales data for an asset
type Sales struct {
	NumPoints          int     `json:"num_points"`
	Timestamp          []int64 `json:"timestamp"`
	AvgDailySalesPrice []int   `json:"avg_daily_sales_price"`
	SalesVolume        []int   `json:"sales_volume"`
}

type SalesPoint struct {
	Date               time.Time
	AvgDailySalesPrice int
	SalesVolume        int
}

//Global map for condensed sales stats
var SalesStats = make(map[string]Stats)
//Global map for raw sales data
var SalesData = make(map[string]*Sales)

//Store sales statistics data to a CSV file
func StoreSalesStats(data []StatsPoint) {
	file, err := os.Create(config.SalesStatsFile)
	if err != nil {
		fmt.Printf("Failed to create CSV file")
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	//Write header
	header := []string{"ID", "Mean", "SD"}
	if err := writer.Write(header); err != nil {
		fmt.Println("Failed to write header")
		return
	}

	//Write data rows
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

//Retrieves statistics data of specific ID from CSV file
func RetrieveSalesStats() map[string]Stats {
	file, err := os.Open(config.SalesStatsFile)
	if err != nil {
		log.Println("Failed to open CSV file", err)
		return nil
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		log.Println("Failed to read CSV file", err)
		return nil
	}

	if len(records) == 0 {
		log.Println("CSV file is empty")
		return nil
	}

	//Add all ids and data to rows
	result := make(map[string]Stats)
	for i := 1; i < len(records); i++ {
		record := records[i]
		if len(record) != 3 {
			continue
		}
		id := record[0]
		mean, err := strconv.ParseFloat(record[1], 64)
		if err != nil {
			log.Println("Failed to parse mean for ID", id)
			return nil
		}

		stdDev, err := strconv.ParseFloat(record[2], 64)
		if err != nil {
			log.Println("failed to parse standard deviation for ID", id)
			return nil
		}

		result[id] = Stats{
			Mean:   mean,
			StdDev: stdDev,
		}
	}
	return result
}

//Stores raw sales data to JSON file
func StoreSalesData(data map[string]*Sales) {
	jsonData, err := json.Marshal(data)
	if (err != nil) {
		log.Println("Error marshalling sales data:", err)
		return
	}
	err = os.WriteFile(config.SalesDataFile, jsonData, 0644)
	if (err != nil) {
		log.Println("Error writing sales data to file:", err)
		return
	}
	log.Println("Successfully wrote sales data to json")
}
//Retrieves raw sales data of specific ID from JSON file
func RetrieveSalesData() (map[string] *Sales) {
	bytes, err := os.ReadFile(config.SalesDataFile)
	var data map[string]*Sales
	if (err != nil) {
		log.Println("Error reading from json file:", err)
		return data
	}
	
	err = json.Unmarshal(bytes, &data)
	if (err != nil) {
		log.Println("Error unmarshaling sales data from json:", err)
		return data
	}
	return data
}