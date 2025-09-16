# Roblox Limited Sniper & Analyzer

A system for analyzing and sniping Roblox limiteds using **Rolimons** and the **Roblox API**.  
Built for automated trading, deal-sniping, and technical analysis of limited items — runs 24/7 on a Raspberry Pi.  

---

# 📌 Features

### Price Sniper
- Auto-buys limiteds within **1 second** of appearing at a low price.  
- Integrates directly with **Rolimons** and **Roblox APIs**
- Formula-driven decisions (using margins)

---

# 🛠 Workflow


## Price Sniper

### Direct Sniping (suitable for few limiteds with high accuracy)
1. **Precompute** all limited data (RAP, Value, Projected, IsDemand)   
   - Fetch JSON from **Rolimon API**.
   - Refresh every few cycles.
3. Parse & filter items to target.  
5. Scrape live best prices from **Rolimons website**:  
   - Uses ChromeDB scraper objects with **mutexes**.  
   - Renews context after every scrape.  
   - CSS targeting of best-price element.  
6. Process results with **batch + multithreading**.  
   - Thread-safe dictionary to store best-price results.  
   - Built-in delays to prevent rate limiting.
7. *Do Common Steps...*  

### Deal Sniping (more efficient & broad)
1. Monitor **Rolimons Deals API** for price updates.  
   - `isRAP = 1` → refresh RAP for deal calculation.  
   - `isRAP = 0` → update new best price.  
2. Update RAP or best price in table
3. *Do Common Steps...*

#### Common Steps (in both methods):
 +  **Buy Decision:** buy or do nothing
    - Simple Evaluation: if price is 25% below RAP or 35% below value
    - Tapered Evaluation: interpolate margin from 40% to 20% (or 50% to 30% for value) on small limiteds (~100R) to big limiteds (3000-1000R)
    - Demand Evaluation: lower margins (25%/35%) on high demand items compared to non-popular items (30%/35%)
    - **Combine both Tapered & Demand for General Evaluation
 +  Confirm Buy Action on two conditions:
    - Refresh RAP / Value via Rolimon’s item details API
    - Real best price listed on Roblox website
 +  Execute Purchase
    - Log in with Roblox Cookie (sensitive) and navigate to item page
    - Final validation on shown best price against RAP / Value
    - Click buy / confirm buttons on real-time Roblox site
 +  Track and log actions to a `.log` file.
 +  Loop every few seconds for near real-time sniping (and to prevent rate limit).

*🎭 Both methods exclude projected (price-manipulated) limited items.<br>
⚠️ Program handles all errors & exceptions (continues running even on failures).*

---

# ⚙️ Deployment

- Designed to run **24/7** on a Raspberry Pi.  
- Logs trades, profit, and system actions.
- Test strategy across long periods to confirm validity.  

---

# 🚧 Future Features
- [ ] Add web dashboard for live tracking.  
- [ ] Smarter buy strategy with ML-driven prediction.  
- [ ] Cloud deployment option.
- [ ] **Limited Analyzer** - Predicts which limiteds will skyrocket in value; Incorporate past trades, buy/sell data, seasonal trends, standard stock market technical analysis.
- [ ] **Auto-Trader** - Automatically sends trades for favorable items.

---

#
