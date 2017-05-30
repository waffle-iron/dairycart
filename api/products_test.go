package api

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	sqlmock "gopkg.in/DATA-DOG/go-sqlmock.v1"
)

const (
	badSKUUpdateJSON = `{"sku": "pooƃ ou sᴉ nʞs sᴉɥʇ"}`
	lolFloats        = 12.34000015258789

	exampleProductCreationInput = `
		{
			"sku": "skateboard",
			"name": "Skateboard",
			"upc": "1234567890",
			"quantity": 123,
			"price": 12.34,
			"cost": 5,
			"description": "This is a skateboard. Please wear a helmet.",
			"taxable": true,
			"product_weight": 8,
			"product_height": 7,
			"product_width": 6,
			"product_length": 5,
			"package_weight": 4,
			"package_height": 3,
			"package_width": 2,
			"package_length": 1
		}
	`

	exampleProductCreationInputWithAttributes = `
		{
			"sku": "skateboard",
			"name": "Skateboard",
			"upc": "1234567890",
			"quantity": 123,
			"price": 12.34,
			"cost": 5,
			"description": "This is a skateboard. Please wear a helmet.",
			"taxable": true,
			"product_weight": 8,
			"product_height": 7,
			"product_width": 6,
			"product_length": 5,
			"package_weight": 4,
			"package_height": 3,
			"package_width": 2,
			"package_length": 1,
			"attributes_and_values": [{
				"name": "something",
				"values": [
					"one",
					"two",
					"three"
				]
			}]
		}
	`

	exampleProductUpdateInput = `
		{
			"sku": "example",
			"name": "Test",
			"quantity": 666,
			"upc": "1234567890",
			"price": 12.34
		}
	`
)

var plainProductHeaders []string
var examplePlainProductData []driver.Value
var productJoinHeaders []string
var exampleProductJoinData []driver.Value
var exampleProduct *Product
var exampleUpdatedProduct *Product

func init() {
	plainProductHeaders = []string{"id", "product_progenitor_id", "sku", "name", "upc", "quantity", "price", "cost", "created_at", "updated_at", "archived_at"}
	examplePlainProductData = []driver.Value{10, 2, "skateboard", "Skateboard", "1234567890", 123, 12.34, 5.00, exampleTime, nil, nil}

	productJoinHeaders = []string{"id", "product_progenitor_id", "sku", "name", "upc", "quantity", "price", "cost", "created_at", "updated_at", "archived_at", "id", "name", "description", "taxable", "price", "product_weight", "product_height", "product_width", "product_length", "package_weight", "package_height", "package_width", "package_length", "created_at", "updated_at", "archived_at"}
	exampleProductJoinData = []driver.Value{10, 2, "skateboard", "Skateboard", "1234567890", 123, 12.34, 5.00, exampleTime, nil, nil, 2, "Skateboard", "This is a skateboard. Please wear a helmet.", false, 99.99, 8, 7, 6, 5, 4, 3, 2, 1, exampleTime, nil, nil}
	exampleProduct = &Product{
		ProductProgenitor: ProductProgenitor{
			ID:            2,
			Name:          "Skateboard",
			Price:         99.99,
			Description:   "This is a skateboard. Please wear a helmet.",
			ProductWeight: 8,
			ProductHeight: 7,
			ProductWidth:  6,
			ProductLength: 5,
			PackageWeight: 4,
			PackageHeight: 3,
			PackageWidth:  2,
			PackageLength: 1,
			CreatedAt:     exampleTime,
		},
		ID:                  10,
		ProductProgenitorID: 2,
		SKU:                 "skateboard",
		Name:                "Skateboard",
		UPC:                 NullString{sql.NullString{String: "1234567890", Valid: true}},
		Quantity:            123,
		Price:               12.34,
		Cost:                5.00,
		CreatedAt:           exampleTime,
	}
	exampleUpdatedProduct = &Product{
		ID:        exampleProduct.ID,
		SKU:       "example",
		Name:      "Test",
		UPC:       NullString{sql.NullString{String: "1234567890", Valid: true}},
		Quantity:  666,
		Cost:      5.00,
		Price:     lolFloats,
		CreatedAt: exampleTime,
	}

}

func setExpectationsForProductExistence(mock sqlmock.Sqlmock, SKU string, exists bool, err error) {
	exampleRows := sqlmock.NewRows([]string{""}).AddRow(strconv.FormatBool(exists))
	skuExistenceQuery := buildProductExistenceQuery(SKU)
	mock.ExpectQuery(formatQueryForSQLMock(skuExistenceQuery)).
		WithArgs(SKU).
		WillReturnRows(exampleRows).
		WillReturnError(err)
}

