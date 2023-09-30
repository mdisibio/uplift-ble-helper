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

var metricsDeskActive = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "uplift_desk_active",
	Help: "User is considered actively working at the desk based on system checks. 1 = yes",
}, []string{"reason"})
