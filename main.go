package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Customer represents a customer in the database
type Customer struct {
	Name         string   `bson:"name"`
	Phone        string   `bson:"phone"`
	OrderedItems []string `bson:"orderedItems"` // Stores ordered menu items
	TotalAmount  float64  `bson:"totalAmount"`  // Total amount for the customer's orders
}

// MenuItem represents a menu item in the database
type MenuItem struct {
	Name  string  `bson:"name"`
	Price float64 `bson:"price"`
}

var client *mongo.Client

// ConnectDB initializes a MongoDB client connection
func ConnectDB() *mongo.Client {
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	// Check the connection
	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}
	fmt.Println("Connected to MongoDB!")
	return client
}

// AddCustomer inserts a new customer into the database
func AddCustomer(name string, phone string) {
	collection := client.Database("restaurant").Collection("customers")
	customer := Customer{Name: name, Phone: phone, OrderedItems: []string{}, TotalAmount: 0}
	_, err := collection.InsertOne(context.TODO(), customer)
	if err != nil {
		log.Fatal("Error adding customer:", err)
	}
	fmt.Println("Customer added:", name)
}

// AddMenuItems adds predefined items to the menu collection
func AddMenuItems() {
	menuItems := []MenuItem{
		{"Pizza", 829.17},
		{"Burger", 497.17},
		{"Pasta", 663.17},
		{"Salad", 414.17},
		{"Sushi", 1078.17},
		{"Sandwich", 331.17},
		{"Tacos", 580.17},
		{"Steak", 1327.17},
		{"Fries", 248.17},
		{"Ice Cream", 290.50},
	}

	collection := client.Database("restaurant").Collection("menu")
	for _, item := range menuItems {
		_, err := collection.InsertOne(context.TODO(), item)
		if err != nil {
			log.Fatal("Error adding menu item:", err)
		}
	}
	fmt.Println("Menu items added to the database!")
}

// ShowMenu displays all items available in the menu
func ShowMenu() {
	collection := client.Database("restaurant").Collection("menu")
	cursor, err := collection.Find(context.TODO(), bson.D{})
	if err != nil {
		log.Fatal("Error retrieving menu:", err)
	}
	defer cursor.Close(context.TODO())

	fmt.Println("Menu:")
	for cursor.Next(context.TODO()) {
		var menuItem MenuItem
		err := cursor.Decode(&menuItem)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s, Price: Rs %.2f\n", menuItem.Name, menuItem.Price)
	}
}

// OrderItem allows a customer to order an item from the menu
func OrderItem(customerName string, itemName string) {
	customersCollection := client.Database("restaurant").Collection("customers")
	menuCollection := client.Database("restaurant").Collection("menu")

	// Check if the item exists in the menu
	var menuItem MenuItem
	err := menuCollection.FindOne(context.TODO(), bson.D{{"name", itemName}}).Decode(&menuItem)
	if err != nil {
		fmt.Printf("Item %s not found in menu\n", itemName)
		return
	}

	// Update customer's ordered items
	filter := bson.D{{"name", customerName}}
	update := bson.D{{"$push", bson.D{{"orderedItems", itemName}}}}

	result, err := customersCollection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		log.Fatal("Error ordering item:", err)
	}
	if result.MatchedCount > 0 {
		fmt.Printf("Customer %s ordered item: %s\n", customerName, itemName)
	} else {
		fmt.Printf("No customer found with name: %s\n", customerName)
	}
}

// PlaceOrder lets a customer choose multiple items from the menu
func PlaceOrder(customerName string) {
	reader := bufio.NewReader(os.Stdin)

	for {
		ShowMenu()
		fmt.Println("Enter the name of the item you want to order (or type 'done' to finish):")
		itemName, _ := reader.ReadString('\n')
		itemName = strings.TrimSpace(itemName)

		if strings.ToLower(itemName) == "done" {
			break
		}

		OrderItem(customerName, itemName)
	}
	CalculateAndStoreTotal(customerName) // Calculate total after order completion
}

func CalculateAndStoreTotal(customerName string) {
	customersCollection := client.Database("restaurant").Collection("customers")
	menuCollection := client.Database("restaurant").Collection("menu")

	// Retrieve the customer's orders
	var customer Customer
	err := customersCollection.FindOne(context.TODO(), bson.D{{"name", customerName}}).Decode(&customer)
	if err != nil {
		fmt.Println("Customer not found.")
		return
	}

	// Calculate total price based on the ordered items
	itemCounts := make(map[string]int)
	for _, itemName := range customer.OrderedItems {
		itemCounts[itemName]++
	}

	var totalAmount float64
	for itemName, count := range itemCounts {
		var menuItem MenuItem
		err := menuCollection.FindOne(context.TODO(), bson.D{{"name", itemName}}).Decode(&menuItem)
		if err == nil {
			totalAmount += menuItem.Price * float64(count)
		}
	}

	// Update the customer's total amount in the database
	filter := bson.D{{"name", customerName}}
	update := bson.D{{"$set", bson.D{{"totalAmount", totalAmount}}}}
	_, err = customersCollection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		log.Fatal("Error updating total amount:", err)
	}

	fmt.Printf("Thank you, %s! Your order has been received. Please wait while we prepare your meal...!\n", customerName)
}

// GetCustomers retrieves all customers from the database
func GetCustomers() {
	collection := client.Database("restaurant").Collection("customers")
	cursor, err := collection.Find(context.TODO(), bson.D{})
	if err != nil {
		log.Fatal("Error retrieving customers:", err)
	}
	defer cursor.Close(context.TODO())

	fmt.Println("Total Customers:")
	for cursor.Next(context.TODO()) {
		var customer Customer
		err := cursor.Decode(&customer)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Name: %s, Phone: %s, Orders: %v, Total Amount: Rs %.2f\n", customer.Name, customer.Phone, customer.OrderedItems, customer.TotalAmount) // Include total amount in display
	}
}

func main() {
	// Initialize the MongoDB connection
	client = ConnectDB()
	if client == nil {
		fmt.Println("Failed to connect to MongoDB")
		return
	}
	defer client.Disconnect(context.TODO())

	// Add sample menu items (only runs once; you can comment it out if items are already in the database)
	AddMenuItems()

	// Add a sample customer
	AddCustomer("Gadapa Raghavendra", "1234567890")

	// Allow customer to place an order from the menu
	fmt.Println("\nWelcome to the Restaurant Ordering System!")
	PlaceOrder("Gadapa Raghavendra")

	// Display customers and their orders
	GetCustomers()
}
