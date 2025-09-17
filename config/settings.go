// settings.go
package config

// Roblox API base URLs
const (
	//Economy APIs
	RolimonsAPI            = "https://www.rolimons.com/itemapi/itemdetails"
	RobloxEconomyDetailsV2 = "https://economy.roblox.com/v2/assets/"
	RolimonsSite           = "https://www.rolimons.com/item/%s"
	RolimonsDeals          = "https://api.rolimons.com/market/v1/dealactivity"

	//Limited Sniper Constraints
	RAPDipD    = 0.25 //Demand % below RAP to buy
	ValueDipD  = 0.35 //Demand % below Value to buy
	RAPDipND   = 0.30 //Non-demand % below RAP to buy
	ValueDipND = 0.35 //Non-demand % below Value to buy

	/* DEPRECATED SellMargin = 1000 //0.1 //% return on investment */

	PriceRangeLow  = 400 //Price range of limiteds to consider
	PriceRangeHigh = 1500

	LiveMoney = false //Run with real money (true) or simulated costs (false)

	MaxLimiteds           = 50   //Number of limiteds to consider (for direct monitoring only)
	HighDemand            = true //Only consider high demand items (for direct monitoring only)
	ProjectedPriceHistory = 1000 //Amount of latest price points to consider when scanning for projecteds

	RefreshRate = 5 //Re-extract RAP / Value off Rolimon's API after this many rounds

	//Catalog page
	RobloxCatalogBaseURL = "https://www.roblox.com/catalog/"

	//Files
	ActionLogFile = "data/actions.log"

	//CSS Selectors
	PriceSelector         = "span.text-robux-lg"                                           //Best Price
	BuyButtonSelector     = "button.shopping-cart-buy-button.btn-growth-lg.PurchaseButton" //Buy Button
	ConfirmButtonSelector = "button.modal-button.btn-primary-md.btn-min-width"             //Confirm Button

	//Private Cookies
	RobloxCookie = ""
)
