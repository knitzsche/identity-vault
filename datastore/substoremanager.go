// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2017-2018 Canonical Ltd
 * License granted by Canonical Limited
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 3 as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package datastore

import (
	"database/sql"
	"fmt"
	"github.com/lib/pq"
)

const createSubstoreTableSQL = `
	CREATE TABLE IF NOT EXISTS substore (
		id               serial primary key not null,
		account_id       int references account not null,
		from_model_id    int references model not null,
		store            varchar(200) not null,
		serial_number    varchar(200) not null,
		model_name       varchar(200) not null
	)
`

// Indexes
const createSubstoreUniqueIndexSQL = `
	CREATE UNIQUE INDEX IF NOT EXISTS substore_idx ON substore 
	(account_id, from_model_id, store, serial_number)`

const createSubstoreSQL = `
	INSERT INTO substore 
	(account_id, from_model_id, store, serial_number, model_name) 
	VALUES ($1,$2,$3,$4,$5)`

const getSubstoreSQL = `
	SELECT id, account_id, from_model_id, store, serial_number, model_name 
	FROM substore 
	WHERE from_model_id=$1 AND serial_number=$2`

const getUserSubstoreSQL = `
	SELECT s.id, s.account_id, s.from_model_id, s.store, s.serial_number, s.model_name
	FROM substore s
	INNER JOIN useraccountlink l ON s.account_id = l.account_id
	INNER JOIN userinfo u ON l.user_id = u.id
	WHERE s.from_model_id=$1 AND s.serial_number=$2 AND u.username=$3`

const getSubstoreModelSQL = `
	SELECT s.id, s.account_id, s.from_model_id, s.store, s.serial_number, s.model_name 
	FROM substore s
	INNER JOIN model m ON m.id = s.from_model_id
	WHERE m.brand_id=$1 AND s.model_name=$2 AND s.serial_number=$3`

const listSubstoreSQL = `
	SELECT id, account_id, from_model_id, store, serial_number, model_name 
	FROM substore 
	WHERE account_id=$1`

const listUserSubstoreSQL = `
	SELECT s.id, s.account_id, s.from_model_id, s.store, s.serial_number, s.model_name 
	FROM substore s
	INNER JOIN useraccountlink l ON s.account_id = l.account_id
	INNER JOIN userinfo u ON l.user_id = u.id
	WHERE s.account_id=$1 AND u.username=$2
`
const updateSubstoreSQL = `
	UPDATE substore 
	SET account_id=$2, from_model_id=$3, store=$4, serial_number=$5, model_name=$6 
	WHERE id=$1`
const updateSubstoreForUserSQL = `
	UPDATE substore s 
	SET account_id=$2, from_model_id=$3, store=$4, serial_number=$5, model_name=$6 
	FROM useraccountlink ua
	INNER JOIN userinfo u ON ua.user_id=u.id
	WHERE s.id=$1 AND u.username=$7
	AND ua.account_id=s.account_id`

const deleteSubstoreSQL = "delete from substore where id=$1"
const deleteSubstoreForUserSQL = `
		DELETE FROM substore s
		USING account acc
		INNER JOIN useraccountlink ua ON ua.account_id=acc.id
		INNER JOIN userinfo u ON ua.user_id=u.id
		WHERE s.id=$1 AND acc.id=s.account_id AND u.username=$2`

// Substore holds the substore details for an account in the local database
type Substore struct {
	ID           int    `json:"id"`
	AccountID    int    `json:"accountID"`
	FromModelID  int    `json:"fromModelID"`
	FromModel    Model  `json:"fromModel"`
	Store        string `json:"store"`
	SerialNumber string `json:"serialnumber"`
	ModelName    string `json:"modelname"`
}

// CreateSubstoreTable creates the database table for a sub-store
func (db *DB) CreateSubstoreTable() error {
	_, err := db.Exec(createSubstoreTableSQL)
	if err != nil {
		return err
	}

	_, err = db.Exec(createSubstoreUniqueIndexSQL)
	return err
}

// createSubstore creates a sub-store in the database
func (db *DB) createSubstore(store Substore) (Substore, error) {
	_, err := db.Exec(createSubstoreSQL, store.AccountID, store.FromModelID, store.Store, store.SerialNumber, store.ModelName)
	if err, ok := err.(*pq.Error); ok {
		// This is a PostgreSQL error...
		if err.Code.Name() == "unique_violation" {
			// Output a more readable message
			return store, fmt.Errorf("a sub-store mapping already exists for this from model, serial-number and "+
				"sub-store (%d, %s, %s)", store.FromModelID, store.SerialNumber, store.Store)
		}
	}
	if err != nil {
		return store, fmt.Errorf("error creating the database sub-store (from model, serial-number and sub-store "+
			"(%d, %s, %s): %v", store.FromModelID, store.SerialNumber, store.Store, err)
	}

	// Return the created substore
	substore, err := db.GetSubstore(store.FromModelID, store.SerialNumber)
	if err != nil {
		return store, fmt.Errorf("error creating the database sub-store (from model, serial-number and sub-store "+
			"(%d, %s, %s): %v", store.FromModelID, store.SerialNumber, store.Store, err)
	}

	return substore, nil
}

