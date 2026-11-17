//go:build sqlite_cgo_test

package main

import "github.com/djshellshoxxx/circula/future/favorites/internal/sqlitefixture"

func init() {
	driverName = sqlitefixture.Driver
	driverDSN = sqlitefixture.DriverDSN
}
