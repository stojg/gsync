package main

import (
	"encoding/csv"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"math"
	"net"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/stojg/gsync/lib/client"
	"github.com/stojg/gsync/lib/msg"
)

type diagnostics struct {
	name         string
	len          int
	min          time.Duration
	mean         time.Duration
	max          time.Duration
	stdDev       float64
	percentile50 time.Duration
	percentile95 time.Duration
	percentile99 time.Duration
}

func (d *diagnostics) String() string {
	res := fmt.Sprintf("files: %d\n", d.len)
	res += fmt.Sprintf("mean: %s\n", d.mean)
	res += fmt.Sprintf("min: %s\n", d.min)
	res += fmt.Sprintf("max: %s\n", d.max)
	res += fmt.Sprintf("std dev: %.2f\n", d.stdDev)
	res += fmt.Sprintf("50%% percentile: %s\n", d.percentile50)
	res += fmt.Sprintf("95%% percentile: %s\n", d.percentile95)
	res += fmt.Sprintf("99%% percentile: %s\n", d.percentile99)
	return res
}

func (d *diagnostics) Headers() []string {
	return []string{
		"name", "num", "mean", "min", "max", "stdDev", "percentile_50", "percentile_95", "percentile_99",
	}
}

func (d *diagnostics) StringSlice() []string {
	return []string{
		d.name,
		fmt.Sprintf("%d", d.len),
		fmt.Sprintf("%s", d.mean),
		fmt.Sprintf("%s", d.min),
		fmt.Sprintf("%s", d.max),
		fmt.Sprintf("%0.1f", d.stdDev),
		fmt.Sprintf("%s", d.percentile50),
		fmt.Sprintf("%s", d.percentile95),
		fmt.Sprintf("%s", d.percentile99),
	}
}

func newDiagnostics(name string, results []*result) *diagnostics {
	// set the min and max to the highest and lowest time duration (see time.minDuration)
	d := &diagnostics{
		name: name,
		min:  time.Duration(1<<63 - 1),
		max:  time.Duration(-1 << 63),
		len:  len(results),
	}
	var sum time.Duration
	for _, res := range results {
		//sizes = append(sizes, res.size)
		sum += res.duration
		d.max = maxDuration(d.max, res.duration)
		d.min = minDuration(d.min, res.duration)
	}

	d.mean = sum / time.Duration(len(results))
	d.stdDev = stdDeviation(results, d.mean)
	d.percentile50 = percentile(results, 50)
	d.percentile95 = percentile(results, 95)
	d.percentile99 = percentile(results, 98)
	return d
}

func runClient(address string, testDirectory string) error {
	fmt.Println("running as client")
	c := client.New(address)

	fmt.Printf("connecting to server %s\n", address)
	transferResults, err := clientHandler(c.Conn(), testDirectory)
	if err != nil {
		return err
	}

	w := csv.NewWriter(os.Stdout)

	overall := newDiagnostics("overall", transferResults.created)
	w.Write(overall.Headers())
	w.Flush()
	if err = w.Write(overall.StringSlice()); err != nil {
		fmt.Println(err)
	}
	w.Flush()
	//fmt.Printf("++++  overall:\n%s\n", overall)

	perSize := make(map[int64][]*result)
	for _, res := range transferResults.created {
		_, found := perSize[res.size]
		if !found {
			perSize[res.size] = make([]*result, 0)
		}
		perSize[res.size] = append(perSize[res.size], res)
	}

	for size, results := range perSize {
		diag := newDiagnostics(humanize.Bytes(uint64(size)), results)
		//fmt.Printf("++++ file size: %s:\n%s\n", humanize.Bytes(uint64(size)), diag)
		if err = w.Write(diag.StringSlice()); err != nil {
			fmt.Println(err)
		}
		w.Flush()
	}

	return nil
}

func stdDeviation(res []*result, mean time.Duration) float64 {
	var sqrSum time.Duration
	n := len(res)
	for i := 0; i < n; i++ {
		tmp := res[i].duration - mean
		sqrSum += tmp * tmp
	}
	// we need to convert  time.Durations to float64
	sumAsNSFloat := float64(sqrSum)
	divisor := float64(sqrSum / time.Duration(n))
	return math.Sqrt(sumAsNSFloat / divisor)
}

