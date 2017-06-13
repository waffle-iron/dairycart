package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

const (
	discountRetrievalQuery = `SELECT * FROM discounts WHERE id = $1`
	discountDeletionQuery  = `UPDATE discounts SET archived_on = NOW() WHERE id = $1 AND archived_on IS NULL`
)

// Discount represents pricing changes that apply temporarily to products
type Discount struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Amount    float32   `json:"amount"`
	StartsOn  time.Time `json:"starts_on"`
	ExpiresOn NullTime  `json:"expires_on"`

	RequiresCode bool   `json:"requires_code"`
	Code         string `json:"code,omitempty"`

	LimitedUse   bool  `json:"limited_use"`
	NumberOfUses int64 `json:"number_of_uses,omitempty"`

	LoginRequired bool `json:"login_required"`

	// Housekeeping
	CreatedOn  time.Time `json:"created_on"`
	UpdatedOn  NullTime  `json:"updated_on,omitempty"`
	ArchivedOn NullTime  `json:"archived_on,omitempty"`
}

// generateScanArgs generates an array of pointers to struct fields for sql.Scan to populate
func (d *Discount) generateScanArgs() []interface{} {
	return []interface{}{
		&d.ID,
		&d.Name,
		&d.Type,
		&d.Amount,
		&d.StartsOn,
		&d.ExpiresOn,
		&d.RequiresCode,
		&d.Code,
		&d.LimitedUse,
		&d.NumberOfUses,
		&d.LoginRequired,
		&d.CreatedOn,
		&d.UpdatedOn,
		&d.ArchivedOn,
	}
}

// generateJoinScanArgsWithCount does everything generateJoinScanArgs does,
// only with an added count parameter
func (d *Discount) generateJoinScanArgsWithCount(count *uint64) []interface{} {
	scanArgs := []interface{}{count}
	discountScanArgs := d.generateScanArgs()
	return append(scanArgs, discountScanArgs...)
}

// DiscountsResponse is a discount response struct
type DiscountsResponse struct {
	ListResponse
	Data []Discount `json:"data"`
}

func (d *Discount) discountTypeIsValid() bool {
	// Because Go doesn't have typed enums (https://github.com/golang/go/issues/19814),
	// this is my only real line of defense against a user attempting to load an invalid
	// discount type into the database. It's lame, type enums aren't, here's hoping.
	return d.Type == "percentage" || d.Type == "flat_amount"
}

// retrieveDiscountFromDB retrieves a discount with a given ID from the database
func retrieveDiscountFromDB(db *sql.DB, discountID string) (*Discount, error) {
	discount := &Discount{}
	scanArgs := discount.generateScanArgs()
	err := db.QueryRow(discountRetrievalQuery, discountID).Scan(scanArgs...)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "Error querying for discount")
	}

	return discount, err
}

func buildDiscountRetrievalHandler(db *sql.DB) http.HandlerFunc {
	// DiscountRetrievalHandler is a request handler that returns a single Discount
	return func(res http.ResponseWriter, req *http.Request) {
		discountID := mux.Vars(req)["discount_id"]

		discount, err := retrieveDiscountFromDB(db, discountID)
		if err != nil {
			notifyOfInternalIssue(res, err, "retrieving discount from database")
			return
		}
		if discount == nil {
			respondThatRowDoesNotExist(req, res, "discount", discountID)
			return
		}

		json.NewEncoder(res).Encode(discount)
	}
}

// retrieveListOfDiscountsFromDB retrieves a list of discounts from the database
func retrieveListOfDiscountsFromDB(db *sql.DB, queryFilter *QueryFilter) ([]Discount, uint64, error) {
	var discounts []Discount
	var count uint64

	query, args := buildDiscountListQuery(queryFilter)
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, errors.Wrap(err, "Error encountered querying for discounts")
	}

	defer rows.Close()
	for rows.Next() {
		var discount Discount
		var queryCount uint64

		scanArgs := discount.generateJoinScanArgsWithCount(&queryCount)
		_ = rows.Scan(scanArgs...)

		count = queryCount
		discounts = append(discounts, discount)
	}
	return discounts, count, nil
}

func buildDiscountListRetrievalHandler(db *sql.DB) http.HandlerFunc {
	// DiscountListRetrievalHandler is a request handler that returns a list of Discounts
	return func(res http.ResponseWriter, req *http.Request) {
		rawFilterParams := req.URL.Query()
		queryFilter := parseRawFilterParams(rawFilterParams)
		discounts, count, err := retrieveListOfDiscountsFromDB(db, queryFilter)
		if err != nil {
			notifyOfInternalIssue(res, err, "retrieve discounts from the database")
			return
		}

		discountsResponse := &DiscountsResponse{
			ListResponse: ListResponse{
				Page:  queryFilter.Page,
				Limit: queryFilter.Limit,
				Count: count,
			},
			Data: discounts,
		}
		json.NewEncoder(res).Encode(discountsResponse)
	}
}