// GetSubstore fetches a sub-store in the database
func (db *DB) GetSubstore(fromModelID int, serialNumber string) (Substore, error) {
	store := Substore{}

	var row *sql.Row

	row = db.QueryRow(getSubstoreSQL, fromModelID, serialNumber)
	err := row.Scan(&store.ID, &store.AccountID, &store.FromModelID, &store.Store, &store.SerialNumber, &store.ModelName)
	if err != nil {
		return store, fmt.Errorf("error retrieving database substore for %s, from model %d: %v", serialNumber, fromModelID, err)
	}

	store.FromModel, err = db.getModel(store.FromModelID)
	if err != nil {
		return store, fmt.Errorf("error retrieving database model %d: %v", store.FromModelID, err)
	}

	return store, nil
}

// GetSubstoreFilteredByUser fetches a sub-store in the database
func (db *DB) GetSubstoreFilteredByUser(fromModelID int, serialNumber, username string) (Substore, error) {
	store := Substore{}

	var row *sql.Row

	row = db.QueryRow(getUserSubstoreSQL, fromModelID, serialNumber, username)
	err := row.Scan(&store.ID, &store.AccountID, &store.FromModelID, &store.Store, &store.SerialNumber, &store.ModelName)
	if err != nil {
		return store, fmt.Errorf("error retrieving database substore for %s, from model %d: %v", serialNumber, fromModelID, err)
	}

	store.FromModel, err = db.getModel(store.FromModelID)
	if err != nil {
		return store, fmt.Errorf("error retrieving database model %d: %v", store.FromModelID, err)
	}

	return store, nil
}

// GetSubstoreModel fetches a sub-store in the database using the pivoted model name
func (db *DB) GetSubstoreModel(brand, model, serialNumber string) (Substore, error) {
	store := Substore{}

	row := db.QueryRow(getSubstoreModelSQL, brand, model, serialNumber)
	err := row.Scan(&store.ID, &store.AccountID, &store.FromModelID, &store.Store, &store.SerialNumber, &store.ModelName)
	if err != nil {
		return store, fmt.Errorf("error retrieving database substore (model name %s, serial %s): %v", model, serialNumber, err)
	}

	store.FromModel, err = db.getModel(store.FromModelID)
	if err != nil {
		return store, fmt.Errorf("error retrieving database model %d: %v", store.FromModelID, err)
	}

	return store, nil
}

// HealthCheck returns an error if there is a problem talking to the underlying Datastore
func (db *DB) HealthCheck() error {
	_, err := db.Exec("select 1;")
	return err
}

// ListSubstores returns a list of sub-stores
func (db *DB) listSubstores(accountID int) ([]Substore, error) {
	rows, err := db.Query(listSubstoreSQL, accountID)
	if err != nil {
		return nil, fmt.Errorf("error retrieving sub-stores: %v", err)
	}
	defer rows.Close()

	return db.rowsToSubstores(rows)
}

// listSubstoresFilteredByUser returns a list of sub-stores
func (db *DB) listSubstoresFilteredByUser(accountID int, username string) ([]Substore, error) {
	rows, err := db.Query(listUserSubstoreSQL, accountID, username)
	if err != nil {
		return nil, fmt.Errorf("error retrieving sub-stores of a user: %v", err)
	}
	defer rows.Close()

	return db.rowsToSubstores(rows)
}

func (db *DB) deleteSubstore(storeID int) (string, error) {
	return db.deleteSubstoreFilteredByUser(storeID, anyUserFilter)
}

func (db *DB) deleteSubstoreFilteredByUser(storeID int, username string) (string, error) {
	var err error

	if len(username) == 0 {
		_, err = db.Exec(deleteSubstoreSQL, storeID)
	} else {
		_, err = db.Exec(deleteSubstoreForUserSQL, storeID, username)
	}
	if err != nil {
		return "", fmt.Errorf("error deleting the database sub-store model %d: %v", storeID, err)
	}

	return "", nil
}

func (db *DB) rowsToSubstores(rows *sql.Rows) ([]Substore, error) {
	stores := []Substore{}

	for rows.Next() {
		store := Substore{}
		err := rows.Scan(&store.ID, &store.AccountID, &store.FromModelID, &store.Store, &store.SerialNumber, &store.ModelName)
		if err != nil {
			return nil, fmt.Errorf("error scanning for substore: %v", err)
		}

		store.FromModel, err = db.getModel(store.FromModelID)
		if err != nil {
			return nil, fmt.Errorf("error retrieving database model %d: %v", store.FromModelID, err)
		}

		stores = append(stores, store)
	}

	return stores, nil
}

func (db *DB) updateSubstore(store Substore) error {
	return db.updateSubstoreFilteredByUser(store, anyUserFilter)
}

func (db *DB) updateSubstoreFilteredByUser(store Substore, username string) error {
	var err error

	if len(username) == 0 {
		_, err = db.Exec(updateSubstoreSQL, store.ID, store.AccountID, store.FromModelID, store.Store, store.SerialNumber, store.ModelName)
	} else {
		_, err = db.Exec(updateSubstoreForUserSQL, store.ID, store.AccountID, store.FromModelID, store.Store, store.SerialNumber, store.ModelName, username)
	}
	if err, ok := err.(*pq.Error); ok {
		// This is a PostgreSQL error...
		if err.Code.Name() == "unique_violation" {
			// Output a more readable message
			return fmt.Errorf("error updating the database sub-store: a sub-store mapping already exists for "+
				"this model, serial-number and sub-store (%d, %s, %s)", store.FromModelID, store.SerialNumber, store.Store)
		}
	}
	if err != nil {
		return fmt.Errorf("error updating the database sub-store with model, serial-number and sub-store (%d, %s, %s): %v",
			store.FromModelID, store.SerialNumber, store.Store, err)
	}

	return nil
}
