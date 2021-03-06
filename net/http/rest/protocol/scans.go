// Copyright 2014 Rafael Dantas Justo. All rights reserved.
// Use of this source code is governed by a GPL
// license that can be found in the LICENSE file.

// Package protocol describes the REST protocol
package protocol

import (
	"fmt"
	"github.com/rafaeljusto/shelter/dao"
	"github.com/rafaeljusto/shelter/model"
)

// ScansResponse store multiple scans objects with pagination support
type ScansResponse struct {
	Page          int            `json:"page"`            // Current page selected
	PageSize      int            `json:"pageSize"`        // Number of scans in a page
	NumberOfPages int            `json:"numberOfPages"`   // Total number of pages for the result set
	NumberOfItems int            `json:"numberOfItems"`   // Total number of scans in the result set
	Scans         []ScanResponse `json:"scans,omitempty"` // List of scan objects for the current page
	Links         []Link         `json:"links,omitempty"` // Links for pagination managment
}

// Convert a list of scan objects into protocol format with pagination support
func ScansToScansResponse(scans []model.Scan, pagination dao.ScanDAOPagination) ScansResponse {
	var scansResponses []ScanResponse
	for _, scan := range scans {
		scansResponses = append(scansResponses, ScanToScanResponse(scan))
	}

	var orderBy string
	for _, sort := range pagination.OrderBy {
		if len(orderBy) > 0 {
			orderBy += "@"
		}

		orderBy += fmt.Sprintf("%s:%s",
			dao.ScanDAOOrderByFieldToString(sort.Field),
			dao.DAOOrderByDirectionToString(sort.Direction),
		)
	}

	// Add pagination managment links to the response. The URI is hard coded, I didn't have
	// any idea on how can we do this dynamically yet. We cannot get the URI from the
	// handler because we are going to have a cross-reference problem
	links := []Link{
		{
			Types: []LinkType{LinkTypeCurrent},
			HRef:  "/scans/?current",
		},
	}

	// Only add fast backward if we aren't in the first page
	if pagination.Page > 1 {
		links = append(links, Link{
			Types: []LinkType{LinkTypeFirst},
			HRef: fmt.Sprintf("/scans/?pagesize=%d&page=%d&orderby=%s",
				pagination.PageSize, 1, orderBy,
			),
		})
	}

	// Only add previous if theres a previous page
	if pagination.Page-1 >= 1 {
		links = append(links, Link{
			Types: []LinkType{LinkTypePrev},
			HRef: fmt.Sprintf("/scans/?pagesize=%d&page=%d&orderby=%s",
				pagination.PageSize, pagination.Page-1, orderBy,
			),
		})
	}

	// Only add next if there's a next page
	if pagination.Page+1 <= pagination.NumberOfPages {
		links = append(links, Link{
			Types: []LinkType{LinkTypeNext},
			HRef: fmt.Sprintf("/scans/?pagesize=%d&page=%d&orderby=%s",
				pagination.PageSize, pagination.Page+1, orderBy,
			),
		})
	}

	// Only add the fast forward if we aren't on the last page
	if pagination.Page < pagination.NumberOfPages {
		links = append(links, Link{
			Types: []LinkType{LinkTypeLast},
			HRef: fmt.Sprintf("/scans/?pagesize=%d&page=%d&orderby=%s",
				pagination.PageSize, pagination.NumberOfPages, orderBy,
			),
		})
	}

	return ScansResponse{
		Page:          pagination.Page,
		PageSize:      pagination.PageSize,
		NumberOfPages: pagination.NumberOfPages,
		NumberOfItems: pagination.NumberOfItems,
		Scans:         scansResponses,
		Links:         links,
	}
}

// Convert a current scan object data being executed of the system into a format easy to interpret
// by the user
func CurrentScanToScansResponse(currentScan model.CurrentScan,
	pagination dao.ScanDAOPagination) ScansResponse {

	var scansResponses []ScanResponse
	scansResponses = append(scansResponses, CurrentScanToScanResponse(currentScan))

	var orderBy string
	for _, sort := range pagination.OrderBy {
		if len(orderBy) > 0 {
			orderBy += "@"
		}

		orderBy += fmt.Sprintf("%s:%s",
			dao.ScanDAOOrderByFieldToString(sort.Field),
			dao.DAOOrderByDirectionToString(sort.Direction),
		)
	}

	// Add pagination managment links to the response. The URI is hard coded, I didn't have
	// any idea on how can we do this dynamically yet. We cannot get the URI from the
	// handler because we are going to have a cross-reference problem
	links := []Link{
		{
			Types: []LinkType{LinkTypeCurrent},
			HRef:  "/scans/?current",
		},
	}

	// Only add fast backward if we aren't in the first page
	if pagination.Page > 1 {
		links = append(links, Link{
			Types: []LinkType{LinkTypeFirst},
			HRef: fmt.Sprintf("/scans/?pagesize=%d&page=%d&orderby=%s",
				pagination.PageSize, 1, orderBy,
			),
		})
	}

	// Only add next if there's a next page
	if pagination.Page+1 <= pagination.NumberOfPages {
		links = append(links, Link{
			Types: []LinkType{LinkTypeNext},
			HRef: fmt.Sprintf("/scans/?pagesize=%d&page=%d&orderby=%s",
				pagination.PageSize, pagination.Page+1, orderBy,
			),
		})
	}

	// Only add the fast forward if we aren't on the last page
	if pagination.Page < pagination.NumberOfPages {
		links = append(links, Link{
			Types: []LinkType{LinkTypeLast},
			HRef: fmt.Sprintf("/scans/?pagesize=%d&page=%d&orderby=%s",
				pagination.PageSize, pagination.NumberOfPages, orderBy,
			),
		})
	}

	return ScansResponse{
		Page:          pagination.Page,
		PageSize:      pagination.PageSize,
		NumberOfPages: pagination.NumberOfPages,
		NumberOfItems: pagination.NumberOfItems,
		Scans:         scansResponses,
		Links:         links,
	}
}
