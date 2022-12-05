package utils

import (
	"io"
	"sync"
)

func FullPipe(a io.ReadWriteCloser, b io.ReadWriteCloser) {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func(wg *sync.WaitGroup, a io.ReadWriteCloser, b io.ReadWriteCloser) {
		defer wg.Done()
		io.Copy(a, b)
		a.Close()
		b.Close()
	}(wg, a, b)

	io.Copy(b, a)
	a.Close()
	b.Close()
	wg.Wait()
}