func setExpectationsForProductListQuery(mock sqlmock.Sqlmock, err error) {
	exampleRows := sqlmock.NewRows(productJoinHeaders).
		AddRow(exampleProductJoinData...).
		AddRow(exampleProductJoinData...).
		AddRow(exampleProductJoinData...)

	allProductsRetrievalQuery, _ := buildAllProductsRetrievalQuery(defaultQueryFilter)
	mock.ExpectQuery(formatQueryForSQLMock(allProductsRetrievalQuery)).WillReturnRows(exampleRows).WillReturnError(err)
}

func setExpectationsForProductRetrieval(mock sqlmock.Sqlmock, err error) {
	exampleRows := sqlmock.NewRows(productJoinHeaders).AddRow(exampleProductJoinData...)

	skuRetrievalQuery := formatQueryForSQLMock(buildCompleteProductRetrievalQuery(exampleSKU))
	mock.ExpectQuery(skuRetrievalQuery).WithArgs(exampleSKU).WillReturnRows(exampleRows).WillReturnError(err)
}

func setExpectationsForProductUpdate(mock sqlmock.Sqlmock, p *Product, err error) {
	exampleRows := sqlmock.NewRows(plainProductHeaders).
		AddRow(examplePlainProductData...)

	productUpdateQuery, queryArgs := buildProductUpdateQuery(p)
	args := argsToDriverValues(queryArgs)
	mock.ExpectQuery(formatQueryForSQLMock(productUpdateQuery)).
		WithArgs(args...).
		WillReturnRows(exampleRows).
		WillReturnError(err)
}

func setExpectationsForProductCreation(mock sqlmock.Sqlmock, p *Product, err error) {
	exampleRows := sqlmock.NewRows([]string{"id"}).AddRow(exampleProduct.ID)
	productCreationQuery, args := buildProductCreationQuery(exampleProduct)
	queryArgs := argsToDriverValues(args)
	mock.ExpectQuery(formatQueryForSQLMock(productCreationQuery)).
		WithArgs(queryArgs...).
		WillReturnRows(exampleRows).
		WillReturnError(err)
}

func setExpectationsForSingleProductRetrieval(mock sqlmock.Sqlmock, err error) {
	exampleRows := sqlmock.NewRows(productJoinHeaders).AddRow(exampleProductJoinData...)
	skuJoinRetrievalQuery := buildCompleteProductRetrievalQuery(exampleProduct.SKU)
	mock.ExpectQuery(formatQueryForSQLMock(skuJoinRetrievalQuery)).
		WithArgs(exampleProduct.SKU).
		WillReturnRows(exampleRows).
		WillReturnError(err)
}

func setExpectationsForProductUpdateHandler(mock sqlmock.Sqlmock, p *Product, err error) {
	exampleRows := sqlmock.NewRows(plainProductHeaders).AddRow(examplePlainProductData...)
	productUpdateQuery, _ := buildProductUpdateQuery(exampleProduct)
	mock.ExpectQuery(formatQueryForSQLMock(productUpdateQuery)).
		WithArgs(
			p.Cost,
			p.Name,
			p.Price,
			p.Quantity,
			p.SKU,
			p.UPC.String,
			p.ID,
		).WillReturnRows(exampleRows).
		WillReturnError(err)
}

func TestValidateProductUpdateInputWithValidInput(t *testing.T) {
	expected := &Product{
		SKU:      exampleSKU,
		Name:     "Test",
		UPC:      NullString{sql.NullString{String: "1234567890", Valid: true}},
		Quantity: 666,
		Price:    12.34,
		Cost:     0,
	}

	req := httptest.NewRequest("GET", "http://example.com", strings.NewReader(exampleProductUpdateInput))
	actual, err := validateProductUpdateInput(req)

	assert.Nil(t, err)
	assert.Equal(t, expected, actual, "valid product input should parse into a proper product struct")
}

func TestValidateProductUpdateInputWithInvalidInput(t *testing.T) {
	exampleInput := strings.NewReader(`{"testing": true}`)

	req := httptest.NewRequest("GET", "http://example.com", exampleInput)
	_, err := validateProductUpdateInput(req)

	assert.NotNil(t, err)
}

