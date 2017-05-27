package api

import (
	"database/sql/driver"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	sqlmock "gopkg.in/DATA-DOG/go-sqlmock.v1"
)

var exampleProgenitor *ProductProgenitor
var productProgenitorHeaders []string
var exampleProgenitorData []driver.Value

func init() {
	exampleProgenitor = &ProductProgenitor{
		ID:            2,
		Name:          "Skateboard",
		Description:   "This is a skateboard. Please wear a helmet.",
		Price:         99.99,
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
	productProgenitorHeaders = []string{"id", "name", "description", "taxable", "price", "product_weight", "product_height", "product_width", "product_length", "package_weight", "package_height", "package_width", "package_length", "created_at", "updated_at", "archived_at"}
	exampleProgenitorData = []driver.Value{2, "Skateboard", "This is a skateboard. Please wear a helmet.", false, 99.99, 8, 7, 6, 5, 4, 3, 2, 1, exampleTime, nil, nil}

}

func setExpectationsForProductProgenitorExistence(mock sqlmock.Sqlmock, id int64, exists bool) {
	exampleRows := sqlmock.NewRows([]string{""}).AddRow(strconv.FormatBool(exists))
	query := formatConstantQueryForSQLMock(buildProgenitorRetrievalQuery(1))
	mock.ExpectQuery(query).WithArgs(id).WillReturnRows(exampleRows)
}

func setExpectationsForProductProgenitorCreation(mock sqlmock.Sqlmock) {
	q, _ := buildProgenitorCreationQuery(exampleProgenitor)
	query := formatConstantQueryForSQLMock(q)
	mock.ExpectQuery(query).
		WithArgs(
			exampleProgenitor.Name,
			exampleProgenitor.Description,
			exampleProgenitor.Taxable,
			exampleProgenitor.Price,
			exampleProgenitor.ProductWeight,
			exampleProgenitor.ProductHeight,
			exampleProgenitor.ProductWidth,
			exampleProgenitor.ProductLength,
			exampleProgenitor.PackageWeight,
			exampleProgenitor.PackageHeight,
			exampleProgenitor.PackageWidth,
			exampleProgenitor.PackageLength,
		).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(exampleProgenitor.ID))
}

func TestCreateProductProgenitorInDB(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()
	setExpectationsForProductProgenitorCreation(mock)

	newProgenitor, err := createProductProgenitorInDB(db, exampleProgenitor)
	assert.Nil(t, err)
	assert.Equal(t, int64(2), newProgenitor.ID, "createProductProgenitorInDB should return the correct ID for a new progenitor")
	ensureExpectationsWereMet(t, mock)
}

func setupExpectationsForProductProgenitorRetrieval(mock sqlmock.Sqlmock) {
	exampleRows := sqlmock.NewRows(productProgenitorHeaders).
		AddRow(exampleProgenitorData...)

	productProgenitorQuery := buildProgenitorRetrievalQuery(exampleProgenitor.ID)
	mock.ExpectQuery(formatConstantQueryForSQLMock(productProgenitorQuery)).
		WithArgs(exampleProgenitor.ID).
		WillReturnRows(exampleRows)
}

func TestRetrieveProductProgenitorFromDB(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()
	setupExpectationsForProductProgenitorRetrieval(mock)

	actual, err := retrieveProductProgenitorFromDB(db, exampleProgenitor.ID)
	assert.Nil(t, err)
	assert.Equal(t, exampleProgenitor, actual, "product progenitor retrieved by query should match")
}