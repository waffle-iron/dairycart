package api

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/lib/pq"
	"github.com/pkg/errors"
)

// Discount represents pricing changes that apply temporarily to products
type Discount struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	ProductID int64     `json:"product_id"`
	Amount    float32   `json:"amount"`
	StartsOn  time.Time `json:"starts_on"`
	ExpiresOn time.Time `json:"expires_on"`

	// Housekeeping
	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  pq.NullTime `json:"-"`
	ArchivedAt pq.NullTime `json:"-"`
}

// generateScanArgs generates an array of pointers to struct fields for sql.Scan to populate
func (d *Discount) generateScanArgs() []interface{} {
	return []interface{}{
		&d.ID,
		&d.Name,
		&d.Type,
		&d.Amount,
		&d.ProductID,
		&d.StartsOn,
		&d.ExpiresOn,
		&d.CreatedAt,
		&d.UpdatedAt,
		&d.ArchivedAt,
	}
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
	discountRetrievalQuery := buildDiscountRetrievalQuery(discountID)
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
		if discount == nil {
			respondThatRowDoesNotExist(req, res, "discount", discountID)
			return
		}
		if err != nil {
			log.Printf(`

			received the following error trying to retrieve discount #%s:
				%s

			`, discountID, err.Error())
			notifyOfInternalIssue(res, err, "retrieving discount from database")
			return
		}

		json.NewEncoder(res).Encode(discount)
	}
}
