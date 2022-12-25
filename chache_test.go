package main

import "testing"

// we created a benchmark or this source code :
func BenchmarkMain(t *testing.B) {
	for i := 0; i < t.N; i++ {
		// call main as a benchmark
		main()
	}
}
