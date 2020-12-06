package main

import (
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
)

func main() {
	inputData := []int{0, 1, 1, 2, 3, 5, 8}

	hashSignJobs := []job{
		job(func(in, out chan interface{}) {
			for _, fibNum := range inputData {
				out <- fibNum
			}
		}),
		job(SingleHash),
		job(MultiHash),
		// job(CombineResults),
	}

	ExecutePipeline(hashSignJobs...)
}

func startJob(job job, in chan interface{}, out chan interface{}) {
	// defer wg.Done()
	job(in, out)
	close(out)
}

var count int32

func inc() {
	fmt.Println(count)
	atomic.AddInt32(&count, 1)
}

func dec(out chan interface{}) {
	atomic.AddInt32(&count, -1)
	if count == 0 {
		close(out)
	}
}

func startJob0(job job, in chan interface{}, out chan interface{}) {
	ch := make(chan interface{})
	go job(in, ch)
	for x := range ch {
		// fmt.Println(x)
		go inc()
		// fmt.("incremented", count)
		out <- x
	}
}

func ExecutePipeline(jobs ...job) {
	ch1 := make(chan interface{})
	ch2 := make(chan interface{})
	ch3 := make(chan interface{})
	// wg.Add(1)
	go startJob(jobs[1], ch1, ch2)
	go startJob(jobs[2], ch2, ch3)
	// wg.Add(1)
	go startJob0(jobs[0], ch1, ch1)
	// wg.Wait()
	for val := range ch3 {
		fmt.Println(val)
	}
}

func SingleHash(in, out chan interface{}) {
	// fmt.Println(count)
	mu := &sync.Mutex{}
	for val := range in {
		// fmt.Println("reading ", count)
		signedMd5Ch := make(chan interface{})
		signedCrc32Ch := make(chan interface{})
		signedMd5Crc32Ch := make(chan interface{})

		go func(out chan interface{}) {
			signedCrc32 := <-signedCrc32Ch
			signedMd5Crc32 := <-signedMd5Crc32Ch
			res := signedCrc32.(string) + "~" + signedMd5Crc32.(string)
			out <- res
			go dec(out)
			// fmt.Println("decremented", count)
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
	// for val := range in {
	// 	fmt.Println("in multihash", val)
	// 	out <- val
	// }

	prefixes := []string{"0", "1", "2", "3", "4", "5"}

	resultCh := make(chan interface{})

	for val := range in {
		go func(out chan interface{}) {
			signedCrc32 := make(chan interface{})
			for i := 0; i < len(prefixes); i++ {
				go func(out chan interface{}, prefix string) {
					out <- DataSignerCrc32(prefix + val.(string))
				}(signedCrc32, prefixes[i])
			}

			var result string
			for i := 0; i < len(prefixes); i++ {
				result = result + (<-signedCrc32).(string)
			}

			out <- result
		}(resultCh)
	}

	for val := range resultCh {
		out <- val
	}
}

func CombineResults(in, out chan interface{}) {
	for val := range in {
		fmt.Println("in combine", val)
		out <- val
	}
}

// crc32(04108050209~502633748) + crc32(14108050209~502633748) + crc32(24108050209~502633748) + crc32(34108050209~502633748) + crc32(44108050209~502633748) + crc32(54108050209~502633748)