func TestValidateProductUpdateInputWithCompletelyInvalidInput(t *testing.T) {
	exampleInput := strings.NewReader(`{"testing":}`)

	req := httptest.NewRequest("GET", "http://example.com", exampleInput)
	_, err := validateProductUpdateInput(req)

	assert.NotNil(t, err)
}

func TestValidateProductUpdateInputWithInvalidSKU(t *testing.T) {
	exampleInput := strings.NewReader(badSKUUpdateJSON)

	req := httptest.NewRequest("GET", "http://example.com", exampleInput)
	_, err := validateProductUpdateInput(req)

	assert.NotNil(t, err)
}

func TestRetrieveProductFromDB(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()
	setExpectationsForProductRetrieval(mock, nil)

	actual, err := retrieveProductFromDB(db, exampleSKU)
	assert.Nil(t, err)
	assert.Equal(t, exampleProduct, actual, "expected and actual products should match")
	ensureExpectationsWereMet(t, mock)
}

func TestRetrieveProductFromDBWhenDBReturnsError(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()
	setExpectationsForProductRetrieval(mock, sql.ErrNoRows)

	_, err = retrieveProductFromDB(db, exampleSKU)
	assert.NotNil(t, err)
	ensureExpectationsWereMet(t, mock)
}

func TestRetrieveProductsFromDB(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()
	setExpectationsForProductListQuery(mock, nil)

	products, err := retrieveProductsFromDB(db, defaultQueryFilter)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(products), "there should be 3 products")
	ensureExpectationsWereMet(t, mock)
}

func TestRetrieveProductsFromDBWhenDBReturnsError(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()
	setExpectationsForProductListQuery(mock, sql.ErrNoRows)

	_, err = retrieveProductsFromDB(db, defaultQueryFilter)
	assert.NotNil(t, err)
	ensureExpectationsWereMet(t, mock)
}

func TestDeleteProductBySKU(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()

	skuDeletionQuery := buildProductDeletionQuery(exampleSKU)
	mock.ExpectExec(formatQueryForSQLMock(skuDeletionQuery)).
		WithArgs(exampleSKU).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = deleteProductBySKU(db, exampleSKU)
	assert.Nil(t, err)
	ensureExpectationsWereMet(t, mock)
}

func TestUpdateProductInDatabase(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()
	setExpectationsForProductUpdate(mock, exampleProduct, nil)

	err = updateProductInDatabase(db, exampleProduct)
	assert.Nil(t, err)
	ensureExpectationsWereMet(t, mock)
}

func TestCreateProductInDB(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()
	mock.ExpectBegin()
	setExpectationsForProductCreation(mock, exampleProduct, nil)
	mock.ExpectCommit()

	tx, err := db.Begin()
	assert.Nil(t, err)

	newID, err := createProductInDB(tx, exampleProduct)
	assert.Nil(t, err)
	assert.Equal(t, exampleProduct.ID, newID, "createProductInDB should return the created ID")

	err = tx.Commit()
	assert.Nil(t, err)
	ensureExpectationsWereMet(t, mock)
}

func TestValidateProductCreationInput(t *testing.T) {
	expected := &ProductCreationInput{
		Description:   "This is a skateboard. Please wear a helmet.",
		Taxable:       true,
		ProductWeight: 8,
		ProductHeight: 7,
		ProductWidth:  6,
		ProductLength: 5,
		PackageWeight: 4,
		PackageHeight: 3,
		PackageWidth:  2,
		PackageLength: 1,
		SKU:           "skateboard",
		Name:          "Skateboard",
		UPC:           "1234567890",
		Quantity:      123,
		Price:         12.34,
		Cost:          5,
	}

	req := httptest.NewRequest("GET", "http://example.com", strings.NewReader(exampleProductCreationInput))
	actual, err := validateProductCreationInput(req)

	assert.Nil(t, err)
	assert.Equal(t, expected, actual, "valid product input should parse into a proper product struct")
}

func TestValidateProductCreationInputWithEmptyInput(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com", nil)
	_, err := validateProductCreationInput(req)

	assert.NotNil(t, err)
}

func TestValidateProductCreationInputWithInvalidInput(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com", strings.NewReader(exampleGarbageInput))
	_, err := validateProductCreationInput(req)

	assert.NotNil(t, err)
}

func TestValidateProductCreationInputWithInvalidSKU(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com", strings.NewReader(badSKUUpdateJSON))
	_, err := validateProductCreationInput(req)

	assert.NotNil(t, err)
}