func percentile(results []*result, percentile int) time.Duration {
	tmp := len(results) * percentile
	index := int(0.01 * float64(tmp))
	var durations []int
	for _, res := range results {
		durations = append(durations, int(res.duration.Nanoseconds()))
	}
	sort.Ints(durations)
	return time.Duration(durations[index])
}

func maxDuration(a time.Duration, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}

func minDuration(a time.Duration, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

type result struct {
	duration time.Duration
	size     int64
}

type results struct {
	timeDifference time.Duration
	created        []*result
	deleted        []*result
	timeouts       int
}

func clientHandler(conn net.Conn, dir string) (*results, error) {
	defer conn.Close()

	dec := gob.NewDecoder(conn)
	enc := gob.NewEncoder(conn)

	var message msg.Message

	res := &results{}

	for {
		err := dec.Decode(&message)
		if err != nil {
			return res, err
		}
		switch message.Type {

		case msg.ClockSync:
			res.timeDifference = time.Since(message.CurrentTime)
			fmt.Printf("clocks syncronised, difference: %s\n", res.timeDifference)

		case msg.Done:
			fmt.Println("\nserver is done, disconnecting")
			return res, nil

		case msg.FileCreate:
			if err := checkCreated(dir, message, res); err != nil {
				return res, err
			}
			if err := msg.Send(enc, msg.OK, "", nil); err != nil {
				return res, err
			}

		case msg.FileDelete:
			if err := checkDeleted(dir, message, res); err != nil {
				return res, err
			}
			if err := msg.Send(enc, msg.OK, "", nil); err != nil {
				return res, err
			}

		default:
			fmt.Printf("unknown message type: %+v\n", message)
		}
	}
}

func checkCreated(dir string, message msg.Message, res *results) error {

	fileName := message.FileName
	timestamp := message.CurrentTime.Add(-res.timeDifference)

	timeoutValue := 61 * time.Second
	timeout := time.After(timeoutValue)
	ticker := time.NewTicker(1 * time.Millisecond)
	defer ticker.Stop()

Loop:
	for {
		select {

		case <-timeout:
			res.timeouts++
			return fmt.Errorf("\ncreated timed out after %s seconds. no %s\n", timeoutValue, fileName)

		case <-ticker.C:
			// we need to read the directory, otherwise filesystems like NFS have cached the files.
			stats, err := ioutil.ReadDir(dir)
			if err != nil {
				return fmt.Errorf("readdir: %s\n", err)
			}

			for _, stat := range stats {
				// we found the file we were waiting for
				if filepath.Base(stat.Name()) != fileName {
					continue
				}

				if stat.Size() != message.Size {
					continue Loop
				}
				res.created = append(res.created, &result{
					duration: time.Since(timestamp),
					size:     stat.Size(),
				})
				fmt.Print(".")
				return nil
			}
		}
	}
}

func checkDeleted(dir string, message msg.Message, res *results) error {

	fileName := message.FileName
	timestamp := message.CurrentTime.Add(-res.timeDifference)

	timeoutValue := 61 * time.Second
	timeout := time.After(timeoutValue)
	ticker := time.NewTicker(2 * time.Millisecond)

	defer ticker.Stop()

Loop:
	for {
		select {

		case <-timeout:
			res.timeouts++
			return fmt.Errorf("delete timed out after %s seconds", timeoutValue)

		case <-ticker.C:
			stats, err := ioutil.ReadDir(dir)
			if err != nil {
				return fmt.Errorf("readdir: %s\n", err)
			}

			for _, stat := range stats {
				// still found the file in the directory list
				if filepath.Base(stat.Name()) == fileName {
					continue Loop
				}
			}
			res.deleted = append(res.deleted, &result{
				duration: time.Since(timestamp),
			})
			fmt.Print("x")
			return nil
		}
	}
}
