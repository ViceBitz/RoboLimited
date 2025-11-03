// settings.go
package config

/*
// Roblox API base URLs
const (
	//Economy APIs
	RolimonsAPI            = "https://www.rolimons.com/itemapi/itemdetails"
	RobloxEconomyDetailsV2 = "https://economy.roblox.com/v2/assets/"
	RolimonsSite           = "https://www.rolimons.com/item/%s"
	RolimonsDeals          = "https://api.rolimons.com/market/v1/dealactivity"

	//Redacted for privacy, security, and ToS
	PurchaseAPI = "url-to-purchase-endpoint"
	AssetAPI = "url-to-asset-endpoint"
	ResellerAPI = "url-to-reseller-endpoint"
	

	//Evaluation Constraints (margin)
	RAPDipD    = 0.25 //Demand: margin below RAP to buy
	ValueDipD  = 0.35 //Demand: margin below Value to buy
	RAPDipND   = 0.30 //Non-demand: margin below RAP to buy
	ValueDipND = 0.35 //Non-demand: margin below Value to buy

	PriceRangeLow  = 10 //Price range of limiteds to consider
	PriceRangeHigh = 90

	RAPRangeLow = 200 //RAP range of limiteds to consider
	RAPRangeHigh = 1000000

	//Operation Modes
	LiveMoney = true //Run with real money (true) or simulated costs (false)

	DeepManipulationCheck = false //Run complex projected check (unnecessary if StrictBuyCondition true)
	StrictBuyCondition    = true  //Use z-score analysis for buy decisions

	//Data Caching (back up old file!)
	PopulateSalesData     = false //Updates all sales data (KEEP FALSE UNLESS UPDATE NEEDED, TAKES A LONG TIME)

	//Statistcal Z-score settings
	OutlierThreshold = 1   //SD from mean to consider item as projected
	DipThreshold     = 0.75   //SD from break even point (-0.3 / CoV) to consider a dip in price
	SellThreshold    = 0.2 //SD from mean to list item for sale
	LookbackPeriod   = 90  //Past number of days to consider for trend analysis

	//Iteration Cycles
	RefreshRate     = 1000       //Re-extract RAP / Value off Rolimon's API after this many rounds
	TotalIterations = 1000000 //Amount of cycles to run
	MonitorThrottle = 1000 //Milliseconds to yield per monitor update

	//Roblox pages
	RobloxCatalogBaseURL = "https://www.roblox.com/catalog/"
	RobloxHome           = "https://www.roblox.com/home"

	//Data Files
	ActionLogFile  = "data/actions.log" //Log of all buy actions
	ConsoleLogFile = "data/console.log" //Log of terminal output
	SalesStatsFile  = "data/sales_stats.csv"   //Mean & SD of past sales data of all items
	SalesDataFile = "data/sales_data.json" //Raw time-series sales data of all times

	//CSS Selectors
	PriceSelector         = "span.text-robux-lg"                                           //Best Price
	BuyButtonSelector     = "button.shopping-cart-buy-button.btn-growth-lg.PurchaseButton" //Buy Button
	ConfirmButtonSelector = "button.modal-button.btn-primary-md.btn-min-width"             //Confirm Button

	//Private Cookies
	RobloxCookie = ""
	RobloxId = 132153132

	//Web Agents
	UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36"
	ProxyFile = "data/proxies.txt" //Stores user & pass and proxy ports
	AgentsFile = "web/agents.txt" //Stores different user agents for requests

	//Logging
	LogConsole = false //Toggle print for processes & stats during execution
)
*/