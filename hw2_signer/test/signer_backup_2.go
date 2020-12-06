package main

import (
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

func main() {
	// inputData := []int{0, 1, 1, 2, 3, 5, 8}
	// testResult := "NOT_SET"
	// hashSignJobs := []job{
	// 	job(func(in, out chan interface{}) {
	// 		for _, fibNum := range inputData {
	// 			out <- fibNum
	// 		}
	// 	}),
	// 	job(SingleHash),
	// 	job(MultiHash),
	// 	job(CombineResults),
	// 	job(func(in, out chan interface{}) {
	// 		dataRaw := <-in

	// 		data := dataRaw.(string)
	// 		// if !ok {
	// 		// 	t.Error("cant convert result data to string")
	// 		// }
	// 		testResult = data
	// 		fmt.Println(testResult)
	// 	}),
	// }

	// handledCount = make([]int32, len(hashSignJobs))
	// ExecutePipeline(hashSignJobs...)
}

var handledCount []int32

func inc() {
	for i := 0; i < len(handledCount); i++ {
		atomic.AddInt32(&handledCount[i], 1)
	}
}

func dec(out chan interface{}, i int32) {
	atomic.AddInt32(&handledCount[i], -1)
	if handledCount[i] == 0 {
		close(out)
	}
}

func startJob(job job, in chan interface{}, out chan interface{}, wg *sync.WaitGroup) {
	defer wg.Done()
	job(in, out)
}

func startJob0(job job, in chan interface{}, out chan interface{}, wg *sync.WaitGroup) {
	defer wg.Done()
	ch := make(chan interface{})

	go func(in, out chan interface{}) {
		job(in, out)
		close(out)
	}(in, ch)
	go func(ch, out chan interface{}) {
		for x := range ch {
			go inc()
			out <- x
		}
		close(out)
	}(ch, out)
}

func ExecutePipeline(jobs ...job) {
	wg := &sync.WaitGroup{}

	wg.Add(len(jobs))
	inputChan := make(chan interface{})
	chans := make([]chan interface{}, len(jobs))
	outPrev := inputChan
	for i := 1; i < len(chans); i++ {
		out := make(chan interface{})
		go startJob(jobs[i], outPrev, out, wg)
		outPrev = out
	}

	go startJob0(jobs[0], inputChan, inputChan, wg)
	wg.Wait()
}

func SingleHash(in, out chan interface{}) {
	mu := &sync.Mutex{}
	for val := range in {
		signedMd5Ch := make(chan interface{})
		signedCrc32Ch := make(chan interface{})
		signedMd5Crc32Ch := make(chan interface{})

		go func(out chan interface{}) {
			signedCrc32 := <-signedCrc32Ch
			signedMd5Crc32 := <-signedMd5Crc32Ch
			res := signedCrc32.(string) + "~" + signedMd5Crc32.(string)
			out <- res
			go dec(out, 1)
		}(out)

		val := val
		go func(in, out chan interface{}) {
			for signedMd5 := range in {
				out <- DataSignerCrc32(signedMd5.(string))
			}
		}(signedMd5Ch, signedMd5Crc32Ch)

		go func(out chan interface{}) {
			out <- DataSignerCrc32(strconv.Itoa(val.(int)))
		}(signedCrc32Ch)

		go func(out chan interface{}) {
			mu.Lock()
			out <- DataSignerMd5(strconv.Itoa(val.(int)))
			mu.Unlock()
		}(signedMd5Ch)
	}
}

func MultiHash(in, out chan interface{}) {
	prefixes := []string{"0", "1", "2", "3", "4", "5"}
	resultCh := make(chan interface{})
	for val := range in {
		go func(out chan interface{}) {
			signedCrc32Steps1 := make(chan interface{})
			signedCrc32Steps2 := make(chan interface{})
			signedCrc32Steps3 := make(chan interface{})
			signedCrc32Steps4 := make(chan interface{})
			signedCrc32Steps5 := make(chan interface{})
			signedCrc32Steps6 := make(chan interface{})

			for i := 0; i < len(prefixes); i++ {
				go func(out chan interface{}, prefix string) {
					out <- DataSignerCrc32(prefix + val.(string))
				}(signedCrc32Steps1, prefixes[0])
				go func(out chan interface{}, prefix string) {
					out <- DataSignerCrc32(prefix + val.(string))
				}(signedCrc32Steps2, prefixes[1])
				go func(out chan interface{}, prefix string) {
					out <- DataSignerCrc32(prefix + val.(string))
				}(signedCrc32Steps3, prefixes[2])
				go func(out chan interface{}, prefix string) {
					out <- DataSignerCrc32(prefix + val.(string))
				}(signedCrc32Steps4, prefixes[3])
				go func(out chan interface{}, prefix string) {
					out <- DataSignerCrc32(prefix + val.(string))
				}(signedCrc32Steps5, prefixes[4])
				go func(out chan interface{}, prefix string) {
					out <- DataSignerCrc32(prefix + val.(string))
				}(signedCrc32Steps6, prefixes[5])
			}
			result1 := <-signedCrc32Steps1
			result2 := <-signedCrc32Steps2
			result3 := <-signedCrc32Steps3
			result4 := <-signedCrc32Steps4
			result5 := <-signedCrc32Steps5
			result6 := <-signedCrc32Steps6
			out <- result1.(string) + result2.(string) + result3.(string) + result4.(string) + result5.(string) + result6.(string)
		}(resultCh)
	}

	go func(resultCh, out chan interface{}) {
		for val := range resultCh {
			out <- val
			go dec(out, 2)
		}
	}(resultCh, out)
}

func CombineResults(in, out chan interface{}) {
	result := make([]string, 7)
	var i int
	for val := range in {
		result[i] = val.(string)
		i++
	}
	sort.Strings(result)
	out <- strings.Join(result, "_")
	close(out)
}
