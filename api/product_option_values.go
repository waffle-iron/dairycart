package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

const (
	productOptionValueExistenceQuery            = `SELECT EXISTS(SELECT 1 FROM product_option_values WHERE id = $1 AND archived_on IS NULL)`
	productOptionValueExistenceForOptionIDQuery = `SELECT EXISTS(SELECT 1 FROM product_option_values WHERE product_option_id = $1 AND value = $2 AND archived_on IS NULL)`
	productOptionValueRetrievalQuery            = `SELECT * FROM product_option_values WHERE id = $1`
	productOptionValueRetrievalForOptionIDQuery = `SELECT * FROM product_option_values WHERE product_option_id = $1 AND archived_on IS NULL`
	productOptionValueDeletionQuery             = `UPDATE product_option_values SET archived_on = NOW() WHERE id = $1 AND archived_on IS NULL`
)

// ProductOptionValue represents a product's option values. If you have a t-shirt that comes in three colors
// and three sizes, then there are two ProductOptions for that base_product, color and size, and six ProductOptionValues,
// One for each color and one for each size.
type ProductOptionValue struct {
	DBRow
	ProductOptionID uint64 `json:"product_option_id"`
	Value           string `json:"value"`
}

func (pav *ProductOptionValue) generateScanArgs() []interface{} {
	return []interface{}{
		&pav.ID,
		&pav.ProductOptionID,
		&pav.Value,
		&pav.CreatedOn,
		&pav.UpdatedOn,
		&pav.ArchivedOn,
	}
}

// ProductOptionValueCreationInput is a struct to use for creating product option values
type ProductOptionValueCreationInput struct {
	ProductOptionID uint64
	Value           string `json:"value"`
}

// ProductOptionValueUpdateInput is a struct to use for updating product option values
type ProductOptionValueUpdateInput struct {
	Value string `json:"value"`
}

// retrieveProductOptionValue retrieves a ProductOptionValue with a given ID from the database
func retrieveProductOptionValueFromDB(db *sqlx.DB, id uint64) (*ProductOptionValue, error) {
	v := &ProductOptionValue{}
	err := db.QueryRow(productOptionValueRetrievalQuery, id).Scan(v.generateScanArgs()...)
	if err == sql.ErrNoRows {
		return v, errors.Wrap(err, "Error querying for product option values")
	}
	return v, err
}

// retrieveProductOptionValue retrieves a ProductOptionValue with a given product option ID from the database
func retrieveProductOptionValueForOptionFromDB(db *sqlx.DB, optionID uint64) ([]ProductOptionValue, error) {
	var values []ProductOptionValue

	rows, err := db.Query(productOptionValueRetrievalForOptionIDQuery, optionID)
	if err != nil {
		return nil, errors.Wrap(err, "Error encountered querying for products")
	}
	defer rows.Close()
	for rows.Next() {
		value := ProductOptionValue{}
		_ = rows.Scan(value.generateScanArgs()...)
		values = append(values, value)
	}
	return values, nil
}

func updateProductOptionValueInDB(db *sqlx.DB, v *ProductOptionValue) error {
	valueUpdateQuery, queryArgs := buildProductOptionValueUpdateQuery(v)
	err := db.QueryRow(valueUpdateQuery, queryArgs...).Scan(v.generateScanArgs()...)
	return err
}

func buildProductOptionValueUpdateHandler(db *sqlx.DB) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		// ProductOptionValueUpdateHandler is a request handler that can update product option values
		optionValueID := chi.URLParam(req, "option_value_id")
		// eating these errors because Mux should validate these for us.
		optionValueIDInt, _ := strconv.Atoi(optionValueID)

		// can't update an option value that doesn't exist!
		optionValueExists, err := rowExistsInDB(db, productOptionValueExistenceQuery, optionValueID)
		if err != nil || !optionValueExists {
			respondThatRowDoesNotExist(req, res, "product option value", optionValueID)
			return
		}

		updatedValueData := &ProductOptionValue{}
		err = validateRequestInput(req, updatedValueData)
		if err != nil {
			notifyOfInvalidRequestBody(res, err)
			return
		}

		existingOptionValue, err := retrieveProductOptionValueFromDB(db, uint64(optionValueIDInt))
		if err != nil {
			notifyOfInternalIssue(res, err, "retrieve product option value from the database")
			return
		}
		existingOptionValue.Value = updatedValueData.Value

		err = updateProductOptionValueInDB(db, existingOptionValue)
		if err != nil {
			notifyOfInternalIssue(res, err, "update product option value in the database")
			return
		}

		json.NewEncoder(res).Encode(existingOptionValue)
	}
}

