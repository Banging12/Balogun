// Copyright 2017 The Celo Authors
// This file is part of the celo library.
//
// The celo library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The celo library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the celo library. If not, see <http://www.gnu.org/licenses/>.

package enodes

import (
	"fmt"
	"strings"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"

	"github.com/celo-org/celo-blockchain/common"
	"github.com/celo-org/celo-blockchain/consensus/istanbul"
	"github.com/celo-org/celo-blockchain/consensus/istanbul/backend/internal/db"
	"github.com/celo-org/celo-blockchain/crypto"
	"github.com/celo-org/celo-blockchain/log"
	"github.com/celo-org/celo-blockchain/rlp"
)

const (
	versionCertificateDBVersion = 0
)

// VersionCertificateDB stores
type VersionCertificateDB struct {
	gdb    *db.GenericDB
	logger log.Logger
}

func versionCertificateEntryFromGenericEntry(entry db.GenericEntry) (*istanbul.VersionCertificate, error) {
	signedAnnVersionEntry, ok := entry.(*istanbul.VersionCertificate)
	if !ok {
		return nil, errIncorrectEntryType
	}
	return signedAnnVersionEntry, nil
}

// OpenVersionCertificateDB opens a signed announce version database for storing
// VersionCertificates. If no path is given an in-memory, temporary database is constructed.
func OpenVersionCertificateDB(path string) (*VersionCertificateDB, error) {
	logger := log.New("db", "VersionCertificateDB")

	gdb, err := db.New(int64(versionCertificateDBVersion), path, logger, &opt.WriteOptions{NoWriteMerge: true})
	if err != nil {
		logger.Error("Error creating db", "err", err)
		return nil, err
	}

	return &VersionCertificateDB{
		gdb:    gdb,
		logger: logger,
	}, nil
}

// Close flushes and closes the database files.
func (svdb *VersionCertificateDB) Close() error {
	return svdb.gdb.Close()
}

// String gives a string representation of the entire db
func (svdb *VersionCertificateDB) String() string {
	var b strings.Builder
	b.WriteString("VersionCertificateDB:")

	err := svdb.iterate(func(address common.Address, entry *istanbul.VersionCertificate) error {
		fmt.Fprintf(&b, " [%s => %s]", address.String(), entry.String())
		return nil
	})

	if err != nil {
		svdb.logger.Error("ValidatorEnodeDB.String error", "err", err)
	}

	return b.String()
}

// Upsert inserts any new entries or entries with a Version higher than the
// existing version. Returns any new or updated entries
func (svdb *VersionCertificateDB) Upsert(savEntries []*istanbul.VersionCertificate) ([]*istanbul.VersionCertificate, error) {
	logger := svdb.logger.New("func", "Upsert")

	var newEntries []*istanbul.VersionCertificate

	getExistingEntry := func(entry db.GenericEntry) (db.GenericEntry, error) {
		savEntry, err := versionCertificateEntryFromGenericEntry(entry)
		if err != nil {
			return entry, err
		}
		return svdb.Get(savEntry.Address())
	}

	onNewEntry := func(batch *leveldb.Batch, entry db.GenericEntry) error {
		savEntry, err := versionCertificateEntryFromGenericEntry(entry)
		if err != nil {
			return err
		}
		savEntryBytes, err := encodeVersionCertificate(savEntry)
		if err != nil {
			return err
		}
		batch.Put(addressKey(savEntry.Address()), savEntryBytes)
		newEntries = append(newEntries, savEntry)
		logger.Trace("Updating with new entry",
			"address", savEntry.Address, "new version", savEntry.Version)
		return nil
	}

	onUpdatedEntry := func(batch *leveldb.Batch, existingEntry db.GenericEntry, newEntry db.GenericEntry) error {
		existingSav, err := versionCertificateEntryFromGenericEntry(existingEntry)
		if err != nil {
			return err
		}
		newSav, err := versionCertificateEntryFromGenericEntry(newEntry)
		if err != nil {
			return err
		}
		if newSav.Version <= existingSav.Version {
			logger.Trace("Skipping new entry whose version is not greater than the existing entry", "existing version", existingSav.Version, "new version", newSav.Version)
			return nil
		}
		return onNewEntry(batch, newEntry)
	}

	entries := make([]db.GenericEntry, len(savEntries))
	for i, sav := range savEntries {
		entries[i] = db.GenericEntry(sav)
	}

	if err := svdb.gdb.Upsert(entries, getExistingEntry, onUpdatedEntry, onNewEntry); err != nil {
		logger.Warn("Error upserting entries", "err", err)
		return nil, err
	}
	return newEntries, nil
}

