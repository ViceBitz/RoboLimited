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
	PurchaseAPI = "CENSORED"

	//Evaluation Constraints (margin)
	RAPDipD    = 0.25 //Demand: margin below RAP to buy
	ValueDipD  = 0.35 //Demand: margin below Value to buy
	RAPDipND   = 0.30 //Non-demand: margin below RAP to buy
	ValueDipND = 0.35 //Non-demand: margin below Value to buy

	PriceRangeLow  = 10 //Price range of limiteds to consider
	PriceRangeHigh = 90

	//Operation Modes
	LiveMoney = true //Run with real money (true) or simulated costs (false)

	DeepManipulationCheck = false //Whether to run complex projected check before buy orders
	StrictBuyCondition    = true  //Use z-scores from standard deviations in buy checks
	PopulateSalesData     = false //Updates sales data for all items in data store (KEEP FALSE UNLESS UPDATE NEEDED, TAKES A LONG TIME)

	//Statistcal Z-score settings
	OutlierThreshold = 1   //SD from mean to consider item as projected
	DipThreshold     = 1.5   //SD from break even point (-0.3 / CoV) to consider a dip in price
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
	RobloxCookie = "_|WARNING:-DO-NOT-SHARE-THIS.--Sharing-this-will-allow-someone-to-log-in-as-you-and-to-steal-your-ROBUX-and-items.|_CAEaAhAB.1D734386DF702B8AAA97CAD7270BC9D7C9D4C4D2A1EAB4EB0008FC45E62A649FD689E33BE86C748A000C3D7F7264140F60A162289F1424E13E191B68D0CD76440C873784674EAA79D983E79F4156FD0BE11D302B53BB09062F35B98B254E1FFFB8927940AAB1DE65571202E59F30AEF72AE44A64DBEE9E7EDE3B69BF0612E0C8CBE3F0D3E62E3DAFD533F8D91367BC93FF78E8613B8095A762DB717304DC30D7113B12B9090E093C12B1DD6E8D9FE035318EDBFA56997EFEDBB4DD58576E7E315F40A009176D0CDD6666467A78D0C90EF011E7065CC97E64459706C2C71F17FC4F9F96DFA8961C674812B72D765599A5F43665C7613C2971C26F97975C032B07F0A0382C1212ACD0A9FF52C127D77ADD219B810FCFF8560100CFA698D2AEED90301852DEA2922EB7508830A55C0564F520F735E3AB461C787EBDF74B1A2F2FA36D8AA1CBD71B63BC0B1AC504DE9121D7BDCFE926CB385D407E1A7A0A0F018AEFCCF0138AC16557B0B1E21E2CF6750595DEE56288CF12DC067AC4815416F3E86A312C87FF12EE6DB60FADF57F3D51328048E88C13A892A4E5B5B7B66D262235AA07DB09AFAE69CDEE276D6DC11006490F3F928C8E9F82103793B2F43968EB06B0452B7131BE80EB4D94435FC5F79BE820F0887256114BE111B9E58C767B0B0144BFAC8633A638D22BFC9E80401A820D038190A3C4AB8DD4F319A1CA5F67D3D51F85CA66D03A7667B6BDB071696ED2880C5D242F5A73BD22231AB25CE7E61AFCE1703B872D8B2A1DF95FA1BFB8EDE3271A0B9FF2DC6BC854DB7D7FF0AB0E7587EB3481D1E30B9957803C57F9241E1F19F3DFBA2B20B53F4172FFE22871C36D77CF0C76BBEBF03A07D660DAAB2A4A360158B2BA727D589A32460A726CC9B97FCEED9CBAD4461D05FCFF8943DFE637FD39BBD5BCBB15CA6D58ACF4F94068658AA4A9C163D97DB270DD1EBE15E5BDE333DD1C9597C7A1BAE30B4CB7E18A5182A760C412993B24A3078DCC989E4F9B774D76D6D93A2B394853F628875E09BC25B0E0ADABEB07C52178E76E54B18CBCBD6AD9805B3886E170A0A6FAE66A56997834A019052A5A1F3958326840EE9F20794A0F94C51AD3EEFEED4CD5F9D5506AA5B718941F054AACEFD3EC24AABA788EBD6809E445417BA9AD5447C5C9D37285569C5392E6DC35C374B81A254D1F99085B6FD12478EA057D"
	RobloxId = 132153132
	//Web Agents
	UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36"
)
*/