// createProductOptionValueInDB creates a ProductOptionValue tied to a ProductOption
func createProductOptionValueInDB(tx *sql.Tx, v *ProductOptionValue) (uint64, error) {
	var newOptionValueID uint64
	query, args := buildProductOptionValueCreationQuery(v)
	err := tx.QueryRow(query, args...).Scan(&newOptionValueID)
	return newOptionValueID, err
}

func optionValueAlreadyExistsForOption(db *sqlx.DB, optionID int64, value string) (bool, error) {
	var exists string

	err := db.QueryRow(productOptionValueExistenceForOptionIDQuery, optionID, value).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}

	return exists == "true", err
}

func buildProductOptionValueCreationHandler(db *sqlx.DB) http.HandlerFunc {
	// productOptionValueCreationHandler is a product creation handler
	return func(res http.ResponseWriter, req *http.Request) {
		optionID := chi.URLParam(req, "option_id")

		// we can eat this error because Mux takes care of validating route params for us
		optionIDInt, _ := strconv.ParseInt(optionID, 10, 64)

		// can't create values for a product option that doesn't exist
		productOptionExistsByID, err := rowExistsInDB(db, productOptionExistenceQuery, optionID)
		if err != nil || !productOptionExistsByID {
			respondThatRowDoesNotExist(req, res, "product option", optionID)
			return
		}

		newProductOptionValue := &ProductOptionValue{}
		err = validateRequestInput(req, newProductOptionValue)
		if err != nil {
			notifyOfInvalidRequestBody(res, err)
			return
		}
		newProductOptionValue.ProductOptionID = uint64(optionIDInt)

		// can't create a product option value that already exists
		productOptionValueExistsByValue, err := optionValueAlreadyExistsForOption(db, optionIDInt, newProductOptionValue.Value)
		if err != nil || productOptionValueExistsByValue {
			notifyOfInvalidRequestBody(res, fmt.Errorf("product option value `%s` already exists for option ID %s", newProductOptionValue.Value, optionID))
			return
		}

		tx, err := db.Begin()
		if err != nil {
			notifyOfInternalIssue(res, err, "starting a transasction")
			return
		}

		newProductOptionValueID, err := createProductOptionValueInDB(tx, newProductOptionValue)
		if err != nil {
			tx.Rollback()
			notifyOfInternalIssue(res, err, "insert product in database")
			return
		}
		newProductOptionValue.ID = newProductOptionValueID

		err = tx.Commit()
		if err != nil {
			notifyOfInternalIssue(res, err, "closing out transaction")
			return
		}

		res.WriteHeader(http.StatusCreated)
		json.NewEncoder(res).Encode(newProductOptionValue)
	}
}

func archiveProductOptionValue(db *sqlx.DB, id uint64) error {
	_, err := db.Exec(productOptionValueDeletionQuery, id)
	return err
}

func buildProductOptionValueDeletionHandler(db *sqlx.DB) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		// ProductOptionValueDeletionHandler is a request handler that can delete product option values
		optionValueID := chi.URLParam(req, "option_value_id")
		// eating these errors because Mux should validate these for us.
		optionValueIDInt, _ := strconv.Atoi(optionValueID)

		// can't delete an option value that doesn't exist!
		optionValueExists, err := rowExistsInDB(db, productOptionValueExistenceQuery, optionValueID)
		if err != nil || !optionValueExists {
			respondThatRowDoesNotExist(req, res, "product option value", optionValueID)
			return
		}

		err = archiveProductOptionValue(db, uint64(optionValueIDInt))
		if err != nil {
			notifyOfInternalIssue(res, err, "closing out transaction")
			return
		}

		res.WriteHeader(http.StatusOK)
	}
}
