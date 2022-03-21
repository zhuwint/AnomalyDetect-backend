package union

import (
	"github.com/go-playground/assert/v2"
	"testing"
	"time"
)

func TestState(t *testing.T) {
	state := NewState()
	for i := 0; i < 10; i++ {
		state.Push(PV{
			T: time.Now().Add(time.Second * time.Duration(i)),
			V: float64(i),
		})
	}
	for i := 40; i > 20; i-- {
		state.Push(PV{
			T: time.Now().Add(time.Second * time.Duration(i)),
			V: float64(i),
		})
	}
	for state.Len() > 0 {
		a := state.Top()
		assert.Equal(t, a, state.Pop())
	}
}
