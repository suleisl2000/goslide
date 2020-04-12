// Copyright (c) 2020, The GoSLIDE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nlpodyssey/goslide/configuration"
	"github.com/nlpodyssey/goslide/network"
	"github.com/nlpodyssey/goslide/node"
)

var logger = log.New(os.Stderr, "", 0)

var globalTime time.Duration

func main() {
	const cowId = 0

	validateArguments()
	loadGlobalConfiguration()

	config := configuration.Global

	// Initialize Network

	numBatches := config.TotRecords / config.BatchSize
	numBatchesTest := config.TotRecordsTest / config.BatchSize

	layersTypes := make([]node.NodeType, config.NumLayer)
	for i := 0; i < config.NumLayer-1; i++ {
		layersTypes[i] = node.ReLU
	}
	layersTypes[config.NumLayer-1] = node.Softmax

	if config.LoadWeight {
		/*
			TODO: load weight...
			cnpy::npz_t arr;
			arr = cnpy::npz_load(Weights);
		*/
	}

	startTime := time.Now()
	myNet := network.New(
		cowId,
		config.SizesOfLayers,
		layersTypes,
		config.NumLayer,
		config.BatchSize,
		config.Lr,
		config.InputDim,
		config.K,
		config.L,
		config.RangePow,
		config.Sparsity,
	)
	endTime := time.Now()
	fmt.Printf("Network Initialization takes %v.\n", endTime.Sub(startTime))

	// Start Training

	for e := 0; e < config.Epoch; e++ {
		fmt.Printf("Epoch %d\n", e)

		// train
		readDataSvm(cowId, numBatches, myNet, e)

		// test
		if e == config.Epoch-1 {
			evalDataSvm(cowId, numBatchesTest, myNet, (e+1)*numBatches)
		} else {
			evalDataSvm(cowId, 50, myNet, (e+1)*numBatches)
		}
	}
}

func readDataSvm(cowId, numBatches int, myNet *network.Network, epoch int) {
	config := configuration.Global

	file, err := os.Open(config.TrainData)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// skip header
	scanner.Scan()
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	for i := 0; i < numBatches; i++ {
		if (i+epoch*numBatches)%config.Stepsize == 0 {
			evalDataSvm(cowId, 20, myNet, epoch*numBatches+i)
		}
		records := make([][]int, config.BatchSize)
		values := make([][]float64, config.BatchSize)
		sizes := make([]int, config.BatchSize)
		labels := make([][]int, config.BatchSize)
		labelSize := make([]int, config.BatchSize)
		nonZeros := 0
		count := 0

		for scanner.Scan() {
			label, list, value := parseLine(scanner.Text())

			nonZeros += len(list)
			records[count] = make([]int, len(list))
			values[count] = make([]float64, len(list))
			labels[count] = make([]int, len(label))
			sizes[count] = len(list)
			labelSize[count] = len(label)

			for currCount, currValue := range list {
				records[count][currCount], err = strconv.Atoi(currValue)
				if err != nil {
					log.Fatal(err)
				}
			}

			for currCount, currValue := range value {
				values[count][currCount], err = strconv.ParseFloat(currValue, 64)
				if err != nil {
					log.Fatal(err)
				}
			}

			for currCount, currValue := range label {
				labels[count][currCount], err = strconv.Atoi(currValue)
				if err != nil {
					log.Fatal(err)
				}
			}

			count++
			if count >= config.BatchSize {
				break
			}
		}

		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}

		rehash := false
		rebuild := false

		if config.LayerMode == configuration.LayerMode1 || config.LayerMode == configuration.LayerMode4 {
			if (epoch*numBatches+i)%(config.Rehash/config.BatchSize) == (config.Rehash/config.BatchSize - 1) {
				rehash = true
			}
			// TODO: probably there was an error in the original code (using rehash at right)
			if (epoch*numBatches+i)%(config.Rebuild/config.BatchSize) == (config.Rebuild/config.BatchSize - 1) {
				rebuild = true
			}
		}

		startTime := time.Now()

		// logloss
		_, myNet = myNet.ProcessInput(
			cowId,
			records,
			values,
			sizes,
			labels,
			labelSize,
			epoch*numBatches+i,
			rehash,
			rebuild,
		)

		endTime := time.Now()
		globalTime += endTime.Sub(startTime)
	}
}

func evalDataSvm(cowId, numBatchesTest int, myNet *network.Network, iter int) {
	config := configuration.Global

	totCorrect := 0

	file, err := os.Open(config.TestData)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// skip header
	scanner.Scan()
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	for i := 0; i < numBatchesTest; i++ {
		records := make([][]int, config.BatchSize)
		values := make([][]float64, config.BatchSize)
		sizes := make([]int, config.BatchSize)
		labels := make([][]int, config.BatchSize)
		labelSize := make([]int, config.BatchSize)
		nonZeros := 0
		count := 0

		for scanner.Scan() {
			label, list, value := parseLine(scanner.Text())

			nonZeros += len(list)
			records[count] = make([]int, len(list))
			values[count] = make([]float64, len(list))
			labels[count] = make([]int, len(label))
			sizes[count] = len(list)
			labelSize[count] = len(label)

			for currCount, currValue := range list {
				records[count][currCount], err = strconv.Atoi(currValue)
				if err != nil {
					log.Fatal(err)
				}
			}

			for currCount, currValue := range value {
				values[count][currCount], err = strconv.ParseFloat(currValue, 64)
				if err != nil {
					log.Fatal(err)
				}
			}

			for currCount, currValue := range label {
				labels[count][currCount], err = strconv.Atoi(currValue)
				if err != nil {
					log.Fatal(err)
				}
			}

			count++
			if count >= config.BatchSize {
				break
			}
		}

		numFeatures := 0
		numLabels := 0
		for i := 0; i < config.BatchSize; i++ {
			numFeatures += sizes[i]
			numLabels += labelSize[i]
		}

		fmt.Printf("%d records, with %d features and %d labels\n",
			config.BatchSize, numFeatures, numLabels)

		var correctPredict int

		// FIXME: reassignment of myNet for CoW problematic
		correctPredict, myNet = myNet.PredictClass(
			cowId,
			records,
			values,
			sizes,
			labels,
			labelSize,
		)

		totCorrect += correctPredict

		fmt.Printf("Iter %d: %f correct\n",
			i, float64(totCorrect)/(float64(config.BatchSize)*(float64(i)+1)))
	}

	fmt.Printf("Over all: %f correct\n",
		float64(totCorrect)/(float64(numBatchesTest)*float64(config.BatchSize)))

	fmt.Printf("%d %v %f\n",
		iter, globalTime, float64(totCorrect)/(float64(numBatchesTest)*float64(config.BatchSize)))
}

func parseLine(line string) (label, list, value []string) {
	spl := strings.Split(line, " ")

	label = make([]string, 0)
	list = make([]string, 0)
	value = make([]string, 0)

	for _, s := range strings.Split(spl[0], ",") {
		label = append(label, s)
	}

	for _, s := range spl[1:len(spl)] {
		spl2 := strings.Split(s, ":")
		list = append(list, spl2[0])
		value = append(value, spl2[1])
	}

	return
}

func validateArguments() {
	if len(os.Args) != 2 {
		logger.Println("Invalid or malformed arguments.")
		logger.Fatal("\nUsage:\n  goslide <json_configuration_file>\n")
	}
}

func loadGlobalConfiguration() {
	configFilename := os.Args[1]
	config, err := configuration.FromJsonFile(configFilename)

	if err != nil {
		logger.Println("An error occurred reading the configuration file.")
		logger.Fatal(err)
	}

	configuration.Global = config
}