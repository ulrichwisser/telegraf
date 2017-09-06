package dnskey

import (
	"testing"
	"time"

	"github.com/influxdata/telegraf/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var servers = []string{"80.80.80.80"}
var domains = []string{"."}

func TestGathering(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network-dependent test in short mode.")
	}
	var dnsConfig = Dnskey{
		Resolvers: servers,
		Domains:   domains,
	}
	var acc testutil.Accumulator

	err := acc.GatherError(dnsConfig.Gather)
	assert.NoError(t, err)
	metric, ok := acc.Get("dns_query")
	require.True(t, ok)
	queryTime, _ := metric.Fields["query_time_ms"].(float64)

	assert.NotEqual(t, 0, queryTime)
}

func TestMetricContainsServerAndDomainAndRecordTypeTags(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network-dependent test in short mode.")
	}
	var dnsConfig = Dnskey{
		Resolvers: servers,
		Domains:   domains,
	}
	var acc testutil.Accumulator
	tags := map[string]string{
		"server": "80.80.80.80",
		"domain": ".",
	}
	fields := map[string]interface{}{}

	err := acc.GatherError(dnsConfig.Gather)
	assert.NoError(t, err)
	metric, ok := acc.Get("dnskey")
	require.True(t, ok)
	queryTime, _ := metric.Fields["query_time_ms"].(float64)

	fields["query_time_ms"] = queryTime
	acc.AssertContainsTaggedFields(t, "dnskey", fields, tags)
}

func TestGatheringTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network-dependent test in short mode.")
	}
	var dnsConfig = Dnskey{
		Resolvers: servers,
		Domains:   domains,
	}
	var acc testutil.Accumulator
	dnsConfig.Timeout = 1
	var err error

	channel := make(chan error, 1)
	go func() {
		channel <- acc.GatherError(dnsConfig.Gather)
	}()
	select {
	case res := <-channel:
		err = res
	case <-time.After(time.Second * 2):
		err = nil
	}

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "i/o timeout")
}

func TestSettingDefaultValues(t *testing.T) {
	dnsConfig := Dnskey{}

	dnsConfig.setDefaultValues()

	assert.Equal(t, []string{"."}, dnsConfig.Domains, "Default domain not equal \".\"")
	assert.Equal(t, 2, dnsConfig.Timeout, "Default timeout not equal 2")
}

func TestAlgorithmName(t *testing.T) {
	assert.Equal(t, "RSAMD5", algorithmName(1))
	assert.Equal(t, "DH", algorithmName(2))
	assert.Equal(t, "RSASHA1", algorithmName(5))
	assert.Equal(t, "RSASHA256", algorithmName(8))
	assert.Equal(t, "242", algorithmName(242))
	assert.Equal(t, "255", algorithmName(255))
}

func TestKeyType(t *testing.T) {
	assert.Equal(t, "KSK", keyType(257))
	assert.Equal(t, "ZSK", keyType(256))
	assert.Equal(t, "0", keyType(0))
	assert.Equal(t, "327", keyType(327))
}
