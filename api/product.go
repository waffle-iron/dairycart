package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/imdario/mergo"
	"github.com/lib/pq"
	"github.com/pkg/errors"
)

const (
	skuExistenceQuery         = `SELECT EXISTS(SELECT 1 FROM products WHERE sku = $1 and archived_at is null);`
	skuDeletionQuery          = `UPDATE products SET archived_at = NOW() WHERE sku = $1 AND p.archived_at IS NULL;`
	skuRetrievalQuery         = `SELECT * FROM products WHERE sku = $1 AND archived_at IS NULL;`
	skuJoinRetrievalQuery     = `SELECT * FROM products p JOIN product_progenitors g ON p.product_progenitor_id = g.id WHERE p.sku = $1 AND p.archived_at IS NULL;`
	allProductsRetrievalQuery = `SELECT * FROM products p JOIN product_progenitors g ON p.product_progenitor_id = g.id WHERE p.id IS NOT NULL AND p.archived_at IS NULL;`
	productUpdateQuery        = `UPDATE products SET "product_progenitor_id"=$1, "sku"=$2, "name"=$3, "upc"=$4, "quantity"=$5, "on_sale"=$6, "price"=$7, "sale_price"=$8, "updated_at"='NOW()' WHERE "id"=$9;`
	productCreationQuery      = `INSERT INTO products ("product_progenitor_id", "sku", "name", "upc", "quantity", "on_sale", "price", "sale_price") VALUES ($1, $2, $3, $4, $5, $6, $7, $8);`
)

// Product describes something a user can buy
type Product struct {
	ProductProgenitor

	// Basic Info
	ID                  int64      `json:"id"`
	ProductProgenitorID int64      `json:"product_progenitor_id"`
	SKU                 string     `json:"sku"`
	Name                string     `json:"name"`
	UPC                 NullString `json:"upc"`
	Quantity            int        `json:"quantity"`

	// Pricing Fields
	OnSale    bool        `json:"on_sale"`
	Price     float32     `json:"price"`
	SalePrice NullFloat64 `json:"sale_price"`

	// // Housekeeping
	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  pq.NullTime `json:"updated_at"`
	ArchivedAt pq.NullTime `json:"-"`
}

// generateScanArgs generates an array of pointers to struct fields for sql.Scan to populate
func (p *Product) generateScanArgs() []interface{} {
	return []interface{}{
		&p.ID,
		&p.ProductProgenitorID,
		&p.SKU,
		&p.Name,
		&p.UPC,
		&p.Quantity,
		&p.OnSale,
		&p.Price,
		&p.SalePrice,
		&p.CreatedAt,
		&p.UpdatedAt,
		&p.ArchivedAt,
	}
}

// generateJoinScanArgs does some stuff TODO: write better docs
func (p *Product) generateJoinScanArgs() []interface{} {
	productScanArgs := p.generateScanArgs()
	progenitorScanArgs := p.ProductProgenitor.generateScanArgs()
	return append(productScanArgs, progenitorScanArgs...)
}

// ProductsResponse is a product response struct
type ProductsResponse struct {
	ListResponse
	Data []Product `json:"data"`
}

func loadProductInput(req *http.Request) (*Product, error) {
	product := &Product{}
	decoder := json.NewDecoder(req.Body)
	defer req.Body.Close()
	err := decoder.Decode(product)

	return product, err
}

// productExistsInDB will return whether or not a product/attribute/etc with a given identifier exists in the database
func productExistsInDB(db *sql.DB, sku string) (bool, error) {
	var exists string

	err := db.QueryRow(skuExistenceQuery, sku).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, errors.Wrap(err, "Error querying for product")
	}

	return exists == "true", err
}

func buildProductExistenceHandler(db *sql.DB) http.HandlerFunc {
	// ProductExistenceHandler handles requests to check if a sku exists
	return func(res http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		sku := vars["sku"]

		productExists, err := rowExistsInDB(db, "products", "sku", sku)
		// productExists, err := productExistsInDB(db, sku)
		if err != nil {
			respondThatRowDoesNotExist(req, res, "product", "sku", sku)
			return
		}

		responseStatus := http.StatusNotFound
		if productExists {
			responseStatus = http.StatusOK
		}
		res.WriteHeader(responseStatus)
	}
}

// retrievePlainProductFromDB retrieves a product with a given SKU from the database
func retrievePlainProductFromDB(db *sql.DB, sku string) (*Product, error) {
	product := &Product{}
	scanArgs := product.generateScanArgs()

	err := db.QueryRow(skuRetrievalQuery, sku).Scan(scanArgs...)
	if err == sql.ErrNoRows {
		return product, errors.Wrap(err, "Error querying for product")
	}

	return product, nil
}

// retrieveProductFromDB retrieves a product with a given SKU from the database
func retrieveProductFromDB(db *sql.DB, sku string) (*Product, error) {
	product := &Product{}
	scanArgs := product.generateJoinScanArgs()

	err := db.QueryRow(skuJoinRetrievalQuery, sku).Scan(scanArgs...)
	if err == sql.ErrNoRows {
		return product, errors.Wrap(err, "Error querying for product")
	}

	return product, err
}

