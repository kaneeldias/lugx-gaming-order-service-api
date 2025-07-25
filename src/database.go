package main

import (
	"database/sql"
	"fmt"
	_ "github.com/jackc/pgx/v5/stdlib"
	"log"
	"os"
	"sync"
)

var (
	db   *sql.DB
	once sync.Once
)

func createConnection() (*sql.DB, error) {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_HOST"),
		os.Getenv("POSTGRES_PORT"),
		os.Getenv("POSTGRES_DB"))

	log.Println("Connecting to PostgreSQL...")

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, fmt.Errorf("unable to open database connection: %v", err)
	}

	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %v", err)
	}

	log.Println("Successfully connected to the database")
	return db, nil
}

func GetDatabaseConnection() (*sql.DB, error) {
	var err error
	once.Do(func() {
		db, err = createConnection()
	})

	if err != nil {
		return nil, fmt.Errorf("error creating database connection: %w", err)
	}

	if db == nil {
		return nil, fmt.Errorf("failed to create or retrieve database connection")
	}

	return db, err
}

func InitializeDatabase() error {
	db, err := GetDatabaseConnection()
	if err != nil {
		return fmt.Errorf("error creating database connection: %w", err)
	}

	var row int
	err = db.QueryRow(`
		SELECT order_id
		FROM Orders
		WHERE order_id = $1
	`, 1).Scan(&row)
	if err == nil {
		log.Println("Orders table already exists. Skipping initialization.")
		return nil
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS Orders (
		    order_id SERIAL PRIMARY KEY,
		    username VARCHAR(255) NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("error creating Orders table: %w", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS OrderItems (
			order_item_id SERIAL PRIMARY KEY,
			order_id INTEGER REFERENCES Orders(order_id),
			game_id INTEGER REFERENCES Games(game_id),
			unit_price DECIMAL(5,2) NOT NULL,
			quantity INTEGER NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("error creating OrderItems table: %w", err)
	}

	order1, err := CreateOrder(CreateOrderRequest{
		Username: "Kaneel",
	})
	if err != nil {
		return fmt.Errorf("error creating order 1: %w", err)
	}

	order2, err := CreateOrder(CreateOrderRequest{
		Username: "Dias",
	})
	if err != nil {
		return fmt.Errorf("error creating order 2: %w", err)
	}

	_, err = CreateOrderItem(CreateOrderItemRequest{
		OrderId:   order1.OrderId,
		GameId:    1,
		UnitPrice: 26.95,
		Quantity:  10,
	})
	if err != nil {
		return fmt.Errorf("error creating order item for order 1: %w", err)
	}

	_, err = CreateOrderItem(CreateOrderItemRequest{
		OrderId:   order1.OrderId,
		GameId:    2,
		UnitPrice: 14.99,
		Quantity:  5,
	})
	if err != nil {
		return fmt.Errorf("error creating order item for order 1: %w", err)
	}

	_, err = CreateOrderItem(CreateOrderItemRequest{
		OrderId:   order2.OrderId,
		GameId:    1,
		UnitPrice: 26.95,
		Quantity:  3,
	})
	if err != nil {
		return fmt.Errorf("error creating order item for order 2: %w", err)
	}

	log.Println("Database initialized.")
	return nil
}

func CreateOrder(request CreateOrderRequest) (Order, error) {
	db, err := GetDatabaseConnection()
	if err != nil {
		return Order{}, fmt.Errorf("error creating database connection: %w", err)
	}

	var id int
	err = db.QueryRow(`
		INSERT INTO Orders (username)
		VALUES ($1)
		RETURNING order_id
	`, request.Username).Scan(&id)
	if err != nil {
		return Order{}, fmt.Errorf("error inserting game category: %w", err)
	}

	order := Order{
		OrderId:  id,
		Username: request.Username,
		Items:    []OrderItem{},
	}

	log.Printf("Order created: %+v\n", order)
	return order, nil
}

func CreateOrderItem(request CreateOrderItemRequest) (OrderItem, error) {
	db, err := GetDatabaseConnection()
	if err != nil {
		return OrderItem{}, fmt.Errorf("error creating database connection: %w", err)
	}

	var id int
	err = db.QueryRow(`
		INSERT INTO OrderItems (order_id, game_id, unit_price, quantity)
		VALUES ($1, $2, $3, $4)
		RETURNING order_item_id
	`, request.OrderId, request.GameId, request.UnitPrice, request.Quantity).Scan(&id)
	if err != nil {
		return OrderItem{}, fmt.Errorf("error inserting order item: %w", err)
	}

	orderItem := OrderItem{
		OrderItemId: id,
		OrderId:     request.OrderId,
		GameId:      request.GameId,
		UnitPrice:   request.UnitPrice,
		Quantity:    request.Quantity,
		TotalPrice:  request.UnitPrice * float64(request.Quantity),
	}

	log.Printf("Order item created: %+v\n", orderItem)
	return orderItem, nil
}

func GetAllOrders() (GetOrdersResponse, error) {
	db, err := GetDatabaseConnection()
	if err != nil {
		return nil, fmt.Errorf("error creating database connection: %w", err)
	}

	rows, err := db.Query(`
		SELECT o.order_id, o.username, oi.order_item_id, oi.game_id, g.name, oi.unit_price, oi.quantity
		FROM Orders o
		LEFT JOIN OrderItems oi ON o.order_id = oi.order_id
		LEFT JOIN Games g ON oi.game_id = g.game_id
	`)
	if err != nil {
		return nil, fmt.Errorf("error querying games: %w", err)
	}
	defer rows.Close()

	var orders GetOrdersResponse
	for rows.Next() {
		var order GetOrderResponse
		var item GetOrderItemResponse

		err := rows.Scan(&order.OrderId, &order.Username, &item.OrderItemId, &item.GameId, &item.GameName, &item.UnitPrice, &item.Quantity)
		if err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}

		item.TotalPrice = item.UnitPrice * float64(item.Quantity)

		// Check if order already exists in orders
		exists := false
		for i, o := range orders {
			if o.OrderId == order.OrderId {
				exists = true
				o.Items = append(o.Items, item)
				o.TotalPrice += item.TotalPrice
				orders[i] = o
				break
			}
		}
		if !exists {
			order.TotalPrice = item.TotalPrice
			order.Items = append(order.Items, item)
			orders = append(orders, order)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %w", err)
	}

	log.Printf("Fetched %v orders\n", orders)
	return orders, nil
}
