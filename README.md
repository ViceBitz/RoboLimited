# Algorithmic Virtual Item Trader

RoboLimited is an algorithmic trading system that buys and sells undervalued virtual assets in Roblox’s online marketplace through price and sales data analysis. The program automatically detects deals and trades by weight demand, volatility, and recent sales data. While centered on digital collectibles, RoboLimited maps to broader topics such as market monitoring, trend analysis, and automated trading.

**⚠️ DISCLAIMER**:
This project was created as a proof-of-concept to explore algorithmic approaches to virtual economies. RoboLimited is solely for educational and demonstrative purposes, not for full-scale deployment or ToS-violating use. All testing was conducted responsibly and with respect for platform integrity.

### 💡 Inspiration ###
Roblox, while best known as a gaming platform for kids, also hosts one of the largest marketplaces for collectible virtual items, known as Limiteds, which function in many ways like NFTs. Being part of my own childhood, I built this project around Roblox's marketplace because of its simplicity, low stakes, and unique features. Limiteds offer a real and dynamic online economy for this experiment.

On the platform, players buy and trade limited accessories with each other, creating price fluctuations and intrinsic value much like shares of stock. Market activity primarily revolves around:
- RAP (Recent Average Price): historical average based on recent trades.
- Value: an adjusted Robux value, often set by third-party aggregators like Rolimons.
- Best Price: the current lowest resale price available on the market.

Players use these indicators to infer a limited's worth. This Go application implements a low-latency acquisition and analysis system that detects underpriced listings, completes purchases quickly, and guides subsequent trades for profit.

---

## 📌 Systems

Two key processes drive the entire system. One acts as the hand, monitoring prices on the live market and executing trades through API requests. The other serves as the brain, analyzing trends in sales data to guide the correct call.

### Price Sniper
- Formula-driven decisions (using margins and statistical sampling)
- Integrates directly with Rolimons and Roblox APIs
- Buys items within seconds of price drops
- Fast purchase execution with direct API integration

### Limited Analyzer
- Informs immediate purchase decisions with z-score and margin analysis
- Cache sales data to classify price points quickly
- Finds trends and identifies outliers in past sales data
- Scans item owners within net worth range for trade opportunity
- Forecasts future prices with STL decomposition and Fourier regression

---

## 🚀 Key Features

### Deal Sniping  
- **Efficient Monitoring** tracks market deals through Rolimon API requests
- **Purchase Execution** sends buy orders to endpoints when price below threshold
- **Flexible Automation** keeps system running through web and connection errors
- **Throttling** to prevent rate-limiting and sustain long-term operation
- **Logging** to track decisions for further reference

### Market Evaluation
- **Spikes & Dips**: Uses statistical measures (z-score, %CV) to identify trends in sales data and guide buying, trading, selling
- **Market Metrics**: Compares prices of item groups to past time periods for market insights
- **Inventory Scan**: Assesses player inventories to estimate item and trading potential
- **Price Prediction**: Predicts item potential with seasonal cycles and trend directions
- **Data Caching**: Precomputes and stores mean / standard deviation of past sales for fast querying

---

## 🔧 Usage

### Command-Line Interface

The project CLI provides a unified way to run various modules and operations related to price sniping, item trading, inventory analysis, and forecasting. You can execute different functions directly from the terminal.

### Running the CLI

```bash
go run . -mode=<mode> [flags]
```


| Mode             | Description | Required Flags | Optional Flags |
| ---------------- | ----------- | --------------- | --------------- |
| monitor          | Starts the deal sniper to track live trades. | None | None |
| analyzeInventory | Displays player inventory metrics and forecasts. | None | -forecast_type |
| analyzeTrade     | Evaluates the future value of a proposed item trade. | -give, -receive | -daysPast, -daysFuture |
| searchDips       | Finds items in the market that are currently dropping in price. | None | -threshold, -priceLow, -priceHigh, -isDemand |
| searchForecast   | Forecasts growth potential using past year data. | None | -priceLow, -priceHigh, -daysPast, -daysFuture, -isDemand |
| searchOwners   | Scans item owners within net worth range. | -item | -priceLow, -priceHigh, -limit |
| forecast         | General price forecasting for a list of items. | -items | -isDemand, -daysPast, -daysFuture |

| Flag           | Type    | Default       | Description |
| -------------- | ------- | ------------- | ----------- |
| -mode          | string  | "monitor"     | Specifies which function/mode to run: monitor, analyzeInventory, analyzeTrade, searchDips, searchForecast, forecast |
| -give          | string  | ""            | Comma-separated list of items to give |
| -receive       | string  | ""            | Comma-separated list of items to receive |
| -forecast_type | string  | "stl"         | Forecast type for inventory analysis |
| -threshold     | float64 | -0.5          | Threshold value for detecting price dips |
| -priceLow      | float64 | 0.0           | Minimum price filter |
| -priceHigh     | float64 | 1000000.0     | Maximum price filter |
| -isDemand      | bool    | true         | Only include high-demand items |
| -item      | string    | ""         | Specific item to target |
| -limit      | int    | 20         | Max number of records to output |
| -items         | string  | ""            | Comma-separated list of items to forecast |
| -daysPast      | int64   | 365*3          | Number of past days of historical data to include in forecasts |
| -daysFuture    | int64   | 30            | Number of days forward to project average price |

Example:
```bash
go run . -mode=analyzeTrade -give=11188705,119040562647325,20573078,11700905898 -receive=928908332
```

## 📊 Results
During experimental tests, the algorithm scanned over **2000 virtual assets** in live markets during a one-month period. These actions netted **30% avg. ROI** after internal marketplace fees but before currency conversion costs.

#
