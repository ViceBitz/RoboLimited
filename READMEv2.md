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
- Auto-buys limiteds within **1 second** of appearing at a low price.  
- Integrates directly with **Rolimons** and **Roblox APIs**
- Formula-driven decisions (using margins and statistical z-scores)

### Limited Analyzer
- Detects price manipulated items on RAP based on past sales data
- Finds dips in prices for strict buy condition on sniper

---

## 🚀 Key Features

### Deal Scanning  
- **Efficient Monitoring** tracking market deals with automated price refresh and adjustment logic through HTTP GET requests to known API endpoints.
- **Resilient Automation** with error handling and fault tolerance.
- **Throttling and rate-limit protection** to sustain long-term operation.
- **Console Messages** to provide constant status reports and information on current operation.

### Market Evaluation
- **Rule-Based Filters**: Excludes manipulated assets based on sales data in past month.
- **Demand-Aware Strategies**: Adjust thresholds based on popularity and liquidity signals.
- **Spikes & Dips**: Identifies abnormalities in sales data to guide buying and selling

### Execution Layer
- **Signal Validation**: Confirms opportunities comparing expected to actual market listing.  
- **Automated Transactions**: Executes purchases programmatically with safeguards against false data.  
- **Logging**: Every decision and action is tracked for post-trade analysis.


---

## ⚙️ Deployment

- Designed to run **24/7**.  
- Logs trades, profit, and system actions.
- Test strategy across long periods to confirm validity.  

---

## 🚧 Future Features
- [ ] Add web dashboard for live tracking.  
- [ ] Smarter buy strategy with ML-driven prediction.  
- [ ] Cloud deployment option.
- [ ] **Limited Analyzer** - Predicts which limiteds will skyrocket in value; Incorporate past trades, buy/sell data, seasonal trends, standard stock market technical analysis.
- [ ] **Auto-Trader** - Automatically sends trades for favorable items.

---

#
