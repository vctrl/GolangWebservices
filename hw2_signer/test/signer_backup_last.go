package main

import (
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

func main() {
	// var (
	// 	DataSignerCrc32Counter uint32
	// )
	// DataSignerCrc32 = func(data string) string {
	// 	atomic.AddUint32(&DataSignerCrc32Counter, 1)
	// 	fmt.Println(DataSignerCrc32Counter)
	// 	data += DataSignerSalt
	// 	crcH := crc32.ChecksumIEEE([]byte(data))
	// 	dataHash := strconv.FormatUint(uint64(crcH), 10)
	// 	time.Sleep(time.Second)
	// 	return dataHash
	// }

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
	// 	}),
	// }

	// handledCount = make([]int32, len(hashSignJobs))
	// ExecutePipeline(hashSignJobs...)

	// var recieved uint32
	// freeFlowJobs := []job{
	// 	job(func(in, out chan interface{}) {
	// 		out <- uint32(1)
	// 		out <- uint32(3)
	// 		out <- uint32(4)
	// 	}),
	// 	job(func(in, out chan interface{}) {
	// 		for val := range in {
	// 			out <- val.(uint32) * 3
	// 			time.Sleep(time.Millisecond * 100)
	// 		}
	// 	}),
	// 	job(func(in, out chan interface{}) {
	// 		for val := range in {
	// 			fmt.Println("collected", val)
	// 			atomic.AddUint32(&recieved, val.(uint32))
	// 		}
	// 	}),
	// }

	// ExecutePipeline(freeFlowJobs...)

}

var handledCount []int32

func inc() {
	for i := 0; i < len(handledCount); i++ {
		atomic.AddInt32(&handledCount[i], 1)
	}
}

func dec(out chan interface{}, i int32) {
	atomic.AddInt32(&handledCount[i], -1)
}

func startJob(job job, in chan interface{}, out chan interface{}, wg *sync.WaitGroup) {
	defer wg.Done()
	job(in, out)
	close(out)
}

func startJob0(job job, in chan interface{}, out chan interface{}, wg *sync.WaitGroup) {
	defer wg.Done()

	job(in, out)
	close(out)
}

func ExecutePipeline(jobs ...job) {
	wg := &sync.WaitGroup{}
	wg.Add(len(jobs))
	ch1 := make(chan interface{})
	outPrev := ch1
	for i := 1; i < len(jobs); i++ {
		out := make(chan interface{})
		go startJob(jobs[i], outPrev, out, wg)
		outPrev = out
	}

	go startJob0(jobs[0], ch1, ch1, wg)
	wg.Wait()
}

func SingleHash(in, out chan interface{}) {
	wg := &sync.WaitGroup{}
	mu := &sync.Mutex{}
	for val := range in {
		wg.Add(1)
		signedMd5Ch := make(chan interface{})
		signedCrc32Ch := make(chan interface{})
		signedMd5Crc32Ch := make(chan interface{})

		go func(out chan interface{}) {
			defer wg.Done()
			signedCrc32 := <-signedCrc32Ch
			signedMd5Crc32 := <-signedMd5Crc32Ch
			res := signedCrc32.(string) + "~" + signedMd5Crc32.(string)
			out <- res
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
	wg.Wait()
}

func MultiHash(in, out chan interface{}) {
	wg := &sync.WaitGroup{}
	prefixes := []string{"0", "1", "2", "3", "4", "5"}
	resultCh := make(chan interface{})
	mu := &sync.Mutex{}
	for val := range in {
		wg.Add(1)
		go func(out chan interface{}) {
			signedCrc32Steps1 := make(chan interface{})
			signedCrc32Steps2 := make(chan interface{})
			signedCrc32Steps3 := make(chan interface{})
			signedCrc32Steps4 := make(chan interface{})
			signedCrc32Steps5 := make(chan interface{})
			signedCrc32Steps6 := make(chan interface{})

			for i := 0; i < len(prefixes); i++ {
				if i == 0 {
					mu.Lock()
					go func(out chan interface{}, prefix string) {
						out <- DataSignerCrc32(prefix + val.(string))
					}(signedCrc32Steps1, prefixes[0])
					mu.Unlock()
				}
				if i == 1 {
					mu.Lock()
					go func(out chan interface{}, prefix string) {
						out <- DataSignerCrc32(prefix + val.(string))
					}(signedCrc32Steps2, prefixes[1])
					mu.Unlock()
				}

				if i == 2 {
					mu.Lock()
					go func(out chan interface{}, prefix string) {
						out <- DataSignerCrc32(prefix + val.(string))
					}(signedCrc32Steps3, prefixes[2])
					mu.Unlock()
				}
				if i == 3 {
					mu.Lock()
					go func(out chan interface{}, prefix string) {
						out <- DataSignerCrc32(prefix + val.(string))
					}(signedCrc32Steps4, prefixes[3])
					mu.Unlock()
				}

				if i == 4 {
					mu.Lock()
					go func(out chan interface{}, prefix string) {

						out <- DataSignerCrc32(prefix + val.(string))

					}(signedCrc32Steps5, prefixes[4])
					mu.Unlock()
				}

				if i == 5 {
					mu.Lock()
					go func(out chan interface{}, prefix string) {

						out <- DataSignerCrc32(prefix + val.(string))

					}(signedCrc32Steps6, prefixes[5])
					mu.Unlock()
				}

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
			wg.Done()
		}
	}(resultCh, out)

	wg.Wait()
}

func CombineResults(in, out chan interface{}) {
	result := make([]string, 0)
	var i int
	for val := range in {
		result = append(result, val.(string))
		i++
	}
	sort.Strings(result)
	out <- strings.Join(result, "_")
}
