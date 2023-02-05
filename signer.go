package main

import (
	"sort"
	"strconv"
	"strings"
	"sync"
)

func ExecutePipeline(jobs ...job) {
	wg := &sync.WaitGroup{}
	in := make(chan interface{}, 100)

	for _, j := range jobs {
		out := make(chan interface{}, 100)
		wg.Add(1)

		go func(j job, in, out chan interface{}) {
			defer wg.Done()
			defer close(out)

			j(in, out)
		}(j, in, out)

		in = out
	}

	wg.Wait()
}

func SingleHash(in, out chan interface{}) {
	mu := &sync.Mutex{}
	wg := &sync.WaitGroup{}

	for data := range in {
		dataString := strconv.Itoa(data.(int))
		wg.Add(1)

		go func(dataString string) {
			defer wg.Done()

			md5 := make(chan string)
			crc32 := make(chan string, 1)
			crc32md5 := make(chan string, 1)

			go func(str string) {
				mu.Lock()
				defer mu.Unlock()
				md5 <- DataSignerMd5(str)
			}(dataString)

			go func(str string) {
				crc32 <- DataSignerCrc32(str)
			}(dataString)

			go func(str string) {
				crc32md5 <- DataSignerCrc32(str)
			}(<-md5)

			result := <-crc32 + "~" + <-crc32md5
			out <- result
		}(dataString)
	}

	wg.Wait()
}

func MultiHash(in, out chan interface{}) {
	wg := sync.WaitGroup{}

	for data := range in {
		dataString := data.(string)
		wg.Add(1)

		go func(dataString string) {
			defer wg.Done()

			var crc32 [6]chan string
			for th := 0; th <= 5; th++ {
				crc32[th] = make(chan string)

				go func(i int, str string) {
					crc32[i] <- DataSignerCrc32(strconv.Itoa(i) + str)
				}(th, dataString)
			}

			result := ""
			for th := 0; th <= 5; th++ {
				result += <-crc32[th]
			}

			out <- result
		}(dataString)
	}

	wg.Wait()
}

func CombineResults(in, out chan interface{}) {
	var lines []string

	for data := range in {
		lines = append(lines, data.(string))
	}
	sort.Strings(lines)

	result := strings.Join(lines, "_")
	out <- result
}
