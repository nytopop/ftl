package ftm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFTMVal(t *testing.T) {
	t.Run("Checkpoint", func(t *testing.T) {
		v := NewFTMValV(64)

		err := v.Atomic(TxVal.Checkpoint(
			func(s FTMVal) error {
				vv := s.Get()
				assert.Equal(t, interface{}(64), vv)
				s.Put(32)
				return nil
			},

			func(s FTMVal) error {
				vv := s.Get()
				assert.Equal(t, interface{}(32), vv)
				s.Put(16)
				return nil
			},

			func(s FTMVal) error {
				vv := s.Get()
				assert.Equal(t, interface{}(16), vv)

				// calling discard should go back to checkpoint beginning
				s.Discard()
				vv = s.Get()
				assert.Equal(t, interface{}(64), vv)
				return nil
			},
		))()
		assert.NoError(t, err)
	})

	t.Run("Seq", func(t *testing.T) {
		v := NewFTMValV(64)

		err := v.Atomic(TxVal.Seq(
			func(s FTMVal) error {
				//vv := s.Get()
				s.Put(32)
				return nil
			},

			func(s FTMVal) error {
				//vv := s.Get()
				s.Put(16)
				return nil
			},

			func(s FTMVal) error {
				vv := s.Get()
				assert.Equal(t, interface{}(16), vv)

				// calling discard should go back to checkpoint beginning
				s.Discard()
				vv = s.Get()
				assert.Equal(t, interface{}(64), vv)
				return nil
			},
		))()
		assert.NoError(t, err)
	})
}
