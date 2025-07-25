package main

type OrderItem struct {
	OrderItemId int
	OrderId     int
	GameId      int
	UnitPrice   float64
	Quantity    int
	TotalPrice  float64
}

type Order struct {
	OrderId  int
	Username string
	Items    []OrderItem
}

type CreateOrderRequest struct {
	Username string
}

type CreateOrderItemRequest struct {
	OrderId   int
	GameId    int
	UnitPrice float64
	Quantity  int
}

type GetOrderItemResponse struct {
	OrderItemId int     `json:"order_item_id"`
	GameId      int     `json:"game_id"`
	GameName    string  `json:"game_name"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	TotalPrice  float64 `json:"total_price"`
}

type GetOrderResponse struct {
	OrderId    int                    `json:"order_id"`
	Username   string                 `json:"username"`
	Items      []GetOrderItemResponse `json:"items"`
	TotalPrice float64                `json:"total_price"`
}

type GetOrdersResponse []GetOrderResponse