//////////////////////////////////////////////////////////////
//                        ,-.             __                //
//  HTTP                ,'   `---.___.---'  `.              //
//    handler         ,'   ,-                 `-._          //
//      tests       ,'    /                       \         //
//               ,\/     /                        \\        //
//           )`._)>)     |                         \\       //
//           `>,'    _   \                  /       |\      //
//             )      \   |   |            |        |\\     //
//    .   ,   /        \  |    `.          |        | ))    //
//    \`. \`-'          )-|      `.        |        /((     //
//     \ `-`   a`     _/ ;\ _     )`-.___.--\      /  `'    //
//      `._         ,'    \`j`.__/        \  `.    \        //
//        / ,    ,'       _)\   /`        _) ( \   /        //
//        \__   /        /nn_) (         /nn__\_) (         //
//          `--'           /nn__\             /nn__\        //
//////////////////////////////////////////////////////////////

func TestProductExistenceHandlerWithExistingProduct(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()
	res, router := setupMockRequestsAndMux(db)
	setExpectationsForProductExistence(mock, exampleSKU, true, nil)

	req, err := http.NewRequest("HEAD", "/v1/product/example", nil)
	assert.Nil(t, err)
	router.ServeHTTP(res, req)

	assert.Equal(t, 200, res.Code, "status code should be 200")
	ensureExpectationsWereMet(t, mock)
}

func TestProductExistenceHandlerWithNonexistentProduct(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()
	res, router := setupMockRequestsAndMux(db)
	setExpectationsForProductExistence(mock, "unreal", false, nil)

	req, err := http.NewRequest("HEAD", "/v1/product/unreal", nil)
	assert.Nil(t, err)
	router.ServeHTTP(res, req)

	assert.Equal(t, 404, res.Code, "status code should be 404")
	ensureExpectationsWereMet(t, mock)
}

func TestProductExistenceHandlerWithExistenceCheckerError(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()
	res, router := setupMockRequestsAndMux(db)
	setExpectationsForProductExistence(mock, "unreal", false, arbitraryError)

	req, err := http.NewRequest("HEAD", "/v1/product/unreal", nil)
	assert.Nil(t, err)
	router.ServeHTTP(res, req)

	assert.Equal(t, 404, res.Code, "status code should be 404")
	ensureExpectationsWereMet(t, mock)
}

func TestProductRetrievalHandlerWithExistingProduct(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()
	res, router := setupMockRequestsAndMux(db)
	setExpectationsForSingleProductRetrieval(mock, nil)

	req, err := http.NewRequest("GET", "/v1/product/skateboard", nil)
	assert.Nil(t, err)

	router.ServeHTTP(res, req)
	assert.Equal(t, 200, res.Code, "status code should be 200")
	ensureExpectationsWereMet(t, mock)
}

func TestProductRetrievalHandlerWithDBError(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()
	res, router := setupMockRequestsAndMux(db)
	setExpectationsForSingleProductRetrieval(mock, sql.ErrNoRows)

	req, err := http.NewRequest("GET", "/v1/product/skateboard", nil)
	assert.Nil(t, err)

	router.ServeHTTP(res, req)
	assert.Equal(t, 404, res.Code, "status code should be 404")
	ensureExpectationsWereMet(t, mock)
}

func TestProductListHandler(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()
	res, router := setupMockRequestsAndMux(db)
	setExpectationsForProductListQuery(mock, nil)

	req, err := http.NewRequest("GET", "/v1/products", nil)
	assert.Nil(t, err)

	router.ServeHTTP(res, req)
	assert.Equal(t, 200, res.Code, "status code should be 200")

	expected := &ProductsResponse{
		ListResponse: ListResponse{
			Page:  1,
			Limit: 25,
			Count: 3,
		},
	}

	actual := &ProductsResponse{}
	err = json.NewDecoder(strings.NewReader(res.Body.String())).Decode(actual)
	assert.Nil(t, err)

	assert.Equal(t, expected.Page, actual.Page, "expected and actual product pages should be equal")
	assert.Equal(t, expected.Limit, actual.Limit, "expected and actual product limits should be equal")
	assert.Equal(t, expected.Count, actual.Count, "expected and actual product counts should be equal")
	assert.Equal(t, uint64(len(actual.Data)), actual.Count, "actual product counts and product response count field should be equal")
	ensureExpectationsWereMet(t, mock)
}

