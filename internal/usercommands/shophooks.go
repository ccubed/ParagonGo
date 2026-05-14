package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// ShopListRequest is the value threaded through OnShopList handlers.
// Stock is the effective inventory for this seller; handlers may add,
// remove, or modify entries (including Price) before the list or buy
// command uses it. Exactly one of SellerMob and SellerUser is non-nil.
// Buyer is always non-nil.
type ShopListRequest struct {
	Stock      characters.Shop
	Buyer      *users.UserRecord
	SellerMob  *mobs.Mob
	SellerUser *users.UserRecord
}

// OnShopList is fired once per seller before the stock list is rendered
// (list command) or matched against a purchase request (buy command).
// Modules register handlers here to dynamically modify shop inventory.
//
// Handlers may add items to Stock, remove items to block purchase, or
// modify ShopItem fields (including Price) to override pricing.
//
// Example registration from a module:
//
//	usercommands.OnShopList.Register(func(r usercommands.ShopListRequest) usercommands.ShopListRequest {
//	    if r.SellerMob != nil && r.SellerMob.MobId == mySpecialMobId {
//	        r.Stock = append(r.Stock, characters.ShopItem{ItemId: 42, Price: 100})
//	    }
//	    return r
//	})
var OnShopList util.Hook[ShopListRequest]
