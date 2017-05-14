package main

import (
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/go-pg/pg"
	"github.com/gorilla/mux"
)

var db *pg.DB
var templates = template.Must(template.ParseGlob("templates/*"))

// HomeHandler serves up our basic web page
func HomeHandler(res http.ResponseWriter, req *http.Request) {
	if val, ok := req.Header["User-Agent"]; ok {
		log.Printf("User-Agent: %v", val)
	}
	indexPage, err := ioutil.ReadFile("templates/home.html")
	if err != nil {
		log.Printf("error occurred reading indexPage: %v\n", err)
	}
	renderTemplates(res, "Dairycart", string(indexPage))
}

// notImplementedHandler is used for endpoints that haven't been implemented yet.
func notImplementedHandler(res http.ResponseWriter, req *http.Request) {
	res.WriteHeader(http.StatusTeapot)
}

func main() {
	// init stuff
	domainName := os.Getenv("DAIRYCART_DOMAIN")
	if domainName == "" {
		domainName = "localhost"
	}

	dbURL := os.Getenv("DAIRYCART_DB_URL")
	dbOptions, err := pg.ParseURL(dbURL)
	if err != nil {
		log.Fatalf("Error parsing database URL: %v", err)
	}
	db = pg.Connect(dbOptions)
	router := mux.NewRouter()

	// // https://github.com/go-pg/pg/wiki/FAQ#how-can-i-view-queries-this-library-generates
	// db.OnQueryProcessed(func(event *pg.QueryProcessedEvent) {
	// 	query, err := event.FormattedQuery()
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	log.Printf("%s %s", time.Since(event.StartTime), query)
	// })

	// Basic business
	router.HandleFunc("/", HomeHandler).Methods("GET")

	// Base Products
	router.HandleFunc("/base_product/{id}", SingleBaseProductHandler).Methods("GET")

	// Products
	router.HandleFunc("/products", ProductListHandler).Methods("GET")
	router.HandleFunc("/product/{sku}", ProductExistenceHandler).Methods("HEAD")
	router.HandleFunc("/product/{sku}", SingleProductHandler).Methods("GET")
	router.HandleFunc("/product/{sku}", ProductUpdateHandler).Methods("PUT")
	router.HandleFunc("/product", ProductCreationHandler).Methods("POST")
	router.HandleFunc("/product/{sku}", ProductDeletionHandler).Methods("DELETE")

	// Product Attribute Values
	router.HandleFunc("/product_attributes/{attribute_id}/value", notImplementedHandler).Methods("POST")

	// Orders
	router.HandleFunc("/orders", OrderListHandler).Methods("GET")
	router.HandleFunc("/order", OrderCreationHandler).Methods("POST")

	// serve 'em up a lil' sauce
	http.Handle("/", router)
	log.Println("Listening at port 8080")
	http.ListenAndServe(":8080", nil)
}
