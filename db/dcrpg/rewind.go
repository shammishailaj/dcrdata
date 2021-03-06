package dcrpg

// Deletion of all data for a certain block (identified by hash), is performed
// using the following relationships:
//
// blocks -> hash identifies: votes, misses, tickets, transactions
//        -> txdbids & stxdbids identifies: transactions
//        -> use previous_hash to continue to parent block
//
// transactions -> vin_db_ids identifies: vins, addresses where is_funding=false
//              -> vout_db_ids identifies vouts, addresses where is_funding=true
//
// tickets -> purchase_tx_db_id identifies the corresponding txn (but rows may
//            be removed directly by block hash)
//
// addresses -> tx_vin_vout_row_id where is_funding=true corresponds to transactions.vout_db_ids
//           -> tx_vin_vout_row_id where is_funding=false corresponds to transactions.vin_db_ids
//
// For example, REMOVAL of a block's data could be performed in the following
// manner, where [] indicates primary key/row ID lookup:
//	1. vin_DB_IDs = transactions[blocks.txdbids].vin_db_ids
//	2. Remove vins[vin_DB_IDs]
//	3. vout_DB_IDs = transactions[blocks.txdbids].vout_db_ids
//	4. Remove vouts[vout_DB_IDs]
//	5. Remove addresses WHERE tx_vin_vout_row_id=vout_DB_IDs AND is_funding=true
//	6. Remove addresses WHERE tx_vin_vout_row_id=vin_DB_IDs AND is_funding=false
//	7. Repeat 1-6 for blocks.stxdbids (instead of blocks.txdbids)
//	8. Remove tickets where purchase_tx_db_id = blocks.stxdbids
//	   OR Remove tickets by block_hash
//	9. Remove votes by block_hash
//	10. Remove misses by block_hash
//	11. Remove transactions[txdbids] and transactions[stxdbids]
//
// Use DeleteBlockData to delete all data across these tables for a certain block.

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/decred/dcrdata/v4/db/dbtypes"
	"github.com/decred/dcrdata/v4/db/dcrpg/internal"
)

func deleteMissesForBlock(dbTx *sql.Tx, hash string) (rowsDeleted int64, err error) {
	return sqlExec(dbTx, internal.DeleteMisses, "failed to delete misses", hash)
}

func deleteVotesForBlock(dbTx *sql.Tx, hash string) (rowsDeleted int64, err error) {
	return sqlExec(dbTx, internal.DeleteVotes, "failed to delete votes", hash)
}

func deleteTicketsForBlock(dbTx *sql.Tx, hash string) (rowsDeleted int64, err error) {
	return sqlExec(dbTx, internal.DeleteTickets, "failed to delete tickets", hash)
}

func deleteTransactionsForBlock(dbTx *sql.Tx, hash string) (rowsDeleted int64, err error) {
	return sqlExec(dbTx, internal.DeleteTransactions, "failed to delete transactions", hash)
}

func deleteVoutsForBlock(dbTx *sql.Tx, hash string) (rowsDeleted int64, err error) {
	return sqlExec(dbTx, internal.DeleteVouts, "failed to delete vouts", hash)
}

func deleteVinsForBlock(dbTx *sql.Tx, hash string) (rowsDeleted int64, err error) {
	return sqlExec(dbTx, internal.DeleteVins, "failed to delete vins", hash)
}

func deleteAddressesForBlock(dbTx *sql.Tx, hash string) (rowsDeleted int64, err error) {
	return sqlExec(dbTx, internal.DeleteAddresses, "failed to delete addresses", hash)
}

func deleteBlock(dbTx *sql.Tx, hash string) (rowsDeleted int64, err error) {
	return sqlExec(dbTx, internal.DeleteBlock, "failed to delete block", hash)
}

func deleteBlockFromChain(dbTx *sql.Tx, hash string) (err error) {
	// Delete the row from block_chain where this_hash is the specified hash,
	// returning the previous block hash in the chain.
	var prev_hash string
	err = dbTx.QueryRow(internal.DeleteBlockFromChain, hash).Scan(&prev_hash)
	if err != nil {
		// If a row with this_hash was not found, and thus prev_hash is not set,
		// attempt to locate a row with next_hash set to the hash of this block,
		// and set it to the empty string.
		if err == sql.ErrNoRows {
			err = UpdateBlockNextByNextHash(dbTx, hash, "")
		}
		return
	}

	// For any row where next_hash is the prev_hash of the removed row, set
	// next_hash to and empty string since that block is no longer in the chain.
	return UpdateBlockNextByHash(dbTx, prev_hash, "")
}

