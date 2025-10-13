# Algorithmic Virtual Item Trader

RoboLimited is a proof-of-concept trading system that applies algorithmic strategies to identify and analyze undervalued virtual assets in Roblox’s online marketplace using data from Rolimons and the Roblox API. It automates deal detection, price analysis, and trades by weighing demand, volatility, and recent sales data, a framework comparable to statistical arbitrage. While centered on digital collectibles, the system’s design maps to broader principles of financial technology such as real-time market monitoring, data analysis, and automated trading logic.

**⚠️ DISCLAIMER**:
This project was created as a proof-of-concept to explore algorithmic approaches to virtual economies. RoboLimited is solely for educational and research purposes, not for full-scale deployment or ToS-violating use. All testing was conducted responsibly and with respect for platform integrity.

### 💡 Inspiration ###
Roblox, while best known as a gaming platform for kids, also hosts one of the largest marketplaces for collectible virtual items, known as Limiteds, which function in many ways like NFTs. Growing up, Roblox was a part of my childhood, so many years later, I'm using its marketplace as the foundation for this project. The limited market space offers the perfect opportunity to practice building an efficient algorithmic trading system within a real, dynamic online economy.

On the platform, players buy and trade limited accessories with each other, creating price fluctuations and intrinsic value much like shares of stock. Market activity revolves around several key metrics:
- RAP (Recent Average Price): historical average based on recent trades.
- Value: an adjusted Robux value, often set by third-party aggregators like Rolimons.
- Best Price: the current lowest resale price available on the market.

Market participants use these indicators to infer a limited's worth and make a profit by getting their hands on them before everyone else catches on. This Go application implements a low-latency acquisition and trading pipeline that detects underpriced listings, completes purchases quickly, and manages subsequent trades for profit.

---

## 📌 Systems

### Price Sniper
- Formula-driven decisions (using margins and statistical sampling)
- Integrates directly with Rolimons and Roblox APIs.
- Buys items within seconds of price drops.
- Fast purchase execution with direct API calls.

### Limited Analyzer
- Finds trends and identifies outliers in past sales data
- Informs trading decisions with z-score and margin analysis
- Cache sales data to classify price points quickly


---

## 🚀 Key Features

### Deal Sniping  
- **Efficient Monitoring** tracks market deals through GET requests with automated price refresh
- **Purchase Execution** sends request to purchase API endpoint when price below threshold
- **Flexible Automation** keeps system running through web errors and loss of connection.
- **Throttling** to prevent rate-limiting and sustain long-term operation.
- **Logging**: Every decision and action is tracked for further reference.

### Market Evaluation
- **Spikes & Dips**: Uses statistical measures (z-score, %CV) to identify trends in sales data and guide buying, trading, selling
- **Market Metrics**: Compares prices of item groups to past time periods for market insights  
- **Data Caching**: Precompute and store mean / standard deviation of past sales for fast querying

---

## 🛠️ Deployment Strategies

- Roblox incurs two taxes on financial actions
    1. Selling Assets - 30% fee
    2. Converting to USD - 75% tax
- Avoid first fee by exchanging limiteds with other players for profit
- Dodge second fee by keeping money in system, don't cash out until end
- Snipe limiteds → analyze promising items → sell optimally OR trade for better items 

---

## 🚧 Future Features  
- [ ] Cloud deployment option.
- [ ] **Sell-side algorithm** to liquidate owned assets optimally
- [ ] **Auto-Trader** - Automatically sends trades for favorable items.

---

#
