package main

import (
	"sort"
	"strconv"
	"strings"
	"sync"
)

func main() {

}

func startJob(job job, in chan interface{}, out chan interface{}, wg *sync.WaitGroup) {
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
	go startJob(jobs[0], ch1, ch1, wg)
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
	for val := range in {
		val := val
		wg.Add(1)
		go func(out chan interface{}) {
			signedCrc32Steps := make([]chan interface{}, len(prefixes))
			for i := 0; i < len(prefixes); i++ {
				signedCrc32Steps[i] = make(chan interface{})
				go func(out chan interface{}, prefix string) {
					out <- DataSignerCrc32(prefix + val.(string))
				}(signedCrc32Steps[i], prefixes[i])
			}
			var result string
			for i := 0; i < len(prefixes); i++ {
				result += (<-signedCrc32Steps[i]).(string)
			}
			out <- result
			wg.Done()
		}(out)
	}
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