// Get gets the istanbul.VersionCertificateEntry entry with address `address`.
// Returns an error if no entry exists.
func (svdb *VersionCertificateDB) Get(address common.Address) (*istanbul.VersionCertificate, error) {
	entryBytes, err := svdb.gdb.Get(addressKey(address))
	if err != nil {
		return nil, err
	}
	return decodeVersionCertificate(entryBytes)
}

// GetVersion gets the version for the entry with address `address`
// Returns an error if no entry exists
func (svdb *VersionCertificateDB) GetVersion(address common.Address) (uint, error) {
	signedAnnVersion, err := svdb.Get(address)
	if err != nil {
		return 0, err
	}
	return signedAnnVersion.Version, nil
}

// GetAll gets each istanbul.VersionCertificateEntry in the db
func (svdb *VersionCertificateDB) GetAll() ([]*istanbul.VersionCertificate, error) {
	var entries []*istanbul.VersionCertificate
	err := svdb.iterate(func(address common.Address, entry *istanbul.VersionCertificate) error {
		entries = append(entries, entry)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return entries, nil
}

// Remove will remove an entry from the table
func (svdb *VersionCertificateDB) Remove(address common.Address) error {
	batch := new(leveldb.Batch)
	batch.Delete(addressKey(address))
	return svdb.gdb.Write(batch)
}

// Prune will remove entries for all addresses not present in addressesToKeep
func (svdb *VersionCertificateDB) Prune(addressesToKeep map[common.Address]bool) error {
	batch := new(leveldb.Batch)
	err := svdb.iterate(func(address common.Address, entry *istanbul.VersionCertificate) error {
		if !addressesToKeep[address] {
			svdb.logger.Trace("Deleting entry", "address", address)
			batch.Delete(addressKey(address))
		}
		return nil
	})
	if err != nil {
		return err
	}
	return svdb.gdb.Write(batch)
}

// Version certificates are serialised differently to network serialisation for
// storage in the version certificate db. Instead of storing just the version
// and signature, all fields are stored. It's not clear why this approach was
// chosen since it is not necessary to store the public key and address because
// they can be derived from the version and signature. Nevertheless we continue
// to use this approach because we can't easily change it without changing the
// storage format and breaking backwards compatibility.
func decodeVersionCertificate(value []byte) (*istanbul.VersionCertificate, error) {
	var content struct {
		Address   common.Address
		PublicKey []byte
		Version   uint
		Signature []byte
	}
	if err := rlp.DecodeBytes(value, &content); err != nil {
		return nil, err
	}
	decodedPublicKey, err := crypto.UnmarshalPubkey(content.PublicKey)
	if err != nil {
		return nil, err
	}
	return istanbul.NewVersionCertificateFromFields(content.Version, content.Signature, content.Address, decodedPublicKey), nil
}

// Version certificates are serialised differently to network serialisation for
// storage in the version certificate db. Instead of storing just the version
// and signature, all fields are stored. It's not clear why this approach was
// chosen since it is not necessary to store the public key and address because
// they can be derived from the version and signature. Nevertheless we continue
// to use this approach because we can't easily change it without changing the
// storage format and breaking backwards compatibility.
func encodeVersionCertificate(vc *istanbul.VersionCertificate) ([]byte, error) {
	encodedPublicKey := crypto.FromECDSAPub(vc.PublicKey())
	return rlp.EncodeToBytes([]interface{}{vc.Address(), encodedPublicKey, vc.Version, vc.Signature})
}

// iterate will call `onEntry` for each entry in the db
func (svdb *VersionCertificateDB) iterate(onEntry func(common.Address, *istanbul.VersionCertificate) error) error {
	logger := svdb.logger.New("func", "iterate")
	// Only target address keys
	keyPrefix := []byte(dbAddressPrefix)

	onDBEntry := func(key []byte, value []byte) error {
		entry, err := decodeVersionCertificate(value)
		if err != nil {
			return err
		}

		address := common.BytesToAddress(key)
		if err := onEntry(address, entry); err != nil {
			return err
		}
		return nil
	}

	if err := svdb.gdb.Iterate(keyPrefix, onDBEntry); err != nil {
		logger.Warn("Error iterating through db entries", "err", err)
		return err
	}
	return nil
}

// VersionCertificateEntryInfo gives basic information for an entry in the DB
type VersionCertificateEntryInfo struct {
	Address string `json:"address"`
	Version uint   `json:"version"`
}

// Info gives a map VersionCertificateEntryInfo where each key is the address.
// Intended for RPC use
func (svdb *VersionCertificateDB) Info() (map[string]*VersionCertificateEntryInfo, error) {
	dbInfo := make(map[string]*VersionCertificateEntryInfo)
	err := svdb.iterate(func(address common.Address, entry *istanbul.VersionCertificate) error {
		dbInfo[address.Hex()] = &VersionCertificateEntryInfo{
			Address: entry.Address().Hex(),
			Version: entry.Version,
		}
		return nil
	})
	return dbInfo, err
}
