// Copyright (c) 2017, Jonathan Chappelow
// See LICENSE for details.

package dcrdataapi

import (
	"github.com/decred/dcrd/dcrjson"
)

// much of the time, dcrdata will be using the types in dcrjson, but others are
// defined here

// Status indicates the state of the server, including the API version and the
// software version.
type Status struct {
	Ready           bool   `json:"ready"`
	DBHeight        uint32 `json:"db_height"`
	Height          uint32 `json:"node_height"`
	NodeConnections int64  `json:"node_connections"`
	APIVersion      int    `json:"api_version"`
	DcrdataVersion  string `json:"dcrdata_version"`
}

// TicketPoolInfo models data about ticket pool
type TicketPoolInfo struct {
	Size   uint32  `json:"size"`
	Value  float64 `json:"value"`
	ValAvg float64 `json:"valavg"`
}

type TicketPoolValsAndSizes struct {
	StartHeight uint32    `json:"start_height"`
	EndHeight   uint32    `json:"end_height"`
	Value       []float64 `json:"value"`
	Size        []float64 `json:"size"`
}

type BlockDataBasic struct {
	Height     uint32  `json:"height"`
	Size       uint32  `json:"size"`
	Hash       string  `json:"hash"`
	Difficulty float64 `json:"diff"`
	StakeDiff  float64 `json:"sdiff"`
	Time       int64   `json:"time"`
	//TicketPoolInfo
	PoolInfo TicketPoolInfo `json:"ticket_pool"`
}

type StakeDiff struct {
	dcrjson.GetStakeDifficultyResult
	Estimates        dcrjson.EstimateStakeDiffResult `json:"estimates"`
	IdxBlockInWindow int                             `json:"window_block_index"`
	PriceWindowNum   int                             `json:"window_number"`
}

type StakeInfoExtended struct {
	Feeinfo          dcrjson.FeeInfoBlock `json:"feeinfo"`
	StakeDiff        float64              `json:"stakediff"`
	PriceWindowNum   int                  `json:"window_number"`
	IdxBlockInWindow int                  `json:"window_block_index"`
	PoolInfo         TicketPoolInfo       `json:"ticket_pool"`
}

type StakeInfoExtendedEstimates struct {
	Feeinfo          dcrjson.FeeInfoBlock `json:"feeinfo"`
	StakeDiff        StakeDiff            `json:"stakediff"`
	PriceWindowNum   int                  `json:"window_number"`
	IdxBlockInWindow int                  `json:"window_block_index"`
	PoolInfo         TicketPoolInfo       `json:"ticket_pool"`
}

type MempoolTicketFeeInfo struct {
	Height uint32 `json:"height"`
	Time   int64  `json:"time"`
	dcrjson.FeeInfoMempool
	LowestMineable float64 `json:"lowest_mineable"`
}

type MempoolTicketFees struct {
	Height   uint32    `json:"height"`
	Time     int64     `json:"time"`
	Length   uint32    `json:"length"`
	Total    uint32    `json:"total"`
	FeeRates []float64 `json:"top_fees"`
}

type TicketDetails struct {
	Hash    string  `json:"hash"`
	Fee     float64 `json:"abs_fee"`
	FeeRate float64 `json:"fee"`
	Size    int32   `json:"size"`
	Height  int64   `json:"height_received"`
}

type MempoolTicketDetails struct {
	Height  uint32         `json:"height"`
	Time    int64          `json:"time"`
	Length  uint32         `json:"length"`
	Total   uint32         `json:"total"`
	Tickets TicketsDetails `json:"tickets"`
}

type TicketsDetails []*TicketDetails
