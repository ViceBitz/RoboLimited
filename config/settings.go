// settings.go
package config

// Roblox API base URLs
const (
	//Economy APIs
	RolimonsAPI            = "https://www.rolimons.com/itemapi/itemdetails"
	RobloxEconomyDetailsV2 = "https://economy.roblox.com/v2/assets/"
	RolimonsSite           = "https://www.rolimons.com/item/%s"
	RolimonsDeals          = "https://api.rolimons.com/market/v1/dealactivity"

	//Evaluation Constraints (margin)
	RAPDipD    = 0.25 //Demand: margin below RAP to buy
	ValueDipD  = 0.35 //Demand: margin below Value to buy
	RAPDipND   = 0.30 //Non-demand: margin below RAP to buy
	ValueDipND = 0.35 //Non-demand: margin below Value to buy

	PriceRangeLow  = 10 //Price range of limiteds to consider
	PriceRangeHigh = 200

	//Operation Modes
	LiveMoney = true //Run with real money (true) or simulated costs (false)

	DeepManipulationCheck = false //Whether to run complex projected check before buy orders
	StrictBuyCondition    = true  //Use z-scores from standard deviations in buy checks
	PopulateSalesData     = false //Updates sales data for all items in data store (KEEP FALSE UNLESS NECESSARY)

	//Statistcal Z-score settings
	OutlierThreshold = 1   //SD from mean to consider item as projected
	DipThreshold     = 1   //SD from break even point (-0.3 / CoV) to consider a dip in price
	SellThreshold    = 0.2 //SD from mean to list item for sale
	LookbackPeriod   = 90  //Past number of days to consider when scanning for projecteds

	//Iteration Cycles
	RefreshRate     = 5       //Re-extract RAP / Value off Rolimon's API after this many rounds
	TotalIterations = 1000000 //Amount of cycles to run

	//Roblox pages
	RobloxCatalogBaseURL = "https://www.roblox.com/catalog/"
	RobloxHome           = "https://www.roblox.com/home"

	//Files
	ActionLogFile  = "data/actions.log" //Log of all buy actions
	ConsoleLogFile = "data/console.log" //Log of terminal output
	SalesDataFile  = "data/sales.csv"   //Mean & SD of past sales data of all items

	//CSS Selectors
	PriceSelector         = "span.text-robux-lg"                                           //Best Price
	BuyButtonSelector     = "button.shopping-cart-buy-button.btn-growth-lg.PurchaseButton" //Buy Button
	ConfirmButtonSelector = "button.modal-button.btn-primary-md.btn-min-width"             //Confirm Button

	//Private Cookies
	RobloxCookie = "_|WARNING:-DO-NOT-SHARE-THIS.--Sharing-this-will-allow-someone-to-log-in-as-you-and-to-steal-your-ROBUX-and-items.|_CAEaAhAB.A8ABD012F2CB14CDCB513A29297FD46F66F2E7831484079E42DDFDC422C3A3729D12C9028C0A74AA078B899F857FC293A85A6FBBB03F5258A0B72EAD5108C912790BD85CD88027ED8453273E21AC8646A1F008B2566C1DD460DE22C4DE86F9523944663B70F846B0A6A8F673E8106F5625336FEB976896C90BC5AC4607871F211CBEE7BE992CE425BB604A15F203B9FA3D2C0813A34E663694B32B6ACB77FB5E2779A97B9EE088007AB31CD960552D3B5A5DBA05145FD5BCA507F4D38858A693218EB76814DE96223391DFDB0D5804D6376AA2071D62256F52ADFCBCFFCE5DD9C7ACB1BECF458A5610E9B3E6CA26B45F3994DFE043C44555768B8FC00373FED687692F9AE45338A1F61F98848728FD655D3E5067D7D2A013CDE5A39602F784EC0C96570F79366993DB595500F1D5A749D7AFB10ACAA67E7403A8101038EE23968F8A4BCF3A081C6B77C3C69835E951A8E31D3064CD29F9F4E1414DA0BAABC7DDBE7FD5E00866633AC9C04A7EE09B34AB5159C3758800DD5404697D5974838FE917E1189F1A1DB888415EC200FD110D903171F804F905D2AB8C8BEEEA35AC8E7867CF81DFC0E98A684CF0E6A6498B0D3D9196C2D6DFDD90A70B2C2D6979D01F5B92C4AAC26BE13F7CB3FF7B33B0411268085024132374C6E93B5D0D2BC6231FFD57B8D8724DEC456924C8E1E82BB08AAECD1B15571AC911F5EB4592742C6EE5414B192E25908FD24B1DB643E444D502DC513FC6D226B3E81EC4FE65CD973320563C76018CB6A557DBE693D6B3B010DB3FFF79E4F6155788BC5A7CAF789CB884120BFDDA774F4F6E02D4077DC4144BE66811159F0F90CD7E556CF9B2D2803B0A02D3B21B0EEF4E175BBF42786902CE8EE326F7339D9E170A35A574A4E1035ECF012B54E8A6AC6777BCB853F5455E865B958B48C3294F9FFF1306F6A15B9A6A2173ADE1AACF3AF15D91C1795B9B401B3408A58579761F6A67EE23316445E1041A936F758E68AA097CDB71A596BB90CD55B9DBB3BED95E0E5688D2A5BADCFD19B2D9348D6E9A8C3EB87074834101302A4AB9732FDDFF7F9C69EE52C7797F5E58DB64F4775499F3420311F02636CDEA41D7C233307EEF9D3839199B8772E4671CA2769B9D8E89696BB470EC924A765BB86553882046458B347B36A3B208C2534E0B6DF455C37B4F5328D935D1493519CA1D5B0EAF3B7793E112CBBC7360DEA5567B4A44AE4898"

	//Web Agents
	UserAgent = "Mozilla/5.0"
)
