# Roblox Limited Sniper & Analyzer

A system for analyzing and sniping Roblox limiteds using **Rolimons** and the **Roblox API** that is built for automated trading, deal-sniping, and technical analysis of limited items. While intended for a digital collectibles marketplace, concepts map directly to broader financial technology systems such as market monitoring & algorithmic trading.

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
- **Identifies outliers** and **finds trends** in past sales data of items
- Utilizes **sales data caching** to classify price points quickly
- Track item sales prices across time period for big-picture trends




---

## 🚀 Key Features

### Deal Scanning  
- **Efficient Monitoring** tracks market deals through HTTP GET requests to API endpoints with automated price refresh and adjustment logic
- **Auto Purchase** buys item when price dips below margin and z-score thresholds
- **Flexible Automation** keeps system running through web errors and loss of connection.
- **Throttling** to prevent rate-limiting and sustain long-term operation.

### Market Evaluation
- **Spikes & Dips**: Finds abnormalities in sales data to guide buying, trading, selling
- **Item Filtering**: Excludes manipulated assets based on sales data in past month.
- **Market Metrics**: Tracks prices of item groups across time period for market insights  
- **Data Caching**: Precompute and store mean / standard deviation of past sales for fast querying

### Execution Layer
- **Signal Validation**: Confirms opportunities by comparing expected to actual market listing.
- **Ready Webpage**: Maintains Chrome webpage logged into account to cut down navigation time.  
- **Automated Transactions**: Executes purchases with safeguards against false data.  
- **Logging**: Every decision and action is tracked for post-trade analysis.

---

## 🚧 Future Features
- [ ] Add web dashboard for live tracking.  
- [ ] Smarter buy strategy with ML-driven prediction.  
- [ ] Cloud deployment option.
- [ ] **Limited Analyzer** - Predicts which limiteds will skyrocket in value; Incorporate past trades, buy/sell data, seasonal trends, standard stock market technical analysis.
- [ ] **Auto-Trader** - Automatically sends trades for favorable items.

---

#
