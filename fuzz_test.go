package ben_test

import (
	"bufio"
	"bytes"
	"io"
	"testing"

	"github.com/fudanchii/ben"
	//nolint: revive // BDD sucks
	. "github.com/onsi/gomega"
)

func FuzzDecode(f *testing.F) {
	f.Add("i23e")
	f.Add("0")
	f.Add("d0:e")
	f.Add("d0:0:e")
	f.Add("d1:a1:be")
	f.Add("li1234:abcde")
	f.Fuzz(func(t *testing.T, input string) {
		g := NewWithT(t)
		l, err := ben.InferredTypeDecode(bufio.NewReader(bytes.NewBufferString(input)))

		if err != nil {
			g.Expect(err).To(SatisfyAny(
				MatchError(io.EOF),
				MatchError(ben.InvalidInputError{"should not reach here"}),
				MatchError(ben.InvalidInputError{"invalid string length"}),
				MatchError(ben.InvalidInputError{"input is not Integer"}),
				MatchError(ben.InvalidInputError{"key without value"}),
				MatchError(ben.ErrEndItemSequence),
			))
		} else {
			g.Expect(l).NotTo(BeNil())
		}
	})
}