// DeleteBlockData removes all data for the specified block from every table.
// Data are removed from tables in the following order: vins, vouts, addresses,
// transactions, tickets, votes, misses, blocks, block_chain.
// WARNING: When no indexes are present, these queries are VERY SLOW.
func DeleteBlockData(ctx context.Context, db *sql.DB, hash string) (res dbtypes.DeletionSummary, err error) {
	// The data purge is an all or nothing operation (no partial removal of
	// data), so use a common sql.Tx for all deletions, and Commit in this
	// function rather after each deletion.
	var dbTx *sql.Tx
	dbTx, err = db.BeginTx(ctx, nil)
	if err != nil {
		err = fmt.Errorf("failed to start new DB transaction: %v", err)
		return
	}

	res.Timings = new(dbtypes.DeletionSummary)

	start := time.Now()
	if res.Vins, err = deleteVinsForBlock(dbTx, hash); err != nil {
		err = fmt.Errorf(`deleteVinsForBlock failed with "%v". Rollback: %v`,
			err, dbTx.Rollback())
		return
	}
	res.Timings.Vins = time.Since(start).Nanoseconds()

	start = time.Now()
	if res.Vouts, err = deleteVoutsForBlock(dbTx, hash); err != nil {
		err = fmt.Errorf(`deleteVoutsForBlock failed with "%v". Rollback: %v`,
			err, dbTx.Rollback())
		return
	}
	res.Timings.Vouts = time.Since(start).Nanoseconds()

	start = time.Now()
	if res.Addresses, err = deleteAddressesForBlock(dbTx, hash); err != nil {
		err = fmt.Errorf(`deleteAddressesForBlock failed with "%v". Rollback: %v`,
			err, dbTx.Rollback())
		return
	}
	res.Timings.Addresses = time.Since(start).Nanoseconds()

	// Deleting transactions rows follow deletion of vins, vouts, and addresses
	// rows since the transactions table is used to identify the vin and vout DB
	// row IDs for a transaction.
	start = time.Now()
	if res.Transactions, err = deleteTransactionsForBlock(dbTx, hash); err != nil {
		err = fmt.Errorf(`deleteTransactionsForBlock failed with "%v". Rollback: %v`,
			err, dbTx.Rollback())
		return
	}
	res.Timings.Transactions = time.Since(start).Nanoseconds()

	start = time.Now()
	if res.Tickets, err = deleteTicketsForBlock(dbTx, hash); err != nil {
		err = fmt.Errorf(`deleteTicketsForBlock failed with "%v". Rollback: %v`,
			err, dbTx.Rollback())
		return
	}
	res.Timings.Tickets = time.Since(start).Nanoseconds()

	start = time.Now()
	if res.Votes, err = deleteVotesForBlock(dbTx, hash); err != nil {
		err = fmt.Errorf(`deleteVotesForBlock failed with "%v". Rollback: %v`,
			err, dbTx.Rollback())
		return
	}
	res.Timings.Votes = time.Since(start).Nanoseconds()

	start = time.Now()
	if res.Misses, err = deleteMissesForBlock(dbTx, hash); err != nil {
		err = fmt.Errorf(`deleteMissesForBlock failed with "%v". Rollback: %v`,
			err, dbTx.Rollback())
		return
	}
	res.Timings.Misses = time.Since(start).Nanoseconds()

	start = time.Now()
	if res.Blocks, err = deleteBlock(dbTx, hash); err != nil {
		err = fmt.Errorf(`deleteBlock failed with "%v". Rollback: %v`,
			err, dbTx.Rollback())
		return
	}
	res.Timings.Blocks = time.Since(start).Nanoseconds()
	if res.Blocks != 1 {
		log.Errorf("Expected to delete 1 row of blocks table; actually removed %d.",
			res.Blocks)
	}

	err = deleteBlockFromChain(dbTx, hash)
	switch err {
	case sql.ErrNoRows:
		// Just warn but do not return the error.
		err = nil
		log.Warnf("Block with hash %s not found in block_chain table.", hash)
	case nil:
		// Great. Go on to Commit.
	default: // err != nil && err != sql.ErrNoRows
		// Do not return an error if deleteBlockFromChain just did not delete
		// exactly 1 row. Commit and be done.
		if strings.HasPrefix(err.Error(), NotOneRowErrMsg) {
			log.Warnf("deleteBlockFromChain: %v", err)
			err = dbTx.Commit()
		} else {
			err = fmt.Errorf(`deleteBlockFromChain failed with "%v". Rollback: %v`,
				err, dbTx.Rollback())
		}
		return
	}

	err = dbTx.Commit()

	return
}

// DeleteBestBlock removes all data for the best block in the DB from every
// table via DeleteBlockData. The returned height and hash are for the best
// block after successful data removal, or the initial best block if removal
// fails as indicated by a non-nil error value.
func DeleteBestBlock(ctx context.Context, db *sql.DB) (res dbtypes.DeletionSummary, height uint64, hash string, err error) {
	height, hash, _, err = RetrieveBestBlockHeight(ctx, db)
	if err != nil {
		return
	}

	res, err = DeleteBlockData(ctx, db, hash)
	if err != nil {
		return
	}

	height, hash, _, err = RetrieveBestBlockHeight(ctx, db)
	return
}

// DeleteBlocks removes all data for the N best blocks in the DB from every
// table via repeated calls to DeleteBestBlock.
func DeleteBlocks(ctx context.Context, N int64, db *sql.DB) (res []dbtypes.DeletionSummary, height uint64, hash string, err error) {
	// If N is less than 1, get the current best block height and hash, then
	// return.
	if N < 1 {
		height, hash, _, err = RetrieveBestBlockHeight(ctx, db)
		if err == sql.ErrNoRows {
			err = nil
		}
		return
	}

	for i := int64(0); i < N; i++ {
		var resi dbtypes.DeletionSummary
		resi, height, hash, err = DeleteBestBlock(ctx, db)
		// Continue if err == sql.ErrNoRows or nil.
		if err != nil && err != sql.ErrNoRows {
			return
		}
		res = append(res, resi)
		if hash == "" {
			err = nil // do not return sql.ErrNoRows
			break
		}
		if (i%100 == 0 && i > 0) || i == N-1 {
			log.Debugf("Removed data for %d blocks.", i+1)
		}
	}

	return
}