func buildSingleProductHandler(db *sql.DB) http.HandlerFunc {
	// SingleProductHandler is a request handler that returns a single Product
	return func(res http.ResponseWriter, req *http.Request) {
		sku := mux.Vars(req)["sku"]

		product, err := retrieveProductFromDB(db, sku)
		if err != nil {
			respondThatRowDoesNotExist(req, res, "product", "sku", sku)
			return
		}

		json.NewEncoder(res).Encode(product)
	}
}

func retrieveProductsFromDB(db *sql.DB) ([]Product, error) {
	var products []Product

	rows, err := db.Query(allProductsRetrievalQuery)
	if err != nil {
		return nil, errors.Wrap(err, "Error encountered querying for products")
	}
	defer rows.Close()
	for rows.Next() {
		var product Product
		_ = rows.Scan(product.generateJoinScanArgs()...)
		products = append(products, product)
	}
	return products, nil
}

func buildProductListHandler(db *sql.DB) http.HandlerFunc {
	// productListHandler is a request handler that returns a list of products
	return func(res http.ResponseWriter, req *http.Request) {
		products, err := retrieveProductsFromDB(db)
		if err != nil {
			notifyOfInternalIssue(res, err, "retrieve products from the database")
			return
		}

		productsResponse := &ProductsResponse{
			ListResponse: ListResponse{
				Page:  1,  // TODO: implement proper paging :(
				Limit: 25, // ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
				Count: len(products),
			},
			Data: products,
		}
		json.NewEncoder(res).Encode(productsResponse)
	}
}

func deleteProductBySku(db *sql.DB, req *http.Request, res http.ResponseWriter, sku string) error {
	// can't delete a product that doesn't exist!
	_, err := rowExistsInDB(db, "products", "sku", sku)
	// _, err := productExistsInDB(db, sku)
	if err != nil {
		respondThatRowDoesNotExist(req, res, "product", "sku", sku)
	}

	_, err = db.Exec(skuDeletionQuery, sku)
	return err
}

func buildProductDeletionHandler(db *sql.DB) http.HandlerFunc {
	// ProductDeletionHandler is a request handler that deletes a single product
	return func(res http.ResponseWriter, req *http.Request) {
		sku := mux.Vars(req)["sku"]
		deleteProductBySku(db, req, res, sku)
		json.NewEncoder(res).Encode("OK")
	}
}

func updateProductInDatabase(db *sql.DB, up *Product) error {
	_, err := db.Exec(productUpdateQuery, up.ProductProgenitorID, up.SKU, up.Name, up.UPC, up.Quantity, up.OnSale, up.Price, up.SalePrice, up.ID)
	return err
}

func buildProductUpdateHandler(db *sql.DB) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		// ProductUpdateHandler is a request handler that can update products
		sku := mux.Vars(req)["sku"]

		// can't update a product that doesn't exist!
		_, err := rowExistsInDB(db, "products", "sku", sku)
		// _, err := productExistsInDB(db, sku)
		if err != nil {
			respondThatRowDoesNotExist(req, res, "product", "sku", sku)
			return
		}
		existingProduct, _ := retrievePlainProductFromDB(db, sku) // eating the error here because we're already certain the sku exists

		updatedProduct, err := loadProductInput(req)
		if err != nil {
			notifyOfInvalidRequestBody(res, err)
			return
		}

		updatedProduct.ID = existingProduct.ID
		if err := mergo.Merge(updatedProduct, existingProduct); err != nil {
			notifyOfInternalIssue(res, err, "merge updated product with existing product")
			return
		}

		err = updateProductInDatabase(db, updatedProduct)
		if err != nil {
			notifyOfInternalIssue(res, err, "update product in database")
			return
		}

		json.NewEncoder(res).Encode(updatedProduct)
	}
}

// createProduct takes a marshalled Product object and creates an entry for it and a base_product in the database
func createProduct(db *sql.DB, new *Product) error {
	_, err := db.Exec(productCreationQuery, new.ProductProgenitorID, new.SKU, new.Name, new.UPC, new.Quantity, new.OnSale, new.Price, new.SalePrice)
	return err
}

func buildProductCreationHandler(db *sql.DB) http.HandlerFunc {
	// ProductCreationHandler is a product creation handler
	return func(res http.ResponseWriter, req *http.Request) {
		progenitorID := mux.Vars(req)["progenitor_id"]

		// we should be able to safely eat this error because gorilla/mux should validate the id
		// id, _ := strconv.ParseInt(progenitorID, 10, 64)

		progenitorExists, err := rowExistsInDB(db, "product_progenitors", "id", progenitorID)
		// progenitorExists, err := productProgenitorExistsInDB(db, id)
		if err != nil || !progenitorExists {
			respondThatProductProgenitorDoesNotExist(req, res, progenitorID)
			return
		}

		newProduct, err := loadProductInput(req)
		if err != nil {
			notifyOfInvalidRequestBody(res, err)
			return
		}

		err = createProduct(db, newProduct)
		if err != nil {
			notifyOfInternalIssue(res, err, "insert product in database")
			return
		}
	}
}
