# Roblox Limited Sniper & Analyzer

A system for analyzing and sniping Roblox limiteds using **Rolimons** and the **Roblox API**.  
Built for automated trading, deal-sniping, and technical analysis of limited items — runs 24/7 on a Raspberry Pi.  

---

## 📌 Features

### Price Sniper
- Auto-buys limiteds within **1 second** of appearing at a low price.  
- Integrates directly with **Rolimons** and **Roblox APIs**:  
  - Example endpoint: [`/v1/assets/{assetId}/resale-data`](https://economy.roblox.com/v1/assets/16652251/resale-data)
- Formula-driven decisions

---

## 🛠 Workflow

### Direct Sniping (suitable for few limiteds with high accuracy)
1. **Precompute** all limited data (RAP, Value, Projected, Demand).  
2. Fetch JSON from **Rolimon API**.  
3. Parse & filter items to target.  
4. Refresh every few cycles.  
5. Scrape live best prices from **Rolimons website**:  
   - Uses ChromeDB scraper objects with **mutexes**.  
   - Renews context after every scrape.  
   - CSS targeting of best-price element.  
6. Process results with **batch + multithreading**.  
   - Thread-safe dictionary to store best-price results.  
   - Built-in delays to prevent rate limiting.  
7. Error handling (continues running even on failures).  

### Deal Sniping (more efficient & broad)
1. Monitor **Rolimons Deals API** for price updates.  
   - `isRAP = 1` → refresh RAP for deal calculation.  
   - `isRAP = 0` → refresh best price.  
2. Compare updates against RAP & Value.  
3. Confirm best price with HTML scraping.  
4. Place purchase order via **Roblox site cookies**.  
5. Loop every few seconds for near real-time sniping (and to prevent rate limit).

#### Final Steps (in both methods):
 +  **Buy** if price dips 25% below RAP or 35% below Value.
 +  Track and log actions to a `.log` file.  

---

## ⚙️ Deployment

- Designed to run **24/7** on a Raspberry Pi.  
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

##
