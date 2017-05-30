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

func TestNewProductProgenitorFromProductCreationInput(t *testing.T) {
	expected := &ProductProgenitor{
		Name:          "Example",
		Description:   "this is a description",
		Taxable:       true,
		Price:         10,
		ProductWeight: 10,
		ProductHeight: 10,
		ProductWidth:  10,
		ProductLength: 10,
		PackageWeight: 10,
		PackageHeight: 10,
		PackageWidth:  10,
		PackageLength: 10,
	}
	input := &ProductCreationInput{
		Name:          "Example",
		Description:   "this is a description",
		Taxable:       true,
		Price:         10,
		ProductWeight: 10,
		ProductHeight: 10,
		ProductWidth:  10,
		ProductLength: 10,
		PackageWeight: 10,
		PackageHeight: 10,
		PackageWidth:  10,
		PackageLength: 10,
	}
	actual := newProductProgenitorFromProductCreationInput(input)
	assert.Equal(t, expected, actual, "Output of newProductProgenitorFromProductCreationInput was unexpected")
}

func setExpectationsForProductProgenitorExistence(mock sqlmock.Sqlmock, id string, exists bool) {
	exampleRows := sqlmock.NewRows([]string{""}).AddRow(strconv.FormatBool(exists))
	query := formatQueryForSQLMock(buildProgenitorExistenceQuery(id))
	mock.ExpectQuery(query).WithArgs(id).WillReturnRows(exampleRows)
}

func setExpectationsForProductProgenitorCreation(mock sqlmock.Sqlmock, g *ProductProgenitor, err error) {
	q, _ := buildProgenitorCreationQuery(exampleProgenitor)
	query := formatQueryForSQLMock(q)
	mock.ExpectQuery(query).
		WithArgs(
			g.Name,
			g.Description,
			g.Taxable,
			g.Price,
			g.ProductWeight,
			g.ProductHeight,
			g.ProductWidth,
			g.ProductLength,
			g.PackageWeight,
			g.PackageHeight,
			g.PackageWidth,
			g.PackageLength,
		).WillReturnRows(sqlmock.NewRows([]string{"id"}).
		AddRow(exampleProgenitor.ID)).
		WillReturnError(err)
}

func TestCreateProductProgenitorInDB(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()
	mock.ExpectBegin()
	setExpectationsForProductProgenitorCreation(mock, exampleProgenitor, nil)
	mock.ExpectCommit()

	tx, err := db.Begin()
	assert.Nil(t, err)

	newProgenitorID, err := createProductProgenitorInDB(tx, exampleProgenitor)
	assert.Nil(t, err)
	assert.Equal(t, int64(2), newProgenitorID, "createProductProgenitorInDB should return the correct ID for a new progenitor")

	err = tx.Commit()
	assert.Nil(t, err)
	ensureExpectationsWereMet(t, mock)
}

func setupExpectationsForProductProgenitorRetrieval(mock sqlmock.Sqlmock) {
	exampleRows := sqlmock.NewRows(productProgenitorHeaders).
		AddRow(exampleProgenitorData...)

	productProgenitorQuery := buildProgenitorRetrievalQuery(exampleProgenitor.ID)
	mock.ExpectQuery(formatQueryForSQLMock(productProgenitorQuery)).
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