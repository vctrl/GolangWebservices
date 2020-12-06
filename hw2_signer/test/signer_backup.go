package main

import (
	"fmt"
	"strconv"
	"sync"
)

func main() {
	inputData := []int{0, 1, 1, 2, 3, 5, 8}

	hashSignJobs := []job{
		job(func(in, out chan interface{}) {
			for _, fibNum := range inputData {
				fmt.Println("ya rabotau")
				out <- fibNum
			}
			fmt.Println("ya zakonchil")

		}),
		job(SingleHash),
		// job(MultiHash),
		// job(CombineResults),
	}

	ExecutePipeline(hashSignJobs...)
}

func startJob(job job, in chan interface{}, out chan interface{}) {
	// defer wg.Done()
	job(in, out)
	fmt.Println("im done1")
	close(out)
}

var dataCount int32

func startJob0(job job, in chan interface{}, out chan interface{}) {
	fmt.Println("heh ok")
	job(in, out)
	fmt.Println("heh da")

	// wg.Done()
	// close(out)
}

func ExecutePipeline(jobs ...job) {
	ch1 := make(chan interface{})
	ch2 := make(chan interface{})
	// wg.Add(1)
	go startJob(jobs[1], ch1, ch2)
	// wg.Add(1)
	go startJob0(jobs[0], ch1, ch1)
	// wg.Wait()
	for val := range ch2 {
		fmt.Println(val)
	}
}

func SingleHash(in, out chan interface{}) {
	fmt.Println(dataCount)
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
			// close(out)
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
	for val := range in {
		fmt.Println("in multihash", val)
		out <- val
	}
}

func CombineResults(in, out chan interface{}) {
	for val := range in {
		fmt.Println("in combine", val)
		out <- val
	}
}
