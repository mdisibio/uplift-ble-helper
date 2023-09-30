package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var metricsDeskHeight = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "uplift_desk_height_inches",
	Help: "Current desk height",
}, []string{"desk_addr"})

var metricsDeskHeightMeters = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "uplift_desk_height_meters",
	Help: "Current desk height in meters",
}, []string{"desk_addr"})
