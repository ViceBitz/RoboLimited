# Roblox Limited Sniper & Analyzer

RoboLimited is a trading algorithm that buys and sells undervalued Roblox virtual items using Rolimons and the Roblox API. It automates trades, snipes deals, and analyzes price trends using custom functions that weigh demand and pricing against recent sales activity. While designed for digital collectibles, the methodology maps to broader financial technology systems such as market monitoring and algorithmic trading.

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
- Auto-buys limiteds within **3 seconds** of appearing at a low price.  
- Integrates directly with **Rolimons** and **Roblox APIs**
- Formula-driven decisions (using margins and statistical sampling)
- Fast purchase execution with **low-latency price checks** and **logged-in webpage**

### Limited Analyzer
- **Finds trends** and **identifies outliers** in past sales data
- Utilizes **sales data caching** to classify price points quickly
- Track item sales prices across time period for big-picture trends


---

## 🚀 Key Features

### Deal Sniping  
- **Efficient Monitoring** tracks market deals through HTTP GET requests to API endpoints with automated price refresh and adjustment logic
- **Purchase Execution** buys item when price dips below adjustable margin and z-score thresholds
- **Flexible Automation** keeps system running through web errors and loss of connection.
- **Throttling** to prevent rate-limiting and sustain long-term operation.
- **Logging**: Every decision and action is tracked for post-trade analysis.

### Market Evaluation
- **Spikes & Dips**: Uses statistics (z-score, %CV) to identify trends in sales data and guide buying, trading, selling
- **Market Metrics**: Tracks prices of item groups across time period for market insights  
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
