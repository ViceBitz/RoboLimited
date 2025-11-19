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
- Forecasts future prices with STL decomposition and linear regression to guide long-term trading


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
- **Price Prediction**: Leverages past seasonal cycles and price patterns to predict item potential  
- **Data Caching**: Precomputes and stores mean / standard deviation of past sales for fast querying

---


## 📊 Results
During experimental tests, the algorithm bought and sold over **20 virtual assets** in live markets during a one-month period. These actions netted **30% ROI** after internal marketplace fees but before currency conversion costs.

---

#
