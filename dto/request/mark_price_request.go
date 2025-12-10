package request

type GetMarketPriceReq struct {
	Page int `json:"page" binding:"required,gte=1,lte=1000"`
	Size int `json:"size" binding:"required,gte=1,lte=1000"`
}