func TestProductListHandlerWithDBError(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()
	res, router := setupMockRequestsAndMux(db)
	setExpectationsForProductListQuery(mock, arbitraryError)

	req, err := http.NewRequest("GET", "/v1/products", nil)
	assert.Nil(t, err)

	router.ServeHTTP(res, req)
	assert.Equal(t, 500, res.Code, "status code should be 500")
	ensureExpectationsWereMet(t, mock)
}

func TestProductUpdateHandler(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()
	res, router := setupMockRequestsAndMux(db)
	setExpectationsForProductExistence(mock, exampleSKU, true, nil)
	setExpectationsForProductRetrieval(mock, nil)
	setExpectationsForProductUpdateHandler(mock, exampleUpdatedProduct, nil)

	req, err := http.NewRequest("PUT", "/v1/product/example", strings.NewReader(exampleProductUpdateInput))
	assert.Nil(t, err)
	router.ServeHTTP(res, req)

	assert.Equal(t, 200, res.Code, "status code should be 200")
	ensureExpectationsWereMet(t, mock)
}

func TestProductUpdateHandlerWithNonexistentProduct(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()
	res, router := setupMockRequestsAndMux(db)
	setExpectationsForProductExistence(mock, exampleSKU, false, nil)

	req, err := http.NewRequest("PUT", "/v1/product/example", strings.NewReader(exampleProductUpdateInput))
	assert.Nil(t, err)
	router.ServeHTTP(res, req)

	assert.Equal(t, 404, res.Code, "status code should be 404")
	ensureExpectationsWereMet(t, mock)
}

func TestProductUpdateHandlerWithInputValidationError(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()
	res, router := setupMockRequestsAndMux(db)
	setExpectationsForProductExistence(mock, exampleSKU, true, nil)

	req, err := http.NewRequest("PUT", "/v1/product/example", strings.NewReader(badSKUUpdateJSON))
	assert.Nil(t, err)
	router.ServeHTTP(res, req)

	assert.Equal(t, 400, res.Code, "status code should be 400")
	ensureExpectationsWereMet(t, mock)
}

func TestProductUpdateHandlerWithDBErrorRetrievingProduct(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()
	res, router := setupMockRequestsAndMux(db)
	setExpectationsForProductExistence(mock, exampleSKU, true, nil)
	setExpectationsForProductRetrieval(mock, arbitraryError)

	req, err := http.NewRequest("PUT", "/v1/product/example", strings.NewReader(exampleProductUpdateInput))
	assert.Nil(t, err)
	router.ServeHTTP(res, req)

	assert.Equal(t, 500, res.Code, "status code should be 500")
	ensureExpectationsWereMet(t, mock)
}

func TestProductUpdateHandlerWithDBErrorUpdatingProduct(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()
	res, router := setupMockRequestsAndMux(db)
	setExpectationsForProductExistence(mock, exampleSKU, true, nil)
	setExpectationsForProductRetrieval(mock, nil)
	setExpectationsForProductUpdateHandler(mock, exampleUpdatedProduct, arbitraryError)

	req, err := http.NewRequest("PUT", "/v1/product/example", strings.NewReader(exampleProductUpdateInput))
	assert.Nil(t, err)
	router.ServeHTTP(res, req)

	assert.Equal(t, 500, res.Code, "status code should be 500")
	ensureExpectationsWereMet(t, mock)
}

func TestProductDeletionHandlerWithExistentProduct(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()
	res, router := setupMockRequestsAndMux(db)
	setExpectationsForProductExistence(mock, exampleSKU, true, nil)

	skuDeletionQuery := buildProductDeletionQuery(exampleSKU)
	mock.ExpectExec(formatQueryForSQLMock(skuDeletionQuery)).
		WithArgs(exampleSKU).
		WillReturnResult(sqlmock.NewResult(1, 1))

	req, err := http.NewRequest("DELETE", "/v1/product/example", nil)
	assert.Nil(t, err)
	router.ServeHTTP(res, req)

	assert.Equal(t, 200, res.Code, "status code should be 200")
	ensureExpectationsWereMet(t, mock)
}

func TestProductDeletionHandlerWithNonexistentProduct(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()
	res, router := setupMockRequestsAndMux(db)
	setExpectationsForProductExistence(mock, exampleSKU, false, nil)

	req, err := http.NewRequest("DELETE", "/v1/product/example", nil)
	assert.Nil(t, err)
	router.ServeHTTP(res, req)

	assert.Equal(t, 404, res.Code, "status code should be 404")
	ensureExpectationsWereMet(t, mock)
}

