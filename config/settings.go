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
	RAPDipF   = 0.25 //0.25 //% below RAP to buy
	ValueDipF = 0.35 //0.35 //% below Value to buy

	SellMargin = 1000 //0.1 //% return on investment

	PriceRangeLow  = 2000
	PriceRangeHigh = 10000 //Price range of limiteds to consider
	MaxLimiteds    = 50    //Number of limiteds to consider
	HighDemand     = true  //Only consider high demand items

	ValueCycles = 5 //Re-extract RAP / Value off Rolimon's API after this many rounds

	//Catalog page
	RobloxCatalogBaseURL = "https://www.roblox.com/catalog"

	//Files
	ActionLogFile = "data/actions.log"
)
