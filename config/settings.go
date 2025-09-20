// settings.go
package config

// Roblox API base URLs
const (
	/**
	[[SNIPER & MONITOR SETTINGS]]
	*/

	//Economy APIs
	RolimonsAPI            = "https://www.rolimons.com/itemapi/itemdetails"
	RobloxEconomyDetailsV2 = "https://economy.roblox.com/v2/assets/"
	RolimonsSite           = "https://www.rolimons.com/item/%s"
	RolimonsDeals          = "https://api.rolimons.com/market/v1/dealactivity"

	//Limited Sniper Constraints (for margin eval)
	RAPDipD    = 0.25 //Demand % below RAP to buy
	ValueDipD  = 0.35 //Demand % below Value to buy
	RAPDipND   = 0.30 //Non-demand % below RAP to buy
	ValueDipND = 0.35 //Non-demand % below Value to buy

	/* DEPRECATED SellMargin = 1000 //0.1 //% return on investment */

	PriceRangeLow  = 500 //Price range of limiteds to consider
	PriceRangeHigh = 1500

	LiveMoney = true //Run with real money (true) or simulated costs (false)

	DeepManipulationCheck = false //Whether to run complex projected check before buy orders
	StrictBuyCondition    = true  //Use z-scores from standard deviations in buy checks
	PopulateSalesData     = false //Updates sales data for all items in data store (KEEP FALSE UNLESS NECESSARY)

	OutlierThreshold = 1  //Standard deviations to consider item as projected
	DipThreshold     = -2 //Standard deviations to consider a dip in price (for z-score eval)

	//Price manipulation analysis
	LookbackPeriod = 90 //Past number of days to consider when scanning for projecteds
	RefreshRate    = 5  //Re-extract RAP / Value off Rolimon's API after this many rounds

	//Roblox pages
	RobloxCatalogBaseURL = "https://www.roblox.com/catalog/"
	RobloxHome           = "https://www.roblox.com/home"

	//Files
	ActionLogFile = "data/actions.log" //Log of all buy actions
	SalesDataFile = "data/sales.csv"   //Mean & SD of past sales data of all items

	//CSS Selectors
	PriceSelector         = "span.text-robux-lg"                                           //Best Price
	BuyButtonSelector     = "button.shopping-cart-buy-button.btn-growth-lg.PurchaseButton" //Buy Button
	ConfirmButtonSelector = "button.modal-button.btn-primary-md.btn-min-width"             //Confirm Button

	//Private Cookies
	RobloxCookie = ""
)