func TestProductCreationHandler(t *testing.T) {
	expectedProgenitor := &ProductProgenitor{
		ID:            2,
		Name:          "Skateboard",
		Description:   "This is a skateboard. Please wear a helmet.",
		Taxable:       true,
		Price:         lolFloats,
		ProductWeight: 8,
		ProductHeight: 7,
		ProductWidth:  6,
		ProductLength: 5,
		PackageWeight: 4,
		PackageHeight: 3,
		PackageWidth:  2,
		PackageLength: 1,
		CreatedAt:     exampleTime,
	}

	expectedProduct := &Product{
		ProductProgenitor: ProductProgenitor{
			ID:            2,
			Name:          "Skateboard",
			Price:         lolFloats,
			Description:   "This is a skateboard. Please wear a helmet.",
			ProductWeight: 8,
			ProductHeight: 7,
			ProductWidth:  6,
			ProductLength: 5,
			PackageWeight: 4,
			PackageHeight: 3,
			PackageWidth:  2,
			PackageLength: 1,
			CreatedAt:     exampleTime,
		},
		ID:                  10,
		ProductProgenitorID: 2,
		SKU:                 "skateboard",
		Name:                "Skateboard",
		UPC:                 NullString{sql.NullString{String: "1234567890", Valid: true}},
		Quantity:            123,
		Price:               lolFloats,
		Cost:                5.0,
		CreatedAt:           exampleTime,
	}

	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()
	res, router := setupMockRequestsAndMux(db)
	setExpectationsForProductExistence(mock, "skateboard", false, nil)

	mock.ExpectBegin()
	setExpectationsForProductProgenitorCreation(mock, expectedProgenitor, nil)
	setExpectationsForProductAttributeCreation(mock, expectedCreatedProductAttribute, nil)
	setExpectationsForProductAttributeValueCreation(mock, expectedCreatedProductAttribute.Values[0], nil)
	setExpectationsForProductAttributeValueCreation(mock, expectedCreatedProductAttribute.Values[1], nil)
	setExpectationsForProductAttributeValueCreation(mock, expectedCreatedProductAttribute.Values[2], nil)
	setExpectationsForProductCreation(mock, expectedProduct, nil)
	mock.ExpectCommit()

	req, err := http.NewRequest("POST", "/v1/product", strings.NewReader(exampleProductCreationInputWithAttributes))
	assert.Nil(t, err)
	router.ServeHTTP(res, req)

	assert.Equal(t, 200, res.Code, "status code should be 200")
	ensureExpectationsWereMet(t, mock)
}

func TestProductCreationHandlerWhereCommitReturnsAnError(t *testing.T) {
	expectedProgenitor := &ProductProgenitor{
		ID:            2,
		Name:          "Skateboard",
		Description:   "This is a skateboard. Please wear a helmet.",
		Taxable:       true,
		Price:         lolFloats,
		ProductWeight: 8,
		ProductHeight: 7,
		ProductWidth:  6,
		ProductLength: 5,
		PackageWeight: 4,
		PackageHeight: 3,
		PackageWidth:  2,
		PackageLength: 1,
		CreatedAt:     exampleTime,
	}

	expectedProduct := &Product{
		ProductProgenitor: ProductProgenitor{
			ID:            2,
			Name:          "Skateboard",
			Price:         lolFloats,
			Description:   "This is a skateboard. Please wear a helmet.",
			ProductWeight: 8,
			ProductHeight: 7,
			ProductWidth:  6,
			ProductLength: 5,
			PackageWeight: 4,
			PackageHeight: 3,
			PackageWidth:  2,
			PackageLength: 1,
			CreatedAt:     exampleTime,
		},
		ID:                  10,
		ProductProgenitorID: 2,
		SKU:                 "skateboard",
		Name:                "Skateboard",
		UPC:                 NullString{sql.NullString{String: "1234567890", Valid: true}},
		Quantity:            123,
		Price:               lolFloats,
		Cost:                5.0,
		CreatedAt:           exampleTime,
	}

	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()
	res, router := setupMockRequestsAndMux(db)
	setExpectationsForProductExistence(mock, "skateboard", false, nil)

	mock.ExpectBegin()
	setExpectationsForProductProgenitorCreation(mock, expectedProgenitor, nil)
	setExpectationsForProductAttributeCreation(mock, expectedCreatedProductAttribute, nil)
	setExpectationsForProductAttributeValueCreation(mock, expectedCreatedProductAttribute.Values[0], nil)
	setExpectationsForProductAttributeValueCreation(mock, expectedCreatedProductAttribute.Values[1], nil)
	setExpectationsForProductAttributeValueCreation(mock, expectedCreatedProductAttribute.Values[2], nil)
	setExpectationsForProductCreation(mock, expectedProduct, nil)
	mock.ExpectCommit().WillReturnError(arbitraryError)

	req, err := http.NewRequest("POST", "/v1/product", strings.NewReader(exampleProductCreationInputWithAttributes))
	assert.Nil(t, err)
	router.ServeHTTP(res, req)

	assert.Equal(t, 500, res.Code, "status code should be 500")
	ensureExpectationsWereMet(t, mock)
}

