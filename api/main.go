package api

import (
	"database/sql"

	"github.com/go-pg/pg"
	"github.com/gorilla/mux"
)

// SetupAPIRoutes takes a mux router and a database connection and creates all the API routes for the API
func SetupAPIRoutes(router *mux.Router, ormDB *pg.DB, properDB *sql.DB) {
	// Base Products
	router.HandleFunc("/base_product/{id:[0-9]+}", buildSingleBaseProductHandler(ormDB)).Methods("GET")

	// Products
	router.HandleFunc("/products", buildProductListHandler(ormDB)).Methods("GET")
	router.HandleFunc("/product/{sku:[a-zA-Z]+}", buildProductExistenceHandler(properDB)).Methods("HEAD")
	router.HandleFunc("/product/{sku:[a-zA-Z]+}", buildSingleProductHandler(ormDB)).Methods("GET")
	router.HandleFunc("/product/{sku:[a-zA-Z]+}", buildProductUpdateHandler(ormDB)).Methods("PUT")
	router.HandleFunc("/product", buildProductCreationHandler(ormDB)).Methods("POST")
	router.HandleFunc("/product/{sku:[a-zA-Z]+}", buildProductDeletionHandler(ormDB)).Methods("DELETE")

	// Product Attribute Values
	router.HandleFunc("/product_attributes/{attribute_id:[0-9]+}/value", buildProductAttributeValueCreationHandler(ormDB)).Methods("POST")

	// Orders
	router.HandleFunc("/orders", buildOrderListHandler(ormDB)).Methods("GET")
	router.HandleFunc("/order", buildOrderCreationHandler(ormDB)).Methods("POST")

}