func TestProductCreationHandlerWhereTransactionFailsToBegin(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()
	res, router := setupMockRequestsAndMux(db)
	setExpectationsForProductExistence(mock, "skateboard", false, nil)

	mock.ExpectBegin().WillReturnError(arbitraryError)

	req, err := http.NewRequest("POST", "/v1/product", strings.NewReader(exampleProductCreationInputWithAttributes))
	assert.Nil(t, err)
	router.ServeHTTP(res, req)

	assert.Equal(t, 500, res.Code, "status code should be 500")
	ensureExpectationsWereMet(t, mock)
}

func TestProductCreationHandlerWithoutAttributes(t *testing.T) {
	expectedProgenitor := &ProductProgenitor{
		ID:            2,
		Name:          "Skateboard",
		Description:   "This is a skateboard. Please wear a helmet.",
		Taxable:       true,
		Price:         lolFloats,
		ProductWeight: 8,
		ProductHeight: 7,
		ProductWidth:  6,
		ProductLength: 5,
		PackageWeight: 4,
		PackageHeight: 3,
		PackageWidth:  2,
		PackageLength: 1,
		CreatedAt:     exampleTime,
	}

	expectedProduct := &Product{
		ProductProgenitor: ProductProgenitor{
			ID:            2,
			Name:          "Skateboard",
			Price:         lolFloats,
			Description:   "This is a skateboard. Please wear a helmet.",
			ProductWeight: 8,
			ProductHeight: 7,
			ProductWidth:  6,
			ProductLength: 5,
			PackageWeight: 4,
			PackageHeight: 3,
			PackageWidth:  2,
			PackageLength: 1,
			CreatedAt:     exampleTime,
		},
		ID:                  10,
		ProductProgenitorID: 2,
		SKU:                 "skateboard",
		Name:                "Skateboard",
		UPC:                 NullString{sql.NullString{String: "1234567890", Valid: true}},
		Quantity:            123,
		Price:               lolFloats,
		Cost:                5.0,
		CreatedAt:           exampleTime,
	}

	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()
	res, router := setupMockRequestsAndMux(db)
	setExpectationsForProductExistence(mock, "skateboard", false, nil)

	mock.ExpectBegin()
	setExpectationsForProductProgenitorCreation(mock, expectedProgenitor, nil)
	setExpectationsForProductCreation(mock, expectedProduct, nil)
	mock.ExpectCommit()

	req, err := http.NewRequest("POST", "/v1/product", strings.NewReader(exampleProductCreationInput))
	assert.Nil(t, err)
	router.ServeHTTP(res, req)

	assert.Equal(t, 200, res.Code, "status code should be 200")
	ensureExpectationsWereMet(t, mock)
}

func TestProductCreationHandlerWithInvalidProductInput(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()
	res, router := setupMockRequestsAndMux(db)

	req, err := http.NewRequest("POST", "/v1/product", strings.NewReader(badSKUUpdateJSON))
	assert.Nil(t, err)
	router.ServeHTTP(res, req)

	assert.Equal(t, 400, res.Code, "status code should be 400")
	ensureExpectationsWereMet(t, mock)
}

func TestProductCreationHandlerForAlreadyExistentProduct(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()
	res, router := setupMockRequestsAndMux(db)
	setExpectationsForProductExistence(mock, "skateboard", true, nil)

	req, err := http.NewRequest("POST", "/v1/product", strings.NewReader(exampleProductCreationInput))
	assert.Nil(t, err)
	router.ServeHTTP(res, req)

	assert.Equal(t, 400, res.Code, "status code should be 400")
	ensureExpectationsWereMet(t, mock)
}

func TestProductCreationHandlerWhereProgenitorCreationFails(t *testing.T) {
	expectedProgenitor := &ProductProgenitor{
		ID:            2,
		Name:          "Skateboard",
		Description:   "This is a skateboard. Please wear a helmet.",
		Taxable:       true,
		Price:         lolFloats,
		ProductWeight: 8,
		ProductHeight: 7,
		ProductWidth:  6,
		ProductLength: 5,
		PackageWeight: 4,
		PackageHeight: 3,
		PackageWidth:  2,
		PackageLength: 1,
		CreatedAt:     exampleTime,
	}

	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()
	res, router := setupMockRequestsAndMux(db)
	setExpectationsForProductExistence(mock, "skateboard", false, nil)
	mock.ExpectBegin()
	setExpectationsForProductProgenitorCreation(mock, expectedProgenitor, arbitraryError)
	mock.ExpectRollback()

	req, err := http.NewRequest("POST", "/v1/product", strings.NewReader(exampleProductCreationInput))
	assert.Nil(t, err)
	router.ServeHTTP(res, req)

	assert.Equal(t, 500, res.Code, "status code should be 500")
	ensureExpectationsWereMet(t, mock)
}

func TestProductCreationHandlerWithErrorCreatingAttributes(t *testing.T) {
	expectedProgenitor := &ProductProgenitor{
		ID:            2,
		Name:          "Skateboard",
		Description:   "This is a skateboard. Please wear a helmet.",
		Taxable:       true,
		Price:         lolFloats,
		ProductWeight: 8,
		ProductHeight: 7,
		ProductWidth:  6,
		ProductLength: 5,
		PackageWeight: 4,
		PackageHeight: 3,
		PackageWidth:  2,
		PackageLength: 1,
		CreatedAt:     exampleTime,
	}

	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()
	res, router := setupMockRequestsAndMux(db)
	setExpectationsForProductExistence(mock, "skateboard", false, nil)
	mock.ExpectBegin()
	setExpectationsForProductProgenitorCreation(mock, expectedProgenitor, nil)
	setExpectationsForProductAttributeCreation(mock, expectedCreatedProductAttribute, arbitraryError)
	mock.ExpectRollback()

	req, err := http.NewRequest("POST", "/v1/product", strings.NewReader(exampleProductCreationInputWithAttributes))
	assert.Nil(t, err)
	router.ServeHTTP(res, req)

	assert.Equal(t, 500, res.Code, "status code should be 500")
	ensureExpectationsWereMet(t, mock)
}

func TestProductCreationHandlerWhereProductCreationFails(t *testing.T) {
	expectedProgenitor := &ProductProgenitor{
		ID:            2,
		Name:          "Skateboard",
		Description:   "This is a skateboard. Please wear a helmet.",
		Taxable:       true,
		Price:         lolFloats,
		ProductWeight: 8,
		ProductHeight: 7,
		ProductWidth:  6,
		ProductLength: 5,
		PackageWeight: 4,
		PackageHeight: 3,
		PackageWidth:  2,
		PackageLength: 1,
		CreatedAt:     exampleTime,
	}

	expectedProduct := &Product{
		ProductProgenitor: ProductProgenitor{
			ID:            2,
			Name:          "Skateboard",
			Price:         lolFloats,
			Description:   "This is a skateboard. Please wear a helmet.",
			ProductWeight: 8,
			ProductHeight: 7,
			ProductWidth:  6,
			ProductLength: 5,
			PackageWeight: 4,
			PackageHeight: 3,
			PackageWidth:  2,
			PackageLength: 1,
			CreatedAt:     exampleTime,
		},
		ID:                  10,
		ProductProgenitorID: 2,
		SKU:                 "skateboard",
		Name:                "Skateboard",
		UPC:                 NullString{sql.NullString{String: "1234567890", Valid: true}},
		Quantity:            123,
		Price:               lolFloats,
		Cost:                5.0,
		CreatedAt:           exampleTime,
	}

	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()
	res, router := setupMockRequestsAndMux(db)
	setExpectationsForProductExistence(mock, "skateboard", false, nil)
	mock.ExpectBegin()
	setExpectationsForProductProgenitorCreation(mock, expectedProgenitor, nil)
	setExpectationsForProductCreation(mock, expectedProduct, arbitraryError)
	mock.ExpectRollback()

	req, err := http.NewRequest("POST", "/v1/product", strings.NewReader(exampleProductCreationInput))
	assert.Nil(t, err)
	router.ServeHTTP(res, req)

	assert.Equal(t, 500, res.Code, "status code should be 500")
	ensureExpectationsWereMet(t, mock